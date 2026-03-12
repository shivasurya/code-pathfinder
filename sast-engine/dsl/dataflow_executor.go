package dsl

import (
	"encoding/json"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/analysis/taint"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/cfg"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// DataflowExecutor wraps existing taint analysis functions.
type DataflowExecutor struct {
	IR          *DataflowIR
	CallGraph   *core.CallGraph
	Config      *QueryTypeConfig
	Diagnostics *DiagnosticCollector
}

// NewDataflowExecutor creates a new executor.
func NewDataflowExecutor(ir *DataflowIR, cg *core.CallGraph) *DataflowExecutor {
	return &DataflowExecutor{
		IR:          ir,
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}
}

// Execute routes to local or global analysis based on scope.
func (e *DataflowExecutor) Execute() []DataflowDetection {
	if e.CallGraph == nil {
		e.Diagnostics.Addf("error", "dataflow", "DataflowExecutor: CallGraph is nil")
		return nil
	}

	if e.IR.Scope == "local" {
		return e.executeLocal()
	}
	return e.executeGlobal()
}

// executeLocal performs intra-procedural taint analysis with 3-tier fallback:
// Tier 1: CFG-aware VDG (highest confidence) — uses control flow graph + variable dependency graph
// Tier 2: Flat VDG — uses variable dependency graph without CFG
// Tier 3: Line-number proximity (legacy fallback) — when no statements available
func (e *DataflowExecutor) executeLocal() []DataflowDetection {
	detections := []DataflowDetection{}

	sourceCalls := e.resolveMatchers(e.IR.Sources)
	if len(sourceCalls) == 0 {
		e.Diagnostics.Addf("debug", "dataflow", "0 sources found, skipping local analysis")
		return detections
	}

	sinkCalls := e.resolveMatchers(e.IR.Sinks)
	if len(sinkCalls) == 0 {
		e.Diagnostics.Addf("debug", "dataflow", "0 sinks found, skipping local analysis")
		return detections
	}

	sanitizerCalls := e.resolveMatchers(e.IR.Sanitizers)

	sourcePatterns := e.extractTargetPatterns(sourceCalls)
	sinkPatterns := e.extractTargetPatterns(sinkCalls)
	sanitizerPatterns := e.extractTargetPatterns(sanitizerCalls)

	candidateFuncs := e.findFunctionsWithSourcesAndSinks(sourceCalls, sinkCalls)

	for _, funcFQN := range candidateFuncs {
		stmts := e.getStatementsForFunction(funcFQN)
		if len(stmts) == 0 {
			// Tier 3: Legacy line-number proximity (no statements available)
			e.executeLocalLegacy(funcFQN, sourceCalls, sinkCalls, sanitizerCalls, &detections)
			continue
		}

		// Tier 1: CFG-aware VDG
		analysisMethod := "flat_vdg"
		var summary *core.TaintSummary

		if raw, exists := e.CallGraph.CFGs[funcFQN]; exists {
			if cfGraph, ok := raw.(*cfg.ControlFlowGraph); ok {
				if rawBS, bsExists := e.CallGraph.CFGBlockStatements[funcFQN]; bsExists {
					if blockStmts, bsOK := rawBS.(cfg.BlockStatements); bsOK && len(blockStmts) > 0 {
						summary = taint.AnalyzeWithCFG(funcFQN, cfGraph, blockStmts,
							sourcePatterns, sinkPatterns, sanitizerPatterns)
						analysisMethod = "cfg_vdg"
					}
				}
			}
		}

		// Tier 2: Flat VDG (if Tier 1 found no detections)
		if summary == nil || !summary.HasDetections() {
			summary = taint.AnalyzeWithVDG(funcFQN, stmts,
				sourcePatterns, sinkPatterns, sanitizerPatterns)
			analysisMethod = "flat_vdg"
		}

		if summary != nil {
			for _, det := range summary.Detections {
				detections = append(detections, DataflowDetection{
					FunctionFQN: funcFQN,
					SourceLine:  int(det.SourceLine),
					SinkLine:    int(det.SinkLine),
					TaintedVar:  det.SourceVar,
					SinkCall:    det.SinkCall,
					Confidence:  e.confidenceForMethod(analysisMethod),
					Sanitized:   false,
					Scope:       "local",
					MatchMethod: analysisMethod,
				})
			}
		}
	}

	return detections
}

// confidenceForMethod returns the confidence score for a given analysis method.
func (e *DataflowExecutor) confidenceForMethod(method string) float64 {
	switch method {
	case "cfg_vdg":
		return 0.95
	case "flat_vdg":
		return 0.85
	case "interprocedural_vdg":
		return 0.80
	case "line_proximity":
		return 0.50
	default:
		return 0.60
	}
}

// executeLocalLegacy performs line-number proximity analysis (Tier 3 fallback).
// Used when no statements are available for a function.
func (e *DataflowExecutor) executeLocalLegacy(
	funcFQN string,
	sourceCalls, sinkCalls, sanitizerCalls []CallSiteMatch,
	detections *[]DataflowDetection,
) {
	for _, source := range sourceCalls {
		if source.FunctionFQN != funcFQN {
			continue
		}
		for _, sink := range sinkCalls {
			if sink.FunctionFQN != funcFQN {
				continue
			}

			hasSanitizer := false
			for _, san := range sanitizerCalls {
				if san.FunctionFQN == funcFQN {
					if (san.Line > source.Line && san.Line < sink.Line) ||
						(san.Line > sink.Line && san.Line < source.Line) {
						hasSanitizer = true
						break
					}
				}
			}
			if hasSanitizer {
				continue
			}

			*detections = append(*detections, DataflowDetection{
				FunctionFQN:  funcFQN,
				SourceLine:   source.Line,
				SourceColumn: source.CallSite.Location.Column,
				SinkLine:     sink.Line,
				SinkColumn:   sink.CallSite.Location.Column,
				SinkCall:     sink.CallSite.Target,
				Confidence:   0.50,
				Sanitized:    false,
				Scope:        "local",
				MatchMethod:  "line_proximity",
			})
		}
	}
}

// executeGlobal performs inter-procedural taint analysis.
// Uses BFS memoization: precomputes reachability from each source once,
// then checks sink reachability in O(1) per pair.
func (e *DataflowExecutor) executeGlobal() []DataflowDetection {
	detections := []DataflowDetection{}

	localDetections := e.executeLocal()
	detections = append(detections, localDetections...)

	sourceCalls := e.resolveMatchers(e.IR.Sources)
	if len(sourceCalls) == 0 {
		e.Diagnostics.Addf("debug", "dataflow", "0 sources found, skipping global analysis")
		return detections
	}

	sinkCalls := e.resolveMatchers(e.IR.Sinks)
	if len(sinkCalls) == 0 {
		e.Diagnostics.Addf("debug", "dataflow", "0 sinks found, skipping global analysis")
		return detections
	}

	sanitizerCalls := e.resolveMatchers(e.IR.Sanitizers)

	// Build sanitizer FQN set for O(1) lookup.
	sanitizerSet := make(map[string]bool, len(sanitizerCalls))
	for _, san := range sanitizerCalls {
		sanitizerSet[san.FunctionFQN] = true
	}

	// Memoize BFS reachability per unique source FQN.
	reachabilityCache := make(map[string]map[string]bool)

	for _, source := range sourceCalls {
		// Compute BFS reachability once per unique source function.
		reachable, cached := reachabilityCache[source.FunctionFQN]
		if !cached {
			reachable = e.bfsReachable(source.FunctionFQN)
			reachabilityCache[source.FunctionFQN] = reachable
		}

		for _, sink := range sinkCalls {
			if source.FunctionFQN == sink.FunctionFQN {
				continue
			}

			if !reachable[sink.FunctionFQN] {
				continue
			}

			// Use findPath for sanitizer path check (need actual path nodes).
			path := e.findPath(source.FunctionFQN, sink.FunctionFQN)
			hasSanitizer := e.pathHasSanitizerSet(path, sanitizerSet)

			if !hasSanitizer {
				detections = append(detections, DataflowDetection{
					FunctionFQN:  source.FunctionFQN,
					SourceLine:   source.Line,
					SourceColumn: source.CallSite.Location.Column,
					SinkLine:     sink.Line,
					SinkColumn:   sink.CallSite.Location.Column,
					SinkCall:     sink.CallSite.Target,
					Confidence:   e.Config.getGlobalScopeConfidence(),
					Sanitized:    false,
					Scope:        "global",
				})
			}
		}
	}

	return detections
}

// bfsReachable computes the set of all functions reachable from startFQN
// via the call graph edges using breadth-first search.
func (e *DataflowExecutor) bfsReachable(startFQN string) map[string]bool {
	visited := map[string]bool{startFQN: true}
	queue := []string{startFQN}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, callee := range e.CallGraph.Edges[current] {
			if !visited[callee] {
				visited[callee] = true
				queue = append(queue, callee)
			}
		}
	}
	return visited
}

