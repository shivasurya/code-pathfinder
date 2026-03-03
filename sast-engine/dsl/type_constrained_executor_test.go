package dsl

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildTypedCallGraph creates a CallGraph with typed call sites for testing.
// Simulates 3 classes (DatabaseCursor, TaskQueue, ThreadPool) each with execute().
func buildTypedCallGraph() *core.CallGraph {
	cg := core.NewCallGraph()

	cg.CallSites["app.views.handle_request"] = []core.CallSite{
		{
			Target:                   "cursor.execute",
			Location:                 core.Location{File: "views.py", Line: 10},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.95,
			TypeSource:               "class_instantiation",
		},
		{
			Target:                   "cursor.execute",
			Location:                 core.Location{File: "views.py", Line: 15},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Cursor",
			TypeConfidence:           0.95,
			TypeSource:               "class_instantiation",
		},
		{
			Target:                   "queue.execute",
			Location:                 core.Location{File: "views.py", Line: 20},
			ResolvedViaTypeInference: true,
			InferredType:             "celery.TaskQueue",
			TypeConfidence:           0.90,
			TypeSource:               "class_instantiation",
		},
		{
			Target:                   "pool.execute",
			Location:                 core.Location{File: "views.py", Line: 25},
			ResolvedViaTypeInference: true,
			InferredType:             "concurrent.ThreadPool",
			TypeConfidence:           0.85,
			TypeSource:               "return_type",
		},
		{
			Target:                   "unknown.execute",
			Location:                 core.Location{File: "views.py", Line: 30},
			ResolvedViaTypeInference: false,
			InferredType:             "",
			TypeConfidence:           0.0,
		},
		{
			Target:                   "conn.close",
			Location:                 core.Location{File: "views.py", Line: 35},
			ResolvedViaTypeInference: true,
			InferredType:             "sqlite3.Connection",
			TypeConfidence:           0.90,
			TypeSource:               "class_instantiation",
		},
	}

	return cg
}

func TestNewTypeConstrainedCallExecutor(t *testing.T) {
	cg := core.NewCallGraph()

	t.Run("creates executor with valid IR", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "Cursor",
			MethodName:   "execute",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)
		assert.NotNil(t, executor)
	})

	t.Run("returns error for empty ReceiverType", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "",
			MethodName:   "execute",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		assert.Error(t, err)
		assert.Nil(t, executor)
		assert.Contains(t, err.Error(), "receiverType must not be empty")
	})

	t.Run("returns error for empty MethodName", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "Cursor",
			MethodName:   "",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		assert.Error(t, err)
		assert.Nil(t, executor)
		assert.Contains(t, err.Error(), "methodName must not be empty")
	})

	t.Run("sets default MinConfidence", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "Cursor",
			MethodName:   "execute",
		}
		_, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)
		assert.Equal(t, 0.5, ir.MinConfidence)
	})

	t.Run("sets default FallbackMode to name", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "Cursor",
			MethodName:   "execute",
		}
		_, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)
		assert.Equal(t, "name", ir.FallbackMode)
	})

	t.Run("corrects invalid FallbackMode to name", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "Cursor",
			MethodName:   "execute",
			FallbackMode: "invalid_mode",
		}
		_, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)
		assert.Equal(t, "name", ir.FallbackMode)
	})
}

