package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/KalessinD/gophprofile/internal/services"
)

const (
	statusError          = "error"
	statusNotInitialized = "not initialized"
)

type (
	// KafkaHealthChecker defines the contract for checking Kafka client health.
	KafkaHealthChecker interface {
		IsHealthy() bool
	}

	// HealthResponse represents the JSON structure for the health check endpoint.
	HealthResponse struct {
		Status     string            `json:"status"`
		Components map[string]string `json:"components"`
	}

	// HealthHandler handles health check requests.
	HealthHandler struct {
		db    *sql.DB
		s3    services.ObjectStorage
		kafka KafkaHealthChecker
	}
)

// NewHealthHandler creates a new instance of HealthHandler.
func NewHealthHandler(db *sql.DB, s3 services.ObjectStorage, kafka KafkaHealthChecker) *HealthHandler {
	return &HealthHandler{
		db:    db,
		s3:    s3,
		kafka: kafka,
	}
}

// CheckHealth performs health checks on critical dependencies (DB, S3, Kafka)
// and returns a JSON response with their statuses.
func (h *HealthHandler) CheckHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	response := &HealthResponse{
		Status:     "ok",
		Components: make(map[string]string),
	}

	// Check Database
	if h.db == nil {
		response.Status = statusError
		response.Components["database"] = statusNotInitialized
	} else if err := h.db.PingContext(ctx); err != nil {
		response.Status = statusError
		response.Components["database"] = statusError
	} else {
		response.Components["database"] = "ok"
	}

	// Check S3 Storage
	if h.s3 == nil {
		response.Status = statusError
		response.Components["s3"] = statusNotInitialized
	} else {
		response.Components["s3"] = "ok"
	}

	// Check Kafka Client (Producer or Consumer)
	if h.kafka == nil || !h.kafka.IsHealthy() {
		response.Status = statusError
		response.Components["kafka"] = statusNotInitialized
	} else {
		response.Components["kafka"] = "ok"
	}

	w.Header().Set("Content-Type", "application/json")

	if response.Status == statusError {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}
