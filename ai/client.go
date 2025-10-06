package ai

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Logger interface for AI client logging
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// FixRequest represents a request for an AI-generated fix
type FixRequest struct {
	Error      string `json:"error"`
	StackTrace string `json:"stack_trace"`
	SourceCode string `json:"source_code"`
	Context    string `json:"context"`
}

// FixResponse represents the AI's response with a proposed fix
type FixResponse struct {
	ProposedFix string  `json:"proposed_fix"`
	Explanation string  `json:"explanation"`
	Confidence  float64 `json:"confidence"`
	IsValid     bool    `json:"is_valid"`
}

// Client interface for AI fix generation
type Client interface {
	GenerateFix(ctx context.Context, request FixRequest) (*FixResponse, error)
}

// OpenAIClient implements the Client interface for OpenAI API integration
type OpenAIClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
	logger     Logger

	// Embedded components
	promptGenerator *PromptGenerator
	responseParser  *ResponseParser
	codeValidator   *CodeValidator
	httpHandler     *HTTPHandler
}

// NewOpenAIClient creates a new OpenAI client with proper HTTP client configuration
func NewOpenAIClient(apiKey, model string, logger Logger) *OpenAIClient {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	client := &OpenAIClient{
		apiKey:     apiKey,
		model:      model,
		httpClient: httpClient,
		logger:     logger,
	}

	// Initialize embedded components
	client.promptGenerator = NewPromptGenerator()
	client.responseParser = NewResponseParser(logger)
	client.codeValidator = NewCodeValidator(logger)
	client.httpHandler = NewHTTPHandler(httpClient, logger)

	return client
}

// GenerateFix sends a request to OpenAI and returns a proposed fix with enhanced error handling
func (ai *OpenAIClient) GenerateFix(ctx context.Context, request FixRequest) (*FixResponse, error) {
	// Add timeout to context if not already present
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
	}

	// Validate and sanitize input request
	if err := ai.validateFixRequest(request); err != nil {
		return nil, fmt.Errorf("invalid fix request: %w", err)
	}

	// Generate structured prompt for Go code fixes
	prompt := ai.promptGenerator.GeneratePrompt(request)

	// Create OpenAI API request with enhanced parameters
	apiRequest := openAIRequest{
		Model: ai.model,
		Messages: []openAIMessage{
			{
				Role:    "system",
				Content: ai.promptGenerator.GetSystemPrompt(),
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.1, // Low temperature for more deterministic code generation
		MaxTokens:   2000,
		TopP:        0.9,
	}

	// Make API call with retry logic for rate limits and transient errors
	response, err := ai.httpHandler.MakeAPICallWithRetry(ctx, apiRequest, ai.apiKey)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API call failed: %w", err)
	}

	// Parse response and create FixResponse with enhanced validation
	fixResponse, err := ai.responseParser.ParseResponseWithValidation(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	// Validate the proposed Go code for syntax correctness and calculate confidence
	fixResponse.IsValid = ai.codeValidator.ValidateGoSyntax(fixResponse.ProposedFix)
	fixResponse.Confidence = ai.adjustConfidenceScore(fixResponse.Confidence, fixResponse.IsValid, request)

	// Log the result for debugging
	if ai.logger != nil {
		ai.logger.Debug("Generated fix with confidence %.2f, valid: %v", fixResponse.Confidence, fixResponse.IsValid)
	}

	return fixResponse, nil
}

// validateFixRequest validates and sanitizes the input request
func (ai *OpenAIClient) validateFixRequest(request FixRequest) error {
	if request.Error == "" {
		return fmt.Errorf("error field is required")
	}

	// Truncate fields if they're too long to prevent API limits
	const maxFieldLength = 8000 // Leave room for other content and API overhead

	if len(request.StackTrace) > maxFieldLength {
		request.StackTrace = request.StackTrace[:maxFieldLength] + "\n... (truncated)"
	}

	if len(request.SourceCode) > maxFieldLength {
		request.SourceCode = request.SourceCode[:maxFieldLength] + "\n// ... (truncated)"
	}

	if len(request.Context) > maxFieldLength {
		request.Context = request.Context[:maxFieldLength] + "\n... (truncated)"
	}

	return nil
}

// adjustConfidenceScore adjusts the confidence score based on validation results and request complexity
func (ai *OpenAIClient) adjustConfidenceScore(originalConfidence float64, isValid bool, request FixRequest) float64 {
	confidence := originalConfidence

	// Reduce confidence if syntax validation failed
	if !isValid {
		confidence *= 0.5
		if ai.logger != nil {
			ai.logger.Debug("Reducing confidence due to syntax validation failure")
		}
	}

	// Adjust confidence based on error complexity
	errorComplexity := ai.assessErrorComplexity(request)
	switch errorComplexity {
	case "simple":
		// Simple errors like nil pointer dereference - boost confidence slightly
		if confidence*1.1 > 1.0 {
			confidence = 1.0
		} else {
			confidence *= 1.1
		}
	case "complex":
		// Complex errors involving concurrency, interfaces, etc. - reduce confidence
		confidence *= 0.8
	case "unknown":
		// Unknown or unusual errors - be more conservative
		confidence *= 0.7
	}

	// Ensure confidence stays within valid range
	if confidence < 0.0 {
		confidence = 0.0
	} else if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// assessErrorComplexity analyzes the error to determine its complexity level
func (ai *OpenAIClient) assessErrorComplexity(request FixRequest) string {
	return ai.codeValidator.AssessErrorComplexity(request)
}
