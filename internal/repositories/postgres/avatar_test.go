package postgres_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/KalessinD/gophprofile/internal/models"
	"github.com/KalessinD/gophprofile/internal/repositories/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLStorage_CreateAvatar(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := postgres.NewSQLStorage(db)
	now := time.Now().Truncate(time.Microsecond)

	t.Run("successful creation", func(t *testing.T) {
		avatar := &models.Avatar{
			UserID:        "user-1",
			OriginalS3Key: "orig.png",
			Status:        models.AvatarStatusProcessing,
			MimeType:      "image/png",
			FileSize:      1024,
		}

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT").
			WithArgs(avatar.UserID, avatar.OriginalS3Key, avatar.Status, avatar.MimeType, avatar.FileSize).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow("uuid-1", now, now))
		mock.ExpectCommit()

		err := storage.CreateAvatar(t.Context(), avatar)
		require.NoError(t, err)
		assert.Equal(t, "uuid-1", avatar.ID)
		assert.Equal(t, now, avatar.CreatedAt)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		avatar := &models.Avatar{UserID: "user-1"}

		mock.ExpectBegin()
		mock.ExpectQuery("INSERT").WillReturnError(errors.New("db fail"))
		mock.ExpectRollback()

		err := storage.CreateAvatar(t.Context(), avatar)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "db fail")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSQLStorage_GetAvatarByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := postgres.NewSQLStorage(db)
	now := time.Now().Truncate(time.Microsecond)
	columns := []string{
		"id", "user_id", "original_s3_key", "status", "mime_type",
		"file_size", "width", "height", "thumbnail_100_s3_key",
		"thumbnail_300_s3_key", "created_at", "updated_at", "deleted_at",
	}

	t.Run("successful get with nulls", func(t *testing.T) {
		rows := sqlmock.NewRows(columns).AddRow("uuid-1", "user-1", "orig.png", "ready", "image/png", 1024, nil, nil, nil, nil, now, now, nil)
		mock.ExpectQuery("SELECT").WithArgs("uuid-1").WillReturnRows(rows)

		avatar, err := storage.GetAvatarByID(t.Context(), "uuid-1")
		require.NoError(t, err)
		assert.NotNil(t, avatar)
		assert.Nil(t, avatar.Width)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful get with full data", func(t *testing.T) {
		rows := sqlmock.NewRows(columns).AddRow("uuid-2", "user-1", "orig.png", "ready", "image/png", 1024, 100, 100, "t100.png", "t300.png", now, now, nil)
		mock.ExpectQuery("SELECT").WithArgs("uuid-2").WillReturnRows(rows)

		avatar, err := storage.GetAvatarByID(t.Context(), "uuid-2")
		require.NoError(t, err)
		assert.NotNil(t, avatar.Width)
		assert.Equal(t, 100, *avatar.Width)
		assert.NotNil(t, avatar.Thumbnail100S3Key)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT").WithArgs("not-found").WillReturnError(sql.ErrNoRows)

		avatar, err := storage.GetAvatarByID(t.Context(), "not-found")
		require.Error(t, err)
		assert.Nil(t, avatar)
		assert.ErrorIs(t, err, models.ErrAvatarNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSQLStorage_GetAvatarsByUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := postgres.NewSQLStorage(db)
	now := time.Now().Truncate(time.Microsecond)
	columns := []string{
		"id", "user_id", "original_s3_key", "status",
		"mime_type", "file_size", "width", "height",
		"thumbnail_100_s3_key", "thumbnail_300_s3_key",
		"created_at", "updated_at", "deleted_at",
	}

	t.Run("successful list", func(t *testing.T) {
		rows := sqlmock.NewRows(columns).
			AddRow("uuid-1", "user-1", "1.png", "ready", "image/png", 1024, nil, nil, nil, nil, now, now, nil).
			AddRow("uuid-2", "user-1", "2.png", "ready", "image/png", 2048, nil, nil, nil, nil, now, now, nil)

		mock.ExpectQuery("SELECT").WithArgs("user-1").WillReturnRows(rows)

		avatars, err := storage.GetAvatarsByUserID(t.Context(), "user-1")
		require.NoError(t, err)
		assert.Len(t, avatars, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty list", func(t *testing.T) {
		mock.ExpectQuery("SELECT").WithArgs("empty-user").WillReturnRows(sqlmock.NewRows(columns))

		avatars, err := storage.GetAvatarsByUserID(t.Context(), "empty-user")
		require.NoError(t, err)
		assert.Empty(t, avatars)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSQLStorage_SoftDeleteAvatar(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := postgres.NewSQLStorage(db)

	t.Run("successful soft delete", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE").WithArgs("uuid-1").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := storage.SoftDeleteAvatar(t.Context(), "uuid-1")
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found (0 rows affected)", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE").WithArgs("uuid-missing").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		err := storage.SoftDeleteAvatar(t.Context(), "uuid-missing")
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrAvatarNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSQLStorage_UpdateAvatarStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := postgres.NewSQLStorage(db)

	t.Run("successful update", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE").
			WithArgs(models.AvatarStatusReady, "t100.png", "t300.png", 100, 100, "uuid-1").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := storage.UpdateAvatarStatus(t.Context(), "uuid-1", models.AvatarStatusReady, "t100.png", "t300.png", 100, 100)
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec("UPDATE").
			WithArgs(models.AvatarStatusError, "", "", 0, 0, "uuid-missing").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectRollback()

		err := storage.UpdateAvatarStatus(t.Context(), "uuid-missing", models.AvatarStatusError, "", "", 0, 0)
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrAvatarNotFound)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestSQLStorage_HardDeleteAvatar(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	storage := postgres.NewSQLStorage(db)

	// Escaping regex special characters (. and $) for strict sqlmock matching
	expectedQuery := `DELETE FROM "gophprofile"\.avatars" WHERE id = \$1`

	t.Run("successful hard delete", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(expectedQuery).WithArgs("uuid-1").WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := storage.HardDeleteAvatar(t.Context(), "uuid-1")
		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error during hard delete", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectExec(expectedQuery).WithArgs("uuid-err").WillReturnError(errors.New("connection lost"))
		mock.ExpectRollback()

		err := storage.HardDeleteAvatar(t.Context(), "uuid-err")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "connection lost")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
