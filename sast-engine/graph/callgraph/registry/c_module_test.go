package registry_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeNode returns a *graph.Node populated with the fields used by the
// module-registry build path. Tests build small CodeGraphs from these
// nodes rather than invoking the parser, so the test stays focused on
// registry behaviour and is independent of tree-sitter.
func makeNode(t *testing.T, n graph.Node) *graph.Node {
	t.Helper()
	if n.ID == "" {
		n.ID = n.Type + ":" + n.Name + "@" + n.File
	}
	if n.SourceLocation == nil {
		n.SourceLocation = &graph.SourceLocation{File: n.File}
	}
	return &n
}

func newGraphFromNodes(nodes ...*graph.Node) *graph.CodeGraph {
	cg := graph.NewCodeGraph()
	for _, n := range nodes {
		cg.AddNode(n)
	}
	return cg
}

// TestBuildCModuleRegistry_FilesAndFunctions verifies that BuildCModuleRegistry
// (a) maps every distinct C source file to its project-relative prefix and
// (b) indexes every function_definition node under its bare name with the
// correct "relpath::funcname" FQN format.
func TestBuildCModuleRegistry_FilesAndFunctions(t *testing.T) {
	root := "/projects/myapp"

	main := makeNode(t, graph.Node{Type: "function_definition", Name: "main", Language: "c", File: root + "/src/main.c"})
	helper := makeNode(t, graph.Node{Type: "function_definition", Name: "helper", Language: "c", File: root + "/src/main.c"})
	createBuf := makeNode(t, graph.Node{Type: "function_definition", Name: "create_buffer", Language: "c", File: root + "/src/buffer.c"})
	freeBuf := makeNode(t, graph.Node{Type: "function_definition", Name: "free_buffer", Language: "c", File: root + "/src/buffer.c"})
	process := makeNode(t, graph.Node{Type: "function_definition", Name: "process", Language: "c", File: root + "/src/util.c"})

	// Decoy nodes to confirm filtering: wrong language, wrong type, empty file/name.
	pyFunc := makeNode(t, graph.Node{Type: "function_definition", Name: "ignored", Language: "python", File: root + "/lib/x.py"})
	cppFunc := makeNode(t, graph.Node{Type: "function_definition", Name: "ignored_cpp", Language: "cpp", File: root + "/src/x.cpp"})
	emptyName := makeNode(t, graph.Node{Type: "function_definition", Name: "", Language: "c", File: root + "/src/empty.c"})
	emptyFile := makeNode(t, graph.Node{Type: "function_definition", Name: "no_file", Language: "c", File: ""})

	cg := newGraphFromNodes(main, helper, createBuf, freeBuf, process, pyFunc, cppFunc, emptyName, emptyFile)
	reg := registry.BuildCModuleRegistry(root, cg)

	require.NotNil(t, reg)

	// emptyFile is dropped (no file), but emptyName still seeds FileToPrefix
	// because its file path is valid — this matches the parser contract
	// where an unnamed declaration can still anchor its translation unit.
	assert.Equal(t, "src/main.c", reg.FileToPrefix[root+"/src/main.c"])
	assert.Equal(t, "src/buffer.c", reg.FileToPrefix[root+"/src/buffer.c"])
	assert.Equal(t, "src/util.c", reg.FileToPrefix[root+"/src/util.c"])
	assert.Equal(t, "src/empty.c", reg.FileToPrefix[root+"/src/empty.c"])
	assert.NotContains(t, reg.FileToPrefix, root+"/lib/x.py", "python files must not appear")
	assert.NotContains(t, reg.FileToPrefix, root+"/src/x.cpp", "cpp files must not appear in C registry")

	assert.ElementsMatch(t, []string{"src/main.c::main"}, reg.FunctionIndex["main"])
	assert.ElementsMatch(t, []string{"src/main.c::helper"}, reg.FunctionIndex["helper"])
	assert.ElementsMatch(t, []string{"src/buffer.c::create_buffer"}, reg.FunctionIndex["create_buffer"])
	assert.ElementsMatch(t, []string{"src/buffer.c::free_buffer"}, reg.FunctionIndex["free_buffer"])
	assert.ElementsMatch(t, []string{"src/util.c::process"}, reg.FunctionIndex["process"])
	assert.NotContains(t, reg.FunctionIndex, "ignored")
	assert.NotContains(t, reg.FunctionIndex, "ignored_cpp")
}

