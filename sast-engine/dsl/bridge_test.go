package dsl

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

func TestExtractTargetPatterns_Basic(t *testing.T) {
	executor := NewDataflowExecutor(&DataflowIR{}, core.NewCallGraph())

	matches := []CallSiteMatch{
		{CallSite: core.CallSite{Target: "eval", TargetFQN: "builtins.eval"}, FunctionFQN: "main.foo", Line: 1},
		{CallSite: core.CallSite{Target: "exec", TargetFQN: ""}, FunctionFQN: "main.bar", Line: 2},
	}

	patterns := executor.extractTargetPatterns(matches)

	expected := map[string]bool{"eval": true, "builtins.eval": true, "exec": true}
	for _, p := range patterns {
		if !expected[p] {
			t.Errorf("unexpected pattern: %q", p)
		}
		delete(expected, p)
	}
	for missing := range expected {
		t.Errorf("missing expected pattern: %q", missing)
	}
}

func TestExtractTargetPatterns_DottedTarget(t *testing.T) {
	executor := NewDataflowExecutor(&DataflowIR{}, core.NewCallGraph())

	matches := []CallSiteMatch{
		{CallSite: core.CallSite{Target: "cursor.execute"}, FunctionFQN: "main.foo", Line: 1},
	}

	patterns := executor.extractTargetPatterns(matches)

	// Should include "cursor.execute" and bare name "execute"
	found := map[string]bool{}
	for _, p := range patterns {
		found[p] = true
	}
	if !found["cursor.execute"] {
		t.Error("expected 'cursor.execute' in patterns")
	}
	if !found["execute"] {
		t.Error("expected bare name 'execute' in patterns")
	}
}

func TestExtractTargetPatterns_Deduplication(t *testing.T) {
	executor := NewDataflowExecutor(&DataflowIR{}, core.NewCallGraph())

	matches := []CallSiteMatch{
		{CallSite: core.CallSite{Target: "eval"}, FunctionFQN: "main.foo", Line: 1},
		{CallSite: core.CallSite{Target: "eval"}, FunctionFQN: "main.bar", Line: 2},
	}

	patterns := executor.extractTargetPatterns(matches)

	if len(patterns) != 1 {
		t.Errorf("expected 1 pattern after dedup, got %d: %v", len(patterns), patterns)
	}
}

func TestExtractTargetPatterns_Empty(t *testing.T) {
	executor := NewDataflowExecutor(&DataflowIR{}, core.NewCallGraph())

	patterns := executor.extractTargetPatterns(nil)

	if len(patterns) != 0 {
		t.Errorf("expected 0 patterns for nil input, got %d", len(patterns))
	}
}

func TestFindFunctionsWithSourcesAndSinks_Intersection(t *testing.T) {
	executor := NewDataflowExecutor(&DataflowIR{}, core.NewCallGraph())

	sources := []CallSiteMatch{
		{FunctionFQN: "pkg.funcA", Line: 1},
		{FunctionFQN: "pkg.funcB", Line: 2},
	}
	sinks := []CallSiteMatch{
		{FunctionFQN: "pkg.funcB", Line: 5},
		{FunctionFQN: "pkg.funcC", Line: 3},
	}

	result := executor.findFunctionsWithSourcesAndSinks(sources, sinks)

	if len(result) != 1 || result[0] != "pkg.funcB" {
		t.Errorf("expected [pkg.funcB], got %v", result)
	}
}

func TestFindFunctionsWithSourcesAndSinks_NoOverlap(t *testing.T) {
	executor := NewDataflowExecutor(&DataflowIR{}, core.NewCallGraph())

	sources := []CallSiteMatch{
		{FunctionFQN: "pkg.funcA", Line: 1},
	}
	sinks := []CallSiteMatch{
		{FunctionFQN: "pkg.funcB", Line: 2},
	}

	result := executor.findFunctionsWithSourcesAndSinks(sources, sinks)

	if len(result) != 0 {
		t.Errorf("expected no overlap, got %v", result)
	}
}

func TestFindFunctionsWithSourcesAndSinks_Deduplication(t *testing.T) {
	executor := NewDataflowExecutor(&DataflowIR{}, core.NewCallGraph())

	sources := []CallSiteMatch{
		{FunctionFQN: "pkg.funcA", Line: 1},
		{FunctionFQN: "pkg.funcA", Line: 3},
	}
	sinks := []CallSiteMatch{
		{FunctionFQN: "pkg.funcA", Line: 5},
		{FunctionFQN: "pkg.funcA", Line: 7},
	}

	result := executor.findFunctionsWithSourcesAndSinks(sources, sinks)

	if len(result) != 1 {
		t.Errorf("expected 1 function (deduped), got %d: %v", len(result), result)
	}
}

func TestResolveMatchers_CallMatcher(t *testing.T) {
	cg := core.NewCallGraph()
	cg.CallSites["test.module.handler"] = []core.CallSite{
		{Target: "os.getenv", TargetFQN: "os.getenv", Location: core.Location{Line: 5}},
		{Target: "eval", TargetFQN: "builtins.eval", Location: core.Location{Line: 10}},
	}

	ir := &DataflowIR{
		Sources:    toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"os.getenv"}}),
		Sinks:      toRawMessages(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}}),
		Sanitizers: emptyRawMessages(),
		Scope:      "local",
	}

	executor := NewDataflowExecutor(ir, cg)

	sourceMatches := executor.resolveMatchers(ir.Sources)
	sinkMatches := executor.resolveMatchers(ir.Sinks)

	if len(sourceMatches) == 0 {
		t.Fatal("expected source matches for os.getenv")
	}
	if len(sinkMatches) == 0 {
		t.Fatal("expected sink matches for eval")
	}

	// Verify match properties
	if sourceMatches[0].CallSite.Target != "os.getenv" {
		t.Errorf("expected source target 'os.getenv', got %q", sourceMatches[0].CallSite.Target)
	}
	if sourceMatches[0].FunctionFQN != "test.module.handler" {
		t.Errorf("expected FunctionFQN 'test.module.handler', got %q", sourceMatches[0].FunctionFQN)
	}
}

func TestResolveMatchers_EmptyInput(t *testing.T) {
	executor := NewDataflowExecutor(&DataflowIR{}, core.NewCallGraph())

	result := executor.resolveMatchers(emptyRawMessages())

	if len(result) != 0 {
		t.Errorf("expected 0 matches for empty input, got %d", len(result))
	}
}

func TestResolveMatchers_NilInput(t *testing.T) {
	executor := NewDataflowExecutor(&DataflowIR{}, core.NewCallGraph())

	result := executor.resolveMatchers(nil)

	if len(result) != 0 {
		t.Errorf("expected 0 matches for nil input, got %d", len(result))
	}
}

func TestNewDataflowExecutor_InitializesDefaults(t *testing.T) {
	ir := &DataflowIR{Scope: "local"}
	cg := core.NewCallGraph()

	executor := NewDataflowExecutor(ir, cg)

	if executor.Config == nil {
		t.Error("Config should be initialized with defaults")
	}
	if executor.Diagnostics == nil {
		t.Error("Diagnostics should be initialized with defaults")
	}
	if executor.IR != ir {
		t.Error("IR should be set")
	}
	if executor.CallGraph != cg {
		t.Error("CallGraph should be set")
	}
}
