//go:generate mockgen -source=storage.go -destination=mocks/mock_storage.go -package=mocks
package s3

import (
	"context"
	"fmt"
	"io"

	"github.com/KalessinD/gophprofile/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

type (
	// Storage implements ObjectStorageInterface using MinIO SDK.
	Storage struct {
		client ClientInterface
		log    *zap.Logger
	}

	// ClientInterface defines the methods of the S3 client that we need to mock in tests.
	ClientInterface interface {
		PutObject(ctx context.Context, bucketName string, objectName string, reader io.Reader, size int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
		GetObject(ctx context.Context, bucketName string, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
		RemoveObject(ctx context.Context, bucketName string, objectName string, opts minio.RemoveObjectOptions) error
	}
)

// NewS3Storage initializes a new MinIO client and checks connection.
func NewS3Storage(ctx context.Context, cfg *config.ServerConfig, log *zap.Logger) (*Storage, error) {
	client, err := minio.New(cfg.S3ListenAddr, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: cfg.S3UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.S3Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to check S3 bucket: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("required S3 bucket '%s' does not exist", cfg.S3Bucket)
	}

	log.Debug("S3 storage initialized successfully")

	return &Storage{
		client: client,
		log:    log,
	}, nil
}

// UploadObject streams an object from reader directly to the S3 bucket.
func (s *Storage) UploadObject(ctx context.Context, bucket string, objectKey string, reader io.Reader) error {
	_, err := s.client.PutObject(ctx, bucket, objectKey, reader, -1, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to upload object to S3: %w", err)
	}

	s.log.Info("Successfully uploaded object", zap.String("bucket", bucket), zap.String("key", objectKey))

	return nil
}

// GetObject retrieves an object from S3.
func (s *Storage) GetObject(ctx context.Context, bucket string, objectKey string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	return obj, nil
}

// DeleteObject removes an object from S3.
func (s *Storage) DeleteObject(ctx context.Context, bucket string, objectKey string) error {
	err := s.client.RemoveObject(ctx, bucket, objectKey, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object from S3: %w", err)
	}

	s.log.Info("Successfully deleted object", zap.String("bucket", bucket), zap.String("key", objectKey))
	return nil
}
