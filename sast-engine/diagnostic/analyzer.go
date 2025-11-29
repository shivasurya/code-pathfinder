package diagnostic

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/analysis/taint"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/extraction"
)

// FunctionTaintResult represents the structured taint analysis result for a single function.
// This is the internal API (not user-facing) used for diagnostic comparison.
type FunctionTaintResult struct {
	// FunctionFQN identifies the function
	FunctionFQN string

	// HasTaintFlow indicates if ANY taint flow was detected (binary result)
	HasTaintFlow bool

	// TaintFlows contains all detected flows (detailed result)
	TaintFlows []ToolTaintFlow

	// AnalysisError indicates if analysis failed
	AnalysisError bool

	// ErrorMessage if AnalysisError == true
	ErrorMessage string
}

// ToolTaintFlow represents a single taint flow detected by our tool.
type ToolTaintFlow struct {
	// Source information
	SourceLine     int
	SourceVariable string
	SourceType     string // e.g., "request.GET['username']"
	SourceCategory string // e.g., "user_input" (semantic)

	// Sink information
	SinkLine       int
	SinkVariable   string
	SinkType       string // e.g., "sqlite3.execute"
	SinkCategory   string // e.g., "sql_execution" (semantic)

	// Flow details
	FlowPath []FlowStep

	// Metadata
	VulnerabilityType string  // e.g., "SQL_INJECTION"
	Confidence        float64 // 0.0-1.0
	IsSanitized       bool    // If sanitizer detected in path
}

// AnalyzeSingleFunction runs intra-procedural taint analysis on a single function.
// This wraps existing taint analysis logic but:
// 1. Analyzes ONLY the specified function (not whole codebase)
// 2. Returns structured result (not text)
// 3. Filters to intra-procedural flows only
//
// Performance: ~1-5ms per function (depends on function size)
//
// Example:
//
//	result, err := AnalyzeSingleFunction(functionMetadata, sources, sinks, sanitizers)
//	if err != nil {
//	    log.Printf("Analysis failed: %v", err)
//	    return nil, err
//	}
//	if result.HasTaintFlow {
//	    fmt.Printf("Found %d flows\n", len(result.TaintFlows))
//	}
func AnalyzeSingleFunction(
	fn *FunctionMetadata,
	sources []string,
	sinks []string,
	sanitizers []string,
) (*FunctionTaintResult, error) {
	result := &FunctionTaintResult{
		FunctionFQN:  fn.FQN,
		HasTaintFlow: false,
		TaintFlows:   []ToolTaintFlow{},
	}

	// Parse function source code
	sourceCode := []byte(fn.SourceCode)
	tree, err := extraction.ParsePythonFile(sourceCode)
	if err != nil {
		result.AnalysisError = true
		result.ErrorMessage = fmt.Sprintf("Parse error: %v", err)
		return result, nil // Return result with error flag (don't fail)
	}

	// Find the function node
	functionNode := findFunctionNodeByName(tree.RootNode(), fn.FunctionName, sourceCode)
	if functionNode == nil {
		result.AnalysisError = true
		result.ErrorMessage = "Function node not found in AST"
		return result, nil
	}

	// Extract statements (using existing logic from statement_extraction.go)
	statements, err := extraction.ExtractStatements(fn.FilePath, sourceCode, functionNode)
	if err != nil {
		result.AnalysisError = true
		result.ErrorMessage = fmt.Sprintf("Statement extraction error: %v", err)
		return result, nil
	}

	// Build def-use chains (using existing logic from statement.go)
	defUseChain := core.BuildDefUseChains(statements)

	// Run taint analysis (using existing logic from taint.go)
	taintSummary := taint.AnalyzeIntraProceduralTaint(
		fn.FQN,
		statements,
		defUseChain,
		sources,
		sinks,
		sanitizers,
	)

	// Check if any flows detected
	if !taintSummary.HasDetections() {
		return result, nil // No flows, return empty result
	}

	// Convert TaintSummary detections to ToolTaintFlow
	result.HasTaintFlow = true
	for _, detection := range taintSummary.Detections {
		// Only include if both source and sink are within function boundaries
		if detection.SourceLine >= uint32(fn.StartLine) &&
			detection.SourceLine <= uint32(fn.EndLine) &&
			detection.SinkLine >= uint32(fn.StartLine) &&
			detection.SinkLine <= uint32(fn.EndLine) {

			flow := ToolTaintFlow{
				SourceLine:     int(detection.SourceLine),
				SourceVariable: detection.SourceVar,
				SinkLine:       int(detection.SinkLine),
				SinkVariable:   detection.SinkVar,
				SinkType:       detection.SinkCall,
				Confidence:     detection.Confidence,
				IsSanitized:    detection.Sanitized,
			}

			// Build flow path from propagation path
			flow.FlowPath = []FlowStep{}
			for _, varName := range detection.PropagationPath {
				flow.FlowPath = append(flow.FlowPath, FlowStep{
					Variable:  varName,
					Operation: "propagate",
				})
			}

			// Categorize source and sink (semantic mapping)
			flow.SourceCategory = categorizePattern(flow.SourceType, sources)
			flow.SinkCategory = categorizePattern(flow.SinkType, sinks)
			flow.VulnerabilityType = inferVulnerabilityType(flow.SourceCategory, flow.SinkCategory)

			result.TaintFlows = append(result.TaintFlows, flow)
		}
	}

	return result, nil
}

