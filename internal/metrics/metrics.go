package metrics

import (
	"context"
	"fmt"
	"sync"

	"github.com/KalessinD/gophprofile/internal/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

type (
	// AvatarMetrics holds the OpenTelemetry instruments for avatar business logic.
	AvatarMetrics struct {
		UploadsTotal   metric.Int64Counter
		UploadDuration metric.Float64Histogram
		StorageBytes   metric.Int64UpDownCounter
	}
)

var (
	mInstance *AvatarMetrics
	mOnce     sync.Once
	mInitErr  error
)

// Init initializes the business metrics for the avatar service.
// It uses sync.Once to ensure thread-safe singleton initialization without global variables.
func Init(_ context.Context) error {
	mOnce.Do(func() {
		meter := otel.Meter(common.OtelServiceName)

		mInstance = &AvatarMetrics{}

		mInstance.UploadsTotal, mInitErr = meter.Int64Counter(
			"avatars_uploads_total",
			metric.WithDescription("Total number of avatar uploads"),
		)
		if mInitErr != nil {
			mInitErr = fmt.Errorf("creating uploads_total counter: %w", mInitErr)
			return
		}

		mInstance.UploadDuration, mInitErr = meter.Float64Histogram(
			"avatars_upload_duration_seconds",
			metric.WithDescription("Avatar upload duration in seconds"),
		)
		if mInitErr != nil {
			mInitErr = fmt.Errorf("creating upload_duration histogram: %w", mInitErr)
			return
		}

		mInstance.StorageBytes, mInitErr = meter.Int64UpDownCounter(
			"avatars_storage_bytes",
			metric.WithDescription("Total storage used by avatars in bytes"),
		)
		if mInitErr != nil {
			mInitErr = fmt.Errorf("creating storage_bytes gauge: %w", mInitErr)
			return
		}
	})
	return mInitErr
}

// Instance returns the initialized AvatarMetrics singleton.
// It panics if Init() was not called or failed.
func Instance() *AvatarMetrics {
	if mInstance == nil {
		panic("metrics not initialized, call metrics.Init() first")
	}
	return mInstance
}
