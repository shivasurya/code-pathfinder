package callgraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeCallGraph(t *testing.T) {
	t.Run("successfully initializes call graph with all components", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.py")
		code := `
def foo():
    x = 1
    return x

def bar():
    y = foo()
    return y
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		codeGraph := graph.Initialize(tmpDir, nil)
		callGraph, moduleRegistry, patternRegistry, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))

		assert.NoError(t, err)
		assert.NotNil(t, callGraph)
		assert.NotNil(t, moduleRegistry)
		assert.NotNil(t, patternRegistry)
		assert.Greater(t, len(callGraph.Functions), 0)
		assert.Greater(t, len(moduleRegistry.Modules), 0)
		assert.Greater(t, len(patternRegistry.Patterns), 0)
	})

	t.Run("handles empty project", func(t *testing.T) {
		tmpDir := t.TempDir()

		codeGraph := graph.Initialize(tmpDir, nil)
		callGraph, moduleRegistry, patternRegistry, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))

		assert.NoError(t, err)
		assert.NotNil(t, callGraph)
		assert.NotNil(t, moduleRegistry)
		assert.NotNil(t, patternRegistry)
	})

	t.Run("handles invalid project path", func(t *testing.T) {
		codeGraph := graph.Initialize("/nonexistent/path", nil)
		_, _, _, err := InitializeCallGraph(codeGraph, "/nonexistent/path", output.NewLogger(output.VerbosityDefault))

		// Should return error for invalid path
		assert.Error(t, err)
	})
}

func TestAnalyzePatterns(t *testing.T) {
	t.Run("detects security vulnerability with code snippets", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "vuln.py")
		code := `
def vulnerable():
    user_input = input("Enter: ")
    eval(user_input)
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		codeGraph := graph.Initialize(tmpDir, nil)
		callGraph, _, patternRegistry, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
		require.NoError(t, err)

		matches := AnalyzePatterns(callGraph, patternRegistry)

		assert.Greater(t, len(matches), 0, "Should detect at least one vulnerability")

		// Check that the first match has required fields
		match := matches[0]
		assert.NotEmpty(t, match.Severity)
		assert.NotEmpty(t, match.PatternName)
		assert.NotEmpty(t, match.Description)
		assert.NotEmpty(t, match.SourceFQN)
		assert.NotEmpty(t, match.SinkFQN)

		// Check that code snippets are populated
		if match.SourceFile != "" {
			assert.Greater(t, match.SourceLine, uint32(0))
			assert.NotEmpty(t, match.SourceCode)
		}
		if match.SinkFile != "" {
			assert.Greater(t, match.SinkLine, uint32(0))
			assert.NotEmpty(t, match.SinkCode)
		}
	})

	t.Run("returns empty for safe code", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "safe.py")
		code := `
def safe_function():
    x = 1 + 2
    return x
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		codeGraph := graph.Initialize(tmpDir, nil)
		callGraph, _, patternRegistry, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
		require.NoError(t, err)

		matches := AnalyzePatterns(callGraph, patternRegistry)

		assert.Equal(t, 0, len(matches), "Should not detect vulnerabilities in safe code")
	})

	t.Run("handles empty call graph", func(t *testing.T) {
		tmpDir := t.TempDir()

		codeGraph := graph.Initialize(tmpDir, nil)
		callGraph, _, patternRegistry, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
		require.NoError(t, err)

		matches := AnalyzePatterns(callGraph, patternRegistry)

		assert.Equal(t, 0, len(matches))
	})

	t.Run("populates all security match fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.py")
		code := `
