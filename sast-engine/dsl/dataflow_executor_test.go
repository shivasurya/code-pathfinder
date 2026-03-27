package dsl

import (
	"encoding/json"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

// toRawMessages marshals CallMatcherIR slices to json.RawMessage for test DataflowIR construction.
func toRawMessages(matchers ...CallMatcherIR) []json.RawMessage {
	raw := make([]json.RawMessage, 0, len(matchers))
	for _, m := range matchers {
		b, _ := json.Marshal(m)
		raw = append(raw, json.RawMessage(b))
	}
	return raw
}

// toRawMessagesTyped marshals TypeConstrainedCallIR slices to json.RawMessage.
func toRawMessagesTyped(matchers ...TypeConstrainedCallIR) []json.RawMessage {
	raw := make([]json.RawMessage, 0, len(matchers))
	for _, m := range matchers {
		b, _ := json.Marshal(m)
		raw = append(raw, json.RawMessage(b))
	}
	return raw
}

func emptyRawMessages() []json.RawMessage {
	return []json.RawMessage{}
}

func TestDataflowExecutor_Local(t *testing.T) {
	t.Run("executes local analysis and finds detections", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.dangerous"] = []core.CallSite{
			{
				Target:   "request.POST",
				Location: core.Location{File: "test.py", Line: 5},
			},
			{
				Target:   "execute",
				Location: core.Location{File: "test.py", Line: 10},
			},
		}

		ir := &DataflowIR{
			Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"request.POST"}}),
			Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}}),
			Sanitizers: emptyRawMessages(),
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.executeLocal()

		assert.Len(t, detections, 1)
		assert.Equal(t, "test.dangerous", detections[0].FunctionFQN)
		assert.Equal(t, 5, detections[0].SourceLine)
		assert.Equal(t, 10, detections[0].SinkLine)
		assert.Equal(t, "execute", detections[0].SinkCall)
		assert.Equal(t, "local", detections[0].Scope)
		assert.Equal(t, 0.50, detections[0].Confidence)
		assert.Equal(t, "line_proximity", detections[0].MatchMethod)
		assert.False(t, detections[0].Sanitized)
	})

	t.Run("sanitizer between source and sink filters detection", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.safe"] = []core.CallSite{
			{Target: "request.GET", Location: core.Location{File: "test.py", Line: 5}},
			{Target: "escape_sql", Location: core.Location{File: "test.py", Line: 8}},
			{Target: "execute", Location: core.Location{File: "test.py", Line: 12}},
		}

		ir := &DataflowIR{
			Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"request.GET"}}),
			Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"execute"}}),
			Sanitizers: toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"escape_sql"}}),
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.executeLocal()

		assert.Empty(t, detections, "Sanitized flow should be filtered out, not reported")
	})

	t.Run("ignores cross-function flows in local scope", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.func1"] = []core.CallSite{
			{Target: "request.GET", Location: core.Location{File: "test.py", Line: 5}},
		}
		cg.CallSites["test.func2"] = []core.CallSite{
			{Target: "eval", Location: core.Location{File: "test.py", Line: 15}},
		}

		ir := &DataflowIR{
			Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"request.GET"}}),
			Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}}),
			Sanitizers: emptyRawMessages(),
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.executeLocal()

		assert.Empty(t, detections, "Local scope should not detect cross-function flows")
	})

	t.Run("handles multiple sources and sinks in same function", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.multi"] = []core.CallSite{
			{Target: "request.GET", Location: core.Location{File: "test.py", Line: 5}},
			{Target: "request.POST", Location: core.Location{File: "test.py", Line: 7}},
			{Target: "eval", Location: core.Location{File: "test.py", Line: 10}},
			{Target: "execute", Location: core.Location{File: "test.py", Line: 15}},
		}

		ir := &DataflowIR{
			Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"request.GET", "request.POST"}}),
			Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval", "execute"}}),
			Sanitizers: emptyRawMessages(),
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.executeLocal()

		// Should find 2 sources * 2 sinks = 4 combinations.
		assert.Len(t, detections, 4)
	})
}

