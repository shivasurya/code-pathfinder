package builder

import (
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
)

// languageCpp is the Node.Language tag emitted by the C++ parser.
// Mirrors `parser_c.go:languageCpp`.
const languageCpp = "cpp"

// nodeMethodDeclaration is the Node.Type emitted for inline method
// declarations and out-of-line method definitions inside a class body.
const nodeMethodDeclaration = "method_declaration"

// nodeClassDeclaration is the Node.Type emitted for `class C { ... }`
// (and `parseCppStructSpecifier` for struct-as-class).
const nodeClassDeclaration = "class_declaration"

// nodeFieldDeclaration is the Node.Type used for both data members and
// non-method field-shaped declarations inside a class body.
const nodeFieldDeclaration = "field_declaration"

// receiverThis is the conventional name of the implicit class instance
// in C++ method bodies — `this->method()` resolution short-circuits to
// the caller's enclosing class.
const receiverThis = "this"

// pointerPrefixes lists the pointer/reference qualifiers stripped from
// receiver type names before NamespaceIndex / ClassIndex lookup. The
// same type strings produced by the parser may be `Dog*`, `Dog&`, or
// `const Dog*` — every form must reduce to the bare `Dog`.
var pointerPrefixes = []string{"const ", "volatile "}

// pointerSuffixes lists trailing modifiers stripped from receiver type
// names. Order matters: long forms first so `&&` does not become `&`.
var pointerSuffixes = []string{"**", "*", "&&", "&"}

// BuildCppCallGraph constructs the C++ call graph using the same
// four-pass structure as the C builder, plus three C++-specific
// resolution paths exercised in Pass 4:
//
//  1. Namespace-/scope-qualified calls — `ns::func()`,
//     `ClassName::staticMethod()` — resolve directly through
//     `registry.NamespaceIndex`.
//  2. Method calls on typed receivers — `obj.method()` /
//     `obj->method()` — look up the receiver's declared type via the
//     type engine, then locate the method on that class.
//  3. `this->method()` — the receiver type is implicit (the caller's
//     enclosing class), so resolution skips the type engine and asks
//     the registry's class index directly.
//
// Plain free-function calls fall through to the same definition-
// preferring resolution used by the C builder
// (`resolveCCallTarget`), which makes the C++ builder a strict
// superset of the C one.
//
// The result is a stand-alone `*core.CallGraph` whose FQNs do not
// collide with C, Python, Go, or Java — `MergeCallGraphs` is safe to
// call with any combination of language graphs.
//
// Parameters:
//   - codeGraph:  parsed graph from graph.Initialize. Nil-safe.
//   - registry:   C++ module registry from PR-05. Nil-safe.
//   - typeEngine: C++ type inference engine from PR-06. Nil-safe.
//
// Returns a fully-populated CallGraph and a nil error. Errors are
// reserved for future failure modes; the current implementation never
// returns one.
func BuildCppCallGraph(
	codeGraph *graph.CodeGraph,
	registry *core.CppModuleRegistry,
	typeEngine *resolution.CppTypeInferenceEngine,
) (*core.CallGraph, error) {
	callGraph := core.NewCallGraph()
	if codeGraph == nil || registry == nil {
		return callGraph, nil
	}

	classes := collectCppClassesByFile(codeGraph)
	indexCppFunctions(codeGraph, callGraph, registry, classes)
	extractCppFunctionTypes(codeGraph, callGraph, typeEngine, classes)
	callSites := extractCppCallSites(callGraph)
	resolveCppCallSites(callSites, callGraph, registry, typeEngine, classes)

	return callGraph, nil
}

// =============================================================================
// Class-by-file index
// =============================================================================

// cppClassByteRange caches a class's source span for byte-range
// containment lookups. Methods declared inside a class body share its
// file and lie inside its [startByte, endByte) range; that is how we
// associate methods with their class without trusting parser-internal
// context tracking.
type cppClassByteRange struct {
	name        string
	packageName string
	startByte   uint32
	endByte     uint32
}

