package builder_test

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCppStdlibLoader implements core.CppStdlibLoader for the
// builder tests. The unit tests construct the loader directly so the
// resolution path is observable in isolation.
type fakeCppStdlibLoader struct {
	headers map[string]*core.CStdlibHeader
}

func newFakeCppStdlibLoader(headers map[string]*core.CStdlibHeader) *fakeCppStdlibLoader {
	return &fakeCppStdlibLoader{headers: headers}
}

func (f *fakeCppStdlibLoader) LoadManifest(_ core.CStdlibLogger) error { return nil }

func (f *fakeCppStdlibLoader) GetHeader(name string) (*core.CStdlibHeader, error) {
	if h, ok := f.headers[name]; ok {
		return h, nil
	}
	return nil, assert.AnError
}

func (f *fakeCppStdlibLoader) GetFunction(headerName, funcName string) (*core.CStdlibFunction, error) {
	h, err := f.GetHeader(headerName)
	if err != nil {
		return nil, err
	}
	if fn, ok := h.Functions[funcName]; ok {
		return fn, nil
	}
	if fn, ok := h.FreeFunctions[funcName]; ok {
		return fn, nil
	}
	return nil, assert.AnError
}

func (f *fakeCppStdlibLoader) GetClass(headerName, classFQN string) (*core.CppStdlibClass, error) {
	h, err := f.GetHeader(headerName)
	if err != nil {
		return nil, err
	}
	if cls, ok := h.Classes[classFQN]; ok {
		return cls, nil
	}
	return nil, assert.AnError
}

func (f *fakeCppStdlibLoader) GetMethod(headerName, classFQN, methodName string) (*core.CStdlibFunction, error) {
	cls, err := f.GetClass(headerName, classFQN)
	if err != nil {
		return nil, err
	}
	if m, ok := cls.Methods[methodName]; ok {
		return m, nil
	}
	return nil, assert.AnError
}

func (f *fakeCppStdlibLoader) GetFreeFunction(headerName, fqn string) (*core.CStdlibFunction, error) {
	h, err := f.GetHeader(headerName)
	if err != nil {
		return nil, err
	}
	if fn, ok := h.FreeFunctions[fqn]; ok {
		return fn, nil
	}
	return nil, assert.AnError
}

func (f *fakeCppStdlibLoader) Platform() string { return "linux" }
func (f *fakeCppStdlibLoader) HeaderCount() int { return len(f.headers) }

func (f *fakeCppStdlibLoader) ListHeaders() []string {
	out := make([]string, 0, len(f.headers))
	for k := range f.headers {
		out = append(out, k)
	}
	return out
}

// TestBuildCppCallGraph_StdlibClassMethod resolves `vec.push_back(...)`
// against the C++ stdlib registry. The receiver type comes from the
// type engine; the resolver canonicalises std::vector<int> → std::vector
// before looking up the method.
func TestBuildCppCallGraph_StdlibClassMethod(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"

	f := newCppFixture(t)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")
	f.engine.ExtractVariableType("src/main.cpp::main", "vec", "std::vector<int>", resolution.Location{Line: 5})
	f.addCall(t, main, "push_back", "vec")

	f.registry.SystemIncludes["src/main.cpp"] = []string{"vector"}
	f.registry.StdlibCppRegistry = newFakeCppStdlibLoader(map[string]*core.CStdlibHeader{
		"vector": {
			Header: "vector",
			Classes: map[string]*core.CppStdlibClass{
				"std::vector": {
					FQN: "std::vector", TypeParams: []string{"T"},
					Methods: map[string]*core.CStdlibFunction{
						"push_back": {FQN: "std::vector::push_back", ReturnType: "void", Source: core.SourceOverlay, Confidence: 1.0},
					},
				},
			},
		},
	})

	cg := f.build(t)

	sites := cg.CallSites["src/main.cpp::main"]
	require.Len(t, sites, 1)
	assert.True(t, sites[0].Resolved)
	assert.Equal(t, "std::vector::push_back", sites[0].TargetFQN)
	assert.Equal(t, "stdlib", sites[0].TypeSource)
}

// TestBuildCppCallGraph_StdlibClassMethod_TemplateSubstitution verifies
// that a method whose return type is the template parameter (`T`) gets
// the concrete argument substituted in (vector<int>::operator[] → int&).
func TestBuildCppCallGraph_StdlibClassMethod_TemplateSubstitution(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"

	f := newCppFixture(t)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")
	f.engine.ExtractVariableType("src/main.cpp::main", "vec", "std::vector<int>", resolution.Location{Line: 5})
	f.addCall(t, main, "front", "vec")

	f.registry.SystemIncludes["src/main.cpp"] = []string{"vector"}
	f.registry.StdlibCppRegistry = newFakeCppStdlibLoader(map[string]*core.CStdlibHeader{
		"vector": {
			Header: "vector",
			Classes: map[string]*core.CppStdlibClass{
				"std::vector": {
					FQN: "std::vector", TypeParams: []string{"T"},
					Methods: map[string]*core.CStdlibFunction{
						"front": {FQN: "std::vector::front", ReturnType: "T&", Source: core.SourceHeader, Confidence: 1.0},
					},
				},
			},
		},
	})

	cg := f.build(t)

	sites := cg.CallSites["src/main.cpp::main"]
	require.Len(t, sites, 1)
	assert.True(t, sites[0].Resolved)
	assert.Equal(t, "int&", sites[0].InferredType, "T must be replaced with the concrete template argument")
}

