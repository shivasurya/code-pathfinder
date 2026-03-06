package dsl

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/analysis/taint"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/cfg"
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
// Uses VDG-based analysis when statements are available for a function,
// with fallback to line-number-based detection otherwise.
func (e *DataflowExecutor) executeLocal() []DataflowDetection {
	detections := []DataflowDetection{}

	sourcePatterns := e.extractPatterns(e.IR.Sources)
	sinkPatterns := e.extractPatterns(e.IR.Sinks)
	sanitizerPatterns := e.extractPatterns(e.IR.Sanitizers)

	sourceCalls := e.findMatchingCalls(sourcePatterns)
	sinkCalls := e.findMatchingCalls(sinkPatterns)
	sanitizerCalls := e.findMatchingCalls(sanitizerPatterns)

	candidateFunctions := e.findFunctionsWithSourcesAndSinks(sourceCalls, sinkCalls)

	if len(candidateFunctions) == 0 {
		return detections
	}

	// Extract target patterns for VDG matching
	srcTargets := e.extractTargetPatterns(sourceCalls)
	sinkTargets := e.extractTargetPatterns(sinkCalls)
	sanTargets := e.extractTargetPatterns(sanitizerCalls)

	for _, funcFQN := range candidateFunctions {
		var summary *core.TaintSummary

		// Prefer CFG-aware analysis (extracts statements from control flow bodies)
		if cfgRaw, ok := e.CallGraph.CFGs[funcFQN]; ok {
			if cfGraph, ok := cfgRaw.(*cfg.ControlFlowGraph); ok {
				if bsRaw, ok := e.CallGraph.CFGBlockStatements[funcFQN]; ok {
					if blockStmts, ok := bsRaw.(cfg.BlockStatements); ok {
						summary = taint.AnalyzeWithCFG(funcFQN, cfGraph, blockStmts, srcTargets, sinkTargets, sanTargets)
					}
				}
			}
		}

		// Fallback to flat VDG if CFG not available
		if summary == nil {
			if stmts, ok := e.CallGraph.Statements[funcFQN]; ok && len(stmts) > 0 {
				summary = taint.AnalyzeWithVDG(funcFQN, stmts, srcTargets, sinkTargets, sanTargets)
			}
		}

		if summary != nil {
			for _, det := range summary.Detections {
				detections = append(detections, DataflowDetection{
					FunctionFQN: funcFQN,
					SourceLine:  int(det.SourceLine),
					SinkLine:    int(det.SinkLine),
					TaintedVar:  det.SourceVar,
					SinkCall:    det.SinkCall,
					Confidence:  det.Confidence,
					Sanitized:   det.Sanitized,
					Scope:       "local",
				})
			}
		} else {
			// Last resort: existing line-number-based detection
			for _, source := range sourceCalls {
				for _, sink := range sinkCalls {
					if source.FunctionFQN != funcFQN || sink.FunctionFQN != funcFQN {
						continue
					}
					hasSanitizer := false
					for _, sanitizer := range sanitizerCalls {
						if sanitizer.FunctionFQN == funcFQN {
							if (sanitizer.Line > source.Line && sanitizer.Line < sink.Line) ||
								(sanitizer.Line > sink.Line && sanitizer.Line < source.Line) {
								hasSanitizer = true
								break
							}
						}
					}
					detections = append(detections, DataflowDetection{
						FunctionFQN: funcFQN,
						SourceLine:  source.Line,
						SinkLine:    sink.Line,
						TaintedVar:  "",
						SinkCall:    sink.CallSite.Target,
						Confidence:  0.7,
						Sanitized:   hasSanitizer,
						Scope:       "local",
					})
				}
			}
		}
	}

	return detections
}

// extractTargetPatterns extracts unique call target names from matched call sites.
func (e *DataflowExecutor) extractTargetPatterns(matches []CallSiteMatch) []string {
	seen := map[string]bool{}
	var patterns []string
	for _, m := range matches {
		target := m.CallSite.Target
		if target != "" && !seen[target] {
			seen[target] = true
			patterns = append(patterns, target)
		}
		if m.CallSite.TargetFQN != "" && !seen[m.CallSite.TargetFQN] {
			seen[m.CallSite.TargetFQN] = true
			patterns = append(patterns, m.CallSite.TargetFQN)
		}
	}
	return patterns
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
				// Try matching against both Target (short name) and TargetFQN (fully qualified).
				// This handles cases like pattern "eval" matching Target "eval" even when
				// TargetFQN is "builtins.eval", and pattern "os.getenv" matching TargetFQN.
				matched := e.matchesPattern(cs.Target, pattern)
				if !matched && cs.TargetFQN != "" {
					matched = e.matchesPattern(cs.TargetFQN, pattern)
				}
				if matched {
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
