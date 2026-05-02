package resolution

import (
	"maps"
	"sync"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// declarationSource is the TypeInfo.Source value used for explicit types
// read directly from C/C++ declarations. It distinguishes types that the
// engine knows for certain (Confidence 1.0) from inferred or deduced
// types added in later phases.
const declarationSource = "declaration"

// CVariableBinding captures the explicit type of a single variable
// declaration inside a C function. Multiple bindings may exist for the
// same name when the variable is reassigned; the latest binding wins
// during lookup.
//
// Example:
//
//	int n = 0;            // CVariableBinding{VarName:"n", Type: int}
//	const char *msg = ""; // CVariableBinding{VarName:"msg", Type: const char*}
//
// Location reuses the package-level resolution.Location so call-site
// reporting and type tracking share one source-location vocabulary.
type CVariableBinding struct {
	// VarName is the bare identifier of the declared variable.
	VarName string

	// Type is the explicit type drawn from the source declaration.
	// For C/C++, the engine sets Confidence=1.0 and Source="declaration"
	// on every entry produced from an explicit type; the only exception
	// is C++ `auto` (see CppTypeInferenceEngine.ExtractVariableType).
	Type *core.TypeInfo

	// Location is the source location of the declaration.
	Location Location
}

// CFunctionScope tracks every variable declared inside one C function.
// Bindings are stored as a slice per name so later phases can audit
// reassignment history; GetVariable always returns the most recent one.
type CFunctionScope struct {
	// FunctionFQN is the fully-qualified name of the owning function
	// (e.g. "src/net/socket.c::handle_request").
	FunctionFQN string

	// Variables maps a bare variable name to every binding observed
	// for it within this function. The latest binding is the last
	// element of each slice.
	Variables map[string][]*CVariableBinding
}

// NewCFunctionScope returns an empty scope keyed to the given function
// FQN with its Variables map pre-allocated.
func NewCFunctionScope(functionFQN string) *CFunctionScope {
	return &CFunctionScope{
		FunctionFQN: functionFQN,
		Variables:   make(map[string][]*CVariableBinding),
	}
}

// AddVariable appends binding to the per-name binding history. nil
// bindings are silently dropped so callers can write
// `scope.AddVariable(makeBinding(...))` without nil checks.
func (s *CFunctionScope) AddVariable(binding *CVariableBinding) {
	if binding == nil || binding.VarName == "" {
		return
	}
	s.Variables[binding.VarName] = append(s.Variables[binding.VarName], binding)
}

// GetVariable returns the latest binding for varName, or nil when the
// variable is unknown to this scope.
func (s *CFunctionScope) GetVariable(varName string) *CVariableBinding {
	bindings := s.Variables[varName]
	if len(bindings) == 0 {
		return nil
	}
	return bindings[len(bindings)-1]
}

// HasVariable reports whether at least one binding exists for varName.
func (s *CFunctionScope) HasVariable(varName string) bool {
	return len(s.Variables[varName]) > 0
}

// GetAllBindings returns every binding recorded for varName, in
// insertion order. Callers must not mutate the slice — return value
// is the live storage for performance.
func (s *CFunctionScope) GetAllBindings(varName string) []*CVariableBinding {
	return s.Variables[varName]
}

// CTypeInferenceEngine indexes explicit type information for a parsed
// C codebase: function return types and per-function variable scopes.
//
// The engine performs no inference, no propagation, and no flow
// analysis — every entry mirrors a type that appears verbatim in the
// source. Higher-confidence handlers (PR-07's call-graph builder) layer
// further analysis on top.
//
// Lifecycle:
//
//   - Construct once with NewCTypeInferenceEngine(registry).
//   - Populate from multiple goroutines during parallel Pass 2
//     extraction (`go test -race` clean).
//   - Read-only consumption during call-graph construction.
//
// Embedding: CppTypeInferenceEngine embeds this type by value to inherit
// every method, so consumers can call ExtractReturnType, GetScope, etc.
// uniformly across both languages.
type CTypeInferenceEngine struct {
	// Scopes maps function FQN to the variables declared inside it.
	Scopes map[string]*CFunctionScope

	// ReturnTypes maps function FQN to its declared return type. void
	// returns are intentionally absent — see ExtractReturnType.
	ReturnTypes map[string]*core.TypeInfo

	// Registry exposes the C module registry for FQN resolution. The
	// engine itself never mutates the registry.
	Registry *core.CModuleRegistry

	scopeMutex sync.RWMutex
	typeMutex  sync.RWMutex
}

// NewCTypeInferenceEngine returns an engine with allocated maps wired
// to the supplied registry. Passing a nil registry is permitted —
// the engine will simply produce no FQN-aware lookups, but type
// extraction still works (useful for unit tests).
func NewCTypeInferenceEngine(registry *core.CModuleRegistry) *CTypeInferenceEngine {
	return &CTypeInferenceEngine{
		Scopes:      make(map[string]*CFunctionScope),
		ReturnTypes: make(map[string]*core.TypeInfo),
		Registry:    registry,
	}
}

// =============================================================================
// Return type management
// =============================================================================

// ExtractReturnType records the explicit return type for the function
// identified by fqn. Empty types and the literal "void" are dropped: a
// void return carries no information for type-driven resolution and
// would only pollute downstream lookups.
//
// Safe for concurrent use.
func (e *CTypeInferenceEngine) ExtractReturnType(fqn, returnType string) {
	if fqn == "" || returnType == "" || returnType == "void" {
		return
	}
	info := &core.TypeInfo{
		TypeFQN:    returnType,
		Confidence: 1.0,
		Source:     declarationSource,
	}
	e.typeMutex.Lock()
	e.ReturnTypes[fqn] = info
	e.typeMutex.Unlock()
}

// AddReturnType stores a precomputed TypeInfo for fqn. Useful when the
// caller has already classified a return type (e.g. through a future
// stdlib registry). Nil typeInfo is ignored.
func (e *CTypeInferenceEngine) AddReturnType(fqn string, typeInfo *core.TypeInfo) {
	if fqn == "" || typeInfo == nil {
		return
	}
	e.typeMutex.Lock()
	e.ReturnTypes[fqn] = typeInfo
	e.typeMutex.Unlock()
}

// GetReturnType returns the recorded return type for fqn, or nil when
// none was registered (which includes void functions).
func (e *CTypeInferenceEngine) GetReturnType(fqn string) *core.TypeInfo {
	e.typeMutex.RLock()
	defer e.typeMutex.RUnlock()
	return e.ReturnTypes[fqn]
}

// HasReturnType reports whether a return type has been recorded for fqn.
func (e *CTypeInferenceEngine) HasReturnType(fqn string) bool {
	e.typeMutex.RLock()
	defer e.typeMutex.RUnlock()
	_, ok := e.ReturnTypes[fqn]
	return ok
}

// GetAllReturnTypes returns a snapshot copy of every registered return
// type. The copy keeps the caller insulated from concurrent writes.
func (e *CTypeInferenceEngine) GetAllReturnTypes() map[string]*core.TypeInfo {
	e.typeMutex.RLock()
	defer e.typeMutex.RUnlock()
	out := make(map[string]*core.TypeInfo, len(e.ReturnTypes))
	maps.Copy(out, e.ReturnTypes)
	return out
}

// =============================================================================
// Scope and variable management
// =============================================================================

// ExtractVariableType registers an explicit variable declaration inside
// functionFQN. Empty arguments are silently dropped so callers do not
// need to pre-validate parser output.
//
// Safe for concurrent use. The function lazily creates the scope on
// first sight of functionFQN, so callers do not have to call AddScope
// before the first variable.
func (e *CTypeInferenceEngine) ExtractVariableType(functionFQN, varName, typeStr string, loc Location) {
	if functionFQN == "" || varName == "" || typeStr == "" {
		return
	}
	binding := &CVariableBinding{
		VarName: varName,
		Type: &core.TypeInfo{
			TypeFQN:    typeStr,
			Confidence: 1.0,
			Source:     declarationSource,
		},
		Location: loc,
	}
	e.appendBinding(functionFQN, binding)
}

// AddScope replaces (or installs) a complete scope for a function. Used
// by tests or by callers that want to batch-build a scope before
// publishing it to the engine. Nil scopes are ignored.
func (e *CTypeInferenceEngine) AddScope(scope *CFunctionScope) {
	if scope == nil || scope.FunctionFQN == "" {
		return
	}
	e.scopeMutex.Lock()
	e.Scopes[scope.FunctionFQN] = scope
	e.scopeMutex.Unlock()
}

// GetScope returns the scope for functionFQN, or nil if none exists.
func (e *CTypeInferenceEngine) GetScope(functionFQN string) *CFunctionScope {
	e.scopeMutex.RLock()
	defer e.scopeMutex.RUnlock()
	return e.Scopes[functionFQN]
}

// HasScope reports whether a scope exists for functionFQN.
func (e *CTypeInferenceEngine) HasScope(functionFQN string) bool {
	e.scopeMutex.RLock()
	defer e.scopeMutex.RUnlock()
	_, ok := e.Scopes[functionFQN]
	return ok
}

// GetAllScopes returns a snapshot copy of every registered scope.
func (e *CTypeInferenceEngine) GetAllScopes() map[string]*CFunctionScope {
	e.scopeMutex.RLock()
	defer e.scopeMutex.RUnlock()
	out := make(map[string]*CFunctionScope, len(e.Scopes))
	maps.Copy(out, e.Scopes)
	return out
}

// appendBinding installs binding inside the scope keyed by functionFQN,
// creating the scope on demand. The mutex protects the map mutation
// only — the per-scope slice is appended after re-acquiring the lock,
// so concurrent ExtractVariableType calls on the same function are
// serialised through this single lock.
func (e *CTypeInferenceEngine) appendBinding(functionFQN string, binding *CVariableBinding) {
	e.scopeMutex.Lock()
	defer e.scopeMutex.Unlock()
	scope, ok := e.Scopes[functionFQN]
	if !ok {
		scope = NewCFunctionScope(functionFQN)
		e.Scopes[functionFQN] = scope
	}
	scope.AddVariable(binding)
}
