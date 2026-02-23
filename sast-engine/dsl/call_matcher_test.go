package dsl

import (
	"fmt"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func TestCallMatcherExecutor_Execute(t *testing.T) {
	// Setup test callgraph
	cg := core.NewCallGraph()

	cg.CallSites["test.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{File: "test.py", Line: 10}},
		{Target: "exec", Location: core.Location{File: "test.py", Line: 15}},
		{Target: "print", Location: core.Location{File: "test.py", Line: 20}},
	}

	t.Run("exact match single pattern", func(t *testing.T) {
		ir := &CallMatcherIR{
			Type:     "call_matcher",
			Patterns: []string{"eval"},
			Wildcard: false,
		}

		executor := NewCallMatcherExecutor(ir, cg)
		matches := executor.Execute()

		assert.Len(t, matches, 1)
		assert.Equal(t, "eval", matches[0].Target)
	})

	t.Run("exact match multiple patterns", func(t *testing.T) {
		ir := &CallMatcherIR{
			Patterns: []string{"eval", "exec"},
			Wildcard: false,
		}

		executor := NewCallMatcherExecutor(ir, cg)
		matches := executor.Execute()

		assert.Len(t, matches, 2)
	})

	t.Run("wildcard prefix match", func(t *testing.T) {
		cg2 := core.NewCallGraph()
		cg2.CallSites["test.main"] = []core.CallSite{
			{Target: "request.GET"},
			{Target: "request.POST"},
			{Target: "utils.sanitize"},
		}

		ir := &CallMatcherIR{
			Patterns: []string{"request.*"},
			Wildcard: true,
		}

		executor := NewCallMatcherExecutor(ir, cg2)
		matches := executor.Execute()

		assert.Len(t, matches, 2)
	})

	t.Run("wildcard suffix match", func(t *testing.T) {
		cg2 := core.NewCallGraph()
		cg2.CallSites["test.main"] = []core.CallSite{
			{Target: "request.json"},
			{Target: "response.json"},
			{Target: "utils.parse"},
		}

		ir := &CallMatcherIR{
			Patterns: []string{"*.json"},
			Wildcard: true,
		}

		executor := NewCallMatcherExecutor(ir, cg2)
		matches := executor.Execute()

		assert.Len(t, matches, 2)
	})

	t.Run("no matches", func(t *testing.T) {
		ir := &CallMatcherIR{
			Patterns: []string{"nonexistent"},
			Wildcard: false,
		}

		executor := NewCallMatcherExecutor(ir, cg)
		matches := executor.Execute()

		assert.Len(t, matches, 0)
	})
}

func TestCallMatcherExecutor_ExecuteWithContext(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["test.process_data"] = []core.CallSite{
		{Target: "eval", Location: core.Location{File: "app.py", Line: 42}},
	}

	ir := &CallMatcherIR{
		Patterns: []string{"eval"},
		Wildcard: false,
	}

	executor := NewCallMatcherExecutor(ir, cg)
	results := executor.ExecuteWithContext()

	assert.Len(t, results, 1)
	assert.Equal(t, "eval", results[0].MatchedBy)
	assert.Equal(t, "test.process_data", results[0].FunctionFQN)
	assert.Equal(t, "app.py", results[0].SourceFile)
	assert.Equal(t, 42, results[0].Line)
}

func BenchmarkCallMatcherExecutor(b *testing.B) {
	// Create realistic callgraph (1000 functions, 7 calls each)
	cg := core.NewCallGraph()
	for i := range 1000 {
		funcName := fmt.Sprintf("module.func_%d", i)
		cg.CallSites[funcName] = []core.CallSite{
			{Target: "print"},
			{Target: "len"},
			{Target: "str"},
			{Target: "request.GET"},
			{Target: "utils.sanitize"},
			{Target: "db.execute"},
			{Target: "json.dumps"},
		}
	}

	ir := &CallMatcherIR{
		Patterns: []string{"request.*", "db.execute"},
		Wildcard: true,
	}

	executor := NewCallMatcherExecutor(ir, cg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.Execute()
	}
}

// TestParseKeywordArguments_EmptyArgs tests parsing with no arguments.
func TestParseKeywordArguments_EmptyArgs(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{},
	}

	args := []core.Argument{}
	result := executor.parseKeywordArguments(args)

	assert.Empty(t, result, "Expected empty map for empty arguments")
}

// TestParseKeywordArguments_PositionalOnly tests parsing with only positional args.
func TestParseKeywordArguments_PositionalOnly(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{},
	}

	args := []core.Argument{
		{Value: "x", Position: 0},
		{Value: "y", Position: 1},
	}
	result := executor.parseKeywordArguments(args)

	assert.Empty(t, result, "Expected empty map for positional-only arguments")
}

// TestParseKeywordArguments_SingleKeyword tests parsing single keyword argument.
func TestParseKeywordArguments_SingleKeyword(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{},
	}

	args := []core.Argument{
		{Value: "debug=True", Position: 0},
	}
	result := executor.parseKeywordArguments(args)

	assert.Len(t, result, 1, "Expected 1 keyword argument")
	assert.Equal(t, "True", result["debug"], "Expected debug=True")
}

// TestParseKeywordArguments_MultipleKeywords tests parsing multiple keyword args.
func TestParseKeywordArguments_MultipleKeywords(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{},
	}

	args := []core.Argument{
		{Value: "host=\"0.0.0.0\"", Position: 0},
		{Value: "port=5000", Position: 1},
		{Value: "debug=True", Position: 2},
	}
	result := executor.parseKeywordArguments(args)

	expected := map[string]string{
		"host":  "\"0.0.0.0\"",
		"port":  "5000",
		"debug": "True",
	}

	assert.Len(t, result, len(expected), "Expected %d keyword arguments", len(expected))
	for key, expectedValue := range expected {
		assert.Equal(t, expectedValue, result[key], "Expected %s=%s", key, expectedValue)
	}
}

// TestParseKeywordArguments_MixedArgs tests parsing mixed positional and keyword args.
func TestParseKeywordArguments_MixedArgs(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{},
	}

	args := []core.Argument{
		{Value: "app", Position: 0},              // Positional
		{Value: "host=\"0.0.0.0\"", Position: 1}, // Keyword
		{Value: "5000", Position: 2},             // Positional
		{Value: "debug=True", Position: 3},       // Keyword
	}
	result := executor.parseKeywordArguments(args)

	expected := map[string]string{
		"host":  "\"0.0.0.0\"",
		"debug": "True",
	}

	assert.Len(t, result, len(expected), "Expected %d keyword arguments", len(expected))
	for key, expectedValue := range expected {
		assert.Equal(t, expectedValue, result[key], "Expected %s=%s", key, expectedValue)
	}
}

