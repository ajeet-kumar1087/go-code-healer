// Package healer provides automatic panic detection, AI-powered fix generation,
// and GitHub pull request creation for Go applications.
//
// The Go Code Healer is designed as a lightweight, importable Go package that provides
// automatic panic detection, AI-powered fix generation, and GitHub pull request creation.
// The architecture prioritizes non-blocking background processing to ensure zero impact
// on the host application's performance while providing intelligent error recovery capabilities.
//
// # Quick Start
//
// Basic setup with automatic panic handling:
//
//	config := healer.Config{
//	    OpenAIAPIKey: "your-openai-key",
//	    GitHubToken:  "your-github-token",
//	    RepoOwner:    "your-username",
//	    RepoName:     "your-repo",
//	    Enabled:      true,
//	}
//
//	healer, err := healer.InstallGlobalPanicHandler(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer healer.Stop()
//
// # Manual Panic Capture
//
// For more control over panic capture, you can manually add panic handling to specific functions:
//
//	func riskyFunction() {
//	    defer healer.HandlePanic() // Captures panic and re-panics
//	    // ... your code that might panic
//	}
//
//	func gracefulFunction() {
//	    defer healer.RecoverAndHandle() // Captures panic and recovers
//	    // ... your code that might panic
//	}
//
// # Function Wrapping
//
// You can wrap existing functions to add automatic panic capture:
//
//	safeFunc := healer.WrapFunction(riskyFunction)
//	safeFunc() // Will capture any panics from riskyFunction
//
//	// For HTTP handlers
//	http.HandleFunc("/api", healer.WrapHTTPHandler(myHandler))
//
//	// For goroutines
//	healer.SafeGoroutine(func() {
//	    // This goroutine will capture and handle panics gracefully
//	})
//
// # Configuration
//
// The healer can be configured through environment variables or configuration files:
//
// Environment Variables:
//   - HEALER_OPENAI_API_KEY: OpenAI API key for fix generation
//   - HEALER_GITHUB_TOKEN: GitHub token for PR creation
//   - HEALER_REPO_OWNER: GitHub repository owner
//   - HEALER_REPO_NAME: GitHub repository name
//   - HEALER_ENABLED: Enable/disable the healer (true/false)
//   - HEALER_MAX_QUEUE_SIZE: Maximum queue size (default: 100)
//   - HEALER_WORKER_COUNT: Number of background workers (default: 2)
//   - HEALER_RETRY_ATTEMPTS: Number of retry attempts (default: 3)
//   - HEALER_LOG_LEVEL: Logging level (debug, info, warn, error)
//
// # Architecture
//
// The healer consists of several key components:
//
//   - Panic Interceptor: Captures runtime panics without interfering with existing recovery
//   - Background Queue: Manages async processing of captured errors
//   - AI Fix Generator: Communicates with OpenAI to generate code fixes
//   - Git Manager: Handles branch creation, commits, and PR operations
//   - Configuration Manager: Manages API keys and settings
//
// All processing happens in background goroutines to ensure zero impact on your application's performance.
//
// # Error Handling
//
// The healer is designed to be fault-tolerant:
//
//   - If AI services are unavailable, panics are still logged
//   - If GitHub is unavailable, fixes are generated but not submitted
//   - If the healer itself encounters errors, it logs them but never crashes your application
//   - All operations have configurable timeouts to prevent hanging
//
// # Security
//
// The healer handles sensitive information carefully:
//
//   - API keys are never logged or exposed
//   - Error messages are sanitized before sending to AI services
//   - Generated code is validated for basic syntax before applying
//   - GitHub operations use minimal required permissions
package healer

import "net/http"

// PublicAPI documents the main public interface of the healer package.
// This interface is stable and follows semantic versioning.
type PublicAPI interface {
	// Core lifecycle management
	Initialize(config Config) (*Healer, error)
	Start() error
	Stop() error

	// Panic handling installation
	InstallPanicHandler()
	RestorePanicHandler()

	// Status and monitoring
	GetStatus() map[string]any
	GetQueueStats() map[string]any
	ResetCircuitBreaker()
}

// GlobalFunctions documents the global functions available for panic handling.
// These functions work with the globally installed healer instance.
type GlobalFunctions interface {
	// Panic capture functions
	HandlePanic()                                                                                              // Captures panic and re-panics
	RecoverAndHandle()                                                                                         // Captures panic and recovers gracefully
	WrapFunction(fn func()) func()                                                                             // Wraps function with panic capture
	WrapFunctionWithRecovery(fn func()) func()                                                                 // Wraps function with graceful recovery
	WrapFunctionWithArgs(fn func(...any)) func(...any)                                                         // Wraps variadic function
	WrapFunctionWithArgsAndRecovery(fn func(...any)) func(...any)                                              // Wraps variadic function with recovery
	WrapHTTPHandler(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) // Wraps HTTP handler
	SafeGoroutine(fn func())                                                                                   // Starts goroutine with panic capture

	// Convenience functions
	InstallGlobalPanicHandler(config Config) (*Healer, error) // Initialize and install in one call
	MustInstallGlobalPanicHandler(config Config) *Healer      // Like InstallGlobalPanicHandler but panics on error
	GetGlobalHealer() *Healer                                 // Returns current global healer
	IsGlobalHealerInstalled() bool                            // Checks if global healer is installed
}

