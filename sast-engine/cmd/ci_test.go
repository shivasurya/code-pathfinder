package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSARIFOutput(t *testing.T) {
	t.Skip("Skipping: generateSARIFOutput replaced with output.SARIFFormatter in PR #5")
	// All tests below are obsolete as the function has been replaced with output.SARIFFormatter
	// See output/sarif_formatter_test.go for comprehensive tests of the new implementation
}

func TestGenerateJSONOutput(t *testing.T) {
	t.Skip("Skipping: generateJSONOutput replaced with output.JSONFormatter in PR #4")
	// All tests below are obsolete as the function has been replaced with output.JSONFormatter
	// See output/json_formatter_test.go for comprehensive tests of the new implementation
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

// TestCICmdValidation tests the RunE validation paths in the ci command.
func TestCICmdValidation(t *testing.T) {
	// resetFlags restores all ci flags to their defaults before each subtest.
	resetFlags := func() {
		ciCmd.Flags().Set("rules", "")
		ciCmd.Flags().Set("project", "")
		ciCmd.Flags().Set("output", "sarif")
		ciCmd.Flags().Set("verbose", "false")
		ciCmd.Flags().Set("debug", "false")
		ciCmd.Flags().Set("fail-on", "")
		ciCmd.Flags().Set("skip-tests", "true")
		ciCmd.Flags().Set("base", "")
		ciCmd.Flags().Set("head", "HEAD")
		ciCmd.Flags().Set("no-diff", "true") // disable diff to avoid git calls
		ciCmd.Flags().Set("github-token", "")
		ciCmd.Flags().Set("github-repo", "")
		ciCmd.Flags().Set("github-pr", "0")
		ciCmd.Flags().Set("pr-comment", "false")
		ciCmd.Flags().Set("pr-inline", "false")
	}

	t.Run("missing rules returns error", func(t *testing.T) {
		resetFlags()
		ciCmd.Flags().Set("project", "/tmp/test-project")
		err := ciCmd.RunE(ciCmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--rules flag is required")
	})

	t.Run("missing project returns error", func(t *testing.T) {
		resetFlags()
		ciCmd.Flags().Set("rules", "/tmp/test-rules.py")
		err := ciCmd.RunE(ciCmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--project flag is required")
	})

	t.Run("invalid output format returns error", func(t *testing.T) {
		resetFlags()
		ciCmd.Flags().Set("rules", "/tmp/test-rules.py")
		ciCmd.Flags().Set("project", "/tmp/test-project")
		ciCmd.Flags().Set("output", "xml")
		err := ciCmd.RunE(ciCmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--output must be")
	})

	t.Run("pr-comment without github-token returns error", func(t *testing.T) {
		resetFlags()
		ciCmd.Flags().Set("rules", "/tmp/test-rules.py")
		ciCmd.Flags().Set("project", "/tmp/test-project")
		ciCmd.Flags().Set("pr-comment", "true")
		err := ciCmd.RunE(ciCmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--github-token is required")
	})

	t.Run("pr-comment without github-repo returns error", func(t *testing.T) {
		resetFlags()
		ciCmd.Flags().Set("rules", "/tmp/test-rules.py")
		ciCmd.Flags().Set("project", "/tmp/test-project")
		ciCmd.Flags().Set("pr-comment", "true")
		ciCmd.Flags().Set("github-token", "test-token")
		err := ciCmd.RunE(ciCmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--github-repo is required")
	})

	t.Run("pr-comment with invalid pr number returns error", func(t *testing.T) {
		resetFlags()
		ciCmd.Flags().Set("rules", "/tmp/test-rules.py")
		ciCmd.Flags().Set("project", "/tmp/test-project")
		ciCmd.Flags().Set("pr-comment", "true")
		ciCmd.Flags().Set("github-token", "test-token")
		ciCmd.Flags().Set("github-repo", "owner/repo")
		ciCmd.Flags().Set("github-pr", "0")
		err := ciCmd.RunE(ciCmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--github-pr must be a positive number")
	})

	t.Run("pr-comment with invalid repo format returns error", func(t *testing.T) {
		resetFlags()
		ciCmd.Flags().Set("rules", "/tmp/test-rules.py")
		ciCmd.Flags().Set("project", "/tmp/test-project")
		ciCmd.Flags().Set("pr-comment", "true")
		ciCmd.Flags().Set("github-token", "test-token")
		ciCmd.Flags().Set("github-repo", "invalidrepo")
		ciCmd.Flags().Set("github-pr", "1")
		err := ciCmd.RunE(ciCmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "owner/repo")
	})

	t.Run("pr-inline without github-token returns error", func(t *testing.T) {
		resetFlags()
		ciCmd.Flags().Set("rules", "/tmp/test-rules.py")
		ciCmd.Flags().Set("project", "/tmp/test-project")
		ciCmd.Flags().Set("pr-inline", "true")
		err := ciCmd.RunE(ciCmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--github-token is required")
	})

	t.Run("pr-inline with valid flags and invalid repo format", func(t *testing.T) {
		resetFlags()
		ciCmd.Flags().Set("rules", "/tmp/test-rules.py")
		ciCmd.Flags().Set("project", "/tmp/test-project")
		ciCmd.Flags().Set("pr-inline", "true")
		ciCmd.Flags().Set("github-token", "test-token")
		ciCmd.Flags().Set("github-repo", "bad")
		ciCmd.Flags().Set("github-pr", "5")
		err := ciCmd.RunE(ciCmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "owner/repo")
	})

	t.Run("both pr-comment and pr-inline without token", func(t *testing.T) {
		resetFlags()
		ciCmd.Flags().Set("rules", "/tmp/test-rules.py")
		ciCmd.Flags().Set("project", "/tmp/test-project")
		ciCmd.Flags().Set("pr-comment", "true")
		ciCmd.Flags().Set("pr-inline", "true")
		err := ciCmd.RunE(ciCmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "--github-token is required")
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := ciCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, flag, "flag %q should be registered on ci command", tt.flag)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}
