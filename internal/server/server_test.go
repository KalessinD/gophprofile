package gophprofile_test

import (
	"testing"
	"time"

	"github.com/KalessinD/gophprofile/internal/config"
	srv "github.com/KalessinD/gophprofile/internal/server"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestGetBaseRouter(t *testing.T) {
	t.Helper()

	log := zap.NewNop()
	cfg := &config.ServerConfig{
		ProcessingTimeout: 10 * time.Second,
	}

	router := srv.GetBaseRouter(cfg, log)

	assert.NotNil(t, router, "Router should not be nil")
}

// TestPsqlConnect_Failure тестирует сценарий ошибки подключения.
// Успешное подключение юнит-тестом не покрывается, так как требует реальной БД.
func TestPsqlConnect_Failure(t *testing.T) {
	log := zap.NewNop()
	ctx := t.Context()

	// Несуществующий DSN
	dsn := "host=localhost port=9999 user=invalid password=invalid dbname=invalid sslmode=disable"

	// sql.Open может не вернуть ошибку сразу, а Ping вернет
	db, err := srv.PsqlConnect(ctx, dsn, log)
	assert.Error(t, err, "Should return error for invalid DSN")
	assert.Nil(t, db)
}
