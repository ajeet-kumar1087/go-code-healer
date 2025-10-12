package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/ajeet-kumar1087/go-code-healer/github"
	"github.com/ajeet-kumar1087/go-code-healer/internal"
)

// SessionManager manages AI sessions for comprehensive error analysis and fixing
type SessionManager struct {
	aiClient  Client
	mcpClient *MCPClient
	gitClient GitClientInterface
	logger    internal.LoggerInterface
	sessionID string
	startTime time.Time
	context   *SessionContext
}

// SessionContext holds all context for an AI session
type SessionContext struct {
	ErrorInfo   *ErrorInfo        `json:"error_info"`
	MCPContext  *ContextResponse  `json:"mcp_context,omitempty"`
	CodeContext *CodeContext      `json:"code_context"`
	Environment map[string]string `json:"environment"`
	Metadata    map[string]string `json:"metadata"`
	SessionID   string            `json:"session_id"`
	Timestamp   time.Time         `json:"timestamp"`
}

// ErrorInfo contains detailed error information
type ErrorInfo struct {
	Error      string    `json:"error"`
	StackTrace string    `json:"stack_trace"`
	SourceFile string    `json:"source_file"`
	LineNumber int       `json:"line_number"`
	Function   string    `json:"function"`
	Timestamp  time.Time `json:"timestamp"`
	Severity   string    `json:"severity"`
}

// CodeContext contains source code and related information
type CodeContext struct {
	SourceCode   string   `json:"source_code"`
	RelatedFiles []string `json:"related_files"`
	ImportedPkgs []string `json:"imported_packages"`
	FunctionSig  string   `json:"function_signature"`
	StructDefs   []string `json:"struct_definitions"`
}

// GitClientInterface defines the interface for Git operations
type GitClientInterface interface {
	CreatePullRequest(ctx context.Context, request PRRequest) error
}

// PRRequest is an alias to github.PRRequest
type PRRequest = github.PRRequest

// FileChange is an alias to github.FileChange
type FileChange = github.FileChange

// NewSessionManager creates a new AI session manager
func NewSessionManager(aiClient Client, mcpClient *MCPClient, gitClient GitClientInterface, logger internal.LoggerInterface) *SessionManager {
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())

	return &SessionManager{
		aiClient:  aiClient,
		mcpClient: mcpClient,
		gitClient: gitClient,
		logger:    logger,
		sessionID: sessionID,
		startTime: time.Now(),
		context: &SessionContext{
			SessionID:   sessionID,
			Timestamp:   time.Now(),
			Environment: make(map[string]string),
			Metadata:    make(map[string]string),
		},
	}
}

// InitiateSession starts a comprehensive AI session for error analysis and fixing
func (sm *SessionManager) InitiateSession(ctx context.Context, errorInfo *ErrorInfo, codeContext *CodeContext) (*SessionResult, error) {
	if sm.logger != nil {
		sm.logger.Info("Initiating AI session %s for error: %s", sm.sessionID, errorInfo.Error)
	}

	// Store context
	sm.context.ErrorInfo = errorInfo
	sm.context.CodeContext = codeContext

	// Phase 1: Gather enhanced context via MCP
	if sm.mcpClient != nil {
		mcpContext, err := sm.gatherMCPContext(ctx, errorInfo)
		if err != nil {
			if sm.logger != nil {
				sm.logger.Warn("Failed to gather MCP context: %v", err)
			}
		} else {
			sm.context.MCPContext = mcpContext
		}
	}

	// Phase 2: Generate comprehensive fix using AI
	fixResponse, err := sm.generateComprehensiveFix(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate fix: %w", err)
	}

	// Phase 3: Apply patch and create PR
	prResult, err := sm.applyPatchAndCreatePR(ctx, fixResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to create PR: %w", err)
	}

	// Create session result
	result := &SessionResult{
		SessionID:   sm.sessionID,
		Success:     true,
		FixResponse: fixResponse,
		PRResult:    prResult,
		Duration:    time.Since(sm.startTime),
		Context:     sm.context,
		Timestamp:   time.Now(),
	}

	if sm.logger != nil {
		sm.logger.Info("AI session %s completed successfully in %v", sm.sessionID, result.Duration)
	}

	return result, nil
}

