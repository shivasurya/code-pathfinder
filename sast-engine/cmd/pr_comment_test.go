package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/github"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- prCommentOptions tests ---

func TestPRCommentOptions_Enabled(t *testing.T) {
	tests := []struct {
		name    string
		opts    prCommentOptions
		enabled bool
	}{
		{name: "both false", opts: prCommentOptions{}, enabled: false},
		{name: "comment only", opts: prCommentOptions{Comment: true}, enabled: true},
		{name: "inline only", opts: prCommentOptions{Inline: true}, enabled: true},
		{name: "both true", opts: prCommentOptions{Comment: true, Inline: true}, enabled: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.enabled, tt.opts.enabled())
		})
	}
}

func TestPRCommentOptions_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    prCommentOptions
		wantErr string
	}{
		{
			name: "disabled passes validation",
			opts: prCommentOptions{Comment: false, Inline: false},
		},
		{
			name:    "missing token",
			opts:    prCommentOptions{Comment: true, Token: "", Repo: "o/r", PRNumber: 1},
			wantErr: "--github-token is required",
		},
		{
			name:    "missing repo",
			opts:    prCommentOptions{Comment: true, Token: "tok", Repo: "", PRNumber: 1},
			wantErr: "--github-repo is required",
		},
		{
			name:    "zero PR number",
			opts:    prCommentOptions{Comment: true, Token: "tok", Repo: "o/r", PRNumber: 0},
			wantErr: "--github-pr must be a positive number",
		},
		{
			name:    "negative PR number",
			opts:    prCommentOptions{Inline: true, Token: "tok", Repo: "o/r", PRNumber: -1},
			wantErr: "--github-pr must be a positive number",
		},
		{
			name:    "invalid repo format",
			opts:    prCommentOptions{Comment: true, Token: "tok", Repo: "noslash", PRNumber: 1},
			wantErr: "owner/repo format",
		},
		{
			name: "valid options",
			opts: prCommentOptions{Comment: true, Inline: true, Token: "tok", Repo: "owner/repo", PRNumber: 42},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.validate()
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// --- parseGitHubRepo tests ---

func TestParseGitHubRepo(t *testing.T) {
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
			owner, repo, err := parseGitHubRepo(tt.input)
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

// --- postPRComments tests ---

// mockGitHubServer returns a test server that handles summary and inline comment flows.
func mockGitHubServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// ListComments (for summary comment PostOrUpdate).
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues/") && strings.Contains(r.URL.Path, "/comments"):
			json.NewEncoder(w).Encode([]github.Comment{})

		// CreateComment (new summary comment).
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/") && strings.Contains(r.URL.Path, "/comments"):
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(github.Comment{ID: 1, Body: "created"})

		// GetPullRequest (for inline comments).
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pulls/") && !strings.Contains(r.URL.Path, "/comments") && !strings.Contains(r.URL.Path, "/reviews"):
			json.NewEncoder(w).Encode(github.PullRequest{
				Number: 1,
				Head:   github.GitRef{SHA: "abc123"},
			})

		// ListReviewComments (for inline comment dedup).
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pulls/") && strings.Contains(r.URL.Path, "/comments"):
			json.NewEncoder(w).Encode([]*github.ReviewComment{})

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

// withMockClient overrides newGitHubClient to point at a test server and restores it on cleanup.
func withMockClient(t *testing.T, serverURL string) {
	t.Helper()
	orig := newGitHubClient
	newGitHubClient = func(token, owner, repo string) *github.Client {
		c := github.NewClient(token, owner, repo)
		c.SetBaseURL(serverURL)
		return c
	}
	t.Cleanup(func() { newGitHubClient = orig })
}

func TestPostPRComments_SummaryOnly(t *testing.T) {
	srv := mockGitHubServer(t)
	defer srv.Close()
	withMockClient(t, srv.URL)

	logger := output.NewLogger(output.VerbosityDefault)
	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "app.py", Line: 10},
			Rule:     dsl.RuleMetadata{ID: "T-1", Name: "Test", Severity: "critical"},
		},
	}
	metrics := github.ScanMetrics{FilesScanned: 5, RulesExecuted: 10}

	opts := prCommentOptions{
		Token:    "fake-token",
		Repo:     "owner/repo",
		PRNumber: 1,
		Comment:  true,
		Inline:   false,
	}
	err := postPRComments(opts, findings, metrics, logger)
	assert.NoError(t, err)
}

