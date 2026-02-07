package output

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeDetection creates a minimal EnrichedDetection for testing.
func makeDetection(relPath string, severity string) *dsl.EnrichedDetection {
	return &dsl.EnrichedDetection{
		Location: dsl.LocationInfo{RelPath: relPath},
		Rule:     dsl.RuleMetadata{Severity: severity},
	}
}

func TestNewDiffFilter(t *testing.T) {
	tests := []struct {
		name         string
		changedFiles []string
		wantCount    int
	}{
		{
			name:         "with files",
			changedFiles: []string{"app/views.py", "app/models.py"},
			wantCount:    2,
		},
		{
			name:         "empty list",
			changedFiles: []string{},
			wantCount:    0,
		},
		{
			name:         "nil list",
			changedFiles: nil,
			wantCount:    0,
		},
		{
			name:         "duplicates are deduplicated",
			changedFiles: []string{"app/views.py", "app/views.py"},
			wantCount:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewDiffFilter(tt.changedFiles)
			require.NotNil(t, filter)
			assert.Equal(t, tt.wantCount, filter.ChangedFileCount())
		})
	}
}

func TestDiffFilter_Filter(t *testing.T) {
	tests := []struct {
		name         string
		changedFiles []string
		detections   []*dsl.EnrichedDetection
		wantCount    int
		wantRelPaths []string
	}{
		{
			name:         "filters to changed files only",
			changedFiles: []string{"app/views.py", "app/auth.py"},
			detections: []*dsl.EnrichedDetection{
				makeDetection("app/views.py", "critical"),
				makeDetection("app/models.py", "high"),
				makeDetection("app/auth.py", "medium"),
				makeDetection("lib/utils.py", "low"),
			},
			wantCount:    2,
			wantRelPaths: []string{"app/views.py", "app/auth.py"},
		},
		{
			name:         "empty changed files returns all detections",
			changedFiles: []string{},
			detections: []*dsl.EnrichedDetection{
				makeDetection("app/views.py", "critical"),
				makeDetection("app/models.py", "high"),
			},
			wantCount:    2,
			wantRelPaths: []string{"app/views.py", "app/models.py"},
		},
		{
			name:         "nil detections",
			changedFiles: []string{"app/views.py"},
			detections:   nil,
			wantCount:    0,
			wantRelPaths: nil,
		},
		{
			name:         "empty detections",
			changedFiles: []string{"app/views.py"},
			detections:   []*dsl.EnrichedDetection{},
			wantCount:    0,
			wantRelPaths: nil,
		},
		{
			name:         "no detections match changed files",
			changedFiles: []string{"app/views.py"},
			detections: []*dsl.EnrichedDetection{
				makeDetection("app/models.py", "critical"),
				makeDetection("lib/utils.py", "high"),
			},
			wantCount:    0,
			wantRelPaths: nil,
		},
		{
			name:         "all detections match changed files",
			changedFiles: []string{"app/views.py", "app/auth.py"},
			detections: []*dsl.EnrichedDetection{
				makeDetection("app/views.py", "critical"),
				makeDetection("app/auth.py", "high"),
			},
			wantCount:    2,
			wantRelPaths: []string{"app/views.py", "app/auth.py"},
		},
		{
			name:         "multiple detections in same changed file",
			changedFiles: []string{"app/views.py"},
			detections: []*dsl.EnrichedDetection{
				makeDetection("app/views.py", "critical"),
				makeDetection("app/views.py", "high"),
				makeDetection("app/views.py", "medium"),
				makeDetection("app/models.py", "low"),
			},
			wantCount:    3,
			wantRelPaths: []string{"app/views.py", "app/views.py", "app/views.py"},
		},
		{
			name:         "detection with empty RelPath is excluded",
			changedFiles: []string{"app/views.py"},
			detections: []*dsl.EnrichedDetection{
				makeDetection("app/views.py", "critical"),
				makeDetection("", "high"),
			},
			wantCount:    1,
			wantRelPaths: []string{"app/views.py"},
		},
		{
			name:         "path matching is exact (no partial match)",
			changedFiles: []string{"app/views.py"},
			detections: []*dsl.EnrichedDetection{
				makeDetection("app/views.py", "critical"),
				makeDetection("app/views.py.bak", "high"),
				makeDetection("other/app/views.py", "medium"),
			},
			wantCount:    1,
			wantRelPaths: []string{"app/views.py"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewDiffFilter(tt.changedFiles)
			result := filter.Filter(tt.detections)

			assert.Len(t, result, tt.wantCount)

			if tt.wantRelPaths != nil {
				gotPaths := make([]string, 0, len(result))
				for _, det := range result {
					gotPaths = append(gotPaths, det.Location.RelPath)
				}
				assert.Equal(t, tt.wantRelPaths, gotPaths)
			}
		})
	}
}

