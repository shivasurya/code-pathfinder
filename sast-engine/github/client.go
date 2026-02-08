package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client wraps GitHub REST API interactions for PR commenting.
type Client struct {
	token      string
	owner      string
	repo       string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a GitHub API client.
func NewClient(token, owner, repo string) *Client {
	return &Client{
		token:   token,
		owner:   owner,
		repo:    repo,
		baseURL: "https://api.github.com",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL overrides the API base URL (used for testing).
func (c *Client) SetBaseURL(url string) {
	c.baseURL = url
}

// GetPullRequest retrieves PR metadata (head SHA, base branch, etc.).
func (c *Client) GetPullRequest(ctx context.Context, prNumber int) (*PullRequest, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", c.owner, c.repo, prNumber)
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get pull request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return nil, fmt.Errorf("get pull request: %w", err)
	}

	var pr PullRequest
	if err := decodeResponse(resp, &pr); err != nil {
		return nil, fmt.Errorf("get pull request: %w", err)
	}
	return &pr, nil
}

// ListComments lists all issue comments on a PR.
// GitHub treats PR comments as issue comments under /issues/{number}/comments.
func (c *Client) ListComments(ctx context.Context, prNumber int) ([]*Comment, error) {
	var all []*Comment
	page := 1
	for {
		path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments?per_page=100&page=%d",
			c.owner, c.repo, prNumber, page)
		resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, fmt.Errorf("list comments: %w", err)
		}
		defer resp.Body.Close()

		if err := checkResponse(resp); err != nil {
			return nil, fmt.Errorf("list comments: %w", err)
		}

		var comments []*Comment
		if err := decodeResponse(resp, &comments); err != nil {
			return nil, fmt.Errorf("list comments: %w", err)
		}
		all = append(all, comments...)

		if len(comments) < 100 {
			break
		}
		page++
	}
	return all, nil
}

// CreateComment creates a new issue comment on a PR.
func (c *Client) CreateComment(ctx context.Context, prNumber int, body string) (*Comment, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", c.owner, c.repo, prNumber)
	payload := createCommentRequest{Body: body}

	resp, err := c.doRequest(ctx, http.MethodPost, path, payload)
	if err != nil {
		return nil, fmt.Errorf("create comment: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return nil, fmt.Errorf("create comment: %w", err)
	}

	var comment Comment
	if err := decodeResponse(resp, &comment); err != nil {
		return nil, fmt.Errorf("create comment: %w", err)
	}
	return &comment, nil
}

// UpdateComment updates an existing issue comment.
func (c *Client) UpdateComment(ctx context.Context, commentID int64, body string) (*Comment, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/comments/%d", c.owner, c.repo, commentID)
	payload := updateCommentRequest{Body: body}

	resp, err := c.doRequest(ctx, http.MethodPatch, path, payload)
	if err != nil {
		return nil, fmt.Errorf("update comment: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return nil, fmt.Errorf("update comment: %w", err)
	}

	var comment Comment
	if err := decodeResponse(resp, &comment); err != nil {
		return nil, fmt.Errorf("update comment: %w", err)
	}
	return &comment, nil
}

// UpdateReviewComment updates an existing inline review comment.
// Uses the pulls/comments endpoint (different from issue comments).
func (c *Client) UpdateReviewComment(ctx context.Context, commentID int64, body string) (*ReviewComment, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/comments/%d", c.owner, c.repo, commentID)
	payload := updateCommentRequest{Body: body}

	resp, err := c.doRequest(ctx, http.MethodPatch, path, payload)
	if err != nil {
		return nil, fmt.Errorf("update review comment: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return nil, fmt.Errorf("update review comment: %w", err)
	}

	var comment ReviewComment
	if err := decodeResponse(resp, &comment); err != nil {
		return nil, fmt.Errorf("update review comment: %w", err)
	}
	return &comment, nil
}

// CreateReview creates a review with inline comments (posted atomically).
func (c *Client) CreateReview(ctx context.Context, prNumber int, commitID string, body string, comments []ReviewCommentInput) error {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", c.owner, c.repo, prNumber)
	payload := createReviewRequest{
		CommitID: commitID,
		Body:     body,
		Event:    "COMMENT",
		Comments: comments,
	}

	resp, err := c.doRequest(ctx, http.MethodPost, path, payload)
	if err != nil {
		return fmt.Errorf("create review: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return fmt.Errorf("create review: %w", err)
	}
	return nil
}

// ListReviewComments lists all inline review comments on a PR.
func (c *Client) ListReviewComments(ctx context.Context, prNumber int) ([]*ReviewComment, error) {
	var all []*ReviewComment
	page := 1
	for {
		path := fmt.Sprintf("/repos/%s/%s/pulls/%d/comments?per_page=100&page=%d",
			c.owner, c.repo, prNumber, page)
		resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, fmt.Errorf("list review comments: %w", err)
		}
		defer resp.Body.Close()

		if err := checkResponse(resp); err != nil {
			return nil, fmt.Errorf("list review comments: %w", err)
		}

		var comments []*ReviewComment
		if err := decodeResponse(resp, &comments); err != nil {
			return nil, fmt.Errorf("list review comments: %w", err)
		}
		all = append(all, comments...)

		if len(comments) < 100 {
			break
		}
		page++
	}
	return all, nil
}

// DeleteReviewComment deletes an inline review comment.
func (c *Client) DeleteReviewComment(ctx context.Context, commentID int64) error {
	path := fmt.Sprintf("/repos/%s/%s/pulls/comments/%d", c.owner, c.repo, commentID)

	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("delete review comment: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return fmt.Errorf("delete review comment: %w", err)
	}
	return nil
}

// doRequest executes an HTTP request with auth headers.
func (c *Client) doRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// checkResponse returns an error for non-2xx status codes.
func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	var apiErr apiError
	if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, apiErr.Message)
}

// decodeResponse decodes a JSON response body into dest.
func decodeResponse(resp *http.Response, dest any) error {
	return json.NewDecoder(resp.Body).Decode(dest)
}
