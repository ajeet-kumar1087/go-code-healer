package github

import (
	"net/http"
	"time"

	"github.com/ajeet-kumar1087/go-code-healer/internal"
)

type Logger = internal.LoggerInterface

type GitHubAPIClient struct {
	token      string
	repoOwner  string
	repoName   string
	httpClient *http.Client
	logger     Logger
	baseURL    string
}

func NewGitHubClient(token, owner, repo string, logger Logger) *GitHubAPIClient {
	return &GitHubAPIClient{
		token:     token,
		repoOwner: owner,
		repoName:  repo,
		logger:    logger,
		baseURL:   "https://api.github.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}
