package cmd

import (
	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

// applyDiffFilter intersects detections with the set of files changed in
// the diff when diff-aware mode is enabled. When diffEnabled is false it
// returns the input unchanged so callers can branch on the bool to decide
// whether to log "Diff filter: X/Y..."
//
// Critically, applyDiffFilter does NOT short-circuit when changedFiles is
// empty: an empty list means the diff genuinely covers no source files
// (delete-only PR, docs-only PR, empty PR), and the right answer is zero
// detections, not "fall back to a full scan." Falling back was the May
// 2026 regression that surfaced as 207 findings on a PR that only
// deleted a YAML workflow file (the public post-mortem in CLAUDE.md).
func applyDiffFilter(
	detections []*dsl.EnrichedDetection,
	changedFiles []string,
	diffEnabled bool,
) (filtered []*dsl.EnrichedDetection, applied bool) {
	if !diffEnabled {
		return detections, false
	}
	return output.NewDiffFilter(changedFiles).Filter(detections), true
}

// countScannedFiles picks the file-count number to report in scan output
// based on whether diff-aware mode was active. When diff-aware, the
// reported count is the number of files in the diff (which may be 0 by
// design; see applyDiffFilter for why we no longer fall back). When not
// diff-aware, the count is the number of unique files the file walk
// observed and the scanner actually parsed.
func countScannedFiles(diffEnabled bool, changedFilesCount, fallbackUniqueFilesCount int) int {
	if diffEnabled {
		return changedFilesCount
	}
	return fallbackUniqueFilesCount
}