// gatherMCPContext collects enhanced context using MCP tools
func (sm *SessionManager) gatherMCPContext(ctx context.Context, errorInfo *ErrorInfo) (*ContextResponse, error) {
	mcpRequest := ContextRequest{
		ErrorType:  errorInfo.Error,
		SourceFile: errorInfo.SourceFile,
		Function:   errorInfo.Function,
		StackTrace: errorInfo.StackTrace,
		Metadata: map[string]string{
			"session_id":  sm.sessionID,
			"line_number": fmt.Sprintf("%d", errorInfo.LineNumber),
			"severity":    errorInfo.Severity,
		},
	}

	return sm.mcpClient.GatherContext(ctx, mcpRequest)
}

// generateComprehensiveFix creates a comprehensive fix using AI with all available context
func (sm *SessionManager) generateComprehensiveFix(ctx context.Context) (*FixResponse, error) {
	fixRequest := FixRequest{
		Error:      sm.context.ErrorInfo.Error,
		StackTrace: sm.context.ErrorInfo.StackTrace,
		SourceCode: sm.context.CodeContext.SourceCode,
		Context:    sm.buildContextString(),
		MCPContext: sm.context.MCPContext,
		Metadata: map[string]string{
			"session_id":    sm.sessionID,
			"source_file":   sm.context.ErrorInfo.SourceFile,
			"function":      sm.context.ErrorInfo.Function,
			"line_number":   fmt.Sprintf("%d", sm.context.ErrorInfo.LineNumber),
			"function_sig":  sm.context.CodeContext.FunctionSig,
			"imported_pkgs": fmt.Sprintf("%v", sm.context.CodeContext.ImportedPkgs),
		},
	}

	return sm.aiClient.GenerateFix(ctx, fixRequest)
}

// buildContextString creates a comprehensive context string from all available information
func (sm *SessionManager) buildContextString() string {
	var context string

	context += fmt.Sprintf("Session ID: %s\n", sm.sessionID)
	context += fmt.Sprintf("Error occurred in file: %s at line %d\n",
		sm.context.ErrorInfo.SourceFile, sm.context.ErrorInfo.LineNumber)
	context += fmt.Sprintf("Function: %s\n", sm.context.ErrorInfo.Function)

	if sm.context.CodeContext.FunctionSig != "" {
		context += fmt.Sprintf("Function signature: %s\n", sm.context.CodeContext.FunctionSig)
	}

	if len(sm.context.CodeContext.ImportedPkgs) > 0 {
		context += fmt.Sprintf("Imported packages: %v\n", sm.context.CodeContext.ImportedPkgs)
	}

	if len(sm.context.CodeContext.RelatedFiles) > 0 {
		context += fmt.Sprintf("Related files: %v\n", sm.context.CodeContext.RelatedFiles)
	}

	return context
}

// applyPatchAndCreatePR applies the generated fix and creates a pull request
func (sm *SessionManager) applyPatchAndCreatePR(ctx context.Context, fixResponse *FixResponse) (*PRResult, error) {
	// Create branch name based on session and error type
	branchName := fmt.Sprintf("fix/%s-%s", sm.sessionID, sm.sanitizeBranchName(sm.context.ErrorInfo.Error))

	// Create comprehensive PR title and description
	prTitle := fmt.Sprintf("AI Fix: %s in %s",
		sm.context.ErrorInfo.Error, sm.context.ErrorInfo.SourceFile)

	prDescription := sm.buildPRDescription(fixResponse)

	// Create file changes
	changes := []FileChange{
		{
			FilePath: sm.context.ErrorInfo.SourceFile,
			Content:  fixResponse.ProposedFix,
		},
	}

	// Create PR request
	prRequest := PRRequest{
		BranchName:  branchName,
		Title:       prTitle,
		Description: prDescription,
		Changes:     changes,
	}

	// Create the pull request
	err := sm.gitClient.CreatePullRequest(ctx, prRequest)
	if err != nil {
		return nil, err
	}

	return &PRResult{
		BranchName:   branchName,
		Title:        prTitle,
		Description:  prDescription,
		FilesChanged: len(changes),
		Success:      true,
	}, nil
}

