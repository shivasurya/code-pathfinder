package diff

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateGitRef_ValidHEAD(t *testing.T) {
	// HEAD is always valid in a git repo with at least one commit.
	dir := setupTestRepo(t)
	err := ValidateGitRef(dir, "HEAD")
	assert.NoError(t, err)
}

func TestValidateGitRef_ValidBranch(t *testing.T) {
	// A branch name that exists should be valid.
	dir := setupTestRepo(t)
	err := ValidateGitRef(dir, "main")
	assert.NoError(t, err)
}

func TestValidateGitRef_ValidCommitSHA(t *testing.T) {
	// A full or short commit SHA should be valid.
	dir := setupTestRepo(t)

	// Get the current commit SHA.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	require.NoError(t, err)

	sha := string(output)[:12] // Short SHA.
	err = ValidateGitRef(dir, sha)
	assert.NoError(t, err)
}

func TestValidateGitRef_ValidRelativeRef(t *testing.T) {
	// HEAD~0 is valid (same as HEAD).
	dir := setupTestRepo(t)
	err := ValidateGitRef(dir, "HEAD~0")
	assert.NoError(t, err)
}

func TestValidateGitRef_InvalidBranch(t *testing.T) {
	// A non-existent branch should fail.
	dir := setupTestRepo(t)

	err := ValidateGitRef(dir, "nonexistent-branch-xyz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid git ref 'nonexistent-branch-xyz'")
	assert.Contains(t, err.Error(), "fetch-depth: 0")
}

func TestValidateGitRef_InvalidRef(t *testing.T) {
	// A completely invalid ref should fail.
	dir := setupTestRepo(t)

	err := ValidateGitRef(dir, "origin/nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid git ref")
}

func TestValidateGitRef_NotARepo(t *testing.T) {
	// A non-git directory should fail.
	dir := t.TempDir()

	err := ValidateGitRef(dir, "HEAD")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid git ref 'HEAD'")
}

func TestValidateGitRef_MultipleValidRefs(t *testing.T) {
	// Table-driven test for multiple valid ref types.
	dir := setupTestRepo(t)

	// Create a tag and branch using the shared helper.
	runGit(t, dir, "tag", "v1.0.0")
	runGit(t, dir, "branch", "develop")

	tests := []struct {
		name string
		ref  string
	}{
		{name: "HEAD", ref: "HEAD"},
		{name: "main branch", ref: "main"},
		{name: "develop branch", ref: "develop"},
		{name: "tag", ref: "v1.0.0"},
		{name: "HEAD~0", ref: "HEAD~0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitRef(dir, tt.ref)
			assert.NoError(t, err, "expected ref %q to be valid", tt.ref)
		})
	}
}

func TestValidateGitRef_MultipleInvalidRefs(t *testing.T) {
	// Table-driven test for multiple invalid ref types.
	dir := setupTestRepo(t)

	tests := []struct {
		name string
		ref  string
	}{
		{name: "nonexistent branch", ref: "nonexistent-branch"},
		{name: "nonexistent remote", ref: "origin/nonexistent"},
		{name: "nonexistent tag", ref: "v99.99.99"},
		{name: "impossible relative", ref: "HEAD~9999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGitRef(dir, tt.ref)
			assert.Error(t, err, "expected ref %q to be invalid", tt.ref)
			assert.Contains(t, err.Error(), "invalid git ref")
		})
	}
}

func TestValidateGitRef_EmptyRef(t *testing.T) {
	// Empty ref should fail (git rev-parse --verify "" fails).
	dir := setupTestRepo(t)

	err := ValidateGitRef(dir, "")
	assert.Error(t, err)
}
