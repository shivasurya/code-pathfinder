package builder

import (
	"slices"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
)

// languageC is the Node.Language tag emitted by the C parser. Mirrors
// `parser_c.go:languageC`; duplicated here to avoid importing the
// parser package into a builder.
const languageC = "c"

// metaIsDeclaration is the Node.Metadata key set on function nodes that
// represent declarations (prototypes / extern decls) rather than
// definitions. Mirrors `parser_c.go:metaIsDeclaration`.
const metaIsDeclaration = "is_declaration"

// fqnSeparator is the C/C++ FQN delimiter — `relpath::funcname`.
const fqnSeparator = "::"

// declarationConfidenceSource is the TypeInfo.Source used for return
// types lifted from explicit declarations.
const declarationConfidenceSource = "declaration"

// resolutionFailedExternal is the CallSite.FailureReason used when a
// callee cannot be resolved within the project. C code calling stdlib
// (`printf`, `malloc`) and unknown function pointers both end up here.
const resolutionFailedExternal = "external_or_unresolved"

// BuildCCallGraph constructs the C call graph using a four-pass
// algorithm. The result is a stand-alone `*core.CallGraph` that can be
// merged into a unified graph via `MergeCallGraphs`.
//
// Passes:
//
//	Pass 1 — index every C function_definition under its FQN
//	         ("relpath::name") and populate registry.FunctionIndex for
//	         cross-file lookup. Declarations are kept with
//	         Metadata["is_declaration"]=true so Pass 4 can prefer
//	         definitions.
//
//	Pass 2 — populate return-type and parameter symbol tables from the
//	         already-parsed AST nodes (no AST walk). The type engine
//	         records every non-void return type with Confidence=1.0.
//
//	Pass 3 — emit one CallSiteInternal per call_expression child of a
//	         function_definition. Edges from the parser already link
//	         each call to its enclosing function, so this pass is a
//	         single deterministic walk over those edges.
//
//	Pass 4 — resolve each call site to a concrete FQN using the
//	         definition-preferring search order documented on
//	         resolveCCallTarget. Resolved sites add an edge; unresolved
//	         sites are stored as CallSite{Resolved:false} for
//	         diagnostics.
//
// Parameters:
//   - codeGraph:  parsed graph from graph.Initialize. Must be non-nil.
//   - registry:   C module registry from PR-05. Must be non-nil.
//   - typeEngine: C type inference engine from PR-06. Must be non-nil.
//
// Returns a fully-populated *core.CallGraph and a nil error. Errors are
// reserved for future failure modes (e.g. cancelled context); the
// current implementation never returns one.
func BuildCCallGraph(
	codeGraph *graph.CodeGraph,
	registry *core.CModuleRegistry,
	typeEngine *resolution.CTypeInferenceEngine,
) (*core.CallGraph, error) {
	callGraph := core.NewCallGraph()
	if codeGraph == nil || registry == nil {
		return callGraph, nil
	}

	indexCFunctions(codeGraph, callGraph, registry)
	extractCFunctionTypes(callGraph, typeEngine)
	callSites := extractCCallSites(callGraph)
	resolveCCallSites(callSites, callGraph, registry)

	return callGraph, nil
}

// =============================================================================
// Pass 1 — index functions
// =============================================================================

// indexCFunctions records every C function_definition node in
// callGraph.Functions under "relpath::name" and ensures the same FQN
// appears in registry.FunctionIndex (so `BuildCCallGraph` works even
// when the registry was constructed before the parser populated all
// node metadata).
func indexCFunctions(codeGraph *graph.CodeGraph, callGraph *core.CallGraph, registry *core.CModuleRegistry) {
	for _, node := range codeGraph.Nodes {
		if !isCFunctionNode(node) {
			continue
		}
		prefix, ok := registry.FileToPrefix[node.File]
		if !ok {
			continue
		}
		fqn := prefix + fqnSeparator + node.Name
		callGraph.Functions[fqn] = node
		appendUniqueFQN(registry.FunctionIndex, node.Name, fqn)
	}
}

// isCFunctionNode is true when node is a C function_definition with a
// usable name. Anonymous declarations (rare in practice) are skipped.
func isCFunctionNode(node *graph.Node) bool {
	return node != nil &&
		node.Language == languageC &&
		node.Type == "function_definition" &&
		node.Name != "" &&
		node.File != ""
}

