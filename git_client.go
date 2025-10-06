package healer

import (
	"context"

	gh "github.com/ajeet-kumar1087/go-code-healer/github"
)

// GitHubAPIClient wraps the github module client to implement GitClient interface
type GitHubAPIClient struct {
	client *gh.GitHubAPIClient
}

// NewGitHubClient creates a new GitHub API client using the github module
func NewGitHubClient(token, repoOwner, repoName string, logger Logger) *GitHubAPIClient {
	return &GitHubAPIClient{
		client: gh.NewGitHubClient(token, repoOwner, repoName, logger),
	}
}

// CreatePullRequest creates a new branch, commits changes, and opens a PR
func (gc *GitHubAPIClient) CreatePullRequest(ctx context.Context, request PRRequest) error {
	// Convert healer types to github module types
	githubRequest := gh.PRRequest{
		BranchName:  request.BranchName,
		Title:       request.Title,
		Description: request.Description,
		Changes:     make([]gh.FileChange, len(request.Changes)),
	}

	for i, change := range request.Changes {
		githubRequest.Changes[i] = gh.FileChange{
			FilePath: change.FilePath,
			Content:  change.Content,
		}
	}

	// Delegate to the github module
	return gc.client.CreatePullRequest(ctx, githubRequest)
}

// GenerateBranchName creates a descriptive branch name for the panic fix
func GenerateBranchName(panicEvent PanicEvent) string {
	// Convert healer PanicEvent to github PanicEvent
	githubEvent := gh.PanicEvent{
		ID:         panicEvent.ID,
		Timestamp:  panicEvent.Timestamp,
		Error:      panicEvent.Error,
		StackTrace: panicEvent.StackTrace,
		SourceFile: panicEvent.SourceFile,
		LineNumber: panicEvent.LineNumber,
		Function:   panicEvent.Function,
		Status:     panicEvent.Status,
	}
	if panicEvent.ProcessedAt != nil {
		githubEvent.ProcessedAt = panicEvent.ProcessedAt
	}

	return gh.GenerateBranchName(githubEvent)
}

// GeneratePRTitle creates a descriptive title for the pull request
func GeneratePRTitle(panicEvent PanicEvent) string {
	// Convert healer PanicEvent to github PanicEvent
	githubEvent := gh.PanicEvent{
		ID:         panicEvent.ID,
		Timestamp:  panicEvent.Timestamp,
		Error:      panicEvent.Error,
		StackTrace: panicEvent.StackTrace,
		SourceFile: panicEvent.SourceFile,
		LineNumber: panicEvent.LineNumber,
		Function:   panicEvent.Function,
		Status:     panicEvent.Status,
	}
	if panicEvent.ProcessedAt != nil {
		githubEvent.ProcessedAt = panicEvent.ProcessedAt
	}

	return gh.GeneratePRTitle(githubEvent)
}

// GeneratePRDescription creates a comprehensive description for the pull request
func GeneratePRDescription(panicEvent PanicEvent, fixResponse *FixResponse) string {
	// Convert healer types to github types
	githubEvent := gh.PanicEvent{
		ID:         panicEvent.ID,
		Timestamp:  panicEvent.Timestamp,
		Error:      panicEvent.Error,
		StackTrace: panicEvent.StackTrace,
		SourceFile: panicEvent.SourceFile,
		LineNumber: panicEvent.LineNumber,
		Function:   panicEvent.Function,
		Status:     panicEvent.Status,
	}
	if panicEvent.ProcessedAt != nil {
		githubEvent.ProcessedAt = panicEvent.ProcessedAt
	}

	var githubFixResponse *gh.FixResponse
	if fixResponse != nil {
		githubFixResponse = &gh.FixResponse{
			ProposedFix: fixResponse.ProposedFix,
			Explanation: fixResponse.Explanation,
			Confidence:  fixResponse.Confidence,
			IsValid:     fixResponse.IsValid,
		}
	}

	return gh.GeneratePRDescription(githubEvent, githubFixResponse)
}
