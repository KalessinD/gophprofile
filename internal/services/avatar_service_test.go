package services_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/KalessinD/gophprofile/internal/models"
	"github.com/KalessinD/gophprofile/internal/services"
	"github.com/KalessinD/gophprofile/internal/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const testBucket = "test-bucket"

func setupServiceTest(t *testing.T) (*gomock.Controller, *mocks.MockAvatarRepository, *mocks.MockObjectStorage, *mocks.MockAvatarProducer, *services.AvatarService) {
	t.Helper()
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAvatarRepository(ctrl)
	s3 := mocks.NewMockObjectStorage(ctrl)
	prod := mocks.NewMockAvatarProducer(ctrl)
	svc := services.NewAvatarService(repo, s3, prod, testBucket)
	return ctrl, repo, s3, prod, svc
}

func TestNewAvatarService(t *testing.T) {
	ctrl, _, _, _, svc := setupServiceTest(t)
	defer ctrl.Finish()
	assert.NotNil(t, svc)
}

func TestEnsureDependencies_NilRepo(t *testing.T) {
	svc := services.NewAvatarService(nil, nil, nil, testBucket)
	assert.Error(t, svc.EnsureDependencies())
	assert.ErrorIs(t, svc.EnsureDependencies(), services.ErrDependenciesNotFound)
}

func TestCreateAvatar_Success(t *testing.T) {
	ctrl, repo, s3, prod, svc := setupServiceTest(t)
	defer ctrl.Finish()

	s3.EXPECT().UploadObject(gomock.Any(), testBucket, gomock.Any(), gomock.Any()).Return(nil)
	repo.EXPECT().CreateAvatar(gomock.Any(), gomock.Any()).Return(nil)
	prod.EXPECT().PublishAvatarCreatedEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	avatar := &models.Avatar{UserID: "user-1", MimeType: "image/jpeg"}
	err := svc.CreateAvatar(t.Context(), avatar, strings.NewReader("fake-img"))
	require.NoError(t, err)
	assert.Equal(t, models.AvatarStatusProcessing, avatar.Status)
	assert.Contains(t, avatar.OriginalS3Key, "avatars/user-1/")
}