func TestDataflowExecutor_Global(t *testing.T) {
	t.Run("detects cross-function flow", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.Edges = make(map[string][]string)
		cg.Edges["test.get_input"] = []string{"test.process"}

		cg.CallSites["test.get_input"] = []core.CallSite{
			{Target: "request.GET", Location: core.Location{Line: 10}},
			{Target: "process", TargetFQN: "test.process", Location: core.Location{Line: 12}},
		}

		cg.CallSites["test.process"] = []core.CallSite{
			{Target: "eval", Location: core.Location{Line: 20}},
		}

		ir := &DataflowIR{
			Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"request.GET"}}),
			Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}}),
			Sanitizers: emptyRawMessages(),
			Scope:      "global",
		}

		executor := NewDataflowExecutor(ir, cg)

		path := executor.findPath("test.get_input", "test.process")
		assert.NotEmpty(t, path)
		assert.Contains(t, path, "test.get_input")
		assert.Contains(t, path, "test.process")
	})

	t.Run("executes global analysis and finds cross-function flows", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.Edges = make(map[string][]string)
		cg.Edges["test.source_func"] = []string{"test.sink_func"}

		cg.CallSites["test.source_func"] = []core.CallSite{
			{Target: "request.GET", Location: core.Location{Line: 10, File: "test.py"}},
		}

		cg.CallSites["test.sink_func"] = []core.CallSite{
			{Target: "eval", Location: core.Location{Line: 20, File: "test.py"}},
		}

		ir := &DataflowIR{
			Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"request.GET"}}),
			Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}}),
			Sanitizers: emptyRawMessages(),
			Scope:      "global",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.executeGlobal()

		assert.NotEmpty(t, detections)
		found := false
		for _, d := range detections {
			if d.FunctionFQN == "test.sink_func" && d.Scope == "global" {
				found = true
				assert.Equal(t, 10, d.SourceLine)
				assert.Equal(t, 20, d.SinkLine)
				assert.Equal(t, "eval", d.SinkCall)
				assert.False(t, d.Sanitized)
				assert.Equal(t, 0.8, d.Confidence)
			}
		}
		assert.True(t, found, "Should find cross-function detection")
	})

	t.Run("excludes flows with sanitizer on path", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.Edges = make(map[string][]string)
		cg.Edges["test.source"] = []string{"test.sanitize"}
		cg.Edges["test.sanitize"] = []string{"test.sink"}

		cg.CallSites["test.source"] = []core.CallSite{
			{Target: "request.POST", Location: core.Location{Line: 5, File: "test.py"}},
		}
		cg.CallSites["test.sanitize"] = []core.CallSite{
			{Target: "escape_html", Location: core.Location{Line: 10, File: "test.py"}},
		}
		cg.CallSites["test.sink"] = []core.CallSite{
			{Target: "render", Location: core.Location{Line: 15, File: "test.py"}},
		}

		ir := &DataflowIR{
			Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"request.POST"}}),
			Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"render"}}),
			Sanitizers: toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"escape_html"}}),
			Scope:      "global",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.executeGlobal()

		globalDetections := []DataflowDetection{}
		for _, d := range detections {
			if d.Scope == "global" {
				globalDetections = append(globalDetections, d)
			}
		}
		assert.Empty(t, globalDetections, "Should not detect flows with sanitizer on path")
	})
}

// matchesPattern tests removed — the method was dead code on DataflowExecutor
// and has been deleted. Pattern matching is still tested via CallMatcherExecutor.

// --- Polymorphic matcher tests ---

func TestDataflowExecutor_TypeConstrainedSources(t *testing.T) {
	t.Run("flows with TypeConstrainedCall source and sink", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["myapp.views.search"] = []core.CallSite{
			{
				Target:    "request.args.get",
				TargetFQN: "flask.request.args.get",
				Location:  core.Location{File: "views.py", Line: 5},
			},
			{
				Target:    "cursor.execute",
				TargetFQN: "sqlite3.Cursor.execute",
				Location:  core.Location{File: "views.py", Line: 10},
				ResolvedViaTypeInference: true,
				InferredType:             "sqlite3.Cursor",
				TypeConfidence:           0.9,
			},
		}

		// Source: TypeConstrainedCall for flask.request
		sourceIR := TypeConstrainedCallIR{
			Type:          "type_constrained_call",
			ReceiverTypes: []string{"flask.request"},
			MethodNames:   []string{"get", "args"},
			MinConfidence: 0.5,
			FallbackMode:  "name",
		}
		// Sink: TypeConstrainedCall for sqlite3.Cursor
		sinkIR := TypeConstrainedCallIR{
			Type:          "type_constrained_call",
			ReceiverTypes: []string{"sqlite3.Cursor"},
			MethodNames:   []string{"execute"},
			MinConfidence: 0.5,
		}

		ir := &DataflowIR{
			Sources:    toRawMessagesTyped(sourceIR),
			Sinks:      toRawMessagesTyped(sinkIR),
			Sanitizers: emptyRawMessages(),
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		assert.NotEmpty(t, detections, "Should detect flow from typed source to typed sink")
	})

	t.Run("mixed CallMatcher source + TypeConstrained sink", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["myapp.views.index"] = []core.CallSite{
			{
				Target:   "request.GET",
				Location: core.Location{File: "views.py", Line: 3},
			},
			{
				Target:                   "cursor.execute",
				Location:                 core.Location{File: "views.py", Line: 8},
				ResolvedViaTypeInference: true,
				InferredType:             "sqlite3.Cursor",
				TypeConfidence:           0.9,
			},
		}

		sourceJSON, _ := json.Marshal(CallMatcherIR{
			Type:     "call_matcher",
			Patterns: []string{"request.GET"},
		})
		sinkJSON, _ := json.Marshal(TypeConstrainedCallIR{
			Type:          "type_constrained_call",
			ReceiverTypes: []string{"sqlite3.Cursor"},
			MethodNames:   []string{"execute"},
			MinConfidence: 0.5,
		})

		ir := &DataflowIR{
			Sources:    []json.RawMessage{sourceJSON},
			Sinks:      []json.RawMessage{sinkJSON},
			Sanitizers: emptyRawMessages(),
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		assert.NotEmpty(t, detections, "Should detect mixed CallMatcher + TypeConstrained flow")
	})
}

