package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KalessinD/gophprofile/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestUserIDMiddleware_ValidHeader(t *testing.T) {
	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		userID := middleware.GetUserID(r.Context())
		assert.Equal(t, "user-123", userID, "User ID should be extracted into context")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(middleware.UserIDHeader, "user-123")
	rec := httptest.NewRecorder()

	mw := middleware.UserIDMiddleware(next)
	mw.ServeHTTP(rec, req)

	assert.True(t, handlerCalled, "Next handler should be called")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUserIDMiddleware_MissingHeader(t *testing.T) {
	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
		t.Error("Next handler should NOT be called when header is missing")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	mw := middleware.UserIDMiddleware(next)
	mw.ServeHTTP(rec, req)

	assert.False(t, handlerCalled, "Next handler should not be called")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "X-User-ID header is required")
}

func TestUserIDMiddleware_EmptyHeader(t *testing.T) {
	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
		t.Error("Next handler should NOT be called when header is empty")
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(middleware.UserIDHeader, "")
	rec := httptest.NewRecorder()

	mw := middleware.UserIDMiddleware(next)
	mw.ServeHTTP(rec, req)

	assert.False(t, handlerCalled, "Next handler should not be called")
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
