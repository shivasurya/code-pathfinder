package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/github"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

// prCommentOptions holds the flags needed for PR commenting.
type prCommentOptions struct {
	Token    string
	Repo     string // "owner/repo" format
	PRNumber int
	Comment  bool // Post summary comment.
	Inline   bool // Post inline review comments.
}

// enabled returns true if any PR commenting feature is requested.
func (o *prCommentOptions) enabled() bool {
	return o.Comment || o.Inline
}

// validate checks that required fields are present when commenting is enabled.
func (o *prCommentOptions) validate() error {
	if !o.enabled() {
		return nil
	}
	if o.Token == "" {
		return fmt.Errorf("--github-token is required for PR commenting")
	}
	if o.Repo == "" {
		return fmt.Errorf("--github-repo is required for PR commenting")
	}
	if o.PRNumber <= 0 {
		return fmt.Errorf("--github-pr must be a positive number")
	}
	if _, _, err := parseGitHubRepo(o.Repo); err != nil {
		return err
	}
	return nil
}

// parseGitHubRepo splits "owner/repo" into owner and repo.
func parseGitHubRepo(repo string) (string, string, error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("--github-repo must be in owner/repo format, got %q", repo)
	}
	return parts[0], parts[1], nil
}

// newGitHubClient creates a GitHub API client. Variable to allow testing with mock server.
var newGitHubClient = github.NewClient

// postPRComments posts summary and/or inline comments on a GitHub PR.
func postPRComments(
	opts prCommentOptions,
	findings []*dsl.EnrichedDetection,
	metrics github.ScanMetrics,
	logger *output.Logger,
) error {
	owner, repo, _ := parseGitHubRepo(opts.Repo) // Already validated.
	client := newGitHubClient(opts.Token, owner, repo)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Post summary comment.
	if opts.Comment {
		logger.Progress("Posting PR summary comment...")
		markdown := github.FormatSummaryComment(findings, metrics)
		cm := github.NewCommentManager(client, opts.PRNumber)
		if err := cm.PostOrUpdate(ctx, markdown); err != nil {
			return fmt.Errorf("post summary comment: %w", err)
		}
		logger.Progress("PR summary comment posted")
	}

	// Post inline review comments for critical/high findings.
	if opts.Inline {
		logger.Progress("Posting inline review comments...")
		pr, err := client.GetPullRequest(ctx, opts.PRNumber)
		if err != nil {
			return fmt.Errorf("get PR metadata: %w", err)
		}
		rm := github.NewReviewManager(client, opts.PRNumber, pr.Head.SHA)
		if err := rm.PostInlineComments(ctx, findings); err != nil {
			return fmt.Errorf("post inline comments: %w", err)
		}
		logger.Progress("Inline review comments posted")
	}

	return nil
}
