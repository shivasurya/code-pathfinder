package dsl

import (
	"encoding/json"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

// buildTypedCallGraph creates a test CallGraph with type inference metadata populated.
func buildTypedCallGraph() *core.CallGraph {
	cg := core.NewCallGraph()

	cg.CallSites["app.views.handle_request"] = []core.CallSite{
		{
			Target:                   "cursor.execute",
			Location:                 core.Location{File: "app/views.py", Line: 8},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.9,
			TypeSource:               "class_instantiation",
		},
		{
			Target:                   "queue.execute",
			Location:                 core.Location{File: "app/views.py", Line: 11},
			ResolvedViaTypeInference: true,
			InferredType:             "app.tasks.TaskQueue",
			TypeConfidence:           0.95,
			TypeSource:               "class_instantiation",
		},
		{
			Target:                   "pool.execute",
			Location:                 core.Location{File: "app/views.py", Line: 14},
			ResolvedViaTypeInference: true,
			InferredType:             "concurrent.futures.ThreadPoolExecutor",
			TypeConfidence:           0.85,
			TypeSource:               "function_call",
		},
	}

	return cg
}

func TestTypeConstrainedExecutor_TypeFilteringCorrectly(t *testing.T) {
	cg := buildTypedCallGraph()

	ir := &TypeConstrainedCallIR{
		Type:          "type_constrained_call",
		ReceiverType:  "Cursor",
		MethodName:    "execute",
		MinConfidence: 0.5,
		FallbackMode:  "none",
	}

	executor := NewTypeConstrainedCallExecutor(ir, cg)
	matches := executor.Execute()

	// Only sqlite3.Cursor should match, not TaskQueue or ThreadPoolExecutor
	assert.Len(t, matches, 1, "should match only the Cursor.execute call")
	assert.Equal(t, "cursor.execute", matches[0].Target)
	assert.Equal(t, "sqlite3.Cursor", matches[0].InferredType)
	assert.Equal(t, 8, matches[0].Location.Line)
}

func TestTypeConstrainedExecutor_ConfidenceThreshold(t *testing.T) {
	cg := core.NewCallGraph()

	cg.CallSites["app.views.process"] = []core.CallSite{
		{
			Target:                   "cursor.execute",
			Location:                 core.Location{File: "app.py", Line: 5},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.3, // Below threshold
			TypeSource:               "heuristic",
		},
		{
			Target:                   "cursor.execute",
			Location:                 core.Location{File: "app.py", Line: 10},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.8, // Above threshold
			TypeSource:               "class_instantiation",
		},
	}

	ir := &TypeConstrainedCallIR{
		Type:          "type_constrained_call",
		ReceiverType:  "Cursor",
		MethodName:    "execute",
		MinConfidence: 0.5,
		FallbackMode:  "none",
	}

	executor := NewTypeConstrainedCallExecutor(ir, cg)
	matches := executor.Execute()

	assert.Len(t, matches, 1, "should only match the high-confidence call")
	assert.Equal(t, 10, matches[0].Location.Line)
	assert.Equal(t, float32(0.8), matches[0].TypeConfidence)
}

func TestTypeConstrainedExecutor_FallbackModes(t *testing.T) {
	cg := core.NewCallGraph()

	// Call site with NO type inference
	cg.CallSites["app.views.mystery"] = []core.CallSite{
		{
			Target:                   "obj.execute",
			Location:                 core.Location{File: "app.py", Line: 20},
			ResolvedViaTypeInference: false,
			InferredType:             "",
			TypeConfidence:           0,
		},
	}

	t.Run("fallback=name matches without type info", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			Type:          "type_constrained_call",
			ReceiverType:  "Cursor",
			MethodName:    "execute",
			MinConfidence: 0.5,
			FallbackMode:  "name",
		}

		executor := NewTypeConstrainedCallExecutor(ir, cg)
		matches := executor.Execute()

		assert.Len(t, matches, 1, "fallback=name should match even without type info")
	})

	t.Run("fallback=none skips without type info", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			Type:          "type_constrained_call",
			ReceiverType:  "Cursor",
			MethodName:    "execute",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		}

		executor := NewTypeConstrainedCallExecutor(ir, cg)
		matches := executor.Execute()

		assert.Len(t, matches, 0, "fallback=none should not match without type info")
	})

	t.Run("fallback=warn matches without type info", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			Type:          "type_constrained_call",
			ReceiverType:  "Cursor",
			MethodName:    "execute",
			MinConfidence: 0.5,
			FallbackMode:  "warn",
		}

		executor := NewTypeConstrainedCallExecutor(ir, cg)
		matches := executor.Execute()

		assert.Len(t, matches, 1, "fallback=warn should match (with warning)")
	})
}

