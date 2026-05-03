package builder_test

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeCStdlibLoader implements core.CStdlibLoader for builder tests
// without going through the registry package — keeps the test focused
// on the builder's resolution path rather than loader plumbing.
type fakeCStdlibLoader struct {
	headers map[string]map[string]*core.CStdlibFunction
}

func newFakeCStdlibLoader(headers map[string]map[string]*core.CStdlibFunction) *fakeCStdlibLoader {
	return &fakeCStdlibLoader{headers: headers}
}

func (f *fakeCStdlibLoader) LoadManifest(_ core.CStdlibLogger) error { return nil }

func (f *fakeCStdlibLoader) GetHeader(name string) (*core.CStdlibHeader, error) {
	fns, ok := f.headers[name]
	if !ok {
		return nil, assert.AnError
	}
	return &core.CStdlibHeader{Header: name, Functions: fns}, nil
}

func (f *fakeCStdlibLoader) GetFunction(headerName, funcName string) (*core.CStdlibFunction, error) {
	if fns, ok := f.headers[headerName]; ok {
		if fn, ok := fns[funcName]; ok {
			return fn, nil
		}
	}
	return nil, assert.AnError
}

func (f *fakeCStdlibLoader) Platform() string { return "linux" }

func (f *fakeCStdlibLoader) HeaderCount() int { return len(f.headers) }

// TestBuildCCallGraph_StdlibFallback verifies that an unresolved call
// falls through to the stdlib registry and emits an enriched CallSite
// (TargetFQN, return type, confidence, security tag).
func TestBuildCCallGraph_StdlibFallback(t *testing.T) {
	root := fixtureRoot
	mainC := root + "/src/main.c"

	f := newCFixture(t)
	mainFn := f.addFunction(t, mainC, "src/main.c", "main", "int", false)
	f.addCall(t, mainFn, "system", []string{`"ls"`})

	// Wire the system include and stdlib loader for the caller file.
	f.registry.SystemIncludes["src/main.c"] = []string{"stdlib.h"}
	f.registry.StdlibRegistry = newFakeCStdlibLoader(map[string]map[string]*core.CStdlibFunction{
		"stdlib.h": {
			"system": {
				FQN:         "c::stdlib::system",
				ReturnType:  "int",
				SecurityTag: "command_injection_sink",
				Source:      core.SourceOverlay,
				Confidence:  0.95,
			},
		},
	})

	cg, _ := f.build(t)

	sites := cg.CallSites["src/main.c::main"]
	require.Len(t, sites, 1)
	assert.True(t, sites[0].Resolved)
	assert.Equal(t, "c::stdlib::system", sites[0].TargetFQN)
	assert.Equal(t, "int", sites[0].InferredType)
	assert.Equal(t, "stdlib", sites[0].TypeSource)
	assert.Equal(t, "command_injection_sink", sites[0].SecurityTag)
	assert.InDelta(t, 0.95, sites[0].TypeConfidence, 1e-6)
}

// TestBuildCCallGraph_StdlibLookupRespectsIncludeOrder confirms the
// resolver walks SystemIncludes in order and stops at the first match —
// later headers must not override earlier ones.
func TestBuildCCallGraph_StdlibLookupRespectsIncludeOrder(t *testing.T) {
	root := fixtureRoot
	mainC := root + "/src/main.c"

	f := newCFixture(t)
	mainFn := f.addFunction(t, mainC, "src/main.c", "main", "int", false)
	f.addCall(t, mainFn, "abort", nil)

	f.registry.SystemIncludes["src/main.c"] = []string{"stdlib.h", "fake.h"}
	f.registry.StdlibRegistry = newFakeCStdlibLoader(map[string]map[string]*core.CStdlibFunction{
		"stdlib.h": {
			"abort": {FQN: "c::stdlib::abort", ReturnType: "void", Source: core.SourceHeader, Confidence: 1.0},
		},
		"fake.h": {
			"abort": {FQN: "fake::abort", ReturnType: "void", Source: core.SourceHeader, Confidence: 1.0},
		},
	})

	cg, _ := f.build(t)

	sites := cg.CallSites["src/main.c::main"]
	require.Len(t, sites, 1)
	assert.Equal(t, "c::stdlib::abort", sites[0].TargetFQN, "first include in order must win")
}

// TestBuildCCallGraph_StdlibFallback_NotConsultedWhenProjectDefinitionExists
// guards against the stdlib path overriding a same-file definition.
// printf is normally a stdlib symbol, but a project that defines its
// own `printf` must keep the project FQN.
func TestBuildCCallGraph_StdlibFallback_NotConsultedWhenProjectDefinitionExists(t *testing.T) {
	root := fixtureRoot
	mainC := root + "/src/main.c"

	f := newCFixture(t)
	mainFn := f.addFunction(t, mainC, "src/main.c", "main", "int", false)
	f.addFunction(t, mainC, "src/main.c", "printf", "int", false)
	f.addCall(t, mainFn, "printf", nil)

	f.registry.SystemIncludes["src/main.c"] = []string{"stdio.h"}
	f.registry.StdlibRegistry = newFakeCStdlibLoader(map[string]map[string]*core.CStdlibFunction{
		"stdio.h": {
			"printf": {FQN: "c::stdio::printf", ReturnType: "int", Source: core.SourceOverlay, Confidence: 1.0},
		},
	})

	cg, _ := f.build(t)

	sites := cg.CallSites["src/main.c::main"]
	require.Len(t, sites, 1)
	assert.Equal(t, "src/main.c::printf", sites[0].TargetFQN, "project definition must shadow stdlib symbol")
	assert.Empty(t, sites[0].SecurityTag, "project resolution must not pick up stdlib SecurityTag")
}

// TestBuildCCallGraph_StdlibFallback_NoIncludesLeavesUnresolved verifies
// that a call to an unknown function with no matching system include
// stays unresolved — the registry must not return arbitrary symbols.
func TestBuildCCallGraph_StdlibFallback_NoIncludesLeavesUnresolved(t *testing.T) {
	root := fixtureRoot
	mainC := root + "/src/main.c"

	f := newCFixture(t)
	mainFn := f.addFunction(t, mainC, "src/main.c", "main", "int", false)
	f.addCall(t, mainFn, "printf", nil)

	// No SystemIncludes entry for this file → stdlib lookup is a no-op.
	f.registry.StdlibRegistry = newFakeCStdlibLoader(map[string]map[string]*core.CStdlibFunction{
		"stdio.h": {"printf": {FQN: "c::stdio::printf", ReturnType: "int"}},
	})

	cg, _ := f.build(t)

	sites := cg.CallSites["src/main.c::main"]
	require.Len(t, sites, 1)
	assert.False(t, sites[0].Resolved, "no matching include => stdlib must not be consulted")
}
