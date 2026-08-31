package worker // nolint: testpackage

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"strings"
	"testing"

	"github.com/KalessinD/gophprofile/internal/broker"
	"github.com/KalessinD/gophprofile/internal/logger"
	"github.com/KalessinD/gophprofile/internal/metrics"
	"github.com/KalessinD/gophprofile/internal/models"
	"github.com/KalessinD/gophprofile/internal/worker/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	testBucket = "test-bucket"
)

func TestMain(m *testing.M) {
	_ = metrics.Init(context.Background())
	m.Run()
}

// setupProcessorTest is a helper to initialize mocks and the processor.
func setupProcessorTest(t *testing.T) (*gomock.Controller, *mocks.MockAvatarRepository, *mocks.MockObjectStorage, *ImageProcessor) {
	t.Helper()
	ctrl := gomock.NewController(t)

	repoMock := mocks.NewMockAvatarRepository(ctrl)
	s3Mock := mocks.NewMockObjectStorage(ctrl)
	proc := NewImageProcessor(repoMock, s3Mock, testBucket, logger.NewNopLogger())

	return ctrl, repoMock, s3Mock, proc
}

// createTestImage generates a minimal valid 10x10 JPEG image.
func createTestImage(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	err := jpeg.Encode(&buf, img, nil)
	require.NoError(t, err)
	return buf.Bytes()
}

func TestProcessAvatar_Success(t *testing.T) {
	ctrl, repoMock, s3Mock, proc := setupProcessorTest(t)
	defer ctrl.Finish()

	imgBytes := createTestImage(t)
	event := &broker.AvatarEvent{AvatarID: "av-1", UserID: "user-1", S3Key: "orig.jpg"}

	repoMock.EXPECT().
		GetAvatarByID(gomock.Any(), event.AvatarID).
		Return(&models.Avatar{ID: event.AvatarID, Status: models.AvatarStatusProcessing, OriginalS3Key: "orig.jpg"}, nil)

	s3Mock.EXPECT().
		GetObject(gomock.Any(), testBucket, "orig.jpg").
		Return(io.NopCloser(bytes.NewReader(imgBytes)), nil)

	// Expect uploads for 100x100 and 300x300
	s3Mock.EXPECT().UploadObject(gomock.Any(), testBucket, gomock.Any(), gomock.Any()).Return(nil).Times(2)

	// Expect final status update to ready
	repoMock.EXPECT().
		UpdateAvatarStatus(gomock.Any(), event.AvatarID, models.AvatarStatusReady, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	err := proc.ProcessAvatar(t.Context(), event)
	require.NoError(t, err)
}

func TestProcessAvatar_SkipIfNotProcessing(t *testing.T) {
	ctrl, repoMock, s3Mock, proc := setupProcessorTest(t)
	defer ctrl.Finish()

	event := &broker.AvatarEvent{AvatarID: "av-1", UserID: "user-1"}

	repoMock.EXPECT().
		GetAvatarByID(gomock.Any(), event.AvatarID).
		Return(&models.Avatar{ID: event.AvatarID, Status: models.AvatarStatusReady}, nil)

	// S3 should strictly not be called
	s3Mock.EXPECT().GetObject(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	err := proc.ProcessAvatar(t.Context(), event)
	require.NoError(t, err)
}

func TestProcessAvatar_FailOnS3Download(t *testing.T) {
	ctrl, repoMock, s3Mock, proc := setupProcessorTest(t)
	defer ctrl.Finish()

	expectedErr := errors.New("s3 connection lost")
	event := &broker.AvatarEvent{AvatarID: "av-1", UserID: "user-1"}

	repoMock.EXPECT().
		GetAvatarByID(gomock.Any(), event.AvatarID).
		Return(&models.Avatar{ID: event.AvatarID, Status: models.AvatarStatusProcessing, OriginalS3Key: "orig.jpg"}, nil)

	s3Mock.EXPECT().
		GetObject(gomock.Any(), testBucket, "orig.jpg").
		Return(nil, expectedErr)

	// Expect fallback to error status
	repoMock.EXPECT().
		UpdateAvatarStatus(gomock.Any(), event.AvatarID, models.AvatarStatusError, "", "", 0, 0).
		Return(nil)

	err := proc.ProcessAvatar(t.Context(), event)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "downloading from s3")
}

func TestDownloadAndDecodeImage_InvalidData(t *testing.T) {
	ctrl, _, s3Mock, proc := setupProcessorTest(t)
	defer ctrl.Finish()

	s3Mock.EXPECT().
		GetObject(gomock.Any(), testBucket, "bad.jpg").
		Return(io.NopCloser(strings.NewReader("this is not an image")), nil)

	_, _, err := proc.downloadAndDecodeImage(t.Context(), "bad.jpg")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decoding image")
}

func TestProcessThumbnail_PNGFormat(t *testing.T) {
	ctrl, _, s3Mock, proc := setupProcessorTest(t)
	defer ctrl.Finish()

	img := image.NewRGBA(image.Rect(0, 0, 50, 50))

	s3Mock.EXPECT().
		UploadObject(gomock.Any(), testBucket, "avatars/user-1/av-1_100x100.png", gomock.Any()).
		Return(nil)

	key, err := proc.processThumbnail(t.Context(), img, ".png", "100x100", "user-1", "av-1", 100)
	require.NoError(t, err)
	assert.Equal(t, "avatars/user-1/av-1_100x100.png", key)
}