// --- Backward compatibility via loader ---

func TestDataflowExecutor_BackwardCompat_ViaLoader(t *testing.T) {
	t.Run("old-style dataflow IR with call_matcher works via loader", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.func"] = []core.CallSite{
			{Target: "request.GET", Location: core.Location{File: "test.py", Line: 5}},
			{Target: "eval", Location: core.Location{File: "test.py", Line: 10}},
		}

		loader := NewRuleLoader("")
		rule := &RuleIR{
			Matcher: map[string]any{
				"type": "dataflow",
				"sources": []any{
					map[string]any{"type": "call_matcher", "patterns": []any{"request.GET"}, "wildcard": false},
				},
				"sinks": []any{
					map[string]any{"type": "call_matcher", "patterns": []any{"eval"}, "wildcard": false},
				},
				"sanitizers":  []any{},
				"propagation": []any{},
				"scope":       "local",
			},
		}

		detections, err := loader.ExecuteRule(rule, cg)
		assert.NoError(t, err)
		assert.Len(t, detections, 1)
		assert.Equal(t, "test.func", detections[0].FunctionFQN)
	})

	t.Run("new-style dataflow IR with type_constrained_call works via loader", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.func"] = []core.CallSite{
			{Target: "request.GET", Location: core.Location{File: "test.py", Line: 5}},
			{
				Target:                   "cursor.execute",
				Location:                 core.Location{File: "test.py", Line: 10},
				ResolvedViaTypeInference: true,
				InferredType:             "sqlite3.Cursor",
				TypeConfidence:           0.9,
			},
		}

		loader := NewRuleLoader("")
		rule := &RuleIR{
			Matcher: map[string]any{
				"type": "dataflow",
				"sources": []any{
					map[string]any{"type": "call_matcher", "patterns": []any{"request.GET"}, "wildcard": false},
				},
				"sinks": []any{
					map[string]any{
						"type":          "type_constrained_call",
						"receiverTypes": []any{"sqlite3.Cursor"},
						"methodNames":   []any{"execute"},
						"minConfidence": 0.5,
						"fallbackMode":  "none",
					},
				},
				"sanitizers":  []any{},
				"propagation": []any{},
				"scope":       "local",
			},
		}

		detections, err := loader.ExecuteRule(rule, cg)
		assert.NoError(t, err)
		assert.NotEmpty(t, detections)
	})
}

// --- resolveMatchers edge cases ---

func TestResolveMatchers_UnknownType(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["test.func"] = []core.CallSite{
		{Target: "eval", Location: core.Location{File: "test.py", Line: 5}},
	}

	ir := &DataflowIR{}
	executor := NewDataflowExecutor(ir, cg)

	// Unknown matcher type should be silently skipped
	unknownJSON, _ := json.Marshal(map[string]any{"type": "unknown_matcher", "patterns": []string{"eval"}})
	matches := executor.resolveMatchers([]json.RawMessage{unknownJSON})
	assert.Empty(t, matches, "Unknown matcher type should be skipped")
}

func TestResolveMatchers_InvalidJSON(t *testing.T) {
	cg := core.NewCallGraph()
	ir := &DataflowIR{}
	executor := NewDataflowExecutor(ir, cg)

	// Invalid JSON in raw message should be skipped (can't even peek type)
	invalidJSON := json.RawMessage([]byte(`{not valid json`))
	matches := executor.resolveMatchers([]json.RawMessage{invalidJSON})
	assert.Empty(t, matches, "Invalid JSON should be skipped")
}

