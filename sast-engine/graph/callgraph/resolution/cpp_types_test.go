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

func TestNewCppTypeInferenceEngine_AllocatesMaps(t *testing.T) {
	cppRegistry := core.NewCppModuleRegistry("/projects/cppapp")
	engine := resolution.NewCppTypeInferenceEngine(cppRegistry)

	require.NotNil(t, engine)
	assert.Same(t, cppRegistry, engine.CppRegistry, "C++ registry must round-trip")
	require.NotNil(t, engine.Registry, "embedded C registry must point to the embedded CModuleRegistry")
	assert.Same(t, &cppRegistry.CModuleRegistry, engine.Registry, "embedded registry must alias the C++ registry's C facet")
	assert.NotNil(t, engine.ClassMethods)
	assert.NotNil(t, engine.ClassFields)

	// nil registry construction does not panic.
	nilEngine := resolution.NewCppTypeInferenceEngine(nil)
	require.NotNil(t, nilEngine)
	assert.Nil(t, nilEngine.CppRegistry)
	assert.Nil(t, nilEngine.Registry)
}

// TestCppTypeInferenceEngine_EmbeddedCMethods exercises the spec's
// "embedded engine" requirement: every C-engine method must be callable
// directly on the C++ engine without re-implementation.
func TestCppTypeInferenceEngine_EmbeddedCMethods(t *testing.T) {
	engine := resolution.NewCppTypeInferenceEngine(nil)
	fqn := "src/main.cpp::main"

	engine.ExtractReturnType(fqn, "int")
	got := engine.GetReturnType(fqn)
	require.NotNil(t, got)
	assert.Equal(t, "int", got.TypeFQN)
	assert.True(t, engine.HasReturnType(fqn))

	engine.ExtractVariableType(fqn, "n", "int", resolution.Location{Line: 3})
	scope := engine.GetScope(fqn)
	require.NotNil(t, scope)
	assert.Equal(t, "int", scope.GetVariable("n").Type.TypeFQN)
}

func TestCppTypeInferenceEngine_AutoStoresWithZeroConfidence(t *testing.T) {
	engine := resolution.NewCppTypeInferenceEngine(nil)
	fqn := "src/main.cpp::main"
	loc := resolution.Location{Line: 7}

	engine.ExtractVariableType(fqn, "x", "auto", loc)

	scope := engine.GetScope(fqn)
	require.NotNil(t, scope)
	binding := scope.GetVariable("x")
	require.NotNil(t, binding)
	assert.Equal(t, "auto", binding.Type.TypeFQN)
	assert.InDelta(t, 0.0, binding.Type.Confidence, 1e-6)
	assert.Equal(t, "unresolved_auto", binding.Type.Source)
	assert.Equal(t, loc, binding.Location)
}

// TestCppTypeInferenceEngine_AutoExactMatch verifies that the auto
// detection uses an exact equality on the type string. Modifiers like
// `auto*` and `auto&` must NOT trigger the unresolved branch — those
// are concrete types in their own right (modifying a deduced type).
// TestCppTypeInferenceEngine_ExtractVariableType_DropsEmptyInputs
// guards the early-return branch that mirrors the C engine's contract.
// Empty FQN, var name, or type string must not produce a binding.
func TestCppTypeInferenceEngine_ExtractVariableType_DropsEmptyInputs(t *testing.T) {
	engine := resolution.NewCppTypeInferenceEngine(nil)
	loc := resolution.Location{}

	engine.ExtractVariableType("", "x", "auto", loc)
	engine.ExtractVariableType("src/m.cpp::m", "", "auto", loc)
	engine.ExtractVariableType("src/m.cpp::m", "x", "", loc)

	assert.Nil(t, engine.GetScope("src/m.cpp::m"))
}

func TestCppTypeInferenceEngine_AutoExactMatch(t *testing.T) {
	engine := resolution.NewCppTypeInferenceEngine(nil)
	fqn := "src/main.cpp::main"

	engine.ExtractVariableType(fqn, "p", "auto*", resolution.Location{})
	engine.ExtractVariableType(fqn, "r", "auto&", resolution.Location{})

	pBinding := engine.GetScope(fqn).GetVariable("p")
	require.NotNil(t, pBinding)
	assert.Equal(t, "auto*", pBinding.Type.TypeFQN)
	assert.InDelta(t, 1.0, pBinding.Type.Confidence, 1e-6, "auto* is concrete, must keep full confidence")
	assert.Equal(t, "declaration", pBinding.Type.Source)

	rBinding := engine.GetScope(fqn).GetVariable("r")
	require.NotNil(t, rBinding)
	assert.InDelta(t, 1.0, rBinding.Type.Confidence, 1e-6)
}

func TestCppTypeInferenceEngine_RegisterClassMethod(t *testing.T) {
	engine := resolution.NewCppTypeInferenceEngine(nil)

	engine.RegisterClassMethod("Socket", "connect", "bool")

	info := engine.GetMethodReturnType("Socket", "connect")
	require.NotNil(t, info)
	assert.Equal(t, "bool", info.TypeFQN)
	assert.InDelta(t, 1.0, info.Confidence, 1e-6)
	assert.Equal(t, "declaration", info.Source)
	assert.True(t, engine.HasClassMethod("Socket", "connect"))
}

