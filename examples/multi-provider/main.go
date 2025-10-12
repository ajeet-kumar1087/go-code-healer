package main

import (
	"fmt"
	"log"
	"time"

	healer "github.com/ajeet-kumar1087/go-code-healer"
)

func main() {
	// Example configuration with multiple AI providers
	config := healer.Config{
		// Primary provider: Claude
		AIProvider:   "claude",
		ClaudeAPIKey: "sk-ant-your-claude-api-key-here",
		ClaudeModel:  "claude-3-sonnet-20240229",

		// Fallback providers (will be used if Claude fails)
		OpenAIAPIKey: "sk-your-openai-api-key-here",
		OpenAIModel:  "gpt-4",
		CodexAPIKey:  "sk-your-codex-api-key-here",
		CodexModel:   "code-davinci-002",

		// MCP configuration for enhanced context
		MCPEnabled: false, // Simplified for this example
		MCPTimeout: 10,

		// GitHub configuration
		GitHubToken: "ghp_your-github-token-here",
		RepoOwner:   "your-username",
		RepoName:    "your-repo",

		// Processing configuration
		Enabled:       true,
		MaxQueueSize:  100,
		WorkerCount:   2,
		RetryAttempts: 3,
		LogLevel:      "info",
	}

	// Initialize the healer
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

	fmt.Println("Multi-provider healer initialized successfully!")
	fmt.Println("Primary provider: Claude")
	fmt.Println("Fallback providers: OpenAI, Codex")
	fmt.Println("MCP context gathering: Disabled (for simplicity)")

	// Demonstrate provider fallback concept
	demonstrateProviderFallback()

	// Simulate some work that might panic
	fmt.Println("\nSimulating potential panic scenarios...")

	// This will be caught by the healer and processed in the background
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic caught and will be processed: %v\n", r)
		}
	}()

	// Example 1: Nil pointer dereference
	simulateNilPointerPanic()

	// Keep the program running to allow background processing
	fmt.Println("Waiting for background processing...")
	time.Sleep(5 * time.Second)
}

// demonstrateProviderFallback shows how the provider manager handles fallbacks
func demonstrateProviderFallback() {
	fmt.Println("\n=== Demonstrating Provider Fallback ===")

	// This would normally be handled by the provider manager internally
	fmt.Println("Fix request created - would be processed by provider fallback chain:")
	fmt.Printf("- Primary: Claude (with optimized structured prompts)\n")
	fmt.Printf("- Fallback 1: OpenAI (with balanced context)\n")
	fmt.Printf("- Fallback 2: Codex (with code-focused prompts)\n")
	fmt.Printf("- Each provider gets 3 retry attempts\n")
	fmt.Printf("- Best response is selected based on confidence score\n")
}

// simulateNilPointerPanic demonstrates a common Go panic scenario
func simulateNilPointerPanic() {
	fmt.Println("\nExample 1: Nil pointer dereference")

	var user *User
	// This will panic and be caught by the healer
	processUser(user)
}

type User struct {
	Name  string
	Email string
}

func processUser(user *User) {
	// This will panic if user is nil - healer will catch and generate a fix
	fmt.Printf("Processing user: %s\n", user.Name)
}