// TestBuildCModuleRegistry_DuplicateFunctionAcrossFiles verifies that a
// function with the same name in both a header (declaration) and a source
// file (definition) produces TWO FQN entries in FunctionIndex — the
// registry deliberately preserves duplicates so the call-graph builder
// can choose between header and source.
func TestBuildCModuleRegistry_DuplicateFunctionAcrossFiles(t *testing.T) {
	root := "/projects/myapp"
	header := makeNode(t, graph.Node{Type: "function_definition", Name: "create_buffer", Language: "c", File: root + "/include/buffer.h"})
	source := makeNode(t, graph.Node{Type: "function_definition", Name: "create_buffer", Language: "c", File: root + "/src/buffer.c"})

	reg := registry.BuildCModuleRegistry(root, newGraphFromNodes(header, source))
	assert.ElementsMatch(t,
		[]string{"include/buffer.h::create_buffer", "src/buffer.c::create_buffer"},
		reg.FunctionIndex["create_buffer"],
	)
}

// TestBuildCModuleRegistry_DuplicateFunctionSameFile guards against the
// most common bug in indexes of this shape: the same function visited
// twice should not appear twice in FunctionIndex.
func TestBuildCModuleRegistry_DuplicateFunctionSameFile(t *testing.T) {
	root := "/projects/myapp"
	a := makeNode(t, graph.Node{ID: "fdup-1", Type: "function_definition", Name: "init", Language: "c", File: root + "/src/init.c"})
	b := makeNode(t, graph.Node{ID: "fdup-2", Type: "function_definition", Name: "init", Language: "c", File: root + "/src/init.c"})

	reg := registry.BuildCModuleRegistry(root, newGraphFromNodes(a, b))
	assert.ElementsMatch(t, []string{"src/init.c::init"}, reg.FunctionIndex["init"],
		"same file + same name must dedupe to one FQN")
}

// TestBuildCModuleRegistry_OutsideProjectRoot verifies that files which
// resolve to a `..`-prefixed relative path (i.e. outside the project)
// are skipped entirely.
func TestBuildCModuleRegistry_OutsideProjectRoot(t *testing.T) {
	root := "/projects/myapp"
	outside := makeNode(t, graph.Node{Type: "function_definition", Name: "external", Language: "c", File: "/projects/other/src/x.c"})
	inside := makeNode(t, graph.Node{Type: "function_definition", Name: "main", Language: "c", File: root + "/src/main.c"})

	reg := registry.BuildCModuleRegistry(root, newGraphFromNodes(outside, inside))
	assert.NotContains(t, reg.FunctionIndex, "external")
	assert.NotContains(t, reg.FileToPrefix, "/projects/other/src/x.c")
	assert.Contains(t, reg.FunctionIndex, "main")
}

// TestBuildCModuleRegistry_EmptyGraph confirms an empty CodeGraph yields
// a non-nil registry with empty (but allocated) maps.
func TestBuildCModuleRegistry_EmptyGraph(t *testing.T) {
	reg := registry.BuildCModuleRegistry("/projects/empty", graph.NewCodeGraph())
	require.NotNil(t, reg)
	assert.Empty(t, reg.FileToPrefix)
	assert.Empty(t, reg.Includes)
	assert.Empty(t, reg.FunctionIndex)
	// nil-safety: a nil graph must not panic.
	reg2 := registry.BuildCModuleRegistry("/projects/nil", nil)
	require.NotNil(t, reg2)
	assert.Empty(t, reg2.FileToPrefix)
}

