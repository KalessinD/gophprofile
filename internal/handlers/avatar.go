package handlers

import (
	"encoding/json"
	"errors"
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
)

var (
	ErrAvatarSizeExceeded  = errors.New("file size exceeds 10 MB limit")
	ErrAvatarInvalidFormat = errors.New("invalid image format, allowed: jpg, png, webp")
	ErrAvatarNotFound      = models.ErrAvatarNotFound
	ErrAvatarForbidden     = models.ErrAvatarForbidden
)

type AvatarHandler struct {
	service *services.AvatarService
}

func NewAvatarHandler(service *services.AvatarService) *AvatarHandler {
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
+Content-Disposition: form-data; name="file"; filename="photo.jpg"
Content-Type: image/jpeg

<binary data>
------WebKitFormBoundary7MA4YWxkTrZu0gW--
```

Возможные коды ответа:
- 201 — аватарка успешно загружена, задача отправлена в воркер;
- 400 — неверный формат файла или отсутствует файл;
- 413 — файл превышает лимит в 10 МБ;
- 500 — внутренняя ошибки сервера.
*/
func (h *AvatarHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	if err := h.service.EnsureDependencies(); err != nil {
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Парсинг multipart формы
	reader, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, `{"error": "failed to parse multipart form", "details": "missing file in request"}`, http.StatusBadRequest)
		return
	}
	defer reader.Close()

	// Валидация размера файла
	if r.ContentLength > MaxAvatarSizeBytes {
		http.Error(w, `{"error": "file too big", "max_size": 10485760}`, http.StatusRequestEntityTooLarge)
		return
	}

	// Валидация расширения файла
	ext := strings.ToLower(path.Ext(r.URL.Path))
	if !strings.Contains(AllowedImageExtensions, ext) {
		http.Error(w, `{"error": "invalid image format", "allowed": "jpg, png, webp"}`, http.StatusBadRequest)
		return
	}

	// Получаем ID пользователя из контекста (установлено мидлварью)
	userID := middleware.GetUserID(r.Context())

	// Вызов бизнес-логики (создание метаданных, загрузка в БД, отправка в Kafka, загрузка в S3)
	err = h.service.CreateAvatar(r.Context(), &models.Avatar{
		UserID: userID,
		// Другие поля заполняет сервис внутри CreateAvatar
	})

	if err != nil {
		// TODO: Обработка ошибок из сервиса (например, конфликт в БД, ошибка S3)
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
}

/*
GetAvatar retrieves an avatar image (original or thumbnail) based on query parameters.
+
Формат запроса:
```
GET /api/vларавatars/550e8400-e29b-41d4-a716-446655440000?size=300x300&format=webp HTTP/1.1
```

Возможные коды ответа:
- 200 — бинарные данные картинки;
- 404 — аватар не найден;
- 500 — внутренняя ошибка сервера.
*/
func (h *AvatarHandler) GetAvatar(w http.ResponseWriter, r *http.Request) {
	if err := h.service.EnsureDependencies(); err != nil {
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	avatarID := chi.URLParam(r, "avatar_id")

	// Вызов бизнес-логики
	obj, err := h.service.GetAvatarByID(r.Context(), avatarID)
	if err != nil {
		if errors.Is(err, models.ErrAvatarNotFound) {
			http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	if obj == nil {
		http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
		return
	}
	// defer obj.Close()

	// Установка заголовков ответа
	mimeType := "image/jpeg" // Дефолтный тип, в реальности берем из models.Avatar.MimeType
	if obj.MimeType != "" {
		mimeType = obj.MimeType
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "max-age=86400")
	w.Header().Set("ETag", `"`+avatarID) // Простой ETag для кеширования

	// Запись бинарных данных в ответ
	// _, err = io.Copy(w, obj)
	// if err != nil {
	//	http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
	//	return
	// }
}

