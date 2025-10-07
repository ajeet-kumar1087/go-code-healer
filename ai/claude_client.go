package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ajeet-kumar1087/go-code-healer/internal"
)

// ClaudeClient implements the Client interface for Anthropic's Claude API
type ClaudeClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	logger     internal.LoggerInterface
	baseURL    string
}

// NewClaudeClient creates a new Claude client
func NewClaudeClient(apiKey, model string, logger internal.LoggerInterface) *ClaudeClient {
	if model == "" {
		model = "claude-3-sonnet-20240229"
	}

	return &ClaudeClient{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.anthropic.com/v1/messages",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}
}

// GenerateFix implements the Client interface for Claude
func (c *ClaudeClient) GenerateFix(ctx context.Context, request FixRequest) (*FixResponse, error) {
	// Add timeout to context if not already present
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
	}

	// Generate Claude-optimized prompt
	prompt := c.generateClaudePrompt(request)
	systemPrompt := c.getClaudeSystemPrompt()

	// Create Claude API request
	claudeReq := claudeRequest{
		Model:     c.model,
		MaxTokens: 2000,
		System:    systemPrompt,
		Messages: []claudeMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Make API call
	response, err := c.makeClaudeAPICall(ctx, claudeReq)
	if err != nil {
		return nil, fmt.Errorf("Claude API call failed: %w", err)
	}

	// Parse response
	fixResponse, err := c.parseClaudeResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Claude response: %w", err)
	}

	// Set provider info
	fixResponse.Provider = "claude"
	fixResponse.UsedMCP = request.MCPContext != nil

	if c.logger != nil {
		c.logger.Debug("Claude generated fix with confidence %.2f", fixResponse.Confidence)
	}

	return fixResponse, nil
}

// GetProviderName returns the provider name
func (c *ClaudeClient) GetProviderName() string {
	return "claude"
}

// ValidateConfiguration validates the Claude client configuration
func (c *ClaudeClient) ValidateConfiguration() error {
	if c.apiKey == "" {
		return fmt.Errorf("Claude API key is required")
	}
	if c.model == "" {
		return fmt.Errorf("Claude model is required")
	}
	return nil
}

// generateClaudePrompt creates a Claude-optimized prompt
func (c *ClaudeClient) generateClaudePrompt(request FixRequest) string {
	prompt := "I need help fixing a Go runtime error. Here are the details:\n\n"
	prompt += "## Error Information\n"
	prompt += fmt.Sprintf("**Error:** %s\n\n", request.Error)
	prompt += "**Stack Trace:**\n```\n"
	prompt += request.StackTrace
	prompt += "\n```\n\n"
	prompt += "**Source Code:**\n```go\n"
	prompt += request.SourceCode
	prompt += "\n```\n\n"

	if request.Context != "" {
		prompt += "**Additional Context:**\n"
		prompt += request.Context
		prompt += "\n\n"
	}

	// Add MCP context if available
	if request.MCPContext != nil {
		prompt += "## Enhanced Context (MCP Tools)\n"

		if request.MCPContext.FileStructure != "" {
			prompt += "**Project Structure:**\n```\n"
			prompt += request.MCPContext.FileStructure
			prompt += "\n```\n\n"
		}

		if len(request.MCPContext.Dependencies) > 0 {
			prompt += "**Dependencies:**\n"
			for _, dep := range request.MCPContext.Dependencies {
				prompt += fmt.Sprintf("- %s\n", dep)
			}
			prompt += "\n"
		}

		if request.MCPContext.CodeAnalysis != "" {
			prompt += "**Code Analysis:**\n"
			prompt += request.MCPContext.CodeAnalysis
			prompt += "\n\n"
		}

		if len(request.MCPContext.Suggestions) > 0 {
			prompt += "**MCP Suggestions:**\n"
			for _, suggestion := range request.MCPContext.Suggestions {
				prompt += fmt.Sprintf("- %s\n", suggestion)
			}
			prompt += "\n"
		}
	}

	prompt += "Please provide a JSON response with the following structure:\n"
	prompt += "{\n"
	prompt += "  \"proposed_fix\": \"// Your corrected Go code here\",\n"
	prompt += "  \"explanation\": \"Detailed explanation of the fix and why it works\",\n"
	prompt += "  \"confidence\": 0.85\n"
	prompt += "}\n\n"
	prompt += "Focus on providing a minimal, targeted fix that addresses the root cause while following Go best practices."

	return prompt
}

// getClaudeSystemPrompt returns the system prompt optimized for Claude
func (c *ClaudeClient) getClaudeSystemPrompt() string {
	return "You are an expert Go developer with deep knowledge of runtime error debugging and code fixing. Your expertise includes:\n\n" +
		"- Analyzing Go panic traces and runtime errors\n" +
		"- Understanding Go memory management and pointer safety\n" +
		"- Applying Go best practices and idioms\n" +
		"- Writing safe, efficient Go code\n" +
		"- Debugging concurrent Go programs\n\n" +
		"When fixing errors:\n" +
		"1. Identify the root cause, not just symptoms\n" +
		"2. Provide minimal, targeted fixes\n" +
		"3. Ensure proper error handling\n" +
		"4. Follow Go conventions and best practices\n" +
		"5. Consider edge cases and potential side effects\n" +
		"6. Be conservative with confidence scores\n\n" +
		"Always respond with valid JSON in the exact format requested."
}

// makeClaudeAPICall makes an HTTP request to Claude API
func (c *ClaudeClient) makeClaudeAPICall(ctx context.Context, request claudeRequest) (*claudeResponse, error) {
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
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Claude API returned status %d", resp.StatusCode)
	}

	var claudeResp claudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &claudeResp, nil
}

// parseClaudeResponse parses Claude API response into FixResponse
func (c *ClaudeClient) parseClaudeResponse(response *claudeResponse) (*FixResponse, error) {
	if len(response.Content) == 0 {
		return nil, fmt.Errorf("empty response from Claude")
	}

	// Extract text content
	text := response.Content[0].Text

	// Try to parse as JSON
	var jsonResponse struct {
		ProposedFix string  `json:"proposed_fix"`
		Explanation string  `json:"explanation"`
		Confidence  float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(text), &jsonResponse); err != nil {
		// If JSON parsing fails, try to extract information from text
		return &FixResponse{
			ProposedFix: text,
			Explanation: "Claude provided a text response that couldn't be parsed as JSON",
			Confidence:  0.5,
			IsValid:     false,
		}, nil
	}

	// Validate confidence score
	if jsonResponse.Confidence < 0 {
		jsonResponse.Confidence = 0
	} else if jsonResponse.Confidence > 1 {
		jsonResponse.Confidence = 1
	}

	return &FixResponse{
		ProposedFix: jsonResponse.ProposedFix,
		Explanation: jsonResponse.Explanation,
		Confidence:  jsonResponse.Confidence,
		IsValid:     jsonResponse.ProposedFix != "",
	}, nil
}