// TestBuildCppModuleRegistry_NamespaceAndClassIndex verifies the C++
// extension: free functions in a namespace land in NamespaceIndex,
// classes land in ClassIndex, and methods inside a class are indexed by
// "ns::Class::method".
func TestBuildCppModuleRegistry_NamespaceAndClassIndex(t *testing.T) {
	root := "/projects/cppapp"
	srcCpp := root + "/src/socket.cpp"

	// Class spans bytes [100, 400] in socket.cpp, namespace = mylib.
	socketClass := makeNode(t, graph.Node{
		Type:           "class_declaration",
		Name:           "Socket",
		Language:       "cpp",
		File:           srcCpp,
		PackageName:    "mylib",
		SourceLocation: &graph.SourceLocation{File: srcCpp, StartByte: 100, EndByte: 400},
	})
	// Method "connect" inside the class (StartByte 150 ∈ [100,400]).
	connect := makeNode(t, graph.Node{
		Type:           "method_declaration",
		Name:           "connect",
		Language:       "cpp",
		File:           srcCpp,
		PackageName:    "mylib",
		SourceLocation: &graph.SourceLocation{File: srcCpp, StartByte: 150, EndByte: 200},
	})
	// Free function "process" in namespace mylib, outside the class.
	process := makeNode(t, graph.Node{
		Type:           "function_definition",
		Name:           "process",
		Language:       "cpp",
		File:           root + "/src/utils.cpp",
		PackageName:    "mylib",
		SourceLocation: &graph.SourceLocation{File: root + "/src/utils.cpp", StartByte: 0, EndByte: 50},
	})
	// Free function "main" with NO namespace and NO class — must NOT add
	// double-colon prefixes (the regression covered by case #9).
	main := makeNode(t, graph.Node{
		Type:           "function_definition",
		Name:           "main",
		Language:       "cpp",
		File:           root + "/src/main.cpp",
		SourceLocation: &graph.SourceLocation{File: root + "/src/main.cpp", StartByte: 0, EndByte: 30},
	})

	reg := registry.BuildCppModuleRegistry(root, newGraphFromNodes(socketClass, connect, process, main))
	require.NotNil(t, reg)

	// FunctionIndex includes free functions only; methods do not appear here.
	assert.ElementsMatch(t, []string{"src/utils.cpp::process"}, reg.FunctionIndex["process"])
	assert.ElementsMatch(t, []string{"src/main.cpp::main"}, reg.FunctionIndex["main"])
	assert.NotContains(t, reg.FunctionIndex, "connect", "methods must not appear in FunctionIndex")

	// NamespaceIndex has one entry per qualified key.
	assert.Equal(t, "src/utils.cpp::mylib::process", reg.NamespaceIndex["mylib::process"])
	assert.Equal(t, "src/socket.cpp::mylib::Socket::connect", reg.NamespaceIndex["mylib::Socket::connect"])
	assert.NotContains(t, reg.NamespaceIndex, "main", "free function with no namespace must not enter NamespaceIndex")

	// ClassIndex maps bare class name to FQN(s).
	assert.ElementsMatch(t, []string{"src/socket.cpp::mylib::Socket"}, reg.ClassIndex["Socket"])
}

// TestBuildCppModuleRegistry_ClassMethodWithoutNamespace covers the
// "class method, no namespace" FQN form: the method key must be
// "Class::method" and the FQN must NOT begin with a leading "::".
func TestBuildCppModuleRegistry_ClassMethodWithoutNamespace(t *testing.T) {
	root := "/projects/cppapp"
	src := root + "/src/app.cpp"

	appClass := makeNode(t, graph.Node{
		Type:           "class_declaration",
		Name:           "App",
		Language:       "cpp",
		File:           src,
		PackageName:    "",
		SourceLocation: &graph.SourceLocation{File: src, StartByte: 0, EndByte: 200},
	})
	run := makeNode(t, graph.Node{
		Type:           "method_declaration",
		Name:           "run",
		Language:       "cpp",
		File:           src,
		PackageName:    "",
		SourceLocation: &graph.SourceLocation{File: src, StartByte: 50, EndByte: 100},
	})

	reg := registry.BuildCppModuleRegistry(root, newGraphFromNodes(appClass, run))

	assert.Equal(t, "src/app.cpp::App::run", reg.NamespaceIndex["App::run"])
	assert.NotContains(t, reg.NamespaceIndex, "::App::run", "no leading separator")
	assert.Equal(t, []string{"src/app.cpp::App"}, reg.ClassIndex["App"])
}

