package ai

import (
	"testing"

	"github.com/ajeet-kumar1087/go-code-healer/internal"
)

func TestProviderCreation(t *testing.T) {
	logger := internal.NewDefaultLogger(internal.LogLevelInfo)

	// Test Claude client creation
	claudeClient := NewClaudeClient("test-key", "claude-3-sonnet-20240229", logger)
	if claudeClient == nil {
		t.Fatal("Failed to create Claude client")
	}
	if claudeClient.GetProviderName() != "claude" {
		t.Errorf("Expected provider name 'claude', got '%s'", claudeClient.GetProviderName())
	}

	// Test Codex client creation
	codexClient := NewCodexClient("test-key", "code-davinci-002", logger)
	if codexClient == nil {
		t.Fatal("Failed to create Codex client")
	}
	if codexClient.GetProviderName() != "codex" {
		t.Errorf("Expected provider name 'codex', got '%s'", codexClient.GetProviderName())
	}

	// Test OpenAI client creation
	openaiClient := NewOpenAIClient("test-key", "gpt-4", logger)
	if openaiClient == nil {
		t.Fatal("Failed to create OpenAI client")
	}
	if openaiClient.GetProviderName() != "openai" {
		t.Errorf("Expected provider name 'openai', got '%s'", openaiClient.GetProviderName())
	}
}

func TestProviderValidation(t *testing.T) {
	logger := internal.NewDefaultLogger(internal.LogLevelInfo)

	// Test validation with empty API key
	claudeClient := NewClaudeClient("", "claude-3-sonnet-20240229", logger)
	if err := claudeClient.ValidateConfiguration(); err == nil {
		t.Error("Expected validation error for empty API key")
	}

	codexClient := NewCodexClient("", "code-davinci-002", logger)
	if err := codexClient.ValidateConfiguration(); err == nil {
		t.Error("Expected validation error for empty API key")
	}

	openaiClient := NewOpenAIClient("", "gpt-4", logger)
	if err := openaiClient.ValidateConfiguration(); err == nil {
		t.Error("Expected validation error for empty API key")
	}

	// Test validation with valid configuration
	claudeClient = NewClaudeClient("sk-ant-test", "claude-3-sonnet-20240229", logger)
	if err := claudeClient.ValidateConfiguration(); err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}

	codexClient = NewCodexClient("sk-test", "code-davinci-002", logger)
	if err := codexClient.ValidateConfiguration(); err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}

	openaiClient = NewOpenAIClient("sk-test", "gpt-4", logger)
	if err := openaiClient.ValidateConfiguration(); err != nil {
		t.Errorf("Unexpected validation error: %v", err)
	}
}

func TestProviderManagerCreation(t *testing.T) {
	logger := internal.NewDefaultLogger(internal.LogLevelInfo)

	// Test with Claude as primary
	config := internal.Config{
		AIProvider:   "claude",
		ClaudeAPIKey: "sk-ant-test",
		ClaudeModel:  "claude-3-sonnet-20240229",
		OpenAIAPIKey: "sk-test",
		OpenAIModel:  "gpt-4",
		CodexAPIKey:  "sk-test",
		CodexModel:   "code-davinci-002",
	}

	pm, err := NewProviderManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create provider manager: %v", err)
	}

	status := pm.GetProviderStatus()
	if status["primary_provider"] != "claude" {
		t.Errorf("Expected primary provider 'claude', got '%v'", status["primary_provider"])
	}

	providers, ok := status["providers"].([]string)
	if !ok {
		t.Fatal("Expected providers to be []string")
	}

	if len(providers) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(providers))
	}

	// Test with OpenAI as primary
	config.AIProvider = "openai"
	pm, err = NewProviderManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create provider manager: %v", err)
	}

	status = pm.GetProviderStatus()
	if status["primary_provider"] != "openai" {
		t.Errorf("Expected primary provider 'openai', got '%v'", status["primary_provider"])
	}
}

func TestProviderOptimization(t *testing.T) {
	logger := internal.NewDefaultLogger(internal.LogLevelInfo)
	config := internal.Config{
		AIProvider:   "claude",
		ClaudeAPIKey: "sk-ant-test",
		OpenAIAPIKey: "sk-test",
		CodexAPIKey:  "sk-test",
	}

	pm, err := NewProviderManager(config, logger)
	if err != nil {
		t.Fatalf("Failed to create provider manager: %v", err)
	}

	request := FixRequest{
		Error:      "runtime error: nil pointer dereference",
		StackTrace: "very long stack trace...",
		SourceCode: "func test() { var p *int; fmt.Println(*p) }",
		Context:    "This is a test context",
	}

	// Test Claude optimization
	claudeReq := pm.optimizeRequestForProvider(request, "claude")
	if claudeReq.Error != request.Error {
		t.Error("Claude optimization should preserve error")
	}

	// Test Codex optimization
	codexReq := pm.optimizeRequestForProvider(request, "codex")
	if codexReq.Error != request.Error {
		t.Error("Codex optimization should preserve error")
	}

	// Test OpenAI optimization
	openaiReq := pm.optimizeRequestForProvider(request, "openai")
	if openaiReq.Error != request.Error {
		t.Error("OpenAI optimization should preserve error")
	}
	if openaiReq.Metadata == nil {
		t.Error("OpenAI optimization should add metadata")
	}
}
