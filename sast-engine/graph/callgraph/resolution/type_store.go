// Package resolution provides scope-based type storage for inference.
package resolution

import (
	"sync"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// TypeBinding represents a variable-to-type binding with metadata.
type TypeBinding struct {
	VarName    string
	Type       core.Type
	Source     core.ConfidenceSource
	File       string
	Line       int
	Column     int
	ScopeDepth int
}

// TypeStore provides hierarchical scope-based type storage.
// Supports push/pop semantics for nested scopes (functions, loops, etc.).
type TypeStore struct {
	scopes     []*scopeFrame
	scopeDepth int
	mutex      sync.RWMutex
}

// scopeFrame represents a single scope level.
type scopeFrame struct {
	bindings map[string]*TypeBinding
	name     string // Optional scope name for debugging
}

// NewTypeStore creates a new TypeStore with a global scope.
func NewTypeStore() *TypeStore {
	ts := &TypeStore{
		scopes:     make([]*scopeFrame, 0),
		scopeDepth: 0,
	}
	ts.PushScope("global")
	return ts
}

// PushScope creates a new scope level.
func (ts *TypeStore) PushScope(name string) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	ts.scopes = append(ts.scopes, &scopeFrame{
		bindings: make(map[string]*TypeBinding),
		name:     name,
	})
	ts.scopeDepth++
}

// PopScope removes the current scope level.
// Returns the removed bindings for debugging.
func (ts *TypeStore) PopScope() map[string]*TypeBinding {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	if len(ts.scopes) <= 1 {
		// Never pop the global scope
		return nil
	}

	last := ts.scopes[len(ts.scopes)-1]
	ts.scopes = ts.scopes[:len(ts.scopes)-1]
	ts.scopeDepth--

	return last.bindings
}

// Set binds a variable to a type in the current scope.
func (ts *TypeStore) Set(varName string, typ core.Type, source core.ConfidenceSource, file string, line, col int) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	if len(ts.scopes) == 0 {
		return
	}

	current := ts.scopes[len(ts.scopes)-1]
	current.bindings[varName] = &TypeBinding{
		VarName:    varName,
		Type:       typ,
		Source:     source,
		File:       file,
		Line:       line,
		Column:     col,
		ScopeDepth: ts.scopeDepth,
	}
}

// Get retrieves the type for a variable, searching from innermost to outermost scope.
func (ts *TypeStore) Get(varName string) *TypeBinding {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()

	// Search from innermost to outermost
	for i := len(ts.scopes) - 1; i >= 0; i-- {
		if binding, found := ts.scopes[i].bindings[varName]; found {
			return binding
		}
	}
	return nil
}

// Lookup is an alias for Get that returns just the type.
func (ts *TypeStore) Lookup(varName string) core.Type {
	binding := ts.Get(varName)
	if binding == nil {
		return nil
	}
	return binding.Type
}

// GetInCurrentScope retrieves a binding only from the current scope.
func (ts *TypeStore) GetInCurrentScope(varName string) *TypeBinding {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()

	if len(ts.scopes) == 0 {
		return nil
	}

	current := ts.scopes[len(ts.scopes)-1]
	return current.bindings[varName]
}

// Update updates an existing binding in its original scope.
// Returns false if the variable doesn't exist.
func (ts *TypeStore) Update(varName string, typ core.Type) bool {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	// Find and update in the scope where it was defined
	for i := len(ts.scopes) - 1; i >= 0; i-- {
		if binding, found := ts.scopes[i].bindings[varName]; found {
			binding.Type = typ
			return true
		}
	}
	return false
}

// CurrentScopeDepth returns the current scope nesting level.
func (ts *TypeStore) CurrentScopeDepth() int {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()
	return ts.scopeDepth
}

// AllBindings returns all bindings across all scopes.
func (ts *TypeStore) AllBindings() []*TypeBinding {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()

	totalBindings := 0
	for _, scope := range ts.scopes {
		totalBindings += len(scope.bindings)
	}
	bindings := make([]*TypeBinding, 0, totalBindings)
	for _, scope := range ts.scopes {
		for _, binding := range scope.bindings {
			bindings = append(bindings, binding)
		}
	}
	return bindings
}

// Clone creates a deep copy of the TypeStore.
// Useful for speculative inference branches.
func (ts *TypeStore) Clone() *TypeStore {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()

	clone := &TypeStore{
		scopes:     make([]*scopeFrame, len(ts.scopes)),
		scopeDepth: ts.scopeDepth,
	}

	for i, scope := range ts.scopes {
		clone.scopes[i] = &scopeFrame{
			bindings: make(map[string]*TypeBinding),
			name:     scope.name,
		}
		for k, v := range scope.bindings {
			// Shallow copy of binding (Type is immutable)
			bindingCopy := *v
			clone.scopes[i].bindings[k] = &bindingCopy
		}
	}

	return clone
}

// Clear removes all bindings except the global scope.
func (ts *TypeStore) Clear() {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	// Keep only global scope
	if len(ts.scopes) > 0 {
		ts.scopes = ts.scopes[:1]
		ts.scopes[0].bindings = make(map[string]*TypeBinding)
		ts.scopeDepth = 1
	}
}

// ScopeNames returns the names of all active scopes (for debugging).
func (ts *TypeStore) ScopeNames() []string {
	ts.mutex.RLock()
	defer ts.mutex.RUnlock()

	names := make([]string, len(ts.scopes))
	for i, scope := range ts.scopes {
		names[i] = scope.name
	}
	return names
}
