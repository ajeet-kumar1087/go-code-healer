package main

import (
	"log"
	"time"

	healer "github.com/ajeet-kumar1087/go-code-healer"
	"github.com/ajeet-kumar1087/go-code-healer/internal"
)

func main() {
	// Example configuration with MCP integration
	config := internal.Config{
		AIProvider:   "openai",
		OpenAIAPIKey: "sk-your-openai-api-key-here",
		OpenAIModel:  "gpt-4",

		// Enable MCP for enhanced context gathering
		MCPEnabled: true,
		MCPTimeout: 10,
		MCPServers: []internal.MCPServerConfig{
			{
				Name:     "filesystem-analyzer",
				Endpoint: "http://localhost:8001/mcp",
				AuthType: "none",
				Tools:    []string{"analyze_structure", "find_dependencies"},
				Timeout:  5,
			},
			{
				Name:      "code-analyzer",
				Endpoint:  "http://localhost:8002/mcp",
				AuthType:  "bearer",
				AuthToken: "your-mcp-token-here",
				Tools:     []string{"parse_ast", "analyze_symbols"},
				Timeout:   8,
			},
		},

		GitHubToken: "ghp_your-github-token-here",
		RepoOwner:   "your-username",
		RepoName:    "your-repository",

		Enabled:       true,
		MaxQueueSize:  100,
		WorkerCount:   2,
		RetryAttempts: 3,
		LogLevel:      "info",
	}

	// Initialize the healer with MCP support
	h, err := healer.Initialize(config)
	if err != nil {
		log.Fatalf("Failed to initialize healer: %v", err)
	}

	// Start background processing
	if err := h.Start(); err != nil {
		log.Fatalf("Failed to start healer: %v", err)
	}
	defer h.Stop()

	// Install the panic handler
	h.InstallPanicHandler()

	log.Println("Healer with MCP integration is now active!")
	log.Println("The healer will:")
	log.Println("1. Capture runtime panics")
	log.Println("2. Gather enhanced context using MCP tools")
	log.Println("3. Generate AI-powered fixes with better context")
	log.Println("4. Create pull requests with comprehensive analysis")

	// Simulate some work that might panic
	go func() {
		time.Sleep(2 * time.Second)
		simulateNilPointerPanic()
	}()

	go func() {
		time.Sleep(4 * time.Second)
		simulateIndexOutOfBounds()
	}()

	// Keep the application running
	select {}
}

// simulateNilPointerPanic demonstrates a nil pointer dereference
func simulateNilPointerPanic() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
			// The healer will capture this panic and process it with MCP context
		}
	}()

	var ptr *string
	// This will cause a nil pointer dereference panic
	// The MCP tools will provide additional context about:
	// - The function structure and call graph
	// - Related files and dependencies
	// - Environment information
	// - Code analysis suggesting nil checks
	log.Println("Length:", len(*ptr))
}

// simulateIndexOutOfBounds demonstrates an index out of bounds error
func simulateIndexOutOfBounds() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
			// The healer will capture this panic and process it with MCP context
		}
	}()

	slice := []int{1, 2, 3}
	// This will cause an index out of bounds panic
	// The MCP tools will provide additional context about:
	// - Slice usage patterns in the codebase
	// - Similar functions that handle bounds checking
	// - Suggested defensive programming patterns
	log.Println("Value:", slice[10])
}
