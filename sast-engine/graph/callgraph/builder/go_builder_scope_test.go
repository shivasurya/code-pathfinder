package builder

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
)

// TestIndexGoFunctions_EagerScopeCreation verifies that indexGoFunctions creates
// an empty GoFunctionScope for every indexed Go function even when no variable
// bindings are available.  This ensures Pattern 1b Source 2 always finds a scope.
func TestIndexGoFunctions_EagerScopeCreation(t *testing.T) {
	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			"fn1": {
				ID:   "fn1",
				Type: "function_declaration",
				Name: "HandleRequest",
				File: "/project/main.go",
			},
			"fn2": {
				ID:   "fn2",
				Type: "method",
				Name: "Close",
				File: "/project/main.go",
			},
			"other": {
				ID:   "other",
				Type: "identifier", // non-function node — must be skipped
				Name: "x",
				File: "/project/main.go",
			},
		},
	}

	callGraph := &core.CallGraph{
		Functions: make(map[string]*graph.Node),
	}

	registry := core.NewGoModuleRegistry()
	registry.ModulePath = "github.com/example/app"
	registry.DirToImport = map[string]string{
		"/project": "github.com/example/app",
	}

	typeEngine := resolution.NewGoTypeInferenceEngine(registry)

	indexGoFunctions(codeGraph, callGraph, registry, typeEngine)

	// Every indexed function should have an eager scope.
	for fqn := range callGraph.Functions {
		scope := typeEngine.GetScope(fqn)
		assert.NotNil(t, scope, "scope should be eagerly created for %s", fqn)
	}

	// Non-function node must not appear in Functions map.
	for fqn := range callGraph.Functions {
		assert.NotContains(t, fqn, "identifier")
	}
}

// TestIndexGoFunctions_EagerScope_NilTypeEngine verifies that passing nil for
// typeEngine does not panic and still indexes functions normally.
func TestIndexGoFunctions_EagerScope_NilTypeEngine(t *testing.T) {
	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			"fn1": {
				ID:   "fn1",
				Type: "function_declaration",
				Name: "Run",
				File: "/project/main.go",
			},
		},
	}

	callGraph := &core.CallGraph{
		Functions: make(map[string]*graph.Node),
	}

	registry := core.NewGoModuleRegistry()
	registry.ModulePath = "github.com/example/app"
	registry.DirToImport = map[string]string{
		"/project": "github.com/example/app",
	}

	// Must not panic with nil typeEngine.
	assert.NotPanics(t, func() {
		indexGoFunctions(codeGraph, callGraph, registry, nil)
	})

	assert.NotEmpty(t, callGraph.Functions)
}

// TestIndexGoFunctions_EagerScope_NotOverwritten verifies that if a scope already
// exists (e.g., created by a previous Pass 2b run), it is not replaced.
func TestIndexGoFunctions_EagerScope_NotOverwritten(t *testing.T) {
	fqn := "github.com/example/app.ExistingFunc"

	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			"fn1": {
				ID:   "fn1",
				Type: "function_declaration",
				Name: "ExistingFunc",
				File: "/project/main.go",
			},
		},
	}

	callGraph := &core.CallGraph{
		Functions: make(map[string]*graph.Node),
	}

	registry := core.NewGoModuleRegistry()
	registry.ModulePath = "github.com/example/app"
	registry.DirToImport = map[string]string{
		"/project": "github.com/example/app",
	}

	typeEngine := resolution.NewGoTypeInferenceEngine(registry)

	// Pre-create scope with a binding to simulate Pass 2b having run first.
	preScope := resolution.NewGoFunctionScope(fqn)
	preScope.AddVariable(&resolution.GoVariableBinding{
		VarName: "existing",
		Type:    &core.TypeInfo{TypeFQN: "builtin.string"},
	})
	typeEngine.AddScope(preScope)

	indexGoFunctions(codeGraph, callGraph, registry, typeEngine)

	// The pre-created scope must still have the binding.
	scope := typeEngine.GetScope(fqn)
	assert.NotNil(t, scope)
	bindings, ok := scope.Variables["existing"]
	assert.True(t, ok, "pre-existing binding should not be overwritten")
	assert.Len(t, bindings, 1)
}
