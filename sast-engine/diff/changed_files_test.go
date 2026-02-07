package diff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChangedFilesProvider_GitHubAPI(t *testing.T) {
	// GitHub API provider is preferred when token, owner, repo, and PR number are available.
	opts := ProviderOptions{
		BaseRef:     "origin/main",
		GitHubToken: "ghp_test_token",
		Owner:       "shivasurya",
		Repo:        "code-pathfinder",
		PRNumber:    42,
	}

	provider, err := NewChangedFilesProvider(opts)
	require.NoError(t, err)

	ghProvider, ok := provider.(*GitHubAPIDiffProvider)
	require.True(t, ok, "expected GitHubAPIDiffProvider")
	assert.Equal(t, "ghp_test_token", ghProvider.Token)
	assert.Equal(t, "shivasurya", ghProvider.Owner)
	assert.Equal(t, "code-pathfinder", ghProvider.Repo)
	assert.Equal(t, 42, ghProvider.PRNumber)
}

func TestNewChangedFilesProvider_GitFallback(t *testing.T) {
	// Git provider is used when no GitHub context is available.
	opts := ProviderOptions{
		ProjectRoot: "/some/project",
		BaseRef:     "origin/main",
		HeadRef:     "feature-branch",
	}

	provider, err := NewChangedFilesProvider(opts)
	require.NoError(t, err)

	gitProvider, ok := provider.(*GitDiffProvider)
	require.True(t, ok, "expected GitDiffProvider")
	assert.Equal(t, "/some/project", gitProvider.ProjectRoot)
	assert.Equal(t, "origin/main", gitProvider.BaseRef)
	assert.Equal(t, "feature-branch", gitProvider.HeadRef)
}

func TestNewChangedFilesProvider_GitDefaultHead(t *testing.T) {
	// HeadRef defaults to "HEAD" when not specified.
	opts := ProviderOptions{
		ProjectRoot: "/some/project",
		BaseRef:     "origin/main",
	}

	provider, err := NewChangedFilesProvider(opts)
	require.NoError(t, err)

	gitProvider, ok := provider.(*GitDiffProvider)
	require.True(t, ok, "expected GitDiffProvider")
	assert.Equal(t, "HEAD", gitProvider.HeadRef)
}

func TestNewChangedFilesProvider_NoBaseRef(t *testing.T) {
	// Returns error when no baseline is available.
	opts := ProviderOptions{
		ProjectRoot: "/some/project",
	}

	provider, err := NewChangedFilesProvider(opts)
	assert.Nil(t, provider)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no baseline ref provided")
}

func TestNewChangedFilesProvider_GitHubMissingOwner(t *testing.T) {
	// Returns error when GitHub token is set but owner/repo is incomplete.
	opts := ProviderOptions{
		GitHubToken: "ghp_test_token",
		PRNumber:    42,
		Owner:       "shivasurya",
		// Repo is missing.
	}

	provider, err := NewChangedFilesProvider(opts)
	assert.Nil(t, provider)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "github-repo must specify both owner and repo")
}

func TestNewChangedFilesProvider_GitHubPrefersOverGit(t *testing.T) {
	// When both GitHub context and git base ref are available, GitHub API is preferred.
	opts := ProviderOptions{
		ProjectRoot: "/some/project",
		BaseRef:     "origin/main",
		GitHubToken: "ghp_test_token",
		Owner:       "shivasurya",
		Repo:        "code-pathfinder",
		PRNumber:    42,
	}

	provider, err := NewChangedFilesProvider(opts)
	require.NoError(t, err)

	_, ok := provider.(*GitHubAPIDiffProvider)
	assert.True(t, ok, "expected GitHubAPIDiffProvider when both are available")
}

func TestNewChangedFilesProvider_GitHubNoBaseRefStillWorks(t *testing.T) {
	// GitHub API provider does not require BaseRef since it uses the PR endpoint.
	opts := ProviderOptions{
		GitHubToken: "ghp_test_token",
		Owner:       "shivasurya",
		Repo:        "code-pathfinder",
		PRNumber:    42,
	}

	provider, err := NewChangedFilesProvider(opts)
	require.NoError(t, err)

	_, ok := provider.(*GitHubAPIDiffProvider)
	assert.True(t, ok, "expected GitHubAPIDiffProvider")
}

func TestNewChangedFilesProvider_PartialGitHubContext(t *testing.T) {
	// Token without PR number falls through to git provider (needs BaseRef).
	tests := []struct {
		name    string
		opts    ProviderOptions
		wantErr bool
	}{
		{
			name: "token only, no PR number, with base ref",
			opts: ProviderOptions{
				GitHubToken: "ghp_test",
				BaseRef:     "origin/main",
			},
			wantErr: false,
		},
		{
			name: "token only, no PR number, no base ref",
			opts: ProviderOptions{
				GitHubToken: "ghp_test",
			},
			wantErr: true,
		},
		{
			name: "PR number only, no token, with base ref",
			opts: ProviderOptions{
				PRNumber: 42,
				BaseRef:  "origin/main",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewChangedFilesProvider(tt.opts)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				require.NoError(t, err)
				_, ok := provider.(*GitDiffProvider)
				assert.True(t, ok, "expected GitDiffProvider for partial GitHub context")
			}
		})
	}
}

func TestHasGitHubPRContext(t *testing.T) {
	tests := []struct {
		name string
		opts ProviderOptions
		want bool
	}{
		{
			name: "full context",
			opts: ProviderOptions{GitHubToken: "tok", Owner: "o", Repo: "r", PRNumber: 1},
			want: true,
		},
		{
			name: "missing token",
			opts: ProviderOptions{Owner: "o", Repo: "r", PRNumber: 1},
			want: false,
		},
		{
			name: "missing PR number",
			opts: ProviderOptions{GitHubToken: "tok", Owner: "o", Repo: "r"},
			want: false,
		},
		{
			name: "zero PR number",
			opts: ProviderOptions{GitHubToken: "tok", Owner: "o", Repo: "r", PRNumber: 0},
			want: false,
		},
		{
			name: "missing owner and repo",
			opts: ProviderOptions{GitHubToken: "tok", PRNumber: 1},
			want: false,
		},
		{
			name: "owner only (no repo)",
			opts: ProviderOptions{GitHubToken: "tok", Owner: "o", PRNumber: 1},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hasGitHubPRContext(tt.opts))
		})
	}
}

func TestParseOwnerRepo(t *testing.T) {
	tests := []struct {
		name      string
		opts      ProviderOptions
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "both set",
			opts:      ProviderOptions{Owner: "shivasurya", Repo: "code-pathfinder"},
			wantOwner: "shivasurya",
			wantRepo:  "code-pathfinder",
		},
		{
			name:    "owner missing",
			opts:    ProviderOptions{Repo: "code-pathfinder"},
			wantErr: true,
		},
		{
			name:    "repo missing",
			opts:    ProviderOptions{Owner: "shivasurya"},
			wantErr: true,
		},
		{
			name:    "both missing",
			opts:    ProviderOptions{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseOwnerRepo(tt.opts)
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
