package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCiCmd(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedOutput string
	}{
		{
			name:           "Basic CI command",
			args:           []string{"ci"},
			expectedOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "pathfinder"}
			cmd.AddCommand(ciCmd)

			b := bytes.NewBufferString("")
			cmd.SetOut(b)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, b.String())
		})
	}
}

func TestCiCmdFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "pathfinder"}
	cmd.AddCommand(ciCmd)

	assert.NotNil(t, ciCmd)
	assert.Equal(t, "ci", ciCmd.Use)
	assert.Equal(t, "Scan a project for vulnerabilities with ruleset in ci mode", ciCmd.Short)
}

func TestCiCmdAddedToRootCmd(t *testing.T) {
	foundCiCmd := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "ci" {
			foundCiCmd = true
			break
		}
	}
	assert.True(t, foundCiCmd, "ci command should be added to root command")
}
