// Package healer provides automatic panic detection, AI-powered fix generation,
// and GitHub pull request creation for Go applications.
package healer

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ajeet-kumar1087/go-code-healer/ai"
	"github.com/ajeet-kumar1087/go-code-healer/internal"
)

// Config is an alias to internal.Config for backward compatibility
type Config = internal.Config

// ProviderManager is an alias to ai.ProviderManager
type ProviderManager = ai.ProviderManager

// DefaultConfig returns a Config with default values
func DefaultConfig() Config {
	return internal.DefaultConfig()
}

// LoadConfig loads configuration from JSON file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	return internal.LoadConfig(configPath)
}

// GetFallbackConfig returns a minimal configuration that disables features when required settings are missing
func GetFallbackConfig() *Config {
	return internal.GetFallbackConfig()
}

// Healer is the main struct that manages error healing
type Healer struct {
	config          Config
	errorQueue      chan PanicEvent
	providerManager *ProviderManager
	gitClient       GitClient
	logger          Logger
	workerPool      *WorkerPool
	queueManager    *QueueManager
	retryManager    *RetryManager
	circuitBreaker  *CircuitBreaker
	panicCapture    *PanicCapture
	ctx             context.Context
	cancel          context.CancelFunc
}

// Initialize creates and starts the healer with the given configuration
func Initialize(config Config) (*Healer, error) {
	// Apply defaults and validate configuration
	config.ApplyDefaults()
	if err := config.ValidateComplete(); err != nil {
		return nil, err
	}

	// Create context for lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	// Create logger
	logger := internal.NewDefaultLogger(config.LogLevel)

	// Create healer instance
	healer := &Healer{
		config:     config,
		errorQueue: make(chan PanicEvent, config.MaxQueueSize),
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Initialize provider manager with multi-AI support and MCP
	if config.Enabled {
		providerManager, err := ai.NewProviderManager(config, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create provider manager: %w", err)
		}
		healer.providerManager = providerManager
		logger.Info("Provider manager initialized with AI providers and MCP support")

		// Validate providers
		if err := providerManager.ValidateProviders(); err != nil {
			logger.Warn("Provider validation warnings: %v", err)
		}
	} else {
		logger.Info("Healer disabled - skipping provider initialization")
	}

	// Initialize Git client if enabled and configured
	if config.Enabled && config.GitHubToken != "" && config.RepoOwner != "" && config.RepoName != "" {
		healer.gitClient = NewGitHubClient(config.GitHubToken, config.RepoOwner, config.RepoName, logger)
		logger.Info("Git client initialized for repository: %s/%s", config.RepoOwner, config.RepoName)
	} else {
		logger.Info("Git client disabled - missing GitHub token, repo owner, or repo name")
	}

	// Create queue manager
	healer.queueManager = NewQueueManager(healer, logger)

	// Create retry manager with configuration from healer config
	retryConfig := RetryConfig{
		MaxAttempts:   config.RetryAttempts,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}
	healer.retryManager = NewRetryManager(retryConfig, logger)

	// Create circuit breaker
	healer.circuitBreaker = NewCircuitBreaker(DefaultCircuitBreakerConfig(), logger)

	// Create worker pool
	healer.workerPool = NewWorkerPool(healer, logger)

	// Set as global healer for panic handling
	SetGlobalHealer(healer)

	return healer, nil
}

// Start begins background processing of errors
func (h *Healer) Start() error {
	if !h.config.Enabled {
		h.logger.Info("Healer is disabled, skipping background processing")
		return nil
	}

	h.logger.Info("Starting healer background processing")

	// Start the worker pool
	if err := h.workerPool.Start(); err != nil {
		return err
	}

	h.logger.Info("Healer started successfully")
	return nil
}

// Stop gracefully shuts down the healer
func (h *Healer) Stop() error {
	h.logger.Info("Stopping healer")

	// Cancel context to signal shutdown
	h.cancel()

	// Stop the worker pool
	if h.workerPool != nil {
		if err := h.workerPool.Stop(); err != nil {
			h.logger.Error("Error stopping worker pool: %v", err)
			return err
		}
	}

	h.logger.Info("Healer stopped successfully")
	return nil
}

// InstallPanicHandler sets up the global panic handler
// This method configures the healer to capture panics when they occur.
// Due to Go's design, automatic panic capture requires explicit defer statements
// in your code. Use the provided helper functions for panic capture.
func (h *Healer) InstallPanicHandler() {
	if h.panicCapture == nil {
		h.panicCapture = NewPanicCapture(h, h.logger)
	}

	// Install the panic handler
	h.panicCapture.InstallHandler()

	if h.logger != nil {
		h.logger.Info("Panic handler installed successfully")
		h.logger.Info("Use the following functions to capture panics in your application:")
		h.logger.Info("  - defer healer.HandlePanic() // Captures panic and re-panics (preserves existing behavior)")
		h.logger.Info("  - defer healer.RecoverAndHandle() // Captures panic and recovers gracefully")
		h.logger.Info("  - healer.WrapFunction(fn) // Returns a wrapped function with panic capture")
		h.logger.Info("  - healer.WrapFunctionWithRecovery(fn) // Returns a wrapped function with graceful recovery")
	}
}

// RestorePanicHandler restores the original panic handling behavior
// This method provides cleanup functionality to restore original handlers
func (h *Healer) RestorePanicHandler() {
	// Clear the global healer to disable panic capture
	SetGlobalHealer(nil)

	if h.logger != nil {
		h.logger.Info("Panic handler restored to original state")
	}
}

// Global healer instance for panic handling
var globalHealer *Healer

// SetGlobalHealer sets the global healer instance for panic handling
func SetGlobalHealer(healer *Healer) {
	globalHealer = healer
}

// HandlePanic should be called in defer statements to capture panics
// Usage: defer healer.HandlePanic()
func HandlePanic() {
	if r := recover(); r != nil {
		if globalHealer != nil && globalHealer.panicCapture != nil {
			// Capture the panic for processing
			globalHealer.panicCapture.CapturePanic(r)
		}

		// Re-panic to maintain normal panic behavior
		panic(r)
	}
}

// RecoverAndHandle captures panics and handles them without re-panicking
// Usage: defer healer.RecoverAndHandle()
func RecoverAndHandle() {
	if r := recover(); r != nil {
		if globalHealer != nil && globalHealer.panicCapture != nil {
			// Capture the panic for processing
			globalHealer.panicCapture.CapturePanic(r)
		}

		// Log the panic but don't re-panic (graceful recovery)
		if globalHealer != nil && globalHealer.logger != nil {
			globalHealer.logger.Error("Recovered from panic: %v", r)
		}
	}
}

// WrapFunction wraps a function to automatically capture any panics
func WrapFunction(fn func()) func() {
	return func() {
		defer HandlePanic()
		fn()
	}
}

// WrapFunctionWithRecovery wraps a function to capture panics without re-panicking
func WrapFunctionWithRecovery(fn func()) func() {
	return func() {
		defer RecoverAndHandle()
		fn()
	}
}

// InstallGlobalPanicHandler is a convenience function that initializes a healer
// with the provided configuration and installs the panic handler
func InstallGlobalPanicHandler(config Config) (*Healer, error) {
	healer, err := Initialize(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize healer: %w", err)
	}

	// Start the healer
	if err := healer.Start(); err != nil {
		return nil, fmt.Errorf("failed to start healer: %w", err)
	}

	// Install the panic handler
	healer.InstallPanicHandler()

	return healer, nil
}

// MustInstallGlobalPanicHandler is like InstallGlobalPanicHandler but panics on error
func MustInstallGlobalPanicHandler(config Config) *Healer {
	healer, err := InstallGlobalPanicHandler(config)
	if err != nil {
		panic(fmt.Sprintf("failed to install global panic handler: %v", err))
	}
	return healer
}

// GetGlobalHealer returns the current global healer instance
func GetGlobalHealer() *Healer {
	return globalHealer
}

// IsGlobalHealerInstalled returns true if a global healer is currently installed
func IsGlobalHealerInstalled() bool {
	return globalHealer != nil
}

// WrapFunctionWithArgs wraps a function that takes arguments
func WrapFunctionWithArgs(fn func(...any)) func(...any) {
	return func(args ...any) {
		defer HandlePanic()
		fn(args...)
	}
}

// WrapFunctionWithArgsAndRecovery wraps a function that takes arguments with graceful recovery
func WrapFunctionWithArgsAndRecovery(fn func(...any)) func(...any) {
	return func(args ...any) {
		defer RecoverAndHandle()
		fn(args...)
	}
}

// WrapHTTPHandler wraps an HTTP handler function with panic capture
func WrapHTTPHandler(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer RecoverAndHandle() // Use graceful recovery for HTTP handlers
		handler(w, r)
	}
}

