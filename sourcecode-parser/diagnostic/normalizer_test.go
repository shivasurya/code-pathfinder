package diagnostic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNormalizeToolResult tests tool result normalization.
func TestNormalizeToolResult(t *testing.T) {
	toolResult := &FunctionTaintResult{
		FunctionFQN:  "test.func",
		HasTaintFlow: true,
		TaintFlows: []ToolTaintFlow{
			{
				SourceLine:        10,
				SourceVariable:    "user_input",
				SourceCategory:    "user_input",
				SinkLine:          20,
				SinkVariable:      "query",
				SinkCategory:      "sql_execution",
				VulnerabilityType: "SQL_INJECTION",
				Confidence:        0.95,
			},
		},
	}

	normalized := NormalizeToolResult(toolResult)

	assert.Equal(t, 1, len(normalized))
	assert.Equal(t, 10, normalized[0].SourceLine)
	assert.Equal(t, "user_input", normalized[0].SourceVariable)
	assert.Equal(t, "user_input", normalized[0].SourceCategory)
	assert.Equal(t, 20, normalized[0].SinkLine)
	assert.Equal(t, "SQL_INJECTION", normalized[0].VulnerabilityType)
}

// TestNormalizeLLMResult tests LLM result normalization.
func TestNormalizeLLMResult(t *testing.T) {
	llmResult := &LLMAnalysisResult{
		DataflowTestCases: []DataflowTestCase{
			{
				Source: TestCaseSource{
					Pattern:  "request.GET['cmd']",
					Line:     5,
					Variable: "cmd",
				},
				Sink: TestCaseSink{
					Pattern:  "os.system",
					Line:     10,
					Variable: "cmd",
				},
				ExpectedDetection: true,
				VulnerabilityType: "COMMAND_INJECTION",
				Confidence:        0.9,
			},
			{
				Source: TestCaseSource{
					Pattern:  "input()",
					Line:     15,
					Variable: "safe",
				},
				Sink: TestCaseSink{
					Pattern:  "print",
					Line:     16,
					Variable: "safe",
				},
				ExpectedDetection: false, // Should NOT be included
			},
		},
	}

	normalized := NormalizeLLMResult(llmResult)

	// Only expected detections are included
	assert.Equal(t, 1, len(normalized))
	assert.Equal(t, 5, normalized[0].SourceLine)
	assert.Equal(t, "cmd", normalized[0].SourceVariable)
	assert.Equal(t, "user_input", normalized[0].SourceCategory)
	assert.Equal(t, "command_exec", normalized[0].SinkCategory)
}

// TestFlowsMatch_LineThreshold tests line number fuzzy matching.
func TestFlowsMatch_LineThreshold(t *testing.T) {
	config := DefaultMatchConfig()

	f1 := NormalizedTaintFlow{
		SourceLine:        10,
		SourceVariable:    "x",
		SourceCategory:    "user_input",
		SinkLine:          20,
		SinkVariable:      "y",
		SinkCategory:      "sql_execution",
		VulnerabilityType: "SQL_INJECTION",
	}

	// Within ±2 lines: SHOULD match
	f2 := NormalizedTaintFlow{
		SourceLine:        11, // +1 line
		SourceVariable:    "x",
		SourceCategory:    "user_input",
		SinkLine:          19, // -1 line
		SinkVariable:      "y",
		SinkCategory:      "sql_execution",
		VulnerabilityType: "SQL_INJECTION",
	}

	assert.True(t, FlowsMatch(f1, f2, config))

	// Outside threshold: should NOT match
	f3 := NormalizedTaintFlow{
		SourceLine:        15, // +5 lines
		SourceVariable:    "x",
		SourceCategory:    "user_input",
		SinkLine:          20,
		SinkVariable:      "y",
		SinkCategory:      "sql_execution",
		VulnerabilityType: "SQL_INJECTION",
	}

	assert.False(t, FlowsMatch(f1, f3, config))
}

// TestFlowsMatch_SSAVariables tests SSA variable alias matching.
func TestFlowsMatch_SSAVariables(t *testing.T) {
	config := DefaultMatchConfig()

	f1 := NormalizedTaintFlow{
		SourceLine:        10,
		SourceVariable:    "user_input",
		SourceCategory:    "user_input",
		SinkLine:          20,
		SinkVariable:      "query",
		SinkCategory:      "sql_execution",
		VulnerabilityType: "SQL_INJECTION",
	}

	// With SSA suffix _1: SHOULD match
	f2 := NormalizedTaintFlow{
		SourceLine:        10,
		SourceVariable:    "user_input_1", // SSA renamed
		SourceCategory:    "user_input",
		SinkLine:          20,
		SinkVariable:      "query_2", // SSA renamed
		SinkCategory:      "sql_execution",
		VulnerabilityType: "SQL_INJECTION",
	}

	assert.True(t, FlowsMatch(f1, f2, config))
}

// TestFlowsMatch_VulnTypes tests semantic vulnerability type matching.
func TestFlowsMatch_VulnTypes(t *testing.T) {
	config := DefaultMatchConfig()

	f1 := NormalizedTaintFlow{
		SourceLine:        10,
		SourceVariable:    "x",
		SourceCategory:    "user_input",
		SinkLine:          20,
		SinkVariable:      "y",
		SinkCategory:      "sql_execution",
		VulnerabilityType: "SQL_INJECTION",
	}

	// Different but equivalent vuln type: SHOULD match
	f2 := NormalizedTaintFlow{
		SourceLine:        10,
		SourceVariable:    "x",
		SourceCategory:    "user_input",
		SinkLine:          20,
		SinkVariable:      "y",
		SinkCategory:      "sql_execution",
		VulnerabilityType: "sqli", // Lowercase variant
	}

	assert.True(t, FlowsMatch(f1, f2, config))
}

// TestStripSSASuffix tests SSA suffix removal.
func TestStripSSASuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user_input_1", "user_input"},
		{"user_input_2", "user_input"},
		{"query_10", "query"},
		{"normal_var", "normal_var"},
		{"x", "x"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := stripSSASuffix(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestNormalizeVulnType tests vulnerability type normalization.
func TestNormalizeVulnType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"SQL_INJECTION", "SQL_INJECTION"},
		{"sqli", "SQL_INJECTION"},
		{"SQL INJECTION", "SQL_INJECTION"},
		{"COMMAND_INJECTION", "COMMAND_INJECTION"},
		{"cmd_injection", "COMMAND_INJECTION"},
		{"XSS", "XSS"},
		{"cross_site_scripting", "XSS"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeVulnType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestCategorizeLLMPattern tests LLM pattern categorization.
func TestCategorizeLLMPattern(t *testing.T) {
	tests := []struct {
		pattern  string
		expected string
	}{
		{"request.GET['cmd']", "user_input"},
		{"request.POST", "user_input"},
		{"input()", "user_input"},
		{"cursor.execute", "sql_execution"},
		{"os.system", "command_exec"},
		{"subprocess.call", "command_exec"},
		{"eval", "code_exec"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			result := categorizeLLMPattern(tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}