// collectCppClassesByFile groups every C++ class_declaration by the
// file that declares it. The result is keyed by absolute file path so
// per-file containment lookups stay O(C) where C is the (small) number
// of classes in that file.
func collectCppClassesByFile(codeGraph *graph.CodeGraph) map[string][]cppClassByteRange {
	classes := make(map[string][]cppClassByteRange)
	for _, node := range codeGraph.Nodes {
		if !isCppClassNode(node) {
			continue
		}
		classes[node.File] = append(classes[node.File], cppClassByteRange{
			name:        node.Name,
			packageName: node.PackageName,
			startByte:   node.SourceLocation.StartByte,
			endByte:     node.SourceLocation.EndByte,
		})
	}
	return classes
}

// isCppClassNode is true when node is a usable C++ class declaration —
// well-named, file-anchored, and carrying a byte range. Anonymous
// classes (Name == "") are intentionally excluded; they cannot
// contribute to FQNs.
func isCppClassNode(node *graph.Node) bool {
	return node != nil &&
		node.Language == languageCpp &&
		node.Type == nodeClassDeclaration &&
		node.Name != "" &&
		node.File != "" &&
		node.SourceLocation != nil
}

// enclosingCppClass returns the smallest class byte-range in classes
// whose [startByte, endByte) span contains node's start byte, or nil
// when none does (free function / file-scope declaration).
//
// Picking the innermost class is a deliberate choice: nested classes
// (`class Outer { class Inner { ... }; };`) require the inner range to
// win so methods of `Inner` are not mis-attributed to `Outer`.
func enclosingCppClass(node *graph.Node, classes map[string][]cppClassByteRange) *cppClassByteRange {
	if node == nil || node.SourceLocation == nil {
		return nil
	}
	candidates := classes[node.File]
	if len(candidates) == 0 {
		return nil
	}
	pos := node.SourceLocation.StartByte
	var best *cppClassByteRange
	for i := range candidates {
		c := &candidates[i]
		if pos < c.startByte || pos >= c.endByte {
			continue
		}
		if best == nil || (c.endByte-c.startByte) < (best.endByte-best.startByte) {
			best = c
		}
	}
	return best
}

// =============================================================================
// Pass 1 — index functions
// =============================================================================

// indexCppFunctions records every C++ function and method in
// callGraph.Functions under its qualified FQN. The FQN composition
// mirrors PR-05's BuildCppModuleRegistry so cross-component lookups
// stay consistent:
//
//	free function (no namespace): "prefix::name"
//	free function with namespace: "prefix::ns::name"
//	class method:                 "prefix::[ns::]Class::name"
//
// In addition, the function ensures every recorded FQN appears in the
// registry's FunctionIndex / NamespaceIndex / ClassIndex tables so
// later passes can look up callees without re-deriving the FQN.
func indexCppFunctions(
	codeGraph *graph.CodeGraph,
	callGraph *core.CallGraph,
	registry *core.CppModuleRegistry,
	classes map[string][]cppClassByteRange,
) {
	for _, node := range codeGraph.Nodes {
		if !isCppFunctionNode(node) {
			continue
		}
		prefix, ok := registry.FileToPrefix[node.File]
		if !ok {
			continue
		}
		cls := enclosingCppClass(node, classes)
		fqn := composeCppFQN(prefix, node, cls)
		callGraph.Functions[fqn] = node

		// Free functions stay reachable through the bare-name
		// FunctionIndex so the C-style fallthrough in Pass 4 can find
		// them with no namespace context.
		if cls == nil && node.Type == "function_definition" {
			appendUniqueFQN(registry.FunctionIndex, node.Name, fqn)
		}

		key := composeCppScopeKey(node, cls)
		if key != "" {
			registry.NamespaceIndex[key] = fqn
		}
	}
}

// isCppFunctionNode is true when node is a usable C++ function- or
// method-shaped node with a name and file.
func isCppFunctionNode(node *graph.Node) bool {
	if node == nil || node.Language != languageCpp || node.Name == "" || node.File == "" {
		return false
	}
	return node.Type == "function_definition" || node.Type == nodeMethodDeclaration
}

