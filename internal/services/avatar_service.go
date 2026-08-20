package services

import (
	"context"
	"errors"
	"io"

	"github.com/KalessinD/gophprofile/internal/models"
)

type (
	AvatarRepository interface {
		CreateAvatar(ctx context.Context, avatar *models.Avatar) error
		GetAvatarByID(ctx context.Context, avatarID string) (*models.Avatar, error)
		GetAvatarsByUserID(ctx context.Context, userID string) ([]*models.Avatar, error)
		SoftDeleteAvatar(ctx context.Context, avatarID string) error
		HardDeleteAvatar(ctx context.Context, avatarID string) error
		UpdateAvatarStatus(ctx context.Context, avatarID string, status string, thumbnail100S3Key string, thumbnail300S3Key string, width int, height int) error
		GetObject(ctx context.Context, bucket string, objectKey string) (io.ReadCloser, error)
		DeleteObject(ctx context.Context, bucket string, objectKey string) error
		PublishAvatarCreatedEvent(ctx context.Context, avatarID string, userID string, s3Key string) error
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

	AvatarService struct {
		repo     AvatarRepository
		s3       ObjectStorage
		producer AvatarProducer
	}
)

var ErrDependenciesNotFound = errors.New("dependencies are not initialized")

// NewAvatarService creates a new instance of AvatarService.
func NewAvatarService(repo AvatarRepository, s3 ObjectStorage, producer AvatarProducer) *AvatarService {
	return &AvatarService{
		repo:     repo,
		s3:       s3,
		producer: producer,
	}
}

func (s *AvatarService) EnsureDependencies() error {
	if s.repo == nil || s.s3 == nil || s.producer == nil {
		return ErrDependenciesNotFound
	}
	return nil
}

func (s *AvatarService) CreateAvatar(ctx context.Context, avatar *models.Avatar) error {
	// TODO: implement
	return nil
}

func (s *AvatarService) GetAvatarByID(ctx context.Context, avatarID string) (*models.Avatar, error) {
	// TODO: implement
	var err error
	return nil, err
}

func (s *AvatarService) GetAvatarsByUserID(ctx context.Context, userID string) ([]*models.Avatar, error) {
	// TODO: implement
	return nil, nil
}

func (s *AvatarService) SoftDeleteAvatar(ctx context.Context, avatarID string) error {
	// TODO: implement
	return nil
}

func (s *AvatarService) HardDeleteAvatar(ctx context.Context, avatarID string) error {
	// TODO: implement
	return nil
}

func (s *AvatarService) UpdateAvatarStatus(ctx context.Context, avatarID string, status string, thumbnail100S3Key string, thumbnail300S3Key string, width int, height int) error {
	// TODO: implement
	return nil
}

func (s *AvatarService) GetObject(ctx context.Context, bucket string, objectKey string) (io.ReadCloser, error) {
	// TODO: implement
	return nil, nil
}

func (s *AvatarService) DeleteObject(ctx context.Context, bucket string, objectKey string) error {
	// TODO: implement
	return nil
}

func (s *AvatarService) PublishAvatarCreatedEvent(ctx context.Context, avatarID string, userID string, s3Key string) error {
	// TODO: implement
	return nil
}