def process():
    data = input()
    eval(data)
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		codeGraph := graph.Initialize(tmpDir, nil)
		callGraph, _, patternRegistry, err := InitializeCallGraph(codeGraph, tmpDir, output.NewLogger(output.VerbosityDefault))
		require.NoError(t, err)

		matches := AnalyzePatterns(callGraph, patternRegistry)

		if len(matches) > 0 {
			match := matches[0]
			// Check all required fields are populated
			assert.NotEmpty(t, match.Severity)
			assert.NotEmpty(t, match.PatternName)
			assert.NotEmpty(t, match.Description)
			assert.NotEmpty(t, match.SourceFQN)
			assert.NotEmpty(t, match.SinkFQN)
		}
	})
}

func TestGetCodeSnippet(t *testing.T) {
	t.Run("reads code snippet from file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.py")
		code := `line 1
line 2
line 3
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		snippet := getCodeSnippet(testFile, 2)
		assert.Equal(t, "line 2", snippet)
	})

	t.Run("returns empty for invalid line number", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.py")
		code := `line 1
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		snippet := getCodeSnippet(testFile, 0)
		assert.Equal(t, "", snippet)

		snippet = getCodeSnippet(testFile, -1)
		assert.Equal(t, "", snippet)

		snippet = getCodeSnippet(testFile, 100)
		assert.Equal(t, "", snippet)
	})

	t.Run("returns empty for nonexistent file", func(t *testing.T) {
		snippet := getCodeSnippet("/nonexistent/file.py", 1)
		assert.Equal(t, "", snippet)
	})

	t.Run("returns empty for empty file path", func(t *testing.T) {
		snippet := getCodeSnippet("", 1)
		assert.Equal(t, "", snippet)
	})

	t.Run("trims whitespace from code snippet", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.py")
		code := `    line with spaces
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		snippet := getCodeSnippet(testFile, 1)
		assert.Equal(t, "line with spaces", snippet)
	})

	t.Run("handles file with multiple lines", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.py")
		code := `def foo():
    x = 1
    y = 2
    return x + y
`
		err := os.WriteFile(testFile, []byte(code), 0644)
		require.NoError(t, err)

		assert.Equal(t, "def foo():", getCodeSnippet(testFile, 1))
		assert.Equal(t, "x = 1", getCodeSnippet(testFile, 2))
		assert.Equal(t, "y = 2", getCodeSnippet(testFile, 3))
		assert.Equal(t, "return x + y", getCodeSnippet(testFile, 4))
	})
}

func TestSecurityMatchStruct(t *testing.T) {
	t.Run("SecurityMatch has all required fields", func(t *testing.T) {
		match := SecurityMatch{
			Severity:      "critical",
			PatternName:   "test-pattern",
			Description:   "test description",
			CWE:           "CWE-89",
			OWASP:         "A03:2021",
			SourceFQN:     "test.source",
			SourceCall:    "input",
			SourceFile:    "/test/file.py",
			SourceLine:    10,
			SourceCode:    "x = input()",
			SinkFQN:       "test.sink",
			SinkCall:      "eval",
			SinkFile:      "/test/file.py",
			SinkLine:      20,
			SinkCode:      "eval(x)",
			DataFlowPath:  []string{"test.source", "test.sink"},
		}

		assert.Equal(t, "critical", match.Severity)
		assert.Equal(t, "test-pattern", match.PatternName)
		assert.Equal(t, "test description", match.Description)
		assert.Equal(t, "CWE-89", match.CWE)
		assert.Equal(t, "A03:2021", match.OWASP)
		assert.Equal(t, "test.source", match.SourceFQN)
		assert.Equal(t, "input", match.SourceCall)
		assert.Equal(t, "/test/file.py", match.SourceFile)
		assert.Equal(t, uint32(10), match.SourceLine)
		assert.Equal(t, "x = input()", match.SourceCode)
		assert.Equal(t, "test.sink", match.SinkFQN)
		assert.Equal(t, "eval", match.SinkCall)
		assert.Equal(t, "/test/file.py", match.SinkFile)
		assert.Equal(t, uint32(20), match.SinkLine)
		assert.Equal(t, "eval(x)", match.SinkCode)
		assert.Len(t, match.DataFlowPath, 2)
	})
}
