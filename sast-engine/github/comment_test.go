package github

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- CommentManager tests ---

func TestPostOrUpdate_CreatesNew(t *testing.T) {
	var createdBody string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/comments"):
			// ListComments returns empty — no existing summary comment.
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]*Comment{})

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments"):
			var req createCommentRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			createdBody = req.Body
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Comment{ID: 1, Body: req.Body})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	client := newTestClient(t, handler)
	cm := NewCommentManager(client, 42)

	err := cm.PostOrUpdate(context.Background(), "## Scan Results")
	require.NoError(t, err)
	assert.Contains(t, createdBody, summaryMarker)
	assert.Contains(t, createdBody, "## Scan Results")
}

func TestPostOrUpdate_UpdatesExisting(t *testing.T) {
	var updatedBody string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/comments"):
			// ListComments returns a comment with the marker.
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]*Comment{
				{ID: 10, Body: "unrelated comment"},
				{ID: 77, Body: summaryMarker + "\nold results"},
			})

		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/comments/77"):
			var req updateCommentRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			updatedBody = req.Body
			json.NewEncoder(w).Encode(Comment{ID: 77, Body: req.Body})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	client := newTestClient(t, handler)
	cm := NewCommentManager(client, 42)

	err := cm.PostOrUpdate(context.Background(), "## Updated Results")
	require.NoError(t, err)
	assert.Contains(t, updatedBody, summaryMarker)
	assert.Contains(t, updatedBody, "## Updated Results")
}

func TestPostOrUpdate_ListError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(apiError{Message: "Bad credentials"})
	})

	client := newTestClient(t, handler)
	cm := NewCommentManager(client, 42)

	err := cm.PostOrUpdate(context.Background(), "body")
	assert.ErrorContains(t, err, "find existing comment")
}

func TestPostOrUpdate_CreateError(t *testing.T) {
	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]*Comment{})
			return
		}
		// POST fails.
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(apiError{Message: "forbidden"})
	})

	client := newTestClient(t, handler)
	cm := NewCommentManager(client, 42)

	err := cm.PostOrUpdate(context.Background(), "body")
	assert.ErrorContains(t, err, "create summary comment")
}

func TestPostOrUpdate_UpdateError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]*Comment{
				{ID: 5, Body: summaryMarker + "\nold"},
			})
			return
		}
		// PATCH fails.
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(apiError{Message: "server error"})
	})

	client := newTestClient(t, handler)
	cm := NewCommentManager(client, 42)

	err := cm.PostOrUpdate(context.Background(), "body")
	assert.ErrorContains(t, err, "update summary comment")
}

// --- FormatSummaryComment tests ---

func TestFormatSummaryComment_NoFindings(t *testing.T) {
	result := FormatSummaryComment(nil, ScanMetrics{FilesScanned: 5, RulesExecuted: 10})

	assert.Contains(t, result, "## [Code Pathfinder](https://codepathfinder.dev) Security Scan")
	assert.Contains(t, result, "Security-Pass-success")
	assert.Contains(t, result, "**No security issues detected.**")
	assert.Contains(t, result, "| Files Scanned | 5 |")
	assert.Contains(t, result, "| Rules | 10 |")
	assert.Contains(t, result, "Code Pathfinder")
	// Should not contain findings table.
	assert.NotContains(t, result, "### Findings")
}

func TestFormatSummaryComment_WithFindings(t *testing.T) {
	// Provide findings in non-severity order to verify sorting.
	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "app/utils.py", Line: 100},
			Rule: dsl.RuleMetadata{
				ID: "PATH-001", Name: "Path Traversal", Severity: "medium",
			},
		},
		{
			Location: dsl.LocationInfo{RelPath: "app/views.py", Line: 47},
			Rule: dsl.RuleMetadata{
				ID: "CMD-001", Name: "Command Injection", Severity: "critical",
				CWE: []string{"CWE-78"}, Description: "User input flows to subprocess.",
			},
		},
		{
			Location: dsl.LocationInfo{RelPath: "app/auth.py", Line: 23},
			Rule: dsl.RuleMetadata{
				ID: "SQL-001", Name: "SQL Injection", Severity: "high",
				CWE: []string{"CWE-89"},
			},
		},
	}
	metrics := ScanMetrics{FilesScanned: 6, RulesExecuted: 23}

	result := FormatSummaryComment(findings, metrics)

	// Status badge.
	assert.Contains(t, result, "Security-Issues_Found-critical")
	// Severity badges.
	assert.Contains(t, result, "Critical-1-critical")
	assert.Contains(t, result, "High-1-orange")
	assert.Contains(t, result, "Medium-1-yellow")
	// Findings table.
	assert.Contains(t, result, "### Findings")
	assert.Contains(t, result, "| `app/views.py` | 47 | Command Injection |")
	assert.Contains(t, result, "| `app/auth.py` | 23 | SQL Injection |")
	assert.Contains(t, result, "| `app/utils.py` | 100 | Path Traversal |")
	// Verify sort order: critical before high before medium.
	critIdx := strings.Index(result, "Command Injection")
	highIdx := strings.Index(result, "SQL Injection")
	medIdx := strings.Index(result, "Path Traversal")
	assert.Less(t, critIdx, highIdx, "critical should appear before high")
	assert.Less(t, highIdx, medIdx, "high should appear before medium")
	// No details section (removed).
	assert.NotContains(t, result, "<details>")
	// Critical warning.
	assert.Contains(t, result, "1 critical issue(s)")
	// Metrics.
	assert.Contains(t, result, "| Files Scanned | 6 |")
	assert.Contains(t, result, "| Rules | 23 |")
}