func TestCppTypeInferenceEngine_RegisterClassMethod_DropsVoidAndEmpty(t *testing.T) {
	engine := resolution.NewCppTypeInferenceEngine(nil)

	engine.RegisterClassMethod("Socket", "init", "void")
	engine.RegisterClassMethod("", "connect", "bool")
	engine.RegisterClassMethod("Socket", "", "bool")
	engine.RegisterClassMethod("Socket", "connect", "")

	assert.Nil(t, engine.GetMethodReturnType("Socket", "init"), "void return must not be stored")
	assert.Nil(t, engine.GetMethodReturnType("Socket", "connect"), "incomplete inputs must not register anything")
	assert.False(t, engine.HasClassMethod("Socket", "init"))
	assert.False(t, engine.HasClassMethod("Other", "x"))
}

func TestCppTypeInferenceEngine_RegisterClassMethod_RedeclarationOverwrites(t *testing.T) {
	engine := resolution.NewCppTypeInferenceEngine(nil)

	engine.RegisterClassMethod("Socket", "connect", "bool")
	engine.RegisterClassMethod("Socket", "connect", "Status")

	info := engine.GetMethodReturnType("Socket", "connect")
	require.NotNil(t, info)
	assert.Equal(t, "Status", info.TypeFQN, "the most recent registration must win")
}

func TestCppTypeInferenceEngine_RegisterClassField(t *testing.T) {
	engine := resolution.NewCppTypeInferenceEngine(nil)

	engine.RegisterClassField("Socket", "port", "int")
	engine.RegisterClassField("Socket", "name", "std::string")

	port := engine.GetFieldType("Socket", "port")
	require.NotNil(t, port)
	assert.Equal(t, "int", port.TypeFQN)
	assert.InDelta(t, 1.0, port.Confidence, 1e-6)
	assert.Equal(t, "declaration", port.Source)
	assert.True(t, engine.HasClassField("Socket", "port"))
	assert.True(t, engine.HasClassField("Socket", "name"))
	assert.False(t, engine.HasClassField("Socket", "missing"))
	assert.False(t, engine.HasClassField("Other", "port"))
}

func TestCppTypeInferenceEngine_RegisterClassField_DropsEmpty(t *testing.T) {
	engine := resolution.NewCppTypeInferenceEngine(nil)

	engine.RegisterClassField("", "port", "int")
	engine.RegisterClassField("Socket", "", "int")
	engine.RegisterClassField("Socket", "port", "")

	assert.Nil(t, engine.GetFieldType("Socket", "port"))
}

// TestCppTypeInferenceEngine_ComplexCppTypes verifies template
// instantiations, references, and multi-argument templates round-trip
// verbatim through every registration path.
func TestCppTypeInferenceEngine_ComplexCppTypes(t *testing.T) {
	engine := resolution.NewCppTypeInferenceEngine(nil)
	fqn := "src/main.cpp::process"

	cases := []struct {
		name string
		typ  string
	}{
		{"vector_int", "std::vector<int>"},
		{"const_string_ref", "const std::string&"},
		{"map_string_int", "std::map<std::string, int>"},
		{"unique_ptr", "std::unique_ptr<Widget>"},
		{"nested_template", "std::vector<std::pair<int, std::string>>"},
	}
	for _, tc := range cases {
		engine.ExtractVariableType(fqn, tc.name, tc.typ, resolution.Location{})
		engine.RegisterClassMethod("Manager", "make_"+tc.name, tc.typ)
		engine.RegisterClassField("Manager", "field_"+tc.name, tc.typ)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			binding := engine.GetScope(fqn).GetVariable(tc.name)
			require.NotNil(t, binding)
			assert.Equal(t, tc.typ, binding.Type.TypeFQN)

			method := engine.GetMethodReturnType("Manager", "make_"+tc.name)
			require.NotNil(t, method)
			assert.Equal(t, tc.typ, method.TypeFQN)

			field := engine.GetFieldType("Manager", "field_"+tc.name)
			require.NotNil(t, field)
			assert.Equal(t, tc.typ, field.TypeFQN)
		})
	}
}

// TestCppTypeInferenceEngine_ConcurrentAccess covers the C++-only
// classMethodMutex / classFieldMutex pair. The C-level mutexes are
// already exercised by TestCTypeInferenceEngine_ConcurrentAccess, so
// this test focuses on the new locks introduced by the C++ engine.
func TestCppTypeInferenceEngine_ConcurrentAccess(t *testing.T) {
	engine := resolution.NewCppTypeInferenceEngine(nil)
	const goroutines = 16
	const opsPerGoroutine = 200

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for g := range goroutines {
		go func(seed int) {
			defer wg.Done()
			for i := range opsPerGoroutine {
				className := "Class" + strconv.Itoa(seed)
				name := "m" + strconv.Itoa(i)
				engine.RegisterClassMethod(className, name, "int")
				engine.RegisterClassField(className, "f"+name, "int")
			}
		}(g)
	}
	for range goroutines {
		go func() {
			defer wg.Done()
			for range opsPerGoroutine {
				_ = engine.GetMethodReturnType("Class0", "m0")
				_ = engine.GetFieldType("Class0", "fm0")
				_ = engine.HasClassMethod("Class0", "m0")
				_ = engine.HasClassField("Class0", "fm0")
			}
		}()
	}
	wg.Wait()

	for g := range goroutines {
		className := "Class" + strconv.Itoa(g)
		assert.True(t, engine.HasClassMethod(className, "m0"))
		assert.True(t, engine.HasClassField(className, "fm0"))
	}
}
