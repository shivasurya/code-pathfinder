package dsl

import (
	"log"
	"strings"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
)

// DataflowExecutor wraps existing taint analysis functions.
type DataflowExecutor struct {
	IR        *DataflowIR
	CallGraph *callgraph.CallGraph
}

// NewDataflowExecutor creates a new executor.
func NewDataflowExecutor(ir *DataflowIR, cg *callgraph.CallGraph) *DataflowExecutor {
	return &DataflowExecutor{
		IR:        ir,
		CallGraph: cg,
	}
}

// Execute routes to local or global analysis based on scope.
func (e *DataflowExecutor) Execute() []DataflowDetection {
	if e.IR.Scope == "local" {
		return e.executeLocal()
	}
	return e.executeGlobal()
}

// executeLocal performs intra-procedural taint analysis.
// REUSES existing AnalyzeIntraProceduralTaint() from callgraph/taint.go.
func (e *DataflowExecutor) executeLocal() []DataflowDetection {
	detections := []DataflowDetection{}

	// Convert IR patterns to strings for existing API
	sourcePatterns := e.extractPatterns(e.IR.Sources)
	sinkPatterns := e.extractPatterns(e.IR.Sinks)
	sanitizerPatterns := e.extractPatterns(e.IR.Sanitizers)

	// Find all source and sink call sites
	sourceCalls := e.findMatchingCalls(sourcePatterns)
	sinkCalls := e.findMatchingCalls(sinkPatterns)

	// For each function that has both sources and sinks
	functionsToAnalyze := e.findFunctionsWithSourcesAndSinks(sourceCalls, sinkCalls)

	for _, functionFQN := range functionsToAnalyze {
		// Call EXISTING intra-procedural analysis
		detection := e.analyzeFunction(functionFQN, sourcePatterns, sinkPatterns, sanitizerPatterns)
		if detection != nil {
			detections = append(detections, *detection)
		}
	}

	return detections
}

// analyzeFunction calls the EXISTING checkIntraProceduralTaint logic.
//
//nolint:unparam // Parameters will be used in future PRs
func (e *DataflowExecutor) analyzeFunction(
	functionFQN string,
	sourcePatterns []string,
	sinkPatterns []string,
	sanitizerPatterns []string,
) *DataflowDetection {
	// Get function node
	funcNode, ok := e.CallGraph.Functions[functionFQN]
	if !ok {
		return nil
	}

	// TODO: Full integration requires AST parsing infrastructure
	// For now, this is a placeholder that demonstrates the integration pattern
	// The actual implementation would:
	// 1. Parse the source file to get AST
	// 2. Find the function node in the AST
	// 3. Call ExtractStatements(filePath, sourceCode, functionNode)
	// 4. Build def-use chains
	// 5. Call AnalyzeIntraProceduralTaint
	// 6. Convert results to DataflowDetection

	log.Printf("Would analyze function %s in file %s", functionFQN, funcNode.File)

	// Placeholder: return nil for now
	// Real implementation will be completed in future PRs
	return nil
}

// executeGlobal performs inter-procedural taint analysis.
// REUSES existing findPath() from callgraph/patterns.go.
func (e *DataflowExecutor) executeGlobal() []DataflowDetection {
	detections := []DataflowDetection{}

	// First, run local analysis (all intra-procedural detections)
	localDetections := e.executeLocal()
	detections = append(detections, localDetections...)

	// Then, find cross-function flows
	sourcePatterns := e.extractPatterns(e.IR.Sources)
	sinkPatterns := e.extractPatterns(e.IR.Sinks)
	sanitizerPatterns := e.extractPatterns(e.IR.Sanitizers)

	sourceCalls := e.findMatchingCalls(sourcePatterns)
	sinkCalls := e.findMatchingCalls(sinkPatterns)
	sanitizerCalls := e.findMatchingCalls(sanitizerPatterns)

	// Check cross-function paths
	for _, source := range sourceCalls {
		for _, sink := range sinkCalls {
			// Skip if same function (already handled by local analysis)
			if source.FunctionFQN == sink.FunctionFQN {
				continue
			}

			// Call EXISTING findPath() logic
			path := e.findPath(source.FunctionFQN, sink.FunctionFQN)
			if len(path) > 1 {
				// Check if sanitizer is on path
				hasSanitizer := e.pathHasSanitizer(path, sanitizerCalls)

				if !hasSanitizer {
					detections = append(detections, DataflowDetection{
						FunctionFQN: source.FunctionFQN,
						SourceLine:  source.Line,
						SinkLine:    sink.Line,
						TaintedVar:  "", // Cross-function, no single var
						SinkCall:    sink.CallSite.Target,
						Confidence:  0.8, // Lower confidence for cross-function
						Sanitized:   false,
						Scope:       "global",
					})
				}
			}
		}
	}

	return detections
}

