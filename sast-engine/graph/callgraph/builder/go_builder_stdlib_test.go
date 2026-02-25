package builder

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errMockNotImplemented is returned by mockStdlibLoader stubs that are not
// exercised in these tests, satisfying the nilnil lint rule.
var errMockNotImplemented = errors.New("not implemented by mock")

// -----------------------------------------------------------------------------
// mockStdlibLoader implements core.GoStdlibLoader for testing.
// ValidateStdlibImport returns true only for import paths explicitly listed in
// the stdlib map; the other methods are stubs that the tests do not call.
// -----------------------------------------------------------------------------

type mockStdlibLoader struct {
	stdlib map[string]bool
}

func (m *mockStdlibLoader) ValidateStdlibImport(importPath string) bool {
	return m.stdlib[importPath]
}

func (m *mockStdlibLoader) GetFunction(_, _ string) (*core.GoStdlibFunction, error) {
	return nil, errMockNotImplemented
}

func (m *mockStdlibLoader) GetType(_, _ string) (*core.GoStdlibType, error) {
	return nil, errMockNotImplemented
}

func (m *mockStdlibLoader) PackageCount() int { return len(m.stdlib) }

// goStdlibPackages is a small set of known stdlib import paths used in tests.
var goStdlibPackages = map[string]bool{
	"fmt":     true,
	"os":      true,
	"net/http": true,
	"strings": true,
}

// newTestImportMap builds a GoImportMap with the given alias→importPath pairs.
func newTestImportMap(file string, entries map[string]string) *core.GoImportMap {
	m := core.NewGoImportMap(file)
	for alias, path := range entries {
		m.AddImport(alias, path)
	}
	return m
}

// newTestRegistry builds a GoModuleRegistry with the given StdlibLoader.
func newTestRegistry(loader core.GoStdlibLoader) *core.GoModuleRegistry {
	reg := core.NewGoModuleRegistry()
	reg.StdlibLoader = loader
	return reg
}

// -----------------------------------------------------------------------------
// Tests for resolveGoCallTarget — Pattern 1a (import resolution + stdlib tag)
// -----------------------------------------------------------------------------

func TestResolveGoCallTarget_StdlibImport(t *testing.T) {
	// fmt is a stdlib package → isStdlib must be true.
	reg := newTestRegistry(&mockStdlibLoader{stdlib: goStdlibPackages})
	importMap := newTestImportMap("main.go", map[string]string{"fmt": "fmt"})

	cs := &CallSiteInternal{FunctionName: "Println", ObjectName: "fmt"}

	targetFQN, resolved, isStdlib := resolveGoCallTarget(cs, importMap, reg, nil, nil, nil)

	require.True(t, resolved)
	assert.Equal(t, "fmt.Println", targetFQN)
	assert.True(t, isStdlib)
}

func TestResolveGoCallTarget_NilStdlibLoader(t *testing.T) {
	// No StdlibLoader — isStdlib must always be false even for stdlib packages.
	reg := newTestRegistry(nil)
	importMap := newTestImportMap("main.go", map[string]string{"fmt": "fmt"})

	cs := &CallSiteInternal{FunctionName: "Println", ObjectName: "fmt"}

	targetFQN, resolved, isStdlib := resolveGoCallTarget(cs, importMap, reg, nil, nil, nil)

	require.True(t, resolved)
	assert.Equal(t, "fmt.Println", targetFQN)
	assert.False(t, isStdlib, "nil StdlibLoader must not classify any call as stdlib")
}

func TestResolveGoCallTarget_ThirdPartyImport(t *testing.T) {
	// Third-party package (not in stdlib map) → isStdlib must be false.
	reg := newTestRegistry(&mockStdlibLoader{stdlib: goStdlibPackages})
	importMap := newTestImportMap("main.go", map[string]string{
		"gin": "github.com/gin-gonic/gin",
	})

	cs := &CallSiteInternal{FunctionName: "Default", ObjectName: "gin"}

	targetFQN, resolved, isStdlib := resolveGoCallTarget(cs, importMap, reg, nil, nil, nil)

	require.True(t, resolved)
	assert.Equal(t, "github.com/gin-gonic/gin.Default", targetFQN)
	assert.False(t, isStdlib)
}

