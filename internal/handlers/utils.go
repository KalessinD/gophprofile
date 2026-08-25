package handlers

import (
	"time"

	"github.com/KalessinD/gophprofile/internal/models"
)

// NewMetadataResponse maps domain Avatar to API metadata response.
func NewMetadataResponse(avatar *models.Avatar, thumb100URL string, thumb300URL string) *models.MetadataResponse {
	resp := &models.MetadataResponse{
		ID:         avatar.ID,
		UserID:     avatar.UserID,
		MimeType:   avatar.MimeType,
		FileSize:   avatar.FileSize,
		Thumbnails: []models.ThumbnailResponse{},
		CreatedAt:  avatar.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  avatar.UpdatedAt.UTC().Format(time.RFC3339),
	}

	if avatar.Width != nil && avatar.Height != nil {
		resp.Dimensions = &models.DimensionsResponse{
			Width:  *avatar.Width,
			Height: *avatar.Height,
		}
	}

	if avatar.Thumbnail100S3Key != nil && *avatar.Thumbnail100S3Key != "" {
		resp.Thumbnails = append(resp.Thumbnails, models.ThumbnailResponse{
			Size: "100x100",
			URL:  thumb100URL,
		})
	}
	if avatar.Thumbnail300S3Key != nil && *avatar.Thumbnail300S3Key != "" {
		resp.Thumbnails = append(resp.Thumbnails, models.ThumbnailResponse{
			Size: "300x300",
			URL:  thumb300URL,
		})
	}

	return resp
}

// NewUploadResponse maps domain Avatar to API upload response.
func NewUploadResponse(avatar *models.Avatar, url string) *models.UploadResponse {
	return &models.UploadResponse{
		ID:        avatar.ID,
		UserID:    avatar.UserID,
		URL:       url,
		Status:    avatar.Status,
		CreatedAt: avatar.CreatedAt.UTC().Format(time.RFC3339),
	}
}
