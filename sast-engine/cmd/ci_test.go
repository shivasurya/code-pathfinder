package cmd

import (
	"os"
	"path/filepath"
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
		ciCmd.Flags().Set("output-file", "")
		ciCmd.Flags().Set("verbose", "false")
		ciCmd.Flags().Set("debug", "false")
		ciCmd.Flags().Set("fail-on", "")
		ciCmd.Flags().Set("skip-tests", "true")
		ciCmd.Flags().Set("refresh-rules", "false")
		ciCmd.Flags().Set("base", "")
		ciCmd.Flags().Set("head", "HEAD")
		ciCmd.Flags().Set("no-diff", "true") // disable diff to avoid git calls
		ciCmd.Flags().Set("github-token", "")
		ciCmd.Flags().Set("github-repo", "")
		ciCmd.Flags().Set("github-pr", "0")
		ciCmd.Flags().Set("pr-comment", "false")
		ciCmd.Flags().Set("pr-inline", "false")
	}

	t.Run("missing rules and ruleset returns error", func(t *testing.T) {
		resetFlags()
		ciCmd.Flags().Set("project", "/tmp/test-project")
		err := ciCmd.RunE(ciCmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "either --rules or --ruleset flag is required")
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

func TestCICommandRulesetFlags(t *testing.T) {
	tests := []struct {
		name     string
		flag     string
		defValue string
	}{
		{name: "ruleset", flag: "ruleset", defValue: "[]"},
		{name: "refresh-rules", flag: "refresh-rules", defValue: "false"},
		{name: "output-file", flag: "output-file", defValue: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := ciCmd.Flags().Lookup(tt.flag)
			require.NotNil(t, flag, "flag %q should be registered on ci command", tt.flag)
			assert.Equal(t, tt.defValue, flag.DefValue)
		})
	}
}

func TestCICommandRulesNotRequired(t *testing.T) {
	// Verify --rules is no longer marked as required (--ruleset can be used instead).
	rulesFlag := ciCmd.Flags().Lookup("rules")
	require.NotNil(t, rulesFlag)

	// The flag should not have the "required" annotation.
	annotations := rulesFlag.Annotations
	_, isRequired := annotations["cobra_annotation_bash_completion_one_required_flags"]
	assert.False(t, isRequired, "--rules should not be marked as required")
}

// --- Integration tests for CI pipeline paths ---

// setupCIIntegrationTest creates a minimal project and rules file for integration tests.
// Returns (projectDir, rulesFile) paths. Cleanup is handled by t.TempDir().
func setupCIIntegrationTest(t *testing.T) (string, string) {
	t.Helper()

	// Create temp project with a simple Python file
	projectDir := t.TempDir()
	err := os.WriteFile(filepath.Join(projectDir, "app.py"), []byte("def hello():\n    print('hello')\n"), 0644)
	require.NoError(t, err)

	// Create a minimal rules file (no @Rule decorators = 0 rules loaded, no error)
	rulesDir := t.TempDir()
	rulesFile := filepath.Join(rulesDir, "empty_rules.py")
	err = os.WriteFile(rulesFile, []byte("# No rules defined\n"), 0644)
	require.NoError(t, err)

	return projectDir, rulesFile
}

// resetCIFlags restores all ci flags to their defaults.
func resetCIFlags() {
	ciCmd.Flags().Set("rules", "")
	ciCmd.Flags().Set("project", "")
	ciCmd.Flags().Set("output", "sarif")
	ciCmd.Flags().Set("output-file", "")
	ciCmd.Flags().Set("verbose", "false")
	ciCmd.Flags().Set("debug", "false")
	ciCmd.Flags().Set("fail-on", "")
	ciCmd.Flags().Set("skip-tests", "true")
	ciCmd.Flags().Set("refresh-rules", "false")
	ciCmd.Flags().Set("base", "")
	ciCmd.Flags().Set("head", "HEAD")
	ciCmd.Flags().Set("no-diff", "true")
	ciCmd.Flags().Set("github-token", "")
	ciCmd.Flags().Set("github-repo", "")
	ciCmd.Flags().Set("github-pr", "0")
	ciCmd.Flags().Set("pr-comment", "false")
	ciCmd.Flags().Set("pr-inline", "false")
}

func TestCICmdOutputFileSARIF(t *testing.T) {
	projectDir, rulesFile := setupCIIntegrationTest(t)
	outputFile := filepath.Join(t.TempDir(), "results.sarif")

	resetCIFlags()
	ciCmd.Flags().Set("rules", rulesFile)
	ciCmd.Flags().Set("project", projectDir)
	ciCmd.Flags().Set("output", "sarif")
	ciCmd.Flags().Set("output-file", outputFile)

	err := ciCmd.RunE(ciCmd, []string{})
	require.NoError(t, err)

	// Verify output file was created and contains valid SARIF
	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"version": "2.1.0"`)
	assert.Contains(t, string(data), "Code Pathfinder")
}

func TestCICmdOutputFileJSON(t *testing.T) {
	projectDir, rulesFile := setupCIIntegrationTest(t)
	outputFile := filepath.Join(t.TempDir(), "results.json")

	resetCIFlags()
	ciCmd.Flags().Set("rules", rulesFile)
	ciCmd.Flags().Set("project", projectDir)
	ciCmd.Flags().Set("output", "json")
	ciCmd.Flags().Set("output-file", outputFile)

	err := ciCmd.RunE(ciCmd, []string{})
	require.NoError(t, err)

	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "results")
}

func TestCICmdOutputFileCSV(t *testing.T) {
	projectDir, rulesFile := setupCIIntegrationTest(t)
	outputFile := filepath.Join(t.TempDir(), "results.csv")

	resetCIFlags()
	ciCmd.Flags().Set("rules", rulesFile)
	ciCmd.Flags().Set("project", projectDir)
	ciCmd.Flags().Set("output", "csv")
	ciCmd.Flags().Set("output-file", outputFile)

	err := ciCmd.RunE(ciCmd, []string{})
	require.NoError(t, err)

	data, err := os.ReadFile(outputFile)
	require.NoError(t, err)
	// CSV formatter writes header even with 0 findings
	assert.True(t, len(data) > 0)
}

func TestCICmdStdoutSARIF(t *testing.T) {
	// Test SARIF output to stdout (no --output-file)
	projectDir, rulesFile := setupCIIntegrationTest(t)

	resetCIFlags()
	ciCmd.Flags().Set("rules", rulesFile)
	ciCmd.Flags().Set("project", projectDir)
	ciCmd.Flags().Set("output", "sarif")

	err := ciCmd.RunE(ciCmd, []string{})
	require.NoError(t, err)
}

func TestCICmdOutputFileCreationError(t *testing.T) {
	projectDir, rulesFile := setupCIIntegrationTest(t)

	resetCIFlags()
	ciCmd.Flags().Set("rules", rulesFile)
	ciCmd.Flags().Set("project", projectDir)
	ciCmd.Flags().Set("output", "sarif")
	ciCmd.Flags().Set("output-file", "/nonexistent/dir/results.sarif")

	err := ciCmd.RunE(ciCmd, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create output file")
}

func TestCICmdPrepareRulesLocalOnly(t *testing.T) {
	// Verify prepareRules passes through with local-only rules (no --ruleset)
	projectDir, rulesFile := setupCIIntegrationTest(t)
	outputFile := filepath.Join(t.TempDir(), "results.sarif")

	resetCIFlags()
	ciCmd.Flags().Set("rules", rulesFile)
	ciCmd.Flags().Set("project", projectDir)
	ciCmd.Flags().Set("output-file", outputFile)

	err := ciCmd.RunE(ciCmd, []string{})
	require.NoError(t, err)

	// Verify output was written (proves prepareRules succeeded and pipeline completed)
	_, err = os.Stat(outputFile)
	require.NoError(t, err)
}
