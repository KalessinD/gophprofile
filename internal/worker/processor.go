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
	"github.com/KalessinD/gophprofile/internal/logger"
	"github.com/KalessinD/gophprofile/internal/metrics"
	"github.com/KalessinD/gophprofile/internal/models"
	"github.com/disintegration/imaging"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
)

const (
	extJPEG = ".jpeg"
	extJPG  = ".jpg"
	extPNG  = ".png"

	tracerName = "gophprofile-worker"
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
		log    logger.Logger
	}
)

// NewImageProcessor creates a new ImageProcessor instance.
func NewImageProcessor(repo AvatarRepository, s3 ObjectStorage, bucket string, log logger.Logger) *ImageProcessor {
	return &ImageProcessor{
		repo:   repo,
		s3:     s3,
		bucket: bucket,
		log:    log,
	}
}

// ProcessAvatar executes the full pipeline for a single avatar event.
func (p *ImageProcessor) ProcessAvatar(ctx context.Context, event *broker.AvatarEvent) error {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "worker.process_avatar")
	defer span.End()

	span.SetAttributes(
		attribute.String("worker.avatar_id", event.AvatarID),
		attribute.String("worker.user_id", event.UserID),
	)

	avatar, err := p.repo.GetAvatarByID(ctx, event.AvatarID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to fetch avatar metadata")
		return fmt.Errorf("fetching avatar metadata: %w", err)
	}

	// Skip if already processed or explicitly failed
	if avatar.Status != models.AvatarStatusProcessing {
		p.log.Info("skipping avatar, status is not processing", "avatar_id", event.AvatarID, "status", avatar.Status)
		return nil
	}

	srcImg, ext, err := p.downloadAndDecodeImage(ctx, avatar.OriginalS3Key)
	if srcImg == nil || err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to download/decode image")
		return p.failProcessing(ctx, event.AvatarID, err)
	}

	bounds := srcImg.Bounds()
	originalWidth, originalHeight := bounds.Dx(), bounds.Dy()

	thumb100Key, err := p.processThumbnail(ctx, srcImg, ext, "100x100", event.UserID, event.AvatarID, 100)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to process 100x100 thumbnail")
		return p.failProcessing(ctx, event.AvatarID, err)
	}

	thumb300Key, err := p.processThumbnail(ctx, srcImg, ext, "300x300", event.UserID, event.AvatarID, 300)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to process 300x300 thumbnail")
		return p.failProcessing(ctx, event.AvatarID, err)
	}

	if err := p.repo.UpdateAvatarStatus(ctx, event.AvatarID, models.AvatarStatusReady, thumb100Key, thumb300Key, originalWidth, originalHeight); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to mark avatar as ready")
		return fmt.Errorf("marking avatar as ready: %w", err)
	}

	// Record business metric: add file size to storage gauge upon successful processing
	m := metrics.Instance()
	m.StorageBytes.Add(ctx, avatar.FileSize, metric.WithAttributes(
		attribute.String("user_id", event.UserID),
		attribute.String("avatar_id", event.AvatarID),
	))

	p.log.Info("avatar processed successfully", "avatar_id", event.AvatarID)
	return nil
}

// failProcessing attempts to mark the avatar as failed in the DB and returns the original error.
func (p *ImageProcessor) failProcessing(ctx context.Context, avatarID string, processErr error) error {
	p.log.Error("failed to process avatar", "avatar_id", avatarID, "error", processErr)

	if updateErr := p.repo.UpdateAvatarStatus(ctx, avatarID, models.AvatarStatusError, "", "", 0, 0); updateErr != nil {
		p.log.Error("failed to update avatar status to error", "avatar_id", avatarID, "error", updateErr)
	}

	return processErr
}

// downloadAndDecodeImage retrieves an image from S3 and decodes it into memory.
func (p *ImageProcessor) downloadAndDecodeImage(ctx context.Context, s3Key string) (image.Image, string, error) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, "worker.download_and_decode_image")
	defer span.End()

	span.SetAttributes(attribute.String("worker.s3_key", s3Key))

	reader, err := p.s3.GetObject(ctx, p.bucket, s3Key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, "", fmt.Errorf("downloading from s3: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, "", fmt.Errorf("reading s3 stream: %w", err)
	}

	// imaging.Decode automatically detects format and handles EXIF orientation
	srcImg, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
	ctx, span := otel.Tracer(tracerName).Start(ctx, "worker.process_thumbnail")
	defer span.End()

	span.SetAttributes(
		attribute.String("worker.target_size", sizeStr),
		attribute.String("worker.user_id", userID),
		attribute.String("worker.avatar_id", avatarID),
	)

	dstImg := imaging.Resize(srcImg, targetSize, 0, imaging.Lanczos)

	var buf bytes.Buffer
	var encodeErr error

	if ext == extJPG {
		encodeErr = jpeg.Encode(&buf, dstImg, &jpeg.Options{Quality: 85})
	} else {
		encodeErr = png.Encode(&buf, dstImg)
	}

	if encodeErr != nil {
		span.RecordError(encodeErr)
		span.SetStatus(codes.Error, encodeErr.Error())
		return "", fmt.Errorf("encoding %s thumbnail: %w", sizeStr, encodeErr)
	}

	s3Key := fmt.Sprintf("avatars/%s/%s_%s%s", userID, avatarID, sizeStr, ext)
	if err := p.s3.UploadObject(ctx, p.bucket, s3Key, &buf); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", fmt.Errorf("uploading %s thumbnail to s3: %w", sizeStr, err)
	}

	return s3Key, nil
}
