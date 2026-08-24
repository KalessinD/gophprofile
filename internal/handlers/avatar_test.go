package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KalessinD/gophprofile/internal/handlers"
	"github.com/KalessinD/gophprofile/internal/middleware"
	"github.com/KalessinD/gophprofile/internal/models"
	"github.com/KalessinD/gophprofile/internal/services"
	"github.com/KalessinD/gophprofile/internal/services/mocks"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

const (
	testBucket = "test-bucket"
	testUserID = "user-123"
)

// setupTestEnv is a helper to initialize mocks, service, and handler.
// nolint: unparam
func setupTestEnv(t *testing.T) (*gomock.Controller, *mocks.MockAvatarRepository, *mocks.MockObjectStorage, *mocks.MockAvatarProducer, *handlers.AvatarHandler) {
	t.Helper()
	ctrl := gomock.NewController(t)

	repoMock := mocks.NewMockAvatarRepository(ctrl)
	s3Mock := mocks.NewMockObjectStorage(ctrl)
	prodMock := mocks.NewMockAvatarProducer(ctrl)

	svc := services.NewAvatarService(repoMock, s3Mock, prodMock, testBucket)

	// Mock URL builder for tests
	mockURLBuilder := func(key string) string { return "http://localhost/" + key }
	h := handlers.NewAvatarHandler(svc, mockURLBuilder)

	return ctrl, repoMock, s3Mock, prodMock, h
}

// addUserAndLoggerToContext injects the X-User-ID into the request context, mimicking the middleware.
// nolint: unparam
func addUserAndLoggerToContext(req *http.Request, userID string) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, userID)
	ctx = context.WithValue(ctx, middleware.LoggerKey, zap.NewNop())
	return req.WithContext(ctx)
}

