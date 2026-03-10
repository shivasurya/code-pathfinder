package dsl

import (
	"encoding/json"

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
		IR:        ir,
		CallGraph: cg,
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

// executeLocal performs intra-procedural taint analysis.
func (e *DataflowExecutor) executeLocal() []DataflowDetection {
	detections := []DataflowDetection{}

	// Resolve matchers polymorphically.
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

	// For local scope, check if source and sink are in the same function.
	for _, source := range sourceCalls {
		for _, sink := range sinkCalls {
			if source.FunctionFQN != sink.FunctionFQN {
				continue
			}

			hasSanitizer := false
			for _, sanitizer := range sanitizerCalls {
				if sanitizer.FunctionFQN == source.FunctionFQN {
					if (sanitizer.Line > source.Line && sanitizer.Line < sink.Line) ||
						(sanitizer.Line > sink.Line && sanitizer.Line < source.Line) {
						hasSanitizer = true
						break
					}
				}
			}

			detection := DataflowDetection{
				FunctionFQN:  source.FunctionFQN,
				SourceLine:   source.Line,
				SourceColumn: source.CallSite.Location.Column,
				SinkLine:     sink.Line,
				SinkColumn:   sink.CallSite.Location.Column,
				SinkCall:     sink.CallSite.Target,
				Confidence:   e.Config.getLocalScopeConfidence(),
				Sanitized:    hasSanitizer,
				Scope:        "local",
			}

			detections = append(detections, detection)
		}
	}

	return detections
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



