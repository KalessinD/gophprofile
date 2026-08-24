package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/KalessinD/gophprofile/internal/common"
	"github.com/KalessinD/gophprofile/internal/middleware"
	"github.com/KalessinD/gophprofile/internal/models"
	"github.com/KalessinD/gophprofile/internal/services"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

const (
	MaxAvatarSizeBytes = 10 * 1024 * 1024

	MIMEImageJPEG = "image/jpeg"
	MIMEImagePNG  = "image/png"
	MIMEImageWebP = "image/webp"

	msgInternalServer       = `{"error": "internal server error"}`
	msgMissingFile          = `{"error": "failed to parse multipart form", "details": "missing file in request"}`
	msgThumbnailNotReady    = `{"error": "requested thumbnail is not ready yet"}`
	msgAvatarNotFound       = `{"error": "avatar not found"}`
	msgAccessForbidden      = `{"error": "you can only delete your own avatars"}`
	msgFailedReadFileHeader = `{"error": "failed to read file header"}`
	msgInvalidFileFormat    = `{"error": "invalid image format", "details": "supported formats: jpeg, png, webp"}`
)

var (
	ErrAvatarSizeExceeded  = errors.New("file size exceeds 10 MB limit")
	ErrAvatarInvalidFormat = errors.New("invalid image format, allowed: jpg, png, webp")

	allowedImageMIMETypes = map[string]struct{}{
		MIMEImageJPEG: {},
		MIMEImagePNG:  {},
		MIMEImageWebP: {},
	}
)

type (
	AvatarHandler struct {
		service      *services.AvatarService
		s3URLBuilder func(key string) string
	}

	// avatarSizeReader is a custom io.Reader that tracks the number of bytes read
	// and updates the FileSize field of the associated Avatar struct.
	avatarSizeReader struct {
		reader io.Reader
		avatar *models.Avatar
	}
)

// Read implements io.Reader. It reads from the underlying reader and accumulates the byte count.
func (sr *avatarSizeReader) Read(p []byte) (int, error) {
	bytesRead, err := sr.reader.Read(p)
	sr.avatar.FileSize += int64(bytesRead)
	return bytesRead, err
}

func NewAvatarHandler(service *services.AvatarService, s3URLBuilder func(key string) string) *AvatarHandler {
	return &AvatarHandler{
		service:      service,
		s3URLBuilder: s3URLBuilder,
	}
}

/*
UploadAvatar handles the uploading of a new user avatar.

Possible response codes:
- 201 — avatar successfully uploaded, task sent to worker;
- 400 — invalid file format or missing file;
- 413 — file exceeds 10 MB limit;
- 500 — internal server error.
*/
func (h *AvatarHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("image") // see web/static/index.html
	if err != nil {
		http.Error(w, msgMissingFile, http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Read first 512 bytes for real MIME type detection (prevents fake extensions)
	sniffBytes, err := io.ReadAll(io.LimitReader(file, 512))
	if err != nil {
		http.Error(w, msgFailedReadFileHeader, http.StatusBadRequest)
		return
	}

	detectedMIME := http.DetectContentType(sniffBytes)
	if _, isAllowed := allowedImageMIMETypes[detectedMIME]; !isAllowed {
		http.Error(w, msgInvalidFileFormat, http.StatusBadRequest)
		return
	}

	// Wrap file in LimitReader to strictly enforce the 10MB limit (prevents Content-Length spoofing)
	safeReader := io.LimitReader(io.MultiReader(bytes.NewReader(sniffBytes), file), MaxAvatarSizeBytes)

	userID := middleware.GetUserID(r.Context())

	newAvatar := &models.Avatar{
		UserID:   userID,
		MimeType: detectedMIME,
	}

	sizeTrackingReader := &avatarSizeReader{
		reader: safeReader,
		avatar: newAvatar,
	}

	ctx := r.Context()
	logger := middleware.GetLogger(ctx).Sugar()

	err = h.service.CreateAvatar(ctx, newAvatar, sizeTrackingReader)
	if err != nil {
		status, message := h.defineResponseStatusByError(err)

		logger.Debugf("can't upload the avatar: %v", err)
		http.Error(w, message, status)

		return
	}

	w.Header().Set("Content-Type", common.AppJSONContentType)
	w.WriteHeader(http.StatusCreated)

	response := NewUploadResponse(newAvatar, h.s3URLBuilder(newAvatar.OriginalS3Key))
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Errorf("can't encode the avatar: %v", err)
		return
	}
}

/*
GetAvatar retrieves an avatar image (original or thumbnail) based on query parameters.

Possible response codes:
- 200 — binary image data;
- 404 — avatar not found or thumbnail not ready;
- 500 — internal server error.
*/
func (h *AvatarHandler) GetAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	avatarID := chi.URLParam(r, "avatar_id")
	logger := middleware.GetLogger(ctx)
	requestedSize := r.URL.Query().Get("size")
	stream, avatar, err := h.service.GetAvatarFileStream(ctx, avatarID, requestedSize)

	if err == nil && (stream == nil || avatar == nil) {
		err = models.ErrAvatarNotFound
	}

	if err != nil {
		status, message := h.defineResponseStatusByError(err)

		logger.Sugar().Debugf("can't retrieve the avatar: %v", err)
		http.Error(w, message, status)
		return
	}

	h.writeImageResponse(w, stream, avatar, avatarID, logger)
}

