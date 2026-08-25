package models

type (
	// ThumbnailResponse represents a single thumbnail entry in the metadata response.
	ThumbnailResponse struct {
		Size string `json:"size"`
		URL  string `json:"url"`
	}

	// UploadResponse represents the JSON structure for a successful avatar upload.
	UploadResponse struct {
		ID        string `json:"id"`
		UserID    string `json:"user_id"`
		URL       string `json:"url"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
	}

	// DimensionsResponse represents image width and height.
	DimensionsResponse struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}

	// MetadataResponse represents the full metadata structure for an avatar.
	MetadataResponse struct {
		ID         string              `json:"id"`
		UserID     string              `json:"user_id"`
		MimeType   string              `json:"mime_type"`
		FileSize   int64               `json:"file_size"`
		Dimensions *DimensionsResponse `json:"dimensions,omitempty"`
		Thumbnails []ThumbnailResponse `json:"thumbnails"`
		CreatedAt  string              `json:"created_at"`
		UpdatedAt  string              `json:"updated_at"`
	}
)
