package dsl

import (
	"encoding/json"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

// buildTypedDataflowCallGraph creates a CallGraph simulating:
// - function "app.handler": calls request.GET (source) and cursor.execute() (typed sink)
// - function "app.process": calls task.execute() (same method, different type)
// - function "app.safe": calls request.GET + escape_html (sanitizer) + cursor.execute()
// - function "app.get_input" → "app.db_write": cross-function flow to typed sink
// - function "app.untyped": execute() with no type info.
func buildTypedDataflowCallGraph() *core.CallGraph {
	cg := core.NewCallGraph()
	cg.Edges = make(map[string][]string)

	cg.CallSites["app.handler"] = []core.CallSite{
		{
			Target:   "request.GET",
			Location: core.Location{File: "app.py", Line: 5},
		},
		{
			Target:                   "cursor.execute",
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.9,
			ResolvedViaTypeInference: true,
			Location:                 core.Location{File: "app.py", Line: 10},
		},
	}

	cg.CallSites["app.process"] = []core.CallSite{
		{
			Target:                   "task.execute",
			InferredType:             "celery.Task",
			TypeConfidence:           0.85,
			ResolvedViaTypeInference: true,
			Location:                 core.Location{File: "app.py", Line: 20},
		},
	}

	cg.CallSites["app.safe"] = []core.CallSite{
		{
			Target:   "request.GET",
			Location: core.Location{File: "app.py", Line: 30},
		},
		{
			Target:   "escape_html",
			Location: core.Location{File: "app.py", Line: 33},
		},
		{
			Target:                   "cursor.execute",
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.9,
			ResolvedViaTypeInference: true,
			Location:                 core.Location{File: "app.py", Line: 36},
		},
	}

	// For global scope tests
	cg.CallSites["app.get_input"] = []core.CallSite{
		{
			Target:   "request.POST",
			Location: core.Location{File: "app.py", Line: 50},
		},
	}

	cg.CallSites["app.db_write"] = []core.CallSite{
		{
			Target:                   "conn.execute",
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.8,
			ResolvedViaTypeInference: true,
			Location:                 core.Location{File: "app.py", Line: 60},
		},
	}
	cg.Edges["app.get_input"] = []string{"app.db_write"}

	// Untyped call for fallback tests
	cg.CallSites["app.untyped"] = []core.CallSite{
		{
			Target:   "request.GET",
			Location: core.Location{File: "app.py", Line: 70},
		},
		{
			Target:                   "cursor.execute",
			InferredType:             "",
			TypeConfidence:           0,
			ResolvedViaTypeInference: false,
			Location:                 core.Location{File: "app.py", Line: 75},
		},
	}

	return cg
}

// typeConstrainedMapFull creates a type_constrained_call matcher with all options
// (including minConfidence). Use the production typeConstrainedMap() for standard cases.
func typeConstrainedMapFull(receiverType, methodName string, minConfidence float64, fallbackMode string) map[string]any {
	return map[string]any{
		"type":          "type_constrained_call",
		"receiverType":  receiverType,
		"methodName":    methodName,
		"minConfidence": minConfidence,
		"fallbackMode":  fallbackMode,
	}
}

func TestDataflowExecutor_TypeConstrainedSink_Local(t *testing.T) {
	t.Run("detects flow to typed sink", func(t *testing.T) {
		cg := buildTypedDataflowCallGraph()
		ir := &DataflowIR{
			Sources:    []any{callMatcherMap("request.GET")},
			Sinks:      []any{typeConstrainedMap("Cursor", "execute", "name")},
			Sanitizers: []any{},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		// Should detect: app.handler (unsanitized) + app.safe (sanitized) + app.untyped (name fallback)
		handlerFound := false
		for _, d := range detections {
			if d.FunctionFQN == "app.handler" {
				handlerFound = true
				assert.Equal(t, 5, d.SourceLine)
				assert.Equal(t, 10, d.SinkLine)
				assert.Equal(t, "cursor.execute", d.SinkCall)
				assert.False(t, d.Sanitized)
			}
		}
		assert.True(t, handlerFound, "Should detect flow in app.handler")
	})

	t.Run("typed sink filters out wrong type", func(t *testing.T) {
		cg := buildTypedDataflowCallGraph()
		// Use Task type — should NOT match Cursor.execute in handler/safe
		ir := &DataflowIR{
			Sources:    []any{callMatcherMap("request.GET")},
			Sinks:      []any{typeConstrainedMapFull("Task", "execute", 0.5, "none")},
			Sanitizers: []any{},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		// With fallback=none, untyped calls are skipped.
		// Task.execute only exists in app.process which has no source.
		assert.Empty(t, detections, "Task.execute should not match Cursor.execute")
	})

	t.Run("sanitizer prevents detection with typed sink", func(t *testing.T) {
		cg := buildTypedDataflowCallGraph()
		ir := &DataflowIR{
			Sources:    []any{callMatcherMap("request.GET")},
			Sinks:      []any{typeConstrainedMap("Cursor", "execute", "name")},
			Sanitizers: []any{callMatcherMap("escape_html")},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		for _, d := range detections {
			if d.FunctionFQN == "app.safe" {
				assert.True(t, d.Sanitized, "app.safe should be sanitized")
			}
			if d.FunctionFQN == "app.handler" {
				assert.False(t, d.Sanitized, "app.handler has no sanitizer")
			}
		}
	})

	t.Run("mixed sinks: call_matcher + type_constrained_call", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.func"] = []core.CallSite{
			{Target: "request.GET", Location: core.Location{File: "t.py", Line: 1}},
			{Target: "eval", Location: core.Location{File: "t.py", Line: 5}},
			{
				Target: "cursor.execute", InferredType: "sqlite3.Cursor",
				TypeConfidence: 0.9, ResolvedViaTypeInference: true,
				Location: core.Location{File: "t.py", Line: 10},
			},
		}

		ir := &DataflowIR{
			Sources: []any{callMatcherMap("request.GET")},
			Sinks: []any{
				callMatcherMap("eval"),
				typeConstrainedMap("Cursor", "execute", "name"),
			},
			Sanitizers: []any{},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		// Should find 2 detections: one for eval, one for cursor.execute
		assert.Len(t, detections, 2)
	})

	t.Run("type_constrained_call as source", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.func"] = []core.CallSite{
			{
				Target: "request.get", InferredType: "flask.Request",
				TypeConfidence: 0.9, ResolvedViaTypeInference: true,
				Location: core.Location{File: "t.py", Line: 1},
			},
			{Target: "eval", Location: core.Location{File: "t.py", Line: 10}},
		}

		ir := &DataflowIR{
			Sources:    []any{typeConstrainedMap("Request", "get", "name")},
			Sinks:      []any{callMatcherMap("eval")},
			Sanitizers: []any{},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		assert.Len(t, detections, 1)
		assert.Equal(t, 1, detections[0].SourceLine)
		assert.Equal(t, 10, detections[0].SinkLine)
	})

	t.Run("fallback none skips untyped calls", func(t *testing.T) {
		cg := buildTypedDataflowCallGraph()
		ir := &DataflowIR{
			Sources:    []any{callMatcherMap("request.GET")},
			Sinks:      []any{typeConstrainedMapFull("Cursor", "execute", 0.5, "none")},
			Sanitizers: []any{},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		// Should NOT include app.untyped (fallback=none skips untyped)
		for _, d := range detections {
			assert.NotEqual(t, "app.untyped", d.FunctionFQN,
				"fallback=none should skip untyped functions")
		}
	})
}

func TestDataflowExecutor_TypeConstrainedSink_Global(t *testing.T) {
	t.Run("cross-function flow to typed sink", func(t *testing.T) {
		cg := buildTypedDataflowCallGraph()
		ir := &DataflowIR{
			Sources:    []any{callMatcherMap("request.POST")},
			Sinks:      []any{typeConstrainedMap("Cursor", "execute", "name")},
			Sanitizers: []any{},
			Scope:      "global",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		// Should detect cross-function flow: app.get_input → app.db_write
		found := false
		for _, d := range detections {
			if d.FunctionFQN == "app.get_input" && d.Scope == "global" {
				found = true
				assert.Equal(t, 50, d.SourceLine)
				assert.Equal(t, 60, d.SinkLine)
			}
		}
		assert.True(t, found, "Should detect cross-function flow to typed sink")
	})
}

func TestDataflowExecutor_LogicOperators(t *testing.T) {
	t.Run("logic_or sink unions call_matcher and type_constrained", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.func"] = []core.CallSite{
			{Target: "request.GET", Location: core.Location{File: "t.py", Line: 1}},
			{Target: "eval", Location: core.Location{File: "t.py", Line: 5}},
			{
				Target: "cursor.execute", InferredType: "sqlite3.Cursor",
				TypeConfidence: 0.9, ResolvedViaTypeInference: true,
				Location: core.Location{File: "t.py", Line: 10},
			},
		}

		ir := &DataflowIR{
			Sources: []any{callMatcherMap("request.GET")},
			Sinks: []any{
				map[string]any{
					"type": "logic_or",
					"matchers": []any{
						callMatcherMap("eval"),
						typeConstrainedMap("Cursor", "execute", "name"),
					},
				},
			},
			Sanitizers: []any{},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		assert.Len(t, detections, 2, "OR(eval, Cursor.execute) should find both")
	})

	t.Run("logic_or deduplicates matches", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.func"] = []core.CallSite{
			{Target: "request.GET", Location: core.Location{File: "t.py", Line: 1}},
			{Target: "eval", Location: core.Location{File: "t.py", Line: 5}},
		}

		ir := &DataflowIR{
			Sources: []any{callMatcherMap("request.GET")},
			Sinks: []any{
				map[string]any{
					"type": "logic_or",
					"matchers": []any{
						callMatcherMap("eval"),
						callMatcherMap("eval"), // duplicate
					},
				},
			},
			Sanitizers: []any{},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		assert.Len(t, detections, 1, "Duplicate eval in OR should be deduplicated")
	})

	t.Run("nested logic_or resolves recursively", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.func"] = []core.CallSite{
			{Target: "request.GET", Location: core.Location{File: "t.py", Line: 1}},
			{Target: "eval", Location: core.Location{File: "t.py", Line: 5}},
			{Target: "exec", Location: core.Location{File: "t.py", Line: 8}},
			{
				Target: "cursor.execute", InferredType: "sqlite3.Cursor",
				TypeConfidence: 0.9, ResolvedViaTypeInference: true,
				Location: core.Location{File: "t.py", Line: 10},
			},
		}

		ir := &DataflowIR{
			Sources: []any{callMatcherMap("request.GET")},
			Sinks: []any{
				map[string]any{
					"type": "logic_or",
					"matchers": []any{
						map[string]any{
							"type": "logic_or",
							"matchers": []any{
								callMatcherMap("eval"),
								callMatcherMap("exec"),
							},
						},
						typeConstrainedMap("Cursor", "execute", "name"),
					},
				},
			},
			Sanitizers: []any{},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		assert.Len(t, detections, 3, "Nested OR should find eval + exec + cursor.execute")
	})

	t.Run("logic_and intersects matchers", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.func"] = []core.CallSite{
			{Target: "request.GET", Location: core.Location{File: "t.py", Line: 1}},
			{Target: "eval", Location: core.Location{File: "t.py", Line: 5}},
		}

		ir := &DataflowIR{
			Sources: []any{callMatcherMap("request.GET")},
			Sinks: []any{
				map[string]any{
					"type": "logic_and",
					"matchers": []any{
						callMatcherMap("eval"),
						callMatcherMap("exec"), // exec doesn't exist
					},
				},
			},
			Sanitizers: []any{},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.Execute()

		assert.Empty(t, detections, "AND(eval, exec) should be empty since exec doesn't exist")
	})
}

