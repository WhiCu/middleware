package circuit

import "errors"

// CircuitBreaker errors
var (
	// ErrCircuitBreakerOpen is returned when the circuit breaker is in open state
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

	// ErrMaxConcurrentCallsReached is returned when max concurrent calls limit is reached in half-open state
	ErrMaxConcurrentCallsReached = errors.New("max concurrent calls reached in half-open state")
)
