package dsl

import (
	"fmt"
	"log"
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
// Returns an error if the IR is invalid (empty ReceiverType or MethodName).
func NewTypeConstrainedCallExecutor(ir *TypeConstrainedCallIR, cg *core.CallGraph) (*TypeConstrainedCallExecutor, error) {
	if err := validateTypeConstrainedCallIR(ir); err != nil {
		return nil, err
	}
	return &TypeConstrainedCallExecutor{
		IR:        ir,
		CallGraph: cg,
	}, nil
}

// validateTypeConstrainedCallIR checks that the IR has valid required fields
// and applies defaults for optional fields.
func validateTypeConstrainedCallIR(ir *TypeConstrainedCallIR) error {
	if ir.ReceiverType == "" {
		return fmt.Errorf("TypeConstrainedCallIR: receiverType must not be empty")
	}
	if ir.MethodName == "" {
		return fmt.Errorf("TypeConstrainedCallIR: methodName must not be empty")
	}
	// Apply defaults
	if ir.MinConfidence == 0 {
		ir.MinConfidence = 0.5
	}
	switch ir.FallbackMode {
	case "", "name", "none", "warn":
		if ir.FallbackMode == "" {
			ir.FallbackMode = "name"
		}
	default:
		log.Printf("WARNING: TypeConstrainedCallIR: unknown fallbackMode %q, defaulting to \"name\"", ir.FallbackMode)
		ir.FallbackMode = "name"
	}
	return nil
}

// Execute finds all call sites matching the type-constrained pattern.
// Returns bare CallSite slice (mirrors CallMatcherExecutor.Execute signature).
func (e *TypeConstrainedCallExecutor) Execute() []core.CallSite {
	if e.CallGraph == nil {
		return []core.CallSite{}
	}

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
// Reuses CallMatchResult from call_matcher.go.
func (e *TypeConstrainedCallExecutor) ExecuteWithContext() []CallMatchResult {
	if e.CallGraph == nil {
		return []CallMatchResult{}
	}

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
//
// Algorithm:
//  1. Method name must match (extracted from Target via last "." split)
//  2. If type inference resolved this call with sufficient confidence, check receiver type
//  3. Otherwise apply fallback mode
func (e *TypeConstrainedCallExecutor) matchesCallSite(cs *core.CallSite) bool {
	// Step 1: Method name must match
	if !e.matchesMethod(cs.Target) {
		return false
	}

	// Step 2: If type inference resolved this call, check receiver type
	if cs.ResolvedViaTypeInference && cs.TypeConfidence >= float32(e.IR.MinConfidence) {
		return matchesReceiverType(cs.InferredType, e.IR.ReceiverType)
	}

	// Step 3: Type inference unavailable or below confidence — apply fallback
	switch e.IR.FallbackMode {
	case "none":
		return false // Strict: no type info = no match
	case "warn":
		log.Printf("WARNING: type_constrained_call: no type info for %s at %s:%d, skipping",
			cs.Target, cs.Location.File, cs.Location.Line)
		return false
	default:
		return true // "name" or default: fall back to name-only matching
	}
}

// matchesMethod extracts the method name from a call target and compares.
// "cursor.execute" -> "execute", "execute" -> "execute".
func (e *TypeConstrainedCallExecutor) matchesMethod(target string) bool {
	if target == "" {
		return false
	}
	methodName := target
	if idx := strings.LastIndex(target, "."); idx >= 0 {
		methodName = target[idx+1:]
	}
	return methodName == e.IR.MethodName
}

// matchesReceiverType checks if an inferred type matches a type pattern.
//
// Supported patterns:
//   - Exact match: "sqlite3.Cursor" == "sqlite3.Cursor"
//   - Wildcard prefix: "*Cursor" matches "sqlite3.Cursor"
//   - Wildcard suffix: "sqlite3.*" matches "sqlite3.Cursor"
//   - Contains wildcard: "*Connection*" matches "DatabaseConnection"
//   - Short name match: "Cursor" (no dots) matches "sqlite3.Cursor" (via suffix ".Cursor")
func matchesReceiverType(actual, pattern string) bool {
	if actual == "" || pattern == "" {
		return false
	}

	// Exact match (fast path)
	if actual == pattern {
		return true
	}

	// Wildcard patterns
	if strings.Contains(pattern, "*") {
		if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
			// *substring* -> contains
			substr := strings.Trim(pattern, "*")
			return strings.Contains(actual, substr)
		}
		if strings.HasPrefix(pattern, "*") {
			// *suffix -> ends with
			return strings.HasSuffix(actual, pattern[1:])
		}
		if strings.HasSuffix(pattern, "*") {
			// prefix* -> starts with
			return strings.HasPrefix(actual, pattern[:len(pattern)-1])
		}
	}

	// Short name match: pattern "Cursor" matches actual "sqlite3.Cursor"
	// Only when pattern has no dots (is a simple class name)
	if !strings.Contains(pattern, ".") {
		suffix := "." + pattern
		return strings.HasSuffix(actual, suffix)
	}

	return false
}
