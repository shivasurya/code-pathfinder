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

// --- ReviewManager tests ---

func TestPostInlineComments_NoEligible(t *testing.T) {
	// No HTTP calls should be made when there are no eligible findings.
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no HTTP call expected")
	})
	client := newTestClient(t, handler)
	rm := NewReviewManager(client, 1, "sha123")

	// All low/medium — none eligible.
	findings := []*dsl.EnrichedDetection{
		{Location: dsl.LocationInfo{RelPath: "a.py", Line: 1}, Rule: dsl.RuleMetadata{Severity: "low"}},
		{Location: dsl.LocationInfo{RelPath: "b.py", Line: 2}, Rule: dsl.RuleMetadata{Severity: "medium"}},
	}
	err := rm.PostInlineComments(context.Background(), findings)
	require.NoError(t, err)
}

func TestPostInlineComments_NilFindings(t *testing.T) {
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("no HTTP call expected")
	})
	client := newTestClient(t, handler)
	rm := NewReviewManager(client, 1, "sha123")

	err := rm.PostInlineComments(context.Background(), nil)
	require.NoError(t, err)
}

func TestPostInlineComments_CreatesNewReview(t *testing.T) {
	var reviewReq createReviewRequest
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			// ListReviewComments — no existing.
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]*ReviewComment{})

		case r.Method == http.MethodPost:
			require.NoError(t, json.NewDecoder(r.Body).Decode(&reviewReq))
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"id": 1})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	client := newTestClient(t, handler)
	rm := NewReviewManager(client, 42, "abc123")

	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "app/views.py", Line: 47},
			Rule:     dsl.RuleMetadata{ID: "CMD-001", Name: "Command Injection", Severity: "critical"},
		},
		{
			Location: dsl.LocationInfo{RelPath: "app/auth.py", Line: 23},
			Rule:     dsl.RuleMetadata{ID: "SQL-001", Name: "SQL Injection", Severity: "high"},
		},
	}

	err := rm.PostInlineComments(context.Background(), findings)
	require.NoError(t, err)

	assert.Equal(t, "abc123", reviewReq.CommitID)
	assert.Equal(t, "COMMENT", reviewReq.Event)
	require.Len(t, reviewReq.Comments, 2)
	assert.Equal(t, "app/views.py", reviewReq.Comments[0].Path)
	assert.Equal(t, 47, reviewReq.Comments[0].Line)
	assert.Equal(t, "RIGHT", reviewReq.Comments[0].Side)
	assert.Contains(t, reviewReq.Comments[0].Body, "Command Injection")
	assert.Contains(t, reviewReq.Comments[0].Body, "<!-- cpf-CMD-001-app/views.py-47 -->")
}

func TestPostInlineComments_UpdatesExisting(t *testing.T) {
	var updatedBody string
	finding := &dsl.EnrichedDetection{
		Location: dsl.LocationInfo{RelPath: "app/views.py", Line: 47},
		Rule:     dsl.RuleMetadata{ID: "CMD-001", Name: "Command Injection", Severity: "critical"},
	}
	marker := ReviewCommentMarker(finding)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			// ListReviewComments — return one with matching marker.
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]*ReviewComment{
				{ID: 99, Body: "old content\n" + marker + "\n", Path: "app/views.py", Line: 47},
			})

		case r.Method == http.MethodPatch:
			// UpdateReviewComment (pulls/comments endpoint).
			assert.Contains(t, r.URL.Path, "/pulls/comments/")
			var req updateCommentRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
			updatedBody = req.Body
			json.NewEncoder(w).Encode(ReviewComment{ID: 99, Body: req.Body})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	client := newTestClient(t, handler)
	rm := NewReviewManager(client, 42, "abc123")

	err := rm.PostInlineComments(context.Background(), []*dsl.EnrichedDetection{finding})
	require.NoError(t, err)
	assert.Contains(t, updatedBody, "Command Injection")
	assert.Contains(t, updatedBody, marker)
}

