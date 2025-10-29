package callgraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPatternRegistry(t *testing.T) {
	registry := NewPatternRegistry()

	assert.NotNil(t, registry)
	assert.NotNil(t, registry.Patterns)
	assert.NotNil(t, registry.PatternsByType)
	assert.Empty(t, registry.Patterns)
	assert.Empty(t, registry.PatternsByType)
}

func TestPatternRegistry_AddPattern(t *testing.T) {
	registry := NewPatternRegistry()

	pattern := &Pattern{
		ID:       "TEST-001",
		Name:     "Test Pattern",
		Type:     PatternTypeDangerousFunction,
		Severity: SeverityHigh,
	}

	registry.AddPattern(pattern)

	assert.Len(t, registry.Patterns, 1)
	assert.Equal(t, pattern, registry.Patterns["TEST-001"])
	assert.Len(t, registry.PatternsByType[PatternTypeDangerousFunction], 1)
}

func TestPatternRegistry_GetPattern(t *testing.T) {
	registry := NewPatternRegistry()

	pattern := &Pattern{ID: "TEST-001", Name: "Test"}
	registry.AddPattern(pattern)

	retrieved, exists := registry.GetPattern("TEST-001")
	assert.True(t, exists)
	assert.Equal(t, pattern, retrieved)

	_, exists = registry.GetPattern("NONEXISTENT")
	assert.False(t, exists)
}

func TestPatternRegistry_GetPatternsByType(t *testing.T) {
	registry := NewPatternRegistry()

	p1 := &Pattern{ID: "P1", Type: PatternTypeDangerousFunction}
	p2 := &Pattern{ID: "P2", Type: PatternTypeDangerousFunction}
	p3 := &Pattern{ID: "P3", Type: PatternTypeSourceSink}

	registry.AddPattern(p1)
	registry.AddPattern(p2)
	registry.AddPattern(p3)

	dangerous := registry.GetPatternsByType(PatternTypeDangerousFunction)
	assert.Len(t, dangerous, 2)

	sourceSink := registry.GetPatternsByType(PatternTypeSourceSink)
	assert.Len(t, sourceSink, 1)
}

func TestPatternRegistry_LoadDefaultPatterns(t *testing.T) {
	registry := NewPatternRegistry()
	registry.LoadDefaultPatterns()

	pattern, exists := registry.GetPattern("CODE-INJECTION-001")
	require.True(t, exists)
	assert.Equal(t, "Code injection via eval with user input", pattern.Name)
	assert.Equal(t, PatternTypeMissingSanitizer, pattern.Type)
	assert.Equal(t, SeverityCritical, pattern.Severity)
	assert.Contains(t, pattern.Sources, "input")
	assert.Contains(t, pattern.Sinks, "eval")
	assert.Contains(t, pattern.Sanitizers, "sanitize")
}