func TestDiffFilter_FilterPreservesOrder(t *testing.T) {
	// Verifies that filtered results maintain the original detection order.
	filter := NewDiffFilter([]string{"a.py", "c.py", "e.py"})

	detections := []*dsl.EnrichedDetection{
		makeDetection("a.py", "critical"),
		makeDetection("b.py", "high"),
		makeDetection("c.py", "medium"),
		makeDetection("d.py", "low"),
		makeDetection("e.py", "info"),
	}

	result := filter.Filter(detections)
	require.Len(t, result, 3)
	assert.Equal(t, "a.py", result[0].Location.RelPath)
	assert.Equal(t, "c.py", result[1].Location.RelPath)
	assert.Equal(t, "e.py", result[2].Location.RelPath)
}

func TestDiffFilter_FilterPreservesDetectionData(t *testing.T) {
	// Verifies that filtering does not modify detection contents.
	filter := NewDiffFilter([]string{"app/views.py"})

	original := &dsl.EnrichedDetection{
		Location: dsl.LocationInfo{
			FilePath:  "/project/app/views.py",
			RelPath:   "app/views.py",
			Line:      42,
			Column:    10,
			Function:  "process_request",
			ClassName: "ViewHandler",
		},
		Rule: dsl.RuleMetadata{
			ID:       "CMD-001",
			Name:     "Command Injection",
			Severity: "critical",
			CWE:      []string{"CWE-78"},
		},
		DetectionType: dsl.DetectionTypePattern,
	}

	result := filter.Filter([]*dsl.EnrichedDetection{original})
	require.Len(t, result, 1)

	// Should be the exact same pointer (not a copy).
	assert.Same(t, original, result[0])
	assert.Equal(t, "CMD-001", result[0].Rule.ID)
	assert.Equal(t, 42, result[0].Location.Line)
	assert.Equal(t, "ViewHandler", result[0].Location.ClassName)
}

func TestDiffFilter_FilteredCount(t *testing.T) {
	tests := []struct {
		name         string
		changedFiles []string
		detections   []*dsl.EnrichedDetection
		wantFiltered int
	}{
		{
			name:         "some filtered out",
			changedFiles: []string{"app/views.py"},
			detections: []*dsl.EnrichedDetection{
				makeDetection("app/views.py", "critical"),
				makeDetection("app/models.py", "high"),
				makeDetection("lib/utils.py", "medium"),
			},
			wantFiltered: 2,
		},
		{
			name:         "none filtered out",
			changedFiles: []string{"app/views.py", "app/models.py"},
			detections: []*dsl.EnrichedDetection{
				makeDetection("app/views.py", "critical"),
				makeDetection("app/models.py", "high"),
			},
			wantFiltered: 0,
		},
		{
			name:         "all filtered out",
			changedFiles: []string{"other.py"},
			detections: []*dsl.EnrichedDetection{
				makeDetection("app/views.py", "critical"),
				makeDetection("app/models.py", "high"),
			},
			wantFiltered: 2,
		},
		{
			name:         "empty changed files means no filtering",
			changedFiles: []string{},
			detections: []*dsl.EnrichedDetection{
				makeDetection("app/views.py", "critical"),
			},
			wantFiltered: 0,
		},
		{
			name:         "nil detections",
			changedFiles: []string{"app/views.py"},
			detections:   nil,
			wantFiltered: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewDiffFilter(tt.changedFiles)
			count := filter.FilteredCount(tt.detections)
			assert.Equal(t, tt.wantFiltered, count)
		})
	}
}

func TestDiffFilter_ChangedFileCount(t *testing.T) {
	tests := []struct {
		name         string
		changedFiles []string
		wantCount    int
	}{
		{
			name:         "multiple files",
			changedFiles: []string{"a.py", "b.py", "c.py"},
			wantCount:    3,
		},
		{
			name:         "single file",
			changedFiles: []string{"a.py"},
			wantCount:    1,
		},
		{
			name:         "empty",
			changedFiles: []string{},
			wantCount:    0,
		},
		{
			name:         "nil",
			changedFiles: nil,
			wantCount:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewDiffFilter(tt.changedFiles)
			assert.Equal(t, tt.wantCount, filter.ChangedFileCount())
		})
	}
}

func TestDiffFilter_FilterConsistency(t *testing.T) {
	// FilteredCount + len(Filter) should equal total detections.
	filter := NewDiffFilter([]string{"app/views.py", "app/auth.py"})

	detections := []*dsl.EnrichedDetection{
		makeDetection("app/views.py", "critical"),
		makeDetection("app/models.py", "high"),
		makeDetection("app/auth.py", "medium"),
		makeDetection("lib/utils.py", "low"),
		makeDetection("app/views.py", "info"),
	}

	filtered := filter.Filter(detections)
	filteredOut := filter.FilteredCount(detections)

	assert.Equal(t, len(detections), len(filtered)+filteredOut,
		"Filter() + FilteredCount() should equal total detections")
}
