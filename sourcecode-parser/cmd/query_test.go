package cmd

import (
	"bytes"
	"encoding/json"
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
			project:        "testproject",
			query:          "FIND function WHERE name = 'test'",
			output:         "",
			stdin:          false,
			expectedOutput: "testproject/main.go:10\n------------\n> func test() {\n------------\n",
			expectedError:  "",
		},
		{
			name:           "JSON output",
			project:        "testproject",
			query:          "FIND function WHERE name = 'test'",
			output:         "json",
			stdin:          false,
			expectedOutput: `[{"code":"func test() {","file":"testproject/main.go","line":10}]`,
			expectedError:  "",
		},
		{
			name:           "Invalid query",
			project:        "testproject",
			query:          "INVALID query",
			output:         "",
			stdin:          false,
			expectedOutput: "",
			expectedError:  "error processing query",
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
		Type:        "function",
		Name:        "testFunc",
		File:        "test.go",
		LineNumber:  5,
		CodeSnippet: "func testFunc() {}",
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
			input:          "FIND function WHERE name = 'testFunc'",
			output:         "",
			expectedResult: "test.go:5\n------------\n> func testFunc() {}\n------------\n",
			expectedError:  "",
		},
		{
			name:           "JSON output",
			input:          "FIND function WHERE name = 'testFunc'",
			output:         "json",
			expectedResult: `[{"code":"func testFunc() {}","file":"test.go","line":5}]`,
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
				FIND function
				WHERE name = 'test'
				AND type = 'public'
			`,
			expectedQuery: "FIND function WHERE name = 'test' AND type = 'public'",
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

func TestQueryCmdExecution(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		input          string
		expectedOutput string
		expectedError  string
	}{
		{
			name:           "Basic query",
			args:           []string{"query", "-p", "testproject", "-q", "FIND function WHERE name = 'test'"},
			input:          "",
			expectedOutput: "Executing query: FIND function WHERE name = 'test'\ntestproject/main.go:10\n------------\n> func test() {\n------------\n",
			expectedError:  "",
		},
		{
			name:           "Query from stdin",
			args:           []string{"query", "-p", "testproject", "--stdin"},
			input:          "FIND function WHERE name = 'test'\n:quit\n",
			expectedOutput: "Path-Finder Query Console: \n>Executing query: FIND function WHERE name = 'test'\ntestproject/main.go:10\n------------\n> func test() {\n------------\nPath-Finder Query Console: \n>Okay, Bye!",
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "pathfinder"}
			cmd.AddCommand(queryCmd)

			b := bytes.NewBufferString(tt.input)
			cmd.SetIn(b)
			cmd.SetArgs(tt.args)

			output := &bytes.Buffer{}
			cmd.SetOut(output)
			cmd.SetErr(output)

			err := cmd.Execute()

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, output.String(), tt.expectedOutput)
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

	input := "FIND function WHERE name = 'test'\n:quit\n"
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		_, _ = w.WriteString(input)
		w.Close()
	}()

	result, err := executeCLIQuery("testproject", "", "", true)
	assert.NoError(t, err)
	assert.Equal(t, "Okay, Bye!", result)

	_, _ = io.Copy(io.Discard, r)
}