// TestParseKeywordArguments_WithWhitespace tests parsing with whitespace.
func TestParseKeywordArguments_WithWhitespace(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{},
	}

	args := []core.Argument{
		{Value: " debug = True ", Position: 0},
		{Value: "host = \"0.0.0.0\"", Position: 1},
	}
	result := executor.parseKeywordArguments(args)

	expected := map[string]string{
		"debug": "True",
		"host":  "\"0.0.0.0\"",
	}

	assert.Len(t, result, len(expected), "Expected %d keyword arguments", len(expected))
	for key, expectedValue := range expected {
		assert.Equal(t, expectedValue, result[key], "Expected %s=%s (whitespace should be trimmed)", key, expectedValue)
	}
}

// TestParseKeywordArguments_ComplexValues tests parsing complex values.
func TestParseKeywordArguments_ComplexValues(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{},
	}

	args := []core.Argument{
		{Value: "config={\"key\": \"value\"}", Position: 0},
		{Value: "url=\"http://example.com/path?param=value\"", Position: 1},
	}
	result := executor.parseKeywordArguments(args)

	assert.Len(t, result, 2, "Expected 2 keyword arguments")
	assert.Equal(t, "{\"key\": \"value\"}", result["config"], "Complex value not parsed correctly")
	assert.Equal(t, "\"http://example.com/path?param=value\"", result["url"], "URL with = not parsed correctly")
}

// TestParseKeywordArguments_EdgeCases tests edge cases.
func TestParseKeywordArguments_EdgeCases(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{},
	}

	// Test with malformed arguments (should skip or include based on behavior)
	args := []core.Argument{
		{Value: "=value", Position: 0}, // Missing key
		{Value: "key=", Position: 1},   // Missing value (should include)
		{Value: "validkey=validvalue", Position: 2},
	}
	result := executor.parseKeywordArguments(args)

	// Should parse "key=" as key with empty value, and "validkey=validvalue"
	// "=value" should be skipped as it has empty key after trim
	assert.Contains(t, result, "key", "Should parse key with empty value")
	assert.Contains(t, result, "validkey", "Should parse valid key=value")
	assert.Equal(t, "", result["key"], "Key with no value should have empty string")
	assert.Equal(t, "validvalue", result["validkey"], "Valid key=value should be parsed")
}

// TestArgumentConstraint_StructUsage tests ArgumentConstraint struct usage.
func TestArgumentConstraint_StructUsage(t *testing.T) {
	constraint := ArgumentConstraint{
		Value:    "0.0.0.0",
		Wildcard: false,
	}

	// Test that it can be used in CallMatcherIR
	ir := CallMatcherIR{
		Type:     "call_matcher",
		Patterns: []string{"app.run"},
		KeywordArgs: map[string]ArgumentConstraint{
			"host": constraint,
		},
	}

	// Verify the struct is valid
	assert.Equal(t, "0.0.0.0", ir.KeywordArgs["host"].Value, "ArgumentConstraint not stored correctly")
	assert.False(t, ir.KeywordArgs["host"].Wildcard, "Wildcard should be false")
}

// TestCallMatcherIR_BackwardCompatibility tests that IR without KeywordArgs still works.
func TestCallMatcherIR_BackwardCompatibility(t *testing.T) {
	// Old IR without KeywordArgs field
	ir := CallMatcherIR{
		Type:      "call_matcher",
		Patterns:  []string{"eval", "exec"},
		Wildcard:  false,
		MatchMode: "any",
	}

	// Should work fine (KeywordArgs is nil/empty)
	assert.Nil(t, ir.KeywordArgs, "Expected nil KeywordArgs for backward compatibility")

	// Verify it can still be used with executor
	cg := core.NewCallGraph()
	cg.CallSites["test.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{File: "test.py", Line: 10}},
	}

	executor := NewCallMatcherExecutor(&ir, cg)
	matches := executor.Execute()

	assert.Len(t, matches, 1, "Old IR should still work")
	assert.Equal(t, "eval", matches[0].Target, "Old IR should match correctly")
}

// TestMatchesArguments_NoConstraints tests backward compatibility with no constraints.
func TestMatchesArguments_NoConstraints(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"app.run"},
			// No KeywordArgs = should always match
		},
		CallGraph: &core.CallGraph{},
	}

	callSite := core.CallSite{
		Target: "app.run",
		Arguments: []core.Argument{
			{Value: "debug=True", Position: 0},
		},
	}

	assert.True(t, executor.matchesArguments(&callSite), "Expected to match when no constraints present (backward compatibility)")
}

// TestMatchesArguments_KeywordMatch_True tests matching debug=True.
func TestMatchesArguments_KeywordMatch_True(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"app.run"},
			KeywordArgs: map[string]ArgumentConstraint{
				"debug": {Value: true, Wildcard: false},
			},
		},
		CallGraph: &core.CallGraph{},
	}

	callSite := core.CallSite{
		Target: "app.run",
		Arguments: []core.Argument{
			{Value: "debug=True", Position: 0},
		},
	}

	assert.True(t, executor.matchesArguments(&callSite), "Expected to match debug=True")
}

// TestMatchesArguments_KeywordMatch_False tests not matching when debug=False.
func TestMatchesArguments_KeywordMatch_False(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"app.run"},
			KeywordArgs: map[string]ArgumentConstraint{
				"debug": {Value: true, Wildcard: false},
			},
		},
		CallGraph: &core.CallGraph{},
	}

	callSite := core.CallSite{
		Target: "app.run",
		Arguments: []core.Argument{
			{Value: "debug=False", Position: 0},
		},
	}

	assert.False(t, executor.matchesArguments(&callSite), "Expected NOT to match debug=False when expecting True")
}

// TestMatchesArguments_StringMatch tests string value matching.
func TestMatchesArguments_StringMatch(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"app.run"},
			KeywordArgs: map[string]ArgumentConstraint{
				"host": {Value: "0.0.0.0", Wildcard: false},
			},
		},
		CallGraph: &core.CallGraph{},
	}

	callSite := core.CallSite{
		Target: "app.run",
		Arguments: []core.Argument{
			{Value: "host=\"0.0.0.0\"", Position: 0},
		},
	}

	assert.True(t, executor.matchesArguments(&callSite), "Expected to match host=\"0.0.0.0\"")
}

// TestMatchesArguments_StringNoMatch tests string mismatch.
func TestMatchesArguments_StringNoMatch(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"app.run"},
			KeywordArgs: map[string]ArgumentConstraint{
				"host": {Value: "0.0.0.0", Wildcard: false},
			},
		},
		CallGraph: &core.CallGraph{},
	}

	callSite := core.CallSite{
		Target: "app.run",
		Arguments: []core.Argument{
			{Value: "host=\"127.0.0.1\"", Position: 0},
		},
	}

	assert.False(t, executor.matchesArguments(&callSite), "Expected NOT to match host=\"127.0.0.1\" when expecting \"0.0.0.0\"")
}

