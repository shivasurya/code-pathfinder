package diagnostic

import (
	"strings"
)

// DualLevelComparison represents comparison results at both binary and detailed levels.
type DualLevelComparison struct {
	FunctionFQN string

	// Level 1: Binary classification
	BinaryToolResult bool // Tool says: has flow
	BinaryLLMResult  bool // LLM says: has flow
	BinaryAgreement  bool // Do they agree?

	// Level 2: Detailed flow comparison (only if both found flows)
	DetailedComparison *FlowComparisonResult // nil if N/A

	// Metrics
	Precision float64
	Recall    float64
	F1Score   float64

	// Categorization (if disagreement)
	FailureCategory string // "control_flow", "sanitizer", etc.
	FailureReason   string // From LLM reasoning
}

// FlowComparisonResult contains detailed flow-by-flow comparison.
type FlowComparisonResult struct {
	ToolFlows []NormalizedTaintFlow
	LLMFlows  []NormalizedTaintFlow

	Matches       []FlowMatch           // TP: Both found
	UnmatchedTool []NormalizedTaintFlow // FP: Tool only
	UnmatchedLLM  []NormalizedTaintFlow // FN: LLM only

	FlowPrecision float64 // Matches / ToolFlows
	FlowRecall    float64 // Matches / LLMFlows
	FlowF1Score   float64 // 2PR/(P+R)
}

// FlowMatch represents a matched flow between tool and LLM.
type FlowMatch struct {
	ToolFlow  NormalizedTaintFlow
	LLMFlow   NormalizedTaintFlow
	ToolIndex int
	LLMIndex  int
}

// CompareFunctionResults performs dual-level comparison between tool and LLM results.
//
// Performance: ~1ms per function
//
// Example:
//
//	comparison := CompareFunctionResults(fn, toolResult, llmResult)
//	if comparison.BinaryAgreement {
//	    fmt.Println("✅ Agreement on binary level")
//	}
//	if comparison.DetailedComparison != nil {
//	    fmt.Printf("Flow precision: %.2f%%\n", comparison.Precision*100)
//	}
func CompareFunctionResults(
	fn *FunctionMetadata,
	toolResult *FunctionTaintResult,
	llmResult *LLMAnalysisResult,
) *DualLevelComparison {
	// Determine if LLM detected any dataflow (not just dangerous flows)
	llmHasFlow := llmResult.AnalysisMetadata.TotalFlows > 0

	comparison := &DualLevelComparison{
		FunctionFQN:      fn.FQN,
		BinaryToolResult: toolResult.HasTaintFlow,
		BinaryLLMResult:  llmHasFlow,
		BinaryAgreement:  toolResult.HasTaintFlow == llmHasFlow,
	}

	// Level 2: Detailed comparison (only if both found flows)
	if toolResult.HasTaintFlow && llmHasFlow {
		toolNorm := NormalizeToolResult(toolResult)
		llmNorm := NormalizeLLMResult(llmResult)

		flowComparison := CompareNormalizedFlows(toolNorm, llmNorm, DefaultMatchConfig())

		comparison.DetailedComparison = flowComparison
		comparison.Precision = flowComparison.FlowPrecision
		comparison.Recall = flowComparison.FlowRecall
		comparison.F1Score = flowComparison.FlowF1Score
	} else if !comparison.BinaryAgreement {
		// Binary disagreement: Categorize failure
		comparison.FailureCategory = categorizeFailureFromLLM(llmResult)
		comparison.FailureReason = extractReasoningFromLLM(llmResult)
	}

	return comparison
}

// CompareNormalizedFlows performs detailed flow-by-flow comparison with fuzzy matching.
func CompareNormalizedFlows(
	toolFlows, llmFlows []NormalizedTaintFlow,
	config MatchConfig,
) *FlowComparisonResult {
	result := &FlowComparisonResult{
		ToolFlows: toolFlows,
		LLMFlows:  llmFlows,
		Matches:   []FlowMatch{},
	}

	matched := make(map[int]bool) // Track which LLM flows are matched

	// For each tool flow, try to find matching LLM flow
	for i, toolFlow := range toolFlows {
		foundMatch := false
		for j, llmFlow := range llmFlows {
			if matched[j] {
				continue // Already matched
			}
			if FlowsMatch(toolFlow, llmFlow, config) {
				result.Matches = append(result.Matches, FlowMatch{
					ToolFlow:  toolFlow,
					LLMFlow:   llmFlow,
					ToolIndex: i,
					LLMIndex:  j,
				})
				matched[j] = true
				foundMatch = true
				break
			}
		}

		if !foundMatch {
			result.UnmatchedTool = append(result.UnmatchedTool, toolFlow)
		}
	}

	// Identify unmatched LLM flows (FN)
	for i, llmFlow := range llmFlows {
		if !matched[i] {
			result.UnmatchedLLM = append(result.UnmatchedLLM, llmFlow)
		}
	}

	// Calculate metrics
	if len(toolFlows) > 0 {
		result.FlowPrecision = float64(len(result.Matches)) / float64(len(toolFlows))
	}
	if len(llmFlows) > 0 {
		result.FlowRecall = float64(len(result.Matches)) / float64(len(llmFlows))
	}
	if result.FlowPrecision+result.FlowRecall > 0 {
		result.FlowF1Score = 2 * result.FlowPrecision * result.FlowRecall /
			(result.FlowPrecision + result.FlowRecall)
	}

	return result
}

