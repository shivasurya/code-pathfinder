package diff

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRepo creates a temporary git repository with an initial commit.
// Returns the path to the repository root.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Initialize repo.
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	// Create initial commit on main.
	writeFile(t, filepath.Join(dir, "README.md"), "# Test Repo")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial commit")
	runGit(t, dir, "branch", "-M", "main")

	return dir
}

// runGit executes a git command in the given directory.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE=2025-01-01T00:00:00Z",
		"GIT_COMMITTER_DATE=2025-01-01T00:00:00Z",
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, string(output))
}

// writeFile creates or overwrites a file with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestGitDiffProvider_SimpleChange(t *testing.T) {
	// Tests a simple linear change: add and modify files on a feature branch.
	dir := setupTestRepo(t)

	runGit(t, dir, "checkout", "-b", "feature")

	// Add a new file and modify existing one.
	writeFile(t, filepath.Join(dir, "new_file.py"), "print('hello')")
	writeFile(t, filepath.Join(dir, "README.md"), "# Updated")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add and modify files")

	provider := &GitDiffProvider{
		ProjectRoot: dir,
		BaseRef:     "main",
		HeadRef:     "HEAD",
	}

	files, err := provider.GetChangedFiles()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"README.md", "new_file.py"}, files)
}

func TestGitDiffProvider_BranchWithMergeBase(t *testing.T) {
	// Tests the merge-base scenario where main has advanced after feature branched.
	dir := setupTestRepo(t)

	// Create feature branch from main.
	runGit(t, dir, "checkout", "-b", "feature")
	writeFile(t, filepath.Join(dir, "feature.py"), "# feature code")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "feature work")

	// Go back to main and add more commits.
	runGit(t, dir, "checkout", "main")
	writeFile(t, filepath.Join(dir, "main_update.py"), "# main update")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "main advanced")

	// Switch back to feature.
	runGit(t, dir, "checkout", "feature")

	provider := &GitDiffProvider{
		ProjectRoot: dir,
		BaseRef:     "main",
		HeadRef:     "HEAD",
	}

	files, err := provider.GetChangedFiles()
	require.NoError(t, err)
	// Only feature.py should show up, not main_update.py.
	assert.Equal(t, []string{"feature.py"}, files)
}

func TestGitDiffProvider_DeletedFileExcluded(t *testing.T) {
	// Tests that deleted files are excluded (--diff-filter=ACMR does not include D).
	dir := setupTestRepo(t)

	// Add a file on main.
	writeFile(t, filepath.Join(dir, "to_delete.py"), "# will be deleted")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add file to delete")

	runGit(t, dir, "checkout", "-b", "feature")

	// Delete the file and add a new one.
	require.NoError(t, os.Remove(filepath.Join(dir, "to_delete.py")))
	writeFile(t, filepath.Join(dir, "new_file.py"), "# new")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "delete and add")

	provider := &GitDiffProvider{
		ProjectRoot: dir,
		BaseRef:     "main",
		HeadRef:     "HEAD",
	}

	files, err := provider.GetChangedFiles()
	require.NoError(t, err)
	// to_delete.py should NOT appear (deleted). Only new_file.py.
	assert.Equal(t, []string{"new_file.py"}, files)
}

func TestGitDiffProvider_EmptyDiff(t *testing.T) {
	// Tests when there are no changes between base and head.
	dir := setupTestRepo(t)

	provider := &GitDiffProvider{
		ProjectRoot: dir,
		BaseRef:     "HEAD",
		HeadRef:     "HEAD",
	}

	files, err := provider.GetChangedFiles()
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestGitDiffProvider_SubdirectoryFiles(t *testing.T) {
	// Tests that files in subdirectories have correct relative paths.
	dir := setupTestRepo(t)

	runGit(t, dir, "checkout", "-b", "feature")

	writeFile(t, filepath.Join(dir, "app", "views.py"), "# views")
	writeFile(t, filepath.Join(dir, "app", "models.py"), "# models")
	writeFile(t, filepath.Join(dir, "tests", "test_views.py"), "# tests")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add nested files")

	provider := &GitDiffProvider{
		ProjectRoot: dir,
		BaseRef:     "main",
		HeadRef:     "HEAD",
	}

	files, err := provider.GetChangedFiles()
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"app/models.py",
		"app/views.py",
		"tests/test_views.py",
	}, files)
}

func TestGitDiffProvider_InvalidBaseRef(t *testing.T) {
	// Tests error handling for a non-existent base ref.
	dir := setupTestRepo(t)

	provider := &GitDiffProvider{
		ProjectRoot: dir,
		BaseRef:     "nonexistent-branch",
		HeadRef:     "HEAD",
	}

	files, err := provider.GetChangedFiles()
	assert.Error(t, err)
	assert.Nil(t, files)
	assert.Contains(t, err.Error(), "failed to find merge-base")
}

func TestGitDiffProvider_InvalidProjectRoot(t *testing.T) {
	// Tests error handling for a non-repository directory.
	dir := t.TempDir() // Not a git repo.

	provider := &GitDiffProvider{
		ProjectRoot: dir,
		BaseRef:     "main",
		HeadRef:     "HEAD",
	}

	files, err := provider.GetChangedFiles()
	assert.Error(t, err)
	assert.Nil(t, files)
}

func TestGitDiffProvider_MultipleCommits(t *testing.T) {
	// Tests that multiple commits on a feature branch are handled correctly.
	dir := setupTestRepo(t)

	runGit(t, dir, "checkout", "-b", "feature")

	// Commit 1.
	writeFile(t, filepath.Join(dir, "file1.py"), "# first")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "commit 1")

	// Commit 2.
	writeFile(t, filepath.Join(dir, "file2.py"), "# second")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "commit 2")

	// Commit 3: modify file from commit 1.
	writeFile(t, filepath.Join(dir, "file1.py"), "# updated first")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "commit 3")

	provider := &GitDiffProvider{
		ProjectRoot: dir,
		BaseRef:     "main",
		HeadRef:     "HEAD",
	}

	files, err := provider.GetChangedFiles()
	require.NoError(t, err)
	// Both files should appear, even though file1.py was modified across two commits.
	assert.ElementsMatch(t, []string{"file1.py", "file2.py"}, files)
}

func TestGitDiffProvider_RenamedFile(t *testing.T) {
	// Tests that renamed files are included (R is in --diff-filter=ACMR).
	dir := setupTestRepo(t)

	// Add a file to rename later.
	writeFile(t, filepath.Join(dir, "old_name.py"), "# some content that is long enough for git to detect a rename")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add file to rename")

	runGit(t, dir, "checkout", "-b", "feature")

	// Rename the file.
	runGit(t, dir, "mv", "old_name.py", "new_name.py")
	runGit(t, dir, "commit", "-m", "rename file")

	provider := &GitDiffProvider{
		ProjectRoot: dir,
		BaseRef:     "main",
		HeadRef:     "HEAD",
	}

	files, err := provider.GetChangedFiles()
	require.NoError(t, err)
	assert.Contains(t, files, "new_name.py")
}

func TestParseFileList(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{
			name:   "normal output",
			input:  "file1.py\nfile2.py\ndir/file3.py\n",
			expect: []string{"file1.py", "file2.py", "dir/file3.py"},
		},
		{
			name:   "empty output",
			input:  "",
			expect: nil,
		},
		{
			name:   "whitespace only",
			input:  "  \n  \n",
			expect: nil,
		},
		{
			name:   "trailing newlines",
			input:  "file.py\n\n\n",
			expect: []string{"file.py"},
		},
		{
			name:   "single file",
			input:  "file.py",
			expect: []string{"file.py"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFileList(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}