// TestMatchesArguments_MultipleConstraints tests AND logic with multiple constraints.
func TestMatchesArguments_MultipleConstraints(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"app.run"},
			KeywordArgs: map[string]ArgumentConstraint{
				"host":  {Value: "0.0.0.0", Wildcard: false},
				"debug": {Value: true, Wildcard: false},
			},
		},
		CallGraph: &core.CallGraph{},
	}

	callSite := core.CallSite{
		Target: "app.run",
		Arguments: []core.Argument{
			{Value: "host=\"0.0.0.0\"", Position: 0},
			{Value: "debug=True", Position: 1},
		},
	}

	assert.True(t, executor.matchesArguments(&callSite), "Expected to match when all constraints satisfied")
}

// TestMatchesArguments_PartialMatch tests that all constraints must match.
func TestMatchesArguments_PartialMatch(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"app.run"},
			KeywordArgs: map[string]ArgumentConstraint{
				"host":  {Value: "0.0.0.0", Wildcard: false},
				"debug": {Value: true, Wildcard: false},
			},
		},
		CallGraph: &core.CallGraph{},
	}

	callSite := core.CallSite{
		Target: "app.run",
		Arguments: []core.Argument{
			{Value: "host=\"0.0.0.0\"", Position: 0}, // Matches
			{Value: "debug=False", Position: 1},      // Doesn't match
		},
	}

	assert.False(t, executor.matchesArguments(&callSite), "Expected NOT to match when one constraint fails (AND logic)")
}

// TestMatchesArguments_MissingArgument tests handling of missing required argument.
func TestMatchesArguments_MissingArgument(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"app.run"},
			KeywordArgs: map[string]ArgumentConstraint{
				"debug": {Value: true, Wildcard: false},
			},
		},
		CallGraph: &core.CallGraph{},
	}

	callSite := core.CallSite{
		Target: "app.run",
		Arguments: []core.Argument{
			{Value: "host=\"0.0.0.0\"", Position: 0}, // Different argument
		},
	}

	assert.False(t, executor.matchesArguments(&callSite), "Expected NOT to match when required argument missing")
}

// TestCleanValue tests quote removal and whitespace trimming.
func TestCleanValue(t *testing.T) {
	executor := &CallMatcherExecutor{IR: &CallMatcherIR{}}

	tests := []struct {
		input    string
		expected string
	}{
		{"\"0.0.0.0\"", "0.0.0.0"},
		{"'localhost'", "localhost"},
		{"  True  ", "True"},
		{"\"quoted string\"", "quoted string"},
		{"'single quoted'", "single quoted"},
		{"no quotes", "no quotes"},
		{"  spaces  ", "spaces"},
	}

	for _, test := range tests {
		result := executor.cleanValue(test.input)
		assert.Equal(t, test.expected, result, "cleanValue(%q) should equal %q", test.input, test.expected)
	}
}

// TestMatchesBoolean tests boolean value matching.
func TestMatchesBoolean(t *testing.T) {
	executor := &CallMatcherExecutor{IR: &CallMatcherIR{}}

	tests := []struct {
		actual   string
		expected bool
		should   bool
	}{
		{"True", true, true},
		{"true", true, true},
		{"TRUE", true, true},
		{"1", true, true},
		{"False", false, true},
		{"false", false, true},
		{"FALSE", false, true},
		{"0", false, true},
		{"True", false, false},
		{"False", true, false},
	}

	for _, test := range tests {
		result := executor.matchesBoolean(test.actual, test.expected)
		assert.Equal(t, test.should, result, "matchesBoolean(%q, %v) should equal %v", test.actual, test.expected, test.should)
	}
}

// TestMatchesNumber tests numeric value matching.
func TestMatchesNumber(t *testing.T) {
	executor := &CallMatcherExecutor{IR: &CallMatcherIR{}}

	tests := []struct {
		actual   string
		expected float64
		should   bool
	}{
		{"777", 777.0, true},
		{"0o777", 511.0, true}, // Octal 777 = decimal 511
		{"0777", 511.0, true},  // Octal
		{"0xFF", 255.0, true},  // Hex
		{"3.14", 3.14, true},
		{"42", 42.0, true},
		{"777", 778.0, false},
		{"not a number", 777.0, false},
	}

	for _, test := range tests {
		result := executor.matchesNumber(test.actual, test.expected)
		assert.Equal(t, test.should, result, "matchesNumber(%q, %v) should equal %v", test.actual, test.expected, test.should)
	}
}

// TestMatchesCallSite_Integration tests end-to-end matching with arguments.
func TestMatchesCallSite_Integration(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"app.run"},
			KeywordArgs: map[string]ArgumentConstraint{
				"debug": {Value: true, Wildcard: false},
			},
		},
		CallGraph: &core.CallGraph{},
	}

	// Should match: correct function AND correct argument
	matchCallSite := core.CallSite{
		Target: "app.run",
		Arguments: []core.Argument{
			{Value: "debug=True", Position: 0},
		},
	}

	assert.True(t, executor.matchesCallSite(&matchCallSite), "Expected to match app.run(debug=True)")

	// Should NOT match: correct function but wrong argument
	noMatchCallSite := core.CallSite{
		Target: "app.run",
		Arguments: []core.Argument{
			{Value: "debug=False", Position: 0},
		},
	}

	assert.False(t, executor.matchesCallSite(&noMatchCallSite), "Expected NOT to match app.run(debug=False)")

	// Should NOT match: wrong function (even with matching argument)
	wrongFunctionCallSite := core.CallSite{
		Target: "other.run",
		Arguments: []core.Argument{
			{Value: "debug=True", Position: 0},
		},
	}

	assert.False(t, executor.matchesCallSite(&wrongFunctionCallSite), "Expected NOT to match other.run() even with correct argument")
}

// TestMatchesPositionalArguments_NoConstraints tests backward compatibility with no positional constraints.
func TestMatchesPositionalArguments_NoConstraints(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"socket.bind"},
			// No PositionalArgs = should always match
		},
	}

	args := []core.Argument{
		{Value: "\"0.0.0.0\"", Position: 0},
		{Value: "8080", Position: 1},
	}

	assert.True(t, executor.matchesPositionalArguments(args), "Expected to match when no positional constraints present")
}

// TestMatchesPositionalArguments_SingleArg tests matching single positional argument.
func TestMatchesPositionalArguments_SingleArg(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"socket.bind"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0": {Value: "0.0.0.0", Wildcard: false},
			},
		},
	}

	args := []core.Argument{
		{Value: "\"0.0.0.0\"", Position: 0},
	}

	assert.True(t, executor.matchesPositionalArguments(args), "Expected to match position 0 with \"0.0.0.0\"")
}

