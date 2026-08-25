package pgerrors_test

import (
	"errors"
	"testing"

	"github.com/KalessinD/gophprofile/internal/services/db/pgerrors"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
)

func TestNewPostgresErrorClassifier(t *testing.T) {
	classifier := pgerrors.NewPostgresErrorClassifier()
	assert.NotNil(t, classifier)
}

func TestClassify_NilError(t *testing.T) {
	classifier := pgerrors.NewPostgresErrorClassifier()
	assert.Equal(t, pgerrors.NonRetriable, classifier.Classify(nil))
}

func TestClassify_NonPgError(t *testing.T) {
	classifier := pgerrors.NewPostgresErrorClassifier()
	assert.Equal(t, pgerrors.NonRetriable, classifier.Classify(errors.New("generic error")))
}

func TestClassify_RetriableError(t *testing.T) {
	classifier := pgerrors.NewPostgresErrorClassifier()
	pgErr := &pgconn.PgError{Code: pgerrcode.DeadlockDetected}
	assert.Equal(t, pgerrors.Retriable, classifier.Classify(pgErr))
}

func TestClassifyPgError_RetriableCodes(t *testing.T) {
	retriableCodes := []string{
		pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure,
		pgerrcode.TransactionRollback,
		pgerrcode.SerializationFailure,
		pgerrcode.DeadlockDetected,
		pgerrcode.CannotConnectNow,
	}

	for _, code := range retriableCodes {
		t.Run(code, func(t *testing.T) {
			pgErr := &pgconn.PgError{Code: code}
			assert.Equal(t, pgerrors.Retriable, pgerrors.ClassifyPgError(pgErr))
		})
	}
}

func TestClassifyPgError_NonRetriableCodes(t *testing.T) {
	nonRetriableCodes := []string{
		pgerrcode.UniqueViolation,
		pgerrcode.NotNullViolation,
		pgerrcode.SyntaxError,
		pgerrcode.UndefinedTable,
	}

	for _, code := range nonRetriableCodes {
		t.Run(code, func(t *testing.T) {
			pgErr := &pgconn.PgError{Code: code}
			assert.Equal(t, pgerrors.NonRetriable, pgerrors.ClassifyPgError(pgErr))
		})
	}
}
