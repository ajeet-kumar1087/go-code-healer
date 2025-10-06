package ai

import (
	"fmt"
	"strings"
)

// PromptGenerator handles the generation of prompts for AI requests
type PromptGenerator struct{}

// NewPromptGenerator creates a new prompt generator
func NewPromptGenerator() *PromptGenerator {
	return &PromptGenerator{}
}

// GeneratePrompt creates a structured prompt for Go code fixes
func (pg *PromptGenerator) GeneratePrompt(request FixRequest) string {
	var prompt strings.Builder

	prompt.WriteString("I need help fixing a Go panic/error. Here are the details:\n\n")

	prompt.WriteString("## Error Information\n")
	prompt.WriteString(fmt.Sprintf("**Error:** %s\n\n", request.Error))

	if request.StackTrace != "" {
		prompt.WriteString("**Stack Trace:**\n```\n")
		prompt.WriteString(request.StackTrace)
		prompt.WriteString("\n```\n\n")
	}

	if request.SourceCode != "" {
		prompt.WriteString("**Source Code Context:**\n```go\n")
		prompt.WriteString(request.SourceCode)
		prompt.WriteString("\n```\n\n")
	}

	if request.Context != "" {
		prompt.WriteString("**Additional Context:**\n")
		prompt.WriteString(request.Context)
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("Please provide:\n")
	prompt.WriteString("1. A corrected version of the problematic code\n")
	prompt.WriteString("2. A clear explanation of what caused the error\n")
	prompt.WriteString("3. Why your proposed fix addresses the issue\n")
	prompt.WriteString("4. A confidence score (0.0-1.0) for your fix\n\n")

	prompt.WriteString("Format your response as JSON with the following structure:\n")
	prompt.WriteString("{\n")
	prompt.WriteString("  \"proposed_fix\": \"// Your corrected Go code here\",\n")
	prompt.WriteString("  \"explanation\": \"Detailed explanation of the fix\",\n")
	prompt.WriteString("  \"confidence\": 0.85\n")
	prompt.WriteString("}")

	return prompt.String()
}

// GetSystemPrompt returns the system prompt for the AI
func (pg *PromptGenerator) GetSystemPrompt() string {
	return `You are an expert Go developer specializing in debugging and fixing runtime errors. 
Your task is to analyze Go panic/error information and provide accurate, safe fixes.

Guidelines:
- Focus on the root cause of the error, not just symptoms
- Provide minimal, targeted fixes that address the specific issue
- Ensure your code follows Go best practices and idioms
- Include proper error handling in your fixes
- Be conservative with confidence scores - only use high confidence (>0.8) for obvious fixes
- If the fix requires significant architectural changes, suggest a simpler interim solution
- Always provide valid Go syntax in your proposed fixes
- Consider edge cases and potential side effects of your fix

Your response must be valid JSON with the exact structure requested.`
}