// TestMatchesPositionalArguments_SingleArgNoMatch tests single positional argument mismatch.
func TestMatchesPositionalArguments_SingleArgNoMatch(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"socket.bind"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0": {Value: "0.0.0.0", Wildcard: false},
			},
		},
	}

	args := []core.Argument{
		{Value: "\"127.0.0.1\"", Position: 0},
	}

	assert.False(t, executor.matchesPositionalArguments(args), "Expected NOT to match position 0 with \"127.0.0.1\"")
}

// TestMatchesPositionalArguments_MultipleArgs tests matching multiple positional arguments.
func TestMatchesPositionalArguments_MultipleArgs(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"chmod"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0": {Value: "/tmp/file", Wildcard: false},
				"1": {Value: float64(511), Wildcard: false}, // 0o777 = 511
			},
		},
	}

	args := []core.Argument{
		{Value: "/tmp/file", Position: 0},
		{Value: "0o777", Position: 1},
	}

	assert.True(t, executor.matchesPositionalArguments(args), "Expected to match both positional arguments")
}

// TestMatchesPositionalArguments_PartialMatch tests that all positional constraints must match.
func TestMatchesPositionalArguments_PartialMatch(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"chmod"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0": {Value: "/tmp/file", Wildcard: false},
				"1": {Value: float64(511), Wildcard: false},
			},
		},
	}

	args := []core.Argument{
		{Value: "/tmp/file", Position: 0}, // Matches
		{Value: "0o755", Position: 1},     // Doesn't match (755 octal = 493 decimal)
	}

	assert.False(t, executor.matchesPositionalArguments(args), "Expected NOT to match when one positional constraint fails")
}

// TestMatchesPositionalArguments_OutOfBounds tests position doesn't exist in arguments.
func TestMatchesPositionalArguments_OutOfBounds(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"open"},
			PositionalArgs: map[string]ArgumentConstraint{
				"2": {Value: "utf-8", Wildcard: false}, // Position 2 doesn't exist
			},
		},
	}

	args := []core.Argument{
		{Value: "file.txt", Position: 0},
		{Value: "w", Position: 1},
		// No position 2
	}

	assert.False(t, executor.matchesPositionalArguments(args), "Expected NOT to match when position doesn't exist")
}

// TestMatchesPositionalArguments_InvalidPosition tests invalid position string.
func TestMatchesPositionalArguments_InvalidPosition(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"test"},
			PositionalArgs: map[string]ArgumentConstraint{
				"invalid": {Value: "test", Wildcard: false},
			},
		},
	}

	args := []core.Argument{
		{Value: "test", Position: 0},
	}

	assert.False(t, executor.matchesPositionalArguments(args), "Expected NOT to match with invalid position string")
}

// TestMatchesPositionalArguments_WithQuotes tests quote stripping.
func TestMatchesPositionalArguments_WithQuotes(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"open"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0": {Value: "file.txt", Wildcard: false},
			},
		},
	}

	// Actual call has quotes, constraint doesn't
	args := []core.Argument{
		{Value: "\"file.txt\"", Position: 0},
	}

	assert.True(t, executor.matchesPositionalArguments(args), "Expected to match after stripping quotes")
}

// TestMatchesPositionalArguments_NumberTypes tests various number types.
func TestMatchesPositionalArguments_NumberTypes(t *testing.T) {
	tests := []struct {
		name        string
		constraint  float64
		actualValue string
		shouldMatch bool
	}{
		{"decimal", 777.0, "777", true},
		{"octal", 511.0, "0o777", true},
		{"hex", 255.0, "0xFF", true},
		{"float", 3.14, "3.14", true},
		{"mismatch", 777.0, "778", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := &CallMatcherExecutor{
				IR: &CallMatcherIR{
					Patterns: []string{"test"},
					PositionalArgs: map[string]ArgumentConstraint{
						"0": {Value: test.constraint, Wildcard: false},
					},
				},
			}

			args := []core.Argument{
				{Value: test.actualValue, Position: 0},
			}

			result := executor.matchesPositionalArguments(args)
			assert.Equal(t, test.shouldMatch, result, "Expected matchesPositionalArguments to be %v for %s", test.shouldMatch, test.name)
		})
	}
}

// TestMatchesArguments_CombinedPositionalAndKeyword tests matching both types together.
func TestMatchesArguments_CombinedPositionalAndKeyword(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"app.run"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0": {Value: "localhost", Wildcard: false},
			},
			KeywordArgs: map[string]ArgumentConstraint{
				"debug": {Value: true, Wildcard: false},
				"port":  {Value: float64(5000), Wildcard: false},
			},
		},
	}

	callSite := core.CallSite{
		Target: "app.run",
		Arguments: []core.Argument{
			{Value: "localhost", Position: 0},
			{Value: "debug=True", Position: 1},
			{Value: "port=5000", Position: 2},
		},
	}

	assert.True(t, executor.matchesArguments(&callSite), "Expected to match when both positional and keyword constraints satisfied")
}

// TestMatchesArguments_CombinedPartialMatch tests partial match with combined constraints.
func TestMatchesArguments_CombinedPartialMatch(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"app.run"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0": {Value: "localhost", Wildcard: false},
			},
			KeywordArgs: map[string]ArgumentConstraint{
				"debug": {Value: true, Wildcard: false},
			},
		},
	}

	callSite := core.CallSite{
		Target: "app.run",
		Arguments: []core.Argument{
			{Value: "localhost", Position: 0},   // Matches positional
			{Value: "debug=False", Position: 1}, // Doesn't match keyword
		},
	}

	assert.False(t, executor.matchesArguments(&callSite), "Expected NOT to match when keyword constraint fails")
}

// TestMatchesArguments_CombinedPositionalFails tests positional failure with combined constraints.
func TestMatchesArguments_CombinedPositionalFails(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"app.run"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0": {Value: "localhost", Wildcard: false},
			},
			KeywordArgs: map[string]ArgumentConstraint{
				"debug": {Value: true, Wildcard: false},
			},
		},
	}

	callSite := core.CallSite{
		Target: "app.run",
		Arguments: []core.Argument{
			{Value: "0.0.0.0", Position: 0},    // Doesn't match positional
			{Value: "debug=True", Position: 1}, // Matches keyword
		},
	}

	assert.False(t, executor.matchesArguments(&callSite), "Expected NOT to match when positional constraint fails")
}

