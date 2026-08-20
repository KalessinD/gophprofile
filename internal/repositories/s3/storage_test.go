package s3 // nolint: testpackage

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/KalessinD/gophprofile/internal/config"
	"github.com/KalessinD/gophprofile/internal/repositories/s3/mocks"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestStorage_UploadObject_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockClientInterface(ctrl)
	testLogger := zap.NewNop()

	storage := &Storage{
		client: mockClient,
		log:    testLogger,
	}

	bucketName := "test-bucket"
	objectKey := "documents/test-file.txt"
	content := "test file content"
	reader := strings.NewReader(content)

	mockClient.EXPECT().
		PutObject(
			gomock.Any(),
			bucketName,
			objectKey,
			gomock.Any(),
			int64(-1),
			minio.PutObjectOptions{},
		).
		Return(minio.UploadInfo{}, nil)

	err := storage.UploadObject(t.Context(), bucketName, objectKey, reader)

	require.NoError(t, err, "UploadObject should not return an error on success")
}

func TestStorage_UploadObject_PutObjectError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockClientInterface(ctrl)
	testLogger := zap.NewNop()

	storage := &Storage{
		client: mockClient,
		log:    testLogger,
	}

	bucketName := "test-bucket"
	objectKey := "documents/error-file.txt"
	reader := strings.NewReader("some data")
	simulatedError := errors.New("s3 network timeout")

	mockClient.EXPECT().
		PutObject(
			gomock.Any(),
			bucketName,
			objectKey,
			gomock.Any(),
			int64(-1),
			minio.PutObjectOptions{},
		).
		Return(minio.UploadInfo{}, simulatedError)

	err := storage.UploadObject(t.Context(), bucketName, objectKey, reader)

	require.Error(t, err, "UploadObject should return an error when PutObject fails")
	assert.ErrorIs(t, err, simulatedError, "Error should wrap the original S3 client error")
	assert.Contains(t, err.Error(), "failed to upload object to S3", "Error should contain context message")
}

func TestNewS3Storage_ConnectionFailed(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	cfg := &config.ServerConfig{
		S3ListenAddr: "localhost:1", // bad address
		S3AccessKey:  "access",
		S3SecretKey:  "secret",
		S3UseSSL:     false,
	}

	storage, err := NewS3Storage(ctx, cfg, zap.NewNop())
	require.Error(t, err)
	assert.Nil(t, storage)
	assert.Contains(t, err.Error(), "failed to check S3 bucket")
}
