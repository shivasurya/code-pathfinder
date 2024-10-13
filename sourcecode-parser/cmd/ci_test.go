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
			args:           []string{"ci", "--help"},
			expectedOutput: "Scan a project for vulnerabilities with ruleset in ci mode\n\nUsage:\n  pathfinder ci [flags]\n\nFlags:\n  -h, --help                     help for ci\n  -o, --output string            Supported output format: json\n  -f, --output-file string       Output file path\n  -p, --project string           Project to analyze\n  -r, --rules-directory string   Rules directory to use\n  -q, --ruleset string           Ruleset to use example: cfp/java\n",
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

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Single predicate",
			input:    "predicate foo()\n{\n    bar\n}",
			expected: "predicate foo() {     bar }",
		},
		{
			name:     "Multiple predicates",
			input:    "some code\npredicate foo()\n{\n    bar\n}\npredicate baz()\n{\n    qux\n}",
			expected: "predicate foo() {     bar } predicate baz() {     qux }",
		},
		{
			name:     "FROM clause",
			input:    "SELECT *\nFROM table\nWHERE condition",
			expected: "FROM table WHERE condition",
		},
		{
			name:     "Mixed predicates and FROM",
			input:    "predicate foo()\n{\n    bar\n}\nSELECT *\nFROM table\nWHERE condition",
			expected: "predicate foo() {     bar } SELECT * FROM table WHERE condition",
		},
		{
			name:     "No matching lines",
			input:    "Some random\ntext without\nmatching lines",
			expected: "",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseQuery(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
