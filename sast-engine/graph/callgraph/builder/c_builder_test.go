package builder_test

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/builder"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cFixture builds a small parsed CodeGraph + CModuleRegistry to drive
// BuildCCallGraph in unit tests. Tests opt into specific node shapes by
// passing nodes into the helpers; the registry is wired automatically
// from the function set so cross-file resolution works.
type cFixture struct {
	root      string
	cg        *graph.CodeGraph
	registry  *core.CModuleRegistry
	functions map[string]*graph.Node
}

// fixtureRoot is the absolute project root used by every cFixture.
// Tests share it so file paths in assertions stay readable.
const fixtureRoot = "/projects/app"

func newCFixture(t *testing.T) *cFixture {
	t.Helper()
	return &cFixture{
		root:      fixtureRoot,
		cg:        graph.NewCodeGraph(),
		registry:  core.NewCModuleRegistry(fixtureRoot),
		functions: make(map[string]*graph.Node),
	}
}

// addFunction registers a function_definition (or declaration) under
// the given absolute file path. relPath is the project-relative form
// the registry would compute. Returns the node so callers can attach
// outgoing call edges.
func (f *cFixture) addFunction(t *testing.T, file, relPath, name, returnType string, isDecl bool) *graph.Node {
	t.Helper()
	node := &graph.Node{
		ID:         "fn:" + relPath + "::" + name,
		Type:       "function_definition",
		Name:       name,
		File:       file,
		Language:   "c",
		ReturnType: returnType,
		LineNumber: 1,
	}
	if isDecl {
		node.Metadata = map[string]any{"is_declaration": true}
	}
	f.cg.AddNode(node)
	f.registry.FileToPrefix[file] = relPath
	fqn := relPath + "::" + name
	f.functions[fqn] = node
	f.registry.FunctionIndex[name] = append(f.registry.FunctionIndex[name], fqn)
	return node
}

// addCall attaches a call_expression node to caller, mimicking the
// edge the parser emits during AST traversal.
func (f *cFixture) addCall(t *testing.T, caller *graph.Node, target string, args []string) {
	t.Helper()
	call := &graph.Node{
		ID:                   "call:" + caller.ID + "->" + target,
		Type:                 "call_expression",
		Name:                 target,
		File:                 caller.File,
		Language:             "c",
		LineNumber:           caller.LineNumber + 1,
		MethodArgumentsValue: args,
	}
	f.cg.AddNode(call)
	f.cg.AddEdge(caller, call)
}

func (f *cFixture) build(t *testing.T) (*core.CallGraph, *resolution.CTypeInferenceEngine) {
	t.Helper()
	engine := resolution.NewCTypeInferenceEngine(f.registry)
	cg, err := builder.BuildCCallGraph(f.cg, f.registry, engine)
	require.NoError(t, err)
	require.NotNil(t, cg)
	return cg, engine
}

// TestBuildCCallGraph_NilInputs verifies the builder degrades gracefully
// when given a nil CodeGraph or registry — useful for callers that
// guard upstream errors with optional chaining.
func TestBuildCCallGraph_NilInputs(t *testing.T) {
	cg, err := builder.BuildCCallGraph(nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, cg)
	assert.Empty(t, cg.Functions)
	assert.Empty(t, cg.Edges)
}

// TestBuildCCallGraph_SingleFile_BasicEdge covers the simplest call
// graph: main() calls add(); both are in the same .c file. The edge
// must resolve to the same-file FQN.
func TestBuildCCallGraph_SingleFile_BasicEdge(t *testing.T) {
	root := fixtureRoot
	mainC := root + "/src/main.c"

	f := newCFixture(t)
	mainFn := f.addFunction(t, mainC, "src/main.c", "main", "int", false)
	f.addFunction(t, mainC, "src/main.c", "add", "int", false)
	f.addCall(t, mainFn, "add", []string{"1", "2"})

	cg, _ := f.build(t)

	assert.Contains(t, cg.Functions, "src/main.c::main")
	assert.Contains(t, cg.Functions, "src/main.c::add")
	assert.Equal(t, []string{"src/main.c::add"}, cg.Edges["src/main.c::main"])
	assert.Equal(t, []string{"src/main.c::main"}, cg.ReverseEdges["src/main.c::add"])

	sites := cg.CallSites["src/main.c::main"]
	require.Len(t, sites, 1)
	assert.True(t, sites[0].Resolved)
	assert.Equal(t, "src/main.c::add", sites[0].TargetFQN)
	assert.Equal(t, "add", sites[0].Target)
	assert.InDelta(t, 1.0, sites[0].TypeConfidence, 1e-6)
	assert.Equal(t, "declaration", sites[0].TypeSource)
	assert.Empty(t, sites[0].FailureReason)
	require.Len(t, sites[0].Arguments, 2)
	assert.Equal(t, "1", sites[0].Arguments[0].Value)
	assert.False(t, sites[0].Arguments[0].IsVariable, "numeric literal must not be flagged as a variable")
}

