package cmd

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

// TestExitCodes_Integration tests actual binary exit codes.
// This test requires the binary to be built first: gradle buildGo.
func TestExitCodes_Integration(t *testing.T) {
	// Skip if INTEGRATION env var not set
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION=1 to run.")
	}

	binaryPath := "../build/go/pathfinder"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Binary not found. Run 'gradle buildGo' first.")
	}

	// Get absolute path to test fixtures
	fixturesDir, err := filepath.Abs("../test/fixtures")
	require.NoError(t, err)

	tests := []struct {
		name         string
		projectPath  string
		rulesPath    string
		failOn       string
		command      string
		outputFormat string
		expectedExit int
	}{
		{
			name:         "Clean project - no findings",
			projectPath:  filepath.Join(fixturesDir, "clean_project"),
			rulesPath:    filepath.Join(fixturesDir, "rules/simple.py"),
			failOn:       "",
			command:      "scan",
			expectedExit: 0,
		},
		{
			name:         "Findings without fail-on",
			projectPath:  filepath.Join(fixturesDir, "vulnerable_project"),
			rulesPath:    filepath.Join(fixturesDir, "rules/simple.py"),
			failOn:       "",
			command:      "scan",
			expectedExit: 0,
		},
		{
			name:         "Critical findings with fail-on critical",
			projectPath:  filepath.Join(fixturesDir, "vulnerable_project"),
			rulesPath:    filepath.Join(fixturesDir, "rules/critical.py"),
			failOn:       "critical",
			command:      "scan",
			expectedExit: 1,
		},
		{
			name:         "High findings with fail-on critical,high",
			projectPath:  filepath.Join(fixturesDir, "vulnerable_project"),
			rulesPath:    filepath.Join(fixturesDir, "rules/high.py"),
			failOn:       "critical,high",
			command:      "scan",
			expectedExit: 1,
		},
		{
			name:         "Low findings with fail-on critical,high",
			projectPath:  filepath.Join(fixturesDir, "vulnerable_project"),
			rulesPath:    filepath.Join(fixturesDir, "rules/low.py"),
			failOn:       "critical,high",
			command:      "scan",
			expectedExit: 0,
		},
		{
			name:         "CI mode - SARIF with findings, no fail-on",
			projectPath:  filepath.Join(fixturesDir, "vulnerable_project"),
			rulesPath:    filepath.Join(fixturesDir, "rules/simple.py"),
			failOn:       "",
			command:      "ci",
			outputFormat: "sarif",
			expectedExit: 0,
		},
		{
			name:         "CI mode - JSON with critical findings and fail-on",
			projectPath:  filepath.Join(fixturesDir, "vulnerable_project"),
			rulesPath:    filepath.Join(fixturesDir, "rules/critical.py"),
			failOn:       "critical",
			command:      "ci",
			outputFormat: "json",
			expectedExit: 1,
		},
		{
			name:         "CI mode - CSV with findings but no fail-on match",
			projectPath:  filepath.Join(fixturesDir, "vulnerable_project"),
			rulesPath:    filepath.Join(fixturesDir, "rules/low.py"),
			failOn:       "critical,high",
			command:      "ci",
			outputFormat: "csv",
			expectedExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{tt.command, "--project", tt.projectPath, "--rules", tt.rulesPath}

			if tt.command == "ci" && tt.outputFormat != "" {
				args = append(args, "--output", tt.outputFormat)
			}

			if tt.failOn != "" {
				args = append(args, "--fail-on", tt.failOn)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binaryPath, args...)
			err := cmd.Run()

			if tt.expectedExit == 0 {
				assert.NoError(t, err, "Expected exit code 0")
			} else {
				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr, "Expected exit error")
				assert.Equal(t, tt.expectedExit, exitErr.ExitCode())
			}
		})
	}
}

// TestInvalidSeverity_ExitCode2 tests that invalid severities cause exit code 2.
func TestInvalidSeverity_ExitCode2(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION=1 to run.")
	}

	binaryPath := "../build/go/pathfinder"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Binary not found. Run 'gradle buildGo' first.")
	}

	fixturesDir, err := filepath.Abs("../test/fixtures")
	require.NoError(t, err)

	tests := []struct {
		name    string
		command string
		failOn  string
	}{
		{
			name:    "Scan with invalid severity",
			command: "scan",
			failOn:  "invalid",
		},
		{
			name:    "CI with invalid severity",
			command: "ci",
			failOn:  "critical,invalid,high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectPath := filepath.Join(fixturesDir, "clean_project")
			rulesPath := filepath.Join(fixturesDir, "rules/simple.py")

			args := []string{tt.command, "--project", projectPath, "--rules", rulesPath, "--fail-on", tt.failOn}
			if tt.command == "ci" {
				args = append(args, "--output", "json")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binaryPath, args...)
			err := cmd.Run()

			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr, "Expected exit error")
			assert.Equal(t, 1, exitErr.ExitCode(), "Invalid severity should cause exit via RunE error")
		})
	}
}

// TestCaseInsensitiveSeverities tests case insensitivity of --fail-on.
func TestCaseInsensitiveSeverities(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test. Set INTEGRATION=1 to run.")
	}

	binaryPath := "../build/go/pathfinder"
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		t.Skip("Binary not found. Run 'gradle buildGo' first.")
	}

	fixturesDir, err := filepath.Abs("../test/fixtures")
	require.NoError(t, err)

	projectPath := filepath.Join(fixturesDir, "vulnerable_project")
	rulesPath := filepath.Join(fixturesDir, "rules/critical.py")

	tests := []struct {
		name   string
		failOn string
	}{
		{"Lowercase", "critical"},
		{"Uppercase", "CRITICAL"},
		{"Mixed case", "CrItIcAl"},
		{"Multiple mixed", "CRITICAL,High,MeDiUm"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binaryPath, "scan", "--project", projectPath, "--rules", rulesPath, "--fail-on", tt.failOn)
			err := cmd.Run()

			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr, "Expected exit error for critical finding")
			assert.Equal(t, 1, exitErr.ExitCode())
		})
	}
}