func TestTypeConstrainedExecutor_TypeMatchingModes(t *testing.T) {
	cg := core.NewCallGraph()

	cg.CallSites["app.db.query"] = []core.CallSite{
		{
			Target:                   "cursor.execute",
			Location:                 core.Location{File: "app.py", Line: 5},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.9,
		},
	}

	t.Run("exact FQN match", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType:  "sqlite3.Cursor",
			MethodName:    "execute",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		}
		executor := NewTypeConstrainedCallExecutor(ir, cg)
		assert.Len(t, executor.Execute(), 1, "exact FQN should match")
	})

	t.Run("short name suffix match", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType:  "Cursor",
			MethodName:    "execute",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		}
		executor := NewTypeConstrainedCallExecutor(ir, cg)
		assert.Len(t, executor.Execute(), 1, "short name Cursor should match sqlite3.Cursor")
	})

	t.Run("wildcard prefix match", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType:  "*Cursor",
			MethodName:    "execute",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		}
		executor := NewTypeConstrainedCallExecutor(ir, cg)
		assert.Len(t, executor.Execute(), 1, "*Cursor should match sqlite3.Cursor")
	})

	t.Run("wildcard suffix match", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType:  "sqlite3.*",
			MethodName:    "execute",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		}
		executor := NewTypeConstrainedCallExecutor(ir, cg)
		assert.Len(t, executor.Execute(), 1, "sqlite3.* should match sqlite3.Cursor")
	})

	t.Run("wrong type does not match", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType:  "Connection",
			MethodName:    "execute",
			MinConfidence: 0.5,
			FallbackMode:  "none",
		}
		executor := NewTypeConstrainedCallExecutor(ir, cg)
		assert.Len(t, executor.Execute(), 0, "Connection should not match sqlite3.Cursor")
	})
}

func TestTypeConstrainedExecutor_LoaderIntegration(t *testing.T) {
	cg := buildTypedCallGraph()

	// Simulate JSON IR as it would come from Python SDK
	matcherJSON := `{
		"type": "type_constrained_call",
		"receiverType": "Cursor",
		"methodName": "execute",
		"minConfidence": 0.5,
		"fallbackMode": "none"
	}`

	var matcherMap map[string]any
	err := json.Unmarshal([]byte(matcherJSON), &matcherMap)
	assert.NoError(t, err)

	// Marshal/unmarshal through the same path as loader
	jsonBytes, err := json.Marshal(matcherMap)
	assert.NoError(t, err)

	var ir TypeConstrainedCallIR
	err = json.Unmarshal(jsonBytes, &ir)
	assert.NoError(t, err)

	assert.Equal(t, "type_constrained_call", ir.Type)
	assert.Equal(t, "Cursor", ir.ReceiverType)
	assert.Equal(t, "execute", ir.MethodName)
	assert.Equal(t, 0.5, ir.MinConfidence)
	assert.Equal(t, "none", ir.FallbackMode)

	// Execute
	executor := NewTypeConstrainedCallExecutor(&ir, cg)
	matches := executor.ExecuteWithContext()

	assert.Len(t, matches, 1, "loader integration should produce 1 match")
	assert.Equal(t, "cursor.execute", matches[0].CallSite.Target)
	assert.Equal(t, "app.views.handle_request", matches[0].FunctionFQN)
	assert.Equal(t, "Cursor.execute", matches[0].MatchedBy)
}

func TestTypeConstrainedExecutor_MethodNameExtraction(t *testing.T) {
	cg := core.NewCallGraph()

	cg.CallSites["app.main"] = []core.CallSite{
		{
			Target:                   "execute", // bare function call, no dot
			Location:                 core.Location{File: "app.py", Line: 1},
			ResolvedViaTypeInference: false,
		},
		{
			Target:                   "db.cursor.execute", // deep chain
			Location:                 core.Location{File: "app.py", Line: 2},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.85,
		},
	}

	ir := &TypeConstrainedCallIR{
		ReceiverType:  "Cursor",
		MethodName:    "execute",
		MinConfidence: 0.5,
		FallbackMode:  "name",
	}

	executor := NewTypeConstrainedCallExecutor(ir, cg)
	matches := executor.Execute()

	// Both should match: bare "execute" via fallback, "db.cursor.execute" via type
	assert.Len(t, matches, 2, "should match both bare and dotted calls")
}
