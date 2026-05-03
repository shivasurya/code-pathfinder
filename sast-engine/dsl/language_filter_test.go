package dsl

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

func TestLanguageFilter_GoOnly(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Functions["testapp.handler"] = &graph.Node{Language: "go"}
	cg.Functions["myapp.handler"] = &graph.Node{Language: "python"}

	sources := []CallSiteMatch{
		{FunctionFQN: "testapp.handler", Line: 1},
		{FunctionFQN: "myapp.handler", Line: 1},
	}
	sinks := []CallSiteMatch{
		{FunctionFQN: "testapp.handler", Line: 5},
		{FunctionFQN: "myapp.handler", Line: 5},
	}

	executor := &DataflowExecutor{
		IR:          &DataflowIR{Language: "go"},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	functions := executor.findFunctionsWithSourcesAndSinks(sources, sinks)

	assert.Len(t, functions, 1, "Should only return Go function")
	assert.Equal(t, "testapp.handler", functions[0])
}

func TestLanguageFilter_NoLanguage(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Functions["testapp.handler"] = &graph.Node{Language: "go"}
	cg.Functions["myapp.handler"] = &graph.Node{Language: "python"}

	sources := []CallSiteMatch{
		{FunctionFQN: "testapp.handler", Line: 1},
		{FunctionFQN: "myapp.handler", Line: 1},
	}
	sinks := []CallSiteMatch{
		{FunctionFQN: "testapp.handler", Line: 5},
		{FunctionFQN: "myapp.handler", Line: 5},
	}

	executor := &DataflowExecutor{
		IR:          &DataflowIR{Language: ""},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	functions := executor.findFunctionsWithSourcesAndSinks(sources, sinks)

	assert.Len(t, functions, 2, "Should return both when no language filter")
}

func TestLanguageFilter_PythonOnly(t *testing.T) {
	cg := core.NewCallGraph()
	cg.Functions["testapp.handler"] = &graph.Node{Language: "go"}
	cg.Functions["myapp.handler"] = &graph.Node{Language: "python"}

	sources := []CallSiteMatch{
		{FunctionFQN: "testapp.handler", Line: 1},
		{FunctionFQN: "myapp.handler", Line: 1},
	}
	sinks := []CallSiteMatch{
		{FunctionFQN: "testapp.handler", Line: 5},
		{FunctionFQN: "myapp.handler", Line: 5},
	}

	executor := &DataflowExecutor{
		IR:          &DataflowIR{Language: "python"},
		CallGraph:   cg,
		Config:      DefaultConfig(),
		Diagnostics: NewDiagnosticCollector(),
	}

	functions := executor.findFunctionsWithSourcesAndSinks(sources, sinks)

	assert.Len(t, functions, 1)
	assert.Equal(t, "myapp.handler", functions[0])
}
