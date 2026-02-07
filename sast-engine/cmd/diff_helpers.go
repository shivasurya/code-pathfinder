package cmd

import (
	"os"

	"github.com/shivasurya/code-pathfinder/sast-engine/diff"
	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

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

// computeChangedFiles resolves changed files using git diff.
// Shared between ci.go and scan.go to avoid duplication.
func computeChangedFiles(baseRef, headRef, projectRoot string, logger *output.Logger) ([]string, error) {
	provider, err := diff.NewChangedFilesProvider(diff.ProviderOptions{
		ProjectRoot: projectRoot,
		BaseRef:     baseRef,
		HeadRef:     headRef,
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
// Returns the filtered detections.
func applyDiffFilter(allEnriched []*dsl.EnrichedDetection, changedFiles []string, logger *output.Logger) []*dsl.EnrichedDetection {
	totalBefore := len(allEnriched)
	diffFilter := output.NewDiffFilter(changedFiles)
	filtered := diffFilter.Filter(allEnriched)
	logger.Progress("Diff filter: %d/%d findings in changed files", len(filtered), totalBefore)
	return filtered
}