func TestFormatSummaryComment_LowOnlyFindings(t *testing.T) {
	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "a.py", Line: 1},
			Rule:     dsl.RuleMetadata{Name: "Minor Issue", Severity: "low"},
		},
	}

	result := FormatSummaryComment(findings, ScanMetrics{})

	// Issues found badge (not pass).
	assert.Contains(t, result, "Issues_Found")
	// Low badge with count.
	assert.Contains(t, result, "Low-1-blue")
	// No critical warning.
	assert.NotContains(t, result, "critical issue(s)")
	// Still has findings table.
	assert.Contains(t, result, "### Findings")
}

func TestFormatSummaryComment_InfoOnlyFindings(t *testing.T) {
	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "Dockerfile", Line: 1},
			Rule:     dsl.RuleMetadata{Name: "Deprecated Maintainer", Severity: "info"},
		},
	}

	result := FormatSummaryComment(findings, ScanMetrics{})

	// Issues found badge (not pass).
	assert.Contains(t, result, "Issues_Found")
	// Info badge with count.
	assert.Contains(t, result, "Info-1-informational")
	// No critical warning.
	assert.NotContains(t, result, "critical issue(s)")
	// Has findings table.
	assert.Contains(t, result, "### Findings")
}

func TestFormatSummaryComment_ZeroBadgesGreen(t *testing.T) {
	result := FormatSummaryComment(nil, ScanMetrics{})

	assert.Contains(t, result, "Critical-0-success")
	assert.Contains(t, result, "High-0-success")
	assert.Contains(t, result, "Medium-0-success")
	assert.Contains(t, result, "Low-0-success")
	assert.Contains(t, result, "Info-0-success")
}

// --- Sorting tests ---

func TestSeverityOrder(t *testing.T) {
	assert.Equal(t, 0, severityOrder("critical"))
	assert.Equal(t, 1, severityOrder("high"))
	assert.Equal(t, 2, severityOrder("medium"))
	assert.Equal(t, 3, severityOrder("low"))
	assert.Equal(t, 4, severityOrder("info"))
	assert.Equal(t, 5, severityOrder("unknown"))
}

func TestSortBySeverity(t *testing.T) {
	findings := []*dsl.EnrichedDetection{
		{Rule: dsl.RuleMetadata{ID: "R1", Severity: "low"}},
		{Rule: dsl.RuleMetadata{ID: "R2", Severity: "critical"}},
		{Rule: dsl.RuleMetadata{ID: "R3", Severity: "medium"}},
		{Rule: dsl.RuleMetadata{ID: "R4", Severity: "high"}},
		{Rule: dsl.RuleMetadata{ID: "R5", Severity: "info"}},
	}

	sorted := sortBySeverity(findings)

	// Verify order: critical, high, medium, low, info.
	assert.Equal(t, "R2", sorted[0].Rule.ID)
	assert.Equal(t, "R4", sorted[1].Rule.ID)
	assert.Equal(t, "R3", sorted[2].Rule.ID)
	assert.Equal(t, "R1", sorted[3].Rule.ID)
	assert.Equal(t, "R5", sorted[4].Rule.ID)

	// Verify original slice is not mutated.
	assert.Equal(t, "R1", findings[0].Rule.ID)
}

func TestSortBySeverity_StableOrder(t *testing.T) {
	findings := []*dsl.EnrichedDetection{
		{Rule: dsl.RuleMetadata{ID: "A", Severity: "high"}},
		{Rule: dsl.RuleMetadata{ID: "B", Severity: "high"}},
		{Rule: dsl.RuleMetadata{ID: "C", Severity: "high"}},
	}

	sorted := sortBySeverity(findings)

	// Same-severity items preserve original order (stable sort).
	assert.Equal(t, "A", sorted[0].Rule.ID)
	assert.Equal(t, "B", sorted[1].Rule.ID)
	assert.Equal(t, "C", sorted[2].Rule.ID)
}