// pathHasSanitizerSet checks if any function on the path is in the sanitizer set.
func (e *DataflowExecutor) pathHasSanitizerSet(path []string, sanitizerSet map[string]bool) bool {
	for _, pathFunc := range path {
		if sanitizerSet[pathFunc] {
			return true
		}
	}
	return false
}

// resolveMatchers dispatches raw JSON matchers to the appropriate executor.
// Supports both CallMatcherIR and TypeConstrainedCallIR.
func (e *DataflowExecutor) resolveMatchers(rawMatchers []json.RawMessage) []CallSiteMatch {
	var allMatches []CallSiteMatch

	for _, raw := range rawMatchers {
		// Peek at "type" field to determine matcher type.
		var peek struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &peek); err != nil {
			continue
		}

		switch peek.Type {
		case "call_matcher":
			var ir CallMatcherIR
			if err := json.Unmarshal(raw, &ir); err != nil {
				continue
			}
			executor := NewCallMatcherExecutor(&ir, e.CallGraph)
			for _, match := range executor.ExecuteWithContext() {
				allMatches = append(allMatches, CallSiteMatch{
					CallSite:    match.CallSite,
					FunctionFQN: match.FunctionFQN,
					Line:        match.Line,
				})
			}

		case "type_constrained_call":
			var ir TypeConstrainedCallIR
			if err := json.Unmarshal(raw, &ir); err != nil {
				continue
			}
			executor := &TypeConstrainedCallExecutor{
				IR:               &ir,
				CallGraph:        e.CallGraph,
				Config:           e.Config,
				ThirdPartyRemote: extractInheritanceChecker(e.CallGraph),
			}
			for _, det := range executor.Execute() {
				cs := core.CallSite{
					Target:   det.SinkCall,
					Location: core.Location{Line: det.SourceLine},
				}
				if det.MatchedCallSite != nil {
					cs = *det.MatchedCallSite
				}
				allMatches = append(allMatches, CallSiteMatch{
					CallSite:    cs,
					FunctionFQN: det.FunctionFQN,
					Line:        det.SourceLine,
				})
			}
		}
	}

	return allMatches
}

