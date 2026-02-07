package diff

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GitDiffProvider computes changed files using git commands.
// It uses git merge-base to find the fork point, then git diff to list changed files.
// This correctly handles merge commits by diffing from the true fork point.
type GitDiffProvider struct {
	// ProjectRoot is the absolute path to the git repository root.
	ProjectRoot string

	// BaseRef is the baseline git ref (e.g., "origin/main", "abc123", "HEAD~1").
	BaseRef string

	// HeadRef is the head git ref (defaults to "HEAD").
	HeadRef string
}

// GetChangedFiles returns relative file paths changed between base and head.
// It uses git merge-base to find the fork point, avoiding merge commit confusion:
//
//	main:    A --- B --- C
//	              \
//	feature:       D --- E (HEAD)
//
// merge-base returns B, so diff B..HEAD returns only D and E's changes.
func (p *GitDiffProvider) GetChangedFiles() ([]string, error) {
	mergeBase, err := p.findMergeBase()
	if err != nil {
		return nil, fmt.Errorf("failed to find merge-base between %s and %s: %w", p.BaseRef, p.HeadRef, err)
	}

	return p.diffFiles(mergeBase)
}

// findMergeBase runs git merge-base to find the common ancestor.
func (p *GitDiffProvider) findMergeBase() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "merge-base", p.BaseRef, p.HeadRef)
	cmd.Dir = p.ProjectRoot

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("git merge-base timed out after 30s")
		}
		return "", fmt.Errorf("git merge-base failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// diffFiles runs git diff --name-only to list changed files from merge-base to head.
// Uses --diff-filter=ACMR to include Added, Copied, Modified, and Renamed files only.
func (p *GitDiffProvider) diffFiles(mergeBase string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	diffRange := mergeBase + ".." + p.HeadRef
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", "--diff-filter=ACMR", diffRange)
	cmd.Dir = p.ProjectRoot

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("git diff timed out after 30s")
		}
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	return parseFileList(string(output)), nil
}

// parseFileList splits newline-separated file paths, filtering empty lines.
func parseFileList(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var files []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			files = append(files, trimmed)
		}
	}
	return files
}