// composeCppFQN composes the canonical C++ FQN for a function or method
// node, using its enclosing class (if any) for the class component.
func composeCppFQN(prefix string, node *graph.Node, cls *cppClassByteRange) string {
	scope := composeCppScope(node, cls)
	if scope == "" {
		return prefix + fqnSeparator + node.Name
	}
	return prefix + fqnSeparator + scope + fqnSeparator + node.Name
}

// composeCppScope returns the namespace + class chain that prefixes a
// function name in its FQN, joined by `::`. Empty when the function is
// a top-level free function with no namespace.
func composeCppScope(node *graph.Node, cls *cppClassByteRange) string {
	switch {
	case cls != nil:
		return joinScopeParts(cls.packageName, cls.name)
	case node.PackageName != "":
		return node.PackageName
	}
	return ""
}

// composeCppScopeKey returns the lookup key used by NamespaceIndex —
// `[ns::]Class::method` for methods, `ns::name` for namespaced free
// functions, "" when the function has no qualifying scope.
func composeCppScopeKey(node *graph.Node, cls *cppClassByteRange) string {
	scope := composeCppScope(node, cls)
	if scope == "" {
		return ""
	}
	return scope + fqnSeparator + node.Name
}

// joinScopeParts joins non-empty scope tokens with `::`. Used so
// callers can pass `node.PackageName` unconditionally without having
// to special-case the no-namespace path.
func joinScopeParts(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, fqnSeparator)
}

// =============================================================================
// Pass 2 — extract types
// =============================================================================

// extractCppFunctionTypes registers explicit return types, parameter
// symbols, class method return types, and class field types.
//
// Free-function and parameter handling mirrors the C builder so the
// resolver can resolve C++ free-function calls identically. Class
// method/field tracking augments the C++ type engine for receiver-typed
// resolution in Pass 4.
func extractCppFunctionTypes(
	codeGraph *graph.CodeGraph,
	callGraph *core.CallGraph,
	typeEngine *resolution.CppTypeInferenceEngine,
	classes map[string][]cppClassByteRange,
) {
	for fqn, node := range callGraph.Functions {
		if isDeclaration(node) {
			// Declarations still register class methods on the type
			// engine — that is the only place inline header method
			// signatures appear — but they do not contribute return
			// types or parameter symbols (no body).
			registerCppClassMember(node, typeEngine, classes)
			continue
		}
		if typeEngine != nil {
			typeEngine.ExtractReturnType(fqn, node.ReturnType)
		}
		registerCParameters(callGraph, fqn, node)
		registerCppClassMember(node, typeEngine, classes)
	}

	if typeEngine != nil {
		registerCppClassFields(codeGraph, typeEngine, classes)
	}
}

// registerCppClassMember records a method's return type on the type
// engine when the node is enclosed by a class. No-op for free
// functions and for `void` returns.
func registerCppClassMember(
	node *graph.Node,
	typeEngine *resolution.CppTypeInferenceEngine,
	classes map[string][]cppClassByteRange,
) {
	if typeEngine == nil || node == nil || node.ReturnType == "" {
		return
	}
	cls := enclosingCppClass(node, classes)
	if cls == nil {
		return
	}
	typeEngine.RegisterClassMethod(cls.name, node.Name, node.ReturnType)
}

// registerCppClassFields walks every field_declaration and records its
// declared type on the type engine, keyed by enclosing class name.
// Used in Pass 4 to resolve `obj.field.method()` chains; the field
// type tells the resolver which class to look up the method on.
func registerCppClassFields(
	codeGraph *graph.CodeGraph,
	typeEngine *resolution.CppTypeInferenceEngine,
	classes map[string][]cppClassByteRange,
) {
	for _, node := range codeGraph.Nodes {
		if !isCppFieldNode(node) {
			continue
		}
		cls := enclosingCppClass(node, classes)
		if cls == nil {
			continue
		}
		typeEngine.RegisterClassField(cls.name, node.Name, node.DataType)
	}
}

