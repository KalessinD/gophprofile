package metrics

import (
	"context"
	"database/sql"

	"github.com/KalessinD/gophprofile/internal/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RecordDBStats starts a background goroutine that periodically records database connection pool statistics.
// It uses UpDownCounter (OTel equivalent of Prometheus Gauge) to imperatively push absolute values.
func RecordDBStats(_ context.Context, db *sql.DB) (context.CancelFunc, error) {
	meter := otel.Meter(common.OtelServiceName)

	openConns, err := meter.Int64ObservableGauge(
		"db_open_connections",
		metric.WithDescription("Number of open connections to the database"),
	)
	if err != nil {
		return nil, err
	}

	idleConns, err := meter.Int64ObservableGauge(
		"db_idle_connections",
		metric.WithDescription("Number of idle connections in the pool"),
	)
	if err != nil {
		return nil, err
	}

	waitCount, err := meter.Int64ObservableGauge(
		"db_wait_count_total",
		metric.WithDescription("Total number of connections waited for"),
	)
	if err != nil {
		return nil, err
	}

	attrs := attribute.String("component", "postgres")

	// RegisterCallback being called by SDK each time while scraping (exporting) of metrics.
	reg, err := meter.RegisterCallback(
		func(_ context.Context, observer metric.Observer) error {
			stats := db.Stats()

			observer.ObserveInt64(openConns, int64(stats.OpenConnections), metric.WithAttributes(attrs))
			observer.ObserveInt64(idleConns, int64(stats.Idle), metric.WithAttributes(attrs))
			observer.ObserveInt64(waitCount, stats.WaitCount, metric.WithAttributes(attrs))

			return nil
		},
		openConns,
		idleConns,
		waitCount,
	)
	if err != nil {
		return nil, err
	}

	cancel := func() {
		_ = reg.Unregister()
	}

	return cancel, nil
}
