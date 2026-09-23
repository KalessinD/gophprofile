package server

import (
	"context"
	"io"

	"github.com/KalessinD/gophprofile/internal/circuitbreaker"
	"github.com/KalessinD/gophprofile/internal/models"
	"github.com/KalessinD/gophprofile/internal/services"
)

// CBRepository wraps AvatarRepository with circuit breaker logic.
type CBRepository struct {
	repo services.AvatarRepository
	cb   *circuitbreaker.CircuitBreaker
}

// NewCBRepository creates a new CBRepository instance.
func NewCBRepository(repo services.AvatarRepository, cb *circuitbreaker.CircuitBreaker) *CBRepository {
	return &CBRepository{repo: repo, cb: cb}
}

// CreateAvatar wraps the underlying method with circuit breaker.
func (c *CBRepository) CreateAvatar(ctx context.Context, avatar *models.Avatar) error {
	return c.cb.Execute(ctx, func() error {
		return c.repo.CreateAvatar(ctx, avatar)
	})
}

// GetAvatarByID wraps the underlying method with circuit breaker.
func (c *CBRepository) GetAvatarByID(ctx context.Context, avatarID string) (*models.Avatar, error) {
	var avatar *models.Avatar
	var err error
	execErr := c.cb.Execute(ctx, func() error {
		avatar, err = c.repo.GetAvatarByID(ctx, avatarID)
		return err
	})
	if execErr != nil {
		return nil, execErr
	}
	return avatar, nil
}

// GetAvatarsByUserID wraps the underlying method with circuit breaker.
func (c *CBRepository) GetAvatarsByUserID(ctx context.Context, userID string) ([]*models.Avatar, error) {
	var avatars []*models.Avatar
	var err error
	execErr := c.cb.Execute(ctx, func() error {
		avatars, err = c.repo.GetAvatarsByUserID(ctx, userID)
		return err
	})
	if execErr != nil {
		return nil, execErr
	}
	return avatars, nil
}

// SoftDeleteAvatar wraps the underlying method with circuit breaker.
func (c *CBRepository) SoftDeleteAvatar(ctx context.Context, avatarID string) error {
	return c.cb.Execute(ctx, func() error {
		return c.repo.SoftDeleteAvatar(ctx, avatarID)
	})
}

// HardDeleteAvatar wraps the underlying method with circuit breaker.
func (c *CBRepository) HardDeleteAvatar(ctx context.Context, avatarID string) error {
	return c.cb.Execute(ctx, func() error {
		return c.repo.HardDeleteAvatar(ctx, avatarID)
	})
}

// UpdateAvatarStatus wraps the underlying method with circuit breaker.
func (c *CBRepository) UpdateAvatarStatus(
	ctx context.Context,
	avatarID string,
	status string,
	thumbnail100S3Key string,
	thumbnail300S3Key string,
	width int,
	height int,
) error {
	return c.cb.Execute(ctx, func() error {
		return c.repo.UpdateAvatarStatus(ctx, avatarID, status, thumbnail100S3Key, thumbnail300S3Key, width, height)
	})
}

// CBObjectStorage wraps ObjectStorage with circuit breaker logic.
type CBObjectStorage struct {
	storage services.ObjectStorage
	cb      *circuitbreaker.CircuitBreaker
}

// NewCBObjectStorage creates a new CBObjectStorage instance.
func NewCBObjectStorage(storage services.ObjectStorage, cb *circuitbreaker.CircuitBreaker) *CBObjectStorage {
	return &CBObjectStorage{storage: storage, cb: cb}
}

// UploadObject wraps the underlying method with circuit breaker.
func (c *CBObjectStorage) UploadObject(ctx context.Context, bucket string, objectKey string, reader io.Reader) error {
	return c.cb.Execute(ctx, func() error {
		return c.storage.UploadObject(ctx, bucket, objectKey, reader)
	})
}

// GetObject wraps the underlying method with circuit breaker.
func (c *CBObjectStorage) GetObject(ctx context.Context, bucket string, objectKey string) (io.ReadCloser, error) {
	var stream io.ReadCloser
	var err error
	execErr := c.cb.Execute(ctx, func() error {
		stream, err = c.storage.GetObject(ctx, bucket, objectKey)
		return err
	})
	if execErr != nil {
		return nil, execErr
	}
	return stream, nil
}

// DeleteObject wraps the underlying method with circuit breaker.
func (c *CBObjectStorage) DeleteObject(ctx context.Context, bucket string, objectKey string) error {
	return c.cb.Execute(ctx, func() error {
		return c.storage.DeleteObject(ctx, bucket, objectKey)
	})
}

// CBAvatarProducer wraps AvatarProducer with circuit breaker logic.
type CBAvatarProducer struct {
	producer services.AvatarProducer
	cb       *circuitbreaker.CircuitBreaker
}

// NewCBAvatarProducer creates a new CBAvatarProducer instance.
func NewCBAvatarProducer(producer services.AvatarProducer, cb *circuitbreaker.CircuitBreaker) *CBAvatarProducer {
	return &CBAvatarProducer{producer: producer, cb: cb}
}

// PublishAvatarCreatedEvent wraps the underlying method with circuit breaker.
func (c *CBAvatarProducer) PublishAvatarCreatedEvent(ctx context.Context, avatarID string, userID string, s3Key string) error {
	return c.cb.Execute(ctx, func() error {
		return c.producer.PublishAvatarCreatedEvent(ctx, avatarID, userID, s3Key)
	})
}
