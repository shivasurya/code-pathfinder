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
//
// Phase 2 (PR-02): if the C++ resolver and the embedded C resolver both
// fail, the C++ stdlib registry is consulted (handled inside
// resolveCppCallTarget). The optional *core.CStdlibFunction return value
// flows return type and security tag into the emitted CallSite.
func resolveCppCallSites(
	sites []*CallSiteInternal,
	callGraph *core.CallGraph,
	registry *core.CppModuleRegistry,
	typeEngine *resolution.CppTypeInferenceEngine,
	classes map[string][]cppClassByteRange,
) {
	for _, cs := range sites {
		targetFQN, resolved, stdlibFn := resolveCppCallTarget(cs, callGraph, registry, typeEngine, classes)
		callSite := buildCCallSite(cs, targetFQN, resolved, stdlibFn)
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
//
// Phase 2 (PR-02): a stdlib step is inserted between project-internal
// resolution and the C-fallthrough. The C++ stdlib registry is consulted
// for namespaced free functions (std::move) and for class methods on a
// receiver whose type the type engine has identified (vec.push_back
// where vec is std::vector<int>). The C-fallthrough also picks up <stdio.h>
// and friends via the embedded C registry's stdlib loader.
func resolveCppCallTarget(
	cs *CallSiteInternal,
	callGraph *core.CallGraph,
	registry *core.CppModuleRegistry,
	typeEngine *resolution.CppTypeInferenceEngine,
	classes map[string][]cppClassByteRange,
) (string, bool, *core.CStdlibFunction) {
	if cs.FunctionName == "" {
		return "", false, nil
	}

	if fqn, ok := lookupQualifiedCall(cs.FunctionName, registry); ok {
		return fqn, true, nil
	}
	if cs.ObjectName == receiverThis {
		if fqn, ok := lookupThisMethod(cs, callGraph, registry, classes); ok {
			return fqn, true, nil
		}
	} else if cs.ObjectName != "" {
		if fqn, ok := lookupReceiverMethod(cs, registry, typeEngine); ok {
			return fqn, true, nil
		}
		// Phase 2: try C++ stdlib method dispatch on the receiver's type.
		if registry.StdlibCppRegistry != nil {
			if fqn, fn := lookupCppStdlibMethod(cs, registry, typeEngine); fn != nil {
				return fqn, true, fn
			}
		}
	}

	// Phase 2: try C++ stdlib free-function (std::move, std::swap, …) before
	// falling through to the C resolver. Qualified-call lookup above already
	// caught the project-internal forms; this step covers the registry side.
	if registry.StdlibCppRegistry != nil {
		if fqn, fn := lookupCppStdlibFreeFunction(cs, &registry.CModuleRegistry, registry.StdlibCppRegistry); fn != nil {
			return fqn, true, fn
		}
	}

	return resolveCCallTarget(cs, callGraph, &registry.CModuleRegistry)
}

// lookupCppStdlibMethod resolves `obj.method()` against the C++ stdlib
// registry. Steps:
//
//  1. Look up the receiver's declared type via the type engine.
//  2. Strip template arguments to get the registry key
//     ("std::vector<int>" → "std::vector").
//  3. Walk the caller's `#include <...>` list and ask the registry for
//     a method with that name on that class. First hit wins.
//
// Returns the synthetic FQN ("<class>::<method>") and the registry record
// on success; "", nil on miss. A receiver type the engine cannot resolve
// is silently treated as a miss — Phase 1's behaviour for unresolved
// receivers is preserved.
func lookupCppStdlibMethod(
	cs *CallSiteInternal,
	registry *core.CppModuleRegistry,
	typeEngine *resolution.CppTypeInferenceEngine,
) (string, *core.CStdlibFunction) {
	if typeEngine == nil {
		return "", nil
	}
	scope := typeEngine.GetScope(cs.CallerFQN)
	if scope == nil {
		return "", nil
	}
	binding := scope.GetVariable(cs.ObjectName)
	if binding == nil || binding.Type == nil {
		return "", nil
	}
	receiverFQN := canonicalizeStdlibType(normaliseTypeName(binding.Type.TypeFQN))
	if receiverFQN == "" {
		return "", nil
	}
	prefix, ok := registry.FileToPrefix[cs.CallerFile]
	if !ok {
		return "", nil
	}
	for _, header := range registry.SystemIncludes[prefix] {
		method, err := registry.StdlibCppRegistry.GetMethod(header, receiverFQN, cs.FunctionName)
		if err == nil && method != nil {
			// Substitute template parameters using the receiver's concrete
			// args (e.g. vector<int>::operator[] T& → int&). Phase 2 covers
			// the common single-T cases; extension to K/V/U is built in.
			cloned := substituteTemplateMethodReturn(method, binding.Type.TypeFQN)
			fqn := receiverFQN + fqnSeparator + cs.FunctionName
			return fqn, cloned
		}
	}
	return "", nil
}

// lookupCppStdlibFreeFunction asks the registry for a namespaced free
// function. The call's FunctionName MUST already be the qualified form
// ("std::move") for the registry's GetFreeFunction to succeed — bare
// names like "move" without the namespace are a different lookup path
// and stay out of scope here.
//
// The lookup happens in two stages:
//
//  1. Walk the caller file's direct system includes. This is the fast
//     path and catches the common case where the calling file
//     `#include <utility>` itself.
//  2. If the direct walk doesn't yield a hit, fall back to scanning
//     every header in the manifest. Real-world C++ code routinely
//     relies on transitive includes (a header pulling in another that
//     pulls in <utility>), so without this fallback std::move /
//     std::forward / std::swap fail to resolve in the majority of
//     calling files.
//
// The fallback is bounded to fully-qualified names (containing "::") so
// unqualified project-internal symbols can never accidentally bind to a
// stdlib entry. Performance is fine: the scan is O(headers) per
// unresolved namespaced call, ~120 headers on libstdc++; first hit wins.
func lookupCppStdlibFreeFunction(
	cs *CallSiteInternal,
	cReg *core.CModuleRegistry,
	cppLoader core.CppStdlibLoader,
) (string, *core.CStdlibFunction) {
	if !strings.Contains(cs.FunctionName, fqnSeparator) || cppLoader == nil {
		return "", nil
	}

	// Stage 1: direct system includes from the caller file.
	if prefix, ok := cReg.FileToPrefix[cs.CallerFile]; ok {
		for _, header := range cReg.SystemIncludes[prefix] {
			if fn, err := cppLoader.GetFreeFunction(header, cs.FunctionName); err == nil && fn != nil {
				return fn.FQN, fn
			}
		}
	}

	// Stage 2: transitive-include fallback. Walk every manifest header.
	// We're trading a constant-time loop per unresolved namespaced call
	// for the ability to resolve symbols pulled in transitively. First
	// hit wins; stdlib FQNs are unique across headers so order doesn't
	// affect correctness (only speed-of-first-hit).
	for _, header := range cppLoader.ListHeaders() {
		if fn, err := cppLoader.GetFreeFunction(header, cs.FunctionName); err == nil && fn != nil {
			return fn.FQN, fn
		}
	}
	return "", nil
}

// canonicalizeStdlibType strips the template-argument suffix from a type
// FQN to produce the registry key used by GetClass / GetMethod. Examples:
//
//	"std::vector<int>"               → "std::vector"
//	"std::map<std::string, int>"     → "std::map"
//	"std::unique_ptr<MyClass>"       → "std::unique_ptr"
//	"int"                            → "int"
//
// The first '<' wins; nested template arguments are ignored at the key
// level (the receiver's own FQN still carries them for substitution).
func canonicalizeStdlibType(typeFQN string) string {
	if idx := strings.IndexByte(typeFQN, '<'); idx > 0 {
		return strings.TrimSpace(typeFQN[:idx])
	}
	return typeFQN
}

// substituteTemplateMethodReturn replaces the canonical template parameter
// names (T, U, V, K) in the registry's recorded return type with the
// concrete arguments parsed from the receiver's FQN. Returns a SHALLOW
// COPY of the registry function with the substituted type — never mutates
// the registry's data, since the registry is shared across calls.
//
// Phase 2 covers common cases: T, T&, T*, const T&, std::pair<T, U>&. More
// elaborate forms (`typename T::iterator`, conditional types, parameter
// packs) fall through unchanged — the resolver still records the resolution,
// just with the original generic form. PR-04's `--diagnose-stdlib` will
// surface these as opportunities for future overlay refinement.
func substituteTemplateMethodReturn(method *core.CStdlibFunction, receiverFQN string) *core.CStdlibFunction {
	args := parseTemplateArgs(receiverFQN)
	if len(args) == 0 {
		return method
	}
	cloned := *method // shallow copy — Params slice still points at the original
	cloned.ReturnType = applyTemplateSubstitution(method.ReturnType, args)
	return &cloned
}

// parseTemplateArgs extracts the comma-separated template arguments from a
// type FQN. "std::vector<int>" → ["int"]; "std::map<std::string, int>" →
// ["std::string", "int"]. Returns nil for non-template FQNs. The parser is
// brace-counting so nested templates are handled correctly:
// "std::map<int, std::vector<int>>" → ["int", "std::vector<int>"].
func parseTemplateArgs(typeFQN string) []string {
	open := strings.IndexByte(typeFQN, '<')
	if open < 0 {
		return nil
	}
	closeIdx := strings.LastIndexByte(typeFQN, '>')
	if closeIdx <= open {
		return nil
	}
	body := typeFQN[open+1 : closeIdx]

	args := make([]string, 0, 2)
	depth := 0
	start := 0
	for i, r := range body {
		switch r {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(body[start:i]))
				start = i + 1
			}
		}
	}
	args = append(args, strings.TrimSpace(body[start:]))
	return args
}