// appendUniqueFQN adds fqn to index[name] unless it is already present.
func appendUniqueFQN(index map[string][]string, name, fqn string) {
	if slices.Contains(index[name], fqn) {
		return
	}
	index[name] = append(index[name], fqn)
}

// =============================================================================
// Pass 2 — extract return types and parameter symbols
// =============================================================================

// extractCFunctionTypes registers every function's return type with the
// type engine and records each parameter as a typed ParameterSymbol on
// the call graph. Both pieces of metadata are consumed by later passes
// (resolution and rule-engine queries).
func extractCFunctionTypes(callGraph *core.CallGraph, typeEngine *resolution.CTypeInferenceEngine) {
	for fqn, node := range callGraph.Functions {
		if isDeclaration(node) {
			continue
		}
		if typeEngine != nil {
			typeEngine.ExtractReturnType(fqn, node.ReturnType)
		}
		registerCParameters(callGraph, fqn, node)
	}
}

// registerCParameters writes one ParameterSymbol per declared parameter
// of node into callGraph.Parameters. Parameters with no name (anonymous
// or void) are skipped — they cannot be referenced and would only
// pollute symbol queries.
func registerCParameters(callGraph *core.CallGraph, fqn string, node *graph.Node) {
	if node == nil {
		return
	}
	for i, paramName := range node.MethodArgumentsValue {
		if paramName == "" {
			continue
		}
		typeAnnotation := ""
		if i < len(node.MethodArgumentsType) {
			typeAnnotation = node.MethodArgumentsType[i]
		}
		paramFQN := fqn + "." + paramName
		callGraph.Parameters[paramFQN] = &core.ParameterSymbol{
			Name:           paramName,
			TypeAnnotation: typeAnnotation,
			ParentFQN:      fqn,
			File:           node.File,
			Line:           node.LineNumber,
		}
	}
}

// =============================================================================
// Pass 3 — extract call sites
// =============================================================================

