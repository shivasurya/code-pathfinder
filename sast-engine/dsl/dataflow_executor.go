package dsl

import (
	"encoding/json"
	"fmt"
	"sort"
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
// Tier 3: Line-number proximity (legacy fallback) — when no statements available.
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
				// Find a sink match for this detection's sink line.
				// Try all matchers at this line — if one rejects via tracked params,
				// try the next matcher (e.g., type_constrained may reject but
				// call_matcher at the same line may accept).
				var matchedSink *CallSiteMatch
				for i, sm := range sinkCalls {
					if sm.Line == int(det.SinkLine) {
						if len(sm.TrackedParams) > 0 && !e.matchesTrackedParams(det, sm) {
							continue // This matcher rejects, try next
						}
						matchedSink = &sinkCalls[i]
						break
					}
				}

				// No matcher accepted this detection
				if matchedSink == nil {
					// Check if any sink was at this line but all rejected by tracked params
					hasSinkAtLine := false
					for _, sm := range sinkCalls {
						if sm.Line == int(det.SinkLine) {
							hasSinkAtLine = true
							break
						}
					}
					if hasSinkAtLine {
						continue // All matchers at this line rejected
					}
					// No sink matcher at this line — keep detection without param filtering
				}

				detection := DataflowDetection{
					FunctionFQN: funcFQN,
					SourceLine:  int(det.SourceLine),
					SinkLine:    int(det.SinkLine),
					TaintedVar:  det.SourceVar,
					SinkCall:    det.SinkCall,
					Confidence:  e.confidenceForMethod(analysisMethod),
					Sanitized:   false,
					Scope:       "local",
					MatchMethod: analysisMethod,
				}
				if matchedSink != nil {
					detection.SinkParamIndex = e.resolveParamIndex(det, *matchedSink)
				}
				detections = append(detections, detection)
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
// Uses VDG transfer summaries to verify data actually flows across function
// boundaries, not just that functions are call-graph reachable.
func (e *DataflowExecutor) executeGlobal() []DataflowDetection {
	detections := []DataflowDetection{}

	// Step 1: Local analysis first.
	localDetections := e.executeLocal()
	detections = append(detections, localDetections...)

	// Step 2: Resolve all matchers.
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

	sourcePatterns := e.extractTargetPatterns(sourceCalls)
	sinkPatterns := e.extractTargetPatterns(sinkCalls)
	sanitizerPatterns := e.extractTargetPatterns(sanitizerCalls)

	// Step 3: Build transfer summaries via fixpoint.
	summaries := e.buildTransferSummaries(sourcePatterns, sinkPatterns, sanitizerPatterns)

	// Step 4: Check source→sink via BFS reachability + summary confirmation.
	sanitizerSet := make(map[string]bool, len(sanitizerCalls))
	for _, san := range sanitizerCalls {
		sanitizerSet[san.FunctionFQN] = true
	}

	reachabilityCache := make(map[string]map[string]bool)

	for _, source := range sourceCalls {
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

			// Summary-based flow confirmation: verify data actually propagates.
			if !e.summaryConfirmsFlow(source, sink, summaries) {
				continue
			}

			path := e.findPath(source.FunctionFQN, sink.FunctionFQN)
			if e.pathHasSanitizerSet(path, sanitizerSet) {
				continue
			}

			detections = append(detections, DataflowDetection{
				FunctionFQN:       sink.FunctionFQN,
				SourceFunctionFQN: source.FunctionFQN,
				SourceLine:        source.Line,
				SourceColumn:      source.CallSite.Location.Column,
				SinkLine:          sink.Line,
				SinkColumn:        sink.CallSite.Location.Column,
				SinkCall:          sink.CallSite.Target,
				Confidence:        e.confidenceForMethod("interprocedural_vdg"),
				Sanitized:         false,
				Scope:             "global",
				MatchMethod:       "interprocedural_vdg",
			})
		}
	}

	// Dedup: multiple matchers can produce identical findings for the same flow
	seen := make(map[string]bool)
	deduped := make([]DataflowDetection, 0, len(detections))
	for _, det := range detections {
		key := fmt.Sprintf("%s:%d:%d:%s", det.FunctionFQN, det.SourceLine, det.SinkLine, det.SinkCall)
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, det)
		}
	}
	return deduped
}

