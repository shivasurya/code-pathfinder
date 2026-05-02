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

// cppFixture builds a CodeGraph + CppModuleRegistry + type engine for
// driving BuildCppCallGraph in unit tests. Tests describe the program
// shape via addClass / addMethod / addFreeFunction / addCall helpers;
// the registry's FileToPrefix is updated automatically so cross-file
// resolution matches what BuildCppModuleRegistry would have produced.
type cppFixture struct {
	cg       *graph.CodeGraph
	registry *core.CppModuleRegistry
	engine   *resolution.CppTypeInferenceEngine
}

const cppFixtureRoot = "/projects/cppapp"

func newCppFixture(t *testing.T) *cppFixture {
	t.Helper()
	registry := core.NewCppModuleRegistry(cppFixtureRoot)
	return &cppFixture{
		cg:       graph.NewCodeGraph(),
		registry: registry,
		engine:   resolution.NewCppTypeInferenceEngine(registry),
	}
}

func (f *cppFixture) addClass(t *testing.T, file, relPath, packageName, name string, startByte, endByte uint32) {
	t.Helper()
	node := &graph.Node{
		ID:             "class:" + relPath + "::" + packageName + "::" + name,
		Type:           "class_declaration",
		Name:           name,
		File:           file,
		Language:       "cpp",
		PackageName:    packageName,
		LineNumber:     1,
		SourceLocation: &graph.SourceLocation{File: file, StartByte: startByte, EndByte: endByte},
	}
	f.cg.AddNode(node)
	f.registry.FileToPrefix[file] = relPath
	scope := packageName
	if scope != "" {
		scope = scope + "::" + name
	} else {
		scope = name
	}
	f.registry.ClassIndex[name] = append(f.registry.ClassIndex[name], relPath+"::"+scope)
}

// addMethod adds a method_declaration anchored at the given byte offset
// inside (or outside) any class declared via addClass. The startByte
// is what enclosingCppClass uses to associate the method with its class.
func (f *cppFixture) addMethod(t *testing.T, file, relPath, packageName, name, returnType string, startByte, endByte uint32) *graph.Node {
	t.Helper()
	node := &graph.Node{
		ID:             "method:" + relPath + "::" + packageName + "::" + name,
		Type:           "method_declaration",
		Name:           name,
		File:           file,
		Language:       "cpp",
		PackageName:    packageName,
		ReturnType:     returnType,
		LineNumber:     1,
		SourceLocation: &graph.SourceLocation{File: file, StartByte: startByte, EndByte: endByte},
	}
	f.cg.AddNode(node)
	f.registry.FileToPrefix[file] = relPath
	return node
}

// addFreeFunction adds a function_definition outside any class.
// packageName may be "" (top-level) or a namespace name.
func (f *cppFixture) addFreeFunction(t *testing.T, file, relPath, packageName, name, returnType string) *graph.Node {
	t.Helper()
	node := &graph.Node{
		ID:             "fn:" + relPath + "::" + packageName + "::" + name,
		Type:           "function_definition",
		Name:           name,
		File:           file,
		Language:       "cpp",
		PackageName:    packageName,
		ReturnType:     returnType,
		LineNumber:     1,
		SourceLocation: &graph.SourceLocation{File: file, StartByte: 9000, EndByte: 9100},
	}
	f.cg.AddNode(node)
	f.registry.FileToPrefix[file] = relPath
	return node
}

// addCall mimics the parser's edge from a containing function to a
// call_expression. metadata fields are sparse so tests can exercise
// the qualified / method / receiver branches.
func (f *cppFixture) addCall(t *testing.T, caller *graph.Node, target string, receiver string) {
	t.Helper()
	metadata := map[string]any{}
	if receiver != "" {
		metadata["is_method"] = true
		metadata["receiver"] = receiver
	}
	call := &graph.Node{
		ID:         "call:" + caller.ID + "->" + target + "@" + receiver,
		Type:       "call_expression",
		Name:       target,
		File:       caller.File,
		Language:   "cpp",
		LineNumber: caller.LineNumber + 1,
		Metadata:   metadata,
	}
	f.cg.AddNode(call)
	f.cg.AddEdge(caller, call)
}

