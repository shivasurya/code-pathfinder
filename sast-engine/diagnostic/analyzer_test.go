package diagnostic

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/extraction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnalyzeSingleFunction_SimpleFlow tests basic taint flow detection.
func TestAnalyzeSingleFunction_SimpleFlow(t *testing.T) {
	fn := &FunctionMetadata{
		FQN:          "test.simple_flow",
		FunctionName: "simple_flow",
		FilePath:     "test.py",
		SourceCode: `def simple_flow():
    user_input = input()
    eval(user_input)`,
		StartLine: 1,
		EndLine:   3,
	}

	result, err := AnalyzeSingleFunction(fn, []string{"input"}, []string{"eval"}, []string{})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "test.simple_flow", result.FunctionFQN)
	assert.True(t, result.HasTaintFlow)
	assert.False(t, result.AnalysisError)
	assert.GreaterOrEqual(t, len(result.TaintFlows), 1)
}

// TestAnalyzeSingleFunction_NoFlow tests function with no taint flows.
func TestAnalyzeSingleFunction_NoFlow(t *testing.T) {
	fn := &FunctionMetadata{
		FQN:          "test.no_flow",
		FunctionName: "no_flow",
		FilePath:     "test.py",
		SourceCode: `def no_flow():
    x = 1
    return x`,
		StartLine: 1,
		EndLine:   3,
	}

	result, err := AnalyzeSingleFunction(fn, []string{"input"}, []string{"eval"}, []string{})
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "test.no_flow", result.FunctionFQN)
	assert.False(t, result.HasTaintFlow)
	assert.False(t, result.AnalysisError)
	assert.Equal(t, 0, len(result.TaintFlows))
}

// TestAnalyzeSingleFunction_ParseError tests error handling for invalid syntax.
func TestAnalyzeSingleFunction_ParseError(t *testing.T) {
	fn := &FunctionMetadata{
		FQN:          "test.invalid",
		FunctionName: "invalid",
		FilePath:     "test.py",
		SourceCode:   `def invalid( # missing closing paren`,
		StartLine:    1,
		EndLine:      1,
	}

	result, err := AnalyzeSingleFunction(fn, []string{"input"}, []string{"eval"}, []string{})
	require.NoError(t, err) // Should not error, but set AnalysisError flag
	require.NotNil(t, result)

	assert.True(t, result.AnalysisError)
	// Can be either parse error or function not found (depends on tree-sitter recovery)
	assert.NotEmpty(t, result.ErrorMessage)
}

// TestCategorizePattern tests semantic pattern categorization.
func TestCategorizePattern(t *testing.T) {
	tests := []struct {
		pattern  string
		expected string
	}{
		{"request.GET", "user_input"},
		{"request.POST", "user_input"},
		{"input()", "user_input"},
		{"os.system", "command_exec"},
		{"subprocess.call", "command_exec"},
		{"eval", "code_exec"},
		{"execute", "sql_execution"},
		{"open()", "file_operation"},
		{"unknown", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result := categorizePattern(tt.pattern, []string{})
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestInferVulnerabilityType tests vulnerability type inference.
func TestInferVulnerabilityType(t *testing.T) {
	tests := []struct {
		source   string
		sink     string
		expected string
	}{
		{"user_input", "sql_execution", "SQL_INJECTION"},
		{"user_input", "command_exec", "COMMAND_INJECTION"},
		{"user_input", "code_exec", "CODE_INJECTION"},
		{"user_input", "file_operation", "PATH_TRAVERSAL"},
		{"other", "other", "TAINT_FLOW"},
	}

	for _, tt := range tests {
		t.Run(tt.source+"_to_"+tt.sink, func(t *testing.T) {
			result := inferVulnerabilityType(tt.source, tt.sink)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestFindFunctionNodeByName tests AST function finding.
func TestFindFunctionNodeByName(t *testing.T) {
	sourceCode := []byte(`
def function_one():
    pass

def function_two():
    pass
`)

	tree, err := extraction.ParsePythonFile(sourceCode)
	require.NoError(t, err)
	require.NotNil(t, tree)

	// Should find function_one
	node := findFunctionNodeByName(tree.RootNode(), "function_one", sourceCode)
	assert.NotNil(t, node)

	// Should find function_two
	node = findFunctionNodeByName(tree.RootNode(), "function_two", sourceCode)
	assert.NotNil(t, node)

	// Should not find non-existent function
	node = findFunctionNodeByName(tree.RootNode(), "function_three", sourceCode)
	assert.Nil(t, node)
}
