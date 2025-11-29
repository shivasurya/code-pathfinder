package diagnostic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompareFunctionResults_BinaryTP tests binary true positive (both detect flow).
func TestCompareFunctionResults_BinaryTP(t *testing.T) {
	fn := &FunctionMetadata{FQN: "test.func"}

	toolResult := &FunctionTaintResult{
		HasTaintFlow: true,
		TaintFlows: []ToolTaintFlow{
			{
				SourceLine:        10,
				SourceVariable:    "x",
				SourceCategory:    "user_input",
				SinkLine:          20,
				SinkVariable:      "y",
				SinkCategory:      "sql_execution",
				VulnerabilityType: "SQL_INJECTION",
			},
		},
	}

	llmResult := &LLMAnalysisResult{
		AnalysisMetadata: AnalysisMetadata{
			DangerousFlows: 1,
			TotalFlows:     1,
		},
		DataflowTestCases: []DataflowTestCase{
			{
				Source:            TestCaseSource{Line: 10, Variable: "x"},
				Sink:              TestCaseSink{Line: 20, Variable: "y"},
				ExpectedDetection: true,
			},
		},
	}

	comparison := CompareFunctionResults(fn, toolResult, llmResult)

	assert.True(t, comparison.BinaryAgreement)
	assert.True(t, comparison.BinaryToolResult)
	assert.True(t, comparison.BinaryLLMResult)
	assert.NotNil(t, comparison.DetailedComparison)
}

// TestCompareFunctionResults_BinaryTN tests binary true negative (both say no flow).
func TestCompareFunctionResults_BinaryTN(t *testing.T) {
	fn := &FunctionMetadata{FQN: "test.func"}

	toolResult := &FunctionTaintResult{
		HasTaintFlow: false,
	}

	llmResult := &LLMAnalysisResult{
		AnalysisMetadata: AnalysisMetadata{
			DangerousFlows: 0,
		},
	}

	comparison := CompareFunctionResults(fn, toolResult, llmResult)

	assert.True(t, comparison.BinaryAgreement)
	assert.False(t, comparison.BinaryToolResult)
	assert.False(t, comparison.BinaryLLMResult)
	assert.Nil(t, comparison.DetailedComparison)
}

// TestCompareFunctionResults_BinaryFP tests false positive (tool detects, LLM doesn't).
func TestCompareFunctionResults_BinaryFP(t *testing.T) {
	fn := &FunctionMetadata{FQN: "test.func"}

	toolResult := &FunctionTaintResult{
		HasTaintFlow: true,
	}

	llmResult := &LLMAnalysisResult{
		AnalysisMetadata: AnalysisMetadata{
			DangerousFlows: 0,
		},
		DataflowTestCases: []DataflowTestCase{
			{
				ExpectedDetection: false,
				Reasoning:         "This flow is sanitized",
			},
		},
	}

	comparison := CompareFunctionResults(fn, toolResult, llmResult)

	assert.False(t, comparison.BinaryAgreement)
	assert.True(t, comparison.BinaryToolResult)
	assert.False(t, comparison.BinaryLLMResult)
	assert.NotEmpty(t, comparison.FailureReason)
}

// TestCompareFunctionResults_BinaryFN tests false negative (LLM detects, tool doesn't).
func TestCompareFunctionResults_BinaryFN(t *testing.T) {
	fn := &FunctionMetadata{FQN: "test.func"}

	toolResult := &FunctionTaintResult{
		HasTaintFlow: false,
	}

	llmResult := &LLMAnalysisResult{
		AnalysisMetadata: AnalysisMetadata{
			DangerousFlows: 1,
			TotalFlows:     1,
		},
		DataflowTestCases: []DataflowTestCase{
			{
				ExpectedDetection: true,
				Reasoning:         "Flow should be detected through control flow",
				FailureCategory:   "control_flow_branch",
			},
		},
	}

	comparison := CompareFunctionResults(fn, toolResult, llmResult)

	assert.False(t, comparison.BinaryAgreement)
	assert.False(t, comparison.BinaryToolResult)
	assert.True(t, comparison.BinaryLLMResult)
	// Failure category should be set (might be "control_flow_branch" or "unknown" depending on reasoning)
	assert.NotEmpty(t, comparison.FailureCategory)
}

