package diagnostic

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCalculateOverallMetrics_AllTruePositives tests perfect agreement (all TP).
func TestCalculateOverallMetrics_AllTruePositives(t *testing.T) {
	comparisons := []*DualLevelComparison{
		{
			FunctionFQN:      "test.func1",
			BinaryToolResult: true,
			BinaryLLMResult:  true,
			BinaryAgreement:  true,
		},
		{
			FunctionFQN:      "test.func2",
			BinaryToolResult: true,
			BinaryLLMResult:  true,
			BinaryAgreement:  true,
		},
	}

	startTime := time.Now().Add(-2 * time.Second)
	metrics := CalculateOverallMetrics(comparisons, startTime)

	assert.Equal(t, 2, metrics.TotalFunctions)
	assert.Equal(t, 2, metrics.TruePositives)
	assert.Equal(t, 0, metrics.FalsePositives)
	assert.Equal(t, 0, metrics.FalseNegatives)
	assert.Equal(t, 0, metrics.TrueNegatives)
	assert.Equal(t, 1.0, metrics.Precision)
	assert.Equal(t, 1.0, metrics.Recall)
	assert.Equal(t, 1.0, metrics.F1Score)
	assert.Equal(t, 1.0, metrics.Agreement)
	assert.Greater(t, metrics.FunctionsPerSecond, 0.0)
}

// TestCalculateOverallMetrics_AllTrueNegatives tests perfect agreement (all TN).
func TestCalculateOverallMetrics_AllTrueNegatives(t *testing.T) {
	comparisons := []*DualLevelComparison{
		{
			FunctionFQN:      "test.func1",
			BinaryToolResult: false,
			BinaryLLMResult:  false,
			BinaryAgreement:  true,
		},
		{
			FunctionFQN:      "test.func2",
			BinaryToolResult: false,
			BinaryLLMResult:  false,
			BinaryAgreement:  true,
		},
	}

	startTime := time.Now().Add(-1 * time.Second)
	metrics := CalculateOverallMetrics(comparisons, startTime)

	assert.Equal(t, 2, metrics.TotalFunctions)
	assert.Equal(t, 0, metrics.TruePositives)
	assert.Equal(t, 0, metrics.FalsePositives)
	assert.Equal(t, 0, metrics.FalseNegatives)
	assert.Equal(t, 2, metrics.TrueNegatives)
	assert.Equal(t, 0.0, metrics.Precision) // No TP + FP
	assert.Equal(t, 0.0, metrics.Recall)    // No TP + FN
	assert.Equal(t, 0.0, metrics.F1Score)
	assert.Equal(t, 1.0, metrics.Agreement)
}

// TestCalculateOverallMetrics_MixedResults tests confusion matrix calculation.
func TestCalculateOverallMetrics_MixedResults(t *testing.T) {
	comparisons := []*DualLevelComparison{
		// TP
		{
			FunctionFQN:      "test.tp",
			BinaryToolResult: true,
			BinaryLLMResult:  true,
			BinaryAgreement:  true,
		},
		// FP
		{
			FunctionFQN:      "test.fp",
			BinaryToolResult: true,
			BinaryLLMResult:  false,
			BinaryAgreement:  false,
			FailureCategory:  "sanitizer_missed",
		},
		// FN
		{
			FunctionFQN:      "test.fn",
			BinaryToolResult: false,
			BinaryLLMResult:  true,
			BinaryAgreement:  false,
			FailureCategory:  "control_flow_branch",
		},
		// TN
		{
			FunctionFQN:      "test.tn",
			BinaryToolResult: false,
			BinaryLLMResult:  false,
			BinaryAgreement:  true,
		},
	}

	startTime := time.Now().Add(-5 * time.Second)
	metrics := CalculateOverallMetrics(comparisons, startTime)

	assert.Equal(t, 4, metrics.TotalFunctions)
	assert.Equal(t, 1, metrics.TruePositives)
	assert.Equal(t, 1, metrics.FalsePositives)
	assert.Equal(t, 1, metrics.FalseNegatives)
	assert.Equal(t, 1, metrics.TrueNegatives)

	// Precision = TP / (TP + FP) = 1 / 2 = 0.5
	assert.Equal(t, 0.5, metrics.Precision)

	// Recall = TP / (TP + FN) = 1 / 2 = 0.5
	assert.Equal(t, 0.5, metrics.Recall)

	// F1 = 2 * P * R / (P + R) = 2 * 0.5 * 0.5 / 1.0 = 0.5
	assert.Equal(t, 0.5, metrics.F1Score)

	// Agreement = (TP + TN) / Total = 2 / 4 = 0.5
	assert.Equal(t, 0.5, metrics.Agreement)

	// Failure categories
	assert.Equal(t, 2, len(metrics.FailuresByCategory))
	assert.Equal(t, 1, metrics.FailuresByCategory["sanitizer_missed"])
	assert.Equal(t, 1, metrics.FailuresByCategory["control_flow_branch"])
}

