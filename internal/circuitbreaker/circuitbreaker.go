// Package circuitbreaker provides a simple circuit breaker implementation.
package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/KalessinD/gophprofile/internal/config"
)

// ErrCircuitOpen is returned when the circuit breaker is in an open state.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// State represents the state of the circuit breaker.
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern to prevent cascading failures.
type CircuitBreaker struct {
	mu               sync.Mutex
	state            State
	failureThreshold int
	failureCount     int
	resetTimeout     time.Duration
	lastFailureTime  time.Time
}

// NewCircuitBreaker creates a new CircuitBreaker instance.
func NewCircuitBreaker(cfg *config.CircuitBreaker) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: cfg.FailureThreshold,
		resetTimeout:     cfg.ResetTimeout,
	}
}

// Returns the current state
func (cb *CircuitBreaker) State() State {
	return cb.state
}

// Execute runs the given operation, applying circuit breaker logic.
// It respects the provided context and returns immediately if the context is canceled.
func (cb *CircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	// Check context before attempting the operation
	if err := ctx.Err(); err != nil {
		return err
	}

	if !cb.allowRequest() {
		return ErrCircuitOpen
	}

	err := operation()
	cb.recordResult(err)
	return err
}

// allowRequest checks if a request is allowed based on the current state.
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = StateHalfOpen
			return true
		}
		return false
	}
	return true
}

// recordResult updates the state based on the operation result.
func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		// Do not count context cancellation as a failure
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		cb.failureCount++
		cb.lastFailureTime = time.Now()
		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
		}
		return
	}

	// On success, reset state
	cb.failureCount = 0
	cb.state = StateClosed
}