func (f *cppFixture) build(t *testing.T) *core.CallGraph {
	t.Helper()
	cg, err := builder.BuildCppCallGraph(f.cg, f.registry, f.engine)
	require.NoError(t, err)
	require.NotNil(t, cg)
	return cg
}

func TestBuildCppCallGraph_NilInputs(t *testing.T) {
	cg, err := builder.BuildCppCallGraph(nil, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, cg)
	assert.Empty(t, cg.Functions)
	assert.Empty(t, cg.Edges)
}

// TestBuildCppCallGraph_NamespaceQualifiedCall covers `ns::func()` —
// the call's Name already carries the qualifier, so resolution is a
// direct NamespaceIndex lookup.
func TestBuildCppCallGraph_NamespaceQualifiedCall(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"
	utilsCpp := root + "/src/utils.cpp"

	f := newCppFixture(t)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")
	f.addFreeFunction(t, utilsCpp, "src/utils.cpp", "mylib", "process", "void")
	f.addCall(t, main, "mylib::process", "")

	cg := f.build(t)

	assert.Equal(t, []string{"src/utils.cpp::mylib::process"}, cg.Edges["src/main.cpp::main"])
	sites := cg.CallSites["src/main.cpp::main"]
	require.Len(t, sites, 1)
	assert.True(t, sites[0].Resolved)
	assert.Equal(t, "src/utils.cpp::mylib::process", sites[0].TargetFQN)
}

// TestBuildCppCallGraph_StaticMethodViaNamespaceIndex covers
// `ClassName::staticMethod()` — the call's Name already contains the
// `::` and resolves via NamespaceIndex without needing a class lookup.
func TestBuildCppCallGraph_StaticMethodViaNamespaceIndex(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"
	socketCpp := root + "/src/socket.cpp"

	f := newCppFixture(t)
	f.addClass(t, socketCpp, "src/socket.cpp", "", "Socket", 0, 1000)
	f.addMethod(t, socketCpp, "src/socket.cpp", "", "create", "Socket*", 100, 200)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")
	f.addCall(t, main, "Socket::create", "")

	cg := f.build(t)

	assert.Equal(t, []string{"src/socket.cpp::Socket::create"}, cg.Edges["src/main.cpp::main"])
}

// TestBuildCppCallGraph_MethodOnTypedReceiver covers `obj.method()`
// where `obj` was declared as `Dog*` inside the caller. Resolution
// goes through the type engine to discover `Dog`, then through
// NamespaceIndex to find `Dog::speak`.
func TestBuildCppCallGraph_MethodOnTypedReceiver(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"
	animalCpp := root + "/src/animal.cpp"

	f := newCppFixture(t)
	f.addClass(t, animalCpp, "src/animal.cpp", "mylib", "Dog", 0, 1000)
	f.addMethod(t, animalCpp, "src/animal.cpp", "mylib", "speak", "void", 100, 200)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")

	// The call site is obj.speak(); register the receiver type via the
	// type engine so resolution can find it.
	f.engine.ExtractVariableType("src/main.cpp::main", "dog", "Dog*", resolution.Location{Line: 5})
	f.addCall(t, main, "speak", "dog")

	cg := f.build(t)

	assert.Equal(t, []string{"src/animal.cpp::mylib::Dog::speak"}, cg.Edges["src/main.cpp::main"])
	sites := cg.CallSites["src/main.cpp::main"]
	require.Len(t, sites, 1)
	assert.True(t, sites[0].Resolved)
}

// TestBuildCppCallGraph_ThisMethodCall covers `this->method()` inside
// a method body — the receiver type is implicit (the caller's
// enclosing class).
func TestBuildCppCallGraph_ThisMethodCall(t *testing.T) {
	root := cppFixtureRoot
	socketCpp := root + "/src/socket.cpp"

	f := newCppFixture(t)
	f.addClass(t, socketCpp, "src/socket.cpp", "", "Socket", 0, 1000)
	connect := f.addMethod(t, socketCpp, "src/socket.cpp", "", "connect", "void", 100, 300)
	f.addMethod(t, socketCpp, "src/socket.cpp", "", "disconnect", "void", 400, 500)
	f.addCall(t, connect, "disconnect", "this")

	cg := f.build(t)

	assert.Equal(t, []string{"src/socket.cpp::Socket::disconnect"},
		cg.Edges["src/socket.cpp::Socket::connect"])
}

