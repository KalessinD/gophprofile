package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	mw "github.com/KalessinD/gophprofile/internal/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRequestIDMiddleware_GeneratesAndSetsHeader(t *testing.T) {
	var capturedID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = mw.GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler := mw.RequestIDMiddleware(next)
	handler.ServeHTTP(rec, req)

	assert.NotEmpty(t, capturedID)
	assert.Equal(t, capturedID, rec.Header().Get("X-Request-Id"))
}

func TestRequestIDMiddleware_UsesExistingHeader(t *testing.T) {
	var capturedID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = mw.GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-Id", "existing-uuid-123")
	rec := httptest.NewRecorder()

	handler := mw.RequestIDMiddleware(next)
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "existing-uuid-123", capturedID)
	assert.Equal(t, "existing-uuid-123", rec.Header().Get("X-Request-Id"))
}

func TestGetRequestID_Missing(t *testing.T) {
	id := mw.GetRequestID(context.Background())
	assert.Equal(t, "00000000-0000-0000-0000-0000", id)
}
