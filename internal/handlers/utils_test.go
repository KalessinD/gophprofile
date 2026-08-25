package handlers_test

import (
	"testing"
	"time"

	"github.com/KalessinD/gophprofile/internal/handlers"
	"github.com/KalessinD/gophprofile/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUploadResponse(t *testing.T) {
	testTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	avatar := &models.Avatar{
		ID:        "upload-id-123",
		UserID:    "user-456",
		Status:    models.AvatarStatusProcessing,
		CreatedAt: testTime,
	}

	response := handlers.NewUploadResponse(avatar, "http://s3.local/bucket/key.jpg")

	require.NotNil(t, response)
	assert.Equal(t, "upload-id-123", response.ID)
	assert.Equal(t, "user-456", response.UserID)
	assert.Equal(t, "http://s3.local/bucket/key.jpg", response.URL)
	assert.Equal(t, models.AvatarStatusProcessing, response.Status)

	expectedTime := testTime.UTC().Format(time.RFC3339)
	assert.Equal(t, expectedTime, response.CreatedAt)
}

func TestNewMetadataResponse(t *testing.T) {
	testTime := time.Date(2026, 5, 10, 15, 30, 0, 0, time.UTC)
	thumb100Key := "avatars/user/av_100.jpg"
	thumb300Key := "avatars/user/av_300.jpg"
	width := 800
	height := 600

	tests := []struct {
		name           string
		avatar         *models.Avatar
		thumb100URL    string
		thumb300URL    string
		expectedThumbs int
		wantDimensions bool
	}{
		{
			name: "success with full data",
			avatar: &models.Avatar{
				ID:                "meta-id-1",
				UserID:            "user-full",
				MimeType:          "image/jpeg",
				FileSize:          1024000,
				Width:             &width,
				Height:            &height,
				Thumbnail100S3Key: &thumb100Key,
				Thumbnail300S3Key: &thumb300Key,
				CreatedAt:         testTime,
				UpdatedAt:         testTime,
			},
			thumb100URL:    "http://s3/100.jpg",
			thumb300URL:    "http://s3/300.jpg",
			expectedThumbs: 2,
			wantDimensions: true,
		},
		{
			name: "success with missing dimensions and thumbnails",
			avatar: &models.Avatar{
				ID:        "meta-id-2",
				UserID:    "user-min",
				MimeType:  "image/png",
				FileSize:  500,
				CreatedAt: testTime,
				UpdatedAt: testTime,
			},
			thumb100URL:    "",
			thumb300URL:    "",
			expectedThumbs: 0,
			wantDimensions: false,
		},
		{
			name: "success with only one thumbnail",
			avatar: &models.Avatar{
				ID:                "meta-id-3",
				Thumbnail100S3Key: &thumb100Key,
				CreatedAt:         testTime,
				UpdatedAt:         testTime,
			},
			thumb100URL:    "http://s3/100.jpg",
			thumb300URL:    "",
			expectedThumbs: 1,
			wantDimensions: false,
		},
		{
			name: "ignores thumbnail if S3 key is empty string",
			avatar: &models.Avatar{
				ID:                "meta-id-4",
				Thumbnail100S3Key: func() *string { s := ""; return &s }(),
				CreatedAt:         testTime,
				UpdatedAt:         testTime,
			},
			thumb100URL:    "http://s3/100.jpg",
			thumb300URL:    "",
			expectedThumbs: 0,
			wantDimensions: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := handlers.NewMetadataResponse(tt.avatar, tt.thumb100URL, tt.thumb300URL)

			require.NotNil(t, response)
			assert.Equal(t, tt.avatar.ID, response.ID)
			assert.Equal(t, tt.avatar.UserID, response.UserID)
			assert.Equal(t, tt.avatar.MimeType, response.MimeType)
			assert.Equal(t, tt.avatar.FileSize, response.FileSize)

			expectedTime := testTime.UTC().Format(time.RFC3339)
			assert.Equal(t, expectedTime, response.CreatedAt)
			assert.Equal(t, expectedTime, response.UpdatedAt)

			assert.Len(t, response.Thumbnails, tt.expectedThumbs)

			if tt.wantDimensions {
				require.NotNil(t, response.Dimensions)
				assert.Equal(t, 800, response.Dimensions.Width)
				assert.Equal(t, 600, response.Dimensions.Height)
			} else {
				assert.Nil(t, response.Dimensions)
			}
		})
	}
}