// CallSiteMatch represents a matched call site.
type CallSiteMatch struct {
	CallSite    core.CallSite
	FunctionFQN string
	Line        int
}

// findPath uses DFS to find a path between two functions.
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

// pathHasSanitizer checks if any sanitizer function is on the path.
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

// ============================================================================
// VDG-specific functions (from demand-driven-dataflow branch)
// These are used by VDG tests and will be wired into executeLocal/executeGlobal
// in PR-04 and PR-05. Kept here to maintain zero test failures across all PRs.
// ============================================================================

// extractTargetPatterns extracts unique call target names from matched call sites.
// Also extracts the bare name (last segment) for dotted targets, since statement
// CallTarget may use the bare name (e.g., "execute" vs callsite "cursor.execute").
func (e *DataflowExecutor) extractTargetPatterns(matches []CallSiteMatch) []string { //nolint:unused // wired in PR-04
	seen := map[string]bool{}
	var patterns []string
	addPattern := func(p string) {
		if p != "" && !seen[p] {
			seen[p] = true
			patterns = append(patterns, p)
		}
	}
	for _, m := range matches {
		addPattern(m.CallSite.Target)
		addPattern(m.CallSite.TargetFQN)
		// Also add the bare name for dotted targets (e.g., "cursor.execute" → "execute")
		if strings.Contains(m.CallSite.Target, ".") {
			parts := strings.Split(m.CallSite.Target, ".")
			addPattern(parts[len(parts)-1])
		}
	}
	return patterns
}

