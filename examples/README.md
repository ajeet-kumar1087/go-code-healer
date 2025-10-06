# Go Code Healer Examples

This directory contains comprehensive examples demonstrating how to integrate the Go Code Healer into different types of applications.

## Prerequisites

Before running the examples, you'll need:

1. **OpenAI API Key**: Get one from [OpenAI](https://platform.openai.com/api-keys)
2. **GitHub Token**: Create a personal access token with repository permissions
3. **Go 1.19+**: Make sure you have Go installed

## Environment Setup

Set the following environment variables:

```bash
export HEALER_OPENAI_API_KEY="sk-your-openai-key-here"
export HEALER_GITHUB_TOKEN="ghp_your-github-token-here"
export HEALER_REPO_OWNER="your-github-username"
export HEALER_REPO_NAME="your-repository-name"
```

Alternatively, you can run the examples without these variables - the healer will be disabled but the examples will still demonstrate panic handling patterns.

## Examples

### 1. Basic Example (`basic/`)

**File**: `examples/basic/main.go`

A simple command-line application that demonstrates:
- Basic healer setup and configuration
- Manual panic capture with `HandlePanic()` and `RecoverAndHandle()`
- Function wrapping with `WrapFunction()` and `WrapFunctionWithRecovery()`
- Safe goroutines with `SafeGoroutine()`
- Various types of panics (nil pointer, index out of bounds, etc.)

**Run it**:
```bash
cd examples/basic
go run main.go
```

**What it demonstrates**:
- Different panic capture patterns
- Background processing with panic protection
- Status monitoring and queue statistics
- Graceful degradation when healer is not configured

### 2. Web Application Example (`webapp/`)

**File**: `examples/webapp/main.go`

A complete web application that shows:
- HTTP handler wrapping with `WrapHTTPHandler()`
- Middleware integration for panic recovery
- Background workers with panic protection
- RESTful API endpoints that may panic
- Health checks and status monitoring

**Run it**:
```bash
cd examples/webapp
go run main.go
```

**Test endpoints**:
```bash
# Health check
curl http://localhost:8080/health

# Get users (may panic randomly)
curl http://localhost:8080/users

# Get specific user (different IDs cause different panics)
curl http://localhost:8080/users/1      # Normal response
curl http://localhost:8080/users/999    # Database panic
curl http://localhost:8080/users/666    # Nil pointer panic
curl http://localhost:8080/users/404    # Index out of bounds panic

# Create user
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com"}'

# Create user that causes panic
curl -X POST http://localhost:8080/users \
  -H "Content-Type: application/json" \
  -d '{"name":"","email":"panic@example.com"}'

# Intentional panic with different types
curl http://localhost:8080/panic?type=nil
curl http://localhost:8080/panic?type=slice
curl http://localhost:8080/panic?type=map

# Check healer status
curl http://localhost:8080/healer/status
```

**What it demonstrates**:
- Web application integration patterns
- HTTP middleware for panic recovery
- Background job processing
- Error response handling
- Production-ready panic management

### 3. Microservice Example (`microservice/`)

**File**: `examples/microservice/main.go`

A microservice architecture example featuring:
- Message queue processing with worker pools
- Background workers with panic protection
- Service-to-service communication patterns
- Health checks and metrics collection
- Retry logic and dead letter queues

**Run it**:
```bash
cd examples/microservice
go run main.go
```

**What it demonstrates**:
- High-throughput message processing
- Worker pool patterns with panic protection
- Microservice resilience patterns
- Background job processing
- Service monitoring and health checks
- Message retry and failure handling

## Common Patterns Demonstrated

### 1. Basic Panic Capture

```go
func riskyFunction() {
    defer healer.HandlePanic() // Captures panic and re-panics
    // Code that might panic
}

func gracefulFunction() {
    defer healer.RecoverAndHandle() // Captures panic and recovers
    // Code that might panic
}
```

### 2. Function Wrapping

```go
safeFunc := healer.WrapFunction(riskyFunction)
safeFunc() // Will capture any panics

// For HTTP handlers
http.HandleFunc("/api", healer.WrapHTTPHandler(myHandler))
```

### 3. Safe Goroutines

```go
healer.SafeGoroutine(func() {
    // This goroutine will capture and handle panics gracefully
    doBackgroundWork()
})
```

### 4. Configuration Patterns

```go
// Environment-based configuration
config := healer.DefaultConfig()
config.LoadFromEnv()

// File-based configuration
config, err := healer.LoadConfig("healer-config.json")

// Fallback configuration (disables if keys missing)
config := healer.GetFallbackConfig()
```

### 5. Status Monitoring

```go
if h := healer.GetGlobalHealer(); h != nil {
    status := h.GetStatus()
    queueStats := h.GetQueueStats()
    
    // Monitor queue health
    if queueStats["queue_length"].(int) > 80 {
        log.Warn("Healer queue is getting full")
    }
}
```

## Expected Behavior

When you run these examples with proper configuration:

1. **Panic Detection**: The healer will capture panics and log them
2. **AI Processing**: Panics will be sent to OpenAI for fix generation (in background)
3. **Pull Request Creation**: If a fix is generated, a PR will be created in your repository
4. **Non-blocking Operation**: Your application continues running normally

## Testing Without Full Configuration

You can run all examples without OpenAI/GitHub credentials:

```bash
# Run with healer disabled
HEALER_ENABLED=false go run main.go

# Or just run without environment variables
go run main.go
```

The examples will still demonstrate panic handling patterns, but won't generate fixes or create pull requests.

## Troubleshooting

### Common Issues

1. **"Failed to install healer"**: Check your API keys and network connectivity
2. **"No pull requests created"**: Verify GitHub token permissions and repository access
3. **"High memory usage"**: Reduce `MaxQueueSize` in configuration
4. **"Panics not captured"**: Ensure you're using `defer healer.HandlePanic()` or similar

### Debug Mode

Enable debug logging to see detailed information:

```bash
export HEALER_LOG_LEVEL=debug
go run main.go
```

### Checking Healer Status

All examples include status endpoints or logging to help you monitor the healer's operation.

## Next Steps

After running these examples:

1. **Review Generated Pull Requests**: Check your GitHub repository for automatically created PRs
2. **Integrate into Your Application**: Use these patterns in your own codebase
3. **Customize Configuration**: Adjust settings based on your application's needs
4. **Monitor Performance**: Use the status monitoring patterns in production

## Production Considerations

- **Security**: Store API keys securely using environment variables or secret management
- **Performance**: Monitor queue sizes and adjust worker counts based on load
- **Rate Limits**: Be aware of OpenAI and GitHub API rate limits
- **Review Process**: Always review AI-generated fixes before merging
- **Monitoring**: Implement proper monitoring and alerting for the healer's health