// isCppFieldNode is true when node is a usable C++ field declaration
// (data member with a type and name). Method declarations also use the
// field_declaration node type but carry no DataType, so the DataType
// check naturally filters them.
func isCppFieldNode(node *graph.Node) bool {
	return node != nil &&
		node.Language == languageCpp &&
		node.Type == nodeFieldDeclaration &&
		node.Name != "" &&
		node.DataType != ""
}

// =============================================================================
// Pass 3 — extract call sites
// =============================================================================

// extractCppCallSites walks the parser-emitted edges from each C++
// function/method to its call_expression children and emits one
// CallSiteInternal per call. The call shape (free, method, qualified)
// is preserved on the call node's metadata; this pass copies it
// forward so Pass 4 can dispatch without re-walking the AST.
func extractCppCallSites(callGraph *core.CallGraph) []*CallSiteInternal {
	sites := make([]*CallSiteInternal, 0)
	for callerFQN, fnNode := range callGraph.Functions {
		if isDeclaration(fnNode) {
			continue
		}
		for _, edge := range fnNode.OutgoingEdges {
			callNode := edge.To
			if !isCppCallNode(callNode) {
				continue
			}
			sites = append(sites, &CallSiteInternal{
				CallerFQN:    callerFQN,
				CallerFile:   fnNode.File,
				CallLine:     callNode.LineNumber,
				FunctionName: callNode.Name,
				ObjectName:   stringMetadata(callNode, "receiver"),
				Arguments:    append([]string(nil), callNode.MethodArgumentsValue...),
			})
		}
	}
	return sites
}

// isCppCallNode is true when node represents a C++ call_expression
// with a usable target.
func isCppCallNode(node *graph.Node) bool {
	return node != nil &&
		node.Language == languageCpp &&
		node.Type == "call_expression" &&
		node.Name != ""
}

// =============================================================================
// Pass 4 — resolve call sites
// =============================================================================

// resolveCppCallSites resolves every call site and adds the
// corresponding edge / CallSite record to the call graph. Unresolved
// sites are still recorded (Resolved=false) for diagnostics — stdlib
// and external calls remain visible.
func resolveCppCallSites(
	sites []*CallSiteInternal,
	callGraph *core.CallGraph,
	registry *core.CppModuleRegistry,
	typeEngine *resolution.CppTypeInferenceEngine,
	classes map[string][]cppClassByteRange,
) {
	for _, cs := range sites {
		targetFQN, resolved := resolveCppCallTarget(cs, callGraph, registry, typeEngine, classes)
		callSite := buildCCallSite(cs, targetFQN, resolved)
		callGraph.AddCallSite(cs.CallerFQN, callSite)
		if resolved {
			callGraph.AddEdge(cs.CallerFQN, targetFQN)
		}
	}
}

// resolveCppCallTarget implements the C++-specific resolution order:
//
//  1. Qualified call (`ns::func`, `Class::staticMethod`) — direct
//     NamespaceIndex lookup with the full qualified name.
//  2. `this->method()` — receiver type is implicit; look up the
//     method on the caller's enclosing class.
//  3. Method on typed receiver — find the receiver's declared type
//     via the type engine, then look up the method on that class.
//  4. C-style fallthrough — definition-preferring lookup that
//     mirrors `resolveCCallTarget`.
//
// Each step short-circuits on the first hit; later steps are tried
// only if earlier ones miss.
func resolveCppCallTarget(
	cs *CallSiteInternal,
	callGraph *core.CallGraph,
	registry *core.CppModuleRegistry,
	typeEngine *resolution.CppTypeInferenceEngine,
	classes map[string][]cppClassByteRange,
) (string, bool) {
	if cs.FunctionName == "" {
		return "", false
	}

	if fqn, ok := lookupQualifiedCall(cs.FunctionName, registry); ok {
		return fqn, true
	}
	if cs.ObjectName == receiverThis {
		if fqn, ok := lookupThisMethod(cs, callGraph, registry, classes); ok {
			return fqn, true
		}
	} else if cs.ObjectName != "" {
		if fqn, ok := lookupReceiverMethod(cs, registry, typeEngine); ok {
			return fqn, true
		}
	}
	return resolveCCallTarget(cs, callGraph, &registry.CModuleRegistry)
}

