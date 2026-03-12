package dsl

import (
	"encoding/json"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func TestBFSReachability_Simple(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Edges["A"] = []string{"B"}

	executor := &DataflowExecutor{CallGraph: cg}
	reachable := executor.bfsReachable("A")

	assert.True(t, reachable["A"])
	assert.True(t, reachable["B"])
	assert.False(t, reachable["C"])
}

func TestBFSReachability_MultiHop(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Edges["A"] = []string{"B"}
	cg.Edges["B"] = []string{"C"}
	cg.Edges["C"] = []string{"D"}

	executor := &DataflowExecutor{CallGraph: cg}
	reachable := executor.bfsReachable("A")

	assert.True(t, reachable["A"])
	assert.True(t, reachable["B"])
	assert.True(t, reachable["C"])
	assert.True(t, reachable["D"])
}

func TestBFSReachability_Disconnected(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Edges["A"] = []string{"B"}
	cg.Edges["C"] = []string{"D"}

	executor := &DataflowExecutor{CallGraph: cg}
	reachable := executor.bfsReachable("A")

	assert.True(t, reachable["A"])
	assert.True(t, reachable["B"])
	assert.False(t, reachable["C"])
	assert.False(t, reachable["D"])
}

func TestBFSReachability_Cycle(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Edges["A"] = []string{"B"}
	cg.Edges["B"] = []string{"C"}
	cg.Edges["C"] = []string{"A"} // Cycle back to A.

	executor := &DataflowExecutor{CallGraph: cg}
	reachable := executor.bfsReachable("A")

	assert.True(t, reachable["A"])
	assert.True(t, reachable["B"])
	assert.True(t, reachable["C"])
	assert.Len(t, reachable, 3)
}

func TestBFSReachability_NoEdges(t *testing.T) {
	cg := core.NewCallGraph()
	executor := &DataflowExecutor{CallGraph: cg}
	reachable := executor.bfsReachable("A")

	assert.True(t, reachable["A"])
	assert.Len(t, reachable, 1)
}

func TestEarlyExit_ZeroSources(t *testing.T) {
	dc := NewDiagnosticCollector()
	cg := core.NewCallGraph()
	ir := &DataflowIR{
		Sources: nil,
		Sinks:   nil,
		Scope:   "local",
	}
	executor := &DataflowExecutor{
		IR:          ir,
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	assert.Empty(t, results)

	debugs := dc.FilterByLevel("debug")
	assert.NotEmpty(t, debugs)
	assert.Contains(t, debugs[0].Message, "0 sources found")
}

func TestEarlyExit_ZeroSinks(t *testing.T) {
	dc := NewDiagnosticCollector()
	cg := core.NewCallGraph()
	cg.CallSites["app.main"] = []core.CallSite{
		{Target: "eval", Location: core.Location{Line: 1}},
	}
	sourceBytes, _ := json.Marshal(CallMatcherIR{Type: "call_matcher", Patterns: []string{"eval"}})
	ir := &DataflowIR{
		Sources: []json.RawMessage{sourceBytes},
		Sinks:   nil,
		Scope:   "local",
	}
	executor := &DataflowExecutor{
		IR:          ir,
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	assert.Empty(t, results)

	debugs := dc.FilterByLevel("debug")
	assert.NotEmpty(t, debugs)
	found := false
	for _, d := range debugs {
		if d.Message == "0 sinks found, skipping local analysis" {
			found = true
		}
	}
	assert.True(t, found, "expected '0 sinks found' diagnostic")
}

func TestEarlyExit_GlobalZeroSources(t *testing.T) {
	dc := NewDiagnosticCollector()
	cg := core.NewCallGraph()
	ir := &DataflowIR{
		Sources: nil,
		Sinks:   nil,
		Scope:   "global",
	}
	executor := &DataflowExecutor{
		IR:          ir,
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: dc,
	}

	results := executor.Execute()
	assert.Empty(t, results)
}

func TestPathHasSanitizerSet(t *testing.T) {
	cg := core.NewCallGraph()
	executor := &DataflowExecutor{CallGraph: cg}

	sanitizerSet := map[string]bool{"sanitize": true}

	assert.True(t, executor.pathHasSanitizerSet([]string{"a", "sanitize", "b"}, sanitizerSet))
	assert.False(t, executor.pathHasSanitizerSet([]string{"a", "b", "c"}, sanitizerSet))
	assert.False(t, executor.pathHasSanitizerSet([]string{}, sanitizerSet))
}
