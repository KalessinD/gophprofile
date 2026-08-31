package metrics

import (
	"context"
	"database/sql"
	"time"

	"github.com/KalessinD/gophprofile/internal/common"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// RecordDBStats starts a background goroutine that periodically records database connection pool statistics.
// It uses UpDownCounter (OTel equivalent of Prometheus Gauge) to imperatively push absolute values.
func RecordDBStats(ctx context.Context, db *sql.DB) (context.CancelFunc, error) {
	meter := otel.Meter(common.OtelServiceName)

	openConns, err := meter.Int64UpDownCounter(
		"db_open_connections",
		metric.WithDescription("Number of open connections to the database"),
	)
	if err != nil {
		return nil, err
	}

	idleConns, err := meter.Int64UpDownCounter(
		"db_idle_connections",
		metric.WithDescription("Number of idle connections in the pool"),
	)
	if err != nil {
		return nil, err
	}

	waitCount, err := meter.Int64UpDownCounter(
		"db_wait_count_total",
		metric.WithDescription("Total number of connections waited for"),
	)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := db.Stats()

				attrs := attribute.String("component", "postgres")

				// For UpDownCounter we use Add to overwrite the absolute state every 10 seconds
				openConns.Add(ctx, int64(stats.OpenConnections), metric.WithAttributes(attrs))
				idleConns.Add(ctx, int64(stats.Idle), metric.WithAttributes(attrs))
				waitCount.Add(ctx, stats.WaitCount, metric.WithAttributes(attrs))
			}
		}
	}()

	return cancel, nil
}
