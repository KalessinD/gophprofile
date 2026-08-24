//go:generate mockgen -source=common.go -destination=mocks/mock_common.gen.go -package=mocks
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/KalessinD/gophprofile/internal/services/db/pgerrors"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	RetryingAttempts  = 3
	RetryingDelay     = 100 * time.Millisecond
	RetryingDelayStep = 200 * time.Millisecond

	PsqlGophkeeperSchema = "gophprofile"
)

type (
	txWrapperFunc func(tx *sql.Tx) error
	wrapperFunc   func(ctx context.Context) (*sql.Row, error)

	SQLStorage struct {
		db *sql.DB
	}

	SQLStorageInterface interface {
		Ping(ctx context.Context) error
	}
)

/*
Конструктор структуры для работы с БД
*/
func NewSQLStorage(psql *sql.DB) *SQLStorage {
	return &SQLStorage{db: psql}
}

/*
Пингуем сервер БД
*/
func (r *SQLStorage) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *SQLStorage) withRetry(ctx context.Context, action wrapperFunc) (*sql.Row, error) {
	var lastErr error

	for attempts := range RetryingAttempts {
		obj, err := action(ctx)
		lastErr = err

		if err == nil {
			return obj, nil
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrors.ClassifyPgError(pgErr) == pgerrors.Retriable {
			delay := RetryingDelay + time.Duration(attempts)*RetryingDelayStep

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		return obj, err
	}

	return nil, lastErr
}

func (r *SQLStorage) withTxRetry(ctx context.Context, action txWrapperFunc) error {
	var lastErr error

	for attempts := range RetryingAttempts {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		err = action(tx)
		if err == nil {
			return tx.Commit()
		}

		_ = tx.Rollback()
		lastErr = err

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgerrors.ClassifyPgError(pgErr) == pgerrors.Retriable {
			delay := RetryingDelay + time.Duration(attempts)*RetryingDelayStep

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		return err
	}

	return lastErr
}
