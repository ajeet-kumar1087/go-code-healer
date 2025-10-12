package main

import (
	"context"
	"log"
	"time"

	healer "github.com/ajeet-kumar1087/go-code-healer"
	"github.com/ajeet-kumar1087/go-code-healer/ai"
	"github.com/ajeet-kumar1087/go-code-healer/internal"
)

// SimpleLogger implements a basic logger for the demo
type SimpleLogger struct{}

func (l *SimpleLogger) Debug(msg string, args ...any) { log.Printf("[DEBUG] "+msg, args...) }
func (l *SimpleLogger) Info(msg string, args ...any)  { log.Printf("[INFO] "+msg, args...) }
func (l *SimpleLogger) Warn(msg string, args ...any)  { log.Printf("[WARN] "+msg, args...) }
func (l *SimpleLogger) Error(msg string, args ...any) { log.Printf("[ERROR] "+msg, args...) }

// MockGitClient implements GitClientInterface for demonstration
type MockGitClient struct{}

func (m *MockGitClient) CreatePullRequest(ctx context.Context, request ai.PRRequest) error {
	log.Printf("Creating PR: %s", request.Title)
	log.Printf("Branch: %s", request.BranchName)
	log.Printf("Files changed: %d", len(request.Changes))
	log.Printf("Description preview: %.100s...", request.Description)
	return nil
}

func main() {
	// Configuration with multiple AI providers and MCP
	config := healer.NewConfig()
	config.AIProvider = "claude" // Primary provider
	config.OpenAIAPIKey = "sk-your-openai-key"
	config.OpenAIModel = "gpt-4"
	config.ClaudeAPIKey = "sk-ant-your-claude-key"
	config.ClaudeModel = "claude-3-sonnet-20240229"
	config.CodexAPIKey = "sk-your-codex-key"
	config.CodexModel = "code-davinci-002"

	// MCP Configuration
	config.MCPEnabled = true
	config.MCPTimeout = 10

	// Create MCP server configurations
	fsAnalyzer := healer.NewMCPServerConfig("filesystem-analyzer", "http://localhost:8001/mcp")
	fsAnalyzer.Tools = []string{"analyze_structure", "find_dependencies"}
	fsAnalyzer.Timeout = 5

	codeAnalyzer := healer.NewMCPServerConfig("code-analyzer", "http://localhost:8002/mcp")
	codeAnalyzer.AuthType = "bearer"
	codeAnalyzer.AuthToken = "your-mcp-token"
	codeAnalyzer.Tools = []string{"parse_ast", "analyze_symbols"}
	codeAnalyzer.Timeout = 8

	config.MCPServers = []healer.MCPServerConfig{fsAnalyzer, codeAnalyzer}
	config.RetryAttempts = 3
	config.LogLevel = "info"

	// Create logger (using a simple log for demo)
	logger := &SimpleLogger{}

	// Create provider manager
	providerManager, err := ai.NewProviderManager(config, logger)
	if err != nil {
		log.Fatalf("Failed to create provider manager: %v", err)
	}

	// Validate providers
	if err := providerManager.ValidateProviders(); err != nil {
		log.Printf("Provider validation warnings: %v", err)
	}

	// Create mock Git client
	gitClient := &MockGitClient{}

	log.Println("=== AI Session-Based Error Handling Demo ===")
	log.Printf("Provider Status: %+v", providerManager.GetProviderStatus())

	// Simulate different types of runtime errors
	simulateNilPointerError(providerManager, gitClient, logger)
	time.Sleep(2 * time.Second)

	simulateIndexOutOfBoundsError(providerManager, gitClient, logger)
	time.Sleep(2 * time.Second)

	simulateConcurrentMapError(providerManager, gitClient, logger)
}

