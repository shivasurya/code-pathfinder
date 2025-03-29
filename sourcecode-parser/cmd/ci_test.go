package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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
			expectedOutput: "Scan a project for vulnerabilities with ruleset in ci mode\n\nUsage:\n  pathfinder ci [flags]\n\nFlags:\n  -h, --help                 help for ci\n  -o, --output string        Supported output format: json, sarif\n  -f, --output-file string   Output file path\n  -p, --project string       Source code to analyze\n  -r, --ruleset string       Ruleset to use example: cfp/java or directory path\n",
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
		expected Rule
	}{
		{
			name:     "Single predicate",
			input:    "predicate foo()\n{\n    bar\n}",
			expected: Rule{ID: "", Description: "", Impact: "", Severity: "", Passed: false, Query: "predicate foo() {     bar }", RuleProvider: ""},
		},
		{
			name:     "Multiple predicates",
			input:    "some code\npredicate foo()\n{\n    bar\n}\npredicate baz()\n{\n    qux\n}",
			expected: Rule{ID: "", Description: "", Impact: "", Severity: "", Passed: false, Query: "predicate foo() {     bar } predicate baz() {     qux }", RuleProvider: ""}},
		{
			name:     "FROM clause",
			input:    "SELECT *\nFROM table\nWHERE condition",
			expected: Rule{ID: "", Description: "", Impact: "", Severity: "", Passed: false, Query: "FROM table WHERE condition", RuleProvider: ""},
		},
		{
			name:     "Mixed predicates and FROM",
			input:    "predicate foo()\n{\n    bar\n}\nSELECT *\nFROM table\nWHERE condition",
			expected: Rule{ID: "", Description: "", Impact: "", Severity: "", Passed: false, Query: "predicate foo() {     bar } SELECT * FROM table WHERE condition", RuleProvider: ""},
		},
		{
			name:     "No matching lines",
			input:    "cmd.Rule(cmd.Rule{ID:\"\", Description:\"\", Impact:\"\", Severity:\"\", Passed:false, Query:\"\", RuleProvider:\"\"})",
			expected: Rule{ID: "", Description: "", Impact: "", Severity: "", Passed: false, Query: "", RuleProvider: ""},
		},
		{
			name:     "Empty input",
			input:    "",
			expected: Rule{ID: "", Description: "", Impact: "", Severity: "", Passed: false, Query: "", RuleProvider: ""},
		},
		{
			name:     "Single line comment",
			input:    "/**\n * @name Android WebView JavaScript settings\n * @description Enabling setAllowFileAccessFromFileURLs leak s&&box access to file:/// URLs.\n * @kind problem\n * @id java/Android/webview-javascript-enabled\n * @problem.severity warning\n * @security-severity 6.1\n * @precision medium\n * @tags security\n * external/cwe/cwe-079\n * @ruleprovider android\n */\nFROM method_invocation AS mi\nWHERE mi.getName() == \"setAllowFileAccessFromFileURLs\" && \"true\" in mi.getArgumentName()\nSELECT mi.getName(), \"File access enabled\"",
			expected: Rule{ID: "java/Android/webview-javascript-enabled", Description: "Enabling setAllowFileAccessFromFileURLs leak s&&box access to file:/// URLs.", Impact: "6.1", Severity: "warning", Passed: false, Query: "FROM method_invocation AS mi WHERE mi.getName() == \"setAllowFileAccessFromFileURLs\" && \"true\" in mi.getArgumentName() SELECT mi.getName(), \"File access enabled\"", RuleProvider: "android"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseQuery(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadRules(t *testing.T) {
	tests := []struct {
		name           string
		rulesDirectory string
		isHosted       bool
		expectedRules  int
		expectError    bool
	}{
		{
			name:           "Local rules directory",
			rulesDirectory: "../../pathfinder-rules",
			isHosted:       false,
			expectedRules:  13,
			expectError:    false,
		},
		{
			name:           "Hosted rules",
			rulesDirectory: "cpf/java",
			isHosted:       true,
			expectedRules:  6,
			expectError:    false,
		},
		{
			name:           "Non-existent local directory",
			rulesDirectory: "testdata/nonexistent",
			isHosted:       false,
			expectedRules:  0,
			expectError:    true,
		},
		{
			name:           "Invalid hosted URL",
			rulesDirectory: "https://invalid.example.com/rules",
			isHosted:       true,
			expectedRules:  0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules, err := loadRules(tt.rulesDirectory, tt.isHosted)
			fmt.Println(len(rules))

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, rules, tt.expectedRules)
			}
		})
	}
}

func TestLoadRulesLocalFileContent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rules_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testRule := "predicate test() { foo }"
	err = os.WriteFile(filepath.Join(tempDir, "test.cql"), []byte(testRule), 0644)
	assert.NoError(t, err)

	rules, err := loadRules(tempDir, false)
	assert.NoError(t, err)
	assert.Len(t, rules, 1)
	assert.Equal(t, testRule, rules[0])
}

func TestLoadRulesIgnoreNonCQLFiles(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rules_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = os.WriteFile(filepath.Join(tempDir, "test.cql"), []byte("predicate test() { foo }"), 0644)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "ignore.txt"), []byte("This should be ignored"), 0644)
	assert.NoError(t, err)

	rules, err := loadRules(tempDir, false)
	assert.NoError(t, err)
	assert.Len(t, rules, 1)
}

func TestLoadRulesNestedDirectories(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "rules_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	nestedDir := filepath.Join(tempDir, "nested")
	err = os.Mkdir(nestedDir, 0755)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "test1.cql"), []byte("predicate test1() { foo }"), 0644)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(nestedDir, "test2.cql"), []byte("predicate test2() { bar }"), 0644)
	assert.NoError(t, err)

	rules, err := loadRules(tempDir, false)
	assert.NoError(t, err)
	assert.Len(t, rules, 2)
}