func TestPostInlineComments_MixedUpdateAndNew(t *testing.T) {
	existingMarker := "<!-- cpf-CMD-001-app/views.py-47 -->"
	var gotPatch, gotPost bool

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]*ReviewComment{
				{ID: 99, Body: "old\n" + existingMarker + "\n"},
			})

		case r.Method == http.MethodPatch:
			gotPatch = true
			json.NewEncoder(w).Encode(ReviewComment{ID: 99, Body: "updated"})

		case r.Method == http.MethodPost:
			gotPost = true
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"id": 2})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	client := newTestClient(t, handler)
	rm := NewReviewManager(client, 42, "sha")

	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "app/views.py", Line: 47},
			Rule:     dsl.RuleMetadata{ID: "CMD-001", Name: "Existing", Severity: "critical"},
		},
		{
			Location: dsl.LocationInfo{RelPath: "app/new.py", Line: 10},
			Rule:     dsl.RuleMetadata{ID: "NEW-001", Name: "New Finding", Severity: "high"},
		},
	}

	err := rm.PostInlineComments(context.Background(), findings)
	require.NoError(t, err)
	assert.True(t, gotPatch, "should have updated existing comment")
	assert.True(t, gotPost, "should have created review for new comment")
}

func TestPostInlineComments_ListError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(apiError{Message: "Bad credentials"})
	})

	client := newTestClient(t, handler)
	rm := NewReviewManager(client, 1, "sha")

	findings := []*dsl.EnrichedDetection{
		{Location: dsl.LocationInfo{RelPath: "a.py", Line: 1}, Rule: dsl.RuleMetadata{Severity: "critical"}},
	}
	err := rm.PostInlineComments(context.Background(), findings)
	assert.ErrorContains(t, err, "list existing review comments")
}

func TestPostInlineComments_UpdateError(t *testing.T) {
	marker := "<!-- cpf-X-a.py-1 -->"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]*ReviewComment{
				{ID: 5, Body: marker},
			})
			return
		}
		// PATCH fails.
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(apiError{Message: "error"})
	})

	client := newTestClient(t, handler)
	rm := NewReviewManager(client, 1, "sha")

	findings := []*dsl.EnrichedDetection{
		{Location: dsl.LocationInfo{RelPath: "a.py", Line: 1}, Rule: dsl.RuleMetadata{ID: "X", Severity: "critical"}},
	}
	err := rm.PostInlineComments(context.Background(), findings)
	assert.ErrorContains(t, err, "update inline comment")
}

func TestPostInlineComments_CreateReviewError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]*ReviewComment{})
			return
		}
		// POST fails.
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(apiError{Message: "Validation Failed"})
	})

	client := newTestClient(t, handler)
	rm := NewReviewManager(client, 1, "sha")

	findings := []*dsl.EnrichedDetection{
		{Location: dsl.LocationInfo{RelPath: "a.py", Line: 1}, Rule: dsl.RuleMetadata{ID: "X", Severity: "high"}},
	}
	err := rm.PostInlineComments(context.Background(), findings)
	assert.ErrorContains(t, err, "create review")
}

// --- ShouldPostInline tests ---

func TestShouldPostInline(t *testing.T) {
	tests := []struct {
		severity string
		want     bool
	}{
		{"critical", true},
		{"CRITICAL", true},
		{"Critical", true},
		{"high", true},
		{"HIGH", true},
		{"High", true},
		{"medium", false},
		{"low", false},
		{"", false},
		{"unknown", false},
	}
	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			assert.Equal(t, tt.want, ShouldPostInline(tt.severity))
		})
	}
}

// --- ReviewCommentMarker tests ---

func TestReviewCommentMarker(t *testing.T) {
	f := &dsl.EnrichedDetection{
		Location: dsl.LocationInfo{RelPath: "app/views.py", Line: 47},
		Rule:     dsl.RuleMetadata{ID: "CMD-001"},
	}
	marker := ReviewCommentMarker(f)
	assert.Equal(t, "<!-- cpf-CMD-001-app/views.py-47 -->", marker)
}

