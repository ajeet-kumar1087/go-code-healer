package github

import "fmt"

type GitHubError struct {
	StatusCode int
	Message    string
	URL        string
}

func (e *GitHubError) Error() string {
	return fmt.Sprintf("GitHub API error %d: %s (URL: %s)", e.StatusCode, e.Message, e.URL)
}
