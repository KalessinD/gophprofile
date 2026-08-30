package s3 // nolint: testpackage

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/KalessinD/gophprofile/internal/config"
	"github.com/KalessinD/gophprofile/internal/logger"
	"github.com/KalessinD/gophprofile/internal/repositories/s3/mocks"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const bucketS3 = "test-bucket"

func TestStorage_UploadObject_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockClientInterface(ctrl)
	testLogger := logger.NewNopLogger()

	storage := &Storage{
		client: mockClient,
		log:    testLogger,
	}

	bucketName := bucketS3
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

func TestStorage_GetObject_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockClientInterface(ctrl)
	testLogger := logger.NewNopLogger()

	storage := &Storage{
		client: mockClient,
		log:    testLogger,
	}

	bucketName := bucketS3
	objectKey := "documents/missing.txt"
	simulatedError := errors.New("s3 not found")

	mockClient.EXPECT().
		GetObject(
			gomock.Any(),
			bucketName,
			objectKey,
			minio.GetObjectOptions{},
		).
		Return(nil, simulatedError)

	obj, err := storage.GetObject(t.Context(), bucketName, objectKey)

	require.Error(t, err, "GetObject should return an error when S3 fails")
	assert.Nil(t, obj)
	assert.ErrorIs(t, err, simulatedError)
	assert.Contains(t, err.Error(), "failed to get object from S3")
}

func TestStorage_DeleteObject_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockClientInterface(ctrl)
	testLogger := logger.NewNopLogger()

	storage := &Storage{
		client: mockClient,
		log:    testLogger,
	}

	bucketName := bucketS3
	objectKey := "documents/to-delete.txt"

	mockClient.EXPECT().
		RemoveObject(
			gomock.Any(),
			bucketName,
			objectKey,
			minio.RemoveObjectOptions{},
		).
		Return(nil)

	err := storage.DeleteObject(t.Context(), bucketName, objectKey)

	require.NoError(t, err, "DeleteObject should not return an error on success")
}

func TestStorage_DeleteObject_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockClientInterface(ctrl)
	testLogger := logger.NewNopLogger()

	storage := &Storage{
		client: mockClient,
		log:    testLogger,
	}

	bucketName := bucketS3
	objectKey := "documents/error-delete.txt"
	simulatedError := errors.New("access denied")

	mockClient.EXPECT().
		RemoveObject(
			gomock.Any(),
			bucketName,
			objectKey,
			minio.RemoveObjectOptions{},
		).
		Return(simulatedError)

	err := storage.DeleteObject(t.Context(), bucketName, objectKey)

	require.Error(t, err, "DeleteObject should return an error when S3 fails")
	assert.ErrorIs(t, err, simulatedError)
	assert.Contains(t, err.Error(), "failed to delete object from S3")
}

func TestNewS3Storage_ConnectionFailed(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	cfg := &config.S3{
		ListenAddr: "localhost:1", // bad address
		AccessKey:  "access",
		SecretKey:  "secret",
		UseSSL:     false,
	}

	storage, err := NewS3Storage(ctx, cfg, logger.NewNopLogger())
	require.Error(t, err)
	assert.Nil(t, storage)
	assert.Contains(t, err.Error(), "failed to check S3 bucket")
}