// SafeGoroutine starts a goroutine with panic capture and recovery
func SafeGoroutine(fn func()) {
	go func() {
		defer RecoverAndHandle()
		fn()
	}()
}

// Example usage documentation:
//
// Basic setup:
//   config := healer.Config{
//       OpenAIAPIKey: "your-openai-key",
//       GitHubToken:  "your-github-token",
//       RepoOwner:    "your-username",
//       RepoName:     "your-repo",
//       Enabled:      true,
//   }
//
//   healer, err := healer.InstallGlobalPanicHandler(config)
//   if err != nil {
//       log.Fatal(err)
//   }
//   defer healer.Stop()
//
// Manual panic capture in functions:
//   func riskyFunction() {
//       defer healer.HandlePanic() // Captures panic and re-panics
//       // ... your code that might panic
//   }
//
//   func gracefulFunction() {
//       defer healer.RecoverAndHandle() // Captures panic and recovers
//       // ... your code that might panic
//   }
//
// Wrapping functions:
//   safeFunc := healer.WrapFunction(riskyFunction)
//   safeFunc() // Will capture any panics from riskyFunction
//
// HTTP handlers:
//   http.HandleFunc("/api", healer.WrapHTTPHandler(myHandler))
//
// Goroutines:
//   healer.SafeGoroutine(func() {
//       // This goroutine will capture and handle panics gracefully
//   })