// summaryConfirmsFlow checks whether VDG transfer summaries confirm that
// taint from a source function actually propagates to a sink function.
// It walks the call path and verifies that at least one parameter chain
// carries taint from source through intermediaries to sink.
func (e *DataflowExecutor) summaryConfirmsFlow(
	source, sink CallSiteMatch,
	summaries map[string]*taint.TaintTransferSummary,
) bool {
	path := e.findPath(source.FunctionFQN, sink.FunctionFQN)
	if len(path) < 2 {
		return false
	}

	// Check if source function has IsSource or ReturnTaintedBySource.
	sourceSummary := summaries[source.FunctionFQN]
	if sourceSummary == nil {
		// No summary available — fall back to accepting the flow.
		return true
	}

	// Source function must produce tainted output.
	if !sourceSummary.IsSource && !sourceSummary.ReturnTaintedBySource {
		// Check if any param flows to return (could be caller-provided taint).
		hasParamToReturn := false
		for _, flows := range sourceSummary.ParamToReturn {
			if flows {
				hasParamToReturn = true
				break
			}
		}
		if !hasParamToReturn {
			return false
		}
	}

	// For direct source→sink (path length 2), check sink summary.
	sinkSummary := summaries[sink.FunctionFQN]
	if sinkSummary != nil {
		hasParamToSink := false

		// Resolve tracked params to positional indices using summary's param names
		trackedIndices := e.resolveTrackedParamIndices(
			sink.TrackedParams,
			sinkSummary.ParamNames,
		)

		if trackedIndices == nil {
			// No tracked params — current behavior: any param reaching sink counts
			for _, flows := range sinkSummary.ParamToSink {
				if flows {
					hasParamToSink = true
					break
				}
			}
		} else {
			// Only check tracked parameter indices
			for idx := range trackedIndices {
				if sinkSummary.ParamToSink[idx] {
					hasParamToSink = true
					break
				}
			}
		}

		// If sink has no ParamToSink but is known to have a sink call,
		// still accept (sink pattern matching already confirmed it).
		if len(sinkSummary.ParamToSink) > 0 && !hasParamToSink {
			return false
		}
	}

	// For multi-hop paths, verify intermediaries propagate taint.
	for i := 1; i < len(path)-1; i++ {
		midSummary := summaries[path[i]]
		if midSummary == nil {
			continue // No summary, optimistically assume propagation.
		}
		if midSummary.IsSanitizer {
			return false // Sanitizer on path kills flow.
		}
		// Check that at least one param flows to return.
		hasFlow := false
		for _, flows := range midSummary.ParamToReturn {
			if flows {
				hasFlow = true
				break
			}
		}
		if midSummary.IsSource || midSummary.ReturnTaintedBySource {
			hasFlow = true
		}
		if !hasFlow {
			return false
		}
	}

	return true
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
					CallSite:      match.CallSite,
					FunctionFQN:   match.FunctionFQN,
					Line:          match.Line,
					TrackedParams: ir.TrackedParams,
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
				} else if len(ir.TrackedParams) > 0 {
					// When TrackedParams are set, we need the actual CallSite
					// with Arguments populated for parameter matching.
					if css, ok := e.CallGraph.CallSites[det.FunctionFQN]; ok {
						for i, candidate := range css {
							if candidate.Location.Line == det.SourceLine {
								cs = css[i]
								break
							}
						}
					}
				}
				allMatches = append(allMatches, CallSiteMatch{
					CallSite:      cs,
					FunctionFQN:   det.FunctionFQN,
					Line:          det.SourceLine,
					TrackedParams: ir.TrackedParams,
				})
			}
		}
	}

	return allMatches
}

// CallSiteMatch represents a matched call site.
type CallSiteMatch struct {
	CallSite      core.CallSite
	FunctionFQN   string
	Line          int
	TrackedParams []TrackedParam // Which parameters are taint-sensitive (from matcher IR)
}

