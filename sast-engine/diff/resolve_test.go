package diff

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveBaseRef(t *testing.T) {
	// All env vars that ResolveBaseRef checks.
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
				"GITHUB_BASE_REF":                     "main",
				"CI_MERGE_REQUEST_TARGET_BRANCH_NAME": "develop",
				"PATHFINDER_BASELINE_REF":             "custom",
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

			got := ResolveBaseRef()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestComputeChangedFiles(t *testing.T) {
	t.Run("returns error for empty base ref", func(t *testing.T) {
		_, err := ComputeChangedFiles("", "HEAD", "/tmp")
		assert.Error(t, err)
	})

	t.Run("returns error for nonexistent project root", func(t *testing.T) {
		_, err := ComputeChangedFiles("main", "HEAD", "/nonexistent/path/xyz")
		assert.Error(t, err)
	})

	t.Run("computes changed files with real git repo", func(t *testing.T) {
		dir := setupTestRepo(t)

		// Create feature branch with a new file.
		runGit(t, dir, "checkout", "-b", "feature")
		writeFile(t, filepath.Join(dir, "new_file.py"), "# new")
		runGit(t, dir, "add", "new_file.py")
		runGit(t, dir, "commit", "-m", "add new file")

		files, err := ComputeChangedFiles("main", "feature", dir)
		require.NoError(t, err)
		assert.Equal(t, []string{"new_file.py"}, files)
	})

	t.Run("returns empty for no changes", func(t *testing.T) {
		dir := setupTestRepo(t)

		// Compare main with itself — no changes.
		files, err := ComputeChangedFiles("main", "main", dir)
		require.NoError(t, err)
		assert.Empty(t, files)
	})
}