// TestBuildCIncludeMap_LocalAndSystemIncludes spans every important
// include-resolution branch on a real on-disk layout:
//   - A header in include/ resolved by directory #2.
//   - A header in the same directory as the source resolved by #1
//     (and shadowing an alternative match in include/).
//   - A header in src/ resolved by #3.
//   - A header at the project root resolved by #4.
//   - A system include skipped.
//   - A missing header silently dropped.
func TestBuildCIncludeMap_LocalAndSystemIncludes(t *testing.T) {
	root := t.TempDir()

	// Layout:
	//   <root>/src/main.c
	//   <root>/src/local.h          (same-dir resolution, shadows include/local.h)
	//   <root>/include/local.h      (would otherwise win for "local.h")
	//   <root>/include/utils.h      (resolved by #2)
	//   <root>/src/extras.h         (resolved by #3)
	//   <root>/version.h            (resolved by #4)
	srcDir := filepath.Join(root, "src")
	includeDir := filepath.Join(root, "include")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.MkdirAll(includeDir, 0o755))
	for _, p := range []string{
		filepath.Join(srcDir, "main.c"),
		filepath.Join(srcDir, "local.h"),
		filepath.Join(includeDir, "local.h"),
		filepath.Join(includeDir, "utils.h"),
		filepath.Join(srcDir, "extras.h"),
		filepath.Join(root, "version.h"),
	} {
		require.NoError(t, os.WriteFile(p, []byte(""), 0o644))
	}

	mainC := filepath.Join(srcDir, "main.c")
	cg := newGraphFromNodes(
		makeNode(t, graph.Node{
			Type: "include_statement", Name: "local.h",
			Language: "c", File: mainC,
			Metadata: map[string]any{"system_include": false},
		}),
		makeNode(t, graph.Node{
			Type: "include_statement", Name: "utils.h",
			Language: "c", File: mainC,
			Metadata: map[string]any{"system_include": false},
		}),
		makeNode(t, graph.Node{
			Type: "include_statement", Name: "extras.h",
			Language: "c", File: mainC,
			Metadata: map[string]any{"system_include": false},
		}),
		makeNode(t, graph.Node{
			Type: "include_statement", Name: "version.h",
			Language: "c", File: mainC,
			Metadata: map[string]any{"system_include": false},
		}),
		makeNode(t, graph.Node{
			Type: "include_statement", Name: "stdio.h",
			Language: "c", File: mainC,
			Metadata: map[string]any{"system_include": true},
		}),
		makeNode(t, graph.Node{
			Type: "include_statement", Name: "missing.h",
			Language: "c", File: mainC,
			Metadata: map[string]any{"system_include": false},
		}),
	)

	includes := registry.BuildCIncludeMap(root, cg, "c")
	assert.ElementsMatch(t,
		[]string{"src/local.h", "include/utils.h", "src/extras.h", "version.h"},
		includes["src/main.c"],
	)
	// stdio.h must not appear; missing.h must not appear.
	for _, v := range includes["src/main.c"] {
		assert.NotEqual(t, "stdio.h", v)
		assert.NotEqual(t, "missing.h", v)
	}
}

// TestBuildCIncludeMap_AppearsInRegistry confirms BuildCModuleRegistry
// wires Includes through to the produced registry (not just the
// standalone helper). This is the integration that PR-07 will actually
// consume.
func TestBuildCIncludeMap_AppearsInRegistry(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "buddy.h"), []byte(""), 0o644))

	mainC := filepath.Join(srcDir, "main.c")
	require.NoError(t, os.WriteFile(mainC, []byte(""), 0o644))

	cg := newGraphFromNodes(
		makeNode(t, graph.Node{Type: "function_definition", Name: "main", Language: "c", File: mainC}),
		makeNode(t, graph.Node{
			Type: "include_statement", Name: "buddy.h",
			Language: "c", File: mainC,
			Metadata: map[string]any{"system_include": false},
		}),
	)

	reg := registry.BuildCModuleRegistry(root, cg)
	assert.ElementsMatch(t, []string{"src/buddy.h"}, reg.Includes["src/main.c"])
}

// TestBuildCIncludeMap_MissingMetadataTreatedAsLocal verifies behaviour
// when an include node lacks the `system_include` metadata entry: the
// function must default to project-local resolution rather than crash
// or skip silently.
func TestBuildCIncludeMap_MissingMetadataTreatedAsLocal(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "neighbour.h"), []byte(""), 0o644))

	mainC := filepath.Join(srcDir, "main.c")
	require.NoError(t, os.WriteFile(mainC, []byte(""), 0o644))

	cg := newGraphFromNodes(makeNode(t, graph.Node{
		Type: "include_statement", Name: "neighbour.h",
		Language: "c", File: mainC,
		// No Metadata at all.
	}))

	includes := registry.BuildCIncludeMap(root, cg, "c")
	assert.ElementsMatch(t, []string{"src/neighbour.h"}, includes["src/main.c"])
}