// Helper: findPath - REUSES existing DFS logic from patterns.go.
func (e *DataflowExecutor) findPath(from, to string) []string {
	if from == to {
		return []string{from}
	}

	visited := make(map[string]bool)
	path := []string{}

	if e.dfs(from, to, visited, &path) {
		return path
	}

	return []string{}
}

func (e *DataflowExecutor) dfs(current, target string, visited map[string]bool, path *[]string) bool {
	*path = append(*path, current)

	if current == target {
		return true
	}

	visited[current] = true

	for _, callee := range e.CallGraph.Edges[current] {
		if !visited[callee] {
			if e.dfs(callee, target, visited, path) {
				return true
			}
		}
	}

	*path = (*path)[:len(*path)-1]
	return false
}

// Helper: Check if sanitizer is on path.
func (e *DataflowExecutor) pathHasSanitizer(path []string, sanitizers []CallSiteMatch) bool {
	for _, pathFunc := range path {
		for _, san := range sanitizers {
			if pathFunc == san.FunctionFQN {
				return true
			}
		}
	}
	return false
}

// Helper: Extract patterns from CallMatcherIR list.
func (e *DataflowExecutor) extractPatterns(matchers []CallMatcherIR) []string {
	patterns := []string{}
	for _, matcher := range matchers {
		patterns = append(patterns, matcher.Patterns...)
	}
	return patterns
}

// CallSiteMatch represents a matched call site.
type CallSiteMatch struct {
	CallSite    callgraph.CallSite
	FunctionFQN string
	Line        int
}

// Helper: Find call sites matching patterns.
func (e *DataflowExecutor) findMatchingCalls(patterns []string) []CallSiteMatch {
	matches := []CallSiteMatch{}

	for functionFQN, callSites := range e.CallGraph.CallSites {
		for _, cs := range callSites {
			for _, pattern := range patterns {
				if e.matchesPattern(cs.Target, pattern) {
					matches = append(matches, CallSiteMatch{
						CallSite:    cs,
						FunctionFQN: functionFQN,
						Line:        cs.Location.Line,
					})
					break
				}
			}
		}
	}

	return matches
}

// Helper: Wildcard pattern matching.
func (e *DataflowExecutor) matchesPattern(target, pattern string) bool {
	if pattern == "*" {
		return true
	}

	if strings.Contains(pattern, "*") {
		if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
			substr := strings.Trim(pattern, "*")
			return strings.Contains(target, substr)
		}
		if strings.HasPrefix(pattern, "*") {
			suffix := strings.TrimPrefix(pattern, "*")
			return strings.HasSuffix(target, suffix)
		}
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			return strings.HasPrefix(target, prefix)
		}
	}

	return target == pattern
}

// Helper: Find functions that have both sources and sinks (candidates for local analysis).
func (e *DataflowExecutor) findFunctionsWithSourcesAndSinks(sources, sinks []CallSiteMatch) []string {
	sourceMap := make(map[string]bool)
	for _, s := range sources {
		sourceMap[s.FunctionFQN] = true
	}

	sinkMap := make(map[string]bool)
	for _, s := range sinks {
		sinkMap[s.FunctionFQN] = true
	}

	functions := []string{}
	for funcFQN := range sourceMap {
		if sinkMap[funcFQN] {
			functions = append(functions, funcFQN)
		}
	}

	return functions
}
