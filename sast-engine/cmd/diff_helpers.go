package cmd

import (
	"os"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/diff"
	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

// githubOptions holds GitHub API context for diff computation and PR comments.
type githubOptions struct {
	Token    string
	Owner    string
	Repo     string
	PRNumber int
}

// parseGitHubRepo splits "owner/repo" into owner and repo components.
// Returns empty strings if the format is invalid.
func parseGitHubRepo(repo string) (string, string) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	return parts[0], parts[1]
}

// resolveBaseRef auto-detects the baseline ref from CI environment variables.
// Used by the ci command when --base is not explicitly provided.
// Returns empty string if no baseline can be detected (full scan).
func resolveBaseRef() string {
	// GitHub Actions.
	if ref := os.Getenv("GITHUB_BASE_REF"); ref != "" {
		return "origin/" + ref
	}
	// GitLab CI.
	if ref := os.Getenv("CI_MERGE_REQUEST_TARGET_BRANCH_NAME"); ref != "" {
		return "origin/" + ref
	}
	// Explicit env var override.
	if ref := os.Getenv("PATHFINDER_BASELINE_REF"); ref != "" {
		return ref
	}
	return "" // No baseline detected → full scan.
}

// computeChangedFiles resolves changed files using the best available provider.
// Shared between ci.go and scan.go to avoid duplication.
func computeChangedFiles(baseRef, headRef, projectRoot string, ghOpts githubOptions, logger *output.Logger) ([]string, error) {
	provider, err := diff.NewChangedFilesProvider(diff.ProviderOptions{
		ProjectRoot: projectRoot,
		BaseRef:     baseRef,
		HeadRef:     headRef,
		GitHubToken: ghOpts.Token,
		Owner:       ghOpts.Owner,
		Repo:        ghOpts.Repo,
		PRNumber:    ghOpts.PRNumber,
	})
	if err != nil {
		return nil, err
	}

	changedFiles, err := provider.GetChangedFiles()
	if err != nil {
		return nil, err
	}

	logger.Progress("Changed files: %d", len(changedFiles))
	return changedFiles, nil
}

// applyDiffFilter filters detections to only those in changed files.
// Returns the filtered detections and the count of detections that were removed.
func applyDiffFilter(allEnriched []*dsl.EnrichedDetection, changedFiles []string, logger *output.Logger) []*dsl.EnrichedDetection {
	totalBefore := len(allEnriched)
	diffFilter := output.NewDiffFilter(changedFiles)
	filtered := diffFilter.Filter(allEnriched)
	logger.Progress("Diff filter: %d/%d findings in changed files", len(filtered), totalBefore)
	return filtered
}