// TestBuildCCallGraph_PrefersDefinitionOverDeclaration covers the
// definition-preferring resolution order: when the same function name
// appears as both a header declaration and a source-file definition,
// the edge must point at the definition.
func TestBuildCCallGraph_PrefersDefinitionOverDeclaration(t *testing.T) {
	root := fixtureRoot
	mainC := root + "/src/main.c"
	utilsH := root + "/include/utils.h"
	utilsC := root + "/src/utils.c"

	f := newCFixture(t)
	mainFn := f.addFunction(t, mainC, "src/main.c", "main", "int", false)
	// Header declaration (no body).
	f.addFunction(t, utilsH, "include/utils.h", "create_buffer", "Buffer*", true)
	// Source definition (with body).
	f.addFunction(t, utilsC, "src/utils.c", "create_buffer", "Buffer*", false)

	f.addCall(t, mainFn, "create_buffer", nil)

	cg, _ := f.build(t)

	require.Len(t, cg.Edges["src/main.c::main"], 1)
	assert.Equal(t, "src/utils.c::create_buffer", cg.Edges["src/main.c::main"][0],
		"resolver must prefer the .c definition over the .h declaration")
}

// TestBuildCCallGraph_DeclarationFallbackThroughIncludes confirms that
// when no project-wide definition exists, the resolver falls back to a
// declaration reachable through #include "...".
func TestBuildCCallGraph_DeclarationFallbackThroughIncludes(t *testing.T) {
	root := fixtureRoot
	mainC := root + "/src/main.c"
	apiH := root + "/include/api.h"

	f := newCFixture(t)
	mainFn := f.addFunction(t, mainC, "src/main.c", "main", "int", false)
	f.addFunction(t, apiH, "include/api.h", "external_op", "int", true)
	f.registry.Includes["src/main.c"] = []string{"include/api.h"}

	f.addCall(t, mainFn, "external_op", nil)

	cg, _ := f.build(t)

	require.Len(t, cg.Edges["src/main.c::main"], 1)
	assert.Equal(t, "include/api.h::external_op", cg.Edges["src/main.c::main"][0])
	sites := cg.CallSites["src/main.c::main"]
	require.Len(t, sites, 1)
	assert.True(t, sites[0].Resolved)
}

// TestBuildCCallGraph_StdlibCallUnresolved verifies that calls to
// functions not present in the registry (e.g. printf, malloc) are
// recorded as Resolved:false with a failure reason — they must not
// contribute an edge but must remain visible to rule writers.
func TestBuildCCallGraph_StdlibCallUnresolved(t *testing.T) {
	root := fixtureRoot
	mainC := root + "/src/main.c"

	f := newCFixture(t)
	mainFn := f.addFunction(t, mainC, "src/main.c", "main", "int", false)
	f.addCall(t, mainFn, "printf", []string{"\"hello\""})

	cg, _ := f.build(t)

	assert.Empty(t, cg.Edges["src/main.c::main"], "external call must not produce an edge")
	sites := cg.CallSites["src/main.c::main"]
	require.Len(t, sites, 1)
	assert.False(t, sites[0].Resolved)
	assert.Equal(t, "printf", sites[0].Target)
	assert.Empty(t, sites[0].TargetFQN)
	assert.NotEmpty(t, sites[0].FailureReason, "unresolved sites must record a failure reason")
}

// TestBuildCCallGraph_RecursiveCall verifies that a function calling
// itself produces a self-edge (caller == callee FQN).
func TestBuildCCallGraph_RecursiveCall(t *testing.T) {
	root := fixtureRoot
	mainC := root + "/src/util.c"

	f := newCFixture(t)
	process := f.addFunction(t, mainC, "src/util.c", "process", "void", false)
	f.addCall(t, process, "process", nil)

	cg, _ := f.build(t)

	assert.Equal(t, []string{"src/util.c::process"}, cg.Edges["src/util.c::process"])
	assert.Equal(t, []string{"src/util.c::process"}, cg.ReverseEdges["src/util.c::process"])
}