func TestCreateAvatar_DBErrorTriggersS3Cleanup(t *testing.T) {
	ctrl, repo, s3, prod, svc := setupServiceTest(t)
	defer ctrl.Finish()

	s3.EXPECT().UploadObject(gomock.Any(), testBucket, gomock.Any(), gomock.Any()).Return(nil)
	repo.EXPECT().CreateAvatar(gomock.Any(), gomock.Any()).Return(errors.New("db down"))
	// Expect cleanup call
	s3.EXPECT().DeleteObject(gomock.Any(), testBucket, gomock.Any()).Return(nil)
	// Producer should NOT be called
	prod.EXPECT().PublishAvatarCreatedEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	err := svc.CreateAvatar(t.Context(), &models.Avatar{UserID: "user-1"}, strings.NewReader("img"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "saving avatar metadata to db")
}

func TestGetAvatarFileStream_OriginalSize(t *testing.T) {
	ctrl, repo, s3, _, svc := setupServiceTest(t)
	defer ctrl.Finish()

	avatarID := "av-1"
	avatar := &models.Avatar{ID: avatarID, OriginalS3Key: "orig.png", MimeType: "image/png"}
	fakeImg := io.NopCloser(strings.NewReader("data"))

	repo.EXPECT().GetAvatarByID(gomock.Any(), avatarID).Return(avatar, nil)
	s3.EXPECT().GetObject(gomock.Any(), testBucket, "orig.png").Return(fakeImg, nil)

	stream, returnedAvatar, err := svc.GetAvatarFileStream(t.Context(), avatarID, models.SizeOriginal)
	require.NoError(t, err)
	assert.NotNil(t, stream)
	assert.Equal(t, avatarID, returnedAvatar.ID)
}

func TestGetAvatarFileStream_ThumbnailNotReady(t *testing.T) {
	ctrl, repo, s3, _, svc := setupServiceTest(t)
	defer ctrl.Finish()

	avatarID := "av-2"
	avatar := &models.Avatar{ID: avatarID, Thumbnail100S3Key: nil} // Not ready

	repo.EXPECT().GetAvatarByID(gomock.Any(), avatarID).Return(avatar, nil)
	// S3 should NOT be called
	s3.EXPECT().GetObject(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	_, _, err := svc.GetAvatarFileStream(t.Context(), avatarID, models.Size100x100)
	require.Error(t, err)
	assert.ErrorIs(t, err, services.ErrThumbnailNotReady)
}

func TestGetAvatarFileStream_FallbackToOriginal(t *testing.T) {
	ctrl, repo, s3, _, svc := setupServiceTest(t)
	defer ctrl.Finish()

	avatarID := "av-3"
	avatar := &models.Avatar{ID: avatarID, OriginalS3Key: "orig.jpg", MimeType: "image/jpeg"}
	fakeImg := io.NopCloser(strings.NewReader("data"))

	repo.EXPECT().GetAvatarByID(gomock.Any(), avatarID).Return(avatar, nil)
	// Unsupported size should fallback to original key
	s3.EXPECT().GetObject(gomock.Any(), testBucket, "orig.jpg").Return(fakeImg, nil)

	stream, _, err := svc.GetAvatarFileStream(t.Context(), avatarID, "500x500")
	require.NoError(t, err)
	assert.NotNil(t, stream)
}

func TestSoftDeleteAvatar_Forbidden(t *testing.T) {
	ctrl, repo, _, _, svc := setupServiceTest(t)
	defer ctrl.Finish()

	repo.EXPECT().GetAvatarByID(gomock.Any(), "av-1").Return(&models.Avatar{ID: "av-1", UserID: "owner"}, nil)
	// Soft delete should not be called
	repo.EXPECT().SoftDeleteAvatar(gomock.Any(), gomock.Any()).Times(0)

	err := svc.SoftDeleteAvatar(t.Context(), "av-1", "intruder")
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrAvatarForbidden)
}

func TestHardDeleteAvatar_Success(t *testing.T) {
	ctrl, repo, s3, _, svc := setupServiceTest(t)
	defer ctrl.Finish()

	t100 := "t100.png"
	t300 := "t300.png"
	avatar := &models.Avatar{
		ID:                "av-1",
		OriginalS3Key:     "orig.png",
		Thumbnail100S3Key: &t100,
		Thumbnail300S3Key: &t300,
	}

	repo.EXPECT().GetAvatarByID(gomock.Any(), "av-1").Return(avatar, nil)
	s3.EXPECT().DeleteObject(gomock.Any(), testBucket, "orig.png").Return(nil)
	s3.EXPECT().DeleteObject(gomock.Any(), testBucket, "t100.png").Return(nil)
	s3.EXPECT().DeleteObject(gomock.Any(), testBucket, "t300.png").Return(nil)
	repo.EXPECT().HardDeleteAvatar(gomock.Any(), "av-1").Return(nil)

	err := svc.HardDeleteAvatar(t.Context(), "av-1")
	require.NoError(t, err)
}

func TestGetAvatarByID_Success(t *testing.T) {
	ctrl, repo, _, _, svc := setupServiceTest(t)
	defer ctrl.Finish()

	expectedAvatar := &models.Avatar{ID: "id-1", UserID: "user-1"}
	repo.EXPECT().GetAvatarByID(gomock.Any(), "id-1").Return(expectedAvatar, nil)

	result, err := svc.GetAvatarByID(t.Context(), "id-1")
	require.NoError(t, err)
	assert.Equal(t, expectedAvatar, result)
}

func TestGetAvatarByID_RepoError(t *testing.T) {
	ctrl, repo, _, _, svc := setupServiceTest(t)
	defer ctrl.Finish()

	repo.EXPECT().GetAvatarByID(gomock.Any(), "id-1").Return(nil, models.ErrAvatarNotFound)

	result, err := svc.GetAvatarByID(t.Context(), "id-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, models.ErrAvatarNotFound)
	assert.Nil(t, result)
}

func TestGetAvatarsByUserID_Success(t *testing.T) {
	ctrl, repo, _, _, svc := setupServiceTest(t)
	defer ctrl.Finish()

	expectedList := []*models.Avatar{{ID: "id-1"}, {ID: "id-2"}}
	repo.EXPECT().GetAvatarsByUserID(gomock.Any(), "user-1").Return(expectedList, nil)

	result, err := svc.GetAvatarsByUserID(t.Context(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, expectedList, result)
}
