package dsl

import (
	"encoding/json"
	"fmt"
	"log"
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

	// Find call sites matching sources and sinks (polymorphic dispatch)
	sourceCalls := e.findMatchingCallsPolymorphic(e.IR.Sources)
	sinkCalls := e.findMatchingCallsPolymorphic(e.IR.Sinks)
	sanitizerCalls := e.findMatchingCallsPolymorphic(e.IR.Sanitizers)

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

	// Then, find cross-function flows (polymorphic dispatch)
	sourceCalls := e.findMatchingCallsPolymorphic(e.IR.Sources)
	sinkCalls := e.findMatchingCallsPolymorphic(e.IR.Sinks)
	sanitizerCalls := e.findMatchingCallsPolymorphic(e.IR.Sanitizers)

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

// findMatchingCallsPolymorphic resolves a list of polymorphic matcher entries
// ([]any from JSON) to call site matches.
func (e *DataflowExecutor) findMatchingCallsPolymorphic(matchers []any) []CallSiteMatch {
	all := []CallSiteMatch{}
	for _, matcher := range matchers {
		matcherMap, ok := matcher.(map[string]any)
		if !ok {
			continue
		}
		all = append(all, e.resolveMatcherToCallSites(matcherMap)...)
	}
	return all
}

// resolveMatcherToCallSites dispatches a single matcher entry to the appropriate
// executor and returns call site matches. Handles call_matcher, type_constrained_call,
// and logic_or/and/not recursively.
func (e *DataflowExecutor) resolveMatcherToCallSites(matcherMap map[string]any) []CallSiteMatch {
	matcherType, _ := matcherMap["type"].(string)

	switch matcherType {
	case "call_matcher", "":
		// "" default for backward compat: treat untyped entries as call_matcher.
		patterns := extractPatternsFromMap(matcherMap)
		return e.findMatchingCalls(patterns)

	case "type_constrained_call":
		return e.resolveTypeConstrainedCall(matcherMap)

	case "logic_or":
		return e.resolveLogicOr(matcherMap)

	case "logic_and":
		return e.resolveLogicAnd(matcherMap)

	case "logic_not":
		return e.resolveLogicNot(matcherMap)

	case "semantic_source":
		return e.resolveSemanticSource(matcherMap)

	case "semantic_sink":
		return e.resolveSemanticSink(matcherMap)

	default:
		log.Printf("WARNING: resolveMatcherToCallSites: unknown matcher type %q, skipping", matcherType)
		return nil
	}
}

// resolveTypeConstrainedCall builds a TypeConstrainedCallExecutor from a raw map
// and returns its matches as CallSiteMatch slice.
func (e *DataflowExecutor) resolveTypeConstrainedCall(matcherMap map[string]any) []CallSiteMatch {
	jsonBytes, err := json.Marshal(matcherMap)
	if err != nil {
		return nil
	}
	var ir TypeConstrainedCallIR
	if err := json.Unmarshal(jsonBytes, &ir); err != nil {
		return nil
	}
	executor, err := NewTypeConstrainedCallExecutor(&ir, e.CallGraph)
	if err != nil {
		return nil
	}
	results := []CallSiteMatch{}
	for _, match := range executor.ExecuteWithContext() {
		results = append(results, CallSiteMatch{
			CallSite:    match.CallSite,
			FunctionFQN: match.FunctionFQN,
			Line:        match.Line,
		})
	}
	return results
}

// resolveLogicOr returns the union of call sites from all sub-matchers.
func (e *DataflowExecutor) resolveLogicOr(matcherMap map[string]any) []CallSiteMatch {
	operands, ok := matcherMap["matchers"].([]any)
	if !ok {
		return nil
	}
	seen := map[string]bool{}
	results := []CallSiteMatch{}
	for _, op := range operands {
		opMap, ok := op.(map[string]any)
		if !ok {
			continue
		}
		for _, match := range e.resolveMatcherToCallSites(opMap) {
			key := fmt.Sprintf("%s:%d", match.FunctionFQN, match.Line)
			if !seen[key] {
				seen[key] = true
				results = append(results, match)
			}
		}
	}
	return results
}

// resolveLogicAnd returns the intersection of call sites from all sub-matchers.
func (e *DataflowExecutor) resolveLogicAnd(matcherMap map[string]any) []CallSiteMatch {
	operands, ok := matcherMap["matchers"].([]any)
	if !ok || len(operands) == 0 {
		return nil
	}

	// Start with results from first operand.
	firstMap, ok := operands[0].(map[string]any)
	if !ok {
		return nil
	}
	current := e.resolveMatcherToCallSites(firstMap)
	if len(current) == 0 {
		return nil
	}

	// Intersect with each subsequent operand.
	for i := 1; i < len(operands); i++ {
		opMap, ok := operands[i].(map[string]any)
		if !ok {
			return nil
		}
		nextMatches := e.resolveMatcherToCallSites(opMap)
		nextSet := map[string]bool{}
		for _, m := range nextMatches {
			nextSet[fmt.Sprintf("%s:%d", m.FunctionFQN, m.Line)] = true
		}

		filtered := []CallSiteMatch{}
		for _, m := range current {
			key := fmt.Sprintf("%s:%d", m.FunctionFQN, m.Line)
			if nextSet[key] {
				filtered = append(filtered, m)
			}
		}
		current = filtered
	}
	return current
}

// resolveLogicNot returns all call sites in the graph EXCEPT those matching the sub-matcher.
func (e *DataflowExecutor) resolveLogicNot(matcherMap map[string]any) []CallSiteMatch {
	operand, ok := matcherMap["matcher"].(map[string]any)
	if !ok {
		return nil
	}
	excluded := e.resolveMatcherToCallSites(operand)
	excludeSet := map[string]bool{}
	for _, m := range excluded {
		excludeSet[fmt.Sprintf("%s:%d", m.FunctionFQN, m.Line)] = true
	}

	// Return all call sites NOT in the excluded set.
	results := []CallSiteMatch{}
	for functionFQN, callSites := range e.CallGraph.CallSites {
		for _, cs := range callSites {
			key := fmt.Sprintf("%s:%d", functionFQN, cs.Location.Line)
			if !excludeSet[key] {
				results = append(results, CallSiteMatch{
					CallSite:    cs,
					FunctionFQN: functionFQN,
					Line:        cs.Location.Line,
				})
			}
		}
	}
	return results
}

// extractPatternsFromMap extracts patterns from a raw call_matcher map.
func extractPatternsFromMap(matcherMap map[string]any) []string {
	patternsRaw, ok := matcherMap["patterns"].([]any)
	if !ok {
		return nil
	}
	patterns := make([]string, 0, len(patternsRaw))
	for _, p := range patternsRaw {
		if s, ok := p.(string); ok {
			patterns = append(patterns, s)
		}
	}
	return patterns
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

// resolveSemanticSource expands a semantic source category and resolves all expanded matchers.
func (e *DataflowExecutor) resolveSemanticSource(matcherMap map[string]any) []CallSiteMatch {
	category, _ := matcherMap["category"].(string)
	framework, _ := matcherMap["framework"].(string)

	expander := NewSemanticExpander()
	expanded := expander.ExpandSource(category, framework)
	if len(expanded) == 0 {
		return nil
	}

	results := []CallSiteMatch{}
	seen := map[string]bool{}
	for _, m := range expanded {
		for _, match := range e.resolveMatcherToCallSites(m) {
			key := fmt.Sprintf("%s:%d", match.FunctionFQN, match.Line)
			if !seen[key] {
				seen[key] = true
				results = append(results, match)
			}
		}
	}
	return results
}

// resolveSemanticSink expands a semantic sink category and resolves all expanded matchers.
func (e *DataflowExecutor) resolveSemanticSink(matcherMap map[string]any) []CallSiteMatch {
	category, _ := matcherMap["category"].(string)
	framework, _ := matcherMap["framework"].(string)

	expander := NewSemanticExpander()
	expanded := expander.ExpandSink(category, framework)
	if len(expanded) == 0 {
		return nil
	}

	results := []CallSiteMatch{}
	seen := map[string]bool{}
	for _, m := range expanded {
		for _, match := range e.resolveMatcherToCallSites(m) {
			key := fmt.Sprintf("%s:%d", match.FunctionFQN, match.Line)
			if !seen[key] {
				seen[key] = true
				results = append(results, match)
			}
		}
	}
	return results
}
