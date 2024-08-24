package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestExecuteCLIQuery(t *testing.T) {
	tests := []struct {
		name           string
		project        string
		query          string
		output         string
		stdin          bool
		expectedOutput string
		expectedError  string
	}{
		{
			name:           "Basic query",
			project:        "../../test-src/android",
			query:          "FIND method_declaration AS md WHERE md.getName() == \"onCreateOptionsMenu\"",
			output:         "",
			stdin:          false,
			expectedOutput: "../../test-src/android/app/src/main/java/com/ivb/udacity/movieListActivity.java:96\n------------\n> @Override\n    public boolean onCreateOptionsMenu(Menu menu) {\n        MenuInflater inflater = getMenuInflater();\n        inflater.inflate(R.menu.main, menu);\n        return true;\n    }\n------------",
			expectedError:  "",
		},
		{
			name:           "JSON output",
			project:        "../../test-src/android",
			query:          "FIND method_declaration AS md WHERE md.getName() == \"onCreateOptionsMenu\"",
			output:         "json",
			stdin:          false,
			expectedOutput: `[{"code":"@Override\n    public boolean onCreateOptionsMenu(Menu menu) {\n        MenuInflater inflater = getMenuInflater();\n        inflater.inflate(R.menu.main, menu);\n        return true;\n    }","file":"../../test-src/android/app/src/main/java/com/ivb/udacity/movieListActivity.java","line":96}]`,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeCLIQuery(tt.project, tt.query, tt.output, tt.stdin)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, strings.TrimSpace(result))
			}
		})
	}
}

func TestProcessQuery(t *testing.T) {
	codeGraph := graph.NewCodeGraph()
	codeGraph.AddNode(&graph.Node{
		Type:        "method_declaration",
		Name:        "testFunc",
		File:        "test.java",
		LineNumber:  5,
		CodeSnippet: "public void testFunc() {}",
	})

	tests := []struct {
		name           string
		input          string
		output         string
		expectedResult string
		expectedError  string
	}{
		{
			name:           "Basic query",
			input:          "FIND method_declaration AS md WHERE md.getName() == \"testFunc\"",
			output:         "",
			expectedResult: "test.java:5\n------------\n> public void testFunc() {}\n------------\n",
			expectedError:  "",
		},
		{
			name:           "JSON output",
			input:          "FIND method_declaration AS md WHERE md.getName() == \"testFunc\"",
			output:         "json",
			expectedResult: `[{"code":"public void testFunc() {}", "file":"test.java", "line":5}]`,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processQuery(tt.input, codeGraph, tt.output)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				if tt.output == "json" {
					var expectedJSON, resultJSON []map[string]interface{}
					err = json.Unmarshal([]byte(tt.expectedResult), &expectedJSON)
					assert.NoError(t, err)
					err = json.Unmarshal([]byte(result), &resultJSON)
					assert.NoError(t, err)
					assert.Equal(t, expectedJSON, resultJSON)
				} else {
					assert.Equal(t, tt.expectedResult, result)
				}
			}
		})
	}
}

func TestExtractQueryFromFile(t *testing.T) {
	tests := []struct {
		name          string
		fileContent   string
		expectedQuery string
		expectedError string
	}{
		{
			name: "Valid query file",
			fileContent: `
				// This is a comment
				FIND method_declaration AS md
				WHERE md.getName() == "test"
				AND md.getVisibility() == "public"
			`,
			expectedQuery: "FIND method_declaration AS md \t\t\t\tWHERE md.getName() == \"test\" \t\t\t\tAND md.getVisibility() == \"public\"",
			expectedError: "",
		},
		{
			name: "Query file without FIND",
			fileContent: `
				// This is a comment
				SELECT function
				WHERE name = 'test'
			`,
			expectedQuery: "",
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile, err := os.CreateTemp("", "query_*.txt")
			assert.NoError(t, err)
			defer os.Remove(tempFile.Name())

			_, err = tempFile.WriteString(tt.fileContent)
			assert.NoError(t, err)
			tempFile.Close()

			result, err := ExtractQueryFromFile(tempFile.Name())

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedQuery, result)
			}
		})
	}
}

func TestQueryCmdFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "pathfinder"}
	cmd.AddCommand(queryCmd)

	tests := []struct {
		name     string
		flag     string
		expected string
	}{
		{"output flag", "output", ""},
		{"output-file flag", "output-file", ""},
		{"project flag", "project", ""},
		{"query flag", "query", ""},
		{"stdin flag", "stdin", "false"},
		{"query-file flag", "query-file", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := queryCmd.Flag(tt.flag)
			assert.NotNil(t, flag)
			assert.Equal(t, tt.expected, flag.Value.String())
		})
	}
}

func TestQueryCmdStdinInput(t *testing.T) {
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	input := ":quit\n"
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		_, _ = w.WriteString(input)
		w.Close()
	}()

	result, err := executeCLIQuery("../../test-src/android", "", "", true)
	fmt.Println(result)
	assert.NoError(t, err)
	assert.Equal(t, "Okay, Bye!", result)

	_, _ = io.Copy(io.Discard, r)
}