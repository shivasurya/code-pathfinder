package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- PRCommentOptions tests ---

func TestPRCommentOptions_Enabled(t *testing.T) {
	tests := []struct {
		name    string
		opts    PRCommentOptions
		enabled bool
	}{
		{name: "both false", opts: PRCommentOptions{}, enabled: false},
		{name: "comment only", opts: PRCommentOptions{Comment: true}, enabled: true},
		{name: "inline only", opts: PRCommentOptions{Inline: true}, enabled: true},
		{name: "both true", opts: PRCommentOptions{Comment: true, Inline: true}, enabled: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.enabled, tt.opts.Enabled())
		})
	}
}

func TestPRCommentOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    PRCommentOptions
		wantErr string
	}{
		{
			name: "disabled passes validation",
			opts: PRCommentOptions{Comment: false, Inline: false},
		},
		{
			name:    "zero PR number",
			opts:    PRCommentOptions{Comment: true, PRNumber: 0},
			wantErr: "--github-pr must be a positive number",
		},
		{
			name:    "negative PR number",
			opts:    PRCommentOptions{Inline: true, PRNumber: -1},
			wantErr: "--github-pr must be a positive number",
		},
		{
			name: "valid options",
			opts: PRCommentOptions{Comment: true, Inline: true, PRNumber: 42},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- ParseRepo tests ---

func TestParseRepo(t *testing.T) {
	tests := []struct {
		input     string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{input: "owner/repo", wantOwner: "owner", wantRepo: "repo"},
		{input: "my-org/my-repo", wantOwner: "my-org", wantRepo: "my-repo"},
		{input: "org/repo/extra", wantOwner: "org", wantRepo: "repo/extra"}, // SplitN keeps rest.
		{input: "noslash", wantErr: true},
		{input: "/repo", wantErr: true},
		{input: "owner/", wantErr: true},
		{input: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			owner, repo, err := ParseRepo(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantOwner, owner)
				assert.Equal(t, tt.wantRepo, repo)
			}
		})
	}
}

// --- PostPRComments tests ---

// mockPRServer returns a test server that handles summary and inline comment flows.
func mockPRServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// ListComments (for summary comment PostOrUpdate).
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues/") && strings.Contains(r.URL.Path, "/comments"):
			json.NewEncoder(w).Encode([]Comment{})

		// CreateComment (new summary comment).
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/") && strings.Contains(r.URL.Path, "/comments"):
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(Comment{ID: 1, Body: "created"})

		// GetPullRequest (for inline comments).
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pulls/") && !strings.Contains(r.URL.Path, "/comments") && !strings.Contains(r.URL.Path, "/reviews"):
			json.NewEncoder(w).Encode(PullRequest{
				Number: 1,
				Head:   GitRef{SHA: "abc123"},
			})

		// ListReviewComments (for inline comment dedup).
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pulls/") && strings.Contains(r.URL.Path, "/comments"):
			json.NewEncoder(w).Encode([]*ReviewComment{})

		// CreateReview (new inline comments).
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/reviews"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"id": 1})

		default:
			t.Logf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// newPRTestClient creates a *Client pointing at a test server URL.
func newPRTestClient(serverURL string) *Client {
	c := NewClient("fake-token", "owner", "repo")
	c.SetBaseURL(serverURL)
	return c
}

// noopProgress is a no-op progress callback for tests.
func noopProgress(string, ...any) {}

func TestPostPRComments_SummaryOnly(t *testing.T) {
	srv := mockPRServer(t)
	defer srv.Close()
	client := newPRTestClient(srv.URL)

	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "app.py", Line: 10},
			Rule:     dsl.RuleMetadata{ID: "T-1", Name: "Test", Severity: "critical"},
		},
	}
	metrics := ScanMetrics{FilesScanned: 5, RulesExecuted: 10}

	opts := PRCommentOptions{
		PRNumber: 1,
		Comment:  true,
		Inline:   false,
	}
	err := PostPRComments(client, opts, findings, metrics, noopProgress)
	assert.NoError(t, err)
}