func TestPostPRComments_InlineOnly(t *testing.T) {
	srv := mockGitHubServer(t)
	defer srv.Close()
	withMockClient(t, srv.URL)

	logger := output.NewLogger(output.VerbosityDefault)
	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "app.py", Line: 10},
			Rule:     dsl.RuleMetadata{ID: "T-1", Name: "Test", Severity: "critical"},
		},
	}
	metrics := github.ScanMetrics{}

	opts := prCommentOptions{
		Token:    "fake-token",
		Repo:     "owner/repo",
		PRNumber: 1,
		Comment:  false,
		Inline:   true,
	}
	err := postPRComments(opts, findings, metrics, logger)
	assert.NoError(t, err)
}

func TestPostPRComments_BothEnabled(t *testing.T) {
	srv := mockGitHubServer(t)
	defer srv.Close()
	withMockClient(t, srv.URL)

	logger := output.NewLogger(output.VerbosityDefault)
	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "app.py", Line: 10},
			Rule:     dsl.RuleMetadata{ID: "T-1", Name: "Test", Severity: "high"},
		},
	}
	metrics := github.ScanMetrics{FilesScanned: 5, RulesExecuted: 10}

	opts := prCommentOptions{
		Token:    "fake-token",
		Repo:     "owner/repo",
		PRNumber: 1,
		Comment:  true,
		Inline:   true,
	}
	err := postPRComments(opts, findings, metrics, logger)
	assert.NoError(t, err)
}

func TestPostPRComments_NoneEnabled(t *testing.T) {
	srv := mockGitHubServer(t)
	defer srv.Close()
	withMockClient(t, srv.URL)

	logger := output.NewLogger(output.VerbosityDefault)
	opts := prCommentOptions{
		Token:    "tok",
		Repo:     "o/r",
		PRNumber: 1,
		Comment:  false,
		Inline:   false,
	}
	err := postPRComments(opts, nil, github.ScanMetrics{}, logger)
	assert.NoError(t, err)
}

func TestPostPRComments_SummaryError(t *testing.T) {
	// Server that succeeds on GetPullRequest + ListComments but fails on CreateComment.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pulls/") && !strings.Contains(r.URL.Path, "/comments"):
			json.NewEncoder(w).Encode(github.PullRequest{Number: 1, Head: github.GitRef{SHA: "abc"}})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/comments"):
			json.NewEncoder(w).Encode([]github.Comment{})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments"):
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"message": "server error"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	withMockClient(t, srv.URL)

	logger := output.NewLogger(output.VerbosityDefault)
	opts := prCommentOptions{
		Token:    "tok",
		Repo:     "owner/repo",
		PRNumber: 1,
		Comment:  true,
	}
	err := postPRComments(opts, nil, github.ScanMetrics{}, logger)
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
	withMockClient(t, srv.URL)

	logger := output.NewLogger(output.VerbosityDefault)
	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "app.py", Line: 10},
			Rule:     dsl.RuleMetadata{ID: "T-1", Name: "Test", Severity: "critical"},
		},
	}
	opts := prCommentOptions{
		Token:    "tok",
		Repo:     "owner/repo",
		PRNumber: 1,
		Inline:   true,
	}
	err := postPRComments(opts, findings, github.ScanMetrics{}, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get PR metadata")
}

func TestPostPRComments_InlinePostError(t *testing.T) {
	// Server that succeeds on GetPR and ListReviewComments but fails on CreateReview.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pulls/") && !strings.Contains(r.URL.Path, "/comments"):
			json.NewEncoder(w).Encode(github.PullRequest{
				Number: 1,
				Head:   github.GitRef{SHA: "abc123"},
			})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/comments"):
			json.NewEncoder(w).Encode([]*github.ReviewComment{})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/reviews"):
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"message": "validation failed"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	withMockClient(t, srv.URL)

	logger := output.NewLogger(output.VerbosityDefault)
	findings := []*dsl.EnrichedDetection{
		{
			Location: dsl.LocationInfo{RelPath: "app.py", Line: 10},
			Rule:     dsl.RuleMetadata{ID: "T-1", Name: "Test", Severity: "critical"},
		},
	}
	opts := prCommentOptions{
		Token:    "tok",
		Repo:     "owner/repo",
		PRNumber: 1,
		Inline:   true,
	}
	err := postPRComments(opts, findings, github.ScanMetrics{}, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "post inline comments")
}

// --- CI command flag registration tests ---

func TestCICommandPRFlags(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		defValue string
	}{
		{name: "github-token", flag: "github-token", defValue: ""},
		{name: "github-repo", flag: "github-repo", defValue: ""},
		{name: "github-pr", flag: "github-pr", defValue: "0"},
		{name: "pr-comment", flag: "pr-comment", defValue: "false"},
		{name: "pr-inline", flag: "pr-inline", defValue: "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := ciCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, flag, "flag %q should be registered on ci command", tt.flag)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}
