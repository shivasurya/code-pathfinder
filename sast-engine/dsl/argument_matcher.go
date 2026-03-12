package dsl

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// MatchesArguments checks both positional and keyword argument constraints.
// Shared between CallMatcherExecutor and TypeConstrainedCallExecutor.
func MatchesArguments(cs *core.CallSite, positionalArgs, keywordArgs map[string]ArgumentConstraint) bool {
	if len(positionalArgs) == 0 && len(keywordArgs) == 0 {
		return true
	}

	if !MatchesPositionalArguments(cs.Arguments, positionalArgs) {
		return false
	}

	if !MatchesKeywordArguments(cs.Arguments, keywordArgs) {
		return false
	}

	return true
}

// MatchesPositionalArguments checks positional argument constraints.
func MatchesPositionalArguments(args []core.Argument, positionalArgs map[string]ArgumentConstraint) bool {
	if len(positionalArgs) == 0 {
		return true
	}

	for posStr, constraint := range positionalArgs {
		pos, tupleIdx, isTupleIndex, valid := parseTupleIndex(posStr)
		if !valid {
			return false
		}

		if pos >= len(args) {
			return false
		}

		actualValue := args[pos].Value

		if isTupleIndex {
			var ok bool
			actualValue, ok = extractTupleElement(actualValue, tupleIdx)
			if !ok {
				return false
			}
		}

		if !MatchesArgumentValue(actualValue, constraint) {
			return false
		}
	}

	return true
}

// MatchesKeywordArguments checks keyword argument constraints with comparator support.
func MatchesKeywordArguments(args []core.Argument, keywordArgs map[string]ArgumentConstraint) bool {
	if len(keywordArgs) == 0 {
		return true
	}

	parsedKwargs := ParseKeywordArguments(args)

	for name, constraint := range keywordArgs {
		actualValue, exists := parsedKwargs[name]

		// Handle "missing" comparator: check that keyword is ABSENT
		if constraint.Comparator == "missing" {
			if exists {
				return false // Keyword IS present — missing() check fails
			}
			continue // Keyword is absent — missing() check passes
		}

		if !exists {
			return false // Required keyword not present
		}

		if !MatchesArgumentValue(actualValue, constraint) {
			return false
		}
	}

	return true
}

// ParseKeywordArguments extracts keyword arguments from CallSite.Arguments.
func ParseKeywordArguments(args []core.Argument) map[string]string {
	kwargs := make(map[string]string)
	for _, arg := range args {
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

// MatchesArgumentValue checks if actual value matches constraint, with comparator support.
func MatchesArgumentValue(actual string, constraint ArgumentConstraint) bool {
	actual = CleanValue(actual)

	switch constraint.Comparator {
	case "lt":
		return compareNumeric(actual, constraint.Value, func(a, b float64) bool { return a < b })
	case "gt":
		return compareNumeric(actual, constraint.Value, func(a, b float64) bool { return a > b })
	case "lte":
		return compareNumeric(actual, constraint.Value, func(a, b float64) bool { return a <= b })
	case "gte":
		return compareNumeric(actual, constraint.Value, func(a, b float64) bool { return a >= b })
	case "regex":
		pattern, ok := constraint.Value.(string)
		if !ok {
			return false
		}
		matched, err := regexp.MatchString(pattern, actual)
		return err == nil && matched
	case "missing":
		return false // "missing" is handled at the keyword level
	default:
		// Existing exact/wildcard/boolean/number matching
		return matchesSingleValueShared(actual, constraint.Value, constraint.Wildcard)
	}
}

// compareNumeric parses actual as a number and compares against expected.
func compareNumeric(actual string, expected any, cmp func(float64, float64) bool) bool {
	actual = strings.TrimSpace(actual)

	var actualNum float64
	if i, err := strconv.ParseInt(actual, 0, 64); err == nil {
		actualNum = float64(i)
	} else if f, err := strconv.ParseFloat(actual, 64); err == nil {
		actualNum = f
	} else {
		return false
	}

	var expectedNum float64
	switch v := expected.(type) {
	case float64:
		expectedNum = v
	case int:
		expectedNum = float64(v)
	case int64:
		expectedNum = float64(v)
	default:
		return false
	}

	return cmp(actualNum, expectedNum)
}

// matchesSingleValueShared handles exact/wildcard/boolean/number matching.
func matchesSingleValueShared(actual string, expected any, wildcard bool) bool {
	switch v := expected.(type) {
	case string:
		if wildcard {
			return wildcardMatchShared(NormalizeValue(actual), NormalizeValue(v))
		}
		return NormalizeValue(actual) == NormalizeValue(v)
	case bool:
		return matchesBooleanShared(actual, v)
	case float64:
		return matchesNumberShared(actual, v)
	case nil:
		return actual == "None" || actual == "nil" || actual == "null"
	default:
		return false
	}
}

// matchesBooleanShared matches boolean argument values.
func matchesBooleanShared(actual string, expected bool) bool {
	actual = strings.ToLower(strings.TrimSpace(actual))
	if expected {
		return actual == "true" || actual == "1"
	}
	return actual == "false" || actual == "0"
}

// matchesNumberShared matches numeric argument values.
func matchesNumberShared(actual string, expected float64) bool {
	actual = strings.TrimSpace(actual)
	if i, err := strconv.ParseInt(actual, 0, 64); err == nil {
		return float64(i) == expected
	}
	if f, err := strconv.ParseFloat(actual, 64); err == nil {
		return f == expected
	}
	return false
}

// CleanValue removes surrounding quotes and whitespace.
func CleanValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	return value
}

// NormalizeValue normalizes string values for comparison.
func NormalizeValue(value string) string {
	lower := strings.ToLower(value)
	if lower == "true" || lower == "false" {
		return lower
	}
	if lower == "none" || lower == "null" || lower == "nil" {
		return "none"
	}
	return value
}

// wildcardMatchShared implements wildcard matching with * and ? support.
func wildcardMatchShared(str, pattern string) bool {
	sIdx, pIdx := 0, 0
	starIdx, matchIdx := -1, 0

	for sIdx < len(str) {
		if pIdx < len(pattern) {
			if pattern[pIdx] == '*' {
				starIdx = pIdx
				matchIdx = sIdx
				pIdx++
				continue
			} else if pattern[pIdx] == '?' || pattern[pIdx] == str[sIdx] {
				sIdx++
				pIdx++
				continue
			}
		}

		if starIdx != -1 {
			pIdx = starIdx + 1
			matchIdx++
			sIdx = matchIdx
			continue
		}

		return false
	}

	for pIdx < len(pattern) && pattern[pIdx] == '*' {
		pIdx++
	}

	return pIdx == len(pattern)
}