// applyTemplateSubstitution replaces the canonical placeholder names with
// concrete types in returnType. Substitution is whole-word — we don't
// replace "T" inside "Type" or "std::pair". Registered placeholders, in
// the order the receiver's args bind to:
//
//	T  → first arg
//	U  → second arg
//	V  → third arg
//	K  → first arg (alias used by std::map / std::unordered_map)
//
// This matches the convention the cpp_stdlib_overlay.yaml authors use.
func applyTemplateSubstitution(returnType string, args []string) string {
	placeholders := []string{"T", "U", "V", "K"}
	out := returnType
	for i, ph := range placeholders {
		// "K" is an alias for the first arg in map-shaped containers — it
		// shares an index with T but is iterated separately so a return
		// type written as "K" still resolves when the registry exposes
		// only T/U bindings.
		idx := i
		if ph == "K" {
			idx = 0
		}
		if idx >= len(args) {
			continue
		}
		out = replaceWholeWord(out, ph, args[idx])
	}
	return out
}

// replaceWholeWord replaces every occurrence of `from` in `s` with `to`,
// but only when `from` is at a word boundary. This avoids T inside Type
// being replaced when args=["int"]: "Type" stays "Type", "T&" becomes "int&".
func replaceWholeWord(s, from, to string) string {
	if from == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if i+len(from) <= len(s) && s[i:i+len(from)] == from &&
			!isWordChar(byteAt(s, i-1)) && !isWordChar(byteAt(s, i+len(from))) {
			b.WriteString(to)
			i += len(from)
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func byteAt(s string, i int) byte {
	if i < 0 || i >= len(s) {
		return 0
	}
	return s[i]
}

func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') || b == '_'
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
