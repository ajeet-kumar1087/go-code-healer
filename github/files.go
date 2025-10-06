package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// updateFile updates or creates a file in the repository
func (gc *GitHubAPIClient) updateFile(ctx context.Context, branchName string, change FileChange) error {
	// First, try to get the current file to get its SHA (needed for updates)
	currentSHA, err := gc.getFileSHA(ctx, change.FilePath, branchName)
	if err != nil {
		gc.logger.Debug("File %s not found, will create new file", change.FilePath)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", gc.baseURL, gc.repoOwner, gc.repoName, change.FilePath)

	// Create commit message
	commitMessage := fmt.Sprintf("Fix panic in %s\n\nAutomatically generated fix for runtime panic", change.FilePath)

	payload := map[string]any{
		"message": commitMessage,
		"content": gc.encodeBase64(change.Content),
		"branch":  branchName,
	}

	// If file exists, include SHA for update
	if currentSHA != "" {
		payload["sha"] = currentSHA
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
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

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error updating file: %d - %s", resp.StatusCode, string(body))
	}

	gc.logger.Debug("Updated file: %s", change.FilePath)
	return nil
}

// getFileSHA gets the SHA of a file (needed for updates)
func (gc *GitHubAPIClient) getFileSHA(ctx context.Context, filePath, branchName string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", gc.baseURL, gc.repoOwner, gc.repoName, filePath, branchName)

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

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("file not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error: %d - %s", resp.StatusCode, string(body))
	}

	var file struct {
		SHA string `json:"sha"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
		return "", err
	}

	return file.SHA, nil
}

// encodeBase64 encodes content to base64 for GitHub API
func (gc *GitHubAPIClient) encodeBase64(content string) string {
	return base64.StdEncoding.EncodeToString([]byte(content))
}