// --- FormatInlineComment tests ---

func TestFormatInlineComment_Basic(t *testing.T) {
	f := &dsl.EnrichedDetection{
		Location: dsl.LocationInfo{RelPath: "app/views.py", Line: 47},
		Rule: dsl.RuleMetadata{
			ID:          "CMD-001",
			Name:        "Command Injection",
			Severity:    "critical",
			Description: "User input flows to subprocess.",
			CWE:         []string{"CWE-78"},
			OWASP:       []string{"A03:2021"},
		},
	}

	result := FormatInlineComment(f)

	assert.Contains(t, result, "**Command Injection**")
	assert.Contains(t, result, "User input flows to subprocess.")
	assert.Contains(t, result, "CWE-78")
	assert.Contains(t, result, "A03:2021")
	assert.Contains(t, result, "<!-- cpf-CMD-001-app/views.py-47 -->")
	// Should have severity emoji.
	assert.True(t, strings.Contains(result, "\xf0\x9f\x94\xb4")) // red circle
}

func TestFormatInlineComment_WithTaintPath(t *testing.T) {
	f := &dsl.EnrichedDetection{
		Location: dsl.LocationInfo{RelPath: "app/views.py", Line: 47},
		Rule:     dsl.RuleMetadata{ID: "T-001", Name: "Taint Flow", Severity: "high"},
		TaintPath: []dsl.TaintPathNode{
			{
				Location: dsl.LocationInfo{RelPath: "app/input.py", Line: 10},
				Variable: "request.GET",
				IsSource: true,
			},
			{
				Location: dsl.LocationInfo{RelPath: "app/views.py", Line: 47},
				Variable: "subprocess.call()",
				IsSink:   true,
			},
		},
	}

	result := FormatInlineComment(f)

	assert.Contains(t, result, "**Flow:**")
	assert.Contains(t, result, "Source: `app/input.py:10`")
	assert.Contains(t, result, "`request.GET`")
	assert.Contains(t, result, "Sink: `app/views.py:47`")
	assert.Contains(t, result, "`subprocess.call()`")
}

func TestFormatInlineComment_NoDescription(t *testing.T) {
	f := &dsl.EnrichedDetection{
		Location: dsl.LocationInfo{RelPath: "a.py", Line: 1},
		Rule:     dsl.RuleMetadata{ID: "X", Name: "Issue", Severity: "high"},
	}

	result := FormatInlineComment(f)

	assert.Contains(t, result, "**Issue**")
	// No double newlines from empty description.
	assert.NotContains(t, result, "\n\n\n")
}

func TestFormatInlineComment_CWEOnly(t *testing.T) {
	f := &dsl.EnrichedDetection{
		Location: dsl.LocationInfo{RelPath: "a.py", Line: 1},
		Rule:     dsl.RuleMetadata{ID: "X", Name: "Issue", Severity: "high", CWE: []string{"CWE-79"}},
	}

	result := FormatInlineComment(f)
	assert.Contains(t, result, "CWE-79")
}

func TestFormatInlineComment_OWASPOnly(t *testing.T) {
	f := &dsl.EnrichedDetection{
		Location: dsl.LocationInfo{RelPath: "a.py", Line: 1},
		Rule:     dsl.RuleMetadata{ID: "X", Name: "Issue", Severity: "high", OWASP: []string{"A01:2021"}},
	}

	result := FormatInlineComment(f)
	assert.Contains(t, result, "A01:2021")
}

func TestFormatInlineComment_NoReferences(t *testing.T) {
	f := &dsl.EnrichedDetection{
		Location: dsl.LocationInfo{RelPath: "a.py", Line: 1},
		Rule:     dsl.RuleMetadata{ID: "X", Name: "Issue", Severity: "critical"},
	}

	result := FormatInlineComment(f)
	// Should still have marker and name, but no reference line.
	assert.Contains(t, result, "**Issue**")
	assert.Contains(t, result, "<!-- cpf-X-a.py-1 -->")
}

