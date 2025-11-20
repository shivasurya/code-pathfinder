package dsl

import (
	"strconv"
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

// matchesCallSite checks if a call site matches the pattern AND argument constraints.
//
// Algorithm:
//  1. Check if function name matches any pattern
//  2. Check if arguments satisfy constraints
//  3. Return true only if both match
//
// Performance: O(P + A) where P=patterns, A=arguments.
func (e *CallMatcherExecutor) matchesCallSite(cs *core.CallSite) bool {
	target := cs.Target

	// Step 1: Check if function name matches
	matchesTarget := false
	for _, pattern := range e.IR.Patterns {
		if e.matchesPattern(target, pattern) {
			matchesTarget = true
			break
		}
	}

	if !matchesTarget {
		return false // Function name doesn't match
	}

	// Step 2: Check argument constraints
	if !e.matchesArguments(cs) {
		return false // Arguments don't match constraints
	}

	return true // Both function name and arguments match!
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

// parseKeywordArguments extracts keyword arguments from CallSite.Arguments.
//
// Example:
//
//	Input:  []Argument{
//	          {Value: "x", Position: 0},
//	          {Value: "y=2", Position: 1},
//	          {Value: "debug=True", Position: 2}
//	        }
//	Output: map[string]string{
//	          "y": "2",
//	          "debug": "True"
//	        }
//
// Algorithm:
//  1. Iterate through all arguments
//  2. Check if argument contains "=" (keyword arg format)
//  3. Split by "=" to get name and value
//  4. Trim whitespace and store in map
//
// Performance: O(N) where N = number of arguments (~2-5 typically).
func (e *CallMatcherExecutor) parseKeywordArguments(args []core.Argument) map[string]string {
	kwargs := make(map[string]string)

	for _, arg := range args {
		// Check if argument is in "key=value" format
		if strings.Contains(arg.Value, "=") {
			parts := strings.SplitN(arg.Value, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				kwargs[key] = value
			}
		}
	}

	return kwargs
}

// matchesArguments checks if CallSite arguments satisfy all constraints.
//
// Algorithm:
//  1. If no constraints, return true (backward compatibility)
//  2. Parse keyword arguments from CallSite
//  3. Check each constraint against actual values
//  4. Return true only if all constraints satisfied
//
// Performance: O(K) where K = number of keyword constraints (~1-3 typically).
func (e *CallMatcherExecutor) matchesArguments(cs *core.CallSite) bool {
	// No constraints = always match (backward compatibility)
	if len(e.IR.KeywordArgs) == 0 {
		return true
	}

	// Parse keyword arguments from CallSite
	keywordArgs := e.parseKeywordArguments(cs.Arguments)

	// Check each keyword argument constraint
	for name, constraint := range e.IR.KeywordArgs {
		actualValue, exists := keywordArgs[name]
		if !exists {
			// Required keyword argument not present in call
			return false
		}

		if !e.matchesArgumentValue(actualValue, constraint) {
			// Argument value doesn't match constraint
			return false
		}
	}

	return true // All constraints satisfied!
}

// matchesArgumentValue checks if actual value matches constraint.
//
// Handles:
//   - Exact string match: "0.0.0.0" == "0.0.0.0"
//   - Boolean match: "True" == true, "False" == false
//   - Number match: "777" == 777, "0o777" == 0o777 (octal)
//   - Case-insensitive for booleans: "true" == "True" == "TRUE"
//
// Performance: O(1) for single values.
func (e *CallMatcherExecutor) matchesArgumentValue(actual string, constraint ArgumentConstraint) bool {
	// Clean actual value (remove quotes, trim whitespace)
	actual = e.cleanValue(actual)

	// Get expected value from constraint
	expected := constraint.Value

	// Type-specific matching
	switch v := expected.(type) {
	case string:
		// String comparison
		return e.normalizeValue(actual) == e.normalizeValue(v)

	case bool:
		// Boolean comparison
		return e.matchesBoolean(actual, v)

	case float64:
		// Number comparison (JSON numbers are float64)
		return e.matchesNumber(actual, v)

	case nil:
		// Null/None comparison
		return actual == "None" || actual == "nil" || actual == "null"

	default:
		// Unknown type
		return false
	}
}

// cleanValue removes surrounding quotes and whitespace from argument values.
//
// Examples:
//   "\"0.0.0.0\"" → "0.0.0.0"
//   "'localhost'" → "localhost"
//   "  True  "    → "True"
func (e *CallMatcherExecutor) cleanValue(value string) string {
	value = strings.TrimSpace(value)

	// Remove surrounding quotes (single or double)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}

	return value
}

// normalizeValue normalizes string values for comparison.
//
// Handles:
//   - Case-insensitive for specific values: "true" == "True" == "TRUE"
//   - Preserves case for general strings: "MyValue" != "myvalue"
func (e *CallMatcherExecutor) normalizeValue(value string) string {
	lower := strings.ToLower(value)

	// Normalize boolean strings (case-insensitive)
	if lower == "true" || lower == "false" {
		return lower
	}

	// Normalize None/null (case-insensitive)
	if lower == "none" || lower == "null" || lower == "nil" {
		return "none"
	}

	// For everything else, return as-is (case-sensitive)
	return value
}

// matchesBoolean checks if string represents a boolean value.
//
// Matches:
//   - Python: True, False, true, false, TRUE, FALSE
//   - JSON: true, false
//   - Numeric: 1 (true), 0 (false)
func (e *CallMatcherExecutor) matchesBoolean(actual string, expected bool) bool {
	actual = strings.ToLower(strings.TrimSpace(actual))

	if expected {
		// Match truthy values
		return actual == "true" || actual == "1"
	}
	// Match falsy values
	return actual == "false" || actual == "0"
}

// matchesNumber checks if string represents a numeric value.
//
// Handles:
//   - Integers: "777", "42"
//   - Octal: "0o777", "0777" (Go format)
//   - Hex: "0xFF", "0xff"
//   - Floats: "3.14"
func (e *CallMatcherExecutor) matchesNumber(actual string, expected float64) bool {
	actual = strings.TrimSpace(actual)

	// Try parsing as integer (supports decimal, octal, hex)
	if i, err := strconv.ParseInt(actual, 0, 64); err == nil {
		return float64(i) == expected
	}

	// Try parsing as float
	if f, err := strconv.ParseFloat(actual, 64); err == nil {
		return f == expected
	}

	return false
}
