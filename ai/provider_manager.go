package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ajeet-kumar1087/go-code-healer/internal"
)

// ProviderManager manages multiple AI providers with fallback capabilities
type ProviderManager struct {
	providers  []Client
	mcpClient  *MCPClient
	logger     internal.LoggerInterface
	maxRetries int
	retryDelay time.Duration
}

// ProviderConfig holds configuration for AI providers
type ProviderConfig struct {
	Primary    string   `json:"primary"`   // Primary provider name
	Fallbacks  []string `json:"fallbacks"` // Fallback provider names in order
	MaxRetries int      `json:"max_retries"`
	RetryDelay int      `json:"retry_delay_seconds"`
}

// NewProviderManager creates a new provider manager
func NewProviderManager(config internal.Config, logger internal.LoggerInterface) (*ProviderManager, error) {
	var providers []Client

	// Create MCP client if enabled
	var mcpClient *MCPClient
	if config.MCPEnabled && len(config.MCPServers) > 0 {
		mcpTimeout := time.Duration(config.MCPTimeout) * time.Second
		mcpClient = NewMCPClient(config.MCPServers, mcpTimeout, logger)
	}

	// Create AI providers based on configuration
	switch config.AIProvider {
	case "openai":
		if config.OpenAIAPIKey != "" {
			openaiClient := NewOpenAIClient(config.OpenAIAPIKey, config.OpenAIModel, logger)
			providers = append(providers, openaiClient)
		}
		// Add fallback providers
		if config.ClaudeAPIKey != "" {
			claudeClient := NewClaudeClient(config.ClaudeAPIKey, config.ClaudeModel, logger)
			providers = append(providers, claudeClient)
		}
		if config.CodexAPIKey != "" {
			codexClient := NewCodexClient(config.CodexAPIKey, config.CodexModel, logger)
			providers = append(providers, codexClient)
		}

	case "claude":
		if config.ClaudeAPIKey != "" {
			claudeClient := NewClaudeClient(config.ClaudeAPIKey, config.ClaudeModel, logger)
			providers = append(providers, claudeClient)
		}
		// Add fallback providers
		if config.OpenAIAPIKey != "" {
			openaiClient := NewOpenAIClient(config.OpenAIAPIKey, config.OpenAIModel, logger)
			providers = append(providers, openaiClient)
		}
		if config.CodexAPIKey != "" {
			codexClient := NewCodexClient(config.CodexAPIKey, config.CodexModel, logger)
			providers = append(providers, codexClient)
		}

	case "codex":
		if config.CodexAPIKey != "" {
			codexClient := NewCodexClient(config.CodexAPIKey, config.CodexModel, logger)
			providers = append(providers, codexClient)
		}
		// Add fallback providers
		if config.OpenAIAPIKey != "" {
			openaiClient := NewOpenAIClient(config.OpenAIAPIKey, config.OpenAIModel, logger)
			providers = append(providers, openaiClient)
		}
		if config.ClaudeAPIKey != "" {
			claudeClient := NewClaudeClient(config.ClaudeAPIKey, config.ClaudeModel, logger)
			providers = append(providers, claudeClient)
		}

	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", config.AIProvider)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("no AI providers configured")
	}

	maxRetries := config.RetryAttempts
	if maxRetries == 0 {
		maxRetries = 3
	}

	return &ProviderManager{
		providers:  providers,
		mcpClient:  mcpClient,
		logger:     logger,
		maxRetries: maxRetries,
		retryDelay: 2 * time.Second,
	}, nil
}

// GenerateFixWithFallback attempts fix generation with primary provider, falls back to others
func (pm *ProviderManager) GenerateFixWithFallback(ctx context.Context, request FixRequest) (*FixResponse, error) {
	// Enhance request with MCP context if available
	if pm.mcpClient != nil {
		mcpContext, err := pm.gatherMCPContext(ctx, request)
		if err != nil {
			if pm.logger != nil {
				pm.logger.Warn("Failed to gather MCP context: %v", err)
			}
		} else {
			request.MCPContext = mcpContext
		}
	}

	var lastError error
	var bestResponse *FixResponse

	// Try each provider in order
	for i, provider := range pm.providers {
		if pm.logger != nil {
			pm.logger.Debug("Attempting fix generation with provider: %s", provider.GetProviderName())
		}

		// Optimize request for specific provider
		optimizedRequest := pm.optimizeRequestForProvider(request, provider.GetProviderName())

		// Try with retries for each provider
		for attempt := 0; attempt < pm.maxRetries; attempt++ {
			response, err := provider.GenerateFix(ctx, optimizedRequest)
			if err == nil && response != nil {
				// Check if this is a valid response
				if pm.isValidResponse(response) {
					if pm.logger != nil {
						pm.logger.Info("Successfully generated fix with provider %s (attempt %d, confidence: %.2f)",
							provider.GetProviderName(), attempt+1, response.Confidence)
					}
					return response, nil
				}

				// Keep track of best response even if not fully valid
				if bestResponse == nil || response.Confidence > bestResponse.Confidence {
					bestResponse = response
				}
			}

			lastError = err
			if pm.logger != nil {
				pm.logger.Warn("Provider %s attempt %d failed: %v",
					provider.GetProviderName(), attempt+1, err)
			}

			// Wait before retry (except for last attempt)
			if attempt < pm.maxRetries-1 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(pm.retryDelay):
					// Continue to next attempt
				}
			}
		}

		if pm.logger != nil {
			pm.logger.Warn("Provider %s failed after %d attempts, trying next provider",
				provider.GetProviderName(), pm.maxRetries)
		}

		// If this is not the last provider, continue to next
		if i < len(pm.providers)-1 {
			continue
		}
	}

	// If we have a best response but no fully valid one, return it with a warning
	if bestResponse != nil {
		if pm.logger != nil {
			pm.logger.Warn("No fully valid response found, returning best response with confidence %.2f",
				bestResponse.Confidence)
		}
		return bestResponse, nil
	}

	return nil, fmt.Errorf("all AI providers failed, last error: %w", lastError)
}