func TestTypeConstrainedCallExecutor_Execute(t *testing.T) {
	cg := buildTypedCallGraph()

	t.Run("filters by receiver type - exact FQN match", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "sqlite3.Cursor",
			MethodName:   "execute",
			FallbackMode: "none",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)

		results := executor.ExecuteWithContext()
		assert.Len(t, results, 2, "should match 2 sqlite3.Cursor.execute calls")
	})

	t.Run("filters by receiver type - short name match", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "Cursor",
			MethodName:   "execute",
			FallbackMode: "none",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)

		results := executor.ExecuteWithContext()
		assert.Len(t, results, 2, "short name 'Cursor' should match 'sqlite3.Cursor'")
	})

	t.Run("filters by receiver type - wildcard prefix", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "*Cursor",
			MethodName:   "execute",
			FallbackMode: "none",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)

		results := executor.ExecuteWithContext()
		assert.Len(t, results, 2, "*Cursor should match sqlite3.Cursor")
	})

	t.Run("filters by receiver type - contains wildcard", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "*Thread*",
			MethodName:   "execute",
			FallbackMode: "none",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)

		results := executor.ExecuteWithContext()
		assert.Len(t, results, 1, "*Thread* should match concurrent.ThreadPool")
	})

	t.Run("does not match wrong type", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "Redis",
			MethodName:   "execute",
			FallbackMode: "none",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)

		results := executor.ExecuteWithContext()
		assert.Empty(t, results, "Redis type should not match any calls")
	})

	t.Run("does not match wrong method", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "Cursor",
			MethodName:   "fetchall",
			FallbackMode: "none",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)

		results := executor.ExecuteWithContext()
		assert.Empty(t, results, "fetchall method should not match execute calls")
	})

	t.Run("confidence threshold - filters low confidence", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType:  "ThreadPool",
			MethodName:    "execute",
			MinConfidence: 0.90,
			FallbackMode:  "none",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)

		results := executor.ExecuteWithContext()
		assert.Empty(t, results, "0.85 confidence should be filtered by 0.90 threshold")
	})

	t.Run("confidence threshold - allows high confidence", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType:  "Cursor",
			MethodName:    "execute",
			MinConfidence: 0.90,
			FallbackMode:  "none",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)

		results := executor.ExecuteWithContext()
		assert.Len(t, results, 2, "0.95 confidence should pass 0.90 threshold")
	})

	t.Run("fallback mode name - includes untyped calls", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "Cursor",
			MethodName:   "execute",
			FallbackMode: "name",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)

		results := executor.Execute()
		// 2 typed Cursor matches + 1 untyped (unknown.execute) falling back to name match
		assert.Len(t, results, 3, "name fallback should include untyped execute call")
	})

	t.Run("fallback mode none - excludes untyped calls", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "Cursor",
			MethodName:   "execute",
			FallbackMode: "none",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)

		results := executor.Execute()
		assert.Len(t, results, 2, "none fallback should exclude untyped call")
	})

	t.Run("fallback mode warn - excludes untyped calls", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "Cursor",
			MethodName:   "execute",
			FallbackMode: "warn",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)

		results := executor.Execute()
		assert.Len(t, results, 2, "warn fallback should exclude untyped call (same as none)")
	})

	t.Run("nil CallGraph returns empty results", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "Cursor",
			MethodName:   "execute",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, nil)
		require.NoError(t, err)

		results := executor.Execute()
		assert.Empty(t, results)
	})

	t.Run("nil CallGraph returns empty from ExecuteWithContext", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "Cursor",
			MethodName:   "execute",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, nil)
		require.NoError(t, err)

		results := executor.ExecuteWithContext()
		assert.Empty(t, results)
	})

	t.Run("empty CallGraph returns empty results", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "Cursor",
			MethodName:   "execute",
		}
		emptyCG := core.NewCallGraph()
		executor, err := NewTypeConstrainedCallExecutor(ir, emptyCG)
		require.NoError(t, err)

		results := executor.Execute()
		assert.Empty(t, results)
	})

	t.Run("ExecuteWithContext result contains correct metadata", func(t *testing.T) {
		ir := &TypeConstrainedCallIR{
			ReceiverType: "sqlite3.Cursor",
			MethodName:   "execute",
			FallbackMode: "none",
		}
		executor, err := NewTypeConstrainedCallExecutor(ir, cg)
		require.NoError(t, err)

		results := executor.ExecuteWithContext()
		require.Len(t, results, 2)

		// Check metadata on first result
		assert.Equal(t, "views.py", results[0].SourceFile)
		assert.Equal(t, "app.views.handle_request", results[0].FunctionFQN)
		assert.Equal(t, "sqlite3.Cursor.execute", results[0].MatchedBy)
	})
}

func TestMatchesReceiverType(t *testing.T) {
	tests := []struct {
		name         string
		inferredType string
		pattern      string
		expected     bool
	}{
		// Exact match
		{"exact match", "Cursor", "Cursor", true},
		{"exact FQN match", "sqlite3.Cursor", "sqlite3.Cursor", true},
		{"exact mismatch", "Cursor", "Connection", false},

		// Short name match
		{"short name matches FQN", "sqlite3.Cursor", "Cursor", true},
		{"short name matches deep FQN", "a.b.c.d.e.Cursor", "Cursor", true},
		{"short name mismatch", "sqlite3.Connection", "Cursor", false},

		// Wildcard prefix (*suffix)
		{"wildcard prefix", "sqlite3.Cursor", "*Cursor", true},
		{"wildcard prefix on bare name", "MySQLCursor", "*Cursor", true},
		{"wildcard prefix mismatch", "sqlite3.Connection", "*Cursor", false},

		// Wildcard suffix (prefix*)
		{"wildcard suffix", "CursorWrapper", "Cursor*", true},
		{"wildcard suffix FQN", "sqlite3.CursorPool", "sqlite3.Cursor*", true},
		{"wildcard suffix mismatch", "Connection", "Cursor*", false},

		// Contains wildcard (*substr*)
		{"contains wildcard", "DatabaseConnection", "*Connection*", true},
		{"contains wildcard FQN", "myapp.db.AsyncConnection", "*Connection*", true},
		{"contains wildcard mismatch", "DatabaseCursor", "*Connection*", false},

		// Edge cases
		{"empty inferred type", "", "Cursor", false},
		{"empty pattern", "Cursor", "", false},
		{"both empty", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesReceiverType(tt.inferredType, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTypeConstrainedCallExecutor_MatchesMethod(t *testing.T) {
	cg := core.NewCallGraph()
	ir := &TypeConstrainedCallIR{
		ReceiverType: "Cursor",
		MethodName:   "execute",
	}
	executor, err := NewTypeConstrainedCallExecutor(ir, cg)
	require.NoError(t, err)

	tests := []struct {
		name     string
		target   string
		expected bool
	}{
		{"bare method name", "execute", true},
		{"dotted target", "cursor.execute", true},
		{"deep dotted target", "db.conn.cursor.execute", true},
		{"wrong method", "fetchall", false},
		{"wrong dotted method", "cursor.fetchall", false},
		{"empty target", "", false},
		{"partial name match should fail", "execute_many", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.matchesMethod(tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}
