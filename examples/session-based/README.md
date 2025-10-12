# Session-Based AI Error Handling

This example demonstrates the advanced session-based approach to AI error handling with the Go Code Healer. The session-based approach provides comprehensive error analysis, enhanced context gathering via MCP, and intelligent fix generation using multiple AI providers.

## Key Features

### 🤖 Multi-AI Provider Support
- **Primary Provider**: Configure your preferred AI provider (OpenAI, Claude, Codex)
- **Automatic Fallback**: If the primary provider fails, automatically try fallback providers
- **Provider-Specific Optimization**: Each provider uses optimized prompts and parsing

### 🔗 MCP (Model Context Protocol) Integration
- **Enhanced Context**: Gather additional context about your codebase structure
- **Tool Integration**: Connect to filesystem analyzers, code analyzers, and environment tools
- **Smart Aggregation**: Combine context from multiple MCP sources for comprehensive analysis

### 🎯 Session-Based Processing
- **Comprehensive Analysis**: Each error gets a dedicated AI session with full context
- **Structured Workflow**: Systematic approach from error capture to PR creation
- **Detailed Tracking**: Complete audit trail of the analysis and fix process

## How It Works

### 1. Session Initiation
```go
// Create provider manager with multiple AI providers
providerManager, err := ai.NewProviderManager(config, logger)

// Create a new AI session
session := providerManager.CreateSession(gitClient)
```

### 2. Context Gathering
```go
// The session automatically:
// - Analyzes the error and stack trace
// - Queries MCP tools for additional context
// - Gathers project structure and dependencies
// - Collects environment information
```

### 3. AI Analysis
```go
// Enhanced AI processing:
// - Uses comprehensive context for better understanding
// - Tries multiple AI providers with fallback
// - Generates targeted fixes with explanations
// - Provides confidence scoring
```

### 4. PR Creation
```go
// Automated PR generation:
// - Creates descriptive branch names
// - Applies the generated fix
// - Creates comprehensive PR descriptions
// - Includes analysis and context information
```

## Configuration Example

```json
{
  "ai_provider": "claude",
  "openai_api_key": "sk-your-openai-key",
  "claude_api_key": "sk-ant-your-claude-key", 
  "codex_api_key": "sk-your-codex-key",
  "mcp_enabled": true,
  "mcp_servers": [
    {
      "name": "filesystem-analyzer",
      "endpoint": "http://localhost:8001/mcp",
      "tools": ["analyze_structure", "find_dependencies"]
    },
    {
      "name": "code-analyzer",
      "endpoint": "http://localhost:8002/mcp",
      "auth_type": "bearer",
      "auth_token": "your-token",
      "tools": ["parse_ast", "analyze_symbols"]
    }
  ],
  "github_token": "ghp_your-github-token",
  "repo_owner": "your-username",
  "repo_name": "your-repository"
}
```

## Running the Example

1. **Configure API Keys**:
   ```bash
   export HEALER_CLAUDE_API_KEY="sk-ant-your-key"
   export HEALER_OPENAI_API_KEY="sk-your-key"
   export HEALER_GITHUB_TOKEN="ghp_your-token"
   export HEALER_REPO_OWNER="your-username"
   export HEALER_REPO_NAME="your-repo"
   ```

2. **Optional: Start MCP Servers**:
   ```bash
   # Start your MCP servers (if available)
   # The example will work without them, but with reduced context
   ```

3. **Run the Demo**:
   ```bash
   go run main.go
   ```

## Example Output

```
=== AI Session-Based Error Handling Demo ===
Provider Status: map[mcp_enabled:true primary_provider:claude providers:[claude openai codex]]

--- Simulating Nil Pointer Error ---
Initiating AI session session_1704067200000000000 for error: runtime error: invalid memory address or nil pointer dereference
Gathered context from 2/2 MCP servers
Claude generated fix with confidence 0.85
Creating PR: AI Fix: runtime error: invalid memory address or nil pointer dereference in user.go
Branch: fix/session_1704067200000000000-runtime-error-invalid-memory
Files changed: 1

Session completed successfully!
Session ID: session_1704067200000000000
Duration: 2.3s
AI Provider: claude
Used MCP: true
Confidence: 0.85
PR Created: fix/session_1704067200000000000-runtime-error-invalid-memory
```

## Session Result Structure

Each session returns comprehensive results:

```go
type SessionResult struct {
    SessionID   string           // Unique session identifier
    Success     bool             // Whether the session completed successfully
    FixResponse *FixResponse     // AI-generated fix with explanation
    PRResult    *PRResult        // Pull request creation result
    Duration    time.Duration    // Total processing time
    Context     *SessionContext  // Complete session context
    Timestamp   time.Time        // Session completion time
}
```

## Benefits

### 🎯 **More Accurate Fixes**
- AI has comprehensive context about your codebase
- Multiple providers ensure the best possible solution
- MCP tools provide deep insights into project structure

### 🔄 **Reliable Processing**
- Automatic fallback between AI providers
- Graceful handling of MCP tool failures
- Comprehensive error handling and retry logic

### 📊 **Complete Visibility**
- Detailed session tracking and audit trails
- Confidence scoring for generated fixes
- Comprehensive PR descriptions with analysis

### 🚀 **Production Ready**
- Non-blocking background processing
- Configurable timeouts and retry logic
- Graceful degradation when services are unavailable

## Error Types Handled

The session-based approach excels at handling various Go runtime errors:

- **Nil Pointer Dereference**: Adds proper nil checks
- **Index Out of Bounds**: Implements bounds checking
- **Concurrent Map Access**: Adds synchronization
- **Interface Conversion**: Adds type assertions
- **Channel Operations**: Handles channel closing and blocking

Each error type gets specialized analysis and context-aware fixes.