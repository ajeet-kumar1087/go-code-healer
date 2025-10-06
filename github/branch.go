package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// getDefaultBranch retrieves the default branch name for the repository
func (gc *GitHubAPIClient) getDefaultBranch(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", gc.baseURL, gc.repoOwner, gc.repoName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "token "+gc.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var repo struct {
		DefaultBranch string `json:"default_branch"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return "", err
	}

	return repo.DefaultBranch, nil
}

// getBranchSHA gets the SHA of a specific branch
func (gc *GitHubAPIClient) getBranchSHA(ctx context.Context, branchName string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/git/refs/heads/%s", gc.baseURL, gc.repoOwner, gc.repoName, branchName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "token "+gc.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var ref struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ref); err != nil {
		return "", err
	}

	return ref.Object.SHA, nil
}

// createBranch creates a new branch from the base SHA
func (gc *GitHubAPIClient) createBranch(ctx context.Context, branchName, baseSHA string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/git/refs", gc.baseURL, gc.repoOwner, gc.repoName)

	payload := map[string]string{
		"ref": "refs/heads/" + branchName,
		"sha": baseSHA,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+gc.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := gc.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error creating branch: %d - %s", resp.StatusCode, string(body))
	}

	gc.logger.Debug("Created branch: %s", branchName)
	return nil
}
