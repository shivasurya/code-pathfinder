package resolution

import (
	"sync"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeStore_NewTypeStore(t *testing.T) {
	ts := NewTypeStore()

	assert.Equal(t, 1, ts.CurrentScopeDepth(), "Should start with global scope")
	assert.Equal(t, []string{"global"}, ts.ScopeNames())
}

func TestTypeStore_SetAndGet(t *testing.T) {
	ts := NewTypeStore()
	typ := core.NewConcreteType("myapp.User", 0.9)

	ts.Set("user", typ, core.ConfidenceAssignment, "file.py", 10, 5)

	binding := ts.Get("user")
	require.NotNil(t, binding)
	assert.Equal(t, "user", binding.VarName)
	assert.True(t, typ.Equals(binding.Type))
	assert.Equal(t, core.ConfidenceAssignment, binding.Source)
	assert.Equal(t, "file.py", binding.File)
	assert.Equal(t, 10, binding.Line)
	assert.Equal(t, 5, binding.Column)
}

func TestTypeStore_Lookup(t *testing.T) {
	ts := NewTypeStore()
	typ := core.NewConcreteType("myapp.User", 0.9)

	ts.Set("user", typ, core.ConfidenceAssignment, "file.py", 10, 5)

	result := ts.Lookup("user")
	require.NotNil(t, result)
	assert.True(t, typ.Equals(result))

	result = ts.Lookup("nonexistent")
	assert.Nil(t, result)
}

func TestTypeStore_PushPopScope(t *testing.T) {
	ts := NewTypeStore()

	ts.Set("global_var", core.NewConcreteType("str", 0.9), core.ConfidenceAssignment, "f.py", 1, 0)

	ts.PushScope("function")
	assert.Equal(t, 2, ts.CurrentScopeDepth())
	assert.Equal(t, []string{"global", "function"}, ts.ScopeNames())

	ts.Set("local_var", core.NewConcreteType("int", 0.9), core.ConfidenceAssignment, "f.py", 5, 0)

	// Both visible from inner scope
	assert.NotNil(t, ts.Lookup("global_var"))
	assert.NotNil(t, ts.Lookup("local_var"))

	removed := ts.PopScope()
	assert.Equal(t, 1, ts.CurrentScopeDepth())
	assert.Contains(t, removed, "local_var")

	// Only global visible now
	assert.NotNil(t, ts.Lookup("global_var"))
	assert.Nil(t, ts.Lookup("local_var"))
}

func TestTypeStore_PopScopeGlobalProtection(t *testing.T) {
	ts := NewTypeStore()

	// Try to pop global scope
	removed := ts.PopScope()
	assert.Nil(t, removed, "Cannot pop global scope")
	assert.Equal(t, 1, ts.CurrentScopeDepth())
}

func TestTypeStore_ShadowingInNestedScopes(t *testing.T) {
	ts := NewTypeStore()

	// Global scope: x = str
	ts.Set("x", core.NewConcreteType("str", 0.9), core.ConfidenceAssignment, "f.py", 1, 0)

	ts.PushScope("inner")

	// Inner scope: x = int (shadows global)
	ts.Set("x", core.NewConcreteType("int", 0.9), core.ConfidenceAssignment, "f.py", 5, 0)

	// Should get inner x
	binding := ts.Get("x")
	require.NotNil(t, binding)
	ct, _ := core.ExtractConcreteType(binding.Type)
	assert.Equal(t, "int", ct.Name)

	ts.PopScope()

	// Should get global x again
	binding = ts.Get("x")
	require.NotNil(t, binding)
	ct, _ = core.ExtractConcreteType(binding.Type)
	assert.Equal(t, "str", ct.Name)
}

func TestTypeStore_GetInCurrentScope(t *testing.T) {
	ts := NewTypeStore()
	ts.Set("global_var", core.NewConcreteType("str", 0.9), core.ConfidenceAssignment, "f.py", 1, 0)

	ts.PushScope("inner")
	ts.Set("local_var", core.NewConcreteType("int", 0.9), core.ConfidenceAssignment, "f.py", 5, 0)

	// local_var is in current scope
	assert.NotNil(t, ts.GetInCurrentScope("local_var"))

	// global_var is NOT in current scope (it's in parent)
	assert.Nil(t, ts.GetInCurrentScope("global_var"))
}

func TestTypeStore_Update(t *testing.T) {
	ts := NewTypeStore()

	ts.Set("x", core.NewConcreteType("str", 0.9), core.ConfidenceAssignment, "f.py", 1, 0)

	ts.PushScope("inner")

	// Update x in its original scope (global)
	updated := ts.Update("x", core.NewConcreteType("int", 0.9))
	assert.True(t, updated)

	ts.PopScope()

	// x should be updated in global
	binding := ts.Get("x")
	ct, _ := core.ExtractConcreteType(binding.Type)
	assert.Equal(t, "int", ct.Name)
}

func TestTypeStore_UpdateNonexistent(t *testing.T) {
	ts := NewTypeStore()

	updated := ts.Update("nonexistent", core.NewConcreteType("str", 0.9))
	assert.False(t, updated)
}

func TestTypeStore_AllBindings(t *testing.T) {
	ts := NewTypeStore()

	ts.Set("a", core.NewConcreteType("A", 0.9), core.ConfidenceAssignment, "f.py", 1, 0)
	ts.PushScope("inner")
	ts.Set("b", core.NewConcreteType("B", 0.9), core.ConfidenceAssignment, "f.py", 5, 0)

	bindings := ts.AllBindings()
	assert.Len(t, bindings, 2)

	names := make(map[string]bool)
	for _, b := range bindings {
		names[b.VarName] = true
	}
	assert.True(t, names["a"])
	assert.True(t, names["b"])
}

func TestTypeStore_Clone(t *testing.T) {
	ts := NewTypeStore()
	ts.Set("x", core.NewConcreteType("str", 0.9), core.ConfidenceAssignment, "f.py", 1, 0)
	ts.PushScope("inner")
	ts.Set("y", core.NewConcreteType("int", 0.9), core.ConfidenceAssignment, "f.py", 5, 0)

	clone := ts.Clone()

	// Modify original
	ts.Set("z", core.NewConcreteType("bool", 0.9), core.ConfidenceAssignment, "f.py", 10, 0)

	// Clone should not have z
	assert.NotNil(t, clone.Lookup("x"))
	assert.NotNil(t, clone.Lookup("y"))
	assert.Nil(t, clone.Lookup("z"))

	// Original should have z
	assert.NotNil(t, ts.Lookup("z"))
}

func TestTypeStore_Clear(t *testing.T) {
	ts := NewTypeStore()
	ts.Set("x", core.NewConcreteType("str", 0.9), core.ConfidenceAssignment, "f.py", 1, 0)
	ts.PushScope("inner")
	ts.Set("y", core.NewConcreteType("int", 0.9), core.ConfidenceAssignment, "f.py", 5, 0)

	ts.Clear()

	assert.Equal(t, 1, ts.CurrentScopeDepth())
	assert.Nil(t, ts.Lookup("x"))
	assert.Nil(t, ts.Lookup("y"))
}

func TestTypeStore_ConcurrentAccess(t *testing.T) {
	ts := NewTypeStore()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			varName := "var" + itoa(i)
			ts.Set(varName, core.NewConcreteType("Type"+itoa(i), 0.9), core.ConfidenceAssignment, "f.py", i, 0)
		}(i)
	}

	// Concurrent reads
	for i := range 50 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			varName := "var" + itoa(i)
			ts.Lookup(varName)
		}(i)
	}

	wg.Wait()
	// No race = test passes
}

func TestTypeStore_EmptyStoreOperations(t *testing.T) {
	ts := &TypeStore{} // Uninitialized

	// Should not panic
	ts.Set("x", core.NewConcreteType("str", 0.9), core.ConfidenceAssignment, "f.py", 1, 0)
	assert.Nil(t, ts.Get("x"))
	assert.Nil(t, ts.Lookup("x"))
}