// TestPositionalArguments_RealWorldPatterns tests real-world security patterns.
func TestPositionalArguments_RealWorldPatterns(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		constraint  map[string]ArgumentConstraint
		callSite    core.CallSite
		shouldMatch bool
	}{
		{
			name:    "socket.bind with 0.0.0.0",
			pattern: "socket.bind",
			constraint: map[string]ArgumentConstraint{
				"0": {Value: "0.0.0.0", Wildcard: false},
			},
			callSite: core.CallSite{
				Target: "socket.bind",
				Arguments: []core.Argument{
					{Value: "\"0.0.0.0\"", Position: 0},
				},
			},
			shouldMatch: true,
		},
		{
			name:    "chmod with dangerous permissions",
			pattern: "chmod",
			constraint: map[string]ArgumentConstraint{
				"1": {Value: float64(511), Wildcard: false}, // 0o777
			},
			callSite: core.CallSite{
				Target: "chmod",
				Arguments: []core.Argument{
					{Value: "/tmp/file", Position: 0},
					{Value: "0o777", Position: 1},
				},
			},
			shouldMatch: true,
		},
		{
			name:    "open with write mode",
			pattern: "open",
			constraint: map[string]ArgumentConstraint{
				"1": {Value: "w", Wildcard: false},
			},
			callSite: core.CallSite{
				Target: "open",
				Arguments: []core.Argument{
					{Value: "file.txt", Position: 0},
					{Value: "\"w\"", Position: 1},
				},
			},
			shouldMatch: true,
		},
		{
			name:    "open with read mode - no match",
			pattern: "open",
			constraint: map[string]ArgumentConstraint{
				"1": {Value: "w", Wildcard: false},
			},
			callSite: core.CallSite{
				Target: "open",
				Arguments: []core.Argument{
					{Value: "file.txt", Position: 0},
					{Value: "\"r\"", Position: 1},
				},
			},
			shouldMatch: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := &CallMatcherExecutor{
				IR: &CallMatcherIR{
					Patterns:       []string{test.pattern},
					PositionalArgs: test.constraint,
				},
			}

			result := executor.matchesCallSite(&test.callSite)
			assert.Equal(t, test.shouldMatch, result, "Expected matchesCallSite to be %v for %s", test.shouldMatch, test.name)
		})
	}
}

// TestCallMatcherExecutor_PositionalIntegration tests full end-to-end with Execute().
func TestCallMatcherExecutor_PositionalIntegration(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["test.main"] = []core.CallSite{
		{
			Target: "socket.bind",
			Arguments: []core.Argument{
				{Value: "\"0.0.0.0\"", Position: 0},
				{Value: "8080", Position: 1},
			},
			Location: core.Location{File: "server.py", Line: 10},
		},
		{
			Target: "socket.bind",
			Arguments: []core.Argument{
				{Value: "\"127.0.0.1\"", Position: 0},
				{Value: "8080", Position: 1},
			},
			Location: core.Location{File: "server.py", Line: 20},
		},
		{
			Target: "chmod",
			Arguments: []core.Argument{
				{Value: "/tmp/file", Position: 0},
				{Value: "0o777", Position: 1},
			},
			Location: core.Location{File: "file_ops.py", Line: 30},
		},
	}

	t.Run("match 0.0.0.0 binds only", func(t *testing.T) {
		ir := &CallMatcherIR{
			Patterns: []string{"socket.bind"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0": {Value: "0.0.0.0", Wildcard: false},
			},
		}

		executor := NewCallMatcherExecutor(ir, cg)
		matches := executor.Execute()

		assert.Len(t, matches, 1, "Expected to match only 0.0.0.0 bind")
		assert.Equal(t, "socket.bind", matches[0].Target)
		assert.Equal(t, 10, matches[0].Location.Line)
	})

	t.Run("match dangerous chmod", func(t *testing.T) {
		ir := &CallMatcherIR{
			Patterns: []string{"chmod"},
			PositionalArgs: map[string]ArgumentConstraint{
				"1": {Value: float64(511), Wildcard: false}, // 0o777
			},
		}

		executor := NewCallMatcherExecutor(ir, cg)
		matches := executor.Execute()

		assert.Len(t, matches, 1, "Expected to match chmod with 0o777")
		assert.Equal(t, "chmod", matches[0].Target)
		assert.Equal(t, 30, matches[0].Location.Line)
	})

	t.Run("no matches for strict constraint", func(t *testing.T) {
		ir := &CallMatcherIR{
			Patterns: []string{"socket.bind"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0": {Value: "192.168.1.1", Wildcard: false},
			},
		}

		executor := NewCallMatcherExecutor(ir, cg)
		matches := executor.Execute()

		assert.Len(t, matches, 0, "Expected no matches for non-existent IP")
	})
}

// TestPositionalArguments_BackwardCompatibility tests that IR without PositionalArgs still works.
func TestPositionalArguments_BackwardCompatibility(t *testing.T) {
	// Old IR without PositionalArgs field
	ir := CallMatcherIR{
		Type:      "call_matcher",
		Patterns:  []string{"eval", "exec"},
		Wildcard:  false,
		MatchMode: "any",
	}

	// Should work fine (PositionalArgs is nil/empty)
	assert.Nil(t, ir.PositionalArgs, "Expected nil PositionalArgs for backward compatibility")

	// Verify it can still be used with executor
	cg := core.NewCallGraph()
	cg.CallSites["test.main"] = []core.CallSite{
		{
			Target: "eval",
			Arguments: []core.Argument{
				{Value: "code", Position: 0},
			},
			Location: core.Location{File: "test.py", Line: 10},
		},
	}

	executor := NewCallMatcherExecutor(&ir, cg)
	matches := executor.Execute()

	assert.Len(t, matches, 1, "Old IR should still work")
	assert.Equal(t, "eval", matches[0].Target, "Old IR should match correctly")
}

// ========== OR Logic Tests ==========

// TestMatchesArgumentValue_ORLogic_Strings tests OR logic with string values.
func TestMatchesArgumentValue_ORLogic_Strings(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"open"},
			PositionalArgs: map[string]ArgumentConstraint{
				"1": {
					Value:    []any{"w", "a", "w+", "a+"},
					Wildcard: false,
				},
			},
		},
	}

	tests := []struct {
		name        string
		actualValue string
		shouldMatch bool
	}{
		{"Match w", "\"w\"", true},
		{"Match a", "\"a\"", true},
		{"Match w+", "\"w+\"", true},
		{"Match a+", "\"a+\"", true},
		{"No match r", "\"r\"", false},
		{"No match rb", "\"rb\"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &core.CallSite{
				Target: "open",
				Arguments: []core.Argument{
					{Value: "\"/tmp/file\"", Position: 0},
					{Value: tt.actualValue, Position: 1},
				},
			}
			result := executor.matchesArguments(cs)
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}

