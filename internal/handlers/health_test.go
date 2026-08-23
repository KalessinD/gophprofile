package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/KalessinD/gophprofile/internal/handlers"
	"github.com/KalessinD/gophprofile/internal/services/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// nolint: unparam
// setupHealthTest initializes mocks and the HealthHandler.
func setupHealthTest(t *testing.T) (*gomock.Controller, sqlmock.Sqlmock, *mocks.MockObjectStorage, *mocks.MockAvatarProducer, *handlers.HealthHandler) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)

	ctrl := gomock.NewController(t)
	s3Mock := mocks.NewMockObjectStorage(ctrl)
	prodMock := mocks.NewMockAvatarProducer(ctrl)

	healthHandler := handlers.NewHealthHandler(db, s3Mock, prodMock)
	return ctrl, mock, s3Mock, prodMock, healthHandler
}

// assertHealthResponse is a helper to decode the JSON and verify status codes and component states.
func assertHealthResponse(t *testing.T, rec *httptest.ResponseRecorder, expectedHTTPStatus int, expectedOverallStatus string, components map[string]string) {
	t.Helper()
	assert.Equal(t, expectedHTTPStatus, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp handlers.HealthResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, expectedOverallStatus, resp.Status)
	require.Len(t, resp.Components, len(components), "Number of components mismatch")

	for key, expectedValue := range components {
		actualValue, exists := resp.Components[key]
		assert.True(t, exists, "Component %s missing in response", key)
		assert.Equal(t, expectedValue, actualValue, "Component %s status mismatch", key)
	}
}

func TestCheckHealth_AllHealthy(t *testing.T) {
	ctrl, dbMock, _, _, h := setupHealthTest(t)
	defer ctrl.Finish()

	// DB ping succeeds
	dbMock.ExpectPing()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.CheckHealth(rec, req)

	assertHealthResponse(t, rec, http.StatusOK, "ok", map[string]string{
		"database": "ok",
		"s3":       "ok",
		"kafka":    "ok",
	})
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestCheckHealth_DatabaseError(t *testing.T) {
	ctrl, dbMock, _, _, h := setupHealthTest(t)
	defer ctrl.Finish()

	// DB ping fails
	dbMock.ExpectPing().WillReturnError(errors.New("connection refused"))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.CheckHealth(rec, req)

	assertHealthResponse(t, rec, http.StatusServiceUnavailable, "error", map[string]string{
		"database": "error",
		"s3":       "ok",
		"kafka":    "ok",
	})
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestCheckHealth_MissingS3AndKafka(t *testing.T) {
	// Pass nil for S3 and Producer
	db, dbMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer db.Close()

	dbMock.ExpectPing()

	h := handlers.NewHealthHandler(db, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.CheckHealth(rec, req)

	assertHealthResponse(t, rec, http.StatusServiceUnavailable, "error", map[string]string{
		"database": "ok",
		"s3":       "not initialized",
		"kafka":    "not initialized",
	})
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestCheckHealth_AllDependenciesNil(t *testing.T) {
	// Completely uninitialized handler
	h := handlers.NewHealthHandler(nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	h.CheckHealth(rec, req)

	assertHealthResponse(t, rec, http.StatusServiceUnavailable, "error", map[string]string{
		"database": "not initialized",
		"s3":       "not initialized",
		"kafka":    "not initialized",
	})
}