// TestBuildCIncludeMap_LanguageFilter confirms BuildCIncludeMap honours
// the language argument: a C++ include statement must NOT appear when
// the registry is built for C, and vice versa.
func TestBuildCIncludeMap_LanguageFilter(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "shared.h"), []byte(""), 0o644))

	cFile := filepath.Join(srcDir, "main.c")
	cppFile := filepath.Join(srcDir, "main.cpp")
	require.NoError(t, os.WriteFile(cFile, []byte(""), 0o644))
	require.NoError(t, os.WriteFile(cppFile, []byte(""), 0o644))

	cg := newGraphFromNodes(
		makeNode(t, graph.Node{
			Type: "include_statement", Name: "shared.h",
			Language: "c", File: cFile,
			Metadata: map[string]any{"system_include": false},
		}),
		makeNode(t, graph.Node{
			Type: "include_statement", Name: "shared.h",
			Language: "cpp", File: cppFile,
			Metadata: map[string]any{"system_include": false},
		}),
	)

	cIncludes := registry.BuildCIncludeMap(root, cg, "c")
	cppIncludes := registry.BuildCIncludeMap(root, cg, "cpp")

	assert.Contains(t, cIncludes, "src/main.c")
	assert.NotContains(t, cIncludes, "src/main.cpp")
	assert.Contains(t, cppIncludes, "src/main.cpp")
	assert.NotContains(t, cppIncludes, "src/main.c")
}

// TestBuildCppModuleRegistry_OutOfLineMethodBody covers a less-common
// shape: a function_definition whose byte range sits inside a class
// body. This is what tree-sitter produces for `void Foo::bar() {...}`
// defined at file scope but enclosed in a class block (e.g. inline
// header definitions). The registry must associate the function with
// its enclosing class and emit "Class::method" in NamespaceIndex.
func TestBuildCppModuleRegistry_OutOfLineMethodBody(t *testing.T) {
	root := "/projects/cppapp"
	src := root + "/include/widget.hpp"

	widgetClass := makeNode(t, graph.Node{
		Type:           "class_declaration",
		Name:           "Widget",
		Language:       "cpp",
		File:           src,
		PackageName:    "ui",
		SourceLocation: &graph.SourceLocation{File: src, StartByte: 0, EndByte: 500},
	})
	render := makeNode(t, graph.Node{
		Type:           "function_definition",
		Name:           "render",
		Language:       "cpp",
		File:           src,
		PackageName:    "ui",
		SourceLocation: &graph.SourceLocation{File: src, StartByte: 200, EndByte: 300},
	})

	reg := registry.BuildCppModuleRegistry(root, newGraphFromNodes(widgetClass, render))
	assert.Equal(t, "include/widget.hpp::ui::Widget::render", reg.NamespaceIndex["ui::Widget::render"])
	// The function still appears in FunctionIndex (free-function form)
	// because the registry preserves both views — call-graph resolution
	// can pick whichever fits.
	assert.ElementsMatch(t, []string{"include/widget.hpp::render"}, reg.FunctionIndex["render"])
}

// TestBuildCppModuleRegistry_FreeFunctionNoNamespaceNoClass verifies
// that a top-level C++ function with neither a namespace nor an
// enclosing class is recorded in FunctionIndex but NOT in
// NamespaceIndex (it has no qualified key to register under).
func TestBuildCppModuleRegistry_FreeFunctionNoNamespaceNoClass(t *testing.T) {
	root := "/projects/cppapp"
	src := root + "/src/standalone.cpp"

	helper := makeNode(t, graph.Node{
		Type:           "function_definition",
		Name:           "helper",
		Language:       "cpp",
		File:           src,
		SourceLocation: &graph.SourceLocation{File: src, StartByte: 10, EndByte: 50},
	})

	reg := registry.BuildCppModuleRegistry(root, newGraphFromNodes(helper))
	assert.ElementsMatch(t, []string{"src/standalone.cpp::helper"}, reg.FunctionIndex["helper"])
	assert.Empty(t, reg.NamespaceIndex, "no namespace + no class => no NamespaceIndex entry")
}

