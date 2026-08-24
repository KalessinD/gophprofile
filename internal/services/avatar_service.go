//go:generate mockgen -source=avatar_service.go -destination=mocks/mock_avatar_service.gen.go -package=mocks
package services

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/KalessinD/gophprofile/internal/models"
	"github.com/google/uuid"
)

var (
	ErrDependenciesNotFound = errors.New("dependencies are not initialized")
	ErrThumbnailNotReady    = errors.New("requested thumbnail is not ready yet")
)

type (
	// AvatarRepository defines the contract for metadata persistence in PostgreSQL.
	AvatarRepository interface {
		CreateAvatar(ctx context.Context, avatar *models.Avatar) error
		GetAvatarByID(ctx context.Context, avatarID string) (*models.Avatar, error)
		GetAvatarsByUserID(ctx context.Context, userID string) ([]*models.Avatar, error)
		SoftDeleteAvatar(ctx context.Context, avatarID string) error
		HardDeleteAvatar(ctx context.Context, avatarID string) error
		UpdateAvatarStatus(ctx context.Context, avatarID string, status string, thumbnail100S3Key string, thumbnail300S3Key string, width int, height int) error
	}

	// ObjectStorage defines the contract for S3 operations.
	ObjectStorage interface {
		UploadObject(ctx context.Context, bucket string, objectKey string, reader io.Reader) error
		GetObject(ctx context.Context, bucket string, objectKey string) (io.ReadCloser, error)
		DeleteObject(ctx context.Context, bucket string, objectKey string) error
	}

	// AvatarProducer defines the contract for publishing events to Kafka.
	AvatarProducer interface {
		PublishAvatarCreatedEvent(ctx context.Context, avatarID string, userID string, s3Key string) error
	}

	// AvatarService handles the business logic for avatars.
	AvatarService struct {
		repo     AvatarRepository
		s3       ObjectStorage
		producer AvatarProducer
		bucket   string
	}
)

// NewAvatarService creates a new instance of AvatarService.
func NewAvatarService(repo AvatarRepository, s3 ObjectStorage, producer AvatarProducer, bucket string) *AvatarService {
	return &AvatarService{
		repo:     repo,
		s3:       s3,
		producer: producer,
		bucket:   bucket,
	}
}

// EnsureDependencies checks if all required dependencies are initialized.
func (s *AvatarService) EnsureDependencies() error {
	if s.repo == nil || s.s3 == nil || s.producer == nil {
		return ErrDependenciesNotFound
	}
	return nil
}

// CreateAvatar handles the business logic for creating a new avatar.
func (s *AvatarService) CreateAvatar(ctx context.Context, avatar *models.Avatar, fileReader io.Reader) error {
	avatar.ID = uuid.New().String()
	avatar.Status = models.AvatarStatusProcessing
	avatar.OriginalS3Key = fmt.Sprintf("avatars/%s/%s", avatar.UserID, avatar.ID)

	// Upload original file to S3
	if err := s.s3.UploadObject(ctx, s.bucket, avatar.OriginalS3Key, fileReader); err != nil {
		return fmt.Errorf("uploading original to s3: %w", err)
	}

	// Save metadata to DB
	if err := s.repo.CreateAvatar(ctx, avatar); err != nil {
		// Attempt to cleanup S3 object if DB save fails
		if delErr := s.s3.DeleteObject(ctx, s.bucket, avatar.OriginalS3Key); delErr != nil {
			// In a real app, log this error: log.Error("failed to cleanup s3 object after db error", zap.Error(delErr))
			_ = delErr
		}
		return fmt.Errorf("saving avatar metadata to db: %w", err)
	}

	// Publish event to Kafka for async processing
	if err := s.producer.PublishAvatarCreatedEvent(ctx, avatar.ID, avatar.UserID, avatar.OriginalS3Key); err != nil {
		return fmt.Errorf("publishing avatar created event: %w", err)
	}

	return nil
}

// GetAvatarByID retrieves avatar metadata by its ID.
func (s *AvatarService) GetAvatarByID(ctx context.Context, avatarID string) (*models.Avatar, error) {
	avatar, err := s.repo.GetAvatarByID(ctx, avatarID)
	if err != nil {
		return nil, fmt.Errorf("getting avatar by id from repo: %w", err)
	}
	return avatar, nil
}