// TestMatchesArgumentValue_ORLogic_Mixed tests OR logic with keyword arguments.
func TestMatchesArgumentValue_ORLogic_Mixed(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"config.set"},
			KeywordArgs: map[string]ArgumentConstraint{
				"mode": {
					Value:    []any{"debug", "development", "test"},
					Wildcard: false,
				},
			},
		},
	}

	// Should match any of the three values
	cs1 := &core.CallSite{
		Target: "config.set",
		Arguments: []core.Argument{
			{Value: "mode=debug", Position: 0},
		},
	}
	assert.True(t, executor.matchesArguments(cs1))

	cs2 := &core.CallSite{
		Target: "config.set",
		Arguments: []core.Argument{
			{Value: "mode=development", Position: 0},
		},
	}
	assert.True(t, executor.matchesArguments(cs2))

	// Should not match production
	cs3 := &core.CallSite{
		Target: "config.set",
		Arguments: []core.Argument{
			{Value: "mode=production", Position: 0},
		},
	}
	assert.False(t, executor.matchesArguments(cs3))
}

// TestMatchesArgumentValue_ORLogic_Numbers tests OR logic with numeric values.
func TestMatchesArgumentValue_ORLogic_Numbers(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"chmod"},
			PositionalArgs: map[string]ArgumentConstraint{
				"1": {
					Value:    []any{float64(511), float64(493), float64(448)}, // 0o777, 0o755, 0o700
					Wildcard: false,
				},
			},
		},
	}

	tests := []struct {
		name        string
		actualValue string
		shouldMatch bool
	}{
		{"Match 0o777", "0o777", true},
		{"Match 0o755", "0o755", true},
		{"Match 0o700", "0o700", true},
		{"No match 0o644", "0o644", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &core.CallSite{
				Target: "chmod",
				Arguments: []core.Argument{
					{Value: "/tmp/file", Position: 0},
					{Value: tt.actualValue, Position: 1},
				},
			}
			result := executor.matchesArguments(cs)
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}

// ========== Wildcard Matching Tests ==========

// TestMatchesWildcard_BasicPatterns tests basic wildcard patterns.
func TestMatchesWildcard_BasicPatterns(t *testing.T) {
	executor := &CallMatcherExecutor{}

	tests := []struct {
		name        string
		actual      string
		pattern     string
		shouldMatch bool
	}{
		// Star wildcard
		{"Star prefix", "0.0.0.0", "0.0.*", true},
		{"Star suffix", "0.0.0.0", "*.0.0", true},
		{"Star middle", "test-file.txt", "test-*", true},
		{"Star all", "anything", "*", true},
		{"Star no match", "192.168.1.1", "10.*", false},

		// Question wildcard
		{"Question single", "abc", "a?c", true},
		{"Question multiple", "test", "t??t", true},
		{"Question no match", "abc", "a?d", false},

		// Combined wildcards
		{"Star and question", "test-01.txt", "test-??.txt", true},
		{"Complex pattern", "192.168.1.100", "192.*.1.*", true},

		// Exact match (no wildcards)
		{"Exact match", "exact", "exact", true},
		{"Exact no match", "exact", "different", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.matchesWildcard(tt.actual, tt.pattern)
			assert.Equal(t, tt.shouldMatch, result,
				"Pattern: %s, Actual: %s", tt.pattern, tt.actual)
		})
	}
}

// TestMatchesWildcard_OctalPatterns tests wildcard matching with octal values.
func TestMatchesWildcard_OctalPatterns(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"chmod"},
			PositionalArgs: map[string]ArgumentConstraint{
				"1": {
					Value:    "0o7*", // Match 0o7XX (world-writable)
					Wildcard: true,
				},
			},
		},
	}

	tests := []struct {
		name        string
		actualValue string
		shouldMatch bool
	}{
		{"0o777", "0o777", true},
		{"0o755", "0o755", true},
		{"0o700", "0o700", true},
		{"0o644", "0o644", false}, // Doesn't start with 7
		{"0o600", "0o600", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &core.CallSite{
				Target: "chmod",
				Arguments: []core.Argument{
					{Value: "\"/tmp/file\"", Position: 0},
					{Value: tt.actualValue, Position: 1},
				},
			}
			result := executor.matchesArguments(cs)
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}

// TestMatchesWildcard_IPAddressPatterns tests wildcard matching with IP addresses.
func TestMatchesWildcard_IPAddressPatterns(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"socket.bind"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0": {
					Value:    "0.0.*", // Match 0.0.X.X
					Wildcard: true,
				},
			},
		},
	}

	tests := []struct {
		name        string
		ip          string
		shouldMatch bool
	}{
		{"0.0.0.0", "\"0.0.0.0\"", true},
		{"0.0.0.1", "\"0.0.0.1\"", true},
		{"0.0.255.255", "\"0.0.255.255\"", true},
		{"127.0.0.1", "\"127.0.0.1\"", false},
		{"192.168.1.1", "\"192.168.1.1\"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &core.CallSite{
				Target: "socket.bind",
				Arguments: []core.Argument{
					{Value: tt.ip, Position: 0},
				},
			}
			result := executor.matchesArguments(cs)
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}

