package github

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
)

// PRCommentOptions holds configuration for PR commenting.
type PRCommentOptions struct {
	PRNumber int
	Comment  bool // Post summary comment.
	Inline   bool // Post inline review comments.
}

// Enabled returns true if any PR commenting feature is requested.
func (o *PRCommentOptions) Enabled() bool {
	return o.Comment || o.Inline
}

// Validate checks that required fields are present when commenting is enabled.
func (o *PRCommentOptions) Validate() error {
	if !o.Enabled() {
		return nil
	}
	if o.PRNumber <= 0 {
		return fmt.Errorf("--github-pr must be a positive number")
	}
	return nil
}

// ParseRepo splits "owner/repo" into owner and repo.
func ParseRepo(repo string) (string, string, error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("--github-repo must be in owner/repo format, got %q", repo)
	}
	return parts[0], parts[1], nil
}

// ProgressFunc is a callback for reporting progress messages.
type ProgressFunc func(format string, args ...any)

// PostPRComments posts summary and/or inline comments on a GitHub PR.
func PostPRComments(
	client *Client,
	opts PRCommentOptions,
	findings []*dsl.EnrichedDetection,
	metrics ScanMetrics,
	progress ProgressFunc,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Fetch PR metadata once for blob links and inline comments.
	pr, err := client.GetPullRequest(ctx, opts.PRNumber)
	if err != nil {
		return fmt.Errorf("get PR metadata: %w", err)
	}
	metrics.BlobBaseURL = fmt.Sprintf("https://github.com/%s/%s/blob/%s", client.owner, client.repo, pr.Head.SHA)

	// Post summary comment.
	if opts.Comment {
		progress("Posting PR summary comment...")
		markdown := FormatSummaryComment(findings, metrics)
		cm := NewCommentManager(client, opts.PRNumber)
		if err := cm.PostOrUpdate(ctx, markdown); err != nil {
			return fmt.Errorf("post summary comment: %w", err)
		}
		progress("PR summary comment posted")
	}

	// Post inline review comments for critical/high findings.
	if opts.Inline {
		progress("Posting inline review comments...")
		rm := NewReviewManager(client, opts.PRNumber, pr.Head.SHA)
		if err := rm.PostInlineComments(ctx, findings); err != nil {
			return fmt.Errorf("post inline comments: %w", err)
		}
		progress("Inline review comments posted")
	}

	return nil
}
