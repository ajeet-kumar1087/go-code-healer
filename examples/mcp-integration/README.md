# MCP Integration Example

This example demonstrates how to use the Go Code Healer with Model Context Protocol (MCP) integration for enhanced context gathering and more accurate AI-generated fixes.

## Features Demonstrated

- **Multi-AI Provider Support**: Configure different AI providers (OpenAI, Claude, Codex)
- **MCP Context Gathering**: Collect additional context from MCP tools
- **Enhanced Fix Generation**: Generate more accurate fixes using comprehensive context
- **Automatic PR Creation**: Create pull requests with detailed analysis

## MCP Tools Integration

The healer can integrate with various MCP tools to gather enhanced context:

### Filesystem Analyzer
- Analyzes project structure and file relationships
- Identifies dependencies and imports
- Provides insights into code organization

### Code Analyzer  
- Performs AST parsing and symbol resolution
- Analyzes code patterns and potential issues
- Suggests best practices and improvements

### Environment Context
- Gathers Go version and build information
- Collects environment variables and build tags
- Provides runtime context information

## Configuration

The example uses a comprehensive configuration that includes:

```json
{
  "ai_provider": "openai",
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
  ]
}
```

## Running the Example

1. **Set up MCP servers** (optional - the healer will work without them):
   ```bash
   # Start your MCP servers on the configured endpoints
   # This is just an example - actual MCP servers would be separate services
   ```

2. **Configure your API keys**:
   ```bash
   export HEALER_OPENAI_API_KEY="sk-your-key-here"
   export HEALER_GITHUB_TOKEN="ghp_your-token-here"
   export HEALER_REPO_OWNER="your-username"
   export HEALER_REPO_NAME="your-repo"
   ```

3. **Run the example**:
   ```bash
   go run main.go
   ```

## What Happens

1. The application starts with MCP integration enabled
2. Simulated panics occur (nil pointer, index out of bounds)
3. The healer captures each panic and:
   - Gathers basic error context (stack trace, source code)
   - Queries configured MCP tools for additional context
   - Sends enhanced context to the AI provider
   - Generates more accurate fixes using comprehensive information
   - Creates pull requests with detailed analysis

## Expected Output

The healer will create pull requests that include:

- **Enhanced Error Analysis**: Detailed breakdown using MCP context
- **Comprehensive Fixes**: Solutions informed by project structure and patterns
- **Related File Analysis**: Understanding of how the fix affects other parts of the codebase
- **Best Practice Recommendations**: Suggestions based on codebase analysis

## MCP Context Examples

### For Nil Pointer Panic:
- Project structure showing related nil-safe patterns
- Dependencies that provide nil-checking utilities
- Environment info about Go version and build constraints
- Code analysis suggesting defensive programming patterns

### For Index Out of Bounds:
- Similar functions in the codebase that handle bounds checking
- Slice usage patterns and best practices
- Related utility functions for safe array access
- Suggested refactoring to prevent similar issues

## Benefits of MCP Integration

1. **More Accurate Fixes**: AI has comprehensive context about your codebase
2. **Consistent Patterns**: Fixes follow existing code patterns and conventions
3. **Comprehensive Analysis**: Understanding of how fixes affect the broader system
4. **Better Documentation**: Pull requests include detailed context and reasoning
5. **Proactive Suggestions**: Recommendations for preventing similar issues

## Fallback Behavior

If MCP tools are unavailable or fail:
- The healer gracefully falls back to basic error context
- Fix generation continues without MCP enhancement
- No impact on application performance or reliability
- Warnings are logged for debugging MCP connectivity issues