// TestBuildCppCallGraph_StdlibFreeFunction_TransitiveFallback covers the
// PR-04 transitive-include fallback. A file that calls `std::move`
// without directly including <utility> still resolves because the
// resolver scans every manifest header when the direct include list
// doesn't yield a hit.
func TestBuildCppCallGraph_StdlibFreeFunction_TransitiveFallback(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"

	f := newCppFixture(t)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")
	f.addCall(t, main, "std::move", "")

	// Notice: NO entry in SystemIncludes for src/main.cpp — the file
	// gets std::move via a transitive include we can't see at the
	// CodeGraph level. Pre-PR-04 this stayed unresolved.
	f.registry.StdlibCppRegistry = newFakeCppStdlibLoader(map[string]*core.CStdlibHeader{
		"utility": {
			Header: "utility",
			FreeFunctions: map[string]*core.CStdlibFunction{
				"std::move": {FQN: "std::move", ReturnType: "T&&", Source: core.SourceOverlay, Confidence: 1.0},
			},
		},
	})

	cg := f.build(t)

	sites := cg.CallSites["src/main.cpp::main"]
	require.Len(t, sites, 1)
	assert.True(t, sites[0].Resolved, "transitive-include fallback must resolve std::move")
	assert.Equal(t, "std::move", sites[0].TargetFQN)
}

// TestBuildCppCallGraph_StdlibFreeFunction handles `std::move(x)` —
// a namespaced free function looked up via GetFreeFunction.
func TestBuildCppCallGraph_StdlibFreeFunction(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"

	f := newCppFixture(t)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")
	// Mimic the parser emitting std::move via the qualified-call path.
	f.addCall(t, main, "std::move", "")

	f.registry.SystemIncludes["src/main.cpp"] = []string{"utility"}
	f.registry.StdlibCppRegistry = newFakeCppStdlibLoader(map[string]*core.CStdlibHeader{
		"utility": {
			Header: "utility",
			FreeFunctions: map[string]*core.CStdlibFunction{
				"std::move": {FQN: "std::move", ReturnType: "T&&", Source: core.SourceOverlay, Confidence: 1.0},
			},
		},
	})

	cg := f.build(t)

	sites := cg.CallSites["src/main.cpp::main"]
	require.Len(t, sites, 1)
	assert.True(t, sites[0].Resolved)
	assert.Equal(t, "std::move", sites[0].TargetFQN)
}

// TestBuildCppCallGraph_StdlibCFallthrough confirms a `printf()` call
// from a .cpp file resolves through the C registry — C++ projects can
// (and routinely do) include <cstdio> / <stdio.h> alongside STL headers.
func TestBuildCppCallGraph_StdlibCFallthrough(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"

	f := newCppFixture(t)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")
	f.addCall(t, main, "printf", "")

	// The caller-file's system includes are stored on the embedded
	// CModuleRegistry, just like for C.
	f.registry.SystemIncludes["src/main.cpp"] = []string{"stdio.h"}
	f.registry.StdlibRegistry = newFakeCStdlibLoader(map[string]map[string]*core.CStdlibFunction{
		"stdio.h": {
			"printf": {FQN: "c::stdio::printf", ReturnType: "int", Source: core.SourceOverlay, Confidence: 1.0},
		},
	})

	cg := f.build(t)

	sites := cg.CallSites["src/main.cpp::main"]
	require.Len(t, sites, 1)
	assert.True(t, sites[0].Resolved)
	assert.Equal(t, "c::stdio::printf", sites[0].TargetFQN)
}

// TestBuildCppCallGraph_StdlibClassMethod_NoReceiver verifies the
// stdlib method path is skipped (gracefully) when the receiver type
// cannot be inferred — falls through to the unresolved branch.
func TestBuildCppCallGraph_StdlibClassMethod_NoReceiver(t *testing.T) {
	root := cppFixtureRoot
	mainCpp := root + "/src/main.cpp"

	f := newCppFixture(t)
	main := f.addFreeFunction(t, mainCpp, "src/main.cpp", "", "main", "int")
	// Caller with no registered variable type for "mystery".
	f.addCall(t, main, "push_back", "mystery")

	f.registry.SystemIncludes["src/main.cpp"] = []string{"vector"}
	f.registry.StdlibCppRegistry = newFakeCppStdlibLoader(map[string]*core.CStdlibHeader{
		"vector": {
			Header: "vector",
			Classes: map[string]*core.CppStdlibClass{
				"std::vector": {
					FQN: "std::vector",
					Methods: map[string]*core.CStdlibFunction{
						"push_back": {FQN: "std::vector::push_back", ReturnType: "void"},
					},
				},
			},
		},
	})

	cg := f.build(t)

	sites := cg.CallSites["src/main.cpp::main"]
	require.Len(t, sites, 1)
	assert.False(t, sites[0].Resolved, "missing receiver type must keep the call unresolved")
}