func TestResolveMatcherToCallSites(t *testing.T) {
	t.Run("call_matcher type", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.func"] = []core.CallSite{
			{Target: "eval", Location: core.Location{File: "t.py", Line: 5}},
		}
		ir := &DataflowIR{Scope: "local"}
		executor := NewDataflowExecutor(ir, cg)

		matches := executor.resolveMatcherToCallSites(callMatcherMap("eval"))
		assert.Len(t, matches, 1)
		assert.Equal(t, "eval", matches[0].CallSite.Target)
	})

	t.Run("type_constrained_call type", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.func"] = []core.CallSite{
			{
				Target: "cursor.execute", InferredType: "sqlite3.Cursor",
				TypeConfidence: 0.9, ResolvedViaTypeInference: true,
				Location: core.Location{File: "t.py", Line: 10},
			},
		}
		ir := &DataflowIR{Scope: "local"}
		executor := NewDataflowExecutor(ir, cg)

		matches := executor.resolveMatcherToCallSites(typeConstrainedMap("Cursor", "execute", "name"))
		assert.Len(t, matches, 1)
		assert.Equal(t, "cursor.execute", matches[0].CallSite.Target)
	})

	t.Run("unknown type returns nil", func(t *testing.T) {
		cg := core.NewCallGraph()
		ir := &DataflowIR{Scope: "local"}
		executor := NewDataflowExecutor(ir, cg)

		matches := executor.resolveMatcherToCallSites(map[string]any{"type": "unknown_thing"})
		assert.Nil(t, matches)
	})

	t.Run("nil type defaults to call_matcher", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.func"] = []core.CallSite{
			{Target: "eval", Location: core.Location{File: "t.py", Line: 5}},
		}
		ir := &DataflowIR{Scope: "local"}
		executor := NewDataflowExecutor(ir, cg)

		// No "type" key → defaults to call_matcher.
		matches := executor.resolveMatcherToCallSites(map[string]any{
			"patterns": []any{"eval"},
		})
		assert.Len(t, matches, 1)
	})
}

