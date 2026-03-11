package dsl

import (
	"encoding/json"
	"testing"

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
		assert.Equal(t, 0.7, detections[0].Confidence)
		assert.False(t, detections[0].Sanitized)
	})

	t.Run("detects sanitizer between source and sink", func(t *testing.T) {
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

		assert.Len(t, detections, 1)
		assert.True(t, detections[0].Sanitized, "Should detect sanitizer between source and sink")
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
			if d.FunctionFQN == "test.source_func" && d.Scope == "global" {
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

// --- pathHasSanitizer edge cases ---

func TestPathHasSanitizer_EmptyPath(t *testing.T) {
	cg := core.NewCallGraph()
	ir := &DataflowIR{}
	executor := NewDataflowExecutor(ir, cg)

	sanitizers := []CallSiteMatch{
		{FunctionFQN: "test.sanitize", Line: 10},
	}

	result := executor.pathHasSanitizer([]string{}, sanitizers)
	assert.False(t, result, "Empty path should return false")
}

func TestPathHasSanitizer_EmptySanitizers(t *testing.T) {
	cg := core.NewCallGraph()
	ir := &DataflowIR{}
	executor := NewDataflowExecutor(ir, cg)

	result := executor.pathHasSanitizer([]string{"test.a", "test.b"}, []CallSiteMatch{})
	assert.False(t, result, "Empty sanitizers should return false")
}

// matchesPattern edge case tests removed — dead code deleted from DataflowExecutor.