// TestMatchesArgumentValue_ORLogicWithWildcards tests combined OR + wildcard.
func TestMatchesArgumentValue_ORLogicWithWildcards(t *testing.T) {
	executor := &CallMatcherExecutor{
		IR: &CallMatcherIR{
			Patterns: []string{"yaml.load"},
			KeywordArgs: map[string]ArgumentConstraint{
				"Loader": {
					Value:    []any{"*Loader", "*UnsafeLoader"},
					Wildcard: true,
				},
			},
		},
	}

	tests := []struct {
		name        string
		loader      string
		shouldMatch bool
	}{
		{"FullLoader", "Loader=FullLoader", true},
		{"UnsafeLoader", "Loader=UnsafeLoader", true},
		{"yaml.UnsafeLoader", "Loader=yaml.UnsafeLoader", true},
		{"SafeLoader", "Loader=SafeLoader", true},
		{"BaseLoader", "Loader=BaseLoader", true},
		{"None", "Loader=None", false}, // Doesn't end with Loader or UnsafeLoader
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &core.CallSite{
				Target: "yaml.load",
				Arguments: []core.Argument{
					{Value: "data", Position: 0},
					{Value: tt.loader, Position: 1},
				},
			}
			result := executor.matchesArguments(cs)
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}

// TestMatchesArgumentValue_EdgeCases tests edge cases for value matching.
func TestMatchesArgumentValue_EdgeCases(t *testing.T) {
	executor := &CallMatcherExecutor{}

	tests := []struct {
		name        string
		actual      string
		expected    any
		wildcard    bool
		shouldMatch bool
	}{
		// Empty values
		{"Empty actual", "", "value", false, false},
		{"Empty pattern", "value", "", false, false},
		{"Both empty", "", "", false, true},

		// Wildcard edge cases
		{"Only star", "anything", "*", true, true},
		{"Multiple stars", "test", "***", true, true},
		{"Star at end", "prefix", "prefix*", true, true},
		{"Star at start", "suffix", "*suffix", true, true},

		// Question mark edge cases
		{"Single question", "a", "?", true, true},
		{"Question too many", "ab", "???", true, false},
		{"Question too few", "abc", "?", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.matchesSingleValue(tt.actual, tt.expected, tt.wildcard)
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}

// TestWildcardMatch_ComplexPatterns tests complex wildcard scenarios.
func TestWildcardMatch_ComplexPatterns(t *testing.T) {
	executor := &CallMatcherExecutor{}

	tests := []struct {
		name        string
		str         string
		pattern     string
		shouldMatch bool
	}{
		// Multiple stars
		{"Two stars", "abcdef", "*c*f", true},
		{"Three stars", "test-file-123.txt", "*-*-*.*", true},

		// Mixed wildcards
		{"Star and questions", "test-01.txt", "test-??.*", true},
		{"Question and star", "file123.txt", "file???.txt", true},

		// Edge patterns
		{"Empty pattern", "test", "", false},
		{"Pattern longer", "ab", "abc", false},
		{"String longer", "abc", "ab", false},

		// Special cases
		{"All questions", "test", "????", true},
		{"All stars", "anything", "***", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.wildcardMatch(tt.str, tt.pattern)
			assert.Equal(t, tt.shouldMatch, result)
		})
	}
}

// Benchmark wildcard matching.
func BenchmarkWildcardMatch_Simple(b *testing.B) {
	executor := &CallMatcherExecutor{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.wildcardMatch("192.168.1.100", "192.*.1.*")
	}
}

func BenchmarkWildcardMatch_Complex(b *testing.B) {
	executor := &CallMatcherExecutor{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.wildcardMatch("test-file-12345.txt", "test-*-?????.txt")
	}
}

// TestParseTupleIndex tests parsing of tuple indexing syntax.
func TestParseTupleIndex(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedPos   int
		expectedIdx   int
		expectedIsTup bool
	}{
		{
			name:          "simple position",
			input:         "0",
			expectedPos:   0,
			expectedIdx:   0,
			expectedIsTup: false,
		},
		{
			name:          "simple position 5",
			input:         "5",
			expectedPos:   5,
			expectedIdx:   0,
			expectedIsTup: false,
		},
		{
			name:          "tuple index first element",
			input:         "0[0]",
			expectedPos:   0,
			expectedIdx:   0,
			expectedIsTup: true,
		},
		{
			name:          "tuple index second element",
			input:         "0[1]",
			expectedPos:   0,
			expectedIdx:   1,
			expectedIsTup: true,
		},
		{
			name:          "tuple index position 3 element 2",
			input:         "3[2]",
			expectedPos:   3,
			expectedIdx:   2,
			expectedIsTup: true,
		},
		{
			name:          "invalid format - no closing bracket",
			input:         "0[1",
			expectedPos:   0,
			expectedIdx:   0,
			expectedIsTup: false,
		},
		{
			name:          "invalid format - non-numeric position",
			input:         "x[0]",
			expectedPos:   0,
			expectedIdx:   0,
			expectedIsTup: false,
		},
		{
			name:          "invalid format - non-numeric index",
			input:         "0[x]",
			expectedPos:   0,
			expectedIdx:   0,
			expectedIsTup: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, idx, isTup, valid := parseTupleIndex(tt.input)
			if tt.expectedIsTup || tt.expectedPos > 0 {
				// For valid cases, should be valid=true
				assert.True(t, valid, "should be valid")
			}
			assert.Equal(t, tt.expectedPos, pos, "position mismatch")
			assert.Equal(t, tt.expectedIdx, idx, "index mismatch")
			assert.Equal(t, tt.expectedIsTup, isTup, "isTupleIndex mismatch")
		})
	}
}

// TestExtractTupleElement tests extraction of elements from tuple strings.
func TestExtractTupleElement(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		index      int
		expected   string
		expectedOk bool
	}{
		{
			name:       "extract first element from string tuple",
			input:      `("0.0.0.0", 8080)`,
			index:      0,
			expected:   "0.0.0.0",
			expectedOk: true,
		},
		{
			name:       "extract second element from string tuple",
			input:      `("0.0.0.0", 8080)`,
			index:      1,
			expected:   "8080",
			expectedOk: true,
		},
		{
			name:       "extract from single-quoted string",
			input:      `('0.0.0.0', 8080)`,
			index:      0,
			expected:   "0.0.0.0",
			expectedOk: true,
		},
		{
			name:       "extract from tuple with spaces",
			input:      `( "0.0.0.0" , 8080 )`,
			index:      0,
			expected:   "0.0.0.0",
			expectedOk: true,
		},
		{
			name:       "extract from three-element tuple",
			input:      `("a", "b", "c")`,
			index:      1,
			expected:   "b",
			expectedOk: true,
		},
		{
			name:       "extract from list syntax",
			input:      `["host", 8080]`,
			index:      0,
			expected:   "host",
			expectedOk: true,
		},
		{
			name:       "index out of bounds",
			input:      `("0.0.0.0", 8080)`,
			index:      5,
			expected:   "",
			expectedOk: false,
		},
		{
			name:       "not a tuple - return as is for index 0",
			input:      `"plain_string"`,
			index:      0,
			expected:   "plain_string",
			expectedOk: true,
		},
		{
			name:       "not a tuple - empty for index > 0",
			input:      `"plain_string"`,
			index:      1,
			expected:   "",
			expectedOk: false,
		},
		{
			name:       "empty tuple",
			input:      `()`,
			index:      0,
			expected:   "",
			expectedOk: false,
		},
		{
			name:       "tuple with numeric values",
			input:      `(80, 443, 8080)`,
			index:      1,
			expected:   "443",
			expectedOk: true,
		},
		{
			name:       "tuple with empty string - should extract empty string successfully",
			input:      `("", 8080)`,
			index:      0,
			expected:   "",
			expectedOk: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, ok := extractTupleElement(tt.input, tt.index)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.expectedOk, ok)
		})
	}
}

