package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/KalessinD/gophprofile/internal/models"
)

const (
	avatarTable = `"` + PsqlGophkeeperSchema + `".avatars"`

	QueryInsertAvatar = `
		INSERT INTO ` + avatarTable + ` (user_id, original_s3_key, status, mime_type, file_size)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	QuerySelectAvatarByID = `
		SELECT id, user_id, original_s3_key, status, mime_type, file_size, width, height, thumbnail_100_s3_key, thumbnail_300_s3_key, created_at, updated_at, deleted_at
		FROM ` + avatarTable + `
		WHERE id = $1 AND deleted_at IS NULL`

	QuerySelectAvatarsByUserID = `
		SELECT id, user_id, original_s3_key, status, mime_type, file_size, width, height, thumbnail_100_s3_key, thumbnail_300_s3_key, created_at, updated_at, deleted_at
		FROM ` + avatarTable + `
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC`

	QuerySoftDeleteAvatar = `UPDATE ` + avatarTable + ` SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	QueryHardDeleteAvatar = `DELETE FROM ` + avatarTable + ` WHERE id = $1`

	QueryUpdateAvatarStatus = `
		UPDATE ` + avatarTable + `
		SET status = $1, thumbnail_100_s3_key = $2, thumbnail_300_s3_key = $3, width = $4, height = $5, updated_at = NOW()
		WHERE id = $6`
)

// scanner is an interface to abstract sql.Row and sql.Rows for the scanning helper.
type scanner interface {
	Scan(dest ...any) error
}

// scanAvatar maps database columns to the models.Avatar struct, handling NULL values correctly.
func scanAvatar(s scanner) (*models.Avatar, error) {
	avatar := &models.Avatar{}
	var nullWidth sql.NullInt32
	var nullHeight sql.NullInt32
	var nullThumb100 sql.NullString
	var nullThumb300 sql.NullString
	var nullDeletedAt sql.NullTime

	err := s.Scan(
		&avatar.ID,
		&avatar.UserID,
		&avatar.OriginalS3Key,
		&avatar.Status,
		&avatar.MimeType,
		&avatar.FileSize,
		&nullWidth,
		&nullHeight,
		&nullThumb100,
		&nullThumb300,
		&avatar.CreatedAt,
		&avatar.UpdatedAt,
		&nullDeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scanning avatar: %w", err)
	}

	if nullWidth.Valid {
		width := int(nullWidth.Int32)
		avatar.Width = &width
	}
	if nullHeight.Valid {
		height := int(nullHeight.Int32)
		avatar.Height = &height
	}
	if nullThumb100.Valid {
		avatar.Thumbnail100S3Key = &nullThumb100.String
	}
	if nullThumb300.Valid {
		avatar.Thumbnail300S3Key = &nullThumb300.String
	}
	if nullDeletedAt.Valid {
		avatar.DeletedAt = &nullDeletedAt.Time
	}

	return avatar, nil
}

// CreateAvatar inserts a new avatar record into the database.
func (r *SQLStorage) CreateAvatar(ctx context.Context, avatar *models.Avatar) error {
	return r.withTxRetry(ctx, func(tx *sql.Tx) error {
		return tx.QueryRowContext(
			ctx,
			QueryInsertAvatar,
			avatar.UserID,
			avatar.OriginalS3Key,
			avatar.Status,
			avatar.MimeType,
			avatar.FileSize,
		).Scan(&avatar.ID, &avatar.CreatedAt, &avatar.UpdatedAt)
	})
}

// GetAvatarByID retrieves a single avatar by its ID.
func (r *SQLStorage) GetAvatarByID(ctx context.Context, avatarID string) (*models.Avatar, error) {
	var avatar *models.Avatar

	_, err := r.withRetry(ctx, func(ctx context.Context) (*sql.Row, error) {
		row := r.db.QueryRowContext(ctx, QuerySelectAvatarByID, avatarID)
		var scanErr error
		avatar, scanErr = scanAvatar(row)
		return row, scanErr
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrAvatarNotFound
		}
		return nil, err
	}

	return avatar, nil
}

// GetAvatarsByUserID retrieves a list of avatars for a specific user.
// Note: Uses standard QueryContext as withRetry only supports single-row scans.
func (r *SQLStorage) GetAvatarsByUserID(ctx context.Context, userID string) ([]*models.Avatar, error) {
	rows, err := r.db.QueryContext(ctx, QuerySelectAvatarsByUserID, userID)
	if err != nil {
		return nil, fmt.Errorf("querying avatars by user id: %w", err)
	}
	defer rows.Close()

	var avatars []*models.Avatar
	for rows.Next() {
		avatar, scanErr := scanAvatar(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		avatars = append(avatars, avatar)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating avatar rows: %w", err)
	}

	return avatars, nil
}

// SoftDeleteAvatar marks an avatar as deleted by setting the deleted_at timestamp.
func (r *SQLStorage) SoftDeleteAvatar(ctx context.Context, avatarID string) error {
	return r.withTxRetry(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, QuerySoftDeleteAvatar, avatarID)
		if err != nil {
			return err
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return models.ErrAvatarNotFound
		}

		return nil
	})
}

// HardDeleteAvatar permanently removes an avatar record from the database.
func (r *SQLStorage) HardDeleteAvatar(ctx context.Context, avatarID string) error {
	return r.withTxRetry(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, QueryHardDeleteAvatar, avatarID)
		return err
	})
}

// UpdateAvatarStatus updates the processing status and dimensions of an avatar.
func (r *SQLStorage) UpdateAvatarStatus(ctx context.Context, avatarID string, status string, thumbnail100S3Key string, thumbnail300S3Key string, width int, height int) error {
	return r.withTxRetry(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, QueryUpdateAvatarStatus, status, thumbnail100S3Key, thumbnail300S3Key, width, height, avatarID)
		if err != nil {
			return err
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return models.ErrAvatarNotFound
		}

		return nil
	})
}
