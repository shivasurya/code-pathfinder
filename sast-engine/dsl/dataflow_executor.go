package dsl

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
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
// Uses summary-based approach: builds TaintTransferSummary per function,
// then uses them to propagate taint across call boundaries in the VDG.
func (e *DataflowExecutor) executeGlobal() []DataflowDetection {
	detections := []DataflowDetection{}

	sourcePatterns := e.extractPatterns(e.IR.Sources)
	sinkPatterns := e.extractPatterns(e.IR.Sinks)
	sanitizerPatterns := e.extractPatterns(e.IR.Sanitizers)

	sourceCalls := e.findMatchingCalls(sourcePatterns)
	sinkCalls := e.findMatchingCalls(sinkPatterns)
	sanitizerCalls := e.findMatchingCalls(sanitizerPatterns)

	srcTargets := e.extractTargetPatterns(sourceCalls)
	sinkTargets := e.extractTargetPatterns(sinkCalls)
	sanTargets := e.extractTargetPatterns(sanitizerCalls)

	// Phase 1: Build taint transfer summaries for all functions
	transferSummaries := e.buildTransferSummaries(srcTargets, sinkTargets, sanTargets)

	// Phase 2: Run inter-procedural analysis for each function that has sinks
	// (not just functions with both sources AND sinks — sources may come from callees)
	sinkFunctions := make(map[string]bool)
	for _, sink := range sinkCalls {
		sinkFunctions[sink.FunctionFQN] = true
	}

	// Also analyze functions that call source functions (taint enters via return value)
	sourceFunctions := make(map[string]bool)
	for _, src := range sourceCalls {
		sourceFunctions[src.FunctionFQN] = true
	}

	// Find all functions that might have inter-procedural flows:
	// any function that has a sink OR calls a function that is a source
	candidateFunctions := make(map[string]bool)
	for funcFQN := range sinkFunctions {
		candidateFunctions[funcFQN] = true
	}
	// Also add callers of source-containing functions
	for funcFQN := range sourceFunctions {
		if callers, ok := e.CallGraph.ReverseEdges[funcFQN]; ok {
			for _, caller := range callers {
				candidateFunctions[caller] = true
			}
		}
	}
	// And add callers of sink-containing functions (taint may enter via arguments)
	for funcFQN := range sinkFunctions {
		if callers, ok := e.CallGraph.ReverseEdges[funcFQN]; ok {
			for _, caller := range callers {
				candidateFunctions[caller] = true
			}
		}
	}

	for funcFQN := range candidateFunctions {
		stmts := e.getStatementsForFunction(funcFQN)
		if len(stmts) == 0 {
			continue
		}

		summary := taint.AnalyzeInterProcedural(
			funcFQN, stmts,
			srcTargets, sinkTargets, sanTargets,
			e.CallGraph, transferSummaries,
		)

		if summary != nil {
			for _, det := range summary.Detections {
				detections = append(detections, DataflowDetection{
					FunctionFQN: funcFQN,
					SourceLine:  int(det.SourceLine),
					SinkLine:    int(det.SinkLine),
					TaintedVar:  det.SourceVar,
					SinkCall:    det.SinkCall,
					Confidence:  det.Confidence * 0.9, // slight decay for inter-procedural
					Sanitized:   det.Sanitized,
					Scope:       "global",
				})
			}
		}
	}

	// Also run local analysis for functions with both source and sink
	localDetections := e.executeLocal()
	detections = append(detections, localDetections...)

	return detections
}

// buildTransferSummaries builds TaintTransferSummary for all functions in the call graph.
func (e *DataflowExecutor) buildTransferSummaries(
	sources, sinks, sanitizers []string,
) map[string]*taint.TaintTransferSummary {
	summaries := make(map[string]*taint.TaintTransferSummary)

	for funcFQN, funcNode := range e.CallGraph.Functions {
		stmts := e.getStatementsForFunction(funcFQN)
		if len(stmts) == 0 {
			continue
		}

		// Get parameter names from the function node
		paramNames := e.getParamNames(funcFQN, funcNode)

		// Build transfer summary (prefer CFG-aware)
		var ts *taint.TaintTransferSummary
		if cfgRaw, ok := e.CallGraph.CFGs[funcFQN]; ok {
			if cfGraph, ok := cfgRaw.(*cfg.ControlFlowGraph); ok {
				if bsRaw, ok := e.CallGraph.CFGBlockStatements[funcFQN]; ok {
					if blockStmts, ok := bsRaw.(cfg.BlockStatements); ok {
						ts = taint.BuildTaintTransferSummaryWithCFG(
							funcFQN, cfGraph, blockStmts,
							paramNames, sources, sinks, sanitizers,
							nil, nil,
						)
					}
				}
			}
		}
		if ts == nil {
			ts = taint.BuildTaintTransferSummary(
				funcFQN, stmts, paramNames, sources, sinks, sanitizers,
				nil, nil,
			)
		}

		summaries[funcFQN] = ts
	}

	return summaries
}

// getStatementsForFunction retrieves flattened statements for a function,
// preferring CFG-flattened statements over raw statements.
func (e *DataflowExecutor) getStatementsForFunction(funcFQN string) []*core.Statement {
	// Try CFG-flattened statements first
	if cfgRaw, ok := e.CallGraph.CFGs[funcFQN]; ok {
		if cfGraph, ok := cfgRaw.(*cfg.ControlFlowGraph); ok {
			if bsRaw, ok := e.CallGraph.CFGBlockStatements[funcFQN]; ok {
				if blockStmts, ok := bsRaw.(cfg.BlockStatements); ok {
					return taint.FlattenBlockStatements(cfGraph, blockStmts)
				}
			}
		}
	}
	// Fallback to flat statements
	if stmts, ok := e.CallGraph.Statements[funcFQN]; ok {
		return stmts
	}
	return nil
}

// getParamNames extracts parameter names from a function's graph.Node.
func (e *DataflowExecutor) getParamNames(funcFQN string, funcNode *graph.Node) []string {
	if funcNode == nil {
		return nil
	}
	// MethodArgumentsValue contains parameter names like ["self", "data", "key"]
	// Filter out "self" and "cls" for methods
	var params []string
	for _, p := range funcNode.MethodArgumentsValue {
		if p != "self" && p != "cls" {
			params = append(params, p)
		}
	}
	return params
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
