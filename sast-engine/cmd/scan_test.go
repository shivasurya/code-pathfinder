package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/dsl"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create test rules (duplicated from ci_test.go).
func createTestRuleScan(id, name, severity, cwe, owasp, description string) dsl.RuleIR {
	rule := dsl.RuleIR{}
	rule.Rule.ID = id
	rule.Rule.Name = name
	rule.Rule.Severity = severity
	rule.Rule.CWE = cwe
	rule.Rule.OWASP = owasp
	rule.Rule.Description = description
	return rule
}

func TestCountTotalCallSites(t *testing.T) {
	t.Run("counts call sites across all functions", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["func1"] = []core.CallSite{
			{Target: "foo", Location: core.Location{Line: 10}},
			{Target: "bar", Location: core.Location{Line: 20}},
		}
		cg.CallSites["func2"] = []core.CallSite{
			{Target: "baz", Location: core.Location{Line: 30}},
		}

		total := countTotalCallSites(cg)
		assert.Equal(t, 3, total)
	})

	t.Run("returns zero for empty callgraph", func(t *testing.T) {
		cg := core.NewCallGraph()
		total := countTotalCallSites(cg)
		assert.Equal(t, 0, total)
	})

	t.Run("handles function with no call sites", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["func1"] = []core.CallSite{}
		total := countTotalCallSites(cg)
		assert.Equal(t, 0, total)
	})
}

func TestPrintDetections(t *testing.T) {
	t.Run("prints detections with all fields", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rule := createTestRuleScan("test-rule", "Test Rule", "high", "CWE-89", "A03:2021", "Test SQL injection detection")

		detections := []dsl.DataflowDetection{
			{
				FunctionFQN: "test.vulnerable_func",
				SourceLine:  10,
				SinkLine:    20,
				SinkCall:    "execute",
				TaintedVar:  "user_input",
				Confidence:  0.9,
				Scope:       "local",
			},
		}

		printDetections(rule, detections)

		// Restore stdout
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Verify output contains expected information
		assert.Contains(t, output, "[high] test-rule (Test Rule)")
		assert.Contains(t, output, "CWE: CWE-89")
		assert.Contains(t, output, "OWASP: A03:2021")
		assert.Contains(t, output, "Test SQL injection detection")
		assert.Contains(t, output, "test.vulnerable_func:20")
		assert.Contains(t, output, "Source: line 10")
		assert.Contains(t, output, "Sink: execute (line 20)")
		assert.Contains(t, output, "Tainted variable: user_input")
		assert.Contains(t, output, "Confidence: 90%")
		assert.Contains(t, output, "Scope: local")
	})

	t.Run("prints detections without optional fields", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rule := createTestRuleScan("simple-rule", "Simple Rule", "medium", "", "", "Simple detection")

		detections := []dsl.DataflowDetection{
			{
				FunctionFQN: "test.func",
				SinkLine:    15,
				Confidence:  0.5,
				Scope:       "global",
			},
		}

		printDetections(rule, detections)

		// Restore stdout
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Verify output
		assert.Contains(t, output, "[medium] simple-rule (Simple Rule)")
		assert.Contains(t, output, "test.func:15")
		assert.Contains(t, output, "Confidence: 50%")
		assert.Contains(t, output, "Scope: global")
		// Should not contain optional fields
		assert.NotContains(t, output, "Source: line 0")
		assert.NotContains(t, output, "Sink: ")
		assert.NotContains(t, output, "Tainted variable: ")
	})

	t.Run("prints multiple detections", func(t *testing.T) {
		// Capture stdout
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rule := createTestRuleScan("multi-rule", "Multi Rule", "critical", "CWE-79", "A03:2021", "XSS detection")

		detections := []dsl.DataflowDetection{
			{
				FunctionFQN: "test.func1",
				SinkLine:    10,
				Confidence:  0.8,
				Scope:       "local",
			},
			{
				FunctionFQN: "test.func2",
				SinkLine:    20,
				Confidence:  0.7,
				Scope:       "local",
			},
		}

		printDetections(rule, detections)

		// Restore stdout
		w.Close()
		os.Stdout = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		output := buf.String()

		// Verify both detections are printed
		assert.Contains(t, output, "test.func1:10")
		assert.Contains(t, output, "test.func2:20")
		assert.Contains(t, output, "Confidence: 80%")
		assert.Contains(t, output, "Confidence: 70%")
	})
}