// findFunctionNodeByName finds a function_definition node by name in the AST.
// Helper for AnalyzeSingleFunction.
func findFunctionNodeByName(node *sitter.Node, functionName string, sourceCode []byte) *sitter.Node {
	if node == nil {
		return nil
	}

	if node.Type() == "function_definition" {
		nameNode := node.ChildByFieldName("name")
		if nameNode != nil && nameNode.Content(sourceCode) == functionName {
			return node
		}
	}

	// Recurse into children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			result := findFunctionNodeByName(child, functionName, sourceCode)
			if result != nil {
				return result
			}
		}
	}

	return nil
}

// categorizePattern maps a pattern to a semantic category.
// Example: "request.GET" → "user_input", "os.system" → "command_exec".
func categorizePattern(pattern string, _ []string) string {
	patternLower := strings.ToLower(pattern)

	// User input sources
	if strings.Contains(patternLower, "request.get") ||
		strings.Contains(patternLower, "request.post") ||
		strings.Contains(patternLower, "input(") {
		return "user_input"
	}

	// File operations
	if strings.Contains(patternLower, "open(") ||
		strings.Contains(patternLower, "file") {
		return "file_operation"
	}

	// SQL sinks
	if strings.Contains(patternLower, "execute") ||
		strings.Contains(patternLower, "cursor") ||
		strings.Contains(patternLower, "sql") {
		return "sql_execution"
	}

	// Command execution sinks
	if strings.Contains(patternLower, "system") ||
		strings.Contains(patternLower, "subprocess") ||
		strings.Contains(patternLower, "popen") {
		return "command_exec"
	}

	// Code execution sinks
	if strings.Contains(patternLower, "eval") ||
		strings.Contains(patternLower, "exec") {
		return "code_exec"
	}

	return "other"
}

// inferVulnerabilityType maps source+sink categories to vulnerability type.
func inferVulnerabilityType(sourceCategory, sinkCategory string) string {
	if sourceCategory == "user_input" && sinkCategory == "sql_execution" {
		return "SQL_INJECTION"
	}
	if sourceCategory == "user_input" && sinkCategory == "command_exec" {
		return "COMMAND_INJECTION"
	}
	if sourceCategory == "user_input" && sinkCategory == "code_exec" {
		return "CODE_INJECTION"
	}
	if sourceCategory == "user_input" && sinkCategory == "file_operation" {
		return "PATH_TRAVERSAL"
	}
	return "TAINT_FLOW"
}
