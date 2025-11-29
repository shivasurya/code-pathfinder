package cmd

import (
	"io"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestVersionCmd(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		gitCommit      string
		expectedOutput string
	}{
		{
			name:           "Default version and commit",
			version:        "0.0.24",
			gitCommit:      "HEAD",
			expectedOutput: "Version: 0.0.24\nGit Commit: HEAD\n",
		},
		{
			name:           "Custom version and commit",
			version:        "1.2.3",
			gitCommit:      "abc123",
			expectedOutput: "Version: 1.2.3\nGit Commit: abc123\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Version = tt.version
			GitCommit = tt.gitCommit

			rootCmd := &cobra.Command{Use: "pathfinder"}
			rootCmd.AddCommand(versionCmd)

			oldStdout := os.Stdout
			r, w, _ := os.Pipe() //nolint:all
			os.Stdout = w
			rootCmd.SetOut(w)
			rootCmd.SetArgs([]string{"version"})
			err := rootCmd.Execute()

			w.Close()
			out, _ := io.ReadAll(r) //nolint:all
			os.Stdout = oldStdout

			output := string(out)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			assert.Equal(t, tt.expectedOutput, output)
		})
	}
}

func TestVersionCmdRegistration(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"version"})
	assert.NoError(t, err)
	assert.NotNil(t, cmd)
	assert.Equal(t, "version", cmd.Name())
	assert.Equal(t, "Print the version and commit information", cmd.Short)
}