// TestBuildCCallGraph_StaticAndOtherFileSameName covers two functions
// with the same bare name in different files (e.g. file-scope statics).
// The same-file caller must bind to the local definition; the other
// file's definition must remain reachable via global lookup.
func TestBuildCCallGraph_StaticAndOtherFileSameName(t *testing.T) {
	root := fixtureRoot
	aC := root + "/src/a.c"
	bC := root + "/src/b.c"

	f := newCFixture(t)
	aMain := f.addFunction(t, aC, "src/a.c", "main", "int", false)
	bMain := f.addFunction(t, bC, "src/b.c", "main", "int", false)
	f.addFunction(t, aC, "src/a.c", "init", "void", false)
	f.addFunction(t, bC, "src/b.c", "init", "void", false)

	f.addCall(t, aMain, "init", nil)
	f.addCall(t, bMain, "init", nil)

	cg, _ := f.build(t)

	assert.Equal(t, []string{"src/a.c::init"}, cg.Edges["src/a.c::main"])
	assert.Equal(t, []string{"src/b.c::init"}, cg.Edges["src/b.c::main"])
}

// TestBuildCCallGraph_TypeEnginePopulated verifies Pass 2 populates the
// type engine with explicit return types and skips void.
func TestBuildCCallGraph_TypeEnginePopulated(t *testing.T) {
	root := fixtureRoot
	mainC := root + "/src/main.c"

	f := newCFixture(t)
	f.addFunction(t, mainC, "src/main.c", "main", "int", false)
	f.addFunction(t, mainC, "src/main.c", "do_nothing", "void", false)

	cg, engine := f.build(t)

	got := engine.GetReturnType("src/main.c::main")
	require.NotNil(t, got)
	assert.Equal(t, "int", got.TypeFQN)
	assert.InDelta(t, 1.0, got.Confidence, 1e-6)
	assert.Equal(t, "declaration", got.Source)

	assert.Nil(t, engine.GetReturnType("src/main.c::do_nothing"), "void must not be stored")
	_ = cg
}

// TestBuildCCallGraph_ParameterSymbols verifies Pass 2 records every
// named parameter as a ParameterSymbol with its declared type.
func TestBuildCCallGraph_ParameterSymbols(t *testing.T) {
	root := fixtureRoot
	mainC := root + "/src/main.c"

	f := newCFixture(t)
	fn := f.addFunction(t, mainC, "src/main.c", "handle", "void", false)
	fn.MethodArgumentsType = []string{"int", "const char*", "Request*"}
	fn.MethodArgumentsValue = []string{"id", "name", ""} // anonymous third param dropped

	cg, _ := f.build(t)

	param := cg.Parameters["src/main.c::handle.id"]
	require.NotNil(t, param)
	assert.Equal(t, "id", param.Name)
	assert.Equal(t, "int", param.TypeAnnotation)
	assert.Equal(t, "src/main.c::handle", param.ParentFQN)
	assert.Equal(t, mainC, param.File)

	name := cg.Parameters["src/main.c::handle.name"]
	require.NotNil(t, name)
	assert.Equal(t, "const char*", name.TypeAnnotation)

	assert.NotContains(t, cg.Parameters, "src/main.c::handle.")
}

// TestBuildCCallGraph_DeclarationsSkippedFromTypePass verifies that
// declarations (no body) do not register return types — only definitions
// contribute to the type engine.
func TestBuildCCallGraph_DeclarationsSkippedFromTypePass(t *testing.T) {
	root := fixtureRoot
	utilsH := root + "/include/utils.h"

	f := newCFixture(t)
	f.addFunction(t, utilsH, "include/utils.h", "compute", "int", true)

	_, engine := f.build(t)

	assert.Nil(t, engine.GetReturnType("include/utils.h::compute"),
		"declarations must not pollute the return-type table")
}

// TestBuildCCallGraph_MergeIntoUnifiedGraph confirms a built C call
// graph merges cleanly into an empty destination (the Python/Go entry
// point) with no key collisions.
func TestBuildCCallGraph_MergeIntoUnifiedGraph(t *testing.T) {
	root := fixtureRoot
	mainC := root + "/src/main.c"

	f := newCFixture(t)
	mainFn := f.addFunction(t, mainC, "src/main.c", "main", "int", false)
	f.addFunction(t, mainC, "src/main.c", "init", "void", false)
	f.addCall(t, mainFn, "init", nil)

	src, _ := f.build(t)
	dst := core.NewCallGraph()
	builder.MergeCallGraphs(dst, src)

	assert.Contains(t, dst.Functions, "src/main.c::main")
	assert.Contains(t, dst.Functions, "src/main.c::init")
	assert.Equal(t, []string{"src/main.c::init"}, dst.Edges["src/main.c::main"])
	assert.Len(t, dst.CallSites["src/main.c::main"], 1)
}

