// Package healer provides automatic panic detection, AI-powered fix generation,
// and GitHub pull request creation for Go applications.
package healer

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Config holds the configuration for the healer
type Config struct {
	// AI Configuration
	OpenAIAPIKey string `json:"openai_api_key"`
	OpenAIModel  string `json:"openai_model,omitempty"` // defaults to "gpt-4"

	// GitHub Configuration
	GitHubToken string `json:"github_token"`
	RepoOwner   string `json:"repo_owner"`
	RepoName    string `json:"repo_name"`

	// Processing Configuration
	Enabled       bool `json:"enabled"`
	MaxQueueSize  int  `json:"max_queue_size,omitempty"` // defaults to 100
	WorkerCount   int  `json:"worker_count,omitempty"`   // defaults to 2
	RetryAttempts int  `json:"retry_attempts,omitempty"` // defaults to 3

	// Logging Configuration
	LogLevel string `json:"log_level,omitempty"` // defaults to "info"
}

// ApplyDefaults applies default values to unset fields
func (c *Config) ApplyDefaults() {
	if c.OpenAIModel == "" {
		c.OpenAIModel = "gpt-4"
	}
	if c.MaxQueueSize == 0 {
		c.MaxQueueSize = 100
	}
	if c.WorkerCount == 0 {
		c.WorkerCount = 2
	}
	if c.RetryAttempts == 0 {
		c.RetryAttempts = 3
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	var errs []error

	// Check required fields when enabled
	if c.Enabled {
		if c.OpenAIAPIKey == "" {
			errs = append(errs, fmt.Errorf("OpenAI API key is required when healer is enabled"))
		}
		if c.GitHubToken == "" {
			errs = append(errs, fmt.Errorf("GitHub token is required when healer is enabled"))
		}
		if c.RepoOwner == "" {
			errs = append(errs, fmt.Errorf("repository owner is required when healer is enabled"))
		}
		if c.RepoName == "" {
			errs = append(errs, fmt.Errorf("repository name is required when healer is enabled"))
		}
	}

	// Validate numeric fields
	if c.MaxQueueSize <= 0 {
		errs = append(errs, fmt.Errorf("max queue size must be greater than 0"))
	}
	if c.WorkerCount <= 0 {
		errs = append(errs, fmt.Errorf("worker count must be greater than 0"))
	}
	if c.RetryAttempts < 0 {
		errs = append(errs, fmt.Errorf("retry attempts cannot be negative"))
	}

	// Validate log level
	validLogLevels := []string{"debug", "info", "warn", "error"}
	logLevelValid := false
	for _, level := range validLogLevels {
		if c.LogLevel == level {
			logLevelValid = true
			break
		}
	}
	if !logLevelValid {
		errs = append(errs, fmt.Errorf("invalid log level '%s', must be one of: %v", c.LogLevel, validLogLevels))
	}

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed: %v", errs)
	}
	return nil
}

// ValidateComplete performs comprehensive validation with clear error messages
func (c *Config) ValidateComplete() error {
	return c.Validate() // For now, same as basic validation
}

// DefaultConfig returns a Config with default values
func DefaultConfig() Config {
	return Config{
		OpenAIModel:   "gpt-4",
		Enabled:       true,
		MaxQueueSize:  100,
		WorkerCount:   2,
		RetryAttempts: 3,
		LogLevel:      "info",
	}
}

// LoadConfig loads configuration from JSON file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	// For now, just return the default config
	// TODO: Implement file loading if needed
	config.ApplyDefaults()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// GetFallbackConfig returns a minimal configuration that disables features when required settings are missing
func GetFallbackConfig() *Config {
	config := DefaultConfig()
	config.ApplyDefaults()

	// Disable if required settings are missing
	if config.OpenAIAPIKey == "" || config.GitHubToken == "" || config.RepoOwner == "" || config.RepoName == "" {
		config.Enabled = false
	}

	return &config
}

// LoadFromEnv loads configuration values from environment variables
func (c *Config) LoadFromEnv() error {
	// This is a placeholder implementation
	// In a real implementation, you would load from os.Getenv()
	return nil
}

// Healer is the main struct that manages error healing
type Healer struct {
	config         Config
	errorQueue     chan PanicEvent
	aiClient       AIClient
	gitClient      GitClient
	logger         Logger
	workerPool     *WorkerPool
	queueManager   *QueueManager
	retryManager   *RetryManager
	circuitBreaker *CircuitBreaker
	panicCapture   *PanicCapture
	ctx            context.Context
	cancel         context.CancelFunc
}

// Initialize creates and starts the healer with the given configuration
func Initialize(config Config) (*Healer, error) {
	// Apply defaults and validate configuration
	config.ApplyDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create context for lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	// Create logger
	logger := NewDefaultLogger(config.LogLevel)

	// Create healer instance
	healer := &Healer{
		config:     config,
		errorQueue: make(chan PanicEvent, config.MaxQueueSize),
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Initialize AI client if enabled and configured
	if config.Enabled && config.OpenAIAPIKey != "" {
		healer.aiClient = NewOpenAIClient(config.OpenAIAPIKey, config.OpenAIModel, logger)
		logger.Info("AI client initialized with model: %s", config.OpenAIModel)
	} else {
		logger.Info("AI client disabled - missing API key or healer disabled")
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
