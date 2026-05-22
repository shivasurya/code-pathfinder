package output

import "github.com/shivasurya/code-pathfinder/sast-engine/dsl"

// DiffFilter filters detections to only include findings in changed files.
// Used for diff-aware scanning where full codebase is scanned but output
// is limited to files changed in the PR/commit.
type DiffFilter struct {
	changedFiles map[string]bool // Set of relative file paths.
}

// NewDiffFilter creates a filter from a list of changed file paths.
// Paths should be relative to the project root (matching LocationInfo.RelPath).
func NewDiffFilter(changedFiles []string) *DiffFilter {
	fileSet := make(map[string]bool, len(changedFiles))
	for _, f := range changedFiles {
		fileSet[f] = true
	}
	return &DiffFilter{changedFiles: fileSet}
}

// Filter returns only detections whose RelPath is in the changed files set.
// An empty set filters everything out: a DiffFilter constructed with no
// changed files represents "nothing in the diff to match," and the right
// answer is zero detections, not pass-through. Callers that want "no
// filtering" must not call Filter at all (see cmd.applyDiffFilter which
// gates on the diffEnabled flag for exactly this reason). Falling back to
// pass-through was the May 2026 207-findings regression.
func (f *DiffFilter) Filter(detections []*dsl.EnrichedDetection) []*dsl.EnrichedDetection {
	filtered := make([]*dsl.EnrichedDetection, 0, len(detections))
	for _, det := range detections {
		if f.changedFiles[det.Location.RelPath] {
			filtered = append(filtered, det)
		}
	}
	return filtered
}

// FilteredCount returns the number of detections that would be removed.
// With an empty changed-files set, every detection would be removed, so
// the count equals len(detections). See Filter for the rationale.
func (f *DiffFilter) FilteredCount(detections []*dsl.EnrichedDetection) int {
	count := 0
	for _, det := range detections {
		if !f.changedFiles[det.Location.RelPath] {
			count++
		}
	}
	return count
}

// ChangedFileCount returns the number of changed files in the filter set.
func (f *DiffFilter) ChangedFileCount() int {
	return len(f.changedFiles)
}