func TestResolveMatchers_MalformedCallMatcherBody(t *testing.T) {
	cg := core.NewCallGraph()
	ir := &DataflowIR{}
	executor := NewDataflowExecutor(ir, cg)

	// Type says call_matcher but patterns field is wrong type — unmarshal into
	// CallMatcherIR will either produce zero-value Patterns or fail; should not panic.
	malformedJSON := json.RawMessage([]byte(`{"type":"call_matcher","patterns":"not_an_array"}`))
	matches := executor.resolveMatchers([]json.RawMessage{malformedJSON})
	// Should either be empty (unmarshal error → skip) or produce no matches (empty patterns)
	// If we get here without panic, the test passes. Matches may be empty or not.
	_ = matches
}

// --- Execute() dispatch ---

func TestExecute_DispatchesGlobalScope(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Edges = make(map[string][]string)
	cg.Edges["test.source"] = []string{"test.sink"}
	cg.CallSites["test.source"] = []core.CallSite{
		{Target: "request.GET", Location: core.Location{File: "test.py", Line: 5}},
	}
	cg.CallSites["test.sink"] = []core.CallSite{
		{Target: "eval", Location: core.Location{File: "test.py", Line: 20}},
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"request.GET"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "global",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	// Execute with scope != "local" should call executeGlobal and find cross-function flow
	foundGlobal := false
	for _, d := range detections {
		if d.Scope == "global" {
			foundGlobal = true
			break
		}
	}
	assert.True(t, foundGlobal, "Execute with global scope should produce global detections")
}

// --- dfs() edge cases ---

func TestDFS_SelfLoop(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Edges = make(map[string][]string)
	cg.Edges["test.loop"] = []string{"test.loop"}

	ir := &DataflowIR{}
	executor := NewDataflowExecutor(ir, cg)

	// Self-loop: findPath from A to A should return [A] immediately (short-circuit)
	path := executor.findPath("test.loop", "test.loop")
	assert.Equal(t, []string{"test.loop"}, path)
}

func TestDFS_NoPath(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Edges = make(map[string][]string)
	cg.Edges["test.a"] = []string{"test.b"}
	// test.c is disconnected

	ir := &DataflowIR{}
	executor := NewDataflowExecutor(ir, cg)

	path := executor.findPath("test.a", "test.c")
	assert.Empty(t, path, "No path should exist between disconnected nodes")
}

func TestDFS_DiamondGraph(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Edges = make(map[string][]string)
	cg.Edges["test.a"] = []string{"test.b", "test.c"}
	cg.Edges["test.b"] = []string{"test.d"}
	cg.Edges["test.c"] = []string{"test.d"}

	ir := &DataflowIR{}
	executor := NewDataflowExecutor(ir, cg)

	path := executor.findPath("test.a", "test.d")
	assert.NotEmpty(t, path, "Should find a path in diamond graph")
	assert.Equal(t, "test.a", path[0])
	assert.Equal(t, "test.d", path[len(path)-1])
	assert.True(t, len(path) == 3, "Path should be length 3 (A->B->D or A->C->D)")
}

// --- pathHasSanitizerSet edge cases ---

func TestPathHasSanitizerSet_EmptyPath(t *testing.T) {
	cg := core.NewCallGraph()
	ir := &DataflowIR{}
	executor := NewDataflowExecutor(ir, cg)

	sanitizerSet := map[string]bool{"test.sanitize": true}

	result := executor.pathHasSanitizerSet([]string{}, sanitizerSet)
	assert.False(t, result, "Empty path should return false")
}

func TestPathHasSanitizerSet_EmptySanitizers(t *testing.T) {
	cg := core.NewCallGraph()
	ir := &DataflowIR{}
	executor := NewDataflowExecutor(ir, cg)

	result := executor.pathHasSanitizerSet([]string{"test.a", "test.b"}, map[string]bool{})
	assert.False(t, result, "Empty sanitizers should return false")
}

// --- Type-constrained sanitizer test ---

func TestDataflowExecutor_SanitizerWorksWithTypeConstrainedSink(t *testing.T) {
	t.Run("sanitizer blocks flow to type_constrained_call sink", func(t *testing.T) {
		// Simulates: input() → int() → cursor.execute()
		// Where cursor.execute is matched via type inference (type_constrained_call).
		// The int() sanitizer at line 8 should block the flow from input() at line 5
		// to cursor.execute at line 15. The sanitized detection should be filtered out.
		cg := core.NewCallGraph()
		cg.CallSites["app.sanitized_sql"] = []core.CallSite{
			{
				Target:   "input",
				Location: core.Location{File: "app.py", Line: 5, Column: 10},
			},
			{
				Target:   "int",
				Location: core.Location{File: "app.py", Line: 8, Column: 14},
			},
			{
				Target:                   "cursor.execute",
				TargetFQN:                "sqlite3.Cursor.execute",
				Location:                 core.Location{File: "app.py", Line: 15, Column: 4},
				ResolvedViaTypeInference: true,
				InferredType:             "sqlite3.Cursor",
				TypeConfidence:           0.9,
			},
		}

		// Source: call_matcher for input()
		sourceJSON, _ := json.Marshal(CallMatcherIR{
			Type:     "call_matcher",
			Patterns: []string{"input"},
		})

		// Sink: type_constrained_call for sqlite3.Cursor.execute
		sinkJSON, _ := json.Marshal(TypeConstrainedCallIR{
			Type:          "type_constrained_call",
			ReceiverTypes: []string{"sqlite3.Cursor"},
			MethodNames:   []string{"execute"},
			MinConfidence: 0.5,
			FallbackMode:  "none",
		})

		// Sanitizer: call_matcher for int()
		sanitizerJSON, _ := json.Marshal(CallMatcherIR{
			Type:     "call_matcher",
			Patterns: []string{"int"},
		})

		ir := &DataflowIR{
			Sources:    []json.RawMessage{sourceJSON},
			Sinks:      []json.RawMessage{sinkJSON},
			Sanitizers: []json.RawMessage{sanitizerJSON},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		assert.Empty(t, detections,
			"Sanitized flow should be filtered out — int() between input() and cursor.execute()")
	})

	t.Run("unsanitized flow to type_constrained_call sink is NOT marked sanitized", func(t *testing.T) {
		// Simulates: input() → cursor.execute() (no sanitizer)
		cg := core.NewCallGraph()
		cg.CallSites["app.unsafe_sql"] = []core.CallSite{
			{
				Target:   "input",
				Location: core.Location{File: "app.py", Line: 5, Column: 10},
			},
			{
				Target:                   "cursor.execute",
				TargetFQN:                "sqlite3.Cursor.execute",
				Location:                 core.Location{File: "app.py", Line: 12, Column: 4},
				ResolvedViaTypeInference: true,
				InferredType:             "sqlite3.Cursor",
				TypeConfidence:           0.9,
			},
		}

		sourceJSON, _ := json.Marshal(CallMatcherIR{
			Type:     "call_matcher",
			Patterns: []string{"input"},
		})
		sinkJSON, _ := json.Marshal(TypeConstrainedCallIR{
			Type:          "type_constrained_call",
			ReceiverTypes: []string{"sqlite3.Cursor"},
			MethodNames:   []string{"execute"},
			MinConfidence: 0.5,
			FallbackMode:  "none",
		})

		ir := &DataflowIR{
			Sources:    []json.RawMessage{sourceJSON},
			Sinks:      []json.RawMessage{sinkJSON},
			Sanitizers: emptyRawMessages(),
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		assert.Len(t, detections, 1, "Should produce exactly one detection")
		assert.False(t, detections[0].Sanitized,
			"Detection should NOT be marked as Sanitized when no sanitizer exists")
	})
}

// matchesPattern edge case tests removed — dead code deleted from DataflowExecutor.

// ============================================================================
// VDG-specific tests (from demand-driven-dataflow branch)
// These test VDG helper functions and inter-procedural analysis.
// Updated to use toRawMessages() for DataflowIR construction.
// ============================================================================

func TestDataflowExecutor_VDG_Local(t *testing.T) {
	t.Run("finds functions with sources and sinks", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.vulnerable"] = []core.CallSite{
			{
				Target:   "request.GET",
				Location: core.Location{File: "test.py", Line: 10},
			},
			{
				Target:   "eval",
				Location: core.Location{File: "test.py", Line: 15},
			},
		}

		ir := &DataflowIR{
			Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"request.GET"}}),
			Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}}),
			Sanitizers: emptyRawMessages(),
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)

		sourceCalls := executor.resolveMatchers(ir.Sources)
		sinkCalls := executor.resolveMatchers(ir.Sinks)

		functions := executor.findFunctionsWithSourcesAndSinks(sourceCalls, sinkCalls)

		assert.Contains(t, functions, "test.vulnerable")
	})

	t.Run("handles multiple sources and sinks in same function via resolveMatchers", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.multi"] = []core.CallSite{
			{
				Target:   "request.GET",
				Location: core.Location{File: "test.py", Line: 5},
			},
			{
				Target:   "request.POST",
				Location: core.Location{File: "test.py", Line: 7},
			},
			{
				Target:   "eval",
				Location: core.Location{File: "test.py", Line: 10},
			},
			{
				Target:   "execute",
				Location: core.Location{File: "test.py", Line: 15},
			},
		}

		ir := &DataflowIR{
			Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"request.GET", "request.POST"}}),
			Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval", "execute"}}),
			Sanitizers: emptyRawMessages(),
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)

		sourceCalls := executor.resolveMatchers(ir.Sources)
		sinkCalls := executor.resolveMatchers(ir.Sinks)

		assert.Len(t, sourceCalls, 2)
		assert.Len(t, sinkCalls, 2)

		functions := executor.findFunctionsWithSourcesAndSinks(sourceCalls, sinkCalls)
		assert.Contains(t, functions, "test.multi")
	})
}