// --- Helper function tests ---

func TestCountBySeverity(t *testing.T) {
	findings := []*dsl.EnrichedDetection{
		{Rule: dsl.RuleMetadata{Severity: "critical"}},
		{Rule: dsl.RuleMetadata{Severity: "critical"}},
		{Rule: dsl.RuleMetadata{Severity: "high"}},
		{Rule: dsl.RuleMetadata{Severity: "medium"}},
		{Rule: dsl.RuleMetadata{Severity: "low"}},
		{Rule: dsl.RuleMetadata{Severity: "low"}},
		{Rule: dsl.RuleMetadata{Severity: "info"}},
		{Rule: dsl.RuleMetadata{Severity: "unknown"}}, // Ignored.
	}

	c := countBySeverity(findings)
	assert.Equal(t, 2, c.Critical)
	assert.Equal(t, 1, c.High)
	assert.Equal(t, 1, c.Medium)
	assert.Equal(t, 2, c.Low)
	assert.Equal(t, 1, c.Info)
}

func TestCountBySeverity_Empty(t *testing.T) {
	c := countBySeverity(nil)
	assert.Equal(t, 0, c.Critical)
	assert.Equal(t, 0, c.High)
	assert.Equal(t, 0, c.Medium)
	assert.Equal(t, 0, c.Low)
	assert.Equal(t, 0, c.Info)
}

func TestSeverityEmoji(t *testing.T) {
	assert.NotEmpty(t, severityEmoji("critical"))
	assert.NotEmpty(t, severityEmoji("high"))
	assert.NotEmpty(t, severityEmoji("medium"))
	assert.NotEmpty(t, severityEmoji("low"))
	assert.NotEmpty(t, severityEmoji("info"))
	assert.Empty(t, severityEmoji("unknown"))
}

func TestSeverityLabel(t *testing.T) {
	assert.Contains(t, severityLabel("critical"), "**Critical**")
	assert.Contains(t, severityLabel("high"), "High")
	assert.Contains(t, severityLabel("medium"), "Medium")
	assert.Contains(t, severityLabel("low"), "Low")
	assert.Contains(t, severityLabel("info"), "Info")
	assert.Equal(t, "other", severityLabel("other"))
}

func TestStatusBadge(t *testing.T) {
	badge := statusBadge("Pass", "success")
	assert.Contains(t, badge, "Security-Pass-success")
	assert.Contains(t, badge, "shields.io")

	badge = statusBadge("Issues Found", "critical")
	assert.Contains(t, badge, "Security-Issues_Found-critical")
}

func TestSeverityBadge(t *testing.T) {
	assert.Contains(t, severityBadge("Critical", 3), "Critical-3-critical")
	assert.Contains(t, severityBadge("Critical", 0), "Critical-0-success")
	assert.Contains(t, severityBadge("High", 1), "High-1-orange")
	assert.Contains(t, severityBadge("High", 0), "High-0-success")
	assert.Contains(t, severityBadge("Medium", 2), "Medium-2-yellow")
	assert.Contains(t, severityBadge("Medium", 0), "Medium-0-success")
	assert.Contains(t, severityBadge("Low", 4), "Low-4-blue")
	assert.Contains(t, severityBadge("Low", 0), "Low-0-success")
	assert.Contains(t, severityBadge("Info", 1), "Info-1-informational")
	assert.Contains(t, severityBadge("Info", 0), "Info-0-success")
}

func TestWriteFindingsTable_NoLinks(t *testing.T) {
	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "x.py", Line: 5},
			Rule:     dsl.RuleMetadata{Name: "Issue X", Severity: "high"},
		},
	}
	var sb strings.Builder
	writeFindingsTable(&sb, findings, "")

	result := sb.String()
	assert.Contains(t, result, "### Findings")
	assert.Contains(t, result, "| Severity | File | Line | Issue |")
	assert.Contains(t, result, "| `x.py` | 5 | Issue X |")
	assert.NotContains(t, result, "\xf0\x9f\x94\x97") // No link emoji.
}

func TestWriteFindingsTable_WithLinks(t *testing.T) {
	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "app/views.py", Line: 42},
			Rule:     dsl.RuleMetadata{Name: "SQL Injection", Severity: "critical"},
		},
	}
	var sb strings.Builder
	writeFindingsTable(&sb, findings, "https://github.com/owner/repo/blob/abc123")

	result := sb.String()
	assert.Contains(t, result, "| Severity | File | Line | Issue | |")
	assert.Contains(t, result, "https://github.com/owner/repo/blob/abc123/app/views.py#L42")
	assert.Contains(t, result, "\xf0\x9f\x94\x97") // Link emoji.
}