func TestMatchesFunctionName(t *testing.T) {
	tests := []struct {
		name     string
		fqn      string
		pattern  string
		expected bool
	}{
		{"Exact match", "eval", "eval", true},
		{"Suffix match", "myapp.utils.eval", "eval", true},
		{"Prefix match", "request.GET.get", "request.GET", true},
		{"No match", "myapp.safe_function", "eval", false},
		{"Partial no match", "evaluation", "eval", false}, // Should NOT match - avoids false positives
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesFunctionName(tt.fqn, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPatternRegistry_MatchDangerousFunction(t *testing.T) {
	registry := NewPatternRegistry()
	pattern := &Pattern{
		ID:                 "TEST-DANGEROUS",
		Type:               PatternTypeDangerousFunction,
		DangerousFunctions: []string{"eval", "exec"},
	}

	callGraph := NewCallGraph()
	callGraph.AddCallSite("myapp.views.process", CallSite{
		Target:    "eval",
		TargetFQN: "builtins.eval",
	})

	matched := registry.MatchPattern(pattern, callGraph)
	assert.NotNil(t, matched)
	assert.True(t, matched.Matched)
}

func TestPatternRegistry_MatchDangerousFunction_NoMatch(t *testing.T) {
	registry := NewPatternRegistry()
	pattern := &Pattern{
		ID:                 "TEST-DANGEROUS",
		Type:               PatternTypeDangerousFunction,
		DangerousFunctions: []string{"eval", "exec"},
	}

	callGraph := NewCallGraph()
	callGraph.AddCallSite("myapp.views.process", CallSite{
		Target:    "safe_function",
		TargetFQN: "myapp.utils.safe_function",
	})

	matched := registry.MatchPattern(pattern, callGraph)
	if matched == nil || !matched.Matched {
		// Pattern didn't match (expected)
		assert.True(t, true)
	} else {
		assert.Fail(t, "Expected no match but found one")
	}
}

func TestPatternRegistry_MatchSourceSink(t *testing.T) {
	registry := NewPatternRegistry()
	pattern := &Pattern{
		ID:      "TEST-SOURCE-SINK",
		Type:    PatternTypeSourceSink,
		Sources: []string{"input"},
		Sinks:   []string{"eval"},
	}

	callGraph := NewCallGraph()

	// Create a path: get_input() -> process() -> execute_code()
	// get_input calls input(), execute_code calls eval()
	callGraph.AddCallSite("myapp.get_input", CallSite{
		Target:    "input",
		TargetFQN: "builtins.input",
	})

	callGraph.AddCallSite("myapp.execute_code", CallSite{
		Target:    "eval",
		TargetFQN: "builtins.eval",
	})

	callGraph.AddEdge("myapp.get_input", "myapp.process")
	callGraph.AddEdge("myapp.process", "myapp.execute_code")

	matched := registry.MatchPattern(pattern, callGraph)
	assert.NotNil(t, matched)
	assert.True(t, matched.Matched)
}

func TestPatternRegistry_MatchMissingSanitizer_WithSanitizer(t *testing.T) {
	registry := NewPatternRegistry()
	pattern := &Pattern{
		ID:         "TEST-SANITIZER",
		Type:       PatternTypeMissingSanitizer,
		Sources:    []string{"input"},
		Sinks:      []string{"eval"},
		Sanitizers: []string{"sanitize"},
	}

	callGraph := NewCallGraph()

	// Path with sanitizer: get_input() -> sanitize_input() -> execute_code()
	callGraph.AddCallSite("myapp.get_input", CallSite{
		Target:    "input",
		TargetFQN: "builtins.input",
	})

	callGraph.AddCallSite("myapp.sanitize_input", CallSite{
		Target:    "sanitize",
		TargetFQN: "myapp.utils.sanitize",
	})

	callGraph.AddCallSite("myapp.execute_code", CallSite{
		Target:    "eval",
		TargetFQN: "builtins.eval",
	})

	callGraph.AddEdge("myapp.get_input", "myapp.sanitize_input")
	callGraph.AddEdge("myapp.sanitize_input", "myapp.execute_code")

	matched := registry.MatchPattern(pattern, callGraph)
	if matched == nil || !matched.Matched {
		// Pattern didn't match (expected)
		assert.True(t, true)
	} else {
		assert.Fail(t, "Expected no match but found one")
	} // Should not match because sanitizer is present
}

func TestPatternRegistry_MatchMissingSanitizer_WithoutSanitizer(t *testing.T) {
	registry := NewPatternRegistry()
	pattern := &Pattern{
		ID:         "TEST-SANITIZER",
		Type:       PatternTypeMissingSanitizer,
		Sources:    []string{"input"},
		Sinks:      []string{"eval"},
		Sanitizers: []string{"sanitize"},
	}

	callGraph := NewCallGraph()

	// Path without sanitizer: get_input() -> execute_code()
	callGraph.AddCallSite("myapp.get_input", CallSite{
		Target:    "input",
		TargetFQN: "builtins.input",
	})

	callGraph.AddCallSite("myapp.execute_code", CallSite{
		Target:    "eval",
		TargetFQN: "builtins.eval",
	})

	callGraph.AddEdge("myapp.get_input", "myapp.execute_code")

	matched := registry.MatchPattern(pattern, callGraph)
	assert.NotNil(t, matched)
	assert.True(t, matched.Matched) // Should match because sanitizer is missing
}

func TestPatternRegistry_HasPath(t *testing.T) {
	registry := NewPatternRegistry()
	callGraph := NewCallGraph()

	// Create path: A -> B -> C
	callGraph.AddEdge("A", "B")
	callGraph.AddEdge("B", "C")

	assert.True(t, registry.hasPath("A", "A", callGraph))
	assert.True(t, registry.hasPath("A", "B", callGraph))
	assert.True(t, registry.hasPath("A", "C", callGraph))
	assert.False(t, registry.hasPath("C", "A", callGraph))
	assert.False(t, registry.hasPath("B", "A", callGraph))
}

func TestPatternRegistry_HasPath_Cycle(t *testing.T) {
	registry := NewPatternRegistry()
	callGraph := NewCallGraph()

	// Create cycle: A -> B -> C -> A
	callGraph.AddEdge("A", "B")
	callGraph.AddEdge("B", "C")
	callGraph.AddEdge("C", "A")

	assert.True(t, registry.hasPath("A", "C", callGraph))
	assert.True(t, registry.hasPath("B", "A", callGraph))
}

func TestPatternRegistry_FindCallsByFunctions(t *testing.T) {
	registry := NewPatternRegistry()
	callGraph := NewCallGraph()

	callGraph.AddCallSite("myapp.func1", CallSite{
		Target:    "input",
		TargetFQN: "builtins.input",
	})

	callGraph.AddCallSite("myapp.func2", CallSite{
		Target:    "eval",
		TargetFQN: "builtins.eval",
	})

	callGraph.AddCallSite("myapp.func3", CallSite{
		Target:    "print",
		TargetFQN: "builtins.print",
	})

	calls := registry.findCallsByFunctions([]string{"input", "eval"}, callGraph)

	assert.Len(t, calls, 2)
	callers := []string{calls[0].caller, calls[1].caller}
	assert.Contains(t, callers, "myapp.func1")
	assert.Contains(t, callers, "myapp.func2")
}

func TestSeverityConstants(t *testing.T) {
	assert.Equal(t, Severity("critical"), SeverityCritical)
	assert.Equal(t, Severity("high"), SeverityHigh)
	assert.Equal(t, Severity("medium"), SeverityMedium)
	assert.Equal(t, Severity("low"), SeverityLow)
}

func TestPatternTypeConstants(t *testing.T) {
	assert.Equal(t, PatternType("source-sink"), PatternTypeSourceSink)
	assert.Equal(t, PatternType("missing-sanitizer"), PatternTypeMissingSanitizer)
	assert.Equal(t, PatternType("dangerous-function"), PatternTypeDangerousFunction)
}