func TestDataflowExecutor_VDG_Global(t *testing.T) {
	t.Run("detects cross-function flow via path finding", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.Edges = make(map[string][]string)
		cg.Edges["test.get_input"] = []string{"test.process"}

		cg.CallSites["test.get_input"] = []core.CallSite{
			{
				Target:   "request.GET",
				Location: core.Location{Line: 10},
			},
			{
				Target:    "process",
				TargetFQN: "test.process",
				Location:  core.Location{Line: 12},
			},
		}

		cg.CallSites["test.process"] = []core.CallSite{
			{
				Target:   "eval",
				Location: core.Location{Line: 20},
			},
		}

		executor := NewDataflowExecutor(&DataflowIR{Scope: "global"}, cg)

		// Test path finding
		path := executor.findPath("test.get_input", "test.process")
		assert.NotEmpty(t, path)
		assert.Contains(t, path, "test.get_input")
		assert.Contains(t, path, "test.process")
	})

	t.Run("detects sanitizer on path via pathHasSanitizerSet", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.Edges = make(map[string][]string)
		cg.Edges["test.source"] = []string{"test.sanitize"}
		cg.Edges["test.sanitize"] = []string{"test.sink"}

		cg.CallSites["test.sanitize"] = []core.CallSite{
			{
				Target:   "escape_sql",
				Location: core.Location{Line: 15},
			},
		}

		ir := &DataflowIR{
			Sources:    emptyRawMessages(),
			Sinks:      emptyRawMessages(),
			Sanitizers: toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"escape_sql"}}),
			Scope:      "global",
		}
		executor := NewDataflowExecutor(ir, cg)

		sanitizerCalls := executor.resolveMatchers(ir.Sanitizers)
		sanitizerSet := make(map[string]bool, len(sanitizerCalls))
		for _, san := range sanitizerCalls {
			sanitizerSet[san.FunctionFQN] = true
		}

		path := []string{"test.source", "test.sanitize", "test.sink"}
		hasSanitizer := executor.pathHasSanitizerSet(path, sanitizerSet)
		assert.True(t, hasSanitizer)
	})

	t.Run("inter-procedural with transfer summaries", func(t *testing.T) {
		cg := core.NewCallGraph()

		cg.Edges["test.caller"] = []string{"test.get_input", "test.sink_func"}
		cg.Edges["test.get_input"] = []string{}
		cg.Edges["test.sink_func"] = []string{}

		cg.ReverseEdges = make(map[string][]string)
		cg.ReverseEdges["test.get_input"] = []string{"test.caller"}
		cg.ReverseEdges["test.sink_func"] = []string{"test.caller"}

		cg.CallSites["test.caller"] = []core.CallSite{
			{
				Target:    "get_input",
				TargetFQN: "test.get_input",
				Location:  core.Location{Line: 10, File: "test.py"},
				Resolved:  true,
			},
			{
				Target:    "sink_func",
				TargetFQN: "test.sink_func",
				Location:  core.Location{Line: 12, File: "test.py"},
				Arguments: []core.Argument{{Value: "x", IsVariable: true, Position: 0}},
				Resolved:  true,
			},
		}

		cg.CallSites["test.get_input"] = []core.CallSite{
			{
				Target:   "request.GET",
				Location: core.Location{Line: 5, File: "test.py"},
				Resolved: true,
			},
		}

		cg.CallSites["test.sink_func"] = []core.CallSite{
			{
				Target:    "eval",
				Location:  core.Location{Line: 20, File: "test.py"},
				Arguments: []core.Argument{{Value: "data", IsVariable: true, Position: 0}},
				Resolved:  true,
			},
		}

		cg.Statements["test.get_input"] = []*core.Statement{
			{Type: core.StatementTypeAssignment, Def: "data", Uses: []string{"request", "GET"}, CallTarget: "request.GET()", LineNumber: 5},
			{Type: core.StatementTypeReturn, Def: "", Uses: []string{"data"}, CallTarget: "data", LineNumber: 6},
		}

		cg.Statements["test.sink_func"] = []*core.Statement{
			{Type: core.StatementTypeCall, Def: "", Uses: []string{"eval", "data"}, CallTarget: "eval", LineNumber: 20},
		}

		cg.Statements["test.caller"] = []*core.Statement{
			{Type: core.StatementTypeAssignment, Def: "x", Uses: []string{"get_input"}, CallTarget: "get_input()", LineNumber: 10},
			{Type: core.StatementTypeCall, Def: "", Uses: []string{"sink_func", "x"}, CallTarget: "sink_func", LineNumber: 12},
		}

		cg.Functions["test.caller"] = nil
		cg.Functions["test.get_input"] = nil
		cg.Functions["test.sink_func"] = &graph.Node{MethodArgumentsValue: []string{"data"}}

		executor := NewDataflowExecutor(&DataflowIR{Scope: "global"}, cg)

		// Test buildTransferSummaries
		summaries := executor.buildTransferSummaries(
			[]string{"request.GET"},
			[]string{"eval"},
			[]string{},
		)

		assert.NotNil(t, summaries)
		// sink_func should have ParamToSink[0]=true
		if sinkSummary, ok := summaries["test.sink_func"]; ok {
			assert.True(t, sinkSummary.ParamToSink[0], "sink_func param 0 should reach sink")
		}
	})
}

