package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/KalessinD/gophprofile/internal/logger"
	"github.com/KalessinD/gophprofile/internal/logger/mocks"
	mw "github.com/KalessinD/gophprofile/internal/middleware"
	"github.com/go-chi/chi/middleware"
)

func TestMiddleware_LogsRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	// Ожидаем, что middleware вызовет With для добавления request_id
	mockLogger.EXPECT().With(gomock.Any()).Return(mockLogger)

	// Ожидаем финальный вызов логирования с точным соответствием ключей и сообщения
	mockLogger.EXPECT().Info(
		"request completed",
		"method", http.MethodGet,
		"path", "/test",
		"remote_addr", gomock.Any(),
		"duration", gomock.Any(),
		"status", gomock.Any(),
		"response_size", gomock.Any(),
		"request-content-encoding", gomock.Any(),
		"request-accept-encoding", gomock.Any(),
		"response-Content-Encoding", gomock.Any(),
		"response-Accept-Encoding", gomock.Any(),
	)

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := mw.Logger(mockLogger)
	handler := middleware(next)

	// Используем t.Context() согласно стандартам проекта
	req := httptest.NewRequest(http.MethodGet, "/test", nil).WithContext(t.Context())
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
}

// TestGetLogger тестирует получение логгера из контекста
func TestGetLogger(t *testing.T) {
	t.Run("Logger exists in context", func(t *testing.T) {
		ctx := mw.AddLoggerToContext(t.Context(), logger.NewNopLogger())
		got := mw.GetLogger(ctx)
		if got == nil {
			t.Error("expected logger, got nil")
		}
	})

	t.Run("Logger not in context", func(t *testing.T) {
		got := mw.GetLogger(t.Context())
		if got != nil {
			t.Error("expected nil, got logger")
		}
	})
}

// TestMiddleware тестирует полную работу мидлвари
func TestMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := mocks.NewMockLogger(ctrl)

	reqID := "test-request-id-123"

	// Ожидаем вызов With для добавления request_id в логгер
	mockLogger.EXPECT().With("request_id", reqID).Return(mockLogger)

	// Ожидаем вызов Info с точным соответствием ключей и значений.
	// Типы должны совпадать с теми, что будут переданы из middleware (например, int для статуса и размера).
	mockLogger.EXPECT().Info(
		"request completed",
		"method", http.MethodPost,
		"path", "/test",
		"remote_addr", gomock.Any(),
		"duration", gomock.Any(),
		"status", http.StatusCreated,
		"response_size", len("created"),
		"request-content-encoding", gomock.Any(),
		"request-accept-encoding", gomock.Any(),
		"response-Content-Encoding", gomock.Any(),
		"response-Accept-Encoding", gomock.Any(),
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, что логгер попал в контекст внутри хендлера
		log := mw.GetLogger(r.Context())
		if log == nil {
			t.Error("logger not found in request context inside handler")
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("created"))
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)

	// Добавляем RequestID в контекст запроса (имитируя chi middleware)
	ctx := context.WithValue(t.Context(), middleware.RequestIDKey, reqID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()

	// Запускаем мидлварю
	mw := mw.Logger(mockLogger)
	mw(handler).ServeHTTP(rec, req)

	// Проверяем, что ответ прошел корректно
	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}
