package builder

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
)

// makePackageVarCodeGraph builds a CodeGraph containing a module_variable node
// that represents `var <varName> <dataType>` in the given file.
func makePackageVarCodeGraph(varName, dataType, file string) *graph.CodeGraph {
	cg := graph.NewCodeGraph()
	cg.Nodes[varName] = &graph.Node{
		ID:       varName,
		Type:     "module_variable",
		Name:     varName,
		DataType: dataType,
		File:     file,
		Language: "go",
	}
	return cg
}

// TestSource3_PackageLevelVariable verifies that Source 3 resolves the type of a
// package-level variable and returns the correct method FQN.
func TestSource3_PackageLevelVariable(t *testing.T) {
	cg := makePackageVarCodeGraph("globalDB", "sql.DB", "/project/main.go")

	callSite := &CallSiteInternal{
		CallerFQN:    "main.handler",
		CallerFile:   "/project/main.go",
		ObjectName:   "globalDB",
		FunctionName: "Query",
	}

	importMap := &core.GoImportMap{
		Imports: map[string]string{"sql": "database/sql"},
	}

	reg := core.NewGoModuleRegistry()
	callGraph := &core.CallGraph{Functions: make(map[string]*graph.Node)}

	targetFQN, resolved, _, _ := resolveGoCallTarget(
		callSite, importMap, reg, nil, nil, callGraph, buildPkgVarIndex(cg),
		nil,
	)

	assert.True(t, resolved)
	assert.Equal(t, "database/sql.DB.Query", targetFQN)
}

// TestSource3_PointerType verifies that Source 3 strips the leading `*` from the
// DataType field (e.g. `var db *sql.DB` stores DataType as "*sql.DB").
func TestSource3_PointerType(t *testing.T) {
	cg := makePackageVarCodeGraph("db", "*sql.DB", "/project/store.go")

	callSite := &CallSiteInternal{
		CallerFQN:    "main.runQuery",
		CallerFile:   "/project/store.go",
		ObjectName:   "db",
		FunctionName: "Exec",
	}

	importMap := &core.GoImportMap{
		Imports: map[string]string{"sql": "database/sql"},
	}

	reg := core.NewGoModuleRegistry()
	callGraph := &core.CallGraph{Functions: make(map[string]*graph.Node)}

	targetFQN, resolved, _, _ := resolveGoCallTarget(
		callSite, importMap, reg, nil, nil, callGraph, buildPkgVarIndex(cg),
		nil,
	)

	assert.True(t, resolved)
	assert.Equal(t, "database/sql.DB.Exec", targetFQN)
}

// TestSource3_SamePackageFilter verifies that Source 3 only resolves variables
// defined in the same package as the caller (same directory).
func TestSource3_SamePackageFilter(t *testing.T) {
	cg := graph.NewCodeGraph()
	// Variable in a DIFFERENT package (/project/other/db.go)
	cg.Nodes["otherDB"] = &graph.Node{
		ID:       "otherDB",
		Type:     "module_variable",
		Name:     "globalDB",
		DataType: "sql.DB",
		File:     "/project/other/db.go",
		Language: "go",
	}

	callSite := &CallSiteInternal{
		CallerFQN:    "main.handler",
		CallerFile:   "/project/main.go", // different directory
		ObjectName:   "globalDB",
		FunctionName: "Query",
	}

	importMap := &core.GoImportMap{Imports: map[string]string{"sql": "database/sql"}}
	reg := core.NewGoModuleRegistry()
	callGraph := &core.CallGraph{Functions: make(map[string]*graph.Node)}

	_, resolved, _, _ := resolveGoCallTarget(
		callSite, importMap, reg, nil, nil, callGraph, buildPkgVarIndex(cg),
		nil,
	)

	// Must NOT resolve: variable is in a different package directory.
	assert.False(t, resolved)
}

// TestSource3_NoTypeAnnotation verifies that Source 3 gracefully skips a
// module_variable node whose DataType is empty (e.g. `var db = sql.Open(...)`).
func TestSource3_NoTypeAnnotation(t *testing.T) {
	cg := makePackageVarCodeGraph("db", "", "/project/main.go") // empty DataType

	callSite := &CallSiteInternal{
		CallerFQN:    "main.handler",
		CallerFile:   "/project/main.go",
		ObjectName:   "db",
		FunctionName: "Query",
	}

	importMap := &core.GoImportMap{Imports: map[string]string{}}
	reg := core.NewGoModuleRegistry()
	callGraph := &core.CallGraph{Functions: make(map[string]*graph.Node)}

	_, resolved, _, _ := resolveGoCallTarget(
		callSite, importMap, reg, nil, nil, callGraph, buildPkgVarIndex(cg),
		nil,
	)

	// Must NOT resolve: no type info available.
	assert.False(t, resolved)
}

// TestSource3_NilCodeGraph verifies that Source 3 does not panic with a nil CodeGraph.
func TestSource3_NilCodeGraph(t *testing.T) {
	callSite := &CallSiteInternal{
		CallerFQN:    "main.handler",
		CallerFile:   "/project/main.go",
		ObjectName:   "globalDB",
		FunctionName: "Query",
	}

	importMap := &core.GoImportMap{Imports: map[string]string{}}
	reg := core.NewGoModuleRegistry()
	callGraph := &core.CallGraph{Functions: make(map[string]*graph.Node)}

	assert.NotPanics(t, func() {
		resolveGoCallTarget(callSite, importMap, reg, nil, nil, callGraph, nil, nil)
	})
}