// TestCalculateOverallMetrics_ZeroDivision tests edge case with no detections.
func TestCalculateOverallMetrics_ZeroDivision(t *testing.T) {
	comparisons := []*DualLevelComparison{}

	startTime := time.Now()
	metrics := CalculateOverallMetrics(comparisons, startTime)

	assert.Equal(t, 0, metrics.TotalFunctions)
	assert.Equal(t, 0.0, metrics.Precision)
	assert.Equal(t, 0.0, metrics.Recall)
	assert.Equal(t, 0.0, metrics.F1Score)
	assert.Equal(t, 0.0, metrics.Agreement)
}

// TestCalculateOverallMetrics_FailureCategoryCounting tests category aggregation.
func TestCalculateOverallMetrics_FailureCategoryCounting(t *testing.T) {
	comparisons := []*DualLevelComparison{
		{
			FunctionFQN:      "test.fp1",
			BinaryToolResult: true,
			BinaryLLMResult:  false,
			BinaryAgreement:  false,
			FailureCategory:  "sanitizer_missed",
		},
		{
			FunctionFQN:      "test.fp2",
			BinaryToolResult: true,
			BinaryLLMResult:  false,
			BinaryAgreement:  false,
			FailureCategory:  "sanitizer_missed",
		},
		{
			FunctionFQN:      "test.fn1",
			BinaryToolResult: false,
			BinaryLLMResult:  true,
			BinaryAgreement:  false,
			FailureCategory:  "control_flow_branch",
		},
		{
			FunctionFQN:      "test.tp",
			BinaryToolResult: true,
			BinaryLLMResult:  true,
			BinaryAgreement:  true,
		},
	}

	startTime := time.Now()
	metrics := CalculateOverallMetrics(comparisons, startTime)

	require.NotNil(t, metrics.FailuresByCategory)
	assert.Equal(t, 2, metrics.FailuresByCategory["sanitizer_missed"])
	assert.Equal(t, 1, metrics.FailuresByCategory["control_flow_branch"])
}

// TestExtractTopFailures tests failure example extraction.
func TestExtractTopFailures(t *testing.T) {
	functionsMap := map[string]*FunctionMetadata{
		"test.fp1": {
			FQN:       "test.fp1",
			FilePath:  "test.py",
			StartLine: 10,
		},
		"test.fp2": {
			FQN:       "test.fp2",
			FilePath:  "test.py",
			StartLine: 20,
		},
		"test.fn1": {
			FQN:       "test.fn1",
			FilePath:  "test.py",
			StartLine: 30,
		},
	}

	comparisons := []*DualLevelComparison{
		// FP
		{
			FunctionFQN:      "test.fp1",
			BinaryToolResult: true,
			BinaryLLMResult:  false,
			BinaryAgreement:  false,
			FailureCategory:  "sanitizer_missed",
			FailureReason:    "Data is sanitized",
		},
		// FP
		{
			FunctionFQN:      "test.fp2",
			BinaryToolResult: true,
			BinaryLLMResult:  false,
			BinaryAgreement:  false,
			FailureCategory:  "sanitizer_missed",
			FailureReason:    "Escape function used",
		},
		// FN
		{
			FunctionFQN:      "test.fn1",
			BinaryToolResult: false,
			BinaryLLMResult:  true,
			BinaryAgreement:  false,
			FailureCategory:  "control_flow_branch",
			FailureReason:    "Flow through if branch",
		},
	}

	failures := ExtractTopFailures(comparisons, functionsMap, 2)

	require.NotNil(t, failures)
	// Should return up to 2 of each type (FP, FN)
	assert.LessOrEqual(t, len(failures), 4)

	// Count types
	fpCount := 0
	fnCount := 0
	for _, f := range failures {
		if f.Type == "FALSE_POSITIVE" {
			fpCount++
			assert.Contains(t, []string{"test.fp1", "test.fp2"}, f.FunctionFQN)
		} else if f.Type == "FALSE_NEGATIVE" {
			fnCount++
			assert.Equal(t, "test.fn1", f.FunctionFQN)
		}
	}

	assert.LessOrEqual(t, fpCount, 2)
	assert.LessOrEqual(t, fnCount, 2)
}

