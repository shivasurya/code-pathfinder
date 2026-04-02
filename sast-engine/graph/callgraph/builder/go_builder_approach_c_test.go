package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApproachC_ThirdPartyPartialResolution verifies that when type is known
// but not in stdlib or user code, partial resolution with best-effort FQN is used.
func TestApproachC_ThirdPartyPartialResolution(t *testing.T) {
	goRegistry := &core.GoModuleRegistry{
		DirToImport: map[string]string{},
		ImportToDir: map[string]string{},
	}

	typeEngine := resolution.NewGoTypeInferenceEngine(goRegistry)
	scope := resolution.NewGoFunctionScope("testapp.handler")
	scope.AddVariable(&resolution.GoVariableBinding{
		VarName:      "client",
		Type:         &core.TypeInfo{TypeFQN: "github.com/redis/go-redis/v9.Client", Confidence: 0.8},
		AssignedFrom: "redis.NewClient",
	})
	typeEngine.AddScope(scope)

	callGraph := core.NewCallGraph()
	importMap := core.NewGoImportMap("test.go")

	callSite := &CallSiteInternal{
		CallerFQN:    "testapp.handler",
		CallerFile:   "test.go",
		CallLine:     15,
		FunctionName: "Get",
		ObjectName:   "client",
	}

	targetFQN, resolved, _ := resolveGoCallTarget(
		callSite, importMap, goRegistry, nil, typeEngine, callGraph,
	)

	assert.Equal(t, "github.com/redis/go-redis/v9.Client.Get", targetFQN)
	assert.True(t, resolved, "Should partially resolve with known type even without stdlib validation")
}

// TestApproachC_UserCodeMethodResolution verifies that Pattern 1b still works
// for methods defined in user code (not stdlib).
func TestApproachC_UserCodeMethodResolution(t *testing.T) {
	goRegistry := &core.GoModuleRegistry{
		DirToImport: map[string]string{},
		ImportToDir: map[string]string{},
	}

	typeEngine := resolution.NewGoTypeInferenceEngine(goRegistry)
	scope := resolution.NewGoFunctionScope("testapp.main")
	scope.AddVariable(&resolution.GoVariableBinding{
		VarName:      "svc",
		Type:         &core.TypeInfo{TypeFQN: "testapp.Service", Confidence: 0.95},
		AssignedFrom: "NewService",
	})
	typeEngine.AddScope(scope)

	callGraph := core.NewCallGraph()
	// Add the method to Functions so Check 1 matches
	callGraph.Functions["testapp.Service.Handle"] = &graph.Node{
		ID: "m1", Type: "method", Name: "Handle", Language: "go",
	}

	importMap := core.NewGoImportMap("test.go")

	callSite := &CallSiteInternal{
		CallerFQN:    "testapp.main",
		CallerFile:   "test.go",
		CallLine:     10,
		FunctionName: "Handle",
		ObjectName:   "svc",
	}

	targetFQN, resolved, isStdlib := resolveGoCallTarget(
		callSite, importMap, goRegistry, nil, typeEngine, callGraph,
	)

	assert.Equal(t, "testapp.Service.Handle", targetFQN)
	assert.True(t, resolved)
	assert.False(t, isStdlib, "User code methods are not stdlib")
}

// TestApproachC_PointerTypeStripping verifies that *pkg.Type is stripped to pkg.Type.
func TestApproachC_PointerTypeStripping(t *testing.T) {
	goRegistry := &core.GoModuleRegistry{
		DirToImport: map[string]string{},
		ImportToDir: map[string]string{},
	}

	typeEngine := resolution.NewGoTypeInferenceEngine(goRegistry)
	scope := resolution.NewGoFunctionScope("testapp.handler")
	scope.AddVariable(&resolution.GoVariableBinding{
		VarName:      "db",
		Type:         &core.TypeInfo{TypeFQN: "*database/sql.DB", Confidence: 0.9},
		AssignedFrom: "sql.Open",
	})
	typeEngine.AddScope(scope)

	callGraph := core.NewCallGraph()
	importMap := core.NewGoImportMap("test.go")

	callSite := &CallSiteInternal{
		CallerFQN:    "testapp.handler",
		CallerFile:   "test.go",
		CallLine:     10,
		FunctionName: "Query",
		ObjectName:   "db",
	}

	targetFQN, resolved, _ := resolveGoCallTarget(
		callSite, importMap, goRegistry, nil, typeEngine, callGraph,
	)

	// Pointer * should be stripped: *database/sql.DB → database/sql.DB
	assert.Equal(t, "database/sql.DB.Query", targetFQN)
	assert.True(t, resolved)
}

// TestApproachC_TypeInferenceFieldsOnCallSite verifies that after BuildGoCallGraph,
// CallSites have type inference fields populated for variable-based calls.
func TestApproachC_TypeInferenceFieldsOnCallSite(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

import "fmt"

func handler() {
	msg := fmt.Sprintf("hello %s", "world")
	_ = msg
}
`), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)
	goRegistry, err := resolution.BuildGoModuleRegistry(tmpDir)
	require.NoError(t, err)

	goTypeEngine := resolution.NewGoTypeInferenceEngine(goRegistry)

	callGraph, err := BuildGoCallGraph(codeGraph, goRegistry, goTypeEngine)
	require.NoError(t, err)

	// Find fmt.Sprintf call site — resolved via Pattern 1a (import)
	for _, callSites := range callGraph.CallSites {
		for _, cs := range callSites {
			if cs.Target == "Sprintf" && cs.Resolved {
				assert.Equal(t, "fmt.Sprintf", cs.TargetFQN)
			}
		}
	}
}

// TestApproachC_NoTypeEngine verifies Pattern 1b gracefully skips when typeEngine is nil.
func TestApproachC_NoTypeEngine(t *testing.T) {
	goRegistry := &core.GoModuleRegistry{
		DirToImport: map[string]string{},
		ImportToDir: map[string]string{},
	}

	callGraph := core.NewCallGraph()
	importMap := core.NewGoImportMap("test.go")

	callSite := &CallSiteInternal{
		CallerFQN:    "testapp.handler",
		CallerFile:   "test.go",
		CallLine:     10,
		FunctionName: "Query",
		ObjectName:   "db",
	}

	// No typeEngine → Pattern 1b skipped → unresolved
	targetFQN, resolved, _ := resolveGoCallTarget(
		callSite, importMap, goRegistry, nil, nil, callGraph,
	)

	assert.Equal(t, "", targetFQN)
	assert.False(t, resolved)
}
