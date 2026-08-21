package services

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/KalessinD/gophprofile/internal/models"
)

var (
	ErrDependenciesNotFound = errors.New("dependencies are not initialized")
	ErrThumbnailNotReady    = errors.New("requested thumbnail is not ready yet")
)

type (
	// AvatarRepository defines the contract for metadata persistence in PostgreSQL.
	// Strictly follows AGENTS.md: no S3 or Broker methods allowed here.
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
func (s *AvatarService) CreateAvatar(ctx context.Context, avatar *models.Avatar) error {
	// TODO: implement
	return nil
}

// GetAvatarByID retrieves avatar metadata by its ID.
func (s *AvatarService) GetAvatarByID(ctx context.Context, avatarID string) (*models.Avatar, error) {
	// TODO: implement
	return nil, nil
}

// GetAvatarsByUserID retrieves a list of avatar metadata for a specific user.
func (s *AvatarService) GetAvatarsByUserID(ctx context.Context, userID string) ([]*models.Avatar, error) {
	// TODO: implement
	return nil, nil
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

// HardDeleteAvatar permanently deletes an avatar from the database.
func (s *AvatarService) HardDeleteAvatar(ctx context.Context, avatarID string) error {
	// TODO: implement
	return nil
}

// UpdateAvatarStatus updates the processing status and thumbnail keys of an avatar.
func (s *AvatarService) UpdateAvatarStatus(ctx context.Context, avatarID string, status string, thumbnail100S3Key string, thumbnail300S3Key string, width int, height int) error {
	// TODO: implement
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