// ConfigurationAPI documents the configuration management interface.
type ConfigurationAPI interface {
	// Configuration loading and validation
	LoadConfig(configPath string) (*Config, error)
	LoadFromEnv() error
	LoadFromFile(filePath string) error
	ApplyDefaults()
	Validate() error
	ValidateComplete() error
	ValidateAPIKeys() error

	// Configuration utilities
	DefaultConfig() Config
	GetFallbackConfig() *Config
	LogConfigStatus() []string
}

// LoggingAPI documents the logging interface.
type LoggingAPI interface {
	// Logging methods
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	SetLevel(level LogLevel)

	// Logger creation
	NewDefaultLogger(levelStr string) Logger
}

// TypesAPI documents the main types and data structures.
type TypesAPI interface {
	// Core types
	Config           // Configuration structure
	Healer           // Main healer instance
	PanicEvent       // Captured panic information
	ProcessingResult // Result of processing a panic
	Logger           // Logging interface
	LogLevel         // Logging level enumeration

	// AI integration types
	AIClient    // AI client interface
	FixRequest  // Request for AI fix generation
	FixResponse // AI-generated fix response

	// Git integration types
	GitClient  // Git client interface
	PRRequest  // Pull request creation request
	FileChange // File modification structure
}

// UtilityAPI documents utility functions and helpers.
type UtilityAPI interface {
	// Git utilities
	GenerateBranchName(panicEvent PanicEvent) string
	GeneratePRTitle(panicEvent PanicEvent) string
	GeneratePRDescription(panicEvent PanicEvent, fixResponse *FixResponse) string

	// Panic event utilities
	NewPanicEvent(panicValue any) *PanicEvent
}

// ExampleUsage provides comprehensive usage examples for the healer package.
type ExampleUsage struct{}

// BasicSetup demonstrates the most common setup pattern.
func (ExampleUsage) BasicSetup() {
	// Example: Basic setup with environment variables
	config := Config{
		OpenAIAPIKey: "sk-your-openai-key-here",
		GitHubToken:  "ghp_your-github-token-here",
		RepoOwner:    "your-username",
		RepoName:     "your-repository",
		Enabled:      true,
	}

	healer, err := InstallGlobalPanicHandler(config)
	if err != nil {
		// Handle initialization error
		return
	}
	defer healer.Stop()

	// Your application code here
	// Panics will now be automatically captured and processed
}

// ManualPanicCapture demonstrates manual panic handling.
func (ExampleUsage) ManualPanicCapture() {
	// Example: Manual panic capture in specific functions
	func() {
		defer HandlePanic() // Will capture panic and re-panic
		// Code that might panic
	}()

	func() {
		defer RecoverAndHandle() // Will capture panic and recover gracefully
		// Code that might panic
	}()
}

// FunctionWrapping demonstrates function wrapping patterns.
func (ExampleUsage) FunctionWrapping() {
	// Example: Wrapping existing functions
	riskyFunction := func() {
		// Code that might panic
	}

	safeFunction := WrapFunction(riskyFunction)
	safeFunction() // Will capture any panics

	// Example: HTTP handler wrapping
	// http.HandleFunc("/api", WrapHTTPHandler(func(w http.ResponseWriter, r *http.Request) {
	//     // Handler code that might panic
	// }))

	// Example: Safe goroutines
	SafeGoroutine(func() {
		// Goroutine code that might panic
		// Will be captured and handled gracefully
	})
}

// ConfigurationExamples demonstrates various configuration patterns.
func (ExampleUsage) ConfigurationExamples() {
	// Example: Loading from environment variables
	config := DefaultConfig()
	err := config.LoadFromEnv()
	if err != nil {
		// Handle configuration error
		return
	}

	// Example: Loading from JSON file
	config2, err := LoadConfig("healer-config.json")
	if err != nil {
		// Handle configuration error
		return
	}

	// Example: Fallback configuration (disables features if keys missing)
	config3 := GetFallbackConfig()

	// Example: Manual configuration with validation
	config4 := Config{
		OpenAIAPIKey:  "sk-...",
		GitHubToken:   "ghp_...",
		RepoOwner:     "username",
		RepoName:      "repo",
		Enabled:       true,
		MaxQueueSize:  200,
		WorkerCount:   4,
		RetryAttempts: 5,
		LogLevel:      "debug",
	}

	if err := config4.ValidateComplete(); err != nil {
		// Handle validation error
		return
	}

	_, _, _ = config2, config3, config4 // Use configurations
}

// MonitoringExamples demonstrates monitoring and status checking.
func (ExampleUsage) MonitoringExamples() {
	healer := GetGlobalHealer()
	if healer == nil {
		return
	}

	// Example: Get overall status
	status := healer.GetStatus()
	// status contains: enabled, running, config, queue_stats

	// Example: Get queue statistics
	queueStats := healer.GetQueueStats()
	// queueStats contains: queue_capacity, queue_length, queue_available,
	// dropped_events, worker_count, workers_running, circuit_breaker_state

	// Example: Reset circuit breaker if needed
	healer.ResetCircuitBreaker()

	_, _ = status, queueStats // Use status information
}
