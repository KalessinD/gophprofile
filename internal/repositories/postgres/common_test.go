package postgres_test

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock" // Стандартная библиотека для моков SQL
	"github.com/KalessinD/gophprofile/internal/repositories/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLStorage_Ping(t *testing.T) {
	t.Run("successful ping", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectPing()

		storage := postgres.NewSQLStorage(db)
		err = storage.Ping(t.Context())

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ping error", func(t *testing.T) {
		db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer db.Close()

		mock.ExpectPing().WillReturnError(errors.New("connection refused"))

		storage := postgres.NewSQLStorage(db)
		err = storage.Ping(t.Context())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