// findCallSiteAtLine returns the CallSite at the given line within a function,
// or nil if not found.
func (e *DataflowExecutor) findCallSiteAtLine(funcFQN string, line uint32) *core.CallSite {
	callSites := e.CallGraph.CallSites[funcFQN]
	for i, cs := range callSites {
		if cs.Location.Line == int(line) {
			return &callSites[i]
		}
	}
	return nil
}

// resolveTrackedParamIndices converts TrackedParams into a set of positional indices.
// Name-based params are resolved via paramNames (from TaintTransferSummary or CallGraph).
// Returns nil if no non-return TrackedParams (means "all params are sensitive").
func (e *DataflowExecutor) resolveTrackedParamIndices(
	tracked []TrackedParam,
	paramNames []string,
) map[int]bool {
	if len(tracked) == 0 {
		return nil
	}

	indices := make(map[int]bool)
	hasNonReturnParam := false
	for _, tp := range tracked {
		if tp.Return {
			continue
		}
		hasNonReturnParam = true
		if tp.Index != nil {
			indices[*tp.Index] = true
		}
		if tp.Name != "" {
			for i, name := range paramNames {
				if name == tp.Name {
					indices[i] = true
				}
			}
		}
	}
	if !hasNonReturnParam {
		return nil
	}
	return indices
}

// getParamNamesForFQN returns the ordered parameter names for a function,
// looked up from the CallGraph.Parameters map. Filters out "self" and "cls".
// Returns nil if the function's parameters are not known.
func (e *DataflowExecutor) getParamNamesForFQN(funcFQN string) []string {
	var params []*core.ParameterSymbol
	prefix := funcFQN + "."
	for key, ps := range e.CallGraph.Parameters {
		if strings.HasPrefix(key, prefix) && ps.ParentFQN == funcFQN {
			params = append(params, ps)
		}
	}
	if len(params) == 0 {
		return nil
	}
	sort.Slice(params, func(i, j int) bool {
		return params[i].Line < params[j].Line
	})
	var names []string
	for _, p := range params {
		if p.Name != "self" && p.Name != "cls" {
			names = append(names, p.Name)
		}
	}
	return names
}

// matchesTrackedParams checks if a taint detection's sink usage matches
// the tracked parameter constraints. Uses det.SinkVar (the variable at the
// sink call site, NOT det.SourceVar which is the variable at the taint source).
func (e *DataflowExecutor) matchesTrackedParams(
	det *core.TaintInfo,
	sinkMatch CallSiteMatch,
) bool {
	if len(sinkMatch.TrackedParams) == 0 {
		return true
	}

	sinkCS := e.findCallSiteAtLine(sinkMatch.FunctionFQN, det.SinkLine)
	if sinkCS == nil {
		return true // Can't resolve — accept conservatively
	}

	var paramNames []string
	if sinkCS.TargetFQN != "" {
		paramNames = e.getParamNamesForFQN(sinkCS.TargetFQN)
	}
	trackedIndices := e.resolveTrackedParamIndices(sinkMatch.TrackedParams, paramNames)
	if trackedIndices == nil {
		return true
	}

	for _, arg := range sinkCS.Arguments {
		if trackedIndices[arg.Position] && arg.IsVariable && arg.Value == det.SinkVar {
			return true
		}
	}
	return false
}

// resolveParamIndex determines the positional index of the tainted parameter
// at the sink call site. Returns nil if it cannot be determined.
func (e *DataflowExecutor) resolveParamIndex(
	det *core.TaintInfo,
	sinkMatch CallSiteMatch,
) *int {
	sinkCS := e.findCallSiteAtLine(sinkMatch.FunctionFQN, det.SinkLine)
	if sinkCS == nil {
		return nil
	}
	for _, arg := range sinkCS.Arguments {
		if arg.IsVariable && arg.Value == det.SinkVar {
			idx := arg.Position
			return &idx
		}
	}
	return nil
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

