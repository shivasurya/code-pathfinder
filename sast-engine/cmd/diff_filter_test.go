package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
)

// detection builds a minimal EnrichedDetection whose RelPath matches the
// given path. Sufficient for DiffFilter, which keys off RelPath / FilePath.
func detection(rel string) *dsl.EnrichedDetection {
	return &dsl.EnrichedDetection{
		Location: dsl.LocationInfo{RelPath: rel, FilePath: rel},
	}
}

// --- applyDiffFilter ---

func TestApplyDiffFilter_DisabledReturnsInputUnchanged(t *testing.T) {
	in := []*dsl.EnrichedDetection{detection("a.py"), detection("b.py")}
	out, applied := applyDiffFilter(in, nil, false)
	assert.False(t, applied, "applied flag must be false when diff is disabled")
	assert.Equal(t, in, out, "input must be returned unchanged when diff is disabled")
}

func TestApplyDiffFilter_DisabledIgnoresChangedFiles(t *testing.T) {
	// changedFiles is non-nil but diffEnabled is false: the filter still must
	// not run. Guards against accidental "always apply if list provided"
	// regressions.
	in := []*dsl.EnrichedDetection{detection("a.py")}
	out, applied := applyDiffFilter(in, []string{"unrelated.py"}, false)
	assert.False(t, applied)
	assert.Equal(t, in, out)
}

func TestApplyDiffFilter_EnabledWithMatchingFile(t *testing.T) {
	in := []*dsl.EnrichedDetection{detection("a.py"), detection("b.py")}
	out, applied := applyDiffFilter(in, []string{"a.py"}, true)
	assert.True(t, applied, "applied flag must be true when diff is enabled")
	assert.Len(t, out, 1, "only a.py finding should survive")
	assert.Equal(t, "a.py", out[0].Location.RelPath)
}

func TestApplyDiffFilter_EnabledWithNoMatches(t *testing.T) {
	in := []*dsl.EnrichedDetection{detection("a.py"), detection("b.py")}
	out, applied := applyDiffFilter(in, []string{"unrelated.py"}, true)
	assert.True(t, applied)
	assert.Empty(t, out)
}

func TestApplyDiffFilter_EnabledWithEmptyChangedFiles(t *testing.T) {
	// The regression case. Empty changedFiles + diffEnabled=true must produce
	// zero detections, NOT a full pass-through. Falling back to a full scan
	// here was the May 2026 207-findings bug.
	in := []*dsl.EnrichedDetection{detection("a.py"), detection("b.py")}
	out, applied := applyDiffFilter(in, []string{}, true)
	assert.True(t, applied, "applied flag must remain true even with empty changedFiles")
	assert.Empty(t, out, "empty diff must filter ALL findings to zero, not pass them through")
}

func TestApplyDiffFilter_EnabledWithNilChangedFiles(t *testing.T) {
	// nil slice is treated the same as an empty slice: nothing to match
	// against, so everything is filtered out. Same regression guard.
	in := []*dsl.EnrichedDetection{detection("a.py")}
	out, applied := applyDiffFilter(in, nil, true)
	assert.True(t, applied)
	assert.Empty(t, out)
}

// --- countScannedFiles ---

func TestCountScannedFiles_DiffEnabledReturnsChangedCount(t *testing.T) {
	assert.Equal(t, 5, countScannedFiles(true, 5, 999),
		"diff-aware count must be the changed-files count, regardless of fallback")
}

func TestCountScannedFiles_DiffEnabledWithZeroChangedReturnsZero(t *testing.T) {
	// Regression guard mirroring applyDiffFilter's empty-diff behaviour: an
	// empty diff must report 0 files scanned, NOT silently use the fallback
	// count (which would mask the empty-diff state in the JSON output).
	assert.Equal(t, 0, countScannedFiles(true, 0, 250))
}

func TestCountScannedFiles_DiffDisabledReturnsFallback(t *testing.T) {
	assert.Equal(t, 42, countScannedFiles(false, 7, 42),
		"non-diff scans must report the unique-file count from the graph")
}

func TestCountScannedFiles_DiffDisabledWithZeroFallback(t *testing.T) {
	assert.Equal(t, 0, countScannedFiles(false, 7, 0))
}