// --- filterEligible tests ---

func TestFilterEligible(t *testing.T) {
	findings := []*dsl.EnrichedDetection{
		{Location: dsl.LocationInfo{RelPath: "a.py", Line: 10}, Rule: dsl.RuleMetadata{Severity: "critical"}},
		{Location: dsl.LocationInfo{RelPath: "b.py", Line: 20}, Rule: dsl.RuleMetadata{Severity: "high"}},
		{Location: dsl.LocationInfo{RelPath: "c.py", Line: 30}, Rule: dsl.RuleMetadata{Severity: "medium"}},
		{Location: dsl.LocationInfo{RelPath: "d.py", Line: 40}, Rule: dsl.RuleMetadata{Severity: "low"}},
	}

	result := filterEligible(findings)

	require.Len(t, result, 2)
	assert.Equal(t, "a.py", result[0].Location.RelPath)
	assert.Equal(t, "b.py", result[1].Location.RelPath)
}

func TestFilterEligible_SkipsInvalidLocations(t *testing.T) {
	findings := []*dsl.EnrichedDetection{
		// Missing RelPath.
		{Location: dsl.LocationInfo{RelPath: "", Line: 10}, Rule: dsl.RuleMetadata{Severity: "critical"}},
		// Zero line.
		{Location: dsl.LocationInfo{RelPath: "a.py", Line: 0}, Rule: dsl.RuleMetadata{Severity: "high"}},
		// Valid.
		{Location: dsl.LocationInfo{RelPath: "b.py", Line: 5}, Rule: dsl.RuleMetadata{Severity: "critical"}},
	}

	result := filterEligible(findings)
	require.Len(t, result, 1)
	assert.Equal(t, "b.py", result[0].Location.RelPath)
}

func TestFilterEligible_Empty(t *testing.T) {
	assert.Empty(t, filterEligible(nil))
	assert.Empty(t, filterEligible([]*dsl.EnrichedDetection{}))
}

// --- indexByMarker tests ---

func TestIndexByMarker(t *testing.T) {
	comments := []*ReviewComment{
		{ID: 1, Body: "some text\n<!-- cpf-CMD-001-app/views.py-47 -->\n"},
		{ID: 2, Body: "no marker here"},
		{ID: 3, Body: "<!-- cpf-SQL-001-auth.py-10 -->"},
	}

	m := indexByMarker(comments)
	assert.Len(t, m, 2)
	assert.Equal(t, int64(1), m["<!-- cpf-CMD-001-app/views.py-47 -->"])
	assert.Equal(t, int64(3), m["<!-- cpf-SQL-001-auth.py-10 -->"])
}

func TestIndexByMarker_Empty(t *testing.T) {
	assert.Empty(t, indexByMarker(nil))
	assert.Empty(t, indexByMarker([]*ReviewComment{}))
}

func TestIndexByMarker_TruncatedMarker(t *testing.T) {
	// Marker starts but never closes — should not match.
	comments := []*ReviewComment{
		{ID: 1, Body: "<!-- cpf-CMD-001-app.py-1"},
	}
	assert.Empty(t, indexByMarker(comments))
}

// --- writeTaintFlow tests ---

func TestWriteTaintFlow_Complete(t *testing.T) {
	path := []dsl.TaintPathNode{
		{Location: dsl.LocationInfo{RelPath: "input.py", Line: 5}, Variable: "user_input", IsSource: true},
		{Location: dsl.LocationInfo{RelPath: "sink.py", Line: 20}, Variable: "exec()", IsSink: true},
	}
	var sb strings.Builder
	writeTaintFlow(&sb, path)

	result := sb.String()
	assert.Contains(t, result, "**Flow:**")
	assert.Contains(t, result, "Source: `input.py:5`")
	assert.Contains(t, result, "`user_input`")
	assert.Contains(t, result, "Sink: `sink.py:20`")
	assert.Contains(t, result, "`exec()`")
}