// buildTransferSummaries builds TaintTransferSummary for all functions using
// iterative fixpoint: each round uses the previous round's summaries to enhance
// callee lookups. Converges when no summary changes or maxIterations reached.
func (e *DataflowExecutor) buildTransferSummaries(
	sources, sinks, sanitizers []string,
) map[string]*taint.TaintTransferSummary {
	const maxIterations = 10
	summaries := make(map[string]*taint.TaintTransferSummary)

	for iteration := 0; iteration < maxIterations; iteration++ {
		changed := false
		newSummaries := make(map[string]*taint.TaintTransferSummary)

		for funcFQN, funcNode := range e.CallGraph.Functions {
			stmts := e.getStatementsForFunction(funcFQN)
			if len(stmts) == 0 {
				continue
			}

			paramNames := e.getParamNames(funcFQN, funcNode)

			var ts *taint.TaintTransferSummary
			if cfgRaw, ok := e.CallGraph.CFGs[funcFQN]; ok {
				if cfGraph, ok := cfgRaw.(*cfg.ControlFlowGraph); ok {
					if bsRaw, ok := e.CallGraph.CFGBlockStatements[funcFQN]; ok {
						if blockStmts, ok := bsRaw.(cfg.BlockStatements); ok {
							ts = taint.BuildTaintTransferSummaryWithCFG(
								funcFQN, cfGraph, blockStmts,
								paramNames, sources, sinks, sanitizers,
								e.CallGraph, summaries,
							)
						}
					}
				}
			}
			if ts == nil {
				ts = taint.BuildTaintTransferSummary(
					funcFQN, stmts, paramNames, sources, sinks, sanitizers,
					e.CallGraph, summaries,
				)
			}

			newSummaries[funcFQN] = ts

			if !summaryEqual(summaries[funcFQN], ts) {
				changed = true
			}
		}

		summaries = newSummaries

		if !changed {
			break
		}
	}

	return summaries
}

// summaryEqual checks if two TaintTransferSummary values are equal.
func summaryEqual(a, b *taint.TaintTransferSummary) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.IsSource != b.IsSource ||
		a.IsSanitizer != b.IsSanitizer ||
		a.ReturnTaintedBySource != b.ReturnTaintedBySource {
		return false
	}
	if len(a.ParamToReturn) != len(b.ParamToReturn) {
		return false
	}
	for k, v := range a.ParamToReturn {
		if b.ParamToReturn[k] != v {
			return false
		}
	}
	if len(a.ParamToSink) != len(b.ParamToSink) {
		return false
	}
	for k, v := range a.ParamToSink {
		if b.ParamToSink[k] != v {
			return false
		}
	}
	return true
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
func (e *DataflowExecutor) getParamNames(_ string, funcNode *graph.Node) []string {
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

// addTransitiveCallers adds all transitive callers of funcFQN to the candidates set.
func (e *DataflowExecutor) addTransitiveCallers(funcFQN string, candidates map[string]bool) { //nolint:unused // wired in PR-05
	visited := make(map[string]bool)
	queue := []string{funcFQN}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		if callers, ok := e.CallGraph.ReverseEdges[current]; ok {
			for _, caller := range callers {
				candidates[caller] = true
				if !visited[caller] {
					queue = append(queue, caller)
				}
			}
		}
	}
}

// extractPatterns extracts string patterns from CallMatcherIR list.
// Used by VDG tests that construct matchers directly.
func (e *DataflowExecutor) extractPatterns(matchers []CallMatcherIR) []string {
	patterns := make([]string, 0, len(matchers))
	for _, matcher := range matchers {
		patterns = append(patterns, matcher.Patterns...)
	}
	return patterns
}

// findMatchingCalls finds call sites matching the given string patterns.
func (e *DataflowExecutor) findMatchingCalls(patterns []string) []CallSiteMatch {
	matches := []CallSiteMatch{}

	for functionFQN, callSites := range e.CallGraph.CallSites {
		for _, cs := range callSites {
			for _, pattern := range patterns {
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

// matchesPattern performs wildcard pattern matching on call targets.
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

// findFunctionsWithSourcesAndSinks finds functions that have both sources and sinks.
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