// addChiURLParams injects URL parameters into the request context, mimicking the Chi router.
func addChiURLParams(req *http.Request, params map[string]string) *http.Request {
	routeCtx := chi.NewRouteContext()
	for key, value := range params {
		routeCtx.URLParams.Add(key, value)
	}
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

// newMultipartRequest is a helper to create a valid multipart upload request.
func newMultipartRequest(t *testing.T, fileName string, content []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("image", fileName)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/avatars", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestUploadAvatar_Success(t *testing.T) {
	ctrl, repoMock, s3Mock, prodMock, h := setupTestEnv(t)
	defer ctrl.Finish()

	s3Mock.EXPECT().
		UploadObject(gomock.Any(), testBucket, gomock.Any(), gomock.Any()).
		Return(nil)

	repoMock.EXPECT().
		CreateAvatar(gomock.Any(), gomock.Any()).
		Return(nil)

	prodMock.EXPECT().
		PublishAvatarCreatedEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil)

	// Fake JPEG header bytes to pass http.DetectContentType validation
	fakeJPEGHeader := []byte("\xFF\xD8\xFF\xE0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00")
	req := newMultipartRequest(t, "image.jpg", fakeJPEGHeader)
	req = addUserAndLoggerToContext(req, testUserID)

	rec := httptest.NewRecorder()
	h.UploadAvatar(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var responseAvatar models.UploadResponse
	err := json.NewDecoder(rec.Body).Decode(&responseAvatar)
	require.NoError(t, err)

	assert.Equal(t, testUserID, responseAvatar.UserID)
	assert.Equal(t, models.AvatarStatusProcessing, responseAvatar.Status)
	assert.NotEmpty(t, responseAvatar.ID)
	assert.Contains(t, responseAvatar.URL, "http://localhost/avatars/"+testUserID+"/")
}

func TestUploadAvatar_InvalidFormat(t *testing.T) {
	ctrl, _, _, _, h := setupTestEnv(t)
	defer ctrl.Finish()

	// Plain text will fail http.DetectContentType image check
	req := newMultipartRequest(t, "document.pdf", []byte("fake-data"))
	req = addUserAndLoggerToContext(req, testUserID)

	rec := httptest.NewRecorder()
	h.UploadAvatar(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid image format")
}

func TestUploadAvatar_MissingFile(t *testing.T) {
	ctrl, _, _, _, h := setupTestEnv(t)
	defer ctrl.Finish()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/avatars", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json") // Wrong content type, no multipart boundary
	req = addUserAndLoggerToContext(req, testUserID)

	rec := httptest.NewRecorder()
	h.UploadAvatar(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "missing file in request")
}

func TestGetAvatar_Success(t *testing.T) {
	ctrl, repoMock, s3Mock, _, h := setupTestEnv(t)
	defer ctrl.Finish()

	avatarID := "av-1"
	fakeImage := io.NopCloser(strings.NewReader("image-bytes"))
	expectedAvatar := &models.Avatar{
		ID:            avatarID,
		OriginalS3Key: "orig.jpg",
		MimeType:      "image/jpeg",
	}

	repoMock.EXPECT().
		GetAvatarByID(gomock.Any(), avatarID).
		Return(expectedAvatar, nil)

	s3Mock.EXPECT().
		GetObject(gomock.Any(), testBucket, "orig.jpg").
		Return(fakeImage, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/avatars/"+avatarID, nil)
	req = addUserAndLoggerToContext(req, testUserID)
	req = addChiURLParams(req, map[string]string{"avatar_id": avatarID})

	rec := httptest.NewRecorder()
	h.GetAvatar(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/jpeg", rec.Header().Get("Content-Type"))
	assert.Equal(t, `"`+avatarID+`"`, rec.Header().Get("ETag"))
	assert.Equal(t, "max-age=86400", rec.Header().Get("Cache-Control"))
}

func TestGetAvatar_NotFound(t *testing.T) {
	ctrl, repoMock, _, _, h := setupTestEnv(t)
	defer ctrl.Finish()

	avatarID := "not-found"

	repoMock.EXPECT().
		GetAvatarByID(gomock.Any(), avatarID).
		Return(nil, models.ErrAvatarNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/avatars/"+avatarID, nil)
	req = addUserAndLoggerToContext(req, testUserID)
	req = addChiURLParams(req, map[string]string{"avatar_id": avatarID})

	rec := httptest.NewRecorder()
	h.GetAvatar(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "avatar not found")
}

func TestGetAvatar_ThumbnailNotReady(t *testing.T) {
	ctrl, repoMock, _, _, h := setupTestEnv(t)
	defer ctrl.Finish()

	avatarID := "av-2"
	expectedAvatar := &models.Avatar{
		ID:                avatarID,
		Thumbnail100S3Key: nil, // Thumbnail not generated yet
	}

	repoMock.EXPECT().
		GetAvatarByID(gomock.Any(), avatarID).
		Return(expectedAvatar, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/avatars/"+avatarID+"?size=100x100", nil)
	req = addUserAndLoggerToContext(req, testUserID)
	req = addChiURLParams(req, map[string]string{"avatar_id": avatarID})

	rec := httptest.NewRecorder()
	h.GetAvatar(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "not ready yet")
}

func TestDeleteAvatar_Success(t *testing.T) {
	ctrl, repoMock, _, _, h := setupTestEnv(t)
	defer ctrl.Finish()

	avatarID := "av-3"

	// SoftDeleteAvatar in service calls GetAvatarByID first to check ownership
	repoMock.EXPECT().
		GetAvatarByID(gomock.Any(), avatarID).
		Return(&models.Avatar{ID: avatarID, UserID: testUserID}, nil)

	repoMock.EXPECT().
		SoftDeleteAvatar(gomock.Any(), avatarID).
		Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/avatars/"+avatarID, nil)
	req = addUserAndLoggerToContext(req, testUserID)
	req = addChiURLParams(req, map[string]string{"avatar_id": avatarID})

	rec := httptest.NewRecorder()
	h.DeleteAvatar(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestDeleteAvatar_Forbidden(t *testing.T) {
	ctrl, repoMock, _, _, h := setupTestEnv(t)
	defer ctrl.Finish()

	avatarID := "av-4"
	ownerID := "user-999" // Different from testUserID

	repoMock.EXPECT().
		GetAvatarByID(gomock.Any(), avatarID).
		Return(&models.Avatar{ID: avatarID, UserID: ownerID}, nil)

	// SoftDeleteAvatar should NOT be called
	repoMock.EXPECT().SoftDeleteAvatar(gomock.Any(), gomock.Any()).Times(0)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/avatars/"+avatarID, nil)
	req = addUserAndLoggerToContext(req, testUserID)
	req = addChiURLParams(req, map[string]string{"avatar_id": avatarID})

	rec := httptest.NewRecorder()
	h.DeleteAvatar(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "you can only delete your own avatars")
}

func TestGetUserAvatar_Success(t *testing.T) {
	ctrl, repoMock, s3Mock, _, h := setupTestEnv(t)
	defer ctrl.Finish()

	userID := "user-1"
	avatarID := "av-5"
	fakeImage := io.NopCloser(strings.NewReader("img-bytes"))

	// Handler first gets the list to find the latest avatar
	repoMock.EXPECT().
		GetAvatarsByUserID(gomock.Any(), userID).
		Return([]*models.Avatar{{ID: avatarID, OriginalS3Key: "orig.png", MimeType: "image/png"}}, nil)

	// Then GetAvatarFileStream internally fetches metadata
	repoMock.EXPECT().
		GetAvatarByID(gomock.Any(), avatarID).
		Return(&models.Avatar{ID: avatarID, OriginalS3Key: "orig.png", MimeType: "image/png"}, nil)

	// And fetches the file from S3
	s3Mock.EXPECT().
		GetObject(gomock.Any(), testBucket, "orig.png").
		Return(fakeImage, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID+"/avatar", nil)
	req = addUserAndLoggerToContext(req, testUserID)
	req = addChiURLParams(req, map[string]string{"user_id": userID})

	rec := httptest.NewRecorder()
	h.GetUserAvatar(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "image/png", rec.Header().Get("Content-Type"))
}

func TestGetUserAvatar_NotFound(t *testing.T) {
	ctrl, repoMock, _, _, h := setupTestEnv(t)
	defer ctrl.Finish()

	userID := "empty-user"

	// Handler checks if list is empty
	repoMock.EXPECT().
		GetAvatarsByUserID(gomock.Any(), userID).
		Return([]*models.Avatar{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID+"/avatar", nil)
	req = addUserAndLoggerToContext(req, testUserID)
	req = addChiURLParams(req, map[string]string{"user_id": userID})

	rec := httptest.NewRecorder()
	h.GetUserAvatar(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "avatar not found")
}

func TestGetUserAvatars_Success(t *testing.T) {
	ctrl, repoMock, _, _, h := setupTestEnv(t)
	defer ctrl.Finish()

	userID := "user-1"

	repoMock.EXPECT().
		GetAvatarsByUserID(gomock.Any(), userID).
		Return([]*models.Avatar{
			{ID: "av-1", UserID: userID},
			{ID: "av-2", UserID: userID},
		}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/"+userID+"/avatars", nil)
	req = addUserAndLoggerToContext(req, testUserID)
	req = addChiURLParams(req, map[string]string{"user_id": userID})

	rec := httptest.NewRecorder()
	h.GetUserAvatars(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "av-1")
	assert.Contains(t, rec.Body.String(), "av-2")
}

func TestGetAvatarMetadata_Success(t *testing.T) {
	ctrl, repoMock, _, _, h := setupTestEnv(t)
	defer ctrl.Finish()

	avatarID := "av-meta-1"
	expectedAvatar := &models.Avatar{
		ID:            avatarID,
		UserID:        testUserID,
		OriginalS3Key: "orig.jpg",
		Status:        models.AvatarStatusReady,
		MimeType:      "image/jpeg",
		FileSize:      2048,
	}

	repoMock.EXPECT().
		GetAvatarByID(gomock.Any(), avatarID).
		Return(expectedAvatar, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/avatars/"+avatarID+"/metadata", nil)
	req = addUserAndLoggerToContext(req, testUserID)
	req = addChiURLParams(req, map[string]string{"avatar_id": avatarID})

	rec := httptest.NewRecorder()
	h.GetAvatarMetadata(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var responseAvatar models.MetadataResponse // Используем DTO
	err := json.NewDecoder(rec.Body).Decode(&responseAvatar)
	require.NoError(t, err)
	assert.Equal(t, avatarID, responseAvatar.ID)
	assert.Len(t, responseAvatar.Thumbnails, 0) // В тесте миниатюры nil
}

func TestGetAvatarMetadata_NotFound(t *testing.T) {
	ctrl, repoMock, _, _, h := setupTestEnv(t)
	defer ctrl.Finish()

	avatarID := "not-found-meta"

	repoMock.EXPECT().
		GetAvatarByID(gomock.Any(), avatarID).
		Return(nil, models.ErrAvatarNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/avatars/"+avatarID+"/metadata", nil)
	req = addUserAndLoggerToContext(req, testUserID)
	req = addChiURLParams(req, map[string]string{"avatar_id": avatarID})

	rec := httptest.NewRecorder()
	h.GetAvatarMetadata(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "avatar not found")
}

func TestDeleteUserAvatar_NotFound(t *testing.T) {
	ctrl, repoMock, _, _, h := setupTestEnv(t)
	defer ctrl.Finish()

	userID := "empty-user-del"

	repoMock.EXPECT().
		GetAvatarsByUserID(gomock.Any(), userID).
		Return([]*models.Avatar{}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID+"/avatar", nil)
	req = addUserAndLoggerToContext(req, testUserID)
	req = addChiURLParams(req, map[string]string{"user_id": userID})

	rec := httptest.NewRecorder()
	h.DeleteUserAvatar(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "avatar not found")
}

func TestDeleteUserAvatar_Success(t *testing.T) {
	ctrl, repoMock, _, _, h := setupTestEnv(t)
	defer ctrl.Finish()

	// Используем testUserID как для URL параметра, так и для владельца аватарок
	userID := testUserID
	avatar1ID := "av-del-1"
	avatar2ID := "av-del-2"

	// Handler gets the list of avatars
	repoMock.EXPECT().
		GetAvatarsByUserID(gomock.Any(), userID).
		Return([]*models.Avatar{{ID: avatar1ID}, {ID: avatar2ID}}, nil)

	// SoftDeleteAvatar loop for avatar 1 (checks ownership internally)
	repoMock.EXPECT().
		GetAvatarByID(gomock.Any(), avatar1ID).
		Return(&models.Avatar{ID: avatar1ID, UserID: userID}, nil)
	repoMock.EXPECT().
		SoftDeleteAvatar(gomock.Any(), avatar1ID).
		Return(nil)

	// SoftDeleteAvatar loop for avatar 2
	repoMock.EXPECT().
		GetAvatarByID(gomock.Any(), avatar2ID).
		Return(&models.Avatar{ID: avatar2ID, UserID: userID}, nil)
	repoMock.EXPECT().
		SoftDeleteAvatar(gomock.Any(), avatar2ID).
		Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/"+userID+"/avatar", nil)
	req = addUserAndLoggerToContext(req, testUserID)
	req = addChiURLParams(req, map[string]string{"user_id": userID})

	rec := httptest.NewRecorder()
	h.DeleteUserAvatar(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}
