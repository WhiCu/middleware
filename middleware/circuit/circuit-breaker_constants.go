package circuit

import "time"

// Default configuration constants
const (
	// DefaultFailureThreshold is the default number of failures before opening circuit
	DefaultFailureThreshold = 5

	// DefaultRecoveryTimeout is the default time to wait before attempting recovery
	DefaultRecoveryTimeout = 30 * time.Second

	// DefaultSuccessThreshold is the default number of successes in half-open state to close circuit
	DefaultSuccessThreshold = 3

	// DefaultWindowSize is the default time window for counting failures
	DefaultWindowSize = 60 * time.Second

	// DefaultMaxConcurrentCalls is the default maximum concurrent calls in half-open state
	DefaultMaxConcurrentCalls = 10
)

// HTTP status codes
const (
	// StatusServiceUnavailable is returned when circuit breaker is open
	StatusServiceUnavailable = 503
)

// HTTP headers
const (
	// HeaderCircuitBreakerState is the header that indicates circuit breaker state
	HeaderCircuitBreakerState = "X-Circuit-Breaker-State"
)
