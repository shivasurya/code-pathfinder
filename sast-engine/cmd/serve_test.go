package cmd

import (
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