func TestExtractContainerFiles(t *testing.T) {
	t.Run("extracts Dockerfile and docker-compose files", func(t *testing.T) {
		cg := &graph.CodeGraph{
			Nodes: map[string]*graph.Node{
				"node1": {Type: "dockerfile_instruction", File: "/path/to/Dockerfile"},
				"node2": {Type: "dockerfile_instruction", File: "/path/to/Dockerfile.dev"},
				"node3": {Type: "compose_service", File: "/path/to/docker-compose.yml"},
				"node4": {Type: "method_declaration", File: "/path/to/main.go"},
			},
		}

		dockerFiles, composeFiles := extractContainerFiles(cg)

		assert.Equal(t, 2, len(dockerFiles))
		assert.Contains(t, dockerFiles, "/path/to/Dockerfile")
		assert.Contains(t, dockerFiles, "/path/to/Dockerfile.dev")

		assert.Equal(t, 1, len(composeFiles))
		assert.Contains(t, composeFiles, "/path/to/docker-compose.yml")
	})

	t.Run("handles duplicates", func(t *testing.T) {
		cg := &graph.CodeGraph{
			Nodes: map[string]*graph.Node{
				"node1": {Type: "dockerfile_instruction", File: "/path/to/Dockerfile"},
				"node2": {Type: "dockerfile_instruction", File: "/path/to/Dockerfile"},
				"node3": {Type: "compose_service", File: "/path/to/docker-compose.yml"},
				"node4": {Type: "compose_service", File: "/path/to/docker-compose.yml"},
			},
		}

		dockerFiles, composeFiles := extractContainerFiles(cg)

		// Should deduplicate
		assert.Equal(t, 1, len(dockerFiles))
		assert.Equal(t, 1, len(composeFiles))
	})

	t.Run("returns empty for no container files", func(t *testing.T) {
		cg := &graph.CodeGraph{
			Nodes: map[string]*graph.Node{
				"node1": {Type: "method_declaration", File: "/path/to/main.go"},
				"node2": {Type: "class_declaration", File: "/path/to/app.java"},
			},
		}

		dockerFiles, composeFiles := extractContainerFiles(cg)

		assert.Equal(t, 0, len(dockerFiles))
		assert.Equal(t, 0, len(composeFiles))
	})
}

func TestSplitLines(t *testing.T) {
	t.Run("splits simple content", func(t *testing.T) {
		content := "line 1\nline 2\nline 3"
		lines := splitLines(content)

		assert.Equal(t, 3, len(lines))
		assert.Equal(t, "line 1", lines[0])
		assert.Equal(t, "line 2", lines[1])
		assert.Equal(t, "line 3", lines[2])
	})

	t.Run("handles empty lines", func(t *testing.T) {
		content := "line 1\n\nline 3"
		lines := splitLines(content)

		assert.Equal(t, 3, len(lines))
		assert.Equal(t, "line 1", lines[0])
		assert.Equal(t, "", lines[1])
		assert.Equal(t, "line 3", lines[2])
	})

	t.Run("handles Windows line endings", func(t *testing.T) {
		content := "line 1\r\nline 2\r\nline 3"
		lines := splitLines(content)

		assert.Equal(t, 3, len(lines))
		assert.Equal(t, "line 1", lines[0])
		assert.Equal(t, "line 2", lines[1])
		assert.Equal(t, "line 3", lines[2])
	})

	t.Run("handles empty content", func(t *testing.T) {
		lines := splitLines("")
		assert.Equal(t, 0, len(lines))
	})

	t.Run("preserves last line without newline", func(t *testing.T) {
		content := "line 1\nline 2"
		lines := splitLines(content)

		assert.Equal(t, 2, len(lines))
		assert.Equal(t, "line 1", lines[0])
		assert.Equal(t, "line 2", lines[1])
	})
}

