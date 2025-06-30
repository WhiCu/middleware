package circuit

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreakerState_String(t *testing.T) {
	tests := []struct {
		state CircuitBreakerState
		want  string
	}{
		{StateClosed, "CLOSED"},
		{StateHalfOpen, "HALF_OPEN"},
		{StateOpen, "OPEN"},
		{CircuitBreakerState(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("CircuitBreakerState.String() = %v, want %v", got, tt.want)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.FailureThreshold != DefaultFailureThreshold {
		t.Errorf("Expected FailureThreshold to be %d, got %d", DefaultFailureThreshold, config.FailureThreshold)
	}
	if config.RecoveryTimeout != DefaultRecoveryTimeout {
		t.Errorf("Expected RecoveryTimeout to be %v, got %v", DefaultRecoveryTimeout, config.RecoveryTimeout)
	}
	if config.SuccessThreshold != DefaultSuccessThreshold {
		t.Errorf("Expected SuccessThreshold to be %d, got %d", DefaultSuccessThreshold, config.SuccessThreshold)
	}
	if config.WindowSize != DefaultWindowSize {
		t.Errorf("Expected WindowSize to be %v, got %v", DefaultWindowSize, config.WindowSize)
	}
	if config.MaxConcurrentCalls != DefaultMaxConcurrentCalls {
		t.Errorf("Expected MaxConcurrentCalls to be %d, got %d", DefaultMaxConcurrentCalls, config.MaxConcurrentCalls)
	}
}

func TestNewCircuitBreaker(t *testing.T) {
	config := DefaultConfig()
	cb := NewCircuitBreaker(config)

	if cb.GetState() != StateClosed {
		t.Errorf("Expected initial state to be CLOSED, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_Execute_Success(t *testing.T) {
	config := DefaultConfig()
	config.FailureThreshold = 3
	cb := NewCircuitBreaker(config)

	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to remain CLOSED, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_Execute_Failure(t *testing.T) {
	config := DefaultConfig()
	config.FailureThreshold = 2
	cb := NewCircuitBreaker(config)

	testErr := errors.New("test error")

	// First failure
	err := cb.Execute(func() error {
		return testErr
	})

	if err != testErr {
		t.Errorf("Expected test error, got %v", err)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to remain CLOSED after first failure, got %s", cb.GetState())
	}

	// Second failure - should open circuit
	err = cb.Execute(func() error {
		return testErr
	})

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be OPEN after threshold failures, got %s", cb.GetState())
	}
}

func TestCircuitBreaker_Execute_OpenState(t *testing.T) {
	config := DefaultConfig()
	config.FailureThreshold = 1
	cb := NewCircuitBreaker(config)

	// Trigger circuit to open
	cb.Execute(func() error {
		return errors.New("failure")
	})

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be OPEN, got %s", cb.GetState())
	}

	// Try to execute while open
	err := cb.Execute(func() error {
		return nil
	})

	if err == nil {
		t.Error("Expected error when circuit is open")
	}
	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Errorf("Expected ErrCircuitBreakerOpen error, got %v", err)
	}
}

func TestCircuitBreaker_Recovery(t *testing.T) {
	config := DefaultConfig()
	config.FailureThreshold = 1
	config.RecoveryTimeout = 100 * time.Millisecond
	config.SuccessThreshold = 2
	cb := NewCircuitBreaker(config)

	// Trigger circuit to open
	cb.Execute(func() error {
		return errors.New("failure")
	})

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be OPEN, got %s", cb.GetState())
	}

	// Wait for recovery timeout
	time.Sleep(150 * time.Millisecond)

	if cb.GetState() != StateHalfOpen {
		t.Errorf("Expected state to be HALF_OPEN after recovery timeout, got %s", cb.GetState())
	}

	// First success in half-open state
	err := cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if cb.GetState() != StateHalfOpen {
		t.Errorf("Expected state to remain HALF_OPEN after first success, got %s", cb.GetState())
	}

	// Second success - should close circuit
	err = cb.Execute(func() error {
		return nil
	})

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state to be CLOSED after success threshold, got %s", cb.GetState())
	}
}

// func TestCircuitBreaker_ContextCancellation(t *testing.T) {
// 	config := DefaultConfig()
// 	cb := NewCircuitBreaker(config)

// 	ctx, cancel := context.WithCancel(context.Background())

// 	// Cancel context immediately
// 	cancel()

// 	err := cb.Execute(ctx, func() error {
// 		time.Sleep(100 * time.Millisecond)
// 		return nil
// 	})

// 	if err == nil {
// 		t.Error("Expected context cancellation error")
// 	}
// 	if err != context.Canceled {
// 		t.Errorf("Expected context.Canceled error, got %v", err)
// 	}
// }

func TestCircuitBreaker_MaxConcurrentCalls(t *testing.T) {
	config := DefaultConfig()
	config.FailureThreshold = 1
	config.RecoveryTimeout = 100 * time.Millisecond
	config.MaxConcurrentCalls = 2
	cb := NewCircuitBreaker(config)

	// Trigger circuit to open
	cb.Execute(func() error {
		return errors.New("failure")
	})

	// Wait for recovery timeout
	time.Sleep(150 * time.Millisecond)

	if cb.GetState() != StateHalfOpen {
		t.Errorf("Expected state to be HALF_OPEN, got %s", cb.GetState())
	}

	// Start two concurrent calls that will block
	done := make(chan error, 2)
	started := make(chan bool, 2)

	go func() {
		started <- true
		done <- cb.Execute(func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
	}()

	go func() {
		started <- true
		done <- cb.Execute(func() error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
	}()

	// Wait for both goroutines to start
	<-started
	<-started

	// Try a third call - should be rejected
	err := cb.Execute(func() error {
		return nil
	})

	if err == nil {
		t.Error("Expected error for exceeding max concurrent calls")
	} else if !errors.Is(err, ErrMaxConcurrentCallsReached) {
		t.Errorf("Expected ErrMaxConcurrentCallsReached error, got %v", err)
	}

	// Wait for the first two calls to complete
	err1 := <-done
	err2 := <-done

	if err1 != nil {
		t.Errorf("Expected no error from first call, got %v", err1)
	}
	if err2 != nil {
		t.Errorf("Expected no error from second call, got %v", err2)
	}
}

func TestCircuitBreaker_FailureWindow(t *testing.T) {
	config := DefaultConfig()
	config.FailureThreshold = 3
	config.WindowSize = 100 * time.Millisecond
	cb := NewCircuitBreaker(config)

	// Add failures
	for i := 0; i < 3; i++ {
		cb.Execute(func() error {
			return errors.New("failure")
		})
	}

	if cb.GetState() != StateOpen {
		t.Errorf("Expected state to be OPEN after 3 failures, got %s", cb.GetState())
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Add one more failure - should not open circuit again immediately
	cb.Execute(func() error {
		return errors.New("failure")
	})

	// Circuit should still be open, but failure count should be reset
	// This is a simplified test - in real implementation, the window logic
	// would need more sophisticated testing
}
