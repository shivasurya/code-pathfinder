package dsl

import (
	"fmt"
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
	"github.com/stretchr/testify/assert"
)

func TestCallMatcherExecutor_Execute(t *testing.T) {
	// Setup test callgraph
	cg := callgraph.NewCallGraph()

	cg.CallSites["test.main"] = []callgraph.CallSite{
		{Target: "eval", Location: callgraph.Location{File: "test.py", Line: 10}},
		{Target: "exec", Location: callgraph.Location{File: "test.py", Line: 15}},
		{Target: "print", Location: callgraph.Location{File: "test.py", Line: 20}},
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
		cg2 := callgraph.NewCallGraph()
		cg2.CallSites["test.main"] = []callgraph.CallSite{
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
		cg2 := callgraph.NewCallGraph()
		cg2.CallSites["test.main"] = []callgraph.CallSite{
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
	cg := callgraph.NewCallGraph()
	cg.CallSites["test.process_data"] = []callgraph.CallSite{
		{Target: "eval", Location: callgraph.Location{File: "app.py", Line: 42}},
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
	cg := callgraph.NewCallGraph()
	for i := 0; i < 1000; i++ {
		funcName := fmt.Sprintf("module.func_%d", i)
		cg.CallSites[funcName] = []callgraph.CallSite{
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
