package diagnostic

import (
	"time"
)

// OverallMetrics contains aggregated metrics across all functions.
type OverallMetrics struct {
	// Total functions analyzed
	TotalFunctions int

	// Confusion Matrix
	TruePositives  int // Tool detected, LLM confirmed ✅
	FalsePositives int // Tool detected, LLM says safe ⚠️
	FalseNegatives int // Tool missed, LLM found vuln ❌
	TrueNegatives  int // Tool skipped, LLM confirmed safe ✅

	// Metrics
	Precision float64 // TP / (TP + FP)
	Recall    float64 // TP / (TP + FN)
	F1Score   float64 // 2 * (P * R) / (P + R)
	Agreement float64 // (TP + TN) / Total

	// Processing stats
	LLMProcessingTime   string
	TotalProcessingTime string
	FunctionsPerSecond  float64

	// Failure breakdown
	FailuresByCategory map[string]int
	TopFailures        []FailureExample
}

// FailureExample represents a specific failure case.
type FailureExample struct {
	Type         string // "FALSE_POSITIVE", "FALSE_NEGATIVE"
	FunctionFQN  string
	FunctionFile string
	FunctionLine int
	Category     string              // "control_flow", "sanitizer", etc.
	Reason       string              // From LLM
	Flow         *NormalizedTaintFlow // Flow details (if applicable)
}

// CalculateOverallMetrics aggregates metrics from all function comparisons.
//
// Performance: O(n) where n = number of comparisons
//
// Example:
//
//	metrics := CalculateOverallMetrics(comparisons, startTime)
//	fmt.Printf("Precision: %.1f%%\n", metrics.Precision*100)
//	fmt.Printf("Recall: %.1f%%\n", metrics.Recall*100)
//	fmt.Printf("F1 Score: %.1f%%\n", metrics.F1Score*100)
func CalculateOverallMetrics(
	comparisons []*DualLevelComparison,
	startTime time.Time,
) *OverallMetrics {
	metrics := &OverallMetrics{
		TotalFunctions:     len(comparisons),
		FailuresByCategory: make(map[string]int),
		TopFailures:        []FailureExample{},
	}

	// Calculate confusion matrix
	for _, cmp := range comparisons {
		switch {
		case cmp.BinaryToolResult && cmp.BinaryLLMResult:
			metrics.TruePositives++
		case cmp.BinaryToolResult && !cmp.BinaryLLMResult:
			metrics.FalsePositives++
		case !cmp.BinaryToolResult && cmp.BinaryLLMResult:
			metrics.FalseNegatives++
		default:
			metrics.TrueNegatives++
		}

		// Track failure categories
		if !cmp.BinaryAgreement && cmp.FailureCategory != "" {
			metrics.FailuresByCategory[cmp.FailureCategory]++
		}
	}

	// Calculate metrics
	if metrics.TruePositives+metrics.FalsePositives > 0 {
		metrics.Precision = float64(metrics.TruePositives) /
			float64(metrics.TruePositives+metrics.FalsePositives)
	}

	if metrics.TruePositives+metrics.FalseNegatives > 0 {
		metrics.Recall = float64(metrics.TruePositives) /
			float64(metrics.TruePositives+metrics.FalseNegatives)
	}

	if metrics.Precision+metrics.Recall > 0 {
		metrics.F1Score = 2 * metrics.Precision * metrics.Recall /
			(metrics.Precision + metrics.Recall)
	}

	if metrics.TotalFunctions > 0 {
		metrics.Agreement = float64(metrics.TruePositives+metrics.TrueNegatives) /
			float64(metrics.TotalFunctions)
	}

	// Processing stats
	totalDuration := time.Since(startTime)
	metrics.TotalProcessingTime = totalDuration.String()
	if totalDuration.Seconds() > 0 {
		metrics.FunctionsPerSecond = float64(metrics.TotalFunctions) / totalDuration.Seconds()
	}

	return metrics
}

// ExtractTopFailures extracts the most important failure examples.
// Returns up to maxPerType failures of each type (FP/FN).
func ExtractTopFailures(
	comparisons []*DualLevelComparison,
	functionsMap map[string]*FunctionMetadata,
	maxPerType int,
) []FailureExample {
	failures := []FailureExample{}

	fpCount := 0
	fnCount := 0

	for _, cmp := range comparisons {
		fn := functionsMap[cmp.FunctionFQN]
		if fn == nil {
			continue
		}

		// False Positives
		if cmp.BinaryToolResult && !cmp.BinaryLLMResult {
			if fpCount < maxPerType {
				failures = append(failures, FailureExample{
					Type:         "FALSE_POSITIVE",
					FunctionFQN:  cmp.FunctionFQN,
					FunctionFile: fn.FilePath,
					FunctionLine: fn.StartLine,
					Category:     cmp.FailureCategory,
					Reason:       cmp.FailureReason,
				})
				fpCount++
			}
		}

		// False Negatives
		if !cmp.BinaryToolResult && cmp.BinaryLLMResult {
			if fnCount < maxPerType {
				failures = append(failures, FailureExample{
					Type:         "FALSE_NEGATIVE",
					FunctionFQN:  cmp.FunctionFQN,
					FunctionFile: fn.FilePath,
					FunctionLine: fn.StartLine,
					Category:     cmp.FailureCategory,
					Reason:       cmp.FailureReason,
				})
				fnCount++
			}
		}
	}

	return failures
}