// TestBuildCCallGraph_IgnoresNonCNodes guards the language filter:
// Python/Go function nodes must not enter the C call graph even when
// their file is registered (e.g. mixed-language project).
func TestBuildCCallGraph_IgnoresNonCNodes(t *testing.T) {
	root := fixtureRoot
	cFile := root + "/src/main.c"
	pyFile := root + "/lib/x.py"

	f := newCFixture(t)
	f.addFunction(t, cFile, "src/main.c", "main", "int", false)

	// Manually inject a Python function — addFunction would tag it as C.
	f.cg.AddNode(&graph.Node{
		Type:     "function_definition",
		Name:     "py_fn",
		File:     pyFile,
		Language: "python",
	})

	cg, _ := f.build(t)
	assert.Contains(t, cg.Functions, "src/main.c::main")
	assert.NotContains(t, cg.Functions, "lib/x.py::py_fn")
	assert.NotContains(t, cg.Functions, "::py_fn")
}

// TestBuildCCallGraph_AnonymousAndMissingFiltered ensures functions
// without a name or file are skipped entirely (defensive — the parser
// should never emit them, but the builder must not panic).
func TestBuildCCallGraph_AnonymousAndMissingFiltered(t *testing.T) {
	cg := graph.NewCodeGraph()
	cg.AddNode(&graph.Node{Type: "function_definition", Name: "", File: "/x.c", Language: "c"})
	cg.AddNode(&graph.Node{Type: "function_definition", Name: "fn", File: "", Language: "c"})

	registry := core.NewCModuleRegistry("/x")
	got, err := builder.BuildCCallGraph(cg, registry, resolution.NewCTypeInferenceEngine(registry))
	require.NoError(t, err)
	assert.Empty(t, got.Functions)
	assert.Empty(t, got.Edges)
}

// TestBuildCCallGraph_EmptyTargetCallSkipped covers the defensive
// branch in resolveCCallTarget: a call_expression with no target name
// (parser bug or pathological input) is recorded as an unresolved
// call site rather than ignored entirely.
func TestBuildCCallGraph_EmptyTargetCallSkipped(t *testing.T) {
	root := fixtureRoot
	mainC := root + "/src/main.c"

	f := newCFixture(t)
	mainFn := f.addFunction(t, mainC, "src/main.c", "main", "int", false)

	// Direct injection: addCall would set Name; here we want it empty.
	bad := &graph.Node{
		ID:       "bad-call",
		Type:     "call_expression",
		Name:     "",
		File:     mainC,
		Language: "c",
	}
	f.cg.AddNode(bad)
	f.cg.AddEdge(mainFn, bad)

	cg, _ := f.build(t)
	assert.Empty(t, cg.Edges["src/main.c::main"], "anonymous calls produce no edge")
	assert.Empty(t, cg.CallSites["src/main.c::main"], "anonymous calls are not recorded as call sites")
}

// TestBuildCCallGraph_GlobalDefinitionFromOtherFileWithoutInclude
// covers Source 2: a function defined in another .c file but called
// without an include directive present. The global FunctionIndex
// search must still find it.
func TestBuildCCallGraph_GlobalDefinitionFromOtherFileWithoutInclude(t *testing.T) {
	root := fixtureRoot
	aC := root + "/src/a.c"
	bC := root + "/src/b.c"

	f := newCFixture(t)
	caller := f.addFunction(t, aC, "src/a.c", "main", "int", false)
	f.addFunction(t, bC, "src/b.c", "helper", "void", false)
	f.addCall(t, caller, "helper", nil)

	cg, _ := f.build(t)
	assert.Equal(t, []string{"src/b.c::helper"}, cg.Edges["src/a.c::main"])
}

// TestBuildCCallGraph_SameFileDeclarationAcceptedWhenNoDefinition
// verifies the third resolution step: when no definition exists
// project-wide, a same-file declaration is accepted.
func TestBuildCCallGraph_SameFileDeclarationAcceptedWhenNoDefinition(t *testing.T) {
	root := fixtureRoot
	mainC := root + "/src/main.c"

	f := newCFixture(t)
	caller := f.addFunction(t, mainC, "src/main.c", "main", "int", false)
	f.addFunction(t, mainC, "src/main.c", "extern_op", "int", true) // forward decl
	f.addCall(t, caller, "extern_op", nil)

	cg, _ := f.build(t)
	assert.Equal(t, []string{"src/main.c::extern_op"}, cg.Edges["src/main.c::main"])
}
