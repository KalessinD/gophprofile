package models

import (
	"errors"
	"time"
)

const (
	AvatarStatusProcessing = "processing"
	AvatarStatusReady      = "ready"
	AvatarStatusError      = "error"

	MaxAvatarFileSizeBytes = 10 * 1024 * 1024 // 10 MB
)

var (
	ErrAvatarNotFound  = errors.New("avatar not found")
	ErrAvatarForbidden = errors.New("you can only delete your own avatars")
)

type Avatar struct {
	ID                string     `json:"id"                             db:"id"`
	UserID            string     `json:"user_id"                        db:"user_id"`
	OriginalS3Key     string     `json:"original_s3_key"                db:"original_s3_key"`
	Status            string     `json:"status"                         db:"status"`
	MimeType          string     `json:"mime_type"                      db:"mime_type"`
	FileSize          int64      `json:"file_size"                      db:"file_size"`
	Width             *int       `json:"width,omitempty"                db:"width,omitempty"`
	Height            *int       `json:"height,omitempty"               db:"height,omitempty"`
	Thumbnail100S3Key *string    `json:"thumbnail_100_s3_key,omitempty" db:"thumbnail_100_s3_key,omitempty"`
	Thumbnail300S3Key *string    `json:"thumbnail_300_s3_key,omitempty" db:"thumbnail_300_s3_key,omitempty"`
	CreatedAt         time.Time  `json:"created_at"                     db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"                     db:"updated_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty"           db:"deleted_at,omitempty"`
}
