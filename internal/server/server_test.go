package gophprofile_test

import (
	"testing"
	"time"

	"github.com/KalessinD/gophprofile/internal/config"
	"github.com/KalessinD/gophprofile/internal/logger"
	srv "github.com/KalessinD/gophprofile/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBaseRouter(t *testing.T) {
	t.Helper()

	log := logger.NewNopLogger()
	cfg := &config.ServerConfig{
		ProcessingTimeout: 10 * time.Second,
	}

	router := srv.GetBaseRouter(cfg, log)

	assert.NotNil(t, router, "Router should not be nil")
}

func TestNewRouter_MissingS3(t *testing.T) {
	log := logger.NewNopLogger()
	cfg := &config.ServerConfig{
		ProcessingTimeout: 10 * time.Second,
		S3:                &config.S3{},
		Kafka:             &config.Kafka{}, // Brokers is empty ""
	}

	router, err := srv.NewRouter(t.Context(), cfg, log, nil)

	require.Error(t, err)
	assert.Nil(t, router)
	assert.Contains(t, err.Error(), "initializing kafka producer")
}