/*
GetAvatarMetadata retrieves the metadata for a specific avatar.

Возможные коды ответа:
- 200 — JSON с метаданными;
- 404 — аватар не найден;
- 500 — внутренняя ошибка сервера.
*/
func (h *AvatarHandler) GetAvatarMetadata(w http.ResponseWriter, r *http.Request) {
	if err := h.service.EnsureDependencies(); err != nil {
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	avatarID := chi.URLParam(r, "avatar_id")

	avatar, err := h.service.GetAvatarByID(r.Context(), avatarID)
	if err != nil {
		if errors.Is(err, models.ErrAvatarNotFound) {
			http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Сериализация ответа
	json.NewEncoder(w).Encode(avatar)
}

/*
DeleteAvatar deletes a specific avatar by its ID.

Возможные коды ответа:
- 204 — успешно удалено;
- 403 — попытка удалить чужую аватарку;
- 404 — аватарка не найдена;
- 500 — внутренняя ошибка сервера.
*/
func (h *AvatarHandler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	if err := h.service.EnsureDependencies(); err != nil {
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	avatarID := chi.URLParam(r, "avatar_id")
	// userID := middleware.GetUserID(r.Context())

	// Вызов бизнес-логики (мягкое удаление в БД, асинхронное удаление из S3)
	err := h.service.SoftDeleteAvatar(r.Context(), avatarID)
	if err != nil {
		if errors.Is(err, models.ErrAvatarNotFound) {
			http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
			return
		}
		if errors.Is(err, models.ErrAvatarForbidden) {
			http.Error(w, `{"error": "you can only delete your own avatars"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

/*
GetUserAvatar retrieves the current (latest) avatar for a user.

Возможные коды ответа:
- 200 — бинарные данные картинки;
- 404 — у пользователя нет аватарок;
- 500 — внутренняя ошибка сервера.
*/
func (h *AvatarHandler) GetUserAvatar(w http.ResponseWriter, r *http.Request) {
	if err := h.service.EnsureDependencies(); err != nil {
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	userID := chi.URLParam(r, "user_id")

	// Получаем список аватарок пользователя. Берем последнюю (сортировка по created_at DESC LIMIT 1).
	avatars, err := h.service.GetAvatarsByUserID(r.Context(), userID)
	if err != nil || len(avatars) == 0 {
		http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
		return
	}

	currentAvatar := avatars[0]

	// Получаем саму картинку
	obj, err := h.service.GetAvatarByID(r.Context(), currentAvatar.ID)
	if err != nil {
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}
	if obj == nil {
		http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
		return
	}
	// defer obj.Close()

	mimeType := "image/jpeg"
	if currentAvatar.MimeType != "" {
		mimeType = currentAvatar.MimeType
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Cache-Control", "max-age=86400")

	// _, err = io.Copy(w, obj)
	// if err != nil {
	//	http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
	//	return
	// }
}

/*
DeleteUserAvatar deletes all avatars for a specific user.

Возможные коды ответа:
- 204 — успешно удалено;
- 404 — у пользователя нет аватарок;
- 500 — внутренняя ошибка сервера.
*/
func (h *AvatarHandler) DeleteUserAvatar(w http.ResponseWriter, r *http.Request) {
	if err := h.service.EnsureDependencies(); err != nil {
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	userID := chi.URLParam(r, "user_id")

	// Удаляем все аватарки пользователя
	avatars, err := h.service.GetAvatarsByUserID(r.Context(), userID)
	if err != nil || len(avatars) == 0 {
		http.Error(w, `{"error": "avatar not found"}`, http.StatusNotFound)
		return
	}

	// Мягкое удаление всех найденных аватарок
	for _, avatar := range avatars {
		if err := h.service.SoftDeleteAvatar(r.Context(), avatar.ID); err != nil {
			// Логируем ошибку, но не прерываем процесс
			_ = err
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

/*
GetUserAvatars returns a list of all avatars belonging to a user.

Возможные коды ответа:
- 200 — JSON массив с аватарками;
- 500 — внутренняя ошибка сервера.
*/
func (h *AvatarHandler) GetUserAvatars(w http.ResponseWriter, r *http.Request) {
	if err := h.service.EnsureDependencies(); err != nil {
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	userID := chi.URLParam(r, "user_id")

	avatars, err := h.service.GetAvatarsByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error": "internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Сериализация списка
	json.NewEncoder(w).Encode(avatars)
}
