package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ResponseParser handles parsing of AI responses
type ResponseParser struct {
	logger Logger
}

// NewResponseParser creates a new response parser
func NewResponseParser(logger Logger) *ResponseParser {
	return &ResponseParser{
		logger: logger,
	}
}

// ParseResponseWithValidation converts OpenAI response to FixResponse with enhanced validation
func (rp *ResponseParser) ParseResponseWithValidation(response *openAIResponse) (*FixResponse, error) {
	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices in OpenAI response")
	}

	choice := response.Choices[0]

	// Check finish reason
	if choice.FinishReason == "length" {
		if rp.logger != nil {
			rp.logger.Warn("OpenAI response was truncated due to length limit")
		}
	}

	content := choice.Message.Content

	// Try to parse as JSON first
	var jsonResponse struct {
		ProposedFix string  `json:"proposed_fix"`
		Explanation string  `json:"explanation"`
		Confidence  float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(content), &jsonResponse); err != nil {
		// If JSON parsing fails, try to extract information from plain text
		if rp.logger != nil {
			rp.logger.Debug("Failed to parse JSON response, attempting plain text extraction: %v", err)
		}

		return rp.parseTextResponse(content)
	}

	// Validate and sanitize the response
	fixResponse := &FixResponse{
		ProposedFix: strings.TrimSpace(jsonResponse.ProposedFix),
		Explanation: strings.TrimSpace(jsonResponse.Explanation),
		Confidence:  jsonResponse.Confidence,
		IsValid:     false, // Will be set by validateGoSyntax
	}

	// Validate confidence score
	if fixResponse.Confidence < 0.0 || fixResponse.Confidence > 1.0 {
		if rp.logger != nil {
			rp.logger.Debug("Invalid confidence score %.2f, defaulting to 0.5", fixResponse.Confidence)
		}
		fixResponse.Confidence = 0.5 // Default to medium confidence
	}

	// Ensure we have some content
	if fixResponse.ProposedFix == "" && fixResponse.Explanation == "" {
		return nil, fmt.Errorf("OpenAI response contains no useful content")
	}

	return fixResponse, nil
}

// parseTextResponse attempts to extract fix information from plain text response
func (rp *ResponseParser) parseTextResponse(content string) (*FixResponse, error) {
	// This is a fallback for when the AI doesn't return proper JSON
	// We'll do our best to extract useful information

	lines := strings.Split(content, "\n")
	var proposedFix, explanation strings.Builder
	var confidence float64 = 0.5 // Default confidence

	inCodeBlock := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect code blocks
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			proposedFix.WriteString(line + "\n")
		} else if line != "" {
			explanation.WriteString(line + " ")
		}
	}

	// Try to extract confidence from text
	lowerContent := strings.ToLower(content)
	if strings.Contains(lowerContent, "high confidence") || strings.Contains(lowerContent, "confident") {
		confidence = 0.8
	} else if strings.Contains(lowerContent, "low confidence") || strings.Contains(lowerContent, "uncertain") {
		confidence = 0.3
	}

	return &FixResponse{
		ProposedFix: strings.TrimSpace(proposedFix.String()),
		Explanation: strings.TrimSpace(explanation.String()),
		Confidence:  confidence,
		IsValid:     false, // Will be set by validateGoSyntax
	}, nil
}
