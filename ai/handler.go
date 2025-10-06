package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HTTPHandler handles HTTP requests to the OpenAI API
type HTTPHandler struct {
	httpClient *http.Client
	logger     Logger
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(httpClient *http.Client, logger Logger) *HTTPHandler {
	return &HTTPHandler{
		httpClient: httpClient,
		logger:     logger,
	}
}

// MakeAPICallWithRetry performs the HTTP request with retry logic for rate limits
func (hh *HTTPHandler) MakeAPICallWithRetry(ctx context.Context, request openAIRequest, apiKey string) (*openAIResponse, error) {
	maxRetries := 3
	baseDelay := 1 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		response, err := hh.makeAPICall(ctx, request, apiKey)
		if err == nil {
			return response, nil
		}

		// Check if we should retry
		shouldRetry, delay := hh.handleAPIRateLimit(err)
		if !shouldRetry || attempt == maxRetries {
			return nil, err
		}

		// Calculate exponential backoff delay
		retryDelay := time.Duration(attempt+1) * baseDelay
		if delay > retryDelay {
			retryDelay = delay
		}

		if hh.logger != nil {
			hh.logger.Debug("API call failed (attempt %d/%d), retrying in %v: %v",
				attempt+1, maxRetries+1, retryDelay, err)
		}

		// Wait before retrying, respecting context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(retryDelay):
			// Continue to next attempt
		}
	}

	return nil, fmt.Errorf("max retries exceeded")
}

// makeAPICall performs the HTTP request to OpenAI API
func (hh *HTTPHandler) makeAPICall(ctx context.Context, request openAIRequest, apiKey string) (*openAIResponse, error) {
	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	// Log the request (without API key)
	if hh.logger != nil {
		hh.logger.Debug("Making OpenAI API request to model: %s", request.Model)
	}

	// Make the request
	resp, err := hh.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var apiResponse openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API errors
	if apiResponse.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)",
			apiResponse.Error.Message, apiResponse.Error.Type, apiResponse.Error.Code)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API returned status %d", resp.StatusCode)
	}

	// Check if we have choices
	if len(apiResponse.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI API returned no choices")
	}

	return &apiResponse, nil
}

// handleAPIRateLimit handles rate limiting and retry logic for API calls
func (hh *HTTPHandler) handleAPIRateLimit(err error) (shouldRetry bool, delay time.Duration) {
	if err == nil {
		return false, 0
	}

	errStr := strings.ToLower(err.Error())

	// Check for rate limit errors
	if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "too many requests") {
		return true, 60 * time.Second // Wait 1 minute for rate limits
	}

	// Check for temporary network errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "connection") {
		return true, 5 * time.Second // Wait 5 seconds for network issues
	}

	// Check for server errors (5xx)
	if strings.Contains(errStr, "status 5") {
		return true, 10 * time.Second // Wait 10 seconds for server errors
	}

	return false, 0
}