// extractCCallSites walks the outgoing edges added by the parser
// (`function_definition → call_expression`) and emits one
// CallSiteInternal per call. The parser already links every call to
// its enclosing function, so this pass is deterministic and avoids a
// second AST traversal.
func extractCCallSites(callGraph *core.CallGraph) []*CallSiteInternal {
	sites := make([]*CallSiteInternal, 0)
	for callerFQN, fnNode := range callGraph.Functions {
		if isDeclaration(fnNode) {
			continue
		}
		for _, edge := range fnNode.OutgoingEdges {
			callNode := edge.To
			if !isCCallNode(callNode) {
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

// isCCallNode is true when node represents a C call_expression with a
// usable target name.
func isCCallNode(node *graph.Node) bool {
	return node != nil &&
		node.Language == languageC &&
		node.Type == "call_expression" &&
		node.Name != ""
}

// stringMetadata returns the string at node.Metadata[key], or "" when
// the key is missing or the value is not a string.
func stringMetadata(node *graph.Node, key string) string {
	if node == nil || node.Metadata == nil {
		return ""
	}
	v, ok := node.Metadata[key].(string)
	if !ok {
		return ""
	}
	return v
}

// =============================================================================
// Pass 4 — resolve call sites
// =============================================================================

// resolveCCallSites attempts to resolve each call site to a concrete
// FQN. Resolved sites add a forward edge in the call graph and record a
// CallSite{Resolved:true} entry; unresolved sites are still recorded so
// rule writers and diagnostics can see external/unknown calls.
//
// Phase 2 (PR-02): when the project-internal lookup fails AND
// registry.StdlibRegistry is configured, the resolver consults the stdlib
// registry. A stdlib hit also flows return-type metadata and any overlay
// security tag into the emitted core.CallSite.
func resolveCCallSites(sites []*CallSiteInternal, callGraph *core.CallGraph, registry *core.CModuleRegistry) {
	for _, cs := range sites {
		targetFQN, resolved, stdlibFn := resolveCCallTarget(cs, callGraph, registry)
		callSite := buildCCallSite(cs, targetFQN, resolved, stdlibFn)
		callGraph.AddCallSite(cs.CallerFQN, callSite)
		if resolved {
			callGraph.AddEdge(cs.CallerFQN, targetFQN)
		}
	}
}

// resolveCCallTarget implements the definition-preferring resolution
// strategy. The order is intentional and is documented inline so future
// maintainers can adjust it without re-reading the spec:
//
//  1. Same-file lookup — the most common pattern (helper function in
//     the same .c file). Always wins because it is deterministic and
//     does not depend on include-resolution state.
//  2. Global definition lookup — scan registry.FunctionIndex for an
//     FQN that the call graph knows is a definition (not just a
//     declaration). This handles cross-file calls into another .c file.
//  3. Same-file declaration — accept a same-file declaration when no
//     definition exists project-wide (forward decls).
//  4. Included declaration — fall back to a header declaration
//     reachable through `#include "..."`. The edge points at the
//     declaration FQN; later phases can still treat it as the entry
//     point for a stdlib/third-party call.
//
// Returns ("", false, nil) when no candidate matches. When the third
// return value (the *core.CStdlibFunction) is non-nil, the caller knows the
// resolution went through the stdlib registry — return type, security tag,
// and confidence are read from that struct.
func resolveCCallTarget(
	cs *CallSiteInternal,
	callGraph *core.CallGraph,
	registry *core.CModuleRegistry,
) (string, bool, *core.CStdlibFunction) {
	if cs.FunctionName == "" {
		return "", false, nil
	}

	if fqn, ok := lookupSameFile(cs.CallerFile, cs.FunctionName, registry, callGraph, true); ok {
		return fqn, true, nil
	}
	if fqn, ok := lookupGlobalDefinition(cs.FunctionName, registry, callGraph); ok {
		return fqn, true, nil
	}
	if fqn, ok := lookupSameFile(cs.CallerFile, cs.FunctionName, registry, callGraph, false); ok {
		return fqn, true, nil
	}
	if fqn, ok := lookupViaIncludes(cs.CallerFile, cs.FunctionName, registry, callGraph); ok {
		return fqn, true, nil
	}

	// Phase 2 fallback: consult the stdlib registry by walking the caller's
	// system includes. First include with a matching symbol wins.
	if registry.StdlibRegistry != nil {
		if fqn, fn := lookupCStdlib(cs.CallerFile, cs.FunctionName, registry); fn != nil {
			return fqn, true, fn
		}
	}
	return "", false, nil
}

// lookupCStdlib resolves a free function against the C stdlib registry.
//
// The lookup runs in two stages:
//
//  1. Direct system includes — walk the caller file's own `#include <…>`
//     list. This is the fast path and covers the case where the calling
//     file directly pulls in the header that owns the symbol.
//  2. Manifest-wide fallback — when the direct walk doesn't yield a hit,
//     scan every header in the manifest. Real-world C codebases routinely
//     route stdlib pulls through a project-internal "common.h" header
//     (redis does this with `server.h`), so without this fallback every
//     `.c` file that doesn't redundantly `#include <string.h>` itself
//     would lose strlen / strcmp / memcpy / etc.
//
// First hit wins. Stdlib symbols are uniquely owned by one header in the
// real world, so first-hit-wins is effectively the same as best-match.
// The fallback is bounded to the manifest's loaded headers — symbols not
// in the manifest can never resolve, so there's no risk of binding a
// project-internal symbol to a stdlib FQN.
func lookupCStdlib(callerFile, funcName string, registry *core.CModuleRegistry) (string, *core.CStdlibFunction) {
	loader := registry.StdlibRegistry
	if loader == nil {
		return "", nil
	}

	// Stage 1: direct system includes from the caller file.
	if prefix, ok := registry.FileToPrefix[callerFile]; ok {
		for _, header := range registry.SystemIncludes[prefix] {
			if fn, err := loader.GetFunction(header, funcName); err == nil && fn != nil {
				return fn.FQN, fn
			}
		}
	}

	// Stage 2: transitive-include fallback. Walk every manifest header.
	// O(headers) per unresolved call — ~1900 headers on Linux glibc;
	// bounded and predictable. Mirrors the PR-04 C++ resolver fallback.
	for _, header := range loader.ListHeaders() {
		if fn, err := loader.GetFunction(header, funcName); err == nil && fn != nil {
			return fn.FQN, fn
		}
	}
	return "", nil
}

// lookupSameFile returns the FQN of a function named `name` declared in
// callerFile. When definitionsOnly is true, only definitions are
// returned (declarations are skipped); when false, any matching node
// is accepted.
func lookupSameFile(
	callerFile, name string,
	registry *core.CModuleRegistry,
	callGraph *core.CallGraph,
	definitionsOnly bool,
) (string, bool) {
	prefix, ok := registry.FileToPrefix[callerFile]
	if !ok {
		return "", false
	}
	fqn := prefix + fqnSeparator + name
	node, ok := callGraph.Functions[fqn]
	if !ok {
		return "", false
	}
	if definitionsOnly && isDeclaration(node) {
		return "", false
	}
	return fqn, true
}

// lookupGlobalDefinition scans every FQN registered for `name` in the
// module registry and returns the first one whose call-graph entry is
// a definition. Order follows registry.FunctionIndex insertion order
// (stable across runs because the registry walks codeGraph.Nodes which
// is append-only during build).
func lookupGlobalDefinition(name string, registry *core.CModuleRegistry, callGraph *core.CallGraph) (string, bool) {
	for _, candidate := range registry.FunctionIndex[name] {
		node, ok := callGraph.Functions[candidate]
		if !ok || isDeclaration(node) {
			continue
		}
		return candidate, true
	}
	return "", false
}

// lookupViaIncludes searches the headers transitively included by
// callerFile for a declaration of `name`. Used as a last resort so
// edges to declared-but-undefined functions (e.g. an extern handed off
// to another translation unit) still appear in the graph.
func lookupViaIncludes(
	callerFile, name string,
	registry *core.CModuleRegistry,
	callGraph *core.CallGraph,
) (string, bool) {
	callerPrefix, ok := registry.FileToPrefix[callerFile]
	if !ok {
		return "", false
	}
	for _, includedRel := range registry.Includes[callerPrefix] {
		fqn := includedRel + fqnSeparator + name
		if _, exists := callGraph.Functions[fqn]; exists {
			return fqn, true
		}
	}
	return "", false
}

// =============================================================================
// Helpers
// =============================================================================

// isDeclaration reports whether node is a function declaration (no
// body) rather than a definition. Used by Pass 4's resolution order to
// prefer the FQN backed by an actual function body.
func isDeclaration(node *graph.Node) bool {
	if node == nil {
		return false
	}
	v, ok := node.Metadata[metaIsDeclaration].(bool)
	return ok && v
}

// buildCCallSite composes a core.CallSite from the internal record and
// the resolution outcome. Tracking unresolved calls (rather than
// dropping them) enables stdlib/third-party rules to inspect external
// invocations.
//
// stdlibFn is non-nil only when resolution went through the stdlib registry
// (Phase 2). When set, its return type, confidence, and security tag are
// propagated into the emitted CallSite.
func buildCCallSite(cs *CallSiteInternal, targetFQN string, resolved bool, stdlibFn *core.CStdlibFunction) core.CallSite {
	site := core.CallSite{
		Target:    cs.FunctionName,
		Location:  core.Location{File: cs.CallerFile, Line: int(cs.CallLine)},
		Arguments: buildCallSiteArguments(cs.Arguments),
		Resolved:  resolved,
	}
	if !resolved {
		site.FailureReason = resolutionFailedExternal
		return site
	}
	site.TargetFQN = targetFQN
	if stdlibFn != nil {
		// Phase 2 stdlib resolution — populate type info from the registry.
		site.TypeConfidence = stdlibFn.Confidence
		site.TypeSource = stdlibSource
		site.InferredType = stdlibFn.ReturnType
		site.SecurityTag = stdlibFn.SecurityTag
		return site
	}
	// Project-internal resolution — Phase 1 path.
	// Confidence 1.0 because resolution went through the FQN registry,
	// not type inference. Source kept consistent with the explicit-types
	// convention used by the type engine.
	site.TypeConfidence = 1.0
	site.TypeSource = declarationConfidenceSource
	return site
}

// stdlibSource is the value stamped on CallSite.TypeSource when resolution
// went through the C/C++ stdlib registry. Distinct from the Phase 1
// declarationConfidenceSource so downstream consumers can filter on it.
const stdlibSource = "stdlib"
