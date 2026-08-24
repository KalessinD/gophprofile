//go:generate mockgen -source=processor.go -destination=mocks/mock_processor.gen.go -package=mocks
package worker

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"
	"strings"

	"github.com/KalessinD/gophprofile/internal/broker"
	"github.com/KalessinD/gophprofile/internal/models"
	"github.com/disintegration/imaging"
	"go.uber.org/zap"
)

const (
	extJPEG = ".jpeg"
	extJPG  = ".jpg"
	extPNG  = ".png"
)

type (
	// AvatarRepository defines the DB operations needed by the worker.
	AvatarRepository interface {
		GetAvatarByID(ctx context.Context, avatarID string) (*models.Avatar, error)
		UpdateAvatarStatus(ctx context.Context, avatarID string, status string, thumbnail100S3Key string, thumbnail300S3Key string, width int, height int) error
	}

	// ObjectStorage defines the S3 operations needed by the worker.
	ObjectStorage interface {
		GetObject(ctx context.Context, bucket string, objectKey string) (io.ReadCloser, error)
		UploadObject(ctx context.Context, bucket string, objectKey string, reader io.Reader) error
	}

	// ImageProcessor handles the core logic of downloading, resizing, and uploading avatars.
	ImageProcessor struct {
		repo   AvatarRepository
		s3     ObjectStorage
		bucket string
		log    *zap.Logger
	}
)

// NewImageProcessor creates a new ImageProcessor instance.
func NewImageProcessor(repo AvatarRepository, s3 ObjectStorage, bucket string, log *zap.Logger) *ImageProcessor {
	return &ImageProcessor{
		repo:   repo,
		s3:     s3,
		bucket: bucket,
		log:    log,
	}
}

// ProcessAvatar executes the full pipeline for a single avatar event.
func (p *ImageProcessor) ProcessAvatar(ctx context.Context, event *broker.AvatarEvent) error {
	avatar, err := p.repo.GetAvatarByID(ctx, event.AvatarID)
	if err != nil {
		return fmt.Errorf("fetching avatar metadata: %w", err)
	}

	// Skip if already processed or explicitly failed
	if avatar.Status != models.AvatarStatusProcessing {
		p.log.Info("skipping avatar, status is not processing", zap.String("avatar_id", event.AvatarID), zap.String("status", avatar.Status))
		return nil
	}

	srcImg, ext, err := p.downloadAndDecodeImage(ctx, avatar.OriginalS3Key)
	if srcImg == nil || err != nil {
		return p.failProcessing(ctx, event.AvatarID, err)
	}

	bounds := srcImg.Bounds()
	originalWidth, originalHeight := bounds.Dx(), bounds.Dy()

	thumb100Key, err := p.processThumbnail(ctx, srcImg, ext, "100x100", event.UserID, event.AvatarID, 100)
	if err != nil {
		return p.failProcessing(ctx, event.AvatarID, err)
	}

	thumb300Key, err := p.processThumbnail(ctx, srcImg, ext, "300x300", event.UserID, event.AvatarID, 300)
	if err != nil {
		return p.failProcessing(ctx, event.AvatarID, err)
	}

	if err := p.repo.UpdateAvatarStatus(ctx, event.AvatarID, models.AvatarStatusReady, thumb100Key, thumb300Key, originalWidth, originalHeight); err != nil {
		return fmt.Errorf("marking avatar as ready: %w", err)
	}

	p.log.Info("avatar processed successfully", zap.String("avatar_id", event.AvatarID))
	return nil
}

// failProcessing attempts to mark the avatar as failed in the DB and returns the original error.
func (p *ImageProcessor) failProcessing(ctx context.Context, avatarID string, processErr error) error {
	p.log.Error("failed to process avatar", zap.String("avatar_id", avatarID), zap.Error(processErr))

	if updateErr := p.repo.UpdateAvatarStatus(ctx, avatarID, models.AvatarStatusError, "", "", 0, 0); updateErr != nil {
		p.log.Error("failed to update avatar status to error", zap.String("avatar_id", avatarID), zap.Error(updateErr))
	}

	return processErr
}

// downloadAndDecodeImage retrieves an image from S3 and decodes it into memory.
func (p *ImageProcessor) downloadAndDecodeImage(ctx context.Context, s3Key string) (image.Image, string, error) {
	reader, err := p.s3.GetObject(ctx, p.bucket, s3Key)
	if err != nil {
		return nil, "", fmt.Errorf("downloading from s3: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", fmt.Errorf("reading s3 stream: %w", err)
	}

	// imaging.Decode automatically detects format and handles EXIF orientation
	srcImg, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, "", fmt.Errorf("decoding image: %w", err)
	}

	// Determine extension for re-encoding based on original key
	ext := strings.ToLower(filepath.Ext(s3Key))
	if ext == extJPG || ext == extJPEG {
		return srcImg, extJPG, nil
	}

	// Default to PNG for webp and png extensions
	return srcImg, extPNG, nil
}

// processThumbnail resizes the image, encodes it, and uploads it to S3.
func (p *ImageProcessor) processThumbnail(
	ctx context.Context,
	srcImg image.Image,
	ext string,
	sizeStr string,
	userID string,
	avatarID string,
	targetSize int,
) (string, error) {
	dstImg := imaging.Resize(srcImg, targetSize, 0, imaging.Lanczos)

	var buf bytes.Buffer
	var encodeErr error

	if ext == extJPG {
		encodeErr = jpeg.Encode(&buf, dstImg, &jpeg.Options{Quality: 85})
	} else {
		encodeErr = png.Encode(&buf, dstImg)
	}

	if encodeErr != nil {
		return "", fmt.Errorf("encoding %s thumbnail: %w", sizeStr, encodeErr)
	}

	s3Key := fmt.Sprintf("avatars/%s/%s_%s%s", userID, avatarID, sizeStr, ext)
	if err := p.s3.UploadObject(ctx, p.bucket, s3Key, &buf); err != nil {
		return "", fmt.Errorf("uploading %s thumbnail to s3: %w", sizeStr, err)
	}

	return s3Key, nil
}