// TestMatchesPositionalArguments_TupleIndexing tests tuple indexing in positional arguments.
func TestMatchesPositionalArguments_TupleIndexing(t *testing.T) {
	tests := []struct {
		name        string
		args        []core.Argument
		constraints map[string]ArgumentConstraint
		shouldMatch bool
		description string
	}{
		{
			name: "tuple first element matches",
			args: []core.Argument{
				{Value: `("0.0.0.0", 8080)`, Position: 0},
			},
			constraints: map[string]ArgumentConstraint{
				"0[0]": {Value: "0.0.0.0", Wildcard: false},
			},
			shouldMatch: true,
			description: "socket.bind(('0.0.0.0', 8080)) with match_position={'0[0]': '0.0.0.0'}",
		},
		{
			name: "tuple second element matches",
			args: []core.Argument{
				{Value: `("0.0.0.0", 8080)`, Position: 0},
			},
			constraints: map[string]ArgumentConstraint{
				"0[1]": {Value: "8080", Wildcard: false},
			},
			shouldMatch: true,
			description: "socket.bind(('0.0.0.0', 8080)) with match_position={'0[1]': 8080}",
		},
		{
			name: "both tuple elements match",
			args: []core.Argument{
				{Value: `("0.0.0.0", 8080)`, Position: 0},
			},
			constraints: map[string]ArgumentConstraint{
				"0[0]": {Value: "0.0.0.0", Wildcard: false},
				"0[1]": {Value: "8080", Wildcard: false},
			},
			shouldMatch: true,
			description: "Both elements match",
		},
		{
			name: "tuple first element doesn't match",
			args: []core.Argument{
				{Value: `("127.0.0.1", 8080)`, Position: 0},
			},
			constraints: map[string]ArgumentConstraint{
				"0[0]": {Value: "0.0.0.0", Wildcard: false},
			},
			shouldMatch: false,
			description: "Wrong IP address",
		},
		{
			name: "tuple index out of bounds",
			args: []core.Argument{
				{Value: `("0.0.0.0", 8080)`, Position: 0},
			},
			constraints: map[string]ArgumentConstraint{
				"0[5]": {Value: "something", Wildcard: false},
			},
			shouldMatch: false,
			description: "Index 5 doesn't exist in 2-element tuple",
		},
		{
			name: "mixed tuple and simple positional",
			args: []core.Argument{
				{Value: `("0.0.0.0", 8080)`, Position: 0},
				{Value: "timeout", Position: 1},
			},
			constraints: map[string]ArgumentConstraint{
				"0[0]": {Value: "0.0.0.0", Wildcard: false},
				"1":    {Value: "timeout", Wildcard: false},
			},
			shouldMatch: true,
			description: "Tuple indexing and simple positional together",
		},
		{
			name: "backward compatibility - simple positional still works",
			args: []core.Argument{
				{Value: "w", Position: 0},
				{Value: "file.txt", Position: 1},
			},
			constraints: map[string]ArgumentConstraint{
				"0": {Value: "w", Wildcard: false},
			},
			shouldMatch: true,
			description: "Ensure backward compatibility with simple positional",
		},
		{
			name: "list syntax extraction",
			args: []core.Argument{
				{Value: `["0.0.0.0", 8080]`, Position: 0},
			},
			constraints: map[string]ArgumentConstraint{
				"0[0]": {Value: "0.0.0.0", Wildcard: false},
			},
			shouldMatch: true,
			description: "List brackets should work like tuples",
		},
		{
			name: "single quoted strings in tuple",
			args: []core.Argument{
				{Value: `('0.0.0.0', 8080)`, Position: 0},
			},
			constraints: map[string]ArgumentConstraint{
				"0[0]": {Value: "0.0.0.0", Wildcard: false},
			},
			shouldMatch: true,
			description: "Handle single-quoted strings",
		},
		{
			name: "tuple with OR logic",
			args: []core.Argument{
				{Value: `("0.0.0.0", 8080)`, Position: 0},
			},
			constraints: map[string]ArgumentConstraint{
				"0[0]": {
					Value:    []any{"0.0.0.0", "127.0.0.1", "localhost"},
					Wildcard: false,
				},
			},
			shouldMatch: true,
			description: "Tuple indexing with OR logic",
		},
		{
			name: "tuple with wildcard",
			args: []core.Argument{
				{Value: `("192.168.1.100", 8080)`, Position: 0},
			},
			constraints: map[string]ArgumentConstraint{
				"0[0]": {Value: "192.168.*", Wildcard: true},
			},
			shouldMatch: true,
			description: "Tuple indexing with wildcard matching",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := &CallMatcherExecutor{
				IR: &CallMatcherIR{
					PositionalArgs: tt.constraints,
				},
			}

			result := executor.matchesPositionalArguments(tt.args)
			assert.Equal(t, tt.shouldMatch, result, tt.description)
		})
	}
}

// TestSocketBindDetection tests end-to-end socket.bind detection with tuples.
func TestSocketBindDetection(t *testing.T) {
	cg := core.NewCallGraph()

	// Simulate various socket.bind calls
	cg.CallSites["test.main"] = []core.CallSite{
		{
			Target: "socket.bind",
			Arguments: []core.Argument{
				{Value: `("0.0.0.0", 8080)`, Position: 0},
			},
			Location: core.Location{File: "test.py", Line: 10},
		},
		{
			Target: "socket.bind",
			Arguments: []core.Argument{
				{Value: `("127.0.0.1", 8080)`, Position: 0},
			},
			Location: core.Location{File: "test.py", Line: 15},
		},
		{
			Target: "socket.bind",
			Arguments: []core.Argument{
				{Value: `("192.168.1.5", 8080)`, Position: 0},
			},
			Location: core.Location{File: "test.py", Line: 20},
		},
	}

	t.Run("detect 0.0.0.0 binding", func(t *testing.T) {
		ir := &CallMatcherIR{
			Patterns: []string{"socket.bind"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0[0]": {Value: "0.0.0.0", Wildcard: false},
			},
		}

		executor := NewCallMatcherExecutor(ir, cg)
		matches := executor.Execute()

		assert.Len(t, matches, 1, "Should match only 0.0.0.0 binding")
		assert.Equal(t, 10, matches[0].Location.Line)
	})

	t.Run("detect private network binding with wildcard", func(t *testing.T) {
		ir := &CallMatcherIR{
			Patterns: []string{"socket.bind"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0[0]": {Value: "192.168.*", Wildcard: true},
			},
		}

		executor := NewCallMatcherExecutor(ir, cg)
		matches := executor.Execute()

		assert.Len(t, matches, 1, "Should match 192.168.x.x binding")
		assert.Equal(t, 20, matches[0].Location.Line)
	})

	t.Run("detect port 8080", func(t *testing.T) {
		ir := &CallMatcherIR{
			Patterns: []string{"socket.bind"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0[1]": {Value: "8080", Wildcard: false},
			},
		}

		executor := NewCallMatcherExecutor(ir, cg)
		matches := executor.Execute()

		assert.Len(t, matches, 3, "All three use port 8080")
	})

	t.Run("detect both host and port", func(t *testing.T) {
		ir := &CallMatcherIR{
			Patterns: []string{"socket.bind"},
			PositionalArgs: map[string]ArgumentConstraint{
				"0[0]": {Value: "0.0.0.0", Wildcard: false},
				"0[1]": {Value: "8080", Wildcard: false},
			},
		}

		executor := NewCallMatcherExecutor(ir, cg)
		matches := executor.Execute()

		assert.Len(t, matches, 1, "Should match specific host+port combo")
		assert.Equal(t, 10, matches[0].Location.Line)
	})
}
