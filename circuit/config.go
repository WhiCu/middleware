package circuit

import "time"

// CircuitBreakerConfig holds configuration for the circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold   uint32        // Number of failures before opening circuit
	RecoveryTimeout    time.Duration // Time to wait before attempting recovery
	SuccessThreshold   uint32        // Number of successes in half-open state to close circuit
	WindowSize         time.Duration // Time window for counting failures
	MaxConcurrentCalls uint32        // Maximum concurrent calls in half-open state
}

// DefaultConfig returns a default circuit breaker configuration
func DefaultConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:   DefaultFailureThreshold,
		RecoveryTimeout:    DefaultRecoveryTimeout,
		SuccessThreshold:   DefaultSuccessThreshold,
		WindowSize:         DefaultWindowSize,
		MaxConcurrentCalls: DefaultMaxConcurrentCalls,
	}
}