func simulateNilPointerError(pm *ai.ProviderManager, gitClient ai.GitClientInterface, logger internal.LoggerInterface) {
	log.Println("\n--- Simulating Nil Pointer Error ---")

	// Create session
	session := pm.CreateSession(gitClient)

	// Prepare error information
	errorInfo := &ai.ErrorInfo{
		Error: "runtime error: invalid memory address or nil pointer dereference",
		StackTrace: `panic: runtime error: invalid memory address or nil pointer dereference

goroutine 1 [running]:
main.processUser(0x0)
	/app/user.go:45 +0x1a
main.main()
	/app/main.go:12 +0x29`,
		SourceFile: "user.go",
		LineNumber: 45,
		Function:   "processUser",
		Timestamp:  time.Now(),
		Severity:   "critical",
	}

	// Prepare code context
	codeContext := &ai.CodeContext{
		SourceCode: `func processUser(user *User) {
	// This line causes nil pointer dereference
	fmt.Printf("Processing user: %s", user.Name)
	
	// Additional processing
	user.LastAccessed = time.Now()
}`,
		RelatedFiles: []string{"main.go", "types.go"},
		ImportedPkgs: []string{"fmt", "time"},
		FunctionSig:  "processUser(user *User)",
		StructDefs:   []string{"type User struct { Name string; LastAccessed time.Time }"},
	}

	// Initiate comprehensive session
	ctx := context.Background()
	result, err := session.InitiateSession(ctx, errorInfo, codeContext)
	if err != nil {
		log.Printf("Session failed: %v", err)
		return
	}

	log.Printf("Session completed successfully!")
	log.Printf("Session ID: %s", result.SessionID)
	log.Printf("Duration: %v", result.Duration)
	log.Printf("AI Provider: %s", result.FixResponse.Provider)
	log.Printf("Used MCP: %v", result.FixResponse.UsedMCP)
	log.Printf("Confidence: %.2f", result.FixResponse.Confidence)
	log.Printf("PR Created: %s", result.PRResult.BranchName)
}
func simulateIndexOutOfBoundsError(pm *ai.ProviderManager, gitClient ai.GitClientInterface, logger internal.LoggerInterface) {
	log.Println("\n--- Simulating Index Out of Bounds Error ---")

	session := pm.CreateSession(gitClient)

	errorInfo := &ai.ErrorInfo{
		Error: "runtime error: index out of range [5] with length 3",
		StackTrace: `panic: runtime error: index out of range [5] with length 3

goroutine 1 [running]:
main.processItems(0xc000010200, 0x3, 0x3)
	/app/processor.go:23 +0x85
main.main()
	/app/main.go:15 +0x45`,
		SourceFile: "processor.go",
		LineNumber: 23,
		Function:   "processItems",
		Timestamp:  time.Now(),
		Severity:   "high",
	}

	codeContext := &ai.CodeContext{
		SourceCode: `func processItems(items []string) {
	for i := 0; i <= len(items); i++ {
		// This causes index out of bounds
		fmt.Printf("Item %d: %s\n", i, items[i])
	}
}`,
		RelatedFiles: []string{"main.go"},
		ImportedPkgs: []string{"fmt"},
		FunctionSig:  "processItems(items []string)",
	}

	ctx := context.Background()
	result, err := session.InitiateSession(ctx, errorInfo, codeContext)
	if err != nil {
		log.Printf("Session failed: %v", err)
		return
	}

	log.Printf("Session completed! Provider: %s, Confidence: %.2f",
		result.FixResponse.Provider, result.FixResponse.Confidence)
}

func simulateConcurrentMapError(pm *ai.ProviderManager, gitClient ai.GitClientInterface, logger internal.LoggerInterface) {
	log.Println("\n--- Simulating Concurrent Map Access Error ---")

	session := pm.CreateSession(gitClient)

	errorInfo := &ai.ErrorInfo{
		Error: "fatal error: concurrent map writes",
		StackTrace: `fatal error: concurrent map writes

goroutine 19 [running]:
runtime.throw(0x4c7b85, 0x15)
	/usr/local/go/src/runtime/panic.go:774 +0x72 fp=0xc000042f50
main.updateCache(0xc000086000, 0x4c6a83, 0x4, 0x4c6a88, 0x5)
	/app/cache.go:34 +0x5c`,
		SourceFile: "cache.go",
		LineNumber: 34,
		Function:   "updateCache",
		Timestamp:  time.Now(),
		Severity:   "critical",
	}

	codeContext := &ai.CodeContext{
		SourceCode: `var cache = make(map[string]string)

func updateCache(key, value string) {
	// Concurrent access without synchronization
	cache[key] = value
}

func getFromCache(key string) string {
	return cache[key]
}`,
		RelatedFiles: []string{"main.go", "server.go"},
		ImportedPkgs: []string{"sync"},
		FunctionSig:  "updateCache(key, value string)",
	}

	ctx := context.Background()
	result, err := session.InitiateSession(ctx, errorInfo, codeContext)
	if err != nil {
		log.Printf("Session failed: %v", err)
		return
	}

	log.Printf("Session completed! Provider: %s, Used MCP: %v",
		result.FixResponse.Provider, result.FixResponse.UsedMCP)
	log.Printf("Fix explanation preview: %.100s...", result.FixResponse.Explanation)
}
