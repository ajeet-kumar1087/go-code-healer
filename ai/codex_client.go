package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ajeet-kumar1087/go-code-healer/internal"
)

// CodexClient implements the Client interface for GitHub Codex API
type CodexClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	logger     internal.LoggerInterface
	baseURL    string
}

// NewCodexClient creates a new Codex client
func NewCodexClient(apiKey, model string, logger internal.LoggerInterface) *CodexClient {
	if model == "" {
		model = "code-davinci-002"
	}

	return &CodexClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.openai.com/v1/completions",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}
}

// GenerateFix implements the Client interface for Codex
func (c *CodexClient) GenerateFix(ctx context.Context, request FixRequest) (*FixResponse, error) {
	// Add timeout to context if not already present
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
	}

	// Generate Codex-optimized prompt
	prompt := c.generateCodexPrompt(request)

	// Create Codex API request
	codexReq := codexRequest{
		Model:       c.model,
		Prompt:      prompt,
		MaxTokens:   1500,
		Temperature: 0.1, // Low temperature for more deterministic code generation
		Stop:        []string{"```", "---END---"},
	}

	// Make API call
	response, err := c.makeCodexAPICall(ctx, codexReq)
	if err != nil {
		return nil, fmt.Errorf("Codex API call failed: %w", err)
	}

	// Parse response
	fixResponse, err := c.parseCodexResponse(response, request)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Codex response: %w", err)
	}

	// Set provider info
	fixResponse.Provider = "codex"
	fixResponse.UsedMCP = request.MCPContext != nil

	if c.logger != nil {
		c.logger.Debug("Codex generated fix with confidence %.2f", fixResponse.Confidence)
	}

	return fixResponse, nil
}

// GetProviderName returns the provider name
func (c *CodexClient) GetProviderName() string {
	return "codex"
}

// ValidateConfiguration validates the Codex client configuration
func (c *CodexClient) ValidateConfiguration() error {
	if c.apiKey == "" {
		return fmt.Errorf("Codex API key is required")
	}
	if c.model == "" {
		return fmt.Errorf("Codex model is required")
	}
	return nil
}

// generateCodexPrompt creates a Codex-optimized prompt focused on code completion
func (c *CodexClient) generateCodexPrompt(request FixRequest) string {
	var prompt strings.Builder

	prompt.WriteString("// Go code fix for runtime error\n")
	prompt.WriteString(fmt.Sprintf("// Error: %s\n", request.Error))

	if request.MCPContext != nil && len(request.MCPContext.Suggestions) > 0 {
		prompt.WriteString("// MCP Suggestions:\n")
		for _, suggestion := range request.MCPContext.Suggestions {
			prompt.WriteString(fmt.Sprintf("// - %s\n", suggestion))
		}
	}

	prompt.WriteString("\n// Original problematic code:\n")
	prompt.WriteString("/*\n")
	prompt.WriteString(request.SourceCode)
	prompt.WriteString("\n*/\n\n")

	prompt.WriteString("// Fixed code:\n")

	// Add context about the fix needed
	if strings.Contains(request.Error, "nil pointer") {
		prompt.WriteString("// Fix: Add nil check before dereferencing pointer\n")
	} else if strings.Contains(request.Error, "index out of range") {
		prompt.WriteString("// Fix: Add bounds checking before array/slice access\n")
	} else if strings.Contains(request.Error, "concurrent map") {
		prompt.WriteString("// Fix: Add proper synchronization for concurrent map access\n")
	}

	return prompt.String()
}

// makeCodexAPICall makes an HTTP request to Codex API
func (c *CodexClient) makeCodexAPICall(ctx context.Context, request codexRequest) (*codexResponse, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Codex API returned status %d", resp.StatusCode)
	}

	var codexResp codexResponse
	if err := json.NewDecoder(resp.Body).Decode(&codexResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &codexResp, nil
}

// parseCodexResponse parses Codex API response into FixResponse
func (c *CodexClient) parseCodexResponse(response *codexResponse, request FixRequest) (*FixResponse, error) {
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("empty response from Codex")
	}

	// Extract the generated code
	generatedText := strings.TrimSpace(response.Choices[0].Text)

	// Clean up the generated code
	proposedFix := c.cleanupGeneratedCode(generatedText)

	// Generate explanation based on the error type and fix
	explanation := c.generateExplanation(request.Error, proposedFix)

	// Calculate confidence based on code quality and completion
	confidence := c.calculateConfidence(response.Choices[0], proposedFix, request)

	return &FixResponse{
		ProposedFix: proposedFix,
		Explanation: explanation,
		Confidence:  confidence,
		IsValid:     proposedFix != "" && !strings.Contains(proposedFix, "TODO"),
	}, nil
}

// cleanupGeneratedCode removes comments and formats the generated code
func (c *CodexClient) cleanupGeneratedCode(text string) string {
	lines := strings.Split(text, "\n")
	var codeLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and comments at the beginning
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		codeLines = append(codeLines, line)
	}

	return strings.Join(codeLines, "\n")
}

// generateExplanation creates an explanation based on the error and fix
func (c *CodexClient) generateExplanation(errorMsg, fix string) string {
	var explanation strings.Builder

	explanation.WriteString("Codex Analysis:\n\n")

	if strings.Contains(errorMsg, "nil pointer") {
		explanation.WriteString("The error was caused by attempting to dereference a nil pointer. ")
		explanation.WriteString("The fix adds a nil check before accessing the pointer value.")
	} else if strings.Contains(errorMsg, "index out of range") {
		explanation.WriteString("The error was caused by accessing an array or slice with an invalid index. ")
		explanation.WriteString("The fix adds bounds checking to ensure the index is valid.")
	} else if strings.Contains(errorMsg, "concurrent map") {
		explanation.WriteString("The error was caused by concurrent access to a map without proper synchronization. ")
		explanation.WriteString("The fix adds mutex protection for thread-safe map operations.")
	} else {
		explanation.WriteString("The error has been analyzed and a fix has been generated based on Go best practices.")
	}

	explanation.WriteString("\n\nThe proposed fix follows Go idioms and includes proper error handling where appropriate.")

	return explanation.String()
}

// calculateConfidence determines confidence score based on various factors
func (c *CodexClient) calculateConfidence(choice codexChoice, fix string, request FixRequest) float64 {
	confidence := 0.7 // Base confidence for Codex

	// Increase confidence if completion finished naturally
	if choice.FinishReason == "stop" {
		confidence += 0.1
	}

	// Increase confidence if fix contains proper Go patterns
	if strings.Contains(fix, "if") && strings.Contains(fix, "!=") && strings.Contains(fix, "nil") {
		confidence += 0.1 // Good nil checking pattern
	}

	if strings.Contains(fix, "len(") && strings.Contains(fix, "<") {
		confidence += 0.1 // Good bounds checking pattern
	}

	// Decrease confidence if fix seems incomplete
	if len(fix) < 10 {
		confidence -= 0.2
	}

	// Increase confidence if MCP context was used
	if request.MCPContext != nil && request.MCPContext.Confidence > 0.5 {
		confidence += 0.1
	}

	// Ensure confidence is within valid range
	if confidence < 0 {
		confidence = 0
	} else if confidence > 1 {
		confidence = 1
	}

	return confidence
}
