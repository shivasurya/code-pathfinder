package dsl

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// buildCallGraphWithCallSites creates a call graph with N call sites spread across functions.
func buildCallGraphWithCallSites(n int) *core.CallGraph {
	cg := core.NewCallGraph()
	for i := 0; i < n; i++ {
		funcFQN := fmt.Sprintf("app.func_%d", i/10)
		cg.CallSites[funcFQN] = append(cg.CallSites[funcFQN], core.CallSite{
			Target:                  fmt.Sprintf("module.method_%d", i%5),
			TargetFQN:               fmt.Sprintf("module.Class.method_%d", i%5),
			Location:                core.Location{Line: i + 1},
			ResolvedViaTypeInference: i%3 == 0,
			InferredType:             "module.Class",
			TypeConfidence:           0.8,
		})
	}
	return cg
}

// buildCallGraphChain creates a linear chain: f0 → f1 → f2 → ... → fN.
func buildCallGraphChain(n int) *core.CallGraph {
	cg := core.NewCallGraph()
	for i := 0; i < n-1; i++ {
		from := fmt.Sprintf("func_%d", i)
		to := fmt.Sprintf("func_%d", i+1)
		cg.Edges[from] = append(cg.Edges[from], to)
		cg.CallSites[from] = []core.CallSite{
			{Target: fmt.Sprintf("call_%d", i), Location: core.Location{Line: i + 1}},
		}
	}
	lastFunc := fmt.Sprintf("func_%d", n-1)
	cg.CallSites[lastFunc] = []core.CallSite{
		{Target: fmt.Sprintf("call_%d", n-1), Location: core.Location{Line: n}},
	}
	return cg
}

func BenchmarkTypeConstrainedCall_100CallSites(b *testing.B) {
	cg := buildCallGraphWithCallSites(100)
	ir := &TypeConstrainedCallIR{
		ReceiverType: "module.Class",
		MethodName:   "method_0",
		FallbackMode: "name",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:        ir,
		CallGraph: cg,
		Config:    DefaultConfig(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.Execute()
	}
}

func BenchmarkTypeConstrainedCall_10kCallSites(b *testing.B) {
	cg := buildCallGraphWithCallSites(10_000)
	ir := &TypeConstrainedCallIR{
		ReceiverType: "module.Class",
		MethodName:   "method_0",
		FallbackMode: "name",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:        ir,
		CallGraph: cg,
		Config:    DefaultConfig(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.Execute()
	}
}

func BenchmarkTypeConstrainedCall_100kCallSites(b *testing.B) {
	cg := buildCallGraphWithCallSites(100_000)
	ir := &TypeConstrainedCallIR{
		ReceiverType: "module.Class",
		MethodName:   "method_0",
		FallbackMode: "name",
	}
	executor := &TypeConstrainedCallExecutor{
		IR:        ir,
		CallGraph: cg,
		Config:    DefaultConfig(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.Execute()
	}
}

func BenchmarkDataflowLocal_100Sources_100Sinks(b *testing.B) {
	cg := core.NewCallGraph()
	for i := 0; i < 100; i++ {
		funcFQN := fmt.Sprintf("app.func_%d", i%10)
		cg.CallSites[funcFQN] = append(cg.CallSites[funcFQN],
			core.CallSite{Target: "source_call", Location: core.Location{Line: i*2 + 1}},
			core.CallSite{Target: "sink_call", Location: core.Location{Line: i*2 + 2}},
		)
	}

	sourceBytes, _ := json.Marshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"source_call"}})
	sinkBytes, _ := json.Marshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sink_call"}})
	ir := &DataflowIR{
		Sources: []json.RawMessage{sourceBytes},
		Sinks:   []json.RawMessage{sinkBytes},
		Scope:   "local",
	}

	executor := &DataflowExecutor{IR: ir, CallGraph: cg, Config: DefaultConfig()}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.Execute()
	}
}

func BenchmarkDataflowGlobal_100Sources_100Sinks_10kNodes(b *testing.B) {
	cg := buildCallGraphChain(100)
	// Add source call sites at start of chain, sink call sites at end.
	for i := 0; i < 10; i++ {
		funcFQN := fmt.Sprintf("func_%d", i)
		cg.CallSites[funcFQN] = append(cg.CallSites[funcFQN],
			core.CallSite{Target: "source_call", Location: core.Location{Line: 1}},
		)
	}
	for i := 90; i < 100; i++ {
		funcFQN := fmt.Sprintf("func_%d", i)
		cg.CallSites[funcFQN] = append(cg.CallSites[funcFQN],
			core.CallSite{Target: "sink_call", Location: core.Location{Line: 100}},
		)
	}

	sourceBytes, _ := json.Marshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"source_call"}})
	sinkBytes, _ := json.Marshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"sink_call"}})
	ir := &DataflowIR{
		Sources: []json.RawMessage{sourceBytes},
		Sinks:   []json.RawMessage{sinkBytes},
		Scope:   "global",
	}

	executor := &DataflowExecutor{IR: ir, CallGraph: cg, Config: DefaultConfig()}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.Execute()
	}
}

func BenchmarkMatchesReceiverType_WithMRO(b *testing.B) {
	// Simulates MRO lookup overhead with a mock checker.
	checker := &mockMROChecker{
		modules: map[string]bool{"django": true},
		mro:     map[string][]string{"views.View": {"django.views.View", "builtins.object"}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchesReceiverType("django.views.View", "View", checker)
	}
}

func BenchmarkArgumentMatcher_WithRegex(b *testing.B) {
	args := []core.Argument{
		{Value: "SELECT * FROM users WHERE id = 42", Position: 0},
	}
	constraints := map[string]ArgumentConstraint{
		"0": {Value: "^SELECT.*FROM", Comparator: "regex"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MatchesPositionalArguments(args, constraints)
	}
}

// mockMROChecker implements InheritanceChecker for benchmarks.
type mockMROChecker struct {
	modules map[string]bool
	mro     map[string][]string
}

func (m *mockMROChecker) HasModule(moduleName string) bool {
	return m.modules[moduleName]
}

func (m *mockMROChecker) IsSubclassSimple(_, _, _ string) bool {
	return false
}

func (m *mockMROChecker) GetClassMRO(_, className string) []string {
	return m.mro[className]
}
