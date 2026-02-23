package dsl

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// DataflowExecutor wraps existing taint analysis functions.
type DataflowExecutor struct {
	IR        *DataflowIR
	CallGraph *core.CallGraph
}

// NewDataflowExecutor creates a new executor.
func NewDataflowExecutor(ir *DataflowIR, cg *core.CallGraph) *DataflowExecutor {
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
// NOTE: This is a simplified implementation that checks for taint flows
// based on call site patterns rather than full dataflow analysis.
// Full taint analysis integration requires re-running analysis with DSL patterns.
func (e *DataflowExecutor) executeLocal() []DataflowDetection {
	detections := []DataflowDetection{}

	// Convert IR patterns to strings
	sourcePatterns := e.extractPatterns(e.IR.Sources)
	sinkPatterns := e.extractPatterns(e.IR.Sinks)
	sanitizerPatterns := e.extractPatterns(e.IR.Sanitizers)

	// Find call sites matching sources and sinks
	sourceCalls := e.findMatchingCalls(sourcePatterns)
	sinkCalls := e.findMatchingCalls(sinkPatterns)
	sanitizerCalls := e.findMatchingCalls(sanitizerPatterns)

	// For local scope, check if source and sink are in the same function
	for _, source := range sourceCalls {
		for _, sink := range sinkCalls {
			// Only detect within same function for local scope
			if source.FunctionFQN != sink.FunctionFQN {
				continue
			}

			// Check if there's a sanitizer in between (same function)
			hasSanitizer := false
			for _, sanitizer := range sanitizerCalls {
				if sanitizer.FunctionFQN == source.FunctionFQN {
					// If sanitizer is between source and sink, mark as sanitized
					if (sanitizer.Line > source.Line && sanitizer.Line < sink.Line) ||
						(sanitizer.Line > sink.Line && sanitizer.Line < source.Line) {
						hasSanitizer = true
						break
					}
				}
			}

			// Create detection
			detection := DataflowDetection{
				FunctionFQN: source.FunctionFQN,
				SourceLine:  source.Line,
				SinkLine:    sink.Line,
				TaintedVar:  "", // Not tracking variable names in this simplified version
				SinkCall:    sink.CallSite.Target,
				Confidence:  0.7, // Medium confidence for pattern-based detection
				Sanitized:   hasSanitizer,
				Scope:       "local",
			}

			detections = append(detections, detection)
		}
	}

	return detections
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
	patterns := make([]string, 0, len(matchers))
	for _, matcher := range matchers {
		patterns = append(patterns, matcher.Patterns...)
	}
	return patterns
}

// CallSiteMatch represents a matched call site.
type CallSiteMatch struct {
	CallSite    core.CallSite
	FunctionFQN string
	Line        int
}

// Helper: Find call sites matching patterns.
func (e *DataflowExecutor) findMatchingCalls(patterns []string) []CallSiteMatch {
	matches := []CallSiteMatch{}

	for functionFQN, callSites := range e.CallGraph.CallSites {
		for _, cs := range callSites {
			for _, pattern := range patterns {
				// Match against TargetFQN if available (Go), otherwise fall back to Target (Python/Java)
				targetToMatch := cs.Target
				if cs.TargetFQN != "" {
					targetToMatch = cs.TargetFQN
				}
				if e.matchesPattern(targetToMatch, pattern) {
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
		if after, ok := strings.CutPrefix(pattern, "*"); ok {
			suffix := after
			return strings.HasSuffix(target, suffix)
		}
		if before, ok := strings.CutSuffix(pattern, "*"); ok {
			prefix := before
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