func TestResolveGoCallTarget_StdlibMultiSegmentPath(t *testing.T) {
	// net/http is a multi-segment stdlib path.
	reg := newTestRegistry(&mockStdlibLoader{stdlib: goStdlibPackages})
	importMap := newTestImportMap("server.go", map[string]string{"http": "net/http"})

	cs := &CallSiteInternal{FunctionName: "ListenAndServe", ObjectName: "http"}

	targetFQN, resolved, isStdlib := resolveGoCallTarget(cs, importMap, reg, nil, nil, nil)

	require.True(t, resolved)
	assert.Equal(t, "net/http.ListenAndServe", targetFQN)
	assert.True(t, isStdlib)
}

// -----------------------------------------------------------------------------
// Tests for resolveGoCallTarget — other patterns (isStdlib always false)
// -----------------------------------------------------------------------------

func TestResolveGoCallTarget_Builtin(t *testing.T) {
	// Builtin functions are not stdlib packages — isStdlib must be false.
	reg := newTestRegistry(&mockStdlibLoader{stdlib: goStdlibPackages})
	importMap := newTestImportMap("main.go", nil)

	cs := &CallSiteInternal{FunctionName: "append", ObjectName: ""}

	targetFQN, resolved, isStdlib := resolveGoCallTarget(cs, importMap, reg, nil, nil, nil)

	require.True(t, resolved)
	assert.Equal(t, "builtin.append", targetFQN)
	assert.False(t, isStdlib)
}

func TestResolveGoCallTarget_Unresolved(t *testing.T) {
	// Unknown object — not in imports, not a builtin.
	reg := newTestRegistry(&mockStdlibLoader{stdlib: goStdlibPackages})
	importMap := newTestImportMap("main.go", nil)

	cs := &CallSiteInternal{FunctionName: "Foo", ObjectName: "unknown"}

	targetFQN, resolved, isStdlib := resolveGoCallTarget(cs, importMap, reg, nil, nil, nil)

	assert.False(t, resolved)
	assert.Empty(t, targetFQN)
	assert.False(t, isStdlib)
}

// -----------------------------------------------------------------------------
// BuildGoCallGraph integration: stdlib tagging via StdlibLoader
//
// Uses callgraph_project/main.go which imports "fmt" and calls fmt.Println.
// The mock StdlibLoader validates "fmt" as stdlib so the stdlibCount branch
// (lines 164-166) and the stdlib stats print (lines 208-212) are covered.
// -----------------------------------------------------------------------------

func TestBuildGoCallGraph_StdlibTagging(t *testing.T) {
	fixturePath := "../../../test-fixtures/golang/callgraph_project/main.go"
	absFixturePath, err := filepath.Abs(fixturePath)
	require.NoError(t, err)

	// Function node: main() in callgraph_project/main.go
	mainFunc := &graph.Node{
		ID:         "main_func",
		Type:       "function_declaration",
		Name:       "main",
		File:       absFixturePath,
		LineNumber:  10,
	}

	// Call node: fmt.Println("Starting server...")
	// ObjectName = "fmt" (from Interface[0]), FunctionName = "Println"
	fmtPrintlnCall := &graph.Node{
		ID:        "call_fmt_println",
		Type:      "call",
		Name:      "Println",
		Interface: []string{"fmt"},
		File:      absFixturePath,
		LineNumber: 12,
	}
	edge := &graph.Edge{From: mainFunc, To: fmtPrintlnCall}
	mainFunc.OutgoingEdges = []*graph.Edge{edge}

	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			"main_func":        mainFunc,
			"call_fmt_println": fmtPrintlnCall,
		},
	}

	// Registry with mock StdlibLoader that recognises "fmt" as stdlib.
	reg := &core.GoModuleRegistry{
		DirToImport: map[string]string{
			filepath.Dir(absFixturePath): "github.com/example/callgraph",
		},
		ImportToDir:    make(map[string]string),
		StdlibPackages: map[string]bool{"fmt": true, "net/http": true},
		StdlibLoader:   &mockStdlibLoader{stdlib: map[string]bool{"fmt": true, "net/http": true}},
	}

	goTypeEngine := resolution.NewGoTypeInferenceEngine(reg)
	callGraph, err := BuildGoCallGraph(codeGraph, reg, goTypeEngine)
	require.NoError(t, err)
	require.NotNil(t, callGraph)

	// At least one call site should be marked IsStdlib=true (fmt.Println).
	foundStdlib := false
	for _, sites := range callGraph.CallSites {
		for _, cs := range sites {
			if cs.IsStdlib {
				foundStdlib = true
				assert.Equal(t, "fmt.Println", cs.TargetFQN)
			}
		}
	}
	assert.True(t, foundStdlib, "expected at least one call site marked IsStdlib=true")
}
