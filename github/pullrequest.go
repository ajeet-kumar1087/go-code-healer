package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CreatePullRequest creates a new branch, commits changes, and opens a PR
func (gc *GitHubAPIClient) CreatePullRequest(ctx context.Context, request PRRequest) error {
	gc.logger.Info("Creating pull request: %s", request.Title)

	// Validate request
	if err := gc.validatePRRequest(request); err != nil {
		return fmt.Errorf("invalid PR request: %w", err)
	}

	// Step 1: Get the default branch SHA
	defaultBranch, err := gc.getDefaultBranch(ctx)
	if err != nil {
		gc.logger.Error("Failed to get default branch: %v", err)
		return fmt.Errorf("failed to get default branch: %w", err)
	}
	gc.logger.Debug("Default branch: %s", defaultBranch)

	baseSHA, err := gc.getBranchSHA(ctx, defaultBranch)
	if err != nil {
		gc.logger.Error("Failed to get base branch SHA: %v", err)
		return fmt.Errorf("failed to get base branch SHA: %w", err)
	}
	gc.logger.Debug("Base SHA: %s", baseSHA)

	// Step 2: Create a new branch
	if err := gc.createBranch(ctx, request.BranchName, baseSHA); err != nil {
		gc.logger.Error("Failed to create branch %s: %v", request.BranchName, err)
		return fmt.Errorf("failed to create branch: %w", err)
	}

	// Step 3: Apply file changes
	for i, change := range request.Changes {
		gc.logger.Debug("Applying change %d/%d: %s", i+1, len(request.Changes), change.FilePath)
		if err := gc.updateFile(ctx, request.BranchName, change); err != nil {
			gc.logger.Error("Failed to update file %s: %v", change.FilePath, err)
			return fmt.Errorf("failed to update file %s: %w", change.FilePath, err)
		}
	}

	// Step 4: Create the pull request
	prResult, err := gc.createPR(ctx, request, defaultBranch)
	if err != nil {
		gc.logger.Error("Failed to create pull request: %v", err)
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	gc.logger.Info("Successfully created pull request #%d: %s", prResult.Number, prResult.URL)
	return nil
}

// createPR creates the actual pull request
func (gc *GitHubAPIClient) createPR(ctx context.Context, request PRRequest, baseBranch string) (*PRResult, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls", gc.baseURL, gc.repoOwner, gc.repoName)

	payload := map[string]string{
		"title": request.Title,
		"head":  request.BranchName,
		"base":  baseBranch,
		"body":  request.Description,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+gc.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, &GitHubError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
			URL:        url,
		}
	}

	var prResponse struct {
		Number  int    `json:"number"`
		HTMLURL string `json:"html_url"`
		Title   string `json:"title"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&prResponse); err != nil {
		return nil, fmt.Errorf("failed to decode PR response: %w", err)
	}

	result := &PRResult{
		URL:    prResponse.HTMLURL,
		Number: prResponse.Number,
		Title:  prResponse.Title,
	}

	gc.logger.Debug("Created pull request: %s", request.Title)
	return result, nil
}