func TestExtractPatternsFromMap(t *testing.T) {
	t.Run("extracts patterns from call_matcher map", func(t *testing.T) {
		m := callMatcherMap("eval", "exec")
		patterns := extractPatternsFromMap(m)
		assert.Equal(t, []string{"eval", "exec"}, patterns)
	})

	t.Run("returns nil for missing patterns key", func(t *testing.T) {
		patterns := extractPatternsFromMap(map[string]any{"type": "call_matcher"})
		assert.Nil(t, patterns)
	})

	t.Run("returns nil for wrong patterns type", func(t *testing.T) {
		patterns := extractPatternsFromMap(map[string]any{
			"type":     "call_matcher",
			"patterns": "not_a_list",
		})
		assert.Nil(t, patterns)
	})
}

func TestBackwardCompatibility_DataflowIR_JSON(t *testing.T) {
	t.Run("old format with call_matcher arrays deserializes correctly", func(t *testing.T) {
		jsonIR := `{
			"type": "dataflow",
			"sources": [{"type": "call_matcher", "patterns": ["request.GET"]}],
			"sinks": [{"type": "call_matcher", "patterns": ["eval"]}],
			"sanitizers": [],
			"scope": "local"
		}`

		var ir DataflowIR
		err := json.Unmarshal([]byte(jsonIR), &ir)
		assert.NoError(t, err)
		assert.Len(t, ir.Sources, 1)
		assert.Len(t, ir.Sinks, 1)
		assert.Equal(t, "local", ir.Scope)

		// Verify resolveMatcherToCallSites works with deserialized data.
		cg := core.NewCallGraph()
		cg.CallSites["test.func"] = []core.CallSite{
			{Target: "request.GET", Location: core.Location{File: "t.py", Line: 1}},
			{Target: "eval", Location: core.Location{File: "t.py", Line: 5}},
		}

		executor := NewDataflowExecutor(&ir, cg)
		detections := executor.Execute()
		assert.Len(t, detections, 1)
	})

	t.Run("new format with mixed matcher types deserializes", func(t *testing.T) {
		jsonIR := `{
			"type": "dataflow",
			"sources": [{"type": "call_matcher", "patterns": ["request.GET"]}],
			"sinks": [
				{"type": "call_matcher", "patterns": ["eval"]},
				{"type": "type_constrained_call", "receiverType": "Cursor", "methodName": "execute"}
			],
			"sanitizers": [],
			"scope": "local"
		}`

		var ir DataflowIR
		err := json.Unmarshal([]byte(jsonIR), &ir)
		assert.NoError(t, err)
		assert.Len(t, ir.Sources, 1)
		assert.Len(t, ir.Sinks, 2)
	})

	t.Run("logic_or sinks deserialize and execute", func(t *testing.T) {
		jsonIR := `{
			"type": "dataflow",
			"sources": [{"type": "call_matcher", "patterns": ["request.GET"]}],
			"sinks": [
				{
					"type": "logic_or",
					"matchers": [
						{"type": "call_matcher", "patterns": ["eval"]},
						{"type": "type_constrained_call", "receiverType": "Cursor", "methodName": "execute"}
					]
				}
			],
			"sanitizers": [],
			"scope": "local"
		}`

		var ir DataflowIR
		err := json.Unmarshal([]byte(jsonIR), &ir)
		assert.NoError(t, err)

		cg := core.NewCallGraph()
		cg.CallSites["test.func"] = []core.CallSite{
			{Target: "request.GET", Location: core.Location{File: "t.py", Line: 1}},
			{Target: "eval", Location: core.Location{File: "t.py", Line: 5}},
			{
				Target: "cursor.execute", InferredType: "sqlite3.Cursor",
				TypeConfidence: 0.9, ResolvedViaTypeInference: true,
				Location: core.Location{File: "t.py", Line: 10},
			},
		}

		executor := NewDataflowExecutor(&ir, cg)
		detections := executor.Execute()
		assert.Len(t, detections, 2, "OR(eval, Cursor.execute) should find both")
	})
}