// GetQueueStats returns statistics about the queue
func (h *Healer) GetQueueStats() map[string]any {
	stats := make(map[string]any)

	// Queue size information
	stats["queue_capacity"] = cap(h.errorQueue)
	stats["queue_length"] = len(h.errorQueue)
	stats["queue_available"] = cap(h.errorQueue) - len(h.errorQueue)

	// Dropped events count
	if h.queueManager != nil {
		stats["dropped_events"] = h.queueManager.GetDroppedCount()
	}

	// Worker pool information
	if h.workerPool != nil {
		stats["worker_count"] = h.workerPool.GetWorkerCount()
		stats["workers_running"] = h.workerPool.IsRunning()
	}

	// Circuit breaker status
	if h.circuitBreaker != nil {
		stats["circuit_breaker_state"] = h.circuitBreaker.GetState().String()
		stats["circuit_breaker_failures"] = h.circuitBreaker.GetFailureCount()
	}

	return stats
}

// GetStatus returns the current status of the healer
func (h *Healer) GetStatus() map[string]any {
	status := make(map[string]any)

	status["enabled"] = h.config.Enabled
	status["running"] = h.workerPool != nil && h.workerPool.IsRunning()

	// Add configuration info
	status["config"] = map[string]any{
		"max_queue_size": h.config.MaxQueueSize,
		"worker_count":   h.config.WorkerCount,
		"retry_attempts": h.config.RetryAttempts,
		"log_level":      h.config.LogLevel,
	}

	// Add queue statistics
	status["queue_stats"] = h.GetQueueStats()

	return status
}

// ResetCircuitBreaker manually resets the circuit breaker
func (h *Healer) ResetCircuitBreaker() {
	if h.circuitBreaker != nil {
		h.circuitBreaker.Reset()
		if h.logger != nil {
			h.logger.Info("Circuit breaker reset manually")
		}
	}
}

// GetQueueManager returns the queue manager (implements HealerInterface)
func (h *Healer) GetQueueManager() QueueManagerInterface {
	return h.queueManager
}

// GetErrorQueue returns the error queue (implements HealerInterface)
func (h *Healer) GetErrorQueue() chan PanicEvent {
	return h.errorQueue
}

// CreateAISession creates a new AI session for comprehensive error analysis and fixing
func (h *Healer) CreateAISession() *ai.SessionManager {
	if h.providerManager == nil {
		return nil
	}
	return h.providerManager.CreateSession(h.gitClient)
}

// ProcessErrorWithSession processes an error using the session-based approach
func (h *Healer) ProcessErrorWithSession(ctx context.Context, panicEvent PanicEvent) (*ai.SessionResult, error) {
	if h.providerManager == nil {
		return nil, fmt.Errorf("provider manager not initialized")
	}

	// Create AI session
	session := h.CreateAISession()
	if session == nil {
		return nil, fmt.Errorf("failed to create AI session")
	}

	// Convert PanicEvent to ErrorInfo
	errorInfo := &ai.ErrorInfo{
		Error:      fmt.Sprintf("%v", panicEvent.Error),
		StackTrace: panicEvent.StackTrace,
		SourceFile: panicEvent.SourceFile,
		LineNumber: panicEvent.LineNumber,
		Function:   panicEvent.Function,
		Timestamp:  panicEvent.Timestamp,
		Severity:   "high", // Default severity
	}

	// Create code context (this could be enhanced with actual source code extraction)
	codeContext := &ai.CodeContext{
		SourceCode:   "// Source code would be extracted here",
		RelatedFiles: []string{panicEvent.SourceFile},
		FunctionSig:  panicEvent.Function,
	}

	// Initiate comprehensive session
	return session.InitiateSession(ctx, errorInfo, codeContext)
}

// GetProviderStatus returns status of AI providers and MCP
func (h *Healer) GetProviderStatus() map[string]interface{} {
	if h.providerManager == nil {
		return map[string]interface{}{
			"enabled": false,
			"reason":  "provider manager not initialized",
		}
	}
	return h.providerManager.GetProviderStatus()
}
