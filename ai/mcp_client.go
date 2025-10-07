package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ajeet-kumar1087/go-code-healer/internal"
)

// MCPServerConfig is an alias to internal.MCPServerConfig
type MCPServerConfig = internal.MCPServerConfig

// ContextRequest represents a request for additional context from MCP tools
type ContextRequest struct {
	ErrorType  string            `json:"error_type"`
	SourceFile string            `json:"source_file"`
	Function   string            `json:"function"`
	StackTrace string            `json:"stack_trace"`
	Metadata   map[string]string `json:"metadata"`
}

// ContextResponse represents aggregated context from MCP tools
type ContextResponse struct {
	FileStructure string            `json:"file_structure,omitempty"`
	Dependencies  []string          `json:"dependencies,omitempty"`
	CodeAnalysis  string            `json:"code_analysis,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
	RelatedFiles  []string          `json:"related_files,omitempty"`
	Suggestions   []string          `json:"suggestions,omitempty"`
	Confidence    float64           `json:"confidence"`
	Sources       []string          `json:"sources"` // which MCP tools provided data
}

// MCPClient handles communication with MCP servers
type MCPClient struct {
	servers    []MCPServerConfig
	httpClient *http.Client
	logger     internal.LoggerInterface
	timeout    time.Duration
}

// NewMCPClient creates a new MCP client with the given configuration
func NewMCPClient(servers []MCPServerConfig, timeout time.Duration, logger internal.LoggerInterface) *MCPClient {
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	return &MCPClient{
		servers: servers,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		logger:  logger,
		timeout: timeout,
	}
}

// GatherContext collects additional context from configured MCP servers
func (mc *MCPClient) GatherContext(ctx context.Context, request ContextRequest) (*ContextResponse, error) {
	if len(mc.servers) == 0 {
		return &ContextResponse{
			Confidence: 0.0,
			Sources:    []string{},
		}, nil
	}

	response := &ContextResponse{
		Environment:  make(map[string]string),
		Dependencies: []string{},
		RelatedFiles: []string{},
		Suggestions:  []string{},
		Sources:      []string{},
		Confidence:   0.0,
	}

	// Gather context from each configured MCP server
	successCount := 0
	for _, server := range mc.servers {
		serverCtx, cancel := context.WithTimeout(ctx, mc.getServerTimeout(server))
		defer cancel()

		serverResponse, err := mc.queryMCPServer(serverCtx, server, request)
		if err != nil {
			if mc.logger != nil {
				mc.logger.Warn("Failed to gather context from MCP server %s: %v", server.Name, err)
			}
			continue
		}

		// Merge server response into aggregated response
		mc.mergeContextResponse(response, serverResponse, server.Name)
		successCount++
	}

	// Calculate overall confidence based on successful responses
	if successCount > 0 {
		response.Confidence = float64(successCount) / float64(len(mc.servers))
	}

	if mc.logger != nil {
		mc.logger.Debug("Gathered context from %d/%d MCP servers", successCount, len(mc.servers))
	}

	return response, nil
}

// ValidateServers checks connectivity and available tools for all configured MCP servers
func (mc *MCPClient) ValidateServers(ctx context.Context) error {
	if len(mc.servers) == 0 {
		return fmt.Errorf("no MCP servers configured")
	}

	var errors []string
	for _, server := range mc.servers {
		if err := mc.validateServer(ctx, server); err != nil {
			errors = append(errors, fmt.Sprintf("server %s: %v", server.Name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("MCP server validation failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

// queryMCPServer queries a specific MCP server for context
func (mc *MCPClient) queryMCPServer(ctx context.Context, server MCPServerConfig, request ContextRequest) (*ContextResponse, error) {
	// Create MCP-compliant request
	mcpRequest := map[string]interface{}{
		"method": "tools/call",
		"params": map[string]interface{}{
			"name":      "gather_context",
			"arguments": request,
		},
	}

	reqBody, err := json.Marshal(mcpRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", server.Endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add authentication if configured
	mc.addAuthentication(httpReq, server)

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "go-code-healer/1.0")

	// Make the request
	resp, err := mc.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP server returned status %d", resp.StatusCode)
	}

	// Parse MCP response
	var mcpResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&mcpResponse); err != nil {
		return nil, fmt.Errorf("failed to decode MCP response: %w", err)
	}

	// Extract context from MCP response
	return mc.extractContextFromMCPResponse(mcpResponse)
}

// validateServer validates connectivity to a specific MCP server
func (mc *MCPClient) validateServer(ctx context.Context, server MCPServerConfig) error {
	// Create a simple ping request to validate connectivity
	pingRequest := map[string]interface{}{
		"method": "ping",
		"params": map[string]interface{}{},
	}

	reqBody, err := json.Marshal(pingRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal ping request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", server.Endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		return fmt.Errorf("failed to create ping request: %w", err)
	}

	mc.addAuthentication(httpReq, server)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := mc.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("ping request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}

// addAuthentication adds authentication headers based on server configuration
func (mc *MCPClient) addAuthentication(req *http.Request, server MCPServerConfig) {
	switch server.AuthType {
	case "bearer":
		if server.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+server.AuthToken)
		}
	case "basic":
		if server.AuthToken != "" {
			req.Header.Set("Authorization", "Basic "+server.AuthToken)
		}
	case "none", "":
		// No authentication required
	default:
		if mc.logger != nil {
			mc.logger.Warn("Unknown auth type %s for MCP server %s", server.AuthType, server.Name)
		}
	}
}

// getServerTimeout returns the timeout for a specific server
func (mc *MCPClient) getServerTimeout(server MCPServerConfig) time.Duration {
	if server.Timeout > 0 {
		return time.Duration(server.Timeout) * time.Second
	}
	return mc.timeout
}

// mergeContextResponse merges a server response into the aggregated response
func (mc *MCPClient) mergeContextResponse(aggregate *ContextResponse, serverResponse *ContextResponse, serverName string) {
	// Add server to sources
	aggregate.Sources = append(aggregate.Sources, serverName)

	// Merge file structure (prefer non-empty values)
	if serverResponse.FileStructure != "" && aggregate.FileStructure == "" {
		aggregate.FileStructure = serverResponse.FileStructure
	}

	// Merge code analysis (prefer non-empty values)
	if serverResponse.CodeAnalysis != "" && aggregate.CodeAnalysis == "" {
		aggregate.CodeAnalysis = serverResponse.CodeAnalysis
	}

	// Merge dependencies (append unique values)
	for _, dep := range serverResponse.Dependencies {
		if !contains(aggregate.Dependencies, dep) {
			aggregate.Dependencies = append(aggregate.Dependencies, dep)
		}
	}

	// Merge related files (append unique values)
	for _, file := range serverResponse.RelatedFiles {
		if !contains(aggregate.RelatedFiles, file) {
			aggregate.RelatedFiles = append(aggregate.RelatedFiles, file)
		}
	}

	// Merge suggestions (append unique values)
	for _, suggestion := range serverResponse.Suggestions {
		if !contains(aggregate.Suggestions, suggestion) {
			aggregate.Suggestions = append(aggregate.Suggestions, suggestion)
		}
	}

	// Merge environment variables
	for key, value := range serverResponse.Environment {
		if _, exists := aggregate.Environment[key]; !exists {
			aggregate.Environment[key] = value
		}
	}
}

// extractContextFromMCPResponse extracts context information from MCP response
func (mc *MCPClient) extractContextFromMCPResponse(mcpResponse map[string]interface{}) (*ContextResponse, error) {
	// Extract result from MCP response
	result, ok := mcpResponse["result"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid MCP response format: missing result")
	}

	response := &ContextResponse{
		Environment:  make(map[string]string),
		Dependencies: []string{},
		RelatedFiles: []string{},
		Suggestions:  []string{},
		Sources:      []string{},
		Confidence:   1.0, // Individual server confidence
	}

	// Extract fields from result
	if fileStructure, ok := result["file_structure"].(string); ok {
		response.FileStructure = fileStructure
	}

	if codeAnalysis, ok := result["code_analysis"].(string); ok {
		response.CodeAnalysis = codeAnalysis
	}

	if deps, ok := result["dependencies"].([]interface{}); ok {
		for _, dep := range deps {
			if depStr, ok := dep.(string); ok {
				response.Dependencies = append(response.Dependencies, depStr)
			}
		}
	}

	if files, ok := result["related_files"].([]interface{}); ok {
		for _, file := range files {
			if fileStr, ok := file.(string); ok {
				response.RelatedFiles = append(response.RelatedFiles, fileStr)
			}
		}
	}

	if suggestions, ok := result["suggestions"].([]interface{}); ok {
		for _, suggestion := range suggestions {
			if suggestionStr, ok := suggestion.(string); ok {
				response.Suggestions = append(response.Suggestions, suggestionStr)
			}
		}
	}

	if env, ok := result["environment"].(map[string]interface{}); ok {
		for key, value := range env {
			if valueStr, ok := value.(string); ok {
				response.Environment[key] = valueStr
			}
		}
	}

	if confidence, ok := result["confidence"].(float64); ok {
		response.Confidence = confidence
	}

	return response, nil
}

// contains checks if a slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
