package diagnostic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBuildAnalysisPrompt tests prompt construction.
func TestBuildAnalysisPrompt(t *testing.T) {
	sourceCode := "def test():\n    x = 1\n    return x"

	prompt := BuildAnalysisPrompt(sourceCode)

	// Verify prompt contains key elements
	assert.Contains(t, prompt, "dataflow analysis expert")
	assert.Contains(t, prompt, sourceCode)
	assert.Contains(t, prompt, "DISCOVER DATA SOURCES")
	assert.Contains(t, prompt, "TRACE INTRA-PROCEDURAL FLOWS")
	assert.Contains(t, prompt, "GENERATE TEST CASES")
	assert.Contains(t, prompt, "discovered_patterns")
	assert.Contains(t, prompt, "dataflow_test_cases")
	assert.Contains(t, prompt, "JSON")
	assert.Contains(t, prompt, "sources")
	assert.Contains(t, prompt, "sinks")
	assert.Contains(t, prompt, "sanitizers")
	assert.Contains(t, prompt, "propagators")
}

// TestBuildAnalysisPrompt_ContainsExamples tests that prompt includes examples.
func TestBuildAnalysisPrompt_ContainsExamples(t *testing.T) {
	prompt := BuildAnalysisPrompt("def dummy(): pass")

	// Check for security examples
	assert.Contains(t, prompt, "request.GET")
	assert.Contains(t, prompt, "os.system")
	assert.Contains(t, prompt, "COMMAND_INJECTION")

	// Check for generic dataflow examples
	assert.Contains(t, prompt, "param")
	assert.Contains(t, prompt, "return")
}

// TestBuildAnalysisPrompt_ContainsGuidelines tests that prompt includes important guidelines.
func TestBuildAnalysisPrompt_ContainsGuidelines(t *testing.T) {
	prompt := BuildAnalysisPrompt("def dummy(): pass")

	assert.Contains(t, prompt, "INTRA-PROCEDURAL ONLY")
	assert.Contains(t, prompt, "BE SPECIFIC")
	assert.Contains(t, prompt, "TRACK SIMPLE DATAFLOWS")
	assert.Contains(t, prompt, "CONFIDENCE SCORES")
	assert.Contains(t, prompt, "Output ONLY the JSON")
}

// TestBuildAnalysisPrompt_JSONStructure tests that prompt shows expected JSON structure.
func TestBuildAnalysisPrompt_JSONStructure(t *testing.T) {
	prompt := BuildAnalysisPrompt("def dummy(): pass")

	// Check for JSON structure elements
	assert.Contains(t, prompt, "pattern")
	assert.Contains(t, prompt, "lines")
	assert.Contains(t, prompt, "variables")
	assert.Contains(t, prompt, "category")
	assert.Contains(t, prompt, "description")
	assert.Contains(t, prompt, "test_id")
	assert.Contains(t, prompt, "expected_detection")
	assert.Contains(t, prompt, "vulnerability_type")
	assert.Contains(t, prompt, "confidence")
	assert.Contains(t, prompt, "reasoning")
	assert.Contains(t, prompt, "variable_tracking")
	assert.Contains(t, prompt, "analysis_metadata")
}

// TestBuildAnalysisPrompt_EmptySourceCode tests with empty source code.
func TestBuildAnalysisPrompt_EmptySourceCode(t *testing.T) {
	prompt := BuildAnalysisPrompt("")

	// Should still generate valid prompt structure
	assert.Contains(t, prompt, "DISCOVER DATA SOURCES")
	assert.Contains(t, prompt, "GENERATE TEST CASES")
	assert.NotEmpty(t, prompt)
}

// TestBuildAnalysisPrompt_ComplexSourceCode tests with realistic source code.
func TestBuildAnalysisPrompt_ComplexSourceCode(t *testing.T) {
	sourceCode := `def process_input(request):
    user_cmd = request.GET['cmd']
    cleaned = shlex.quote(user_cmd)
    os.system(cleaned)`

	prompt := BuildAnalysisPrompt(sourceCode)

	// Verify source code is embedded
	assert.Contains(t, prompt, "process_input")
	assert.Contains(t, prompt, "user_cmd")
	assert.Contains(t, prompt, "shlex.quote")

	// Verify prompt structure intact
	assert.Contains(t, prompt, "```python")
	assert.Contains(t, prompt, "```json")
}