// TestBuildCppCallGraph_FreeFunctionFallthrough verifies that plain
// (non-qualified, non-method) calls fall through to the C-style
// resolution and find a free C++ function in the same file.
func TestBuildCppCallGraph_FreeFunctionFallthrough(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"

	f := newCppFixture(t)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")
	f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "helper", "void")
	f.addCall(t, main, "helper", "")

	cg := f.build(t)

	assert.Equal(t, []string{"src/main.cpp::helper"}, cg.Edges["src/main.cpp::main"])
}

// TestBuildCppCallGraph_StdlibCallUnresolved confirms unknown calls
// (e.g. `printf`) are recorded as Resolved:false rather than dropped.
func TestBuildCppCallGraph_StdlibCallUnresolved(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"

	f := newCppFixture(t)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")
	f.addCall(t, main, "printf", "")

	cg := f.build(t)
	assert.Empty(t, cg.Edges["src/main.cpp::main"])
	sites := cg.CallSites["src/main.cpp::main"]
	require.Len(t, sites, 1)
	assert.False(t, sites[0].Resolved)
	assert.NotEmpty(t, sites[0].FailureReason)
}

// TestBuildCppCallGraph_ReceiverNotInScope falls back gracefully when
// the receiver variable was not registered with the type engine.
func TestBuildCppCallGraph_ReceiverNotInScope(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"

	f := newCppFixture(t)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")
	// dog.speak() but no type registered for `dog`.
	f.addCall(t, main, "speak", "dog")

	cg := f.build(t)
	assert.Empty(t, cg.Edges["src/main.cpp::main"], "missing receiver type must not produce an edge")
	require.Len(t, cg.CallSites["src/main.cpp::main"], 1)
	assert.False(t, cg.CallSites["src/main.cpp::main"][0].Resolved)
}

// TestBuildCppCallGraph_MethodReturnTypeRegistered verifies Pass 2
// records class method return types on the type engine.
func TestBuildCppCallGraph_MethodReturnTypeRegistered(t *testing.T) {
	root := cppFixtureRoot
	socketCpp := root + "/src/socket.cpp"

	f := newCppFixture(t)
	f.addClass(t, socketCpp, "src/socket.cpp", "", "Socket", 0, 1000)
	f.addMethod(t, socketCpp, "src/socket.cpp", "", "is_open", "bool", 100, 200)
	f.addMethod(t, socketCpp, "src/socket.cpp", "", "destroy", "void", 300, 400) // dropped
	f.build(t)

	got := f.engine.GetMethodReturnType("Socket", "is_open")
	require.NotNil(t, got)
	assert.Equal(t, "bool", got.TypeFQN)
	assert.Nil(t, f.engine.GetMethodReturnType("Socket", "destroy"), "void method must not be registered")
}

// TestBuildCppCallGraph_FieldTypeRegistered verifies Pass 2 records
// class field types on the type engine.
func TestBuildCppCallGraph_FieldTypeRegistered(t *testing.T) {
	root := cppFixtureRoot
	socketCpp := root + "/src/socket.cpp"

	f := newCppFixture(t)
	f.addClass(t, socketCpp, "src/socket.cpp", "", "Socket", 0, 1000)
	field := &graph.Node{
		ID:             "field:port",
		Type:           "field_declaration",
		Name:           "port",
		Language:       "cpp",
		File:           socketCpp,
		DataType:       "int",
		SourceLocation: &graph.SourceLocation{File: socketCpp, StartByte: 50, EndByte: 60},
	}
	f.cg.AddNode(field)

	f.build(t)
	got := f.engine.GetFieldType("Socket", "port")
	require.NotNil(t, got)
	assert.Equal(t, "int", got.TypeFQN)
}

