package dsl

import (
	"fmt"
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph/core"
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
	for i := 0; i < 1000; i++ {
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
		{Value: "=value", Position: 0},            // Missing key
		{Value: "key=", Position: 1},              // Missing value (should include)
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
