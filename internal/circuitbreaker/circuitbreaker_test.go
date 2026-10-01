package circuitbreaker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	cb "github.com/KalessinD/gophprofile/internal/circuitbreaker"

	"github.com/KalessinD/gophprofile/internal/config"
)

func TestCircuitBreaker_Execute_ClosedToOpen(t *testing.T) {
	ctx := t.Context()
	toogle := cb.NewCircuitBreaker(&config.CircuitBreaker{FailureThreshold: 2, ResetTimeout: 50 * time.Millisecond})

	err := toogle.Execute(ctx, func() error { return nil })
	assert.NoError(t, err)

	err = toogle.Execute(ctx, func() error { return errors.New("fail") })
	assert.Error(t, err)

	err = toogle.Execute(ctx, func() error { return errors.New("fail") })
	assert.Error(t, err)

	// Should be open now
	err = toogle.Execute(ctx, func() error { return nil })
	assert.ErrorIs(t, err, cb.ErrCircuitOpen)
}

func TestCircuitBreaker_Execute_HalfOpen(t *testing.T) {
	ctx := t.Context()
	toogle := cb.NewCircuitBreaker(&config.CircuitBreaker{FailureThreshold: 1, ResetTimeout: 50 * time.Millisecond})

	// Trip the breaker
	err := toogle.Execute(ctx, func() error { return errors.New("fail") })
	assert.Error(t, err)

	// Wait for reset timeout
	time.Sleep(60 * time.Millisecond)

	// Should be half-open, allow request
	err = toogle.Execute(ctx, func() error { return nil })
	assert.NoError(t, err)

	// Should be closed again
	assert.Equal(t, cb.StateClosed, toogle.State())
}

func TestCircuitBreaker_Execute_ContextCancelledNotCounted(t *testing.T) {
	ctx := t.Context()
	toogle := cb.NewCircuitBreaker(&config.CircuitBreaker{FailureThreshold: 1, ResetTimeout: 50 * time.Millisecond})

	cancelErr := context.Canceled
	err := toogle.Execute(ctx, func() error { return cancelErr })
	assert.ErrorIs(t, err, cancelErr)

	// Should not be open
	err = toogle.Execute(ctx, func() error { return nil })
	assert.NoError(t, err)
}