// lookupQualifiedCall handles `ns::func` and `Class::staticMethod` by
// querying NamespaceIndex with the verbatim call name. The check is
// scoped to names containing `::` so plain `func()` calls fall through
// to later stages without an extra map lookup.
func lookupQualifiedCall(name string, registry *core.CppModuleRegistry) (string, bool) {
	if !strings.Contains(name, fqnSeparator) {
		return "", false
	}
	fqn, ok := registry.NamespaceIndex[name]
	return fqn, ok
}

// lookupReceiverMethod resolves `obj.method()` / `obj->method()` by
// looking up the receiver's declared type in the type engine and then
// finding the method on that class via NamespaceIndex.
func lookupReceiverMethod(
	cs *CallSiteInternal,
	registry *core.CppModuleRegistry,
	typeEngine *resolution.CppTypeInferenceEngine,
) (string, bool) {
	if typeEngine == nil {
		return "", false
	}
	scope := typeEngine.GetScope(cs.CallerFQN)
	if scope == nil {
		return "", false
	}
	binding := scope.GetVariable(cs.ObjectName)
	if binding == nil || binding.Type == nil {
		return "", false
	}
	className := normaliseTypeName(binding.Type.TypeFQN)
	if className == "" {
		return "", false
	}
	return findMethodOnClass(className, cs.FunctionName, registry)
}

// lookupThisMethod handles `this->method()` by deriving the caller's
// enclosing class from the caller node and then locating the method on
// that class.
func lookupThisMethod(
	cs *CallSiteInternal,
	callGraph *core.CallGraph,
	registry *core.CppModuleRegistry,
	classes map[string][]cppClassByteRange,
) (string, bool) {
	caller, ok := callGraph.Functions[cs.CallerFQN]
	if !ok {
		return "", false
	}
	cls := enclosingCppClass(caller, classes)
	if cls == nil {
		return "", false
	}
	return findMethodOnClass(cls.name, cs.FunctionName, registry)
}

// findMethodOnClass tries the registry's NamespaceIndex with each
// known qualifier prefix that could match a method on className. The
// canonical key is `Class::method` (no namespace), but classes living
// in a namespace appear as `ns::Class::method`; we scan the index
// values keyed on the bare-class form and fall back to a structural
// suffix match when neither key shape exists yet.
func findMethodOnClass(
	className, methodName string,
	registry *core.CppModuleRegistry,
) (string, bool) {
	bareKey := className + fqnSeparator + methodName
	if fqn, ok := registry.NamespaceIndex[bareKey]; ok {
		return fqn, true
	}

	// `Class::method` may have been registered as `ns::Class::method`
	// because the class lives inside a namespace. Scan the namespace
	// index for any key whose tail matches `Class::method`.
	suffix := fqnSeparator + bareKey
	for key, fqn := range registry.NamespaceIndex {
		if strings.HasSuffix(key, suffix) {
			return fqn, true
		}
	}
	return "", false
}

// normaliseTypeName strips C++ qualifiers (`const`, `volatile`) and
// pointer/reference suffixes (`*`, `**`, `&`, `&&`) from a type
// expression so it can be matched against the bare class names stored
// in the registry's ClassIndex.
//
// Templates are left intact (`std::vector<int>` stays
// `std::vector<int>`); resolving template instantiations is a Phase 2
// concern. Nested namespace prefixes (`ns::Type`) are also left as-is —
// the registry's NamespaceIndex stores the same form.
func normaliseTypeName(raw string) string {
	t := strings.TrimSpace(raw)
	for _, p := range pointerPrefixes {
		t = strings.TrimPrefix(t, p)
	}
	t = strings.TrimSpace(t)
	for _, s := range pointerSuffixes {
		for strings.HasSuffix(t, s) {
			t = strings.TrimSuffix(t, s)
			t = strings.TrimSpace(t)
		}
	}
	return t
}
