package resolution_test

import (
	"strconv"
	"sync"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCTypeInferenceEngine_AllocatesMaps(t *testing.T) {
	registry := core.NewCModuleRegistry("/projects/myapp")
	engine := resolution.NewCTypeInferenceEngine(registry)

	require.NotNil(t, engine)
	assert.Same(t, registry, engine.Registry, "registry pointer must round-trip")
	assert.NotNil(t, engine.Scopes)
	assert.NotNil(t, engine.ReturnTypes)

	// nil-registry construction is permitted (used by tests + future
	// callers that want type-only extraction without FQN context).
	nilEngine := resolution.NewCTypeInferenceEngine(nil)
	require.NotNil(t, nilEngine)
	assert.Nil(t, nilEngine.Registry)
}

func TestCTypeInferenceEngine_ExtractReturnType(t *testing.T) {
	engine := resolution.NewCTypeInferenceEngine(nil)
	fqn := "src/main.c::compute"

	engine.ExtractReturnType(fqn, "int")

	got := engine.GetReturnType(fqn)
	require.NotNil(t, got)
	assert.Equal(t, "int", got.TypeFQN)
	assert.InDelta(t, 1.0, got.Confidence, 1e-6)
	assert.Equal(t, "declaration", got.Source)
	assert.True(t, engine.HasReturnType(fqn))
}

func TestCTypeInferenceEngine_ExtractReturnType_VoidIsDropped(t *testing.T) {
	engine := resolution.NewCTypeInferenceEngine(nil)
	fqn := "src/main.c::do_nothing"

	engine.ExtractReturnType(fqn, "void")

	assert.Nil(t, engine.GetReturnType(fqn), "void must not be stored")
	assert.False(t, engine.HasReturnType(fqn))

	// Empty arguments are also no-ops.
	engine.ExtractReturnType("", "int")
	engine.ExtractReturnType(fqn, "")
	assert.False(t, engine.HasReturnType(fqn))
}

func TestCTypeInferenceEngine_AddReturnType_PreservesProvidedTypeInfo(t *testing.T) {
	engine := resolution.NewCTypeInferenceEngine(nil)
	fqn := "src/main.c::adopt"

	custom := &core.TypeInfo{TypeFQN: "Buffer*", Confidence: 0.7, Source: "declaration"}
	engine.AddReturnType(fqn, custom)

	got := engine.GetReturnType(fqn)
	require.NotNil(t, got)
	assert.Equal(t, "Buffer*", got.TypeFQN)
	assert.InDelta(t, 0.7, got.Confidence, 1e-6)
	assert.Equal(t, "declaration", got.Source)

	// nil typeInfo and empty fqn are silent no-ops.
	engine.AddReturnType(fqn, nil)
	engine.AddReturnType("", custom)
	assert.Same(t, custom, engine.GetReturnType(fqn), "existing entry must not be overwritten by nil/empty calls")
}

func TestCTypeInferenceEngine_GetAllReturnTypes_ReturnsCopy(t *testing.T) {
	engine := resolution.NewCTypeInferenceEngine(nil)
	engine.ExtractReturnType("src/a.c::a", "int")
	engine.ExtractReturnType("src/b.c::b", "char*")

	all := engine.GetAllReturnTypes()
	require.Len(t, all, 2)

	// Mutating the snapshot does not affect engine state.
	delete(all, "src/a.c::a")
	assert.Len(t, engine.GetAllReturnTypes(), 2, "snapshot must be a copy")
}

func TestCTypeInferenceEngine_ExtractVariableType(t *testing.T) {
	engine := resolution.NewCTypeInferenceEngine(nil)
	fqn := "src/main.c::main"

	loc := resolution.Location{File: "/projects/myapp/src/main.c", Line: 10, Column: 5}
	engine.ExtractVariableType(fqn, "buf", "char*", loc)

	scope := engine.GetScope(fqn)
	require.NotNil(t, scope)
	assert.Equal(t, fqn, scope.FunctionFQN)
	assert.True(t, scope.HasVariable("buf"))

	binding := scope.GetVariable("buf")
	require.NotNil(t, binding)
	assert.Equal(t, "buf", binding.VarName)
	require.NotNil(t, binding.Type)
	assert.Equal(t, "char*", binding.Type.TypeFQN)
	assert.InDelta(t, 1.0, binding.Type.Confidence, 1e-6)
	assert.Equal(t, "declaration", binding.Type.Source)
	assert.Equal(t, loc, binding.Location)
}

func TestCTypeInferenceEngine_ExtractVariableType_LatestWins(t *testing.T) {
	engine := resolution.NewCTypeInferenceEngine(nil)
	fqn := "src/main.c::main"

	engine.ExtractVariableType(fqn, "n", "int", resolution.Location{Line: 1})
	engine.ExtractVariableType(fqn, "n", "long", resolution.Location{Line: 5})

	binding := engine.GetScope(fqn).GetVariable("n")
	require.NotNil(t, binding)
	assert.Equal(t, "long", binding.Type.TypeFQN, "GetVariable must return the most recent binding")

	all := engine.GetScope(fqn).GetAllBindings("n")
	require.Len(t, all, 2)
	assert.Equal(t, "int", all[0].Type.TypeFQN)
	assert.Equal(t, "long", all[1].Type.TypeFQN)
}

func TestCTypeInferenceEngine_ExtractVariableType_DropsEmptyInputs(t *testing.T) {
	engine := resolution.NewCTypeInferenceEngine(nil)
	loc := resolution.Location{}

	engine.ExtractVariableType("", "x", "int", loc)
	engine.ExtractVariableType("src/m.c::m", "", "int", loc)
	engine.ExtractVariableType("src/m.c::m", "x", "", loc)

	assert.Nil(t, engine.GetScope("src/m.c::m"), "no scope should be created from empty inputs")
}

func TestCTypeInferenceEngine_GetScope_Miss(t *testing.T) {
	engine := resolution.NewCTypeInferenceEngine(nil)
	assert.Nil(t, engine.GetScope("nonexistent"))
	assert.False(t, engine.HasScope("nonexistent"))
}

func TestCTypeInferenceEngine_AddScope_StandaloneInsertion(t *testing.T) {
	engine := resolution.NewCTypeInferenceEngine(nil)

	scope := resolution.NewCFunctionScope("src/x.c::f")
	scope.AddVariable(&resolution.CVariableBinding{
		VarName: "p",
		Type:    &core.TypeInfo{TypeFQN: "int*", Confidence: 1.0, Source: "declaration"},
	})
	engine.AddScope(scope)

	got := engine.GetScope("src/x.c::f")
	require.NotNil(t, got)
	assert.Equal(t, "int*", got.GetVariable("p").Type.TypeFQN)

	// nil scopes and empty FQNs are dropped.
	engine.AddScope(nil)
	engine.AddScope(resolution.NewCFunctionScope(""))
	assert.Len(t, engine.GetAllScopes(), 1)
}

func TestCFunctionScope_DefensiveAdd(t *testing.T) {
	scope := resolution.NewCFunctionScope("src/x.c::f")

	// nil binding is silently dropped.
	scope.AddVariable(nil)
	// Empty VarName is silently dropped.
	scope.AddVariable(&resolution.CVariableBinding{VarName: ""})

	assert.Empty(t, scope.Variables)
	assert.Nil(t, scope.GetVariable("anything"))
	assert.Empty(t, scope.GetAllBindings("anything"))
}

// TestCTypeInferenceEngine_ConcurrentAccess intentionally mixes reads
// and writes from multiple goroutines so `go test -race` exercises the
// internal RWMutex pair. Failures here indicate a missing lock, a
// double-Unlock, or a map mutation outside the critical section.
func TestCTypeInferenceEngine_ConcurrentAccess(t *testing.T) {
	engine := resolution.NewCTypeInferenceEngine(nil)
	const goroutines = 16
	const opsPerGoroutine = 200

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// Writers: each goroutine writes a unique slice of FQNs.
	for g := range goroutines {
		go func(seed int) {
			defer wg.Done()
			for i := range opsPerGoroutine {
				fqn := "src/c" + strconv.Itoa(seed) + ".c::fn" + strconv.Itoa(i)
				engine.ExtractReturnType(fqn, "int")
				engine.ExtractVariableType(fqn, "x", "int", resolution.Location{Line: uint32(i)})
			}
		}(g)
	}
	// Readers: hammer the snapshot accessors and lookup methods.
	for range goroutines {
		go func() {
			defer wg.Done()
			for range opsPerGoroutine {
				_ = engine.GetAllReturnTypes()
				_ = engine.GetAllScopes()
				_ = engine.GetReturnType("src/c0.c::fn0")
				if scope := engine.GetScope("src/c0.c::fn0"); scope != nil {
					_ = scope.GetVariable("x")
				}
			}
		}()
	}
	wg.Wait()

	assert.Len(t, engine.GetAllReturnTypes(), goroutines*opsPerGoroutine)
	assert.Len(t, engine.GetAllScopes(), goroutines*opsPerGoroutine)
}

// TestCTypeInferenceEngine_ComplexCTypes verifies the engine preserves
// the exact type string the parser produced — pointer modifiers, const
// qualifiers, multi-word types, and tag-prefixed struct references must
// all round-trip without normalisation.
func TestCTypeInferenceEngine_ComplexCTypes(t *testing.T) {
	engine := resolution.NewCTypeInferenceEngine(nil)
	fqn := "src/main.c::work"
	loc := resolution.Location{}

	cases := []struct {
		name string
		typ  string
	}{
		{"const_pointer", "const char*"},
		{"unsigned_long_long", "unsigned long long"},
		{"struct_pointer", "struct Buffer*"},
		{"void_pointer", "void*"},
		{"size_t", "size_t"},
		{"function_pointer", "int (*)(int, int)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			engine.ExtractVariableType(fqn, tc.name, tc.typ, loc)
			binding := engine.GetScope(fqn).GetVariable(tc.name)
			require.NotNil(t, binding)
			assert.Equal(t, tc.typ, binding.Type.TypeFQN, "complex type must round-trip verbatim")
		})
	}
}
