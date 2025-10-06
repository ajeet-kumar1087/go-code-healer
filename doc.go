// Package healer provides comprehensive documentation and usage examples.
//
// # Integration Guide
//
// This guide shows how to integrate the Go Code Healer into your existing applications.
//
// ## Installation
//
// Add the healer to your Go module:
//
//	go get github.com/your-org/go-code-healer
//
// ## Basic Integration
//
// The simplest way to add the healer to your application:
//
//	package main
//
//	import (
//	    "log"
//	    "github.com/your-org/go-code-healer"
//	)
//
//	func main() {
//	    // Configure the healer
//	    config := healer.Config{
//	        OpenAIAPIKey: "your-openai-key",
//	        GitHubToken:  "your-github-token",
//	        RepoOwner:    "your-username",
//	        RepoName:     "your-repo",
//	        Enabled:      true,
//	    }
//
//	    // Install global panic handler
//	    h, err := healer.InstallGlobalPanicHandler(config)
//	    if err != nil {
//	        log.Fatal("Failed to install healer:", err)
//	    }
//	    defer h.Stop()
//
//	    // Your application code here
//	    runApplication()
//	}
//
// ## Web Application Integration
//
// For web applications, wrap your HTTP handlers:
//
//	package main
//
//	import (
//	    "net/http"
//	    "github.com/your-org/go-code-healer"
//	)
//
//	func main() {
//	    // Install healer (configuration omitted for brevity)
//	    healer.MustInstallGlobalPanicHandler(config)
//
//	    // Wrap handlers to capture panics
//	    http.HandleFunc("/api/users", healer.WrapHTTPHandler(handleUsers))
//	    http.HandleFunc("/api/orders", healer.WrapHTTPHandler(handleOrders))
//
//	    log.Fatal(http.ListenAndServe(":8080", nil))
//	}
//
//	func handleUsers(w http.ResponseWriter, r *http.Request) {
//	    // Handler code that might panic
//	    // Panics will be captured and handled gracefully
//	}
//
// ## Microservice Integration
//
// For microservices with background workers:
//
//	package main
//
//	import (
//	    "context"
//	    "time"
//	    "github.com/your-org/go-code-healer"
//	)
//
//	func main() {
//	    // Install healer
//	    h := healer.MustInstallGlobalPanicHandler(config)
//	    defer h.Stop()
//
//	    // Start background workers with panic protection
//	    for i := 0; i < 5; i++ {
//	        healer.SafeGoroutine(func() {
//	            worker(context.Background())
//	        })
//	    }
//
//	    // Keep main running
//	    select {}
//	}
//
//	func worker(ctx context.Context) {
//	    for {
//	        select {
//	        case <-ctx.Done():
//	            return
//	        default:
//	            // Worker logic that might panic
//	            processJob()
//	            time.Sleep(time.Second)
//	        }
//	    }
//	}
//
// ## Library Integration
//
// For libraries that want to provide panic protection to their users:
//
//	package mylib
//
//	import "github.com/your-org/go-code-healer"
//
//	// PublicFunction wraps the internal implementation with panic protection
//	func PublicFunction(data string) (result string, err error) {
//	    // Use defer with recovery to handle panics gracefully
//	    defer func() {
//	        if r := recover(); r != nil {
//	            // Capture the panic if healer is installed
//	            if healer.IsGlobalHealerInstalled() {
//	                healer.RecoverAndHandle()
//	            }
//	            // Convert panic to error
//	            err = fmt.Errorf("internal error: %v", r)
//	        }
//	    }()
//
//	    return internalFunction(data), nil
//	}
//
// ## Configuration Management
//
// ### Environment Variables
//
// The healer supports configuration through environment variables:
//
//	export HEALER_OPENAI_API_KEY="sk-your-key-here"
//	export HEALER_GITHUB_TOKEN="ghp_your-token-here"
//	export HEALER_REPO_OWNER="your-username"
//	export HEALER_REPO_NAME="your-repo"
//	export HEALER_ENABLED="true"
//	export HEALER_LOG_LEVEL="info"
//
// Then load the configuration:
//
//	config := healer.DefaultConfig()
//	if err := config.LoadFromEnv(); err != nil {
//	    log.Fatal("Failed to load config:", err)
//	}
//
// ### JSON Configuration File
//
// Create a `healer-config.json` file:
//
//	{
//	    "openai_api_key": "sk-your-key-here",
//	    "github_token": "ghp_your-token-here",
//	    "repo_owner": "your-username",
//	    "repo_name": "your-repo",
//	    "enabled": true,
//	    "max_queue_size": 100,
//	    "worker_count": 2,
//	    "retry_attempts": 3,
//	    "log_level": "info"
//	}
//
// Load the configuration:
//
//	config, err := healer.LoadConfig("healer-config.json")
//	if err != nil {
//	    log.Fatal("Failed to load config:", err)
//	}
//
// ## Advanced Usage Patterns
//
// ### Selective Panic Capture
//
// For fine-grained control over which functions have panic capture:
//
//	func criticalFunction() {
//	    defer healer.HandlePanic() // Capture and re-panic (preserves existing behavior)
//	    // Critical code that should panic if something goes wrong
//	}
//
//	func gracefulFunction() {
//	    defer healer.RecoverAndHandle() // Capture and recover gracefully
//	    // Code that should continue running even if this fails
//	}
//
// ### Custom Error Handling
//
// Combine healer with your existing error handling:
//
//	func processData(data []byte) (err error) {
//	    defer func() {
//	        if r := recover(); r != nil {
//	            // Let healer capture the panic
//	            if healer.IsGlobalHealerInstalled() {
//	                healer.RecoverAndHandle()
//	            }
//	            // Convert to error for your API
//	            err = fmt.Errorf("processing failed: %v", r)
//	        }
//	    }()
//
//	    // Processing logic that might panic
//	    return processInternal(data)
//	}
//
// ### Monitoring and Alerting
//
// Monitor healer status and integrate with your monitoring system:
//
//	func healthCheck() map[string]interface{} {
//	    health := make(map[string]interface{})
//
//	    if healer.IsGlobalHealerInstalled() {
//	        h := healer.GetGlobalHealer()
//	        status := h.GetStatus()
//	        queueStats := h.GetQueueStats()
//
//	        health["healer_enabled"] = status["enabled"]
//	        health["healer_running"] = status["running"]
//	        health["queue_length"] = queueStats["queue_length"]
//	        health["dropped_events"] = queueStats["dropped_events"]
//	    } else {
//	        health["healer_enabled"] = false
//	    }
//
//	    return health
//	}
//
// ## Best Practices
//
// ### 1. Graceful Degradation
//
// Always design your application to work even if the healer is disabled:
//
//	config := healer.GetFallbackConfig() // Disables healer if keys are missing
//	healer.MustInstallGlobalPanicHandler(config)
//
// ### 2. Appropriate Panic Capture
//
// Use `HandlePanic()` for critical paths where panics should still crash the application:
//
//	func main() {
//	    defer healer.HandlePanic() // Still panics, but captures for analysis
//	    // Main application logic
//	}
//
// Use `RecoverAndHandle()` for background workers and non-critical paths:
//
//	func backgroundWorker() {
//	    defer healer.RecoverAndHandle() // Recovers gracefully
//	    // Background processing
//	}
//
// ### 3. Testing Considerations
//
// Disable the healer in tests to avoid interference:
//
//	func TestMyFunction(t *testing.T) {
//	    // Disable healer for testing
//	    config := healer.Config{Enabled: false}
//	    h, _ := healer.Initialize(config)
//	    defer h.Stop()
//
//	    // Your test code
//	}
//
// ### 4. Security Considerations
//
// - Store API keys securely (environment variables, secret management systems)
// - Use GitHub tokens with minimal required permissions
// - Review AI-generated fixes before merging
// - Consider rate limiting for high-traffic applications
//
// ## Troubleshooting
//
// ### Common Issues
//
// 1. **Healer not capturing panics**: Ensure you're using `defer healer.HandlePanic()` or similar
// 2. **No pull requests created**: Check GitHub token permissions and repository settings
// 3. **AI fixes not generated**: Verify OpenAI API key and check rate limits
// 4. **High memory usage**: Reduce `MaxQueueSize` in configuration
// 5. **Slow performance**: Reduce `WorkerCount` or disable healer in high-traffic scenarios
//
// ### Debug Mode
//
// Enable debug logging to troubleshoot issues:
//
//	config := healer.Config{
//	    LogLevel: "debug",
//	    // ... other configuration
//	}
//
// ### Status Monitoring
//
// Check healer status programmatically:
//
//	if h := healer.GetGlobalHealer(); h != nil {
//	    status := h.GetStatus()
//	    if !status["running"].(bool) {
//	        log.Warn("Healer is not running")
//	    }
//
//	    queueStats := h.GetQueueStats()
//	    if queueStats["dropped_events"].(int) > 0 {
//	        log.Warn("Healer has dropped events, consider increasing queue size")
//	    }
//	}
//
// ## Performance Impact
//
// The healer is designed for minimal performance impact:
//
// - Panic capture: ~1-2 microseconds overhead per function call with defer
// - Background processing: Runs in separate goroutines, no blocking
// - Memory usage: Configurable queue size (default 100 events)
// - Network calls: Only made in background, with circuit breaker protection
//
// For high-performance applications, consider:
// - Using selective panic capture instead of global installation
// - Reducing worker count and queue size
// - Disabling in production if performance is critical
package healer