// TestExtractTopFailures_EmptyComparisons tests with no failures.
func TestExtractTopFailures_EmptyComparisons(t *testing.T) {
	functionsMap := map[string]*FunctionMetadata{}
	comparisons := []*DualLevelComparison{}

	failures := ExtractTopFailures(comparisons, functionsMap, 5)

	require.NotNil(t, failures)
	assert.Equal(t, 0, len(failures))
}

// TestExtractTopFailures_LimitPerType tests maxPerType limit.
func TestExtractTopFailures_LimitPerType(t *testing.T) {
	functionsMap := map[string]*FunctionMetadata{
		"test.fp1": {FQN: "test.fp1", FilePath: "test.py", StartLine: 10},
		"test.fp2": {FQN: "test.fp2", FilePath: "test.py", StartLine: 20},
		"test.fp3": {FQN: "test.fp3", FilePath: "test.py", StartLine: 30},
	}

	comparisons := []*DualLevelComparison{
		{
			FunctionFQN:      "test.fp1",
			BinaryToolResult: true,
			BinaryLLMResult:  false,
			BinaryAgreement:  false,
			FailureCategory:  "sanitizer_missed",
			FailureReason:    "Sanitized 1",
		},
		{
			FunctionFQN:      "test.fp2",
			BinaryToolResult: true,
			BinaryLLMResult:  false,
			BinaryAgreement:  false,
			FailureCategory:  "sanitizer_missed",
			FailureReason:    "Sanitized 2",
		},
		{
			FunctionFQN:      "test.fp3",
			BinaryToolResult: true,
			BinaryLLMResult:  false,
			BinaryAgreement:  false,
			FailureCategory:  "sanitizer_missed",
			FailureReason:    "Sanitized 3",
		},
	}

	// Limit to 2 per type
	failures := ExtractTopFailures(comparisons, functionsMap, 2)

	require.NotNil(t, failures)
	// Should only return 2 FPs (limited by maxPerType)
	assert.Equal(t, 2, len(failures))

	for _, f := range failures {
		assert.Equal(t, "FALSE_POSITIVE", f.Type)
	}
}

// TestExtractTopFailures_MissingMetadata tests handling of missing function metadata.
func TestExtractTopFailures_MissingMetadata(t *testing.T) {
	functionsMap := map[string]*FunctionMetadata{
		// Only test.fp1 has metadata
		"test.fp1": {
			FQN:       "test.fp1",
			FilePath:  "test.py",
			StartLine: 10,
		},
	}

	comparisons := []*DualLevelComparison{
		{
			FunctionFQN:      "test.fp1",
			BinaryToolResult: true,
			BinaryLLMResult:  false,
			BinaryAgreement:  false,
			FailureCategory:  "sanitizer_missed",
			FailureReason:    "Sanitized",
		},
		{
			FunctionFQN:      "test.fp2", // No metadata
			BinaryToolResult: true,
			BinaryLLMResult:  false,
			BinaryAgreement:  false,
			FailureCategory:  "sanitizer_missed",
			FailureReason:    "Sanitized",
		},
	}

	failures := ExtractTopFailures(comparisons, functionsMap, 5)

	require.NotNil(t, failures)
	// Should only include test.fp1 (has metadata)
	assert.Equal(t, 1, len(failures))
	assert.Equal(t, "test.fp1", failures[0].FunctionFQN)
}

// TestCalculateOverallMetrics_ProcessingTime tests timing calculation.
func TestCalculateOverallMetrics_ProcessingTime(t *testing.T) {
	comparisons := []*DualLevelComparison{
		{
			FunctionFQN:      "test.func1",
			BinaryToolResult: true,
			BinaryLLMResult:  true,
			BinaryAgreement:  true,
		},
	}

	startTime := time.Now().Add(-3 * time.Second)
	metrics := CalculateOverallMetrics(comparisons, startTime)

	assert.NotEmpty(t, metrics.TotalProcessingTime)
	assert.Greater(t, metrics.FunctionsPerSecond, 0.0)
	// Should process ~0.33 functions/second (1 function in 3 seconds)
	assert.Less(t, metrics.FunctionsPerSecond, 1.0)
}