// categorizeFailureFromLLM extracts failure category from LLM analysis.
// First tries to use LLM-provided category, falls back to keyword matching.
func categorizeFailureFromLLM(llmResult *LLMAnalysisResult) string {
	// Strategy 1: Use LLM-provided category (most reliable)
	for _, testCase := range llmResult.DataflowTestCases {
		if testCase.FailureCategory != "" && testCase.FailureCategory != "none" {
			return testCase.FailureCategory
		}
	}

	// Strategy 2: Fallback to keyword matching (for older LLM responses)
	for _, testCase := range llmResult.DataflowTestCases {
		reasoning := strings.ToLower(testCase.Reasoning)

		// Check sanitizers first (high priority issue)
		if strings.Contains(reasoning, "sanitiz") || strings.Contains(reasoning, "escape") ||
			strings.Contains(reasoning, "quote") || strings.Contains(reasoning, "clean") ||
			strings.Contains(reasoning, "filter") || strings.Contains(reasoning, "validate") {
			return "sanitizer_missed"
		}

		// Control flow branches (high priority - common limitation)
		if strings.Contains(reasoning, "if ") || strings.Contains(reasoning, "branch") ||
			strings.Contains(reasoning, "conditional") || strings.Contains(reasoning, "else") ||
			strings.Contains(reasoning, "inside") {
			return "control_flow_branch"
		}

		// Field sensitivity (object attribute tracking)
		if (strings.Contains(reasoning, "field") || strings.Contains(reasoning, "attribute") ||
			strings.Contains(reasoning, "self.") || strings.Contains(reasoning, "obj.")) &&
			!strings.Contains(reasoning, "dict") {
			return "field_sensitivity"
		}

		// Container operations (list/dict/set)
		if strings.Contains(reasoning, "list") || strings.Contains(reasoning, "dict") ||
			strings.Contains(reasoning, "append") || strings.Contains(reasoning, "array") ||
			strings.Contains(reasoning, "container") || strings.Contains(reasoning, "[") {
			return "container_operation"
		}

		// String formatting operations
		if strings.Contains(reasoning, "f-string") || strings.Contains(reasoning, "format") ||
			strings.Contains(reasoning, "concatenat") || strings.Contains(reasoning, "join") ||
			strings.Contains(reasoning, "%s") || strings.Contains(reasoning, ".format(") {
			return "string_formatting"
		}

		// Method call propagation
		if strings.Contains(reasoning, "method") || strings.Contains(reasoning, ".upper()") ||
			strings.Contains(reasoning, ".lower()") || strings.Contains(reasoning, ".strip()") ||
			strings.Contains(reasoning, "string method") {
			return "method_call_propagation"
		}

		// Assignment chain tracking
		if strings.Contains(reasoning, "assignment") && (strings.Contains(reasoning, "chain") ||
			strings.Contains(reasoning, "through") || strings.Contains(reasoning, "via")) {
			return "assignment_chain"
		}

		// Return flow tracking
		if strings.Contains(reasoning, "return") && strings.Contains(reasoning, "flow") {
			return "return_flow"
		}

		// Function parameter flow
		if strings.Contains(reasoning, "parameter") && strings.Contains(reasoning, "flow") {
			return "parameter_flow"
		}

		// Complex expressions (method chains, nested calls)
		if strings.Contains(reasoning, "complex") || strings.Contains(reasoning, "nested") ||
			strings.Contains(reasoning, "chain") || strings.Contains(reasoning, "multiple") {
			return "complex_expression"
		}

		// Inter-procedural (out of scope for intra-procedural analysis)
		if strings.Contains(reasoning, "function call") || strings.Contains(reasoning, "called function") ||
			strings.Contains(reasoning, "inter-procedural") || strings.Contains(reasoning, "cross-function") {
			return "context_required"
		}
	}

	return "unknown"
}

// extractReasoningFromLLM gets the reasoning from first test case.
func extractReasoningFromLLM(llmResult *LLMAnalysisResult) string {
	if len(llmResult.DataflowTestCases) > 0 {
		return llmResult.DataflowTestCases[0].Reasoning
	}
	return ""
}
