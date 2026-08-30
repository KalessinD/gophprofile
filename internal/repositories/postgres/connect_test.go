package postgres_test

import (
	"testing"

	"github.com/KalessinD/gophprofile/internal/repositories/postgres"
	"github.com/stretchr/testify/assert"
)

func TestPsqlConnect_Failure(t *testing.T) {
	ctx := t.Context()

	// Несуществующий DSN
	dsn := "host=localhost port=9999 user=invalid password=invalid dbname=invalid sslmode=disable"

	// sql.Open может не вернуть ошибку сразу, а Ping вернет
	db, err := postgres.Connect(ctx, dsn)
	assert.Error(t, err, "Should return error for invalid DSN")
	assert.Nil(t, db)
}
