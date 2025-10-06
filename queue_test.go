package healer

import (
	"context"
	"testing"
	"time"
)

func TestQueueManager_EnqueueEvent(t *testing.T) {
	// Create a small queue for testing overflow
	config := DefaultConfig()
	config.MaxQueueSize = 2
	config.LogLevel = "debug"
	config.Enabled = false // Disable to avoid API key requirements

	healer, err := Initialize(config)
	if err != nil {
		t.Fatalf("Failed to initialize healer: %v", err)
	}

	// Test normal enqueue
	event1 := PanicEvent{ID: "test1", Error: "test error 1"}
	success := healer.queueManager.EnqueueEvent(event1)
	if !success {
		t.Error("Expected first event to be enqueued successfully")
	}

	// Test second enqueue
	event2 := PanicEvent{ID: "test2", Error: "test error 2"}
	success = healer.queueManager.EnqueueEvent(event2)
	if !success {
		t.Error("Expected second event to be enqueued successfully")
	}

	// Test overflow handling - this should drop the oldest and add the new one
	event3 := PanicEvent{ID: "test3", Error: "test error 3"}
	success = healer.queueManager.EnqueueEvent(event3)
	if !success {
		t.Error("Expected third event to be enqueued after dropping oldest")
	}

	// Check that one event was dropped
	droppedCount := healer.queueManager.GetDroppedCount()
	if droppedCount != 1 {
		t.Errorf("Expected 1 dropped event, got %d", droppedCount)
	}
}

func TestRetryManager_ExecuteWithRetry(t *testing.T) {
	logger := NewDefaultLogger("debug")
	retryManager := NewRetryManager(DefaultRetryConfig(), logger)

	ctx := context.Background()
	attempts := 0

	// Test successful operation
	err := retryManager.ExecuteWithRetry(ctx, "test-operation", func() error {
		attempts++
		if attempts < 2 {
			return &testError{"temporary failure"}
		}
		return nil
	})

	if err != nil {
		t.Errorf("Expected operation to succeed after retry, got error: %v", err)
	}

	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestCircuitBreaker_Execute(t *testing.T) {
	logger := NewDefaultLogger("debug")
	config := CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  100 * time.Millisecond,
		ResetTimeout:     200 * time.Millisecond,
	}
	cb := NewCircuitBreaker(config, logger)

	ctx := context.Background()

	// Test initial state (should be closed)
	if cb.GetState() != CircuitBreakerClosed {
		t.Error("Expected circuit breaker to start in CLOSED state")
	}

	// Cause failures to open the circuit
	for i := 0; i < 2; i++ {
		err := cb.Execute(ctx, "test-op", func() error {
			return &testError{"failure"}
		})
		if err == nil {
			t.Error("Expected error from failing operation")
		}
	}

	// Circuit should now be open
	if cb.GetState() != CircuitBreakerOpen {
		t.Error("Expected circuit breaker to be OPEN after failures")
	}

	// Attempt should fail immediately
	err := cb.Execute(ctx, "test-op", func() error {
		return nil // This shouldn't be called
	})
	if err == nil {
		t.Error("Expected circuit breaker to reject operation when OPEN")
	}

	// Wait for recovery timeout
	time.Sleep(150 * time.Millisecond)

	// Should transition to half-open and allow one attempt
	err = cb.Execute(ctx, "test-op", func() error {
		return nil // Success
	})
	if err != nil {
		t.Errorf("Expected successful operation in HALF_OPEN state, got: %v", err)
	}

	// Should now be closed
	if cb.GetState() != CircuitBreakerClosed {
		t.Error("Expected circuit breaker to be CLOSED after successful operation")
	}
}

// testError is a simple error type for testing
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}
