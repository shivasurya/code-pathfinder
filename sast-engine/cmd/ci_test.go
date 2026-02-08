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