func TestScanCommandOutputFormats(t *testing.T) {
	// Note: These are integration-style tests that verify the command flags are properly registered
	t.Run("scan command has output flag", func(t *testing.T) {
		flag := scanCmd.Flags().Lookup("output")
		require.NotNil(t, flag, "output flag should be registered")
		assert.Equal(t, "text", flag.DefValue, "default output should be text")
	})

	t.Run("scan command has output-file flag", func(t *testing.T) {
		flag := scanCmd.Flags().Lookup("output-file")
		require.NotNil(t, flag, "output-file flag should be registered")
		assert.Equal(t, "", flag.DefValue, "default output-file should be empty")
	})

	t.Run("scan command has rules flag", func(t *testing.T) {
		flag := scanCmd.Flags().Lookup("rules")
		require.NotNil(t, flag, "rules flag should be registered")
	})

	t.Run("scan command has project flag", func(t *testing.T) {
		flag := scanCmd.Flags().Lookup("project")
		require.NotNil(t, flag, "project flag should be registered")
	})

	t.Run("scan command has verbose flag", func(t *testing.T) {
		flag := scanCmd.Flags().Lookup("verbose")
		require.NotNil(t, flag, "verbose flag should be registered")
	})

	t.Run("scan command has debug flag", func(t *testing.T) {
		flag := scanCmd.Flags().Lookup("debug")
		require.NotNil(t, flag, "debug flag should be registered")
	})

	t.Run("scan command has fail-on flag", func(t *testing.T) {
		flag := scanCmd.Flags().Lookup("fail-on")
		require.NotNil(t, flag, "fail-on flag should be registered")
	})

	t.Run("output format validation", func(t *testing.T) {
		// Valid formats
		validFormats := []string{"text", "json", "sarif", "csv"}
		for _, format := range validFormats {
			t.Run("accepts "+format, func(t *testing.T) {
				// This just verifies the flag accepts these values
				err := scanCmd.Flags().Set("output", format)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("output flag short form", func(t *testing.T) {
		flag := scanCmd.Flags().ShorthandLookup("o")
		require.NotNil(t, flag, "output flag should have short form -o")
		assert.Equal(t, "output", flag.Name)
	})

	t.Run("output-file flag short form", func(t *testing.T) {
		flag := scanCmd.Flags().ShorthandLookup("f")
		require.NotNil(t, flag, "output-file flag should have short form -f")
		assert.Equal(t, "output-file", flag.Name)
	})
}

func TestGenerateCodeSnippet(t *testing.T) {
	// Create a temporary test file
	content := `line 1
line 2
line 3
line 4
line 5
line 6
line 7`

	tmpFile, err := os.CreateTemp("", "test-snippet-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	tmpFile.Close()

	t.Run("generates snippet with context", func(t *testing.T) {
		snippet := generateCodeSnippet(tmpFile.Name(), 4, 2)

		assert.Equal(t, 5, len(snippet.Lines))
		assert.Equal(t, 2, snippet.StartLine)
		assert.Equal(t, 4, snippet.HighlightLine)

		// Check line numbers and content
		assert.Equal(t, 2, snippet.Lines[0].Number)
		assert.Equal(t, "line 2", snippet.Lines[0].Content)
		assert.False(t, snippet.Lines[0].IsHighlight)

		assert.Equal(t, 4, snippet.Lines[2].Number)
		assert.Equal(t, "line 4", snippet.Lines[2].Content)
		assert.True(t, snippet.Lines[2].IsHighlight)

		assert.Equal(t, 6, snippet.Lines[4].Number)
		assert.Equal(t, "line 6", snippet.Lines[4].Content)
	})

	t.Run("handles line at start of file", func(t *testing.T) {
		snippet := generateCodeSnippet(tmpFile.Name(), 1, 2)

		assert.Equal(t, 3, len(snippet.Lines)) // Lines 1, 2, 3
		assert.Equal(t, 1, snippet.StartLine)
		assert.Equal(t, 1, snippet.HighlightLine)
		assert.True(t, snippet.Lines[0].IsHighlight)
	})

	t.Run("handles line at end of file", func(t *testing.T) {
		snippet := generateCodeSnippet(tmpFile.Name(), 7, 2)

		assert.Equal(t, 3, len(snippet.Lines)) // Lines 5, 6, 7
		assert.Equal(t, 5, snippet.StartLine)
		assert.Equal(t, 7, snippet.HighlightLine)
		assert.True(t, snippet.Lines[2].IsHighlight)
	})

	t.Run("handles invalid line number", func(t *testing.T) {
		snippet := generateCodeSnippet(tmpFile.Name(), 100, 2)

		assert.Equal(t, 0, len(snippet.Lines))
		assert.Equal(t, 0, snippet.StartLine)
		assert.Equal(t, 0, snippet.HighlightLine)
	})

	t.Run("handles nonexistent file", func(t *testing.T) {
		snippet := generateCodeSnippet("/nonexistent/file.txt", 1, 2)

		assert.Equal(t, 0, len(snippet.Lines))
	})
}
