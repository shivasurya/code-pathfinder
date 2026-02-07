package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGitHubRepo(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantOwner string
		wantRepo  string
	}{
		{
			name:      "valid owner/repo",
			input:     "shivasurya/code-pathfinder",
			wantOwner: "shivasurya",
			wantRepo:  "code-pathfinder",
		},
		{
			name:      "valid with dots and hyphens",
			input:     "my-org/my.repo.name",
			wantOwner: "my-org",
			wantRepo:  "my.repo.name",
		},
		{
			name:      "empty string",
			input:     "",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "no slash",
			input:     "noslash",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "empty owner",
			input:     "/repo",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "empty repo",
			input:     "owner/",
			wantOwner: "",
			wantRepo:  "",
		},
		{
			name:      "multiple slashes keeps rest in repo",
			input:     "owner/repo/extra",
			wantOwner: "owner",
			wantRepo:  "repo/extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo := parseGitHubRepo(tt.input)
			assert.Equal(t, tt.wantOwner, owner)
			assert.Equal(t, tt.wantRepo, repo)
		})
	}
}

func TestResolveBaseRef(t *testing.T) {
	// All env vars that resolveBaseRef checks.
	envKeys := []string{
		"GITHUB_BASE_REF",
		"CI_MERGE_REQUEST_TARGET_BRANCH_NAME",
		"PATHFINDER_BASELINE_REF",
	}

	tests := []struct {
		name    string
		envVars map[string]string
		want    string
	}{
		{
			name:    "GITHUB_BASE_REF set",
			envVars: map[string]string{"GITHUB_BASE_REF": "main"},
			want:    "origin/main",
		},
		{
			name:    "CI_MERGE_REQUEST_TARGET_BRANCH_NAME set",
			envVars: map[string]string{"CI_MERGE_REQUEST_TARGET_BRANCH_NAME": "develop"},
			want:    "origin/develop",
		},
		{
			name:    "PATHFINDER_BASELINE_REF set",
			envVars: map[string]string{"PATHFINDER_BASELINE_REF": "abc123"},
			want:    "abc123",
		},
		{
			name:    "no env vars set",
			envVars: map[string]string{},
			want:    "",
		},
		{
			name: "GITHUB_BASE_REF takes priority over GitLab CI",
			envVars: map[string]string{
				"GITHUB_BASE_REF":                      "main",
				"CI_MERGE_REQUEST_TARGET_BRANCH_NAME":  "develop",
				"PATHFINDER_BASELINE_REF":              "custom",
			},
			want: "origin/main",
		},
		{
			name: "GitLab CI takes priority over PATHFINDER_BASELINE_REF",
			envVars: map[string]string{
				"CI_MERGE_REQUEST_TARGET_BRANCH_NAME": "develop",
				"PATHFINDER_BASELINE_REF":             "custom",
			},
			want: "origin/develop",
		},
		{
			name: "PATHFINDER_BASELINE_REF returns raw value without origin prefix",
			envVars: map[string]string{
				"PATHFINDER_BASELINE_REF": "origin/main",
			},
			want: "origin/main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all relevant env vars.
			for _, key := range envKeys {
				t.Setenv(key, "")
			}
			// Set test-specific env vars.
			for key, val := range tt.envVars {
				t.Setenv(key, val)
			}

			got := resolveBaseRef()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestApplyDiffFilter(t *testing.T) {
	logger := output.NewLogger(output.VerbosityDefault)

	t.Run("filters detections to changed files", func(t *testing.T) {
		detections := []*dsl.EnrichedDetection{
			{Location: dsl.LocationInfo{RelPath: "app/views.py"}, Rule: dsl.RuleMetadata{Severity: "critical"}},
			{Location: dsl.LocationInfo{RelPath: "app/models.py"}, Rule: dsl.RuleMetadata{Severity: "high"}},
			{Location: dsl.LocationInfo{RelPath: "app/auth.py"}, Rule: dsl.RuleMetadata{Severity: "medium"}},
		}
		changedFiles := []string{"app/views.py", "app/auth.py"}

		result := applyDiffFilter(detections, changedFiles, logger)

		require.Len(t, result, 2)
		assert.Equal(t, "app/views.py", result[0].Location.RelPath)
		assert.Equal(t, "app/auth.py", result[1].Location.RelPath)
	})

	t.Run("returns all when changed files is empty", func(t *testing.T) {
		detections := []*dsl.EnrichedDetection{
			{Location: dsl.LocationInfo{RelPath: "app/views.py"}, Rule: dsl.RuleMetadata{Severity: "critical"}},
		}

		result := applyDiffFilter(detections, []string{}, logger)

		assert.Len(t, result, 1)
	})

	t.Run("returns empty when no detections match", func(t *testing.T) {
		detections := []*dsl.EnrichedDetection{
			{Location: dsl.LocationInfo{RelPath: "app/views.py"}, Rule: dsl.RuleMetadata{Severity: "critical"}},
		}
		changedFiles := []string{"other/file.py"}

		result := applyDiffFilter(detections, changedFiles, logger)

		assert.Empty(t, result)
	})

	t.Run("handles nil detections", func(t *testing.T) {
		result := applyDiffFilter(nil, []string{"app/views.py"}, logger)

		assert.Empty(t, result)
	})

	t.Run("preserves detection order", func(t *testing.T) {
		detections := []*dsl.EnrichedDetection{
			{Location: dsl.LocationInfo{RelPath: "z.py"}, Rule: dsl.RuleMetadata{Severity: "low"}},
			{Location: dsl.LocationInfo{RelPath: "a.py"}, Rule: dsl.RuleMetadata{Severity: "high"}},
			{Location: dsl.LocationInfo{RelPath: "m.py"}, Rule: dsl.RuleMetadata{Severity: "medium"}},
		}
		changedFiles := []string{"z.py", "a.py", "m.py"}

		result := applyDiffFilter(detections, changedFiles, logger)

		require.Len(t, result, 3)
		assert.Equal(t, "z.py", result[0].Location.RelPath)
		assert.Equal(t, "a.py", result[1].Location.RelPath)
		assert.Equal(t, "m.py", result[2].Location.RelPath)
	})
}

// setupDiffTestRepo creates a minimal git repo with a main branch and feature branch.
// Returns the repo directory path.
func setupDiffTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, out)
	}

	run("init", "--initial-branch=main")
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")

	// Initial commit on main.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "existing.py"), []byte("# existing"), 0644))
	run("add", "existing.py")
	run("commit", "-m", "initial commit")

	// Feature branch with new file.
	run("checkout", "-b", "feature")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "new_file.py"), []byte("# new"), 0644))
	run("add", "new_file.py")
	run("commit", "-m", "add new file")

	return dir
}