/*
GetAvatarMetadata retrieves the metadata for a specific avatar.

Possible response codes:
- 200 — JSON with metadata;
- 404 — avatar not found;
- 500 — internal server error.
*/
func (h *AvatarHandler) GetAvatarMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	avatarID := chi.URLParam(r, "avatar_id")

	avatar, err := h.service.GetAvatarByID(ctx, avatarID)
	if err != nil {
		status, message := h.defineResponseStatusByError(err)
		middleware.GetLogger(ctx).Sugar().Debugf("can't retrieve the avatar: %v", err)
		http.Error(w, message, status)
		return
	}

	w.Header().Set("Content-Type", common.AppJSONContentType)
	w.WriteHeader(http.StatusOK)

	var thumb100URL, thumb300URL string
	if avatar.Thumbnail100S3Key != nil {
		thumb100URL = h.s3URLBuilder(*avatar.Thumbnail100S3Key)
	}
	if avatar.Thumbnail300S3Key != nil {
		thumb300URL = h.s3URLBuilder(*avatar.Thumbnail300S3Key)
	}

	response := NewMetadataResponse(avatar, thumb100URL, thumb300URL)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		middleware.GetLogger(ctx).Sugar().Errorf("can't encode the avatar: %v", err)
		return
	}
}

/*
DeleteAvatar deletes a specific avatar by its ID.

Possible response codes:
- 204 — successfully deleted;
- 403 — attempt to delete someone else's avatar;
- 404 — avatar not found;
- 500 — internal server error.
*/
func (h *AvatarHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	avatarID := chi.URLParam(r, "avatar_id")
	userID := middleware.GetUserID(ctx)

	err := h.service.SoftDeleteAvatar(ctx, avatarID, userID)
	if err != nil {
		status, message := h.defineResponseStatusByError(err)

		middleware.GetLogger(ctx).Sugar().Debugf("can't delete user's avatar: %v", err)
		http.Error(w, message, status)

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

/*
GetUserAvatar retrieves the current (latest) avatar for a user.

Possible response codes:
- 200 — binary image data;
- 404 — user has no avatars;
- 500 — internal server error.
*/
func (h *AvatarHandler) GetUserAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "user_id")
	avatars, err := h.service.GetAvatarsByUserID(ctx, userID)

	if err == nil && len(avatars) == 0 {
		err = models.ErrAvatarNotFound
	}

	if err != nil {
		status, message := h.defineResponseStatusByError(err)

		middleware.GetLogger(ctx).Sugar().Debugf("can't get user's avatar: %v", err)
		http.Error(w, message, status)

		return
	}

	currentAvatar := avatars[0]
	requestedSize := r.URL.Query().Get("size")
	logger := middleware.GetLogger(ctx)
	stream, avatar, err := h.service.GetAvatarFileStream(r.Context(), currentAvatar.ID, requestedSize)

	if err == nil && (stream == nil || avatar == nil) {
		err = models.ErrAvatarNotFound
	}

	if err != nil {
		status, message := h.defineResponseStatusByError(err)

		logger.Sugar().Debugf("can't get user's avatar: %v", err)
		http.Error(w, message, status)

		return
	}

	h.writeImageResponse(w, stream, avatar, currentAvatar.ID, logger)
}

