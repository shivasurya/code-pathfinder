package cmd

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestScanCmd(t *testing.T) {
	tests := []struct {
		name           string
		rulesetDir     string
		projectDir     string
		expectedOutput string
	}{
		{
			name:           "Missing ruleset",
			rulesetDir:     "",
			projectDir:     "testproject",
			expectedOutput: "Please provide a ruleset directory path\n",
		},
		{
			name:           "Missing project",
			rulesetDir:     "testruleset",
			projectDir:     "",
			expectedOutput: "Please provide a project directory path\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("ruleset", tt.rulesetDir, "")
			cmd.Flags().String("project", tt.projectDir, "")

			output := captureOutput(func() {
				scanCmd.Run(cmd, []string{})
			})

			assert.Contains(t, output, tt.expectedOutput)
		})
	}
}

func TestGetAllRulesetFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ruleset_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test files
	validFiles := []string{"rule1.cql", "rule2.cql", "nested/rule3.cql"}
	invalidFiles := []string{"invalid1.txt", "invalid2.json"}

	for _, file := range append(validFiles, invalidFiles...) {
		path := filepath.Join(tempDir, file)
		err := os.MkdirAll(filepath.Dir(path), 0o755)
		assert.NoError(t, err)
		_, err = os.Create(path)
		assert.NoError(t, err)
	}

	result := getAllRulesetFile(tempDir)

	assert.Equal(t, len(validFiles), len(result))
	for _, file := range validFiles {
		assert.Contains(t, result, filepath.Join(tempDir, file))
	}
	for _, file := range invalidFiles {
		assert.NotContains(t, result, filepath.Join(tempDir, file))
	}
}

func TestGetAllRulesetFileInvalidDirectory(t *testing.T) {
	invalidDir := "/nonexistent/directory"
	result := getAllRulesetFile(invalidDir)
	assert.Nil(t, result)
}

func captureOutput(f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout

	return string(out)
}
