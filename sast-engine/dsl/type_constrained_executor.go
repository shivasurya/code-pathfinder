package dsl

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// TypeConstrainedCallExecutor executes type_constrained_call IR against a callgraph.
// It matches method calls where the receiver's inferred type matches the constraint.
type TypeConstrainedCallExecutor struct {
	IR        *TypeConstrainedCallIR
	CallGraph *core.CallGraph
}

// NewTypeConstrainedCallExecutor creates a new executor.
func NewTypeConstrainedCallExecutor(ir *TypeConstrainedCallIR, cg *core.CallGraph) *TypeConstrainedCallExecutor {
	return &TypeConstrainedCallExecutor{
		IR:        ir,
		CallGraph: cg,
	}
}

// Execute finds all call sites matching the type-constrained pattern.
func (e *TypeConstrainedCallExecutor) Execute() []core.CallSite {
	matches := []core.CallSite{}

	for _, callSites := range e.CallGraph.CallSites {
		for _, cs := range callSites {
			if e.matchesCallSite(&cs) {
				matches = append(matches, cs)
			}
		}
	}

	return matches
}

// ExecuteWithContext returns matches with function FQN context (for loader integration).
func (e *TypeConstrainedCallExecutor) ExecuteWithContext() []CallMatchResult {
	results := []CallMatchResult{}

	for functionFQN, callSites := range e.CallGraph.CallSites {
		for _, cs := range callSites {
			if e.matchesCallSite(&cs) {
				results = append(results, CallMatchResult{
					CallSite:    cs,
					MatchedBy:   e.IR.ReceiverType + "." + e.IR.MethodName,
					FunctionFQN: functionFQN,
					SourceFile:  cs.Location.File,
					Line:        cs.Location.Line,
				})
			}
		}
	}

	return results
}

// matchesCallSite checks if a call site matches the type-constrained pattern.
func (e *TypeConstrainedCallExecutor) matchesCallSite(cs *core.CallSite) bool {
	// Step 1: Method name must match
	if !e.matchesMethod(cs.Target) {
		return false
	}

	// Step 2: If type inference resolved this call, check receiver type
	if cs.ResolvedViaTypeInference && cs.TypeConfidence >= float32(e.IR.MinConfidence) {
		return e.matchesReceiverType(cs.InferredType, e.IR.ReceiverType)
	}

	// Step 3: Type inference unavailable — apply fallback
	switch e.IR.FallbackMode {
	case "none":
		return false // Strict: no type info = no match
	case "name", "warn":
		return true // Lenient: fall back to name-only matching
	default:
		return true // Default: name fallback
	}
}

// matchesMethod extracts the method name from a call target and compares.
// "cursor.execute" → "execute", "execute" → "execute"
func (e *TypeConstrainedCallExecutor) matchesMethod(target string) bool {
	methodName := target
	if idx := strings.LastIndex(target, "."); idx >= 0 {
		methodName = target[idx+1:]
	}
	return methodName == e.IR.MethodName
}

// matchesReceiverType checks if the inferred type matches the pattern.
// Supports: exact match, short-name suffix, wildcard prefix/suffix.
func (e *TypeConstrainedCallExecutor) matchesReceiverType(actual, pattern string) bool {
	if actual == "" {
		return false
	}

	// Exact match: "sqlite3.Cursor" == "sqlite3.Cursor"
	if actual == pattern {
		return true
	}

	// Wildcard prefix: "*Cursor" matches "sqlite3.Cursor"
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(actual, pattern[1:])
	}

	// Wildcard suffix: "sqlite3.*" matches "sqlite3.Cursor"
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(actual, pattern[:len(pattern)-1])
	}

	// Short name match: pattern "Cursor" matches "sqlite3.Cursor"
	// Only when pattern has no dots (is a simple class name)
	if !strings.Contains(pattern, ".") {
		suffix := "." + pattern
		return strings.HasSuffix(actual, suffix)
	}

	return false
}
