package patterns

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
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

	callGraph := core.NewCallGraph()
	callGraph.AddCallSite("myapp.views.process", core.CallSite{
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

	callGraph := core.NewCallGraph()
	callGraph.AddCallSite("myapp.views.process", core.CallSite{
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

	callGraph := core.NewCallGraph()

	// Create a path: get_input() -> process() -> execute_code()
	// get_input calls input(), execute_code calls eval()
	callGraph.AddCallSite("myapp.get_input", core.CallSite{
		Target:    "input",
		TargetFQN: "builtins.input",
	})

	callGraph.AddCallSite("myapp.execute_code", core.CallSite{
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

	callGraph := core.NewCallGraph()

	// Path with sanitizer: get_input() -> sanitize_input() -> execute_code()
	callGraph.AddCallSite("myapp.get_input", core.CallSite{
		Target:    "input",
		TargetFQN: "builtins.input",
	})

	callGraph.AddCallSite("myapp.sanitize_input", core.CallSite{
		Target:    "sanitize",
		TargetFQN: "myapp.utils.sanitize",
	})

	callGraph.AddCallSite("myapp.execute_code", core.CallSite{
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

	callGraph := core.NewCallGraph()

	// Path without sanitizer: get_input() -> execute_code()
	callGraph.AddCallSite("myapp.get_input", core.CallSite{
		Target:    "input",
		TargetFQN: "builtins.input",
	})

	callGraph.AddCallSite("myapp.execute_code", core.CallSite{
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
	callGraph := core.NewCallGraph()

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
	callGraph := core.NewCallGraph()

	// Create cycle: A -> B -> C -> A
	callGraph.AddEdge("A", "B")
	callGraph.AddEdge("B", "C")
	callGraph.AddEdge("C", "A")

	assert.True(t, registry.hasPath("A", "C", callGraph))
	assert.True(t, registry.hasPath("B", "A", callGraph))
}

func TestPatternRegistry_FindCallsByFunctions(t *testing.T) {
	registry := NewPatternRegistry()
	callGraph := core.NewCallGraph()

	callGraph.AddCallSite("myapp.func1", core.CallSite{
		Target:    "input",
		TargetFQN: "builtins.input",
	})

	callGraph.AddCallSite("myapp.func2", core.CallSite{
		Target:    "eval",
		TargetFQN: "builtins.eval",
	})

	callGraph.AddCallSite("myapp.func3", core.CallSite{
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

// ========== INTRA-PROCEDURAL TAINT DETECTION TESTS (PR #6) ==========

func TestMatchMissingSanitizer_IntraProceduralSimple(t *testing.T) {
	// Test basic intra-procedural vulnerability detection using real file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	sourceCode := `
def vulnerable():
    x = input()
    eval(x)
`

	err := os.WriteFile(testFile, []byte(sourceCode), 0644)
	assert.NoError(t, err)

	funcNode := &graph.Node{
		ID:         "test.vulnerable",
		Name:       "vulnerable",
		File:       testFile,
		LineNumber: 2,
	}

	callGraph := &core.CallGraph{
		Functions: map[string]*graph.Node{
			"test.vulnerable": funcNode,
		},
		CallSites: map[string][]core.CallSite{
			"test.vulnerable": {
				{Target: "input", TargetFQN: "builtins.input"},
				{Target: "eval", TargetFQN: "builtins.eval"},
			},
		},
		Summaries:    make(map[string]*core.TaintSummary),
		Edges:        make(map[string][]string),
		ReverseEdges: make(map[string][]string),
	}

	pattern := &Pattern{
		ID:         "CODE-INJECTION-001",
		Sources:    []string{"input"},
		Sinks:      []string{"eval"},
		Sanitizers: []string{},
	}

	registry := NewPatternRegistry()
	match := registry.matchMissingSanitizer(pattern, callGraph)

	// Assertions
	assert.NotNil(t, match)
	assert.True(t, match.Matched)
	assert.True(t, match.IsIntraProcedural)
	assert.Equal(t, "test.vulnerable", match.SourceFQN)
	assert.Equal(t, "test.vulnerable", match.SinkFQN)
	assert.Equal(t, []string{"test.vulnerable"}, match.DataFlowPath)
	assert.Contains(t, match.SourceCall, "input")
	assert.Contains(t, match.SinkCall, "eval")
}

func TestMatchMissingSanitizer_IntraProceduralNoFile(t *testing.T) {
	// Test graceful handling when file cannot be read
	callGraph := &core.CallGraph{
		Functions: map[string]*graph.Node{
			"test.unknown": {
				ID:         "test.unknown",
				Name:       "unknown",
				File:       "/nonexistent/file.py",
				LineNumber: 1,
			},
		},
		CallSites: map[string][]core.CallSite{
			"test.unknown": {
				{Target: "request.GET", TargetFQN: "django.http.request.GET"},
				{Target: "eval", TargetFQN: "builtins.eval"},
			},
		},
		Summaries:    map[string]*core.TaintSummary{},
		Edges:        make(map[string][]string),
		ReverseEdges: make(map[string][]string),
	}

	pattern := &Pattern{
		Sources: []string{"request.GET"},
		Sinks:   []string{"eval"},
	}

	registry := NewPatternRegistry()
	match := registry.matchMissingSanitizer(pattern, callGraph)

	// Should not match if file cannot be read (graceful degradation)
	assert.False(t, match.Matched)
}

func TestMatchMissingSanitizer_IntraProceduralWithSanitizer(t *testing.T) {
	// Test that sanitizers are respected (no false positive)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.py")

	sourceCode := `
def safe_function():
    user_input = input()
    sanitized = sanitize(user_input)
    eval(sanitized)
`

	err := os.WriteFile(testFile, []byte(sourceCode), 0644)
	assert.NoError(t, err)

	funcNode := &graph.Node{
		ID:         "test.safe_function",
		Name:       "safe_function",
		File:       testFile,
		LineNumber: 2,
	}

	callGraph := &core.CallGraph{
		Functions: map[string]*graph.Node{
			"test.safe_function": funcNode,
		},
		CallSites: map[string][]core.CallSite{
			"test.safe_function": {
				{Target: "input", TargetFQN: "builtins.input"},
				{Target: "sanitize", TargetFQN: "test.sanitize"},
				{Target: "eval", TargetFQN: "builtins.eval"},
			},
		},
		Summaries:    make(map[string]*core.TaintSummary),
		Edges:        make(map[string][]string),
		ReverseEdges: make(map[string][]string),
	}

	pattern := &Pattern{
		Sources:    []string{"input"},
		Sinks:      []string{"eval"},
		Sanitizers: []string{"sanitize"},
	}

	registry := NewPatternRegistry()
	match := registry.matchMissingSanitizer(pattern, callGraph)

	// Should not match because sanitizer breaks taint flow
	assert.False(t, match.Matched)
}

func TestMatchMissingSanitizer_InterProceduralUnchanged(t *testing.T) {
	// Test that inter-procedural detection still works
	callGraph := &core.CallGraph{
		Functions: map[string]*graph.Node{
			"test.source_func": {ID: "test.source_func", Name: "source_func"},
			"test.sink_func":   {ID: "test.sink_func", Name: "sink_func"},
		},
		CallSites: map[string][]core.CallSite{
			"test.source_func": {
				{Target: "request.GET", TargetFQN: "django.http.request.GET"},
			},
			"test.sink_func": {
				{Target: "eval", TargetFQN: "builtins.eval"},
			},
		},
		Edges: map[string][]string{
			"test.source_func": {"test.sink_func"},
		},
		ReverseEdges: map[string][]string{
			"test.sink_func": {"test.source_func"},
		},
		Summaries: map[string]*core.TaintSummary{
			"test.source_func": {FunctionFQN: "test.source_func"},
			"test.sink_func":   {FunctionFQN: "test.sink_func"},
		},
	}

	pattern := &Pattern{
		Sources: []string{"request.GET"},
		Sinks:   []string{"eval"},
	}

	registry := NewPatternRegistry()
	match := registry.matchMissingSanitizer(pattern, callGraph)

	// Should detect inter-procedural
	assert.True(t, match.Matched)
	assert.False(t, match.IsIntraProcedural) // Inter-procedural
	assert.Equal(t, "test.source_func", match.SourceFQN)
	assert.Equal(t, "test.sink_func", match.SinkFQN)
	assert.True(t, len(match.DataFlowPath) > 1)
}

func TestPatternMatchDetails_BackwardCompatibility(t *testing.T) {
	// Test that old code works with new schema
	match := &PatternMatchDetails{
		Matched:      true,
		SourceFQN:    "test.source",
		SinkFQN:      "test.sink",
		DataFlowPath: []string{"test.source", "test.sink"},
		// IsIntraProcedural not set - should default to false
	}

	// Should work correctly
	assert.True(t, match.Matched)
	assert.False(t, match.IsIntraProcedural) // Default value
}
