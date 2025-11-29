package cmd

import (
	"testing"
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
