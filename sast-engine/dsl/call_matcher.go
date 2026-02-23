package dsl

import (
	"strconv"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
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
//
//	F = number of functions
//	C = avg call sites per function (~5-10)
//	P = number of patterns (~2-5)
//
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
//
//	matchesPattern("eval", "eval") → true
//	matchesPattern("request.GET", "request.*") → true
//	matchesPattern("utils.sanitize", "*.sanitize") → true
//	matchesPattern("os.system", "eval") → false
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

	if after, ok := strings.CutPrefix(pattern, "*"); ok {
		// *suffix → ends with
		suffix := after
		return strings.HasSuffix(target, suffix)
	}

	if before, ok := strings.CutSuffix(pattern, "*"); ok {
		// prefix* → starts with
		prefix := before
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
// Also checks argument constraints to ensure full matching logic is applied.
func (e *CallMatcherExecutor) getMatchedPattern(cs *core.CallSite) string {
	// Check if the callsite matches (both pattern AND arguments)
	if !e.matchesCallSite(cs) {
		return "" // Doesn't match pattern or arguments don't satisfy constraints
	}

	// Find which pattern matched
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

// matchesArguments checks both positional and keyword argument constraints.
//
// Algorithm:
//  1. If no constraints, return true (backward compatibility)
//  2. Check positional arguments first
//  3. Check keyword arguments
//  4. Return true only if all constraints satisfied
//
// Performance: O(P + K) where P=positional constraints, K=keyword constraints.
func (e *CallMatcherExecutor) matchesArguments(cs *core.CallSite) bool {
	// No constraints = always match (backward compatibility)
	if len(e.IR.PositionalArgs) == 0 && len(e.IR.KeywordArgs) == 0 {
		return true
	}

	// Check positional arguments first
	if !e.matchesPositionalArguments(cs.Arguments) {
		return false
	}

	// Check keyword arguments
	if !e.matchesKeywordArguments(cs.Arguments) {
		return false
	}

	return true // All constraints satisfied!
}

// parseTupleIndex parses position strings with optional tuple indexing.
//
// Supports:
//   - Simple position: "0" → (0, 0, false)
//   - Tuple indexing: "0[1]" → (0, 1, true)
//
// Parameters:
//   - posStr: position string from IR (e.g., "0", "0[1]")
//
// Returns:
//   - pos: argument position (0-indexed)
//   - tupleIdx: tuple element index (0-indexed, only valid if isTupleIndex=true)
//   - isTupleIndex: whether tuple indexing syntax was used
func parseTupleIndex(posStr string) (int, int, bool, bool) {
	// Check if it looks like tuple indexing syntax
	hasOpenBracket := strings.Contains(posStr, "[")
	hasCloseBracket := strings.Contains(posStr, "]")

	// Simple position (no brackets)
	if !hasOpenBracket {
		pos, err := strconv.Atoi(posStr)
		if err != nil {
			return 0, 0, false, false // Parse error
		}
		return pos, 0, false, true // Valid simple position
	}

	// Malformed: has bracket but not both open and close
	if !hasCloseBracket {
		// Try to parse just the part before [ as a fallback
		parts := strings.SplitN(posStr, "[", 2)
		pos, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, false, false // Parse error
		}
		return pos, 0, false, true // Return simple position, not tuple index
	}

	// Parse "0[1]" format
	parts := strings.SplitN(posStr, "[", 2)
	pos, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false, false // Parse error
	}

	// Extract index from "[1]" (remove brackets)
	idxStr := strings.TrimSuffix(parts[1], "]")
	idx, err := strconv.Atoi(idxStr)
	if err != nil {
		return 0, 0, false, false // Parse error
	}

	return pos, idx, true, true // Valid tuple index
}

// extractTupleElement extracts an element at the specified index from a tuple string.
//
// Algorithm:
//  1. Check if string looks like a tuple (starts with '(' or '[')
//  2. Strip outer parentheses/brackets
//  3. Split by comma (handles simple cases, not nested structures)
//  4. Extract element at index
//  5. Clean up quotes and whitespace
//
// Examples:
//   - extractTupleElement("(\"0.0.0.0\", 8080)", 0) → ("0.0.0.0", true)
//   - extractTupleElement("(\"0.0.0.0\", 8080)", 1) → ("8080", true)
//   - extractTupleElement("(\"a\", \"b\", \"c\")", 1) → ("b", true)
//   - extractTupleElement("(\"a\", \"b\")", 5) → ("", false)  // out of bounds
//   - extractTupleElement("(\"\", 8080)", 0) → ("", true)     // empty string is valid
//   - extractTupleElement("not_a_tuple", 0) → ("not_a_tuple", true)
//
// Limitations:
//   - Does not handle nested tuples/lists
//   - Uses simple comma splitting (could break on complex expressions)
//
// Parameters:
//   - tupleStr: string representation of tuple from AST
//   - index: 0-indexed position of element to extract
//
// Returns:
//   - value: Extracted element as string (can be empty string if that's the actual value)
//   - ok: true if extraction succeeded, false if index out of bounds
func extractTupleElement(tupleStr string, index int) (string, bool) {
	tupleStr = strings.TrimSpace(tupleStr)

	// Check if it's a tuple or list
	if !strings.HasPrefix(tupleStr, "(") && !strings.HasPrefix(tupleStr, "[") {
		// Not a tuple/list, return as-is if index is 0
		if index == 0 {
			// Remove quotes from plain strings too
			result := strings.Trim(tupleStr, `"'`)
			return result, true
		}
		return "", false // Index out of bounds for non-tuple
	}

	// Strip outer parentheses or brackets
	inner := tupleStr[1 : len(tupleStr)-1]
	inner = strings.TrimSpace(inner)

	// Handle empty tuple/list
	if inner == "" {
		return "", false // Empty tuple has no elements
	}

	// Split by comma
	// Note: This is a simple implementation that doesn't handle nested structures
	// For production, we'd need a proper parser
	elements := strings.Split(inner, ",")

	if index >= len(elements) {
		return "", false // Index out of bounds
	}

	element := strings.TrimSpace(elements[index])

	// Remove quotes if present (handles both single and double quotes)
	element = strings.Trim(element, `"'`)

	return element, true
}

// matchesPositionalArguments checks positional argument constraints.
//
// Algorithm:
//  1. If no positional constraints, return true
//  2. For each position constraint:
//     a. Parse position string (supports tuple indexing: "0[1]")
//     b. Check if position exists in arguments
//     c. Extract tuple element if tuple indexing used
//     d. Match argument value against constraint
//
// Supports:
//   - Simple positional: {0: "value"} matches args[0] == "value"
//   - Tuple indexing: {"0[1]": "value"} matches args[0] tuple element 1 == "value"
//
// Performance: O(P) where P = number of positional constraints.
func (e *CallMatcherExecutor) matchesPositionalArguments(args []core.Argument) bool {
	if len(e.IR.PositionalArgs) == 0 {
		return true // No positional constraints
	}

	for posStr, constraint := range e.IR.PositionalArgs {
		// Parse position string (supports tuple indexing)
		pos, tupleIdx, isTupleIndex, valid := parseTupleIndex(posStr)

		// Check if position string was valid
		if !valid {
			return false // Invalid position string
		}

		// Check if position exists in arguments
		if pos >= len(args) {
			return false // Argument at position doesn't exist
		}

		// Get actual argument value
		actualValue := args[pos].Value

		// Extract tuple element if tuple indexing used
		if isTupleIndex {
			var ok bool
			actualValue, ok = extractTupleElement(actualValue, tupleIdx)
			if !ok {
				return false // Tuple index out of bounds
			}
			// Note: actualValue can be empty string if that's the actual tuple element value
		}

		// Match against constraint
		if !e.matchesArgumentValue(actualValue, constraint) {
			return false
		}
	}

	return true
}

// matchesKeywordArguments checks keyword argument constraints (refactored from PR #2).
//
// Algorithm:
//  1. If no keyword constraints, return true
//  2. Parse keyword arguments from CallSite
//  3. Check each constraint against actual values
//
// Performance: O(K) where K = number of keyword constraints.
func (e *CallMatcherExecutor) matchesKeywordArguments(args []core.Argument) bool {
	if len(e.IR.KeywordArgs) == 0 {
		return true // No keyword constraints
	}

	// Parse keyword arguments from CallSite
	keywordArgs := e.parseKeywordArguments(args)

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
//   - OR logic: ["w", "a", "w+"] matches any value in list
//   - Wildcard matching: "0o7*" matches "0o777", "0o755", etc.
//
// Performance: O(1) for single values, O(N) for OR logic with N values.
func (e *CallMatcherExecutor) matchesArgumentValue(actual string, constraint ArgumentConstraint) bool {
	// Clean actual value (remove quotes, trim whitespace)
	actual = e.cleanValue(actual)

	// Get expected value from constraint
	expected := constraint.Value

	// Handle list of values (OR logic)
	if values, isList := expected.([]any); isList {
		return e.matchesAnyValue(actual, values, constraint.Wildcard)
	}

	// Single value matching
	return e.matchesSingleValue(actual, expected, constraint.Wildcard)
}

// matchesAnyValue checks if actual matches any value in list (OR logic).
//
// Algorithm:
//  1. Iterate through all values in list
//  2. Return true if any value matches
//  3. Return false if no values match
//
// Performance: O(N) where N = number of values in list.
func (e *CallMatcherExecutor) matchesAnyValue(actual string, values []any, wildcard bool) bool {
	for _, v := range values {
		if e.matchesSingleValue(actual, v, wildcard) {
			return true
		}
	}
	return false
}

// matchesSingleValue checks if actual matches a single expected value.
//
// Algorithm:
//  1. Determine type of expected value
//  2. Call appropriate type-specific matcher
//  3. Return match result
//
// Performance: O(1) for non-wildcard, O(M) for wildcard where M = string length.
func (e *CallMatcherExecutor) matchesSingleValue(actual string, expected any, wildcard bool) bool {
	switch v := expected.(type) {
	case string:
		// String comparison with optional wildcard
		return e.matchesString(actual, v, wildcard)

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

// matchesString matches string values with optional wildcard support.
//
// Algorithm:
//  1. If wildcard is enabled, use wildcard matching
//  2. Otherwise, use normalized exact match
//
// Performance: O(1) for exact match, O(M) for wildcard where M = string length.
func (e *CallMatcherExecutor) matchesString(actual string, expected string, wildcard bool) bool {
	if wildcard {
		return e.matchesWildcard(actual, expected)
	}
	return e.normalizeValue(actual) == e.normalizeValue(expected)
}

// matchesWildcard performs wildcard pattern matching.
//
// Supports:
//   - * (zero or more characters)
//   - ? (single character)
//
// Examples:
//   - "0.0.*" matches "0.0.0.0", "0.0.255.255"
//   - "0o7*" matches "0o777", "0o755"
//   - "test-??.txt" matches "test-01.txt"
//
// Performance: O(M + N) where M = string length, N = pattern length.
func (e *CallMatcherExecutor) matchesWildcard(actual string, pattern string) bool {
	// Normalize both strings
	actual = e.normalizeValue(actual)
	pattern = e.normalizeValue(pattern)

	return e.wildcardMatch(actual, pattern)
}

// wildcardMatch implements wildcard matching algorithm.
//
// Algorithm:
//  1. Use two-pointer approach with backtracking
//  2. Handle * by recording position and trying to match rest
//  3. Handle ? by matching single character
//  4. Backtrack to last * if current match fails
//
// Performance: O(M + N) average case, O(M * N) worst case.
func (e *CallMatcherExecutor) wildcardMatch(str string, pattern string) bool {
	sIdx, pIdx := 0, 0
	starIdx, matchIdx := -1, 0

	for sIdx < len(str) {
		if pIdx < len(pattern) {
			if pattern[pIdx] == '*' {
				// Record star position
				starIdx = pIdx
				matchIdx = sIdx
				pIdx++
				continue
			} else if pattern[pIdx] == '?' || pattern[pIdx] == str[sIdx] {
				// Match single character
				sIdx++
				pIdx++
				continue
			}
		}

		// No match, backtrack to last star
		if starIdx != -1 {
			pIdx = starIdx + 1
			matchIdx++
			sIdx = matchIdx
			continue
		}

		return false
	}

	// Handle remaining stars in pattern
	for pIdx < len(pattern) && pattern[pIdx] == '*' {
		pIdx++
	}

	return pIdx == len(pattern)
}

// cleanValue removes surrounding quotes and whitespace from argument values.
//
// Examples:
//
//	"\"0.0.0.0\"" → "0.0.0.0"
//	"'localhost'" → "localhost"
//	"  True  "    → "True"
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
