package handlers

import (
	"context"
	"net/http"

	"github.com/KalessinD/gophprofile/internal/middleware"
)

type (
	// AvatarServiceInterface defines the contract for avatar business logic.
	// Will be implemented by internal/services/avatar_service.go
	AvatarServiceInterface interface {
		UploadAvatar(ctx context.Context, userID string, fileData []byte, mimeType string) (*AvatarUploadResponse, error)
		GetAvatar(ctx context.Context, avatarID string, size string, format string) ([]byte, string, error)
		GetAvatarMetadata(ctx context.Context, avatarID string) (*AvatarMetadataResponse, error)
		DeleteAvatar(ctx context.Context, userID string, avatarID string) error
		GetUserAvatar(ctx context.Context, userID string) ([]byte, string, error)
		DeleteUserAvatar(ctx context.Context, userID string) error
		GetUserAvatars(ctx context.Context, userID string) ([]*UserAvatarsResponse, error)
	}

	// AvatarHandler handles HTTP requests for avatar operations.
	AvatarHandler struct {
		service AvatarServiceInterface
	}

	// AvatarUploadResponse represents the JSON response after a successful upload.
	AvatarUploadResponse struct {
		ID     string `json:"id"`
		UserID string `json:"user_id"`
		URL    string `json:"url"`
		Status string `json:"status"`
	}

	// AvatarMetadataResponse represents the detailed metadata of an avatar.
	AvatarMetadataResponse struct {
		ID         string           `json:"id"`
		UserID     string           `json:"user_id"`
		FileName   string           `json:"file_name"`
		MimeType   string           `json:"mime_type"`
		Size       int64            `json:"size"`
		Dimensions *Dimensions      `json:"dimensions,omitempty"`
		Thumbnails []*ThumbnailInfo `json:"thumbnails"`
		CreatedAt  string           `json:"created_at"`
		UpdatedAt  string           `json:"updated_at"`
	}

	// Dimensions represents the width and height of an image.
	Dimensions struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}

	// ThumbnailInfo contains the URL and size of a generated thumbnail.
	ThumbnailInfo struct {
		Size string `json:"size"`
		URL  string `json:"url"`
	}

	// UserAvatarsResponse represents a single avatar in the user's list.
	UserAvatarsResponse struct {
		ID        string `json:"id"`
		URL       string `json:"url"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
	}
)

// NewAvatarHandler creates a new instance of AvatarHandler.
func NewAvatarHandler(service AvatarServiceInterface) *AvatarHandler {
	return &AvatarHandler{service: service}
}

/*
UploadAvatar handles the uploading of a new user avatar.

Формат запроса:
```
POST /api/v1/avatars HTTP/1.1
Content-Type: multipart/form-data
X-User-ID: user-123

------WebKitFormBoundary7MA4YWxkTrZu0gW
Content-Disposition: form-data; name="file"; filename="photo.jpg"
Content-Type: image/jpeg

<binary data>
------WebKitFormBoundary7MA4YWxkTrZu0gW--
```

Возможные коды ответа:
- 201 — аватарка успешно загружена, задача отправлена в воркер;
- 400 — неверный формат файла или отсутствует файл;
- 413 — файл превышает лимит в 10 МБ;
- 500 — внутренняя ошибка сервера.
*/
func (h *AvatarHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	_ = userID // TODO: implement
	w.WriteHeader(http.StatusNotImplemented)
}

/*
GetAvatar retrieves an avatar image (original or thumbnail).

Формат запроса:
```
GET /api/v1/avatars/550e8400-e29b-41d4-a716-446655440000?size=300x300&format=webp HTTP/1.1
```

Возможные коды ответа:
- 200 — бинарные данные картинки;
- 404 — аватар не найден;
- 500 — внутренняя ошибка сервера.
*/
func (h *AvatarHandler) GetAvatar(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

/*
GetAvatarMetadata retrieves the metadata for a specific avatar.

Возможные коды ответа:
- 200 — JSON с метаданными;
- 404 — аватар не найден;
- 500 — внутренняя ошибка сервера.
*/
func (h *AvatarHandler) GetAvatarMetadata(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

/*
DeleteAvatar deletes a specific avatar by its ID.

Возможные коды ответа:
- 204 — успешно удалено;
- 403 — попытка удалить чужую аватарку;
- 404 — аватарка не найдена;
- 500 — внутренняя ошибка сервера.
*/
func (h *AvatarHandler) DeleteAvatar(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

/*
GetUserAvatar retrieves the current (latest or specific) avatar for a user.

Возможные коды ответа:
- 200 — бинарные данные картинки;
- 404 — у пользователя нет аватарок;
- 500 — внутренняя ошибка сервера.
*/
func (h *AvatarHandler) GetUserAvatar(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

/*
DeleteUserAvatar deletes all avatars or the current one for a specific user.

Возможные коды ответа:
- 204 — успешно удалено;
- 500 — внутренняя ошибка сервера.
*/
func (h *AvatarHandler) DeleteUserAvatar(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

/*
GetUserAvatars returns a list of all avatars belonging to a user.

Возможные коды ответа:
- 200 — JSON массив с аватарками;
- 500 — внутренняя ошибка сервера.
*/
func (h *AvatarHandler) GetUserAvatars(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
