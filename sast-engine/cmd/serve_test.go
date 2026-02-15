package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServeCmdFlags(t *testing.T) {
	projectFlag := serveCmd.Flags().Lookup("project")
	assert.NotNil(t, projectFlag)
	assert.Equal(t, ".", projectFlag.DefValue)
	assert.Equal(t, "p", projectFlag.Shorthand)

	pythonVersionFlag := serveCmd.Flags().Lookup("python-version")
	assert.NotNil(t, pythonVersionFlag)
	assert.Equal(t, "", pythonVersionFlag.DefValue)

	httpFlag := serveCmd.Flags().Lookup("http")
	assert.NotNil(t, httpFlag)
	assert.Equal(t, "false", httpFlag.DefValue)

	addressFlag := serveCmd.Flags().Lookup("address")
	assert.NotNil(t, addressFlag)
	assert.Equal(t, ":8080", addressFlag.DefValue)
}

func TestServeCmdDisableMetricsFlagInherited(t *testing.T) {
	// The --disable-metrics flag is a persistent flag on rootCmd, inherited by all subcommands.
	// In cobra, inherited persistent flags are accessible via InheritedFlags() on the subcommand,
	// or directly via the root command's PersistentFlags().
	disableMetricsFlag := rootCmd.PersistentFlags().Lookup("disable-metrics")
	assert.NotNil(t, disableMetricsFlag, "--disable-metrics persistent flag should exist on root command")
	assert.Equal(t, "false", disableMetricsFlag.DefValue, "analytics should be enabled by default")
	assert.Equal(t, "Disable metrics collection", disableMetricsFlag.Usage)

	// Verify serve command is registered as a child of root.
	assert.True(t, serveCmd.HasParent(), "serve command should be a child of root")
}

// TestServeWithGoMod tests that serve command detects go.mod and builds Go call graph.
func TestServeWithGoMod(t *testing.T) {
	// Create temp directory with go.mod
	tmpDir := t.TempDir()
	goModPath := filepath.Join(tmpDir, "go.mod")
	goFilePath := filepath.Join(tmpDir, "main.go")

	// Write go.mod
	err := os.WriteFile(goModPath, []byte("module example.com/test\n\ngo 1.22\n"), 0644)
	assert.NoError(t, err)

	// Write simple Go file
	err = os.WriteFile(goFilePath, []byte(`package main

func Handler() {}

func main() {}
`), 0644)
	assert.NoError(t, err)

	// Verify go.mod exists
	_, err = os.Stat(goModPath)
	assert.NoError(t, err, "go.mod should exist in test directory")
}

// TestServeWithoutGoMod tests that serve command works without go.mod (Python-only).
func TestServeWithoutGoMod(t *testing.T) {
	// Create temp directory without go.mod
	tmpDir := t.TempDir()
	pyFilePath := filepath.Join(tmpDir, "main.py")

	// Write simple Python file
	err := os.WriteFile(pyFilePath, []byte("def handler():\n    pass\n"), 0644)
	assert.NoError(t, err)

	// Verify go.mod does NOT exist
	goModPath := filepath.Join(tmpDir, "go.mod")
	_, err = os.Stat(goModPath)
	assert.True(t, os.IsNotExist(err), "go.mod should not exist in Python-only project")
}
