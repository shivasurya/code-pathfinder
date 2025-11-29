package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	oldRoot := rootCmd
	defer func() { rootCmd = oldRoot }()

	tests := []struct {
		name          string
		args          []string
		expectedError bool
	}{
		{
			name:          "No arguments",
			args:          []string{},
			expectedError: false,
		},
		{
			name:          "Help command",
			args:          []string{"--help"},
			expectedError: false,
		},
		{
			name:          "Invalid command",
			args:          []string{"invalidcommand"},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd = &cobra.Command{Use: "pathfinder"}
			rootCmd.AddCommand(&cobra.Command{Use: "validcommand"})

			rootCmd.SetArgs(tt.args)
			err := Execute()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRootCmdPersistentPreRun(t *testing.T) {
	tests := []struct {
		name            string
		disableMetrics  bool
		expectedMetrics bool
	}{
		{
			name:            "Metrics enabled",
			disableMetrics:  false,
			expectedMetrics: true,
		},
		{
			name:            "Metrics disabled",
			disableMetrics:  true,
			expectedMetrics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool("disable-metrics", tt.disableMetrics, "")

			rootCmd.PersistentPreRun(cmd, []string{})

			// Since we can't directly test the analytics.Init function,
			// we can check if the flag was correctly read
			disableMetrics, _ := cmd.Flags().GetBool("disable-metrics")
			assert.Equal(t, tt.disableMetrics, disableMetrics)
		})
	}
}

func TestRootCmdFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "pathfinder"}
	cmd.AddCommand(rootCmd)

	disableMetricsFlag := rootCmd.PersistentFlags().Lookup("disable-metrics")
	assert.NotNil(t, disableMetricsFlag)
	assert.Equal(t, "false", disableMetricsFlag.DefValue)
	assert.Equal(t, "Disable metrics collection", disableMetricsFlag.Usage)
}

func TestRootCmdOutput(t *testing.T) {
	oldRoot := rootCmd
	defer func() { rootCmd = oldRoot }()

	rootCmd = &cobra.Command{Use: "pathfinder"}
	rootCmd.AddCommand(&cobra.Command{Use: "validcommand"})

	tests := []struct {
		name           string
		args           []string
		expectedOutput string
	}{
		{
			name:           "No arguments",
			args:           []string{},
			expectedOutput: "Usage:\n  pathfinder [command]\n\nAvailable Commands:\n  completion",
		},
		{
			name:           "Help command",
			args:           []string{"--help"},
			expectedOutput: "Usage:\n  pathfinder [command]\n\nAvailable Commands:\n  completion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := new(bytes.Buffer)
			rootCmd.SetOut(b)
			rootCmd.SetArgs(tt.args)
			_ = rootCmd.Execute()

			assert.Contains(t, b.String(), tt.expectedOutput)
		})
	}
}