func TestPostPRComments_InlineOnly(t *testing.T) {
	srv := mockPRServer(t)
	defer srv.Close()
	client := newPRTestClient(srv.URL)

	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "app.py", Line: 10},
			Rule:     dsl.RuleMetadata{ID: "T-1", Name: "Test", Severity: "critical"},
		},
	}
	metrics := ScanMetrics{}

	opts := PRCommentOptions{
		PRNumber: 1,
		Comment:  false,
		Inline:   true,
	}
	err := PostPRComments(client, opts, findings, metrics, noopProgress)
	assert.NoError(t, err)
}

func TestPostPRComments_BothEnabled(t *testing.T) {
	srv := mockPRServer(t)
	defer srv.Close()
	client := newPRTestClient(srv.URL)

	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "app.py", Line: 10},
			Rule:     dsl.RuleMetadata{ID: "T-1", Name: "Test", Severity: "high"},
		},
	}
	metrics := ScanMetrics{FilesScanned: 5, RulesExecuted: 10}

	opts := PRCommentOptions{
		PRNumber: 1,
		Comment:  true,
		Inline:   true,
	}
	err := PostPRComments(client, opts, findings, metrics, noopProgress)
	assert.NoError(t, err)
}

func TestPostPRComments_NoneEnabled(t *testing.T) {
	srv := mockPRServer(t)
	defer srv.Close()
	client := newPRTestClient(srv.URL)

	opts := PRCommentOptions{
		PRNumber: 1,
		Comment:  false,
		Inline:   false,
	}
	err := PostPRComments(client, opts, nil, ScanMetrics{}, noopProgress)
	assert.NoError(t, err)
}

func TestPostPRComments_SummaryError(t *testing.T) {
	// Server that succeeds on GetPullRequest + ListComments but fails on CreateComment.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pulls/") && !strings.Contains(r.URL.Path, "/comments"):
			json.NewEncoder(w).Encode(PullRequest{Number: 1, Head: GitRef{SHA: "abc"}})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/comments"):
			json.NewEncoder(w).Encode([]Comment{})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments"):
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": "server error"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	client := newPRTestClient(srv.URL)

	opts := PRCommentOptions{
		PRNumber: 1,
		Comment:  true,
	}
	err := PostPRComments(client, opts, nil, ScanMetrics{}, noopProgress)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "post summary comment")
}

func TestPostPRComments_InlineGetPRError(t *testing.T) {
	// Server that fails on GetPullRequest.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "not found"})
	}))
	defer srv.Close()
	client := newPRTestClient(srv.URL)

	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "app.py", Line: 10},
			Rule:     dsl.RuleMetadata{ID: "T-1", Name: "Test", Severity: "critical"},
		},
	}
	opts := PRCommentOptions{
		PRNumber: 1,
		Inline:   true,
	}
	err := PostPRComments(client, opts, findings, ScanMetrics{}, noopProgress)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get PR metadata")
}

func TestPostPRComments_InlinePostError(t *testing.T) {
	// Server that succeeds on GetPR and ListReviewComments but fails on CreateReview.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pulls/") && !strings.Contains(r.URL.Path, "/comments"):
			json.NewEncoder(w).Encode(PullRequest{
				Number: 1,
				Head:   GitRef{SHA: "abc123"},
			})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/comments"):
			json.NewEncoder(w).Encode([]*ReviewComment{})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/reviews"):
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"message": "validation failed"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	client := newPRTestClient(srv.URL)

	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "app.py", Line: 10},
			Rule:     dsl.RuleMetadata{ID: "T-1", Name: "Test", Severity: "critical"},
		},
	}
	opts := PRCommentOptions{
		PRNumber: 1,
		Inline:   true,
	}
	err := PostPRComments(client, opts, findings, ScanMetrics{}, noopProgress)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "post inline comments")
}
