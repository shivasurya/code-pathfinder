package dsl

import (
	"encoding/json"
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
func (e *DataflowExecutor) executeLocal() []DataflowDetection {
	detections := []DataflowDetection{}

	// Resolve matchers polymorphically
	sourceCalls := e.resolveMatchers(e.IR.Sources)
	sinkCalls := e.resolveMatchers(e.IR.Sinks)
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
				FunctionFQN: source.FunctionFQN,
				SourceLine:  source.Line,
				SinkLine:    sink.Line,
				SinkCall:    sink.CallSite.Target,
				Confidence:  0.7,
				Sanitized:   hasSanitizer,
				Scope:       "local",
			}

			detections = append(detections, detection)
		}
	}

	return detections
}

// executeGlobal performs inter-procedural taint analysis.
func (e *DataflowExecutor) executeGlobal() []DataflowDetection {
	detections := []DataflowDetection{}

	localDetections := e.executeLocal()
	detections = append(detections, localDetections...)

	sourceCalls := e.resolveMatchers(e.IR.Sources)
	sinkCalls := e.resolveMatchers(e.IR.Sinks)
	sanitizerCalls := e.resolveMatchers(e.IR.Sanitizers)

	for _, source := range sourceCalls {
		for _, sink := range sinkCalls {
			if source.FunctionFQN == sink.FunctionFQN {
				continue
			}

			path := e.findPath(source.FunctionFQN, sink.FunctionFQN)
			if len(path) > 1 {
				hasSanitizer := e.pathHasSanitizer(path, sanitizerCalls)

				if !hasSanitizer {
					detections = append(detections, DataflowDetection{
						FunctionFQN: source.FunctionFQN,
						SourceLine:  source.Line,
						SinkLine:    sink.Line,
						SinkCall:    sink.CallSite.Target,
						Confidence:  0.8,
						Sanitized:   false,
						Scope:       "global",
					})
				}
			}
		}
	}

	return detections
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

