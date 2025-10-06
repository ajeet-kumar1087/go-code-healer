package github

import "fmt"

// validatePRRequest validates the pull request request
func (gc *GitHubAPIClient) validatePRRequest(request PRRequest) error {
	if request.BranchName == "" {
		return fmt.Errorf("branch name is required")
	}
	if request.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(request.Changes) == 0 {
		return fmt.Errorf("at least one file change is required")
	}
	for i, change := range request.Changes {
		if change.FilePath == "" {
			return fmt.Errorf("file path is required for change %d", i)
		}
		if change.Content == "" {
			return fmt.Errorf("content is required for change %d", i)
		}
	}
	return nil
}
