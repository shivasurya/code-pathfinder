package extraction

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
)

func TestExtractGoReturnTypes_Basic(t *testing.T) {
	// Setup
	callGraph := core.NewCallGraph()
	registry := &core.GoModuleRegistry{
		ModulePath: "github.com/example/myapp",
		DirToImport: map[string]string{
			"/project/handlers": "github.com/example/myapp/handlers",
		},
	}
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)

	// Add test functions to call graph
	callGraph.Functions["myapp.GetCount"] = &graph.Node{
		ReturnType: "int",
		File:       "/project/handlers/main.go",
	}
	callGraph.Functions["myapp.GetUser"] = &graph.Node{
		ReturnType: "*User",
		File:       "/project/handlers/user.go",
	}
	callGraph.Functions["myapp.LoadConfig"] = &graph.Node{
		ReturnType: "(string, error)",
		File:       "/project/handlers/config.go",
	}

	// Execute
	err := ExtractGoReturnTypes(callGraph, registry, typeEngine)

	// Verify
	assert.NoError(t, err)

	// Check GetCount
	typeInfo, ok := typeEngine.GetReturnType("myapp.GetCount")
	assert.True(t, ok)
	assert.Equal(t, "builtin.int", typeInfo.TypeFQN)

	// Check GetUser (same-package type resolved via registry)
	typeInfo, ok = typeEngine.GetReturnType("myapp.GetUser")
	assert.True(t, ok)
	assert.Equal(t, "github.com/example/myapp/handlers.User", typeInfo.TypeFQN)

	// Check LoadConfig
	typeInfo, ok = typeEngine.GetReturnType("myapp.LoadConfig")
	assert.True(t, ok)
	assert.Equal(t, "builtin.string", typeInfo.TypeFQN) // First return type
}

func TestExtractGoReturnTypes_EmptyReturnType(t *testing.T) {
	// Setup
	callGraph := core.NewCallGraph()
	registry := &core.GoModuleRegistry{}
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)

	// Add function with no return type (void)
	callGraph.Functions["myapp.DoSomething"] = &graph.Node{
		ReturnType: "",
		File:       "/project/main.go",
	}

	// Execute
	err := ExtractGoReturnTypes(callGraph, registry, typeEngine)

	// Verify
	assert.NoError(t, err)

	// Should not be in engine
	_, ok := typeEngine.GetReturnType("myapp.DoSomething")
	assert.False(t, ok)
}

func TestExtractGoReturnTypes_MultipleBuiltins(t *testing.T) {
	// Setup
	callGraph := core.NewCallGraph()
	registry := &core.GoModuleRegistry{}
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)

	// Add multiple functions with builtin return types
	builtins := map[string]string{
		"GetInt":    "int",
		"GetString": "string",
		"GetBool":   "bool",
		"GetError":  "error",
	}

	for name, returnType := range builtins {
		callGraph.Functions["myapp."+name] = &graph.Node{
			ReturnType: returnType,
			File:       "/project/main.go",
		}
	}

	// Execute
	err := ExtractGoReturnTypes(callGraph, registry, typeEngine)

	// Verify
	assert.NoError(t, err)

	// Check all were extracted
	allTypes := typeEngine.GetAllReturnTypes()
	assert.Len(t, allTypes, 4)

	// Verify each one
	for name, expectedType := range map[string]string{
		"myapp.GetInt":    "builtin.int",
		"myapp.GetString": "builtin.string",
		"myapp.GetBool":   "builtin.bool",
		"myapp.GetError":  "builtin.error",
	} {
		typeInfo, ok := typeEngine.GetReturnType(name)
		assert.True(t, ok, "Expected %s to be in engine", name)
		assert.Equal(t, expectedType, typeInfo.TypeFQN)
	}
}

func TestExtractGoReturnTypes_EmptyCallGraph(t *testing.T) {
	// Setup
	callGraph := core.NewCallGraph()
	registry := &core.GoModuleRegistry{}
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)

	// Execute with empty call graph
	err := ExtractGoReturnTypes(callGraph, registry, typeEngine)

	// Verify
	assert.NoError(t, err)

	// Engine should be empty
	allTypes := typeEngine.GetAllReturnTypes()
	assert.Len(t, allTypes, 0)
}