// GetAvatarsByUserID retrieves a list of avatar metadata for a specific user.
func (s *AvatarService) GetAvatarsByUserID(ctx context.Context, userID string) ([]*models.Avatar, error) {
	avatars, err := s.repo.GetAvatarsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting avatars by user id from repo: %w", err)
	}
	return avatars, nil
}

// SoftDeleteAvatar performs a soft delete of an avatar after verifying user ownership.
func (s *AvatarService) SoftDeleteAvatar(ctx context.Context, avatarID string, userID string) error {
	avatar, err := s.repo.GetAvatarByID(ctx, avatarID)
	if err != nil {
		return fmt.Errorf("failed to get avatar for deletion: %w", err)
	}

	if avatar.UserID != userID {
		return models.ErrAvatarForbidden
	}

	if err := s.repo.SoftDeleteAvatar(ctx, avatarID); err != nil {
		return fmt.Errorf("failed to soft delete avatar in db: %w", err)
	}

	return nil
}

// HardDeleteAvatar permanently deletes an avatar from the database and S3.
func (s *AvatarService) HardDeleteAvatar(ctx context.Context, avatarID string) error {
	avatar, err := s.repo.GetAvatarByID(ctx, avatarID)
	if err != nil {
		return fmt.Errorf("getting avatar for hard delete: %w", err)
	}

	// Delete original from S3
	if err := s.s3.DeleteObject(ctx, s.bucket, avatar.OriginalS3Key); err != nil {
		return fmt.Errorf("deleting original s3 object: %w", err)
	}

	// Delete thumbnails from S3 if they exist
	if avatar.Thumbnail100S3Key != nil && *avatar.Thumbnail100S3Key != "" {
		if err := s.s3.DeleteObject(ctx, s.bucket, *avatar.Thumbnail100S3Key); err != nil {
			return fmt.Errorf("deleting 100x100 s3 object: %w", err)
		}
	}
	if avatar.Thumbnail300S3Key != nil && *avatar.Thumbnail300S3Key != "" {
		if err := s.s3.DeleteObject(ctx, s.bucket, *avatar.Thumbnail300S3Key); err != nil {
			return fmt.Errorf("deleting 300x300 s3 object: %w", err)
		}
	}

	// Delete metadata from DB
	if err := s.repo.HardDeleteAvatar(ctx, avatarID); err != nil {
		return fmt.Errorf("hard deleting avatar from db: %w", err)
	}

	return nil
}

// UpdateAvatarStatus updates the processing status, thumbnail keys, and dimensions of an avatar.
func (s *AvatarService) UpdateAvatarStatus(
	ctx context.Context,
	avatarID string,
	status string,
	thumbnail100S3Key string,
	thumbnail300S3Key string,
	width int,
	height int,
) error {
	if err := s.repo.UpdateAvatarStatus(ctx, avatarID, status, thumbnail100S3Key, thumbnail300S3Key, width, height); err != nil {
		return fmt.Errorf("updating avatar status in repository: %w", err)
	}
	return nil
}

// GetAvatarFileStream retrieves an io.ReadCloser for the avatar file from S3,
// along with its metadata. It resolves the correct S3 key based on the requested size.
func (s *AvatarService) GetAvatarFileStream(ctx context.Context, avatarID string, requestedSize string) (io.ReadCloser, *models.Avatar, error) {
	avatar, err := s.repo.GetAvatarByID(ctx, avatarID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get avatar metadata: %w", err)
	}

	s3Key, err := s.resolveS3Key(avatar, requestedSize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve s3 key for size %s: %w", requestedSize, err)
	}

	stream, err := s.s3.GetObject(ctx, s.bucket, s3Key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get avatar object from s3: %w", err)
	}

	return stream, avatar, nil
}

// resolveS3Key determines the correct S3 object key based on the requested size parameter.
func (s *AvatarService) resolveS3Key(avatar *models.Avatar, requestedSize string) (string, error) {
	switch requestedSize {
	case models.Size100x100:
		if avatar.Thumbnail100S3Key == nil || *avatar.Thumbnail100S3Key == "" {
			return "", ErrThumbnailNotReady
		}
		return *avatar.Thumbnail100S3Key, nil
	case models.Size300x300:
		if avatar.Thumbnail300S3Key == nil || *avatar.Thumbnail300S3Key == "" {
			return "", ErrThumbnailNotReady
		}
		return *avatar.Thumbnail300S3Key, nil
	case models.SizeOriginal:
		return avatar.OriginalS3Key, nil
	default:
		// Fallback to original if an unsupported size is provided
		return avatar.OriginalS3Key, nil
	}
}