func TestDataflowExecutor_VDG_Global_MultiLevelSource(t *testing.T) {
	// 3-level chain: main() -> wrapper() -> get_input()
	cg := core.NewCallGraph()

	cg.Edges["test.main"] = []string{"test.wrapper"}
	cg.Edges["test.wrapper"] = []string{"test.get_input"}
	cg.Edges["test.get_input"] = []string{}

	cg.ReverseEdges["test.wrapper"] = []string{"test.main"}
	cg.ReverseEdges["test.get_input"] = []string{"test.wrapper"}

	cg.Statements["test.get_input"] = []*core.Statement{
		{Type: core.StatementTypeAssignment, Def: "data", Uses: []string{"os", "getenv"}, CallTarget: "os.getenv", LineNumber: 2},
		{Type: core.StatementTypeReturn, Def: "", Uses: []string{"data"}, LineNumber: 3},
	}
	cg.CallSites["test.get_input"] = []core.CallSite{
		{Target: "os.getenv", TargetFQN: "os.getenv", Location: core.Location{Line: 2}},
	}

	cg.Statements["test.wrapper"] = []*core.Statement{
		{Type: core.StatementTypeAssignment, Def: "result", Uses: []string{"get_input"}, CallTarget: "get_input", LineNumber: 5},
		{Type: core.StatementTypeReturn, Def: "", Uses: []string{"result"}, LineNumber: 6},
	}
	cg.CallSites["test.wrapper"] = []core.CallSite{
		{Target: "get_input", TargetFQN: "test.get_input", Location: core.Location{Line: 5}},
	}

	cg.Statements["test.main"] = []*core.Statement{
		{Type: core.StatementTypeAssignment, Def: "x", Uses: []string{"wrapper"}, CallTarget: "wrapper", LineNumber: 10},
		{Type: core.StatementTypeCall, Def: "", Uses: []string{"eval", "x"}, CallTarget: "eval", LineNumber: 11},
	}
	cg.CallSites["test.main"] = []core.CallSite{
		{Target: "wrapper", TargetFQN: "test.wrapper", Location: core.Location{Line: 10}},
		{Target: "eval", TargetFQN: "builtins.eval", Location: core.Location{Line: 11}, Arguments: []core.Argument{{Value: "x", IsVariable: true, Position: 0}}},
	}

	cg.Functions["test.get_input"] = nil
	cg.Functions["test.wrapper"] = nil
	cg.Functions["test.main"] = nil

	executor := NewDataflowExecutor(&DataflowIR{Scope: "global"}, cg)

	// Verify transfer summaries propagate through chain
	summaries := executor.buildTransferSummaries(
		[]string{"os.getenv"},
		[]string{"eval"},
		[]string{},
	)
	assert.NotNil(t, summaries)

	// get_input should be IsSource (it calls os.getenv and returns result)
	if s, ok := summaries["test.get_input"]; ok {
		assert.True(t, s.ReturnTaintedBySource, "get_input should return tainted data")
	}
}

