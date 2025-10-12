# go-code-healer

A Go package that automatically catches runtime errors, generates AI-powered fixes using multiple providers (OpenAI, Claude, Codex), and creates pull requests in the background without halting your main application.

## ✨ Features

- **🔍 Automatic Panic Detection**: Captures runtime panics without interfering with your application
- **🤖 Multi-AI Provider Support**: Choose from OpenAI GPT, Claude, or Codex for fix generation
- **🔗 MCP Integration**: Enhanced context gathering using Model Context Protocol tools
- **🚀 Background Processing**: All AI requests and Git operations happen asynchronously
- **📝 Smart PR Creation**: Generates comprehensive pull requests with detailed analysis
- **⚙️ Zero Configuration**: Works out of the box with minimal setup
- **🛡️ Production Ready**: Designed for production environments with graceful error handling

## 🚀 Quick Start

```go
package main

import (
    "github.com/ajeet-kumar1087/go-code-healer"
    "github.com/ajeet-kumar1087/go-code-healer/internal"
)

func main() {
    config := internal.Config{
        AIProvider:   "openai",
        OpenAIAPIKey: "sk-your-api-key",
        GitHubToken:  "ghp_your-token",
        RepoOwner:    "your-username",
        RepoName:     "your-repo",
        Enabled:      true,
    }

    h, err := healer.Initialize(config)
    if err != nil {
        panic(err)
    }

    h.Start()
    defer h.Stop()
    
    h.InstallPanicHandler()
    
    // Your application code here
    // Any panics will be automatically caught and processed
}
```

## 🔧 Advanced Configuration

### Multi-AI Provider Setup

```json
{
  "ai_provider": "openai",
  "openai_api_key": "sk-your-openai-key",
  "claude_api_key": "sk-ant-your-claude-key", 
  "codex_api_key": "sk-your-codex-key",
  "github_token": "ghp_your-github-token",
  "repo_owner": "your-username",
  "repo_name": "your-repository"
}
```

### MCP Integration for Enhanced Context

```json
{
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
      "auth_token": "your-mcp-token",
      "tools": ["parse_ast", "analyze_symbols"]
    }
  ]
}
```

## 🎯 How It Works

1. **Panic Capture**: The healer installs a global panic handler that captures runtime errors
2. **Context Gathering**: Collects error details, stack traces, and optionally queries MCP tools for enhanced context
3. **AI Processing**: Sends context to your configured AI provider for fix generation
4. **Code Validation**: Validates the proposed fix for syntax correctness
5. **PR Creation**: Creates a new branch, applies the fix, and opens a pull request
6. **Background Execution**: All processing happens in background goroutines without blocking your app

## 🔗 MCP (Model Context Protocol) Integration

The healer can integrate with MCP tools to gather enhanced context for more accurate fixes:

- **Filesystem Analysis**: Project structure, dependencies, file relationships
- **Code Analysis**: AST parsing, symbol resolution, pattern detection  
- **Environment Context**: Go version, build tags, environment variables
- **Best Practices**: Codebase-specific patterns and conventions

This results in AI-generated fixes that are more accurate and consistent with your codebase.

## 📁 Examples

- [Basic Usage](examples/basic/) - Simple integration example
- [Web Application](examples/webapp/) - Integration with HTTP server
- [Microservice](examples/microservice/) - Production-ready microservice setup
- [MCP Integration](examples/mcp-integration/) - Enhanced context gathering with MCP tools

## 🛠️ Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `ai_provider` | AI provider to use (openai, claude, codex) | `openai` |
| `mcp_enabled` | Enable MCP integration for enhanced context | `false` |
| `enabled` | Enable/disable the healer | `true` |
| `max_queue_size` | Maximum number of queued errors | `100` |
| `worker_count` | Number of background workers | `2` |
| `retry_attempts` | Number of retry attempts for failed operations | `3` |
| `log_level` | Logging level (debug, info, warn, error) | `info` |

## 🔒 Security & Privacy

- API keys are handled securely and never logged
- Error context is sanitized before sending to AI providers
- MCP communication uses configurable authentication
- Minimal GitHub permissions required (repo access only)
- All operations respect rate limits and timeouts

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