func TestWriteTaintFlow_NoVariables(t *testing.T) {
	path := []dsl.TaintPathNode{
		{Location: dsl.LocationInfo{RelPath: "a.py", Line: 1}, IsSource: true},
		{Location: dsl.LocationInfo{RelPath: "b.py", Line: 2}, IsSink: true},
	}
	var sb strings.Builder
	writeTaintFlow(&sb, path)

	result := sb.String()
	assert.Contains(t, result, "Source: `a.py:1`")
	assert.NotContains(t, result, "\u2014") // No em dash when no variable.
}

func TestWriteTaintFlow_FallbackToFilePath(t *testing.T) {
	// Source has no RelPath but has FilePath.
	path := []dsl.TaintPathNode{
		{Location: dsl.LocationInfo{FilePath: "/abs/path/a.py", Line: 1}, IsSource: true},
		{Location: dsl.LocationInfo{RelPath: "b.py", Line: 2}, IsSink: true},
	}
	var sb strings.Builder
	writeTaintFlow(&sb, path)

	assert.Contains(t, sb.String(), "Source: `/abs/path/a.py:1`")
}

func TestWriteTaintFlow_MissingSourceOrSink(t *testing.T) {
	// No source marked.
	path := []dsl.TaintPathNode{
		{Location: dsl.LocationInfo{RelPath: "a.py", Line: 1}},
		{Location: dsl.LocationInfo{RelPath: "b.py", Line: 2}, IsSink: true},
	}
	var sb strings.Builder
	writeTaintFlow(&sb, path)
	assert.Empty(t, sb.String())

	// No sink marked.
	path2 := []dsl.TaintPathNode{
		{Location: dsl.LocationInfo{RelPath: "a.py", Line: 1}, IsSource: true},
		{Location: dsl.LocationInfo{RelPath: "b.py", Line: 2}},
	}
	sb.Reset()
	writeTaintFlow(&sb, path2)
	assert.Empty(t, sb.String())
}

// --- writeReferences tests ---

func TestWriteReferences_Both(t *testing.T) {
	var sb strings.Builder
	writeReferences(&sb, []string{"CWE-78"}, []string{"A03:2021"})
	assert.Contains(t, sb.String(), "CWE-78")
	assert.Contains(t, sb.String(), "A03:2021")
	assert.Contains(t, sb.String(), "\u00b7") // Middle dot separator.
}

func TestWriteReferences_CWEOnly(t *testing.T) {
	var sb strings.Builder
	writeReferences(&sb, []string{"CWE-89", "CWE-90"}, nil)
	assert.Contains(t, sb.String(), "CWE-89, CWE-90")
	assert.NotContains(t, sb.String(), "\u00b7")
}

func TestWriteReferences_OWASPOnly(t *testing.T) {
	var sb strings.Builder
	writeReferences(&sb, nil, []string{"A01:2021"})
	assert.Contains(t, sb.String(), "A01:2021")
}

func TestWriteReferences_None(t *testing.T) {
	var sb strings.Builder
	writeReferences(&sb, nil, nil)
	assert.Empty(t, sb.String())
}

func TestWriteReferences_EmptySlices(t *testing.T) {
	var sb strings.Builder
	writeReferences(&sb, []string{}, []string{})
	assert.Empty(t, sb.String())
}

// --- NewReviewManager tests ---

func TestNewReviewManager(t *testing.T) {
	client := NewClient("tok", "o", "r")
	rm := NewReviewManager(client, 42, "sha123")
	assert.Equal(t, 42, rm.prNumber)
	assert.Equal(t, "sha123", rm.commitSHA)
	assert.Same(t, client, rm.client)
}
