package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

const (
	maxConnectionRetries           = 3
	waitIntervalBetweenConnections = time.Second * 3
	otelPgxDriverName              = "otelpgx"
)

// init registers a custom database/sql driver that wraps pgx with OpenTelemetry tracing.
func init() {
	sql.Register(otelPgxDriverName, stdlib.GetDefaultDriver())
}

// Connect establishes a connection to the PostgreSQL database using a pgx driver
// instrumented with OpenTelemetry. It includes a retry mechanism for the initial connection.
func Connect(ctx context.Context, dsn string) (*sql.DB, error) {
	pgxConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parsing DSN: %w", err)
	}

	// Inject OTel tracer into pgx config for automatic SQL query tracing
	pgxConfig.Tracer = otelpgx.NewTracer()

	db, err := sql.Open(otelPgxDriverName, stdlib.RegisterConnConfig(pgxConfig))
	if err != nil {
		return nil, fmt.Errorf("opening db connection: %w", err)
	}

	var lastErr error

	for attempt := range maxConnectionRetries {
		lastErr = db.PingContext(ctx)
		if lastErr == nil {
			break
		}

		if attempt < maxConnectionRetries {
			select {
			case <-ctx.Done():
				db.Close()
				return nil, ctx.Err()
			case <-time.After(waitIntervalBetweenConnections):
				// Time expired, go to next retry
			}
		}
	}

	if lastErr != nil {
		db.Close()
		return nil, fmt.Errorf("db connection failed after %d retries: %w", maxConnectionRetries, lastErr)
	}

	go func() {
		<-ctx.Done()
		db.Close()
	}()

	return db, nil
}
