package server_test

import (
	"errors"
	"io"
	"testing"
	"time"

	"github.com/KalessinD/gophprofile/internal/circuitbreaker"
	"github.com/KalessinD/gophprofile/internal/config"
	srv "github.com/KalessinD/gophprofile/internal/server"
	mocks "github.com/KalessinD/gophprofile/internal/services/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCBRepository_GetAvatarByID_CircuitOpen(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockAvatarRepository(ctrl)
	cb := circuitbreaker.NewCircuitBreaker(&config.CircuitBreaker{FailureThreshold: 1, ResetTimeout: 10 * time.Millisecond})
	cbRepo := srv.NewCBRepository(mockRepo, cb)

	expectedErr := errors.New("db error")
	mockRepo.EXPECT().GetAvatarByID(gomock.Any(), "1").Return(nil, expectedErr).Times(1)

	_, err := cbRepo.GetAvatarByID(ctx, "1")
	assert.ErrorIs(t, err, expectedErr)

	// Now should be open
	_, err = cbRepo.GetAvatarByID(ctx, "1")
	assert.ErrorIs(t, err, circuitbreaker.ErrCircuitOpen)
}

func TestCBObjectStorage_GetObject_Success(t *testing.T) {
	ctx := t.Context()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStorage := mocks.NewMockObjectStorage(ctrl)
	cb := circuitbreaker.NewCircuitBreaker(&config.CircuitBreaker{FailureThreshold: 1, ResetTimeout: 10 * time.Millisecond})
	cbStorage := srv.NewCBObjectStorage(mockStorage, cb)

	mockReader := io.NopCloser(io.Reader(nil))
	mockStorage.EXPECT().GetObject(gomock.Any(), "bucket", "key").Return(mockReader, nil).Times(1)

	reader, err := cbStorage.GetObject(ctx, "bucket", "key")
	assert.NoError(t, err)
	assert.Equal(t, mockReader, reader)
}
