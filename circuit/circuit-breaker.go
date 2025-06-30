package circuit

import (
	"sync"
	"sync/atomic"
	"time"
)

// CircuitBreaker represents a circuit breaker instance
type CircuitBreaker struct {
	config CircuitBreakerConfig
	state  atomic.Int32
	mu     sync.RWMutex

	// Failure tracking
	failTimestamps []int64
	nextIndex      atomic.Uint32

	// Success tracking in half-open state
	successCount atomic.Uint32
	concurrent   atomic.Uint32

	// Recovery timer
	recoveryTimer *time.Timer
}

// NewCircuitBreaker creates a new circuit breaker instance
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	cb := &CircuitBreaker{
		config:         config,
		failTimestamps: make([]int64, config.FailureThreshold),
	}
	cb.state.Store(int32(StateClosed))
	return cb
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return CircuitBreakerState(cb.state.Load())
}

func (cb *CircuitBreaker) Execute(fn func() error) error {

	state := cb.GetState()

	switch state {
	case StateOpen:
		return ErrCircuitBreakerOpen

	case StateHalfOpen:
		if cb.concurrent.Load() >= cb.config.MaxConcurrentCalls {
			return ErrMaxConcurrentCallsReached
		}
		cb.concurrent.Add(1)
		defer cb.concurrent.Add(^uint32(0)) // Decrement
	}

	err := fn()
	cb.recordResult(err)
	return err
}

// recordResult records the result of an operation and updates circuit breaker state
func (cb *CircuitBreaker) recordResult(err error) {

	now := time.Now().UnixNano()

	switch cb.GetState() {
	case StateClosed:
		if err != nil {
			cb.recordFailure(now)
			if cb.shouldOpen(now) {
				cb.transitionTo(StateOpen)
			}
		}

	case StateHalfOpen:
		if err != nil {
			cb.transitionTo(StateOpen)
		}
		if cb.successCount.Add(1) >= cb.config.SuccessThreshold {
			cb.transitionTo(StateClosed)
		}

	}
}

// recordFailure records a failure and maintains the failure window
func (cb *CircuitBreaker) recordFailure(ut int64) {
	idx := cb.nextIndex.Add(1) - 1 // атомарно увеличиваем и получаем индекс
	cb.failTimestamps[idx%uint32(len(cb.failTimestamps))] = ut
}

// shouldOpen determines if the circuit breaker should open
func (cb *CircuitBreaker) shouldOpen(now int64) bool {
	// Нет необходимости в блокировке, т.к. failTimestamps и nextIndex атомарны
	idx := cb.nextIndex.Load() % uint32(len(cb.failTimestamps))
	oldest := cb.failTimestamps[idx]

	if cb.nextIndex.Load() >= uint32(len(cb.failTimestamps)) && oldest > 0 {
		return time.Duration(now-oldest) <= cb.config.WindowSize
	}
	return false
}

// transitionTo changes the circuit breaker state
func (cb *CircuitBreaker) transitionTo(newState CircuitBreakerState) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.GetState()
	if oldState == newState {
		return
	}

	cb.state.Store(int32(newState))

	switch newState {
	case StateOpen:
		// Schedule transition to half-open
		if cb.recoveryTimer != nil {
			cb.recoveryTimer.Stop()
		}
		cb.recoveryTimer = time.AfterFunc(cb.config.RecoveryTimeout, func() {
			cb.transitionTo(StateHalfOpen)
		})
	case StateClosed:
		// Reset counters
		cb.Reset()

	case StateHalfOpen:
		// Reset success counter for half-open state
		cb.successCount.Store(0)
	}
}

func (cb *CircuitBreaker) Reset() {
	cb.successCount.Store(0)
	if cb.recoveryTimer != nil {
		cb.recoveryTimer.Stop()
		cb.recoveryTimer = nil
	}
	for i := range cb.failTimestamps {
		cb.failTimestamps[i] = 0
	}
	cb.nextIndex.Store(0)
}