/*
DeleteUserAvatar deletes all avatars for a specific user.

Possible response codes:
- 204 — successfully deleted;
- 404 — user has no avatars;
- 500 — internal server error.
*/
func (h *AvatarHandler) DeleteUserAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "user_id")
	logger := middleware.GetLogger(ctx).Sugar()
	avatars, err := h.service.GetAvatarsByUserID(ctx, userID)

	if err == nil && len(avatars) == 0 {
		err = models.ErrAvatarNotFound
	}

	if err != nil {
		status, message := h.defineResponseStatusByError(err)

		logger.Debugf("can't delete user's avatar: %v", err)
		http.Error(w, message, status)

		return
	}

	for _, avatar := range avatars {
		if deleteErr := h.service.SoftDeleteAvatar(ctx, avatar.ID, userID); deleteErr != nil {
			logger.Errorf("can't delete the avatar %s for user %s: %v", avatar.ID, userID, deleteErr)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

/*
GetUserAvatars returns a list of all avatars belonging to a user.

Possible response codes:
- 200 — JSON array with avatars;
- 500 — internal server error.
*/
func (h *AvatarHandler) GetUserAvatars(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := chi.URLParam(r, "user_id")
	logger := middleware.GetLogger(ctx).Sugar()
	avatars, err := h.service.GetAvatarsByUserID(ctx, userID)
	if err != nil {
		status, message := h.defineResponseStatusByError(err)

		logger.Debugf("can't get user's avatar: %v", err)
		http.Error(w, message, status)

		return
	}

	w.Header().Set("Content-Type", common.AppJSONContentType)
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(avatars); err != nil {
		logger.Errorf("can't encode avatars: %v", err)
		return
	}
}

// writeImageResponse streams the image data to the HTTP response and sets appropriate headers.
func (h *AvatarHandler) writeImageResponse(w http.ResponseWriter, stream io.ReadCloser, avatar *models.Avatar, avatarID string, logger *zap.Logger) {
	defer stream.Close()

	mimeType := "image/jpeg" // Default fallback according to TZ
	if avatar.MimeType != "" {
		mimeType = avatar.MimeType
	}

	header := w.Header()

	header.Set("Content-Type", mimeType)
	header.Set("Cache-Control", "max-age=86400")
	header.Set("ETag", `"`+avatarID+`"`)

	if _, err := io.Copy(w, stream); err != nil {
		logger.Debug("can't write into stream", zap.Error(err))
		return
	}
}

func (h *AvatarHandler) defineResponseStatusByError(err error) (status int, message string) {
	switch {
	case errors.Is(err, models.ErrAvatarNotFound):
		status = http.StatusNotFound
		message = msgAvatarNotFound
	case errors.Is(err, services.ErrThumbnailNotReady):
		status = http.StatusNotFound
		message = msgThumbnailNotReady
	case errors.Is(err, models.ErrAvatarForbidden):
		status = http.StatusForbidden
		message = msgAccessForbidden
	default:
		status = http.StatusInternalServerError
		message = msgInternalServer
	}

	return
}
