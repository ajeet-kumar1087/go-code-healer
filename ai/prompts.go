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

// GeneratePrompt creates a structured prompt for Go code fixes (legacy method)
func (pg *PromptGenerator) GeneratePrompt(request FixRequest) string {
	return pg.GeneratePromptWithMCP(request)
}

// GeneratePromptWithMCP creates a structured prompt for Go code fixes with MCP context
func (pg *PromptGenerator) GeneratePromptWithMCP(request FixRequest) string {
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

	// Add MCP context if available
	if request.MCPContext != nil {
		prompt.WriteString("## Enhanced Context (from MCP tools)\n")
		pg.addMCPContextToPrompt(&prompt, request.MCPContext)
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

// addMCPContextToPrompt adds MCP-gathered context to the prompt
func (pg *PromptGenerator) addMCPContextToPrompt(prompt *strings.Builder, mcpContext *ContextResponse) {
	if mcpContext.FileStructure != "" {
		prompt.WriteString("**Project Structure:**\n```\n")
		prompt.WriteString(mcpContext.FileStructure)
		prompt.WriteString("\n```\n\n")
	}

	if len(mcpContext.Dependencies) > 0 {
		prompt.WriteString("**Dependencies:**\n")
		for _, dep := range mcpContext.Dependencies {
			prompt.WriteString(fmt.Sprintf("- %s\n", dep))
		}
		prompt.WriteString("\n")
	}

	if mcpContext.CodeAnalysis != "" {
		prompt.WriteString("**Code Analysis:**\n")
		prompt.WriteString(mcpContext.CodeAnalysis)
		prompt.WriteString("\n\n")
	}

	if len(mcpContext.RelatedFiles) > 0 {
		prompt.WriteString("**Related Files:**\n")
		for _, file := range mcpContext.RelatedFiles {
			prompt.WriteString(fmt.Sprintf("- %s\n", file))
		}
		prompt.WriteString("\n")
	}

	if len(mcpContext.Environment) > 0 {
		prompt.WriteString("**Environment Information:**\n")
		for key, value := range mcpContext.Environment {
			prompt.WriteString(fmt.Sprintf("- %s: %s\n", key, value))
		}
		prompt.WriteString("\n")
	}

	if len(mcpContext.Suggestions) > 0 {
		prompt.WriteString("**MCP Tool Suggestions:**\n")
		for _, suggestion := range mcpContext.Suggestions {
			prompt.WriteString(fmt.Sprintf("- %s\n", suggestion))
		}
		prompt.WriteString("\n")
	}

	if len(mcpContext.Sources) > 0 {
		prompt.WriteString(fmt.Sprintf("*Context gathered from MCP tools: %s*\n\n",
			strings.Join(mcpContext.Sources, ", ")))
	}
}