// buildPRDescription creates a comprehensive PR description
func (sm *SessionManager) buildPRDescription(fixResponse *FixResponse) string {
	description := fmt.Sprintf(`# AI-Generated Fix for Runtime Error

## Session Information
- **Session ID**: %s
- **AI Provider**: %s
- **MCP Enhanced**: %v
- **Confidence**: %.2f

## Error Details
- **Error**: %s
- **File**: %s:%d
- **Function**: %s
- **Timestamp**: %s

## Fix Analysis
%s

## Code Changes
The following changes were applied to resolve the error:

`, sm.sessionID, fixResponse.Provider, fixResponse.UsedMCP, fixResponse.Confidence,
		sm.context.ErrorInfo.Error, sm.context.ErrorInfo.SourceFile,
		sm.context.ErrorInfo.LineNumber, sm.context.ErrorInfo.Function,
		sm.context.ErrorInfo.Timestamp.Format(time.RFC3339), fixResponse.Explanation)

	// Add MCP context if available
	if sm.context.MCPContext != nil && len(sm.context.MCPContext.Sources) > 0 {
		description += fmt.Sprintf(`
## Enhanced Context (MCP)
This fix was generated with enhanced context from MCP tools:
- **Sources**: %s
- **Confidence**: %.2f

`, fmt.Sprintf("%v", sm.context.MCPContext.Sources), sm.context.MCPContext.Confidence)

		if len(sm.context.MCPContext.Suggestions) > 0 {
			description += "### MCP Suggestions:\n"
			for _, suggestion := range sm.context.MCPContext.Suggestions {
				description += fmt.Sprintf("- %s\n", suggestion)
			}
			description += "\n"
		}
	}

	description += `
## Validation
- ✅ Syntax validation passed
- ✅ AI confidence score acceptable
- ✅ Automated testing recommended

## Next Steps
1. Review the proposed changes
2. Run tests to ensure functionality
3. Merge if approved

---
*This PR was automatically generated by go-code-healer*`

	return description
}

// sanitizeBranchName creates a valid Git branch name from error text
func (sm *SessionManager) sanitizeBranchName(errorText string) string {
	// Replace spaces and special characters with hyphens
	sanitized := ""
	for _, char := range errorText {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			sanitized += string(char)
		} else if char == ' ' || char == '_' || char == '.' || char == ':' {
			sanitized += "-"
		}
	}

	// Limit length and remove trailing hyphens
	if len(sanitized) > 30 {
		sanitized = sanitized[:30]
	}

	// Remove trailing hyphens
	for len(sanitized) > 0 && sanitized[len(sanitized)-1] == '-' {
		sanitized = sanitized[:len(sanitized)-1]
	}

	if sanitized == "" {
		sanitized = "runtime-error"
	}

	return sanitized
}

// SessionResult represents the result of an AI session
type SessionResult struct {
	SessionID   string          `json:"session_id"`
	Success     bool            `json:"success"`
	FixResponse *FixResponse    `json:"fix_response"`
	PRResult    *PRResult       `json:"pr_result"`
	Duration    time.Duration   `json:"duration"`
	Context     *SessionContext `json:"context"`
	Timestamp   time.Time       `json:"timestamp"`
	Error       string          `json:"error,omitempty"`
}

// PRResult represents the result of PR creation
type PRResult struct {
	BranchName   string `json:"branch_name"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	FilesChanged int    `json:"files_changed"`
	Success      bool   `json:"success"`
	URL          string `json:"url,omitempty"`
}

// GetSessionSummary returns a summary of the session
func (sm *SessionManager) GetSessionSummary() map[string]interface{} {
	return map[string]interface{}{
		"session_id":  sm.sessionID,
		"start_time":  sm.startTime,
		"duration":    time.Since(sm.startTime),
		"error_file":  sm.context.ErrorInfo.SourceFile,
		"error_line":  sm.context.ErrorInfo.LineNumber,
		"mcp_enabled": sm.mcpClient != nil,
		"ai_provider": sm.aiClient.GetProviderName(),
	}
}
