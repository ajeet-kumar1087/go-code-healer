# Multi-Provider Example

This example demonstrates the Go Code Healer's multi-provider AI support with automatic fallback capabilities.

## Features Demonstrated

1. **Multiple AI Providers**: Claude (primary), OpenAI (fallback), Codex (fallback)
2. **Provider-Specific Optimization**: Each provider gets optimized prompts
3. **Automatic Fallback**: If one provider fails, automatically tries the next
4. **MCP Integration**: Enhanced context gathering from MCP tools
5. **Confidence-Based Selection**: Best response selection based on confidence scores

## Configuration

The example shows how to configure multiple AI providers:

```json
{
  "ai_provider": "claude",           // Primary provider
  "claude_api_key": "sk-ant-...",    // Claude API key
  "openai_api_key": "sk-...",        // OpenAI fallback
  "codex_api_key": "sk-...",         // Codex fallback
  "mcp_enabled": true,               // Enhanced context
  "retry_attempts": 3                // Retries per provider
}
```

## Provider Optimization

Each AI provider receives optimized prompts:

### Claude
- Structured, detailed context
- Comprehensive error analysis
- Prefers markdown formatting

### OpenAI  
- Balanced context with metadata
- Error type classification
- JSON response format

### Codex
- Code-focused, minimal context
- Truncated verbose information
- Emphasizes stack traces and source code

## Fallback Strategy

1. **Primary Provider**: Attempts fix with Claude (3 retries)
2. **First Fallback**: If Claude fails, tries OpenAI (3 retries)  
3. **Second Fallback**: If OpenAI fails, tries Codex (3 retries)
4. **Best Response**: Returns highest confidence response if no fully valid fix

## Running the Example

```bash
cd examples/multi-provider
go mod tidy
go run main.go
```

## Expected Behavior

1. Healer initializes with multi-provider support
2. Panic handler is installed
3. When a panic occurs:
   - MCP tools gather additional context
   - Primary provider (Claude) attempts fix generation
   - If Claude fails, OpenAI is tried
   - If OpenAI fails, Codex is tried
   - Best available fix is used for PR creation

## MCP Integration

When MCP is enabled, the healer gathers enhanced context:

- **File Structure**: Project layout and organization
- **Dependencies**: Go modules and imports
- **Code Analysis**: AST parsing and symbol resolution
- **Environment**: Go version, build tags, etc.
- **Suggestions**: MCP tool recommendations

This context is included in prompts to all AI providers for better fix quality.