// TestBuildCppCallGraph_NormaliseTypeName covers the receiver-type
// stripping logic by exercising every common qualifier shape.
func TestBuildCppCallGraph_NormaliseTypeName(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"
	animalCpp := root + "/src/animal.cpp"

	cases := []struct {
		name        string
		typeStr     string
		varName     string
		expectedFQN string
	}{
		{"plain pointer", "Dog*", "dog", "src/animal.cpp::Dog::speak"},
		{"const pointer", "const Dog*", "dog", "src/animal.cpp::Dog::speak"},
		{"reference", "Dog&", "dog", "src/animal.cpp::Dog::speak"},
		{"double pointer", "Dog**", "dog", "src/animal.cpp::Dog::speak"},
		{"rvalue reference", "Dog&&", "dog", "src/animal.cpp::Dog::speak"},
		{"value with whitespace", "  Dog  ", "dog", "src/animal.cpp::Dog::speak"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCppFixture(t)
			f.addClass(t, animalCpp, "src/animal.cpp", "", "Dog", 0, 1000)
			f.addMethod(t, animalCpp, "src/animal.cpp", "", "speak", "void", 100, 200)
			main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")

			f.engine.ExtractVariableType("src/main.cpp::main", tc.varName, tc.typeStr, resolution.Location{Line: 1})
			f.addCall(t, main, "speak", tc.varName)

			cg := f.build(t)
			require.Len(t, cg.Edges["src/main.cpp::main"], 1)
			assert.Equal(t, tc.expectedFQN, cg.Edges["src/main.cpp::main"][0])
		})
	}
}

// TestBuildCppCallGraph_NestedClasses confirms enclosingCppClass picks
// the innermost class when classes are nested.
func TestBuildCppCallGraph_NestedClasses(t *testing.T) {
	root := cppFixtureRoot
	src := root + "/src/nested.cpp"

	f := newCppFixture(t)
	f.addClass(t, src, "src/nested.cpp", "", "Outer", 0, 1000)
	f.addClass(t, src, "src/nested.cpp", "", "Inner", 100, 500)
	// Method anchored inside the inner class' range.
	inner := f.addMethod(t, src, "src/nested.cpp", "", "ping", "void", 200, 300)
	main := f.addFreeFunction(t, root+"/src/main.cpp", "src/main.cpp", "", "main", "int")
	f.addCall(t, main, "Inner::ping", "")

	cg := f.build(t)
	assert.Equal(t, []string{"src/nested.cpp::Inner::ping"}, cg.Edges["src/main.cpp::main"])
	_ = inner
}

// TestBuildCppCallGraph_MergeIntoUnifiedGraph confirms a built C++
// call graph merges cleanly into an empty destination, replicating
// what cmd/scan.go will do when assembling the unified graph.
func TestBuildCppCallGraph_MergeIntoUnifiedGraph(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"

	f := newCppFixture(t)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")
	f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "helper", "void")
	f.addCall(t, main, "helper", "")

	src := f.build(t)
	dst := core.NewCallGraph()
	builder.MergeCallGraphs(dst, src)

	assert.Contains(t, dst.Functions, "src/main.cpp::main")
	assert.Contains(t, dst.Functions, "src/main.cpp::helper")
	assert.Equal(t, []string{"src/main.cpp::helper"}, dst.Edges["src/main.cpp::main"])
}

// TestBuildCppCallGraph_FindMethodOnClass_NamespacedSuffixMatch
// covers the suffix fallback in findMethodOnClass: a class lives in a
// namespace and the call site has only the bare class name.
func TestBuildCppCallGraph_FindMethodOnClass_NamespacedSuffixMatch(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"
	libCpp := root + "/src/lib.cpp"

	f := newCppFixture(t)
	f.addClass(t, libCpp, "src/lib.cpp", "ns", "Foo", 0, 1000)
	f.addMethod(t, libCpp, "src/lib.cpp", "ns", "bar", "void", 100, 200)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")

	// Receiver registered as bare `Foo` (no namespace prefix).
	f.engine.ExtractVariableType("src/main.cpp::main", "foo", "Foo*", resolution.Location{})
	f.addCall(t, main, "bar", "foo")

	cg := f.build(t)
	assert.Equal(t, []string{"src/lib.cpp::ns::Foo::bar"}, cg.Edges["src/main.cpp::main"])
}

// TestBuildCppCallGraph_ReceiverBindingMissingType covers the
// defensive branch in lookupReceiverMethod: a binding exists for the
// receiver but Type is nil. This shouldn't happen for parser output
// but the resolver must not panic.
func TestBuildCppCallGraph_ReceiverBindingMissingType(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"

	f := newCppFixture(t)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")

	// Manually inject a binding with no Type.
	scope := resolution.NewCFunctionScope("src/main.cpp::main")
	scope.AddVariable(&resolution.CVariableBinding{VarName: "obj", Type: nil})
	f.engine.AddScope(scope)

	f.addCall(t, main, "method", "obj")

	cg := f.build(t)
	assert.Empty(t, cg.Edges["src/main.cpp::main"])
	require.Len(t, cg.CallSites["src/main.cpp::main"], 1)
	assert.False(t, cg.CallSites["src/main.cpp::main"][0].Resolved)
}

// TestBuildCppCallGraph_ThisOutsideKnownCaller covers the defensive
// branch in lookupThisMethod where the call's CallerFQN is not in
// callGraph.Functions. The resolver must skip the `this` path and
// fall through cleanly.
func TestBuildCppCallGraph_ThisOutsideKnownCaller(t *testing.T) {
	root := cppFixtureRoot
	socketCpp := root + "/src/socket.cpp"

	f := newCppFixture(t)
	// A free function (not a method) calling `this->method()` is
	// nonsensical in C++ but exercises the `caller not a class member`
	// branch of lookupThisMethod via byte-range miss.
	main := f.addFreeFunction(t, socketCpp, "src/socket.cpp", "", "main", "int")
	f.addCall(t, main, "stray", "this")

	cg := f.build(t)
	assert.Empty(t, cg.Edges["src/socket.cpp::main"])
	require.Len(t, cg.CallSites["src/socket.cpp::main"], 1)
	assert.False(t, cg.CallSites["src/socket.cpp::main"][0].Resolved)
}

// TestBuildCppCallGraph_DeclarationStillRegistersClassMethod verifies
// Pass 2 registers method return types from declarations even though
// declarations themselves don't get return-type entries on the function
// map (the C builder skips them) — class method tracking is needed for
// Pass 4 receiver resolution to work on header-declared methods.
func TestBuildCppCallGraph_DeclarationStillRegistersClassMethod(t *testing.T) {
	root := cppFixtureRoot
	socketHpp := root + "/include/socket.hpp"

	f := newCppFixture(t)
	f.addClass(t, socketHpp, "include/socket.hpp", "", "Socket", 0, 1000)
	// Declaration: no body, marked is_declaration.
	decl := f.addMethod(t, socketHpp, "include/socket.hpp", "", "is_open", "bool", 100, 200)
	decl.Metadata = map[string]any{"is_declaration": true}

	f.build(t)
	got := f.engine.GetMethodReturnType("Socket", "is_open")
	require.NotNil(t, got)
	assert.Equal(t, "bool", got.TypeFQN)
}

// TestBuildCppCallGraph_IgnoresNonCppNodes verifies the language
// filter — Python or C nodes must not enter the C++ call graph.
func TestBuildCppCallGraph_IgnoresNonCppNodes(t *testing.T) {
	root := cppFixtureRoot
	cppFile := root + "/src/main.cpp"
	cFile := root + "/src/legacy.c"

	f := newCppFixture(t)
	f.addFreeFunction(t, cppFile, "src/main.cpp", "", "main", "int")
	f.cg.AddNode(&graph.Node{
		Type:     "function_definition",
		Name:     "legacy_init",
		File:     cFile,
		Language: "c",
	})

	cg := f.build(t)
	assert.Contains(t, cg.Functions, "src/main.cpp::main")
	assert.NotContains(t, cg.Functions, "src/legacy.c::legacy_init")
}
