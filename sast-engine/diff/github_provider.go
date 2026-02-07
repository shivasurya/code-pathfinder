package diff

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

const (
	// githubAPIBaseURL is the base URL for the GitHub REST API.
	githubAPIBaseURL = "https://api.github.com"

	// githubPerPage is the maximum items per page for GitHub API pagination.
	githubPerPage = 100

	// githubTimeout is the HTTP request timeout for GitHub API calls.
	githubTimeout = 30 * time.Second
)

// GitHubAPIDiffProvider gets changed files from the GitHub Pull Request API.
// This is preferred over git-based diff because it handles edge cases better:
// works with shallow clones, immune to merge commit confusion, and returns
// the same file list as GitHub's "Files changed" tab.
type GitHubAPIDiffProvider struct {
	// Token is the GitHub API token for authentication.
	Token string

	// Owner is the GitHub repository owner.
	Owner string

	// Repo is the GitHub repository name.
	Repo string

	// PRNumber is the pull request number.
	PRNumber int

	// BaseURL overrides the GitHub API base URL (for testing).
	BaseURL string
}

// pullRequestFile represents a file in a GitHub pull request API response.
type pullRequestFile struct {
	Filename string `json:"filename"`
	Status   string `json:"status"` // "added", "modified", "removed", "renamed", "copied", "changed", "unchanged".
}

// GetChangedFiles returns relative file paths changed in the pull request.
// It calls the GitHub PR files endpoint with pagination and filters out removed files.
func (p *GitHubAPIDiffProvider) GetChangedFiles() ([]string, error) {
	var allFiles []string
	page := 1

	for {
		files, hasMore, err := p.fetchPage(page)
		if err != nil {
			return nil, err
		}

		for _, f := range files {
			// Exclude removed files — they no longer exist in the PR head.
			if f.Status != "removed" {
				allFiles = append(allFiles, f.Filename)
			}
		}

		if !hasMore {
			break
		}
		page++
	}

	return allFiles, nil
}

// fetchPage fetches a single page of PR files from the GitHub API.
// Returns the files, whether there are more pages, and any error.
func (p *GitHubAPIDiffProvider) fetchPage(page int) ([]pullRequestFile, bool, error) {
	baseURL := p.BaseURL
	if baseURL == "" {
		baseURL = githubAPIBaseURL
	}

	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/files?per_page=%d&page=%d",
		baseURL, p.Owner, p.Repo, p.PRNumber, githubPerPage, page)

	ctx, cancel := context.WithTimeout(context.Background(), githubTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create GitHub API request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.Token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var files []pullRequestFile
	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, false, fmt.Errorf("failed to decode GitHub API response: %w", err)
	}

	// Check for more pages via Link header.
	hasMore := hasNextPage(resp.Header.Get("Link"))

	return files, hasMore, nil
}

// linkNextRe matches the "next" relation in a GitHub Link header.
var linkNextRe = regexp.MustCompile(`<[^>]+>;\s*rel="next"`)

// hasNextPage checks if the Link header indicates more pages.
func hasNextPage(linkHeader string) bool {
	if linkHeader == "" {
		return false
	}
	return linkNextRe.MatchString(linkHeader)
}
