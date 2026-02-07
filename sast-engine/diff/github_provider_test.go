package diff

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubAPIDiffProvider_NormalResponse(t *testing.T) {
	// Tests a normal response with mixed file statuses.
	files := []pullRequestFile{
		{Filename: "app/views.py", Status: "modified"},
		{Filename: "app/new_file.py", Status: "added"},
		{Filename: "app/old_file.py", Status: "removed"},
		{Filename: "app/renamed.py", Status: "renamed"},
		{Filename: "app/copied.py", Status: "copied"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))
		assert.Equal(t, "2022-11-28", r.Header.Get("X-GitHub-Api-Version"))
		assert.Contains(t, r.URL.Path, "/repos/owner/repo/pulls/42/files")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	}))
	defer server.Close()

	provider := &GitHubAPIDiffProvider{
		Token:    "test-token",
		Owner:    "owner",
		Repo:     "repo",
		PRNumber: 42,
		BaseURL:  server.URL,
	}

	result, err := provider.GetChangedFiles()
	require.NoError(t, err)
	// "removed" should be excluded.
	assert.ElementsMatch(t, []string{
		"app/views.py",
		"app/new_file.py",
		"app/renamed.py",
		"app/copied.py",
	}, result)
}

func TestGitHubAPIDiffProvider_Pagination(t *testing.T) {
	// Tests pagination with Link header containing rel="next".
	page1Files := make([]pullRequestFile, 100)
	for i := range page1Files {
		page1Files[i] = pullRequestFile{
			Filename: fmt.Sprintf("file_%03d.py", i),
			Status:   "modified",
		}
	}
	page2Files := []pullRequestFile{
		{Filename: "file_100.py", Status: "added"},
		{Filename: "file_101.py", Status: "modified"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")

		switch page {
		case "1", "":
			// Include Link header for next page.
			w.Header().Set("Link", fmt.Sprintf(`<%s/repos/owner/repo/pulls/1/files?page=2>; rel="next"`, r.URL.Scheme+"://"+r.Host))
			json.NewEncoder(w).Encode(page1Files)
		case "2":
			// No Link header = last page.
			json.NewEncoder(w).Encode(page2Files)
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	provider := &GitHubAPIDiffProvider{
		Token:    "test-token",
		Owner:    "owner",
		Repo:     "repo",
		PRNumber: 1,
		BaseURL:  server.URL,
	}

	result, err := provider.GetChangedFiles()
	require.NoError(t, err)
	assert.Len(t, result, 102)
	assert.Contains(t, result, "file_000.py")
	assert.Contains(t, result, "file_100.py")
	assert.Contains(t, result, "file_101.py")
}

func TestGitHubAPIDiffProvider_EmptyPR(t *testing.T) {
	// Tests a PR with no changed files.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]pullRequestFile{})
	}))
	defer server.Close()

	provider := &GitHubAPIDiffProvider{
		Token:    "test-token",
		Owner:    "owner",
		Repo:     "repo",
		PRNumber: 1,
		BaseURL:  server.URL,
	}

	result, err := provider.GetChangedFiles()
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestGitHubAPIDiffProvider_APIErrors(t *testing.T) {
	// Tests various API error responses.
	tests := []struct {
		name       string
		statusCode int
		body       string
		errContain string
	}{
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"message": "Bad credentials"}`,
			errContain: "status 401",
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			body:       `{"message": "Not Found"}`,
			errContain: "status 404",
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			body:       `{"message": "Internal Server Error"}`,
			errContain: "status 500",
		},
		{
			name:       "forbidden",
			statusCode: http.StatusForbidden,
			body:       `{"message": "Resource not accessible by integration"}`,
			errContain: "status 403",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				fmt.Fprint(w, tt.body)
			}))
			defer server.Close()

			provider := &GitHubAPIDiffProvider{
				Token:    "bad-token",
				Owner:    "owner",
				Repo:     "repo",
				PRNumber: 1,
				BaseURL:  server.URL,
			}

			result, err := provider.GetChangedFiles()
			assert.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tt.errContain)
		})
	}
}

func TestGitHubAPIDiffProvider_InvalidJSON(t *testing.T) {
	// Tests handling of invalid JSON response.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, "not valid json{{{")
	}))
	defer server.Close()

	provider := &GitHubAPIDiffProvider{
		Token:    "test-token",
		Owner:    "owner",
		Repo:     "repo",
		PRNumber: 1,
		BaseURL:  server.URL,
	}

	result, err := provider.GetChangedFiles()
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode GitHub API response")
}

func TestGitHubAPIDiffProvider_NetworkError(t *testing.T) {
	// Tests handling of network errors (server not reachable).
	provider := &GitHubAPIDiffProvider{
		Token:    "test-token",
		Owner:    "owner",
		Repo:     "repo",
		PRNumber: 1,
		BaseURL:  "http://127.0.0.1:1", // Port 1 is unlikely to be open.
	}

	result, err := provider.GetChangedFiles()
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "GitHub API request failed")
}

func TestGitHubAPIDiffProvider_AllRemovedFiles(t *testing.T) {
	// Tests that a PR with only removed files returns empty list.
	files := []pullRequestFile{
		{Filename: "deleted1.py", Status: "removed"},
		{Filename: "deleted2.py", Status: "removed"},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(files)
	}))
	defer server.Close()

	provider := &GitHubAPIDiffProvider{
		Token:    "test-token",
		Owner:    "owner",
		Repo:     "repo",
		PRNumber: 1,
		BaseURL:  server.URL,
	}

	result, err := provider.GetChangedFiles()
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestHasNextPage(t *testing.T) {
	tests := []struct {
		name       string
		linkHeader string
		want       bool
	}{
		{
			name:       "empty header",
			linkHeader: "",
			want:       false,
		},
		{
			name:       "has next",
			linkHeader: `<https://api.github.com/repos/owner/repo/pulls/1/files?page=2>; rel="next"`,
			want:       true,
		},
		{
			name:       "last page only",
			linkHeader: `<https://api.github.com/repos/owner/repo/pulls/1/files?page=1>; rel="prev"`,
			want:       false,
		},
		{
			name:       "multiple links with next",
			linkHeader: `<https://api.github.com/repos/owner/repo/pulls/1/files?page=1>; rel="prev", <https://api.github.com/repos/owner/repo/pulls/1/files?page=3>; rel="next"`,
			want:       true,
		},
		{
			name:       "multiple links without next",
			linkHeader: `<https://api.github.com/repos/owner/repo/pulls/1/files?page=1>; rel="prev", <https://api.github.com/repos/owner/repo/pulls/1/files?page=3>; rel="last"`,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hasNextPage(tt.linkHeader))
		})
	}
}

func TestGitHubAPIDiffProvider_RequestURL(t *testing.T) {
	// Verifies the correct URL is constructed for the API request.
	var capturedURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]pullRequestFile{})
	}))
	defer server.Close()

	provider := &GitHubAPIDiffProvider{
		Token:    "test-token",
		Owner:    "my-org",
		Repo:     "my-repo",
		PRNumber: 99,
		BaseURL:  server.URL,
	}

	_, err := provider.GetChangedFiles()
	require.NoError(t, err)
	assert.Equal(t, "/repos/my-org/my-repo/pulls/99/files?per_page=100&page=1", capturedURL)
}