// TestCompareNormalizedFlows_AllMatch tests when all flows match.
func TestCompareNormalizedFlows_AllMatch(t *testing.T) {
	toolFlows := []NormalizedTaintFlow{
		{
			SourceLine:        10,
			SourceVariable:    "x",
			SourceCategory:    "user_input",
			SinkLine:          20,
			SinkVariable:      "y",
			SinkCategory:      "sql_execution",
			VulnerabilityType: "SQL_INJECTION",
		},
	}

	llmFlows := []NormalizedTaintFlow{
		{
			SourceLine:        10,
			SourceVariable:    "x",
			SourceCategory:    "user_input",
			SinkLine:          20,
			SinkVariable:      "y",
			SinkCategory:      "sql_execution",
			VulnerabilityType: "SQL_INJECTION",
		},
	}

	result := CompareNormalizedFlows(toolFlows, llmFlows, DefaultMatchConfig())

	assert.Equal(t, 1, len(result.Matches))
	assert.Equal(t, 0, len(result.UnmatchedTool))
	assert.Equal(t, 0, len(result.UnmatchedLLM))
	assert.Equal(t, 1.0, result.FlowPrecision)
	assert.Equal(t, 1.0, result.FlowRecall)
	assert.Equal(t, 1.0, result.FlowF1Score)
}

// TestCompareNormalizedFlows_PartialMatch tests partial matching.
func TestCompareNormalizedFlows_PartialMatch(t *testing.T) {
	toolFlows := []NormalizedTaintFlow{
		{
			SourceLine:        10,
			SourceVariable:    "x",
			SourceCategory:    "user_input",
			SinkLine:          20,
			SinkVariable:      "y",
			SinkCategory:      "sql_execution",
			VulnerabilityType: "SQL_INJECTION",
		},
		{
			SourceLine:        30,
			SourceVariable:    "a",
			SourceCategory:    "user_input",
			SinkLine:          40,
			SinkVariable:      "b",
			SinkCategory:      "command_exec",
			VulnerabilityType: "COMMAND_INJECTION",
		},
	}

	llmFlows := []NormalizedTaintFlow{
		{
			SourceLine:        10,
			SourceVariable:    "x",
			SourceCategory:    "user_input",
			SinkLine:          20,
			SinkVariable:      "y",
			SinkCategory:      "sql_execution",
			VulnerabilityType: "SQL_INJECTION",
		},
	}

	result := CompareNormalizedFlows(toolFlows, llmFlows, DefaultMatchConfig())

	assert.Equal(t, 1, len(result.Matches))
	assert.Equal(t, 1, len(result.UnmatchedTool)) // Tool found extra flow
	assert.Equal(t, 0, len(result.UnmatchedLLM))
	assert.Equal(t, 0.5, result.FlowPrecision) // 1/2
	assert.Equal(t, 1.0, result.FlowRecall)    // 1/1
}

// TestCategorizeFailureFromLLM tests failure categorization.
func TestCategorizeFailureFromLLM(t *testing.T) {
	tests := []struct {
		reasoning        string
		expectedCategory string
	}{
		{"Flow depends on if condition", "control_flow_branch"},
		{"Field access through self.field", "field_sensitivity"},
		{"Data is sanitized by escape function", "sanitizer_missed"},
		{"Flow through list append", "container_operation"},
		{"Flow through f-string formatting", "string_formatting"},
		{"Unknown reason", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedCategory, func(t *testing.T) {
			llmResult := &LLMAnalysisResult{
				DataflowTestCases: []DataflowTestCase{
					{Reasoning: tt.reasoning},
				},
			}

			category := categorizeFailureFromLLM(llmResult)
			assert.Equal(t, tt.expectedCategory, category)
		})
	}
}

// TestExtractReasoningFromLLM tests reasoning extraction.
func TestExtractReasoningFromLLM(t *testing.T) {
	llmResult := &LLMAnalysisResult{
		DataflowTestCases: []DataflowTestCase{
			{Reasoning: "First reasoning"},
			{Reasoning: "Second reasoning"},
		},
	}

	reasoning := extractReasoningFromLLM(llmResult)
	assert.Equal(t, "First reasoning", reasoning)

	// Empty case
	emptyResult := &LLMAnalysisResult{}
	reasoning = extractReasoningFromLLM(emptyResult)
	assert.Empty(t, reasoning)
}

// TestCompareNormalizedFlows_EmptyFlows tests with empty flow lists.
func TestCompareNormalizedFlows_EmptyFlows(t *testing.T) {
	toolFlows := []NormalizedTaintFlow{}
	llmFlows := []NormalizedTaintFlow{}

	result := CompareNormalizedFlows(toolFlows, llmFlows, DefaultMatchConfig())

	require.NotNil(t, result)
	assert.Equal(t, 0, len(result.Matches))
	assert.Equal(t, 0.0, result.FlowPrecision)
	assert.Equal(t, 0.0, result.FlowRecall)
	assert.Equal(t, 0.0, result.FlowF1Score)
}