// TestBuildCppModuleRegistry_OrphanMethodWithNamespace covers the
// defensive fallback in indexCppMethod: a method_declaration that has a
// PackageName but no enclosing class node in the graph (this should not
// happen for well-formed input, but the registry must not drop it).
func TestBuildCppModuleRegistry_OrphanMethodWithNamespace(t *testing.T) {
	root := "/projects/cppapp"
	src := root + "/src/orphan.cpp"

	orphan := makeNode(t, graph.Node{
		Type:           "method_declaration",
		Name:           "lonely",
		Language:       "cpp",
		File:           src,
		PackageName:    "ghost",
		SourceLocation: &graph.SourceLocation{File: src, StartByte: 0, EndByte: 30},
	})

	reg := registry.BuildCppModuleRegistry(root, newGraphFromNodes(orphan))
	assert.Equal(t, "src/orphan.cpp::ghost::lonely", reg.NamespaceIndex["ghost::lonely"])
}

// TestBuildCppModuleRegistry_OrphanMethodNoNamespace checks the other
// branch of the defensive fallback: a method with neither a class nor a
// PackageName must be skipped from NamespaceIndex entirely.
func TestBuildCppModuleRegistry_OrphanMethodNoNamespace(t *testing.T) {
	root := "/projects/cppapp"
	src := root + "/src/orphan.cpp"

	orphan := makeNode(t, graph.Node{
		Type:           "method_declaration",
		Name:           "stray",
		Language:       "cpp",
		File:           src,
		SourceLocation: &graph.SourceLocation{File: src, StartByte: 0, EndByte: 30},
	})

	reg := registry.BuildCppModuleRegistry(root, newGraphFromNodes(orphan))
	assert.Empty(t, reg.NamespaceIndex)
}

// TestBuildCIncludeMap_EmptyHeaderName guards against include nodes
// with an empty Name field. The parser should never emit one, but the
// resolver must not stat("") and must not pollute the map.
func TestBuildCIncludeMap_EmptyHeaderName(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	mainC := filepath.Join(srcDir, "main.c")
	require.NoError(t, os.WriteFile(mainC, []byte(""), 0o644))

	cg := newGraphFromNodes(makeNode(t, graph.Node{
		Type: "include_statement", Name: "",
		Language: "c", File: mainC,
		Metadata: map[string]any{"system_include": false},
	}))

	includes := registry.BuildCIncludeMap(root, cg, "c")
	assert.Empty(t, includes)
}

// TestBuildCIncludeMap_HeaderIsDirectoryRejected confirms resolveLocalInclude
// rejects directory matches. A directory named the same as the header
// must not be returned as a resolved include.
func TestBuildCIncludeMap_HeaderIsDirectoryRejected(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "src")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	// A directory named "config.h" alongside main.c
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "config.h"), 0o755))

	mainC := filepath.Join(srcDir, "main.c")
	require.NoError(t, os.WriteFile(mainC, []byte(""), 0o644))

	cg := newGraphFromNodes(makeNode(t, graph.Node{
		Type: "include_statement", Name: "config.h",
		Language: "c", File: mainC,
		Metadata: map[string]any{"system_include": false},
	}))

	includes := registry.BuildCIncludeMap(root, cg, "c")
	assert.Empty(t, includes["src/main.c"], "a directory must not satisfy include resolution")
}

// TestBuildCppModuleRegistry_ClassWithDuplicateName covers the case of a
// class declared in both a header and a source file: ClassIndex must
// list both FQNs (the call-graph builder will choose by usage site).
func TestBuildCppModuleRegistry_ClassWithDuplicateName(t *testing.T) {
	root := "/projects/cppapp"

	headerSocket := makeNode(t, graph.Node{
		Type:           "class_declaration",
		Name:           "Socket",
		Language:       "cpp",
		File:           root + "/include/socket.hpp",
		PackageName:    "mylib",
		SourceLocation: &graph.SourceLocation{File: root + "/include/socket.hpp", StartByte: 0, EndByte: 100},
	})
	sourceSocket := makeNode(t, graph.Node{
		Type:           "class_declaration",
		Name:           "Socket",
		Language:       "cpp",
		File:           root + "/src/socket.cpp",
		PackageName:    "mylib",
		SourceLocation: &graph.SourceLocation{File: root + "/src/socket.cpp", StartByte: 0, EndByte: 200},
	})

	reg := registry.BuildCppModuleRegistry(root, newGraphFromNodes(headerSocket, sourceSocket))
	assert.ElementsMatch(t,
		[]string{"include/socket.hpp::mylib::Socket", "src/socket.cpp::mylib::Socket"},
		reg.ClassIndex["Socket"],
	)
}
