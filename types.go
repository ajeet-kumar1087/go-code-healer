package healer

import (
	"context"

	"github.com/ajeet-kumar1087/go-code-healer/ai"
	"github.com/ajeet-kumar1087/go-code-healer/github"
	"github.com/ajeet-kumar1087/go-code-healer/internal"
)

// Re-export types for backward compatibility

// Panic types are now defined directly in panic.go

// AI client types (directly from ai module)
type AIClient = ai.Client
type FixRequest = ai.FixRequest
type FixResponse = ai.FixResponse

// Git client types (directly from github module)
type PRRequest = github.PRRequest
type FileChange = github.FileChange

// GitClient interface for Git operations and GitHub API calls
type GitClient interface {
	CreatePullRequest(ctx context.Context, request PRRequest) error
}

// Worker interface for background processing
type Worker interface {
	Start(ctx context.Context) error
	Stop() error
}

// Logger types (from types module)
type LogLevel = internal.LogLevel
type Logger = internal.LoggerInterface
type LoggerInterface = internal.LoggerInterface
type DefaultLogger = internal.DefaultLogger

// Re-export constants
const (
	LogLevelDebug = internal.LogLevelDebug
	LogLevelInfo  = internal.LogLevelInfo
	LogLevelWarn  = internal.LogLevelWarn
	LogLevelError = internal.LogLevelError
)

// Re-export functions
var (
	// Panic functions are now defined directly in panic.go
	NewDefaultLogger = internal.NewDefaultLogger
)