func TestComputeChangedFiles(t *testing.T) {
	logger := output.NewLogger(output.VerbosityDefault)

	t.Run("returns error for empty base ref", func(t *testing.T) {
		ghOpts := githubOptions{}
		_, err := computeChangedFiles("", "HEAD", "/tmp", ghOpts, logger)
		assert.Error(t, err)
	})

	t.Run("returns error for nonexistent project root", func(t *testing.T) {
		ghOpts := githubOptions{}
		_, err := computeChangedFiles("main", "HEAD", "/nonexistent/path/xyz", ghOpts, logger)
		assert.Error(t, err)
	})

	t.Run("computes changed files with real git repo", func(t *testing.T) {
		dir := setupDiffTestRepo(t)
		ghOpts := githubOptions{}

		files, err := computeChangedFiles("main", "feature", dir, ghOpts, logger)
		require.NoError(t, err)
		assert.Equal(t, []string{"new_file.py"}, files)
	})

	t.Run("returns empty for no changes", func(t *testing.T) {
		dir := setupDiffTestRepo(t)
		ghOpts := githubOptions{}

		// Compare main with itself — no changes.
		files, err := computeChangedFiles("main", "main", dir, ghOpts, logger)
		require.NoError(t, err)
		assert.Empty(t, files)
	})
}

func TestCICommandDiffFlags(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		defValue string
	}{
		{name: "base", flag: "base", defValue: ""},
		{name: "head", flag: "head", defValue: "HEAD"},
		{name: "no-diff", flag: "no-diff", defValue: "false"},
		{name: "github-token", flag: "github-token", defValue: ""},
		{name: "github-repo", flag: "github-repo", defValue: ""},
		{name: "github-pr", flag: "github-pr", defValue: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := ciCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, flag, "flag %q should be registered on ci command", tt.flag)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestScanCommandDiffFlags(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		defValue string
	}{
		{name: "diff-aware", flag: "diff-aware", defValue: "false"},
		{name: "base", flag: "base", defValue: ""},
		{name: "head", flag: "head", defValue: "HEAD"},
		{name: "github-token", flag: "github-token", defValue: ""},
		{name: "github-repo", flag: "github-repo", defValue: ""},
		{name: "github-pr", flag: "github-pr", defValue: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := scanCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, flag, "flag %q should be registered on scan command", tt.flag)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestGithubOptions(t *testing.T) {
	// Verify that githubOptions correctly stores all fields.
	opts := githubOptions{
		Token:    "ghp_test123",
		Owner:    "testowner",
		Repo:     "testrepo",
		PRNumber: 42,
	}
	assert.Equal(t, "ghp_test123", opts.Token)
	assert.Equal(t, "testowner", opts.Owner)
	assert.Equal(t, "testrepo", opts.Repo)
	assert.Equal(t, 42, opts.PRNumber)
}