// optimizeRequestForProvider optimizes the request for a specific provider
func (pm *ProviderManager) optimizeRequestForProvider(request FixRequest, providerName string) FixRequest {
	optimized := request

	switch providerName {
	case "claude":
		// Claude prefers more structured, detailed context
		optimized = pm.optimizeForClaude(request)
	case "codex":
		// Codex prefers code-focused, minimal context
		optimized = pm.optimizeForCodex(request)
	case "openai":
		// OpenAI works well with the standard format
		optimized = pm.optimizeForOpenAI(request)
	}

	return optimized
}

// optimizeForClaude optimizes the request for Claude's preferences
func (pm *ProviderManager) optimizeForClaude(request FixRequest) FixRequest {
	optimized := request

	// Claude prefers detailed explanations and structured context
	if optimized.Context == "" && optimized.MCPContext != nil {
		// Add more context from MCP if available
		if len(optimized.MCPContext.Suggestions) > 0 {
			optimized.Context = "Additional insights: " + strings.Join(optimized.MCPContext.Suggestions, "; ")
		}
	}

	return optimized
}

// optimizeForCodex optimizes the request for Codex's preferences
func (pm *ProviderManager) optimizeForCodex(request FixRequest) FixRequest {
	optimized := request

	// Codex prefers minimal, code-focused context
	// Truncate verbose context to focus on the essential parts
	if len(optimized.Context) > 500 {
		optimized.Context = optimized.Context[:500] + "..."
	}

	// Prioritize stack trace and source code over other context
	if len(optimized.StackTrace) > 1000 {
		lines := strings.Split(optimized.StackTrace, "\n")
		if len(lines) > 20 {
			// Keep first 10 and last 10 lines
			optimized.StackTrace = strings.Join(lines[:10], "\n") + "\n... (truncated) ...\n" + strings.Join(lines[len(lines)-10:], "\n")
		}
	}

	return optimized
}

// optimizeForOpenAI optimizes the request for OpenAI's preferences
func (pm *ProviderManager) optimizeForOpenAI(request FixRequest) FixRequest {
	optimized := request

	// OpenAI works well with balanced context
	// Ensure we have good metadata for better analysis
	if optimized.Metadata == nil {
		optimized.Metadata = make(map[string]string)
	}

	// Add error type classification for better prompt engineering
	if strings.Contains(optimized.Error, "nil pointer") {
		optimized.Metadata["error_type"] = "nil_pointer"
	} else if strings.Contains(optimized.Error, "index out of range") {
		optimized.Metadata["error_type"] = "bounds_check"
	} else if strings.Contains(optimized.Error, "concurrent map") {
		optimized.Metadata["error_type"] = "concurrency"
	}

	return optimized
}

// isValidResponse checks if a response is valid and usable
func (pm *ProviderManager) isValidResponse(response *FixResponse) bool {
	if response == nil {
		return false
	}

	// Check basic validity
	if response.ProposedFix == "" {
		return false
	}

	// Check confidence threshold
	if response.Confidence < 0.3 {
		return false
	}

	// Check if marked as valid by the provider
	if !response.IsValid {
		return false
	}

	return true
}

// gatherMCPContext collects context using MCP tools
func (pm *ProviderManager) gatherMCPContext(ctx context.Context, request FixRequest) (*ContextResponse, error) {
	mcpRequest := ContextRequest{
		ErrorType:  request.Error,
		StackTrace: request.StackTrace,
		Metadata:   request.Metadata,
	}

	// Extract source file from metadata if available
	if request.Metadata != nil {
		if sourceFile, ok := request.Metadata["source_file"]; ok {
			mcpRequest.SourceFile = sourceFile
		}
		if function, ok := request.Metadata["function"]; ok {
			mcpRequest.Function = function
		}
	}

	return pm.mcpClient.GatherContext(ctx, mcpRequest)
}

// CreateSession creates a new AI session for comprehensive error handling
func (pm *ProviderManager) CreateSession(gitClient GitClientInterface) *SessionManager {
	// Use the first available provider for the session
	var primaryProvider Client
	if len(pm.providers) > 0 {
		primaryProvider = pm.providers[0]
	}

	return NewSessionManager(primaryProvider, pm.mcpClient, gitClient, pm.logger)
}

// ValidateProviders validates all configured providers
func (pm *ProviderManager) ValidateProviders() error {
	var errors []string

	for _, provider := range pm.providers {
		if err := provider.ValidateConfiguration(); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", provider.GetProviderName(), err))
		}
	}

	if pm.mcpClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := pm.mcpClient.ValidateServers(ctx); err != nil {
			errors = append(errors, fmt.Sprintf("MCP: %v", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("provider validation failed: %v", errors)
	}

	return nil
}

// GetProviderStatus returns status information for all providers
func (pm *ProviderManager) GetProviderStatus() map[string]interface{} {
	status := make(map[string]interface{})

	var providerNames []string
	for _, provider := range pm.providers {
		providerNames = append(providerNames, provider.GetProviderName())
	}

	status["providers"] = providerNames
	status["primary_provider"] = ""
	if len(pm.providers) > 0 {
		status["primary_provider"] = pm.providers[0].GetProviderName()
	}
	status["mcp_enabled"] = pm.mcpClient != nil
	status["max_retries"] = pm.maxRetries

	return status
}
