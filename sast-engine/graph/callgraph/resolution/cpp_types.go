package resolution

import (
	"sync"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// autoTypeName is the literal C++ keyword for placeholder types (`auto x = ...`).
// Detected exactly — qualified forms such as `auto*` or `auto&` are kept
// as-is because the parser already strips the keyword from those.
const autoTypeName = "auto"

// autoSource is the TypeInfo.Source value used for unresolved `auto`
// declarations. Distinct from declarationSource so resolvers can skip
// auto bindings until Phase 2 deduces a concrete type.
const autoSource = "unresolved_auto"

// CppTypeInferenceEngine extends CTypeInferenceEngine with C++ class
// member tracking. By embedding the C engine it inherits every
// scope- and return-type method, so callers can use a single engine to
// resolve both C-style functions and C++ classes.
//
// In addition to the C-level data, it indexes:
//
//   - Method return types per class — used by call-graph resolution to
//     compute the type of `obj.method()` once the receiver type is
//     known.
//   - Field types per class — used by call-graph resolution when a
//     method is invoked via a member like `this->buffer.write(...)`.
//
// The maps are keyed by bare class name (e.g. "Socket") rather than
// fully-qualified class FQN; that mirrors how the parser emits class
// declarations and keeps lookups fast on hot paths. Callers requiring
// disambiguation across namespaces should pass FQNs explicitly to
// RegisterClassMethod.
type CppTypeInferenceEngine struct {
	// CTypeInferenceEngine provides function- and variable-level
	// indexing. Embedded by value so methods like ExtractReturnType,
	// GetScope, and GetVariable resolve uniformly through the C++ engine.
	CTypeInferenceEngine

	// CppRegistry is the C++-aware module registry. The embedded C
	// engine holds a pointer to its CModuleRegistry for the C-only
	// lookups; CppRegistry preserves access to NamespaceIndex and
	// ClassIndex without forcing callers to type-assert.
	CppRegistry *core.CppModuleRegistry

	// ClassMethods maps className -> methodName -> return type. nil
	// outer entries are created lazily on first registration.
	ClassMethods map[string]map[string]*core.TypeInfo

	// ClassFields maps className -> fieldName -> field type. Same
	// lazy-allocation contract as ClassMethods.
	ClassFields map[string]map[string]*core.TypeInfo

	classMethodMutex sync.RWMutex
	classFieldMutex  sync.RWMutex
}

// NewCppTypeInferenceEngine constructs an engine wired to a C++ module
// registry. The embedded C engine is bound to the same root by
// reference (it borrows registry's CModuleRegistry), so any field
// added to the registry post-construction is visible to both.
//
// A nil registry is permitted; the engine still functions for tests
// and isolated extraction.
func NewCppTypeInferenceEngine(registry *core.CppModuleRegistry) *CppTypeInferenceEngine {
	var cReg *core.CModuleRegistry
	if registry != nil {
		cReg = &registry.CModuleRegistry
	}
	return &CppTypeInferenceEngine{
		CTypeInferenceEngine: *NewCTypeInferenceEngine(cReg),
		CppRegistry:          registry,
		ClassMethods:         make(map[string]map[string]*core.TypeInfo),
		ClassFields:          make(map[string]map[string]*core.TypeInfo),
	}
}

// =============================================================================
// auto handling
// =============================================================================

// ExtractVariableType overrides the embedded C engine's behaviour to
// recognise the C++ `auto` placeholder. Auto declarations are recorded
// with Confidence=0 and Source="unresolved_auto" so later inference
// phases can find and refine them; resolvers gate on Confidence>=1.0
// for explicit-only resolution and skip these.
//
// All non-auto types delegate to the C engine for identical handling.
func (e *CppTypeInferenceEngine) ExtractVariableType(functionFQN, varName, typeStr string, loc Location) {
	if functionFQN == "" || varName == "" || typeStr == "" {
		return
	}
	if typeStr != autoTypeName {
		e.CTypeInferenceEngine.ExtractVariableType(functionFQN, varName, typeStr, loc)
		return
	}
	binding := &CVariableBinding{
		VarName: varName,
		Type: &core.TypeInfo{
			TypeFQN:    autoTypeName,
			Confidence: 0.0,
			Source:     autoSource,
		},
		Location: loc,
	}
	e.appendBinding(functionFQN, binding)
}

// =============================================================================
// Class method registration
// =============================================================================

// RegisterClassMethod records the explicit return type of methodName on
// className. Empty arguments are silently dropped. Calling the function
// twice for the same key replaces the previous entry — the most recent
// declaration wins, mirroring C++ overload behaviour where redeclarations
// must agree.
//
// Safe for concurrent use.
func (e *CppTypeInferenceEngine) RegisterClassMethod(className, methodName, returnType string) {
	if className == "" || methodName == "" || returnType == "" || returnType == "void" {
		return
	}
	info := &core.TypeInfo{
		TypeFQN:    returnType,
		Confidence: 1.0,
		Source:     declarationSource,
	}
	e.classMethodMutex.Lock()
	defer e.classMethodMutex.Unlock()
	methods, ok := e.ClassMethods[className]
	if !ok {
		methods = make(map[string]*core.TypeInfo)
		e.ClassMethods[className] = methods
	}
	methods[methodName] = info
}

// GetMethodReturnType looks up the recorded return type of methodName
// on className. Returns nil when the class is unknown or the method is
// unregistered (including void methods, which are intentionally not
// stored).
func (e *CppTypeInferenceEngine) GetMethodReturnType(className, methodName string) *core.TypeInfo {
	e.classMethodMutex.RLock()
	defer e.classMethodMutex.RUnlock()
	if methods, ok := e.ClassMethods[className]; ok {
		return methods[methodName]
	}
	return nil
}

// HasClassMethod reports whether a method type has been registered for
// className/methodName.
func (e *CppTypeInferenceEngine) HasClassMethod(className, methodName string) bool {
	return e.GetMethodReturnType(className, methodName) != nil
}

// =============================================================================
// Class field registration
// =============================================================================

// RegisterClassField records the explicit type of fieldName on
// className. Empty arguments are silently dropped. Like
// RegisterClassMethod, repeated calls overwrite — duplicate field
// declarations should never happen in well-formed C++.
//
// Safe for concurrent use.
func (e *CppTypeInferenceEngine) RegisterClassField(className, fieldName, typeStr string) {
	if className == "" || fieldName == "" || typeStr == "" {
		return
	}
	info := &core.TypeInfo{
		TypeFQN:    typeStr,
		Confidence: 1.0,
		Source:     declarationSource,
	}
	e.classFieldMutex.Lock()
	defer e.classFieldMutex.Unlock()
	fields, ok := e.ClassFields[className]
	if !ok {
		fields = make(map[string]*core.TypeInfo)
		e.ClassFields[className] = fields
	}
	fields[fieldName] = info
}

// GetFieldType looks up the recorded type of fieldName on className.
// Returns nil when the class is unknown or the field is unregistered.
func (e *CppTypeInferenceEngine) GetFieldType(className, fieldName string) *core.TypeInfo {
	e.classFieldMutex.RLock()
	defer e.classFieldMutex.RUnlock()
	if fields, ok := e.ClassFields[className]; ok {
		return fields[fieldName]
	}
	return nil
}

// HasClassField reports whether a field type has been registered for
// className/fieldName.
func (e *CppTypeInferenceEngine) HasClassField(className, fieldName string) bool {
	return e.GetFieldType(className, fieldName) != nil
}