func TestDataflowExecutor_VDG_Global_MultiLevelSink(t *testing.T) {
	// 3-level sink chain: main() -> wrap_eval() -> dangerous_eval() -> eval()
	cg := core.NewCallGraph()

	cg.Edges["test.main"] = []string{"test.wrap_eval"}
	cg.Edges["test.wrap_eval"] = []string{"test.dangerous_eval"}
	cg.Edges["test.dangerous_eval"] = []string{}

	cg.ReverseEdges["test.wrap_eval"] = []string{"test.main"}
	cg.ReverseEdges["test.dangerous_eval"] = []string{"test.wrap_eval"}

	cg.Statements["test.dangerous_eval"] = []*core.Statement{
		{Type: core.StatementTypeCall, Def: "", Uses: []string{"eval", "code"}, CallTarget: "eval", LineNumber: 2},
	}
	cg.CallSites["test.dangerous_eval"] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 2}, Arguments: []core.Argument{{Value: "code", IsVariable: true, Position: 0}}},
	}

	cg.Statements["test.wrap_eval"] = []*core.Statement{
		{Type: core.StatementTypeCall, Def: "", Uses: []string{"dangerous_eval", "data"}, CallTarget: "dangerous_eval", LineNumber: 5},
	}
	cg.CallSites["test.wrap_eval"] = []core.CallSite{
		{Target: "dangerous_eval", TargetFQN: "test.dangerous_eval", Location: core.Location{Line: 5}, Arguments: []core.Argument{{Value: "data", IsVariable: true, Position: 0}}},
	}

	cg.Statements["test.main"] = []*core.Statement{
		{Type: core.StatementTypeAssignment, Def: "x", Uses: []string{"os", "getenv"}, CallTarget: "os.getenv", LineNumber: 10},
		{Type: core.StatementTypeCall, Def: "", Uses: []string{"wrap_eval", "x"}, CallTarget: "wrap_eval", LineNumber: 11},
	}
	cg.CallSites["test.main"] = []core.CallSite{
		{Target: "os.getenv", TargetFQN: "os.getenv", Location: core.Location{Line: 10}},
		{Target: "wrap_eval", TargetFQN: "test.wrap_eval", Location: core.Location{Line: 11}, Arguments: []core.Argument{{Value: "x", IsVariable: true, Position: 0}}},
	}

	cg.Functions["test.main"] = nil
	cg.Functions["test.wrap_eval"] = &graph.Node{MethodArgumentsValue: []string{"data"}}
	cg.Functions["test.dangerous_eval"] = &graph.Node{MethodArgumentsValue: []string{"code"}}

	executor := NewDataflowExecutor(&DataflowIR{Scope: "global"}, cg)

	summaries := executor.buildTransferSummaries(
		[]string{"os.getenv"},
		[]string{"eval"},
		[]string{},
	)

	// dangerous_eval should have ParamToSink[0]=true
	if s, ok := summaries["test.dangerous_eval"]; ok {
		assert.True(t, s.ParamToSink[0], "dangerous_eval param 0 should reach eval sink")
	}
}

