package healer

import (
	"context"

	"github.com/ajeet-kumar1087/go-code-healer/ai"
)

// OpenAIClient implements the AIClient interface for OpenAI API integration
// This is a wrapper around the modular AI client for backward compatibility
type OpenAIClient struct {
	client *ai.OpenAIClient
}

// NewOpenAIClient creates a new OpenAI client with proper HTTP client configuration
func NewOpenAIClient(apiKey, model string, logger Logger) *OpenAIClient {
	return &OpenAIClient{
		client: ai.NewOpenAIClient(apiKey, model, logger),
	}
}

// GenerateFix delegates to the modular AI client
func (oc *OpenAIClient) GenerateFix(ctx context.Context, request FixRequest) (*FixResponse, error) {
	// Convert healer types to ai types
	aiRequest := ai.FixRequest{
		Error:      request.Error,
		StackTrace: request.StackTrace,
		SourceCode: request.SourceCode,
		Context:    request.Context,
	}

	// Call the modular client
	aiResponse, err := oc.client.GenerateFix(ctx, aiRequest)
	if err != nil {
		return nil, err
	}

	// Convert ai types back to healer types
	return &FixResponse{
		ProposedFix: aiResponse.ProposedFix,
		Explanation: aiResponse.Explanation,
		Confidence:  aiResponse.Confidence,
		IsValid:     aiResponse.IsValid,
	}, nil
}
