package healer

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// QueueManager handles queue overflow and management
type QueueManager struct {
	healer       *Healer
	logger       Logger
	mu           sync.RWMutex
	droppedCount int64
}

// NewQueueManager creates a new queue manager
func NewQueueManager(healer *Healer, logger Logger) *QueueManager {
	return &QueueManager{
		healer: healer,
		logger: logger,
	}
}

// EnqueueEvent attempts to enqueue a panic event with overflow handling
func (qm *QueueManager) EnqueueEvent(event PanicEvent) bool {
	select {
	case qm.healer.errorQueue <- event:
		if qm.logger != nil {
			qm.logger.Debug("Event %s enqueued successfully", event.ID)
		}
		return true
	default:
		// Queue is full, handle overflow
		return qm.handleQueueOverflow(event)
	}
}

// handleQueueOverflow implements oldest-item dropping strategy
func (qm *QueueManager) handleQueueOverflow(newEvent PanicEvent) bool {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	// Try to drop the oldest item and add the new one
	select {
	case oldEvent := <-qm.healer.errorQueue:
		qm.droppedCount++
		if qm.logger != nil {
			qm.logger.Warn("Queue overflow: dropped oldest event %s to make room for %s", oldEvent.ID, newEvent.ID)
		}

		// Now try to add the new event
		select {
		case qm.healer.errorQueue <- newEvent:
			if qm.logger != nil {
				qm.logger.Debug("New event %s enqueued after dropping oldest", newEvent.ID)
			}
			return true
		default:
			// Still couldn't add, this shouldn't happen but handle it
			if qm.logger != nil {
				qm.logger.Error("Failed to enqueue event %s even after dropping oldest", newEvent.ID)
			}
			return false
		}
	default:
		// Queue became empty while we were waiting, try again
		select {
		case qm.healer.errorQueue <- newEvent:
			if qm.logger != nil {
				qm.logger.Debug("Event %s enqueued on retry", newEvent.ID)
			}
			return true
		default:
			qm.droppedCount++
			if qm.logger != nil {
				qm.logger.Error("Failed to enqueue event %s, queue still full", newEvent.ID)
			}
			return false
		}
	}
}

// GetDroppedCount returns the number of events dropped due to queue overflow
func (qm *QueueManager) GetDroppedCount() int64 {
	qm.mu.RLock()
	defer qm.mu.RUnlock()
	return qm.droppedCount
}

// RetryConfig holds configuration for retry logic
type RetryConfig struct {
	MaxAttempts   int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}
}

// RetryManager handles retry logic with exponential backoff
type RetryManager struct {
	config RetryConfig
	logger Logger
}

// NewRetryManager creates a new retry manager
func NewRetryManager(config RetryConfig, logger Logger) *RetryManager {
	return &RetryManager{
		config: config,
		logger: logger,
	}
}

// ExecuteWithRetry executes a function with exponential backoff retry
func (rm *RetryManager) ExecuteWithRetry(ctx context.Context, operation string, fn func() error) error {
	var lastErr error
	delay := rm.config.InitialDelay

	for attempt := 1; attempt <= rm.config.MaxAttempts; attempt++ {
		if rm.logger != nil {
			rm.logger.Debug("Attempting %s (attempt %d/%d)", operation, attempt, rm.config.MaxAttempts)
		}

		err := fn()
		if err == nil {
			if attempt > 1 && rm.logger != nil {
				rm.logger.Info("%s succeeded on attempt %d", operation, attempt)
			}
			return nil
		}

		lastErr = err
		if rm.logger != nil {
			rm.logger.Warn("%s failed on attempt %d: %v", operation, attempt, err)
		}

		// Don't wait after the last attempt
		if attempt == rm.config.MaxAttempts {
			break
		}

		// Wait with exponential backoff
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s cancelled: %w", operation, ctx.Err())
		case <-time.After(delay):
			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * rm.config.BackoffFactor)
			if delay > rm.config.MaxDelay {
				delay = rm.config.MaxDelay
			}
		}
	}

	return fmt.Errorf("%s failed after %d attempts, last error: %w", operation, rm.config.MaxAttempts, lastErr)
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// String returns the string representation of the circuit breaker state
func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitBreakerClosed:
		return "CLOSED"
	case CircuitBreakerOpen:
		return "OPEN"
	case CircuitBreakerHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold int
	RecoveryTimeout  time.Duration
	ResetTimeout     time.Duration
}

// DefaultCircuitBreakerConfig returns default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  30 * time.Second,
		ResetTimeout:     60 * time.Second,
	}
}

// CircuitBreaker implements the circuit breaker pattern for external API failures
type CircuitBreaker struct {
	config       CircuitBreakerConfig
	state        CircuitBreakerState
	failures     int
	lastFailTime time.Time
	logger       Logger
	mu           sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig, logger Logger) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  CircuitBreakerClosed,
		logger: logger,
	}
}

// Execute executes a function through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, operation string, fn func() error) error {
	if !cb.canExecute() {
		return fmt.Errorf("circuit breaker is OPEN for %s", operation)
	}

	err := fn()
	cb.recordResult(operation, err)
	return err
}

// canExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitBreakerClosed:
		return true
	case CircuitBreakerOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailTime) > cb.config.RecoveryTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			// Double-check after acquiring write lock
			if cb.state == CircuitBreakerOpen && time.Since(cb.lastFailTime) > cb.config.RecoveryTimeout {
				cb.state = CircuitBreakerHalfOpen
				if cb.logger != nil {
					cb.logger.Info("Circuit breaker transitioning to HALF_OPEN")
				}
			}
			cb.mu.Unlock()
			cb.mu.RLock()
			return cb.state == CircuitBreakerHalfOpen
		}
		return false
	case CircuitBreakerHalfOpen:
		return true
	default:
		return false
	}
}

// recordResult records the result of an operation and updates circuit breaker state
func (cb *CircuitBreaker) recordResult(operation string, err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailTime = time.Now()

		if cb.logger != nil {
			cb.logger.Debug("Circuit breaker recorded failure for %s (failures: %d)", operation, cb.failures)
		}

		// Check if we should open the circuit
		if cb.failures >= cb.config.FailureThreshold {
			if cb.state != CircuitBreakerOpen {
				cb.state = CircuitBreakerOpen
				if cb.logger != nil {
					cb.logger.Warn("Circuit breaker OPENED for %s after %d failures", operation, cb.failures)
				}
			}
		} else if cb.state == CircuitBreakerHalfOpen {
			// Failed in half-open state, go back to open
			cb.state = CircuitBreakerOpen
			if cb.logger != nil {
				cb.logger.Warn("Circuit breaker returned to OPEN state for %s", operation)
			}
		}
	} else {
		// Success
		if cb.logger != nil {
			cb.logger.Debug("Circuit breaker recorded success for %s", operation)
		}

		if cb.state == CircuitBreakerHalfOpen {
			// Success in half-open state, close the circuit
			cb.state = CircuitBreakerClosed
			cb.failures = 0
			if cb.logger != nil {
				cb.logger.Info("Circuit breaker CLOSED for %s after successful operation", operation)
			}
		} else if cb.state == CircuitBreakerClosed {
			// Reset failure count on success
			cb.failures = 0
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetFailureCount returns the current failure count
func (cb *CircuitBreaker) GetFailureCount() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitBreakerClosed
	cb.failures = 0

	if cb.logger != nil {
		cb.logger.Info("Circuit breaker manually reset to CLOSED state")
	}
}
