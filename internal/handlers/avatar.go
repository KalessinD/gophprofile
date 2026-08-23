package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/KalessinD/gophprofile/internal/middleware"
	"github.com/KalessinD/gophprofile/internal/models"
	"github.com/KalessinD/gophprofile/internal/services"
	"github.com/go-chi/chi/v5"
)

const (
	MaxAvatarSizeBytes     = 10 * 1024 * 1024
	AllowedImageExtensions = ".jpg,.jpeg,.png,.webp"

	msgInternalServer    = `{"error": "internal server error"}`
	msgMissingFile       = `{"error": "failed to parse multipart form", "details": "missing file in request"}`
	msgThumbnailNotReady = `{"error": "requested thumbnail is not ready yet"}`
)

var (
	ErrAvatarSizeExceeded  = errors.New("file size exceeds 10 MB limit")
	ErrAvatarInvalidFormat = errors.New("invalid image format, allowed: jpg, png, webp")
)

type AvatarHandler struct {
	service *services.AvatarService
}

func NewAvatarHandler(service *services.AvatarService) *AvatarHandler {
	return &AvatarHandler{service: service}
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
	if err := h.service.EnsureDependencies(); err != nil {
		http.Error(w, msgInternalServer, http.StatusInternalServerError)
		return
	}

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		http.Error(w, msgMissingFile, http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate file extension from the uploaded file header, not the URL path
	ext := strings.ToLower(path.Ext(fileHeader.Filename))
	if !strings.Contains(AllowedImageExtensions, ext) {
		http.Error(w, `{"error": "invalid image format", "allowed": "jpg, png, webp"}`, http.StatusBadRequest)
		return
	}

	if r.ContentLength > MaxAvatarSizeBytes {
		http.Error(w, `{"error": "file too big", "max_size": 10485760}`, http.StatusRequestEntityTooLarge)
		return
	}

	userID := middleware.GetUserID(r.Context())

	newAvatar := &models.Avatar{
		UserID:   userID,
		MimeType: fileHeader.Header.Get("Content-Type"),
	}

	err = h.service.CreateAvatar(r.Context(), newAvatar, file)
	if err != nil {
		http.Error(w, msgInternalServer, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(newAvatar); err != nil {
		// Log error in real implementation
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
	if err := h.service.EnsureDependencies(); err != nil {
		http.Error(w, msgInternalServer, http.StatusInternalServerError)
		return
	}

	avatarID := chi.URLParam(r, "avatar_id")
	requestedSize := r.URL.Query().Get("size")

	stream, avatar, err := h.service.GetAvatarFileStream(r.Context(), avatarID, requestedSize)
	if err != nil {
		if errors.Is(err, models.ErrAvatarNotFound) {
			http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
			return
		}
		if errors.Is(err, services.ErrThumbnailNotReady) {
			http.Error(w, msgThumbnailNotReady, http.StatusNotFound)
			return
		}
		http.Error(w, msgInternalServer, http.StatusInternalServerError)
		return
	}

	if stream == nil || avatar == nil {
		http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
		return
	}

	h.writeImageResponse(w, stream, avatar, avatarID)
}

/*
GetAvatarMetadata retrieves the metadata for a specific avatar.

Possible response codes:
- 200 — JSON with metadata;
- 404 — avatar not found;
- 500 — internal server error.
*/
func (h *AvatarHandler) GetAvatarMetadata(w http.ResponseWriter, r *http.Request) {
	if err := h.service.EnsureDependencies(); err != nil {
		http.Error(w, msgInternalServer, http.StatusInternalServerError)
		return
	}

	avatarID := chi.URLParam(r, "avatar_id")

	avatar, err := h.service.GetAvatarByID(r.Context(), avatarID)
	if err != nil {
		if errors.Is(err, models.ErrAvatarNotFound) {
			http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, msgInternalServer, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(avatar); err != nil {
		// Log error in real implementation, for now just return
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
	if err := h.service.EnsureDependencies(); err != nil {
		http.Error(w, msgInternalServer, http.StatusInternalServerError)
		return
	}

	avatarID := chi.URLParam(r, "avatar_id")
	userID := middleware.GetUserID(r.Context())

	err := h.service.SoftDeleteAvatar(r.Context(), avatarID, userID)
	if err != nil {
		if errors.Is(err, models.ErrAvatarNotFound) {
			http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
			return
		}
		if errors.Is(err, models.ErrAvatarForbidden) {
			http.Error(w, `{"error": "you can only delete your own avatars"}`, http.StatusForbidden)
			return
		}
		http.Error(w, msgInternalServer, http.StatusInternalServerError)
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
	if err := h.service.EnsureDependencies(); err != nil {
		http.Error(w, msgInternalServer, http.StatusInternalServerError)
		return
	}

	userID := chi.URLParam(r, "user_id")

	avatars, err := h.service.GetAvatarsByUserID(r.Context(), userID)
	if err != nil || len(avatars) == 0 {
		http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
		return
	}

	currentAvatar := avatars[0]
	requestedSize := r.URL.Query().Get("size")

	stream, avatar, err := h.service.GetAvatarFileStream(r.Context(), currentAvatar.ID, requestedSize)
	if err != nil {
		http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
		return
	}
	if stream == nil || avatar == nil {
		http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
		return
	}

	h.writeImageResponse(w, stream, avatar, currentAvatar.ID)
}

/*
DeleteUserAvatar deletes all avatars for a specific user.

Possible response codes:
- 204 — successfully deleted;
- 404 — user has no avatars;
- 500 — internal server error.
*/
func (h *AvatarHandler) DeleteUserAvatar(w http.ResponseWriter, r *http.Request) {
	if err := h.service.EnsureDependencies(); err != nil {
		http.Error(w, msgInternalServer, http.StatusInternalServerError)
		return
	}

	userID := chi.URLParam(r, "user_id")

	avatars, err := h.service.GetAvatarsByUserID(r.Context(), userID)
	if err != nil || len(avatars) == 0 {
		http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
		return
	}

	for _, avatar := range avatars {
		if deleteErr := h.service.SoftDeleteAvatar(r.Context(), avatar.ID, userID); deleteErr != nil {
			_ = deleteErr
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
	if err := h.service.EnsureDependencies(); err != nil {
		http.Error(w, msgInternalServer, http.StatusInternalServerError)
		return
	}

	userID := chi.URLParam(r, "user_id")

	avatars, err := h.service.GetAvatarsByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, msgInternalServer, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(avatars); err != nil {
		return
	}
}

// writeImageResponse streams the image data to the HTTP response and sets appropriate headers.
func (h *AvatarHandler) writeImageResponse(w http.ResponseWriter, stream io.ReadCloser, avatar *models.Avatar, avatarID string) {
	defer stream.Close()

	mimeType := "image/jpeg" // Default fallback according to TZ
	if avatar.MimeType != "" {
		mimeType = avatar.MimeType
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "max-age=86400")
	w.Header().Set("ETag", `"`+avatarID+`"`)

	if _, err := io.Copy(w, stream); err != nil {
		// If headers are already sent, we can't return a JSON error.
		// In a real application, this should be logged using the logger from context.
		return
	}
}
