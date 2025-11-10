package dsl

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph/callgraph"
	"github.com/stretchr/testify/assert"
)

func TestDataflowExecutor_Local(t *testing.T) {
	t.Run("finds functions with sources and sinks", func(t *testing.T) {
		// Setup: Function with source and sink in same function
		cg := callgraph.NewCallGraph()
		cg.CallSites["test.vulnerable"] = []callgraph.CallSite{
			{
				Target:   "request.GET",
				Location: callgraph.Location{File: "test.py", Line: 10},
			},
			{
				Target:   "eval",
				Location: callgraph.Location{File: "test.py", Line: 15},
			},
		}

		ir := &DataflowIR{
			Sources:    []CallMatcherIR{{Patterns: []string{"request.GET"}}},
			Sinks:      []CallMatcherIR{{Patterns: []string{"eval"}}},
			Sanitizers: []CallMatcherIR{},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)

		// Test helper functions
		sourcePatterns := executor.extractPatterns(ir.Sources)
		sinkPatterns := executor.extractPatterns(ir.Sinks)

		sourceCalls := executor.findMatchingCalls(sourcePatterns)
		sinkCalls := executor.findMatchingCalls(sinkPatterns)

		functions := executor.findFunctionsWithSourcesAndSinks(sourceCalls, sinkCalls)

		assert.Contains(t, functions, "test.vulnerable")
	})
}

func TestDataflowExecutor_Global(t *testing.T) {
	t.Run("detects cross-function flow", func(t *testing.T) {
		// Setup: Source in func A, sink in func B, A calls B
		cg := callgraph.NewCallGraph()
		cg.Edges = make(map[string][]string)
		cg.Edges["test.get_input"] = []string{"test.process"}

		cg.CallSites["test.get_input"] = []callgraph.CallSite{
			{
				Target:   "request.GET",
				Location: callgraph.Location{Line: 10},
			},
			{
				Target:    "process",
				TargetFQN: "test.process",
				Location:  callgraph.Location{Line: 12},
			},
		}

		cg.CallSites["test.process"] = []callgraph.CallSite{
			{
				Target:   "eval",
				Location: callgraph.Location{Line: 20},
			},
		}

		ir := &DataflowIR{
			Sources:    []CallMatcherIR{{Patterns: []string{"request.GET"}}},
			Sinks:      []CallMatcherIR{{Patterns: []string{"eval"}}},
			Sanitizers: []CallMatcherIR{},
			Scope:      "global",
		}

		executor := NewDataflowExecutor(ir, cg)

		// Test path finding
		path := executor.findPath("test.get_input", "test.process")
		assert.NotEmpty(t, path)
		assert.Contains(t, path, "test.get_input")
		assert.Contains(t, path, "test.process")
	})

	t.Run("detects sanitizer on path", func(t *testing.T) {
		cg := callgraph.NewCallGraph()
		cg.Edges = make(map[string][]string)
		cg.Edges["test.source"] = []string{"test.sanitize"}
		cg.Edges["test.sanitize"] = []string{"test.sink"}

		cg.CallSites["test.sanitize"] = []callgraph.CallSite{
			{
				Target:   "escape_sql",
				Location: callgraph.Location{Line: 15},
			},
		}

		ir := &DataflowIR{
			Sources:    []CallMatcherIR{{Patterns: []string{"request.GET"}}},
			Sinks:      []CallMatcherIR{{Patterns: []string{"eval"}}},
			Sanitizers: []CallMatcherIR{{Patterns: []string{"escape_sql"}}},
			Scope:      "global",
		}

		executor := NewDataflowExecutor(ir, cg)

		path := []string{"test.source", "test.sanitize", "test.sink"}
		sanitizerPatterns := executor.extractPatterns(ir.Sanitizers)
		sanitizerCalls := executor.findMatchingCalls(sanitizerPatterns)

		hasSanitizer := executor.pathHasSanitizer(path, sanitizerCalls)
		assert.True(t, hasSanitizer)
	})
}

func TestDataflowExecutor_PatternMatching(t *testing.T) {
	cg := callgraph.NewCallGraph()
	ir := &DataflowIR{}
	executor := NewDataflowExecutor(ir, cg)

	t.Run("exact match", func(t *testing.T) {
		assert.True(t, executor.matchesPattern("eval", "eval"))
		assert.False(t, executor.matchesPattern("eval", "exec"))
	})

	t.Run("wildcard prefix", func(t *testing.T) {
		assert.True(t, executor.matchesPattern("request.GET", "request.*"))
		assert.True(t, executor.matchesPattern("request.POST", "request.*"))
		assert.False(t, executor.matchesPattern("utils.sanitize", "request.*"))
	})

	t.Run("wildcard suffix", func(t *testing.T) {
		assert.True(t, executor.matchesPattern("user_input", "*_input"))
		assert.True(t, executor.matchesPattern("admin_input", "*_input"))
		assert.False(t, executor.matchesPattern("user_data", "*_input"))
	})

	t.Run("wildcard match all", func(t *testing.T) {
		assert.True(t, executor.matchesPattern("anything", "*"))
	})
}
