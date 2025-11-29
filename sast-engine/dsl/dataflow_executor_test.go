package dsl

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func TestDataflowExecutor_Local(t *testing.T) {
	t.Run("finds functions with sources and sinks", func(t *testing.T) {
		// Setup: Function with source and sink in same function
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
			Sources:    []CallMatcherIR{{Patterns: []string{"request.POST"}}},
			Sinks:      []CallMatcherIR{{Patterns: []string{"execute"}}},
			Sanitizers: []CallMatcherIR{},
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
			{
				Target:   "request.GET",
				Location: core.Location{File: "test.py", Line: 5},
			},
			{
				Target:   "escape_sql",
				Location: core.Location{File: "test.py", Line: 8},
			},
			{
				Target:   "execute",
				Location: core.Location{File: "test.py", Line: 12},
			},
		}

		ir := &DataflowIR{
			Sources:    []CallMatcherIR{{Patterns: []string{"request.GET"}}},
			Sinks:      []CallMatcherIR{{Patterns: []string{"execute"}}},
			Sanitizers: []CallMatcherIR{{Patterns: []string{"escape_sql"}}},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.executeLocal()

		assert.Len(t, detections, 1)
		assert.True(t, detections[0].Sanitized, "Should detect sanitizer between source and sink")
	})

	t.Run("detects sanitizer in reverse order (sink before source)", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.reverse"] = []core.CallSite{
			{
				Target:   "execute",
				Location: core.Location{File: "test.py", Line: 5},
			},
			{
				Target:   "escape_sql",
				Location: core.Location{File: "test.py", Line: 8},
			},
			{
				Target:   "request.GET",
				Location: core.Location{File: "test.py", Line: 12},
			},
		}

		ir := &DataflowIR{
			Sources:    []CallMatcherIR{{Patterns: []string{"request.GET"}}},
			Sinks:      []CallMatcherIR{{Patterns: []string{"execute"}}},
			Sanitizers: []CallMatcherIR{{Patterns: []string{"escape_sql"}}},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.executeLocal()

		assert.Len(t, detections, 1)
		assert.True(t, detections[0].Sanitized, "Should detect sanitizer even when sink appears before source")
	})

	t.Run("ignores cross-function flows in local scope", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.CallSites["test.func1"] = []core.CallSite{
			{
				Target:   "request.GET",
				Location: core.Location{File: "test.py", Line: 5},
			},
		}
		cg.CallSites["test.func2"] = []core.CallSite{
			{
				Target:   "eval",
				Location: core.Location{File: "test.py", Line: 15},
			},
		}

		ir := &DataflowIR{
			Sources:    []CallMatcherIR{{Patterns: []string{"request.GET"}}},
			Sinks:      []CallMatcherIR{{Patterns: []string{"eval"}}},
			Sanitizers: []CallMatcherIR{},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.executeLocal()

		assert.Empty(t, detections, "Local scope should not detect cross-function flows")
	})

	t.Run("handles multiple sources and sinks in same function", func(t *testing.T) {
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
			Sources:    []CallMatcherIR{{Patterns: []string{"request.GET", "request.POST"}}},
			Sinks:      []CallMatcherIR{{Patterns: []string{"eval", "execute"}}},
			Sanitizers: []CallMatcherIR{},
			Scope:      "local",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.executeLocal()

		// Should find 2 sources * 2 sinks = 4 combinations
		assert.Len(t, detections, 4)
	})
}

func TestDataflowExecutor_Global(t *testing.T) {
	t.Run("detects cross-function flow", func(t *testing.T) {
		// Setup: Source in func A, sink in func B, A calls B
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

	t.Run("executes global analysis and finds cross-function flows", func(t *testing.T) {
		// Setup: Source in func A, sink in func B, A calls B
		cg := core.NewCallGraph()
		cg.Edges = make(map[string][]string)
		cg.Edges["test.source_func"] = []string{"test.sink_func"}

		cg.CallSites["test.source_func"] = []core.CallSite{
			{
				Target:   "request.GET",
				Location: core.Location{Line: 10, File: "test.py"},
			},
		}

		cg.CallSites["test.sink_func"] = []core.CallSite{
			{
				Target:   "eval",
				Location: core.Location{Line: 20, File: "test.py"},
			},
		}

		ir := &DataflowIR{
			Sources:    []CallMatcherIR{{Patterns: []string{"request.GET"}}},
			Sinks:      []CallMatcherIR{{Patterns: []string{"eval"}}},
			Sanitizers: []CallMatcherIR{},
			Scope:      "global",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.executeGlobal()

		// Should detect cross-function flow
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

	t.Run("detects sanitizer on path", func(t *testing.T) {
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

	t.Run("excludes flows with sanitizer on path", func(t *testing.T) {
		cg := core.NewCallGraph()
		cg.Edges = make(map[string][]string)
		cg.Edges["test.source"] = []string{"test.sanitize"}
		cg.Edges["test.sanitize"] = []string{"test.sink"}

		cg.CallSites["test.source"] = []core.CallSite{
			{
				Target:   "request.POST",
				Location: core.Location{Line: 5, File: "test.py"},
			},
		}

		cg.CallSites["test.sanitize"] = []core.CallSite{
			{
				Target:   "escape_html",
				Location: core.Location{Line: 10, File: "test.py"},
			},
		}

		cg.CallSites["test.sink"] = []core.CallSite{
			{
				Target:   "render",
				Location: core.Location{Line: 15, File: "test.py"},
			},
		}

		ir := &DataflowIR{
			Sources:    []CallMatcherIR{{Patterns: []string{"request.POST"}}},
			Sinks:      []CallMatcherIR{{Patterns: []string{"render"}}},
			Sanitizers: []CallMatcherIR{{Patterns: []string{"escape_html"}}},
			Scope:      "global",
		}

		executor := NewDataflowExecutor(ir, cg)
		detections := executor.executeGlobal()

		// Should NOT detect because sanitizer is on the path
		globalDetections := []DataflowDetection{}
		for _, d := range detections {
			if d.Scope == "global" {
				globalDetections = append(globalDetections, d)
			}
		}
		assert.Empty(t, globalDetections, "Should not detect flows with sanitizer on path")
	})
}

func TestDataflowExecutor_PatternMatching(t *testing.T) {
	cg := core.NewCallGraph()
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