// matchesPattern and findMatchingCalls tests removed — these dead methods
// were deleted from DataflowExecutor. Pattern matching is tested via
// CallMatcherExecutor tests; resolveMatchers is tested in bridge_test.go.

func TestDataflowExecutor_AttributeMatcherInFlows(t *testing.T) {
	cg := core.NewCallGraph()

	funcFQN := "test.proxy"
	cg.Functions[funcFQN] = &graph.Node{
		Name: "proxy",
		File: "test.py",
	}

	cg.Statements[funcFQN] = []*core.Statement{
		{
			Type:            core.StatementTypeAssignment,
			LineNumber:      uint32(2),
			Def:             "url",
			Uses:            []string{"request"},
			AttributeAccess: "request.url",
		},
		{
			Type:       core.StatementTypeCall,
			LineNumber: uint32(3),
			Def:        "response",
			Uses:       []string{"url"},
			CallTarget: "requests.get",
			CallArgs:   []string{"url"},
		},
	}

	cg.CallSites[funcFQN] = []core.CallSite{
		{
			Target:   "requests.get",
			Location: core.Location{File: "test.py", Line: 3},
		},
	}

	sourceJSON, _ := json.Marshal(AttributeMatcherIR{Type: "attribute_matcher", Patterns: []string{"request.url"}})
	sinkJSON, _ := json.Marshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"requests.get"}})

	ir := &DataflowIR{
		Sources:    []json.RawMessage{sourceJSON},
		Sinks:      []json.RawMessage{sinkJSON},
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)
	detections := executor.Execute()

	assert.NotEmpty(t, detections, "Should detect flow: request.url (attribute) -> requests.get (call)")
	if len(detections) > 0 {
		assert.Equal(t, funcFQN, detections[0].FunctionFQN)
		assert.Equal(t, 2, detections[0].SourceLine)
		assert.Equal(t, 3, detections[0].SinkLine)
	}
}
