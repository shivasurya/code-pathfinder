package dsl

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
)

// CallMatcherExecutor executes call_matcher IR against callgraph.
type CallMatcherExecutor struct {
	IR        *CallMatcherIR
	CallGraph *core.CallGraph
}

// NewCallMatcherExecutor creates a new executor.
func NewCallMatcherExecutor(ir *CallMatcherIR, cg *core.CallGraph) *CallMatcherExecutor {
	return &CallMatcherExecutor{
		IR:        ir,
		CallGraph: cg,
	}
}

// Execute finds all call sites matching the patterns
//
// Algorithm:
//  1. Iterate over callGraph.CallSites (keyed by function FQN)
//  2. For each function's call sites, check if Target matches any pattern
//  3. Support wildcards (* matching)
//  4. Return list of matching CallSite objects
//
// Performance: O(F * C * P) where:
//   F = number of functions
//   C = avg call sites per function (~5-10)
//   P = number of patterns (~2-5)
// Typical: 1000 functions * 7 calls * 3 patterns = 21,000 comparisons (fast!)
func (e *CallMatcherExecutor) Execute() []core.CallSite {
	matches := []core.CallSite{}

	// Iterate over all functions' call sites
	for _, callSites := range e.CallGraph.CallSites {
		for _, callSite := range callSites {
			if e.matchesCallSite(&callSite) {
				matches = append(matches, callSite)
			}
		}
	}

	return matches
}

// matchesCallSite checks if a call site matches any pattern.
func (e *CallMatcherExecutor) matchesCallSite(cs *core.CallSite) bool {
	target := cs.Target

	for _, pattern := range e.IR.Patterns {
		if e.matchesPattern(target, pattern) {
			return true // match_mode="any" (default)
		}
	}

	return false
}

// matchesPattern checks if target matches pattern (with wildcard support)
//
// Examples:
//   matchesPattern("eval", "eval") → true
//   matchesPattern("request.GET", "request.*") → true
//   matchesPattern("utils.sanitize", "*.sanitize") → true
//   matchesPattern("os.system", "eval") → false
func (e *CallMatcherExecutor) matchesPattern(target, pattern string) bool {
	if !e.IR.Wildcard {
		// Exact match (fast path)
		return target == pattern
	}

	// Wildcard matching
	if pattern == "*" {
		return true // Match everything
	}

	if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
		// *substring* → contains
		substr := strings.Trim(pattern, "*")
		return strings.Contains(target, substr)
	}

	if strings.HasPrefix(pattern, "*") {
		// *suffix → ends with
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(target, suffix)
	}

	if strings.HasSuffix(pattern, "*") {
		// prefix* → starts with
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(target, prefix)
	}

	// No wildcards, exact match
	return target == pattern
}

// CallMatchResult represents a match with additional context.
type CallMatchResult struct {
	CallSite    core.CallSite
	MatchedBy   string // Which pattern matched
	FunctionFQN string // Which function contains this call
	SourceFile  string // Which file
	Line        int    // Line number
}

// ExecuteWithContext returns matches with additional context.
func (e *CallMatcherExecutor) ExecuteWithContext() []CallMatchResult {
	results := []CallMatchResult{}

	for functionFQN, callSites := range e.CallGraph.CallSites {
		for _, callSite := range callSites {
			if matchedPattern := e.getMatchedPattern(&callSite); matchedPattern != "" {
				results = append(results, CallMatchResult{
					CallSite:    callSite,
					MatchedBy:   matchedPattern,
					FunctionFQN: functionFQN,
					SourceFile:  callSite.Location.File,
					Line:        callSite.Location.Line,
				})
			}
		}
	}

	return results
}

// getMatchedPattern returns which pattern matched (or empty string if no match).
func (e *CallMatcherExecutor) getMatchedPattern(cs *core.CallSite) string {
	for _, pattern := range e.IR.Patterns {
		if e.matchesPattern(cs.Target, pattern) {
			return pattern
		}
	}
	return ""
}
