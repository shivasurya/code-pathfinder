package registry

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// Node-type constants mirror the literals emitted by the C/C++ parsers in
// `graph/parser_c.go` and `graph/parser_cpp.go`. Keeping them as package-
// local constants avoids importing the parser package (which would create
// a cycle through `graph -> callgraph -> graph`) while still documenting
// the contract between the two packages.
const (
	cNodeFunctionDefinition = "function_definition"
	cNodeMethodDeclaration  = "method_declaration"
	cNodeClassDeclaration   = "class_declaration"
	cNodeIncludeStatement   = "include_statement"
)

// Language tags emitted by the C/C++ parsers. These mirror the
// `languageC` / `languageCpp` constants in the parser package.
const (
	languageC   = "c"
	languageCpp = "cpp"
)

// metaSystemInclude is the metadata key set on `include_statement` nodes
// to distinguish `#include <...>` from `#include "..."`. Defined in the
// parser package; duplicated here to avoid an import cycle.
const metaSystemInclude = "system_include"

// fqnSeparator is the delimiter used between every FQN component for
// C/C++. It matches C++'s native scope-resolution operator and is also
// used for C even though C has no namespaces — keeping a single
// separator simplifies cross-language consumers.
const fqnSeparator = "::"

// Conventional include search directories for `#include "..."`
// resolution. Order matters: the first matching path wins.
const (
	includeDirInclude = "include"
	includeDirSrc     = "src"
)

// BuildCModuleRegistry walks a CodeGraph and produces a CModuleRegistry
// suitable for C call-graph construction.
//
// The function:
//
//  1. Records every distinct C source file in `FileToPrefix`, mapping
//     absolute path to its project-relative form. Files outside
//     projectPath (relative path beginning with `..`) are skipped — they
//     cannot participate in project-local FQNs.
//  2. Indexes every `function_definition` node under its bare name, so
//     `FunctionIndex["create_buffer"]` lists every FQN defining that
//     symbol.
//  3. Resolves project-local `#include "..."` directives into a
//     file -> [included files] map via BuildCIncludeMap.
//
// The registry is the read-only foundation used by the call-graph
// builder (PR-07) to compute caller/callee FQNs and follow header-to-source
// edges.
//
// Parameters:
//   - projectPath: absolute path to the project root used as the FQN base.
//   - codeGraph:   parsed graph whose Nodes will be filtered by Language.
//
// Returns a fully-initialised, non-nil registry. An empty graph yields a
// registry with empty (but allocated) maps.
func BuildCModuleRegistry(projectPath string, codeGraph *graph.CodeGraph) *core.CModuleRegistry {
	registry := core.NewCModuleRegistry(projectPath)
	if codeGraph == nil {
		return registry
	}

	indexFilesAndFunctions(codeGraph, projectPath, languageC, registry, nil)
	registry.Includes = BuildCIncludeMap(projectPath, codeGraph, languageC)
	return registry
}

// BuildCppModuleRegistry walks a CodeGraph and produces a
// CppModuleRegistry suitable for C++ call-graph construction.
//
// In addition to everything BuildCModuleRegistry does for C, this:
//
//  1. Builds a class lookup table from `class_declaration` nodes so
//     methods can be qualified with their enclosing class name.
//  2. Indexes free functions whose `PackageName` carries a namespace
//     under `NamespaceIndex["ns::funcname"]`.
//  3. Indexes class methods (whether parsed as `method_declaration` or
//     `function_definition` whose body is inside a class) under
//     `NamespaceIndex["[ns::]Class::method"]` and records the class
//     itself in `ClassIndex`.
//
// Method-to-class association uses byte-range containment within the
// same file: a method is associated with the innermost class whose
// `[StartByte, EndByte]` range encloses the method's start byte. This
// keeps the registry independent of parser-internal context tracking.
func BuildCppModuleRegistry(projectPath string, codeGraph *graph.CodeGraph) *core.CppModuleRegistry {
	registry := core.NewCppModuleRegistry(projectPath)
	if codeGraph == nil {
		return registry
	}

	classes := collectCppClasses(codeGraph, projectPath, registry)
	indexFilesAndFunctions(codeGraph, projectPath, languageCpp, &registry.CModuleRegistry, func(node *graph.Node, prefix string) {
		switch node.Type {
		case cNodeFunctionDefinition:
			indexCppFreeFunction(node, classes, prefix, registry)
		case cNodeMethodDeclaration:
			indexCppMethod(node, classes, prefix, registry)
		}
	})

	registry.Includes = BuildCIncludeMap(projectPath, codeGraph, languageCpp)
	return registry
}

// BuildCIncludeMap resolves project-local `#include "..."` directives in
// a CodeGraph into a relative-path-keyed map.
//
// For every `include_statement` node whose Language matches the language
// argument and whose `system_include` metadata is false (i.e. quoted
// includes), the function searches a fixed set of project directories
// for the named header. When found, both the source and resolved file
// are stored as project-relative paths in the result.
//
// Search order (first match wins):
//
//  1. The directory containing the source file.
//  2. `<projectRoot>/include/<header>`
//  3. `<projectRoot>/src/<header>`
//  4. `<projectRoot>/<header>`
//
// System includes (`#include <...>`) are intentionally skipped — they
// are resolved later by a stdlib registry. Headers that cannot be
// located are silently dropped: a missing file is recorded by the
// parser as an include statement but contributes nothing to call-graph
// construction.
//
// The function never returns nil; an empty map signals "no resolvable
// project-local includes" rather than "registry not built".
func BuildCIncludeMap(projectPath string, codeGraph *graph.CodeGraph, language string) map[string][]string {
	includes := make(map[string][]string)
	if codeGraph == nil {
		return includes
	}

	for _, node := range codeGraph.Nodes {
		if !isProjectInclude(node, language) {
			continue
		}
		resolved := resolveLocalInclude(projectPath, node.File, node.Name)
		if resolved == "" {
			continue
		}
		relSource, ok := relativeProjectPath(projectPath, node.File)
		if !ok {
			continue
		}
		relResolved, ok := relativeProjectPath(projectPath, resolved)
		if !ok {
			continue
		}
		includes[relSource] = appendUnique(includes[relSource], relResolved)
	}
	return includes
}

// indexFilesAndFunctions populates FileToPrefix and FunctionIndex on the
// supplied CModuleRegistry. The optional onFunction callback fires for
// every indexed function node with the file's project-relative prefix
// — C++ uses it to compose namespace- and class-qualified FQNs without
// re-walking the graph.
func indexFilesAndFunctions(
	codeGraph *graph.CodeGraph,
	projectPath, language string,
	registry *core.CModuleRegistry,
	onFunction func(node *graph.Node, prefix string),
) {
	for _, node := range codeGraph.Nodes {
		if node == nil || node.Language != language || node.File == "" {
			continue
		}
		prefix, ok := ensureFilePrefix(registry, node.File, projectPath)
		if !ok {
			continue
		}
		if !isFunctionLikeNode(node) || node.Name == "" {
			continue
		}

		// Free functions go straight into FunctionIndex under the bare
		// "prefix::name" form. C++ methods (Type=="method_declaration")
		// are NOT recorded here because they are reachable only via
		// NamespaceIndex; mixing them in would mask overload resolution
		// downstream.
		if node.Type == cNodeFunctionDefinition {
			fqn := joinFQN(prefix, node.Name)
			registry.FunctionIndex[node.Name] = appendUnique(registry.FunctionIndex[node.Name], fqn)
		}
		if onFunction != nil {
			onFunction(node, prefix)
		}
	}
}

// ensureFilePrefix records node.File in registry.FileToPrefix on first
// sight and returns the resulting prefix. Files that fall outside
// projectPath (relative path begins with `..`) are skipped: their
// (false, false) return tells the caller to drop the node. Already-seen
// files return their cached prefix.
func ensureFilePrefix(registry *core.CModuleRegistry, file, projectPath string) (string, bool) {
	if prefix, seen := registry.FileToPrefix[file]; seen {
		return prefix, true
	}
	rel, ok := relativeProjectPath(projectPath, file)
	if !ok {
		return "", false
	}
	registry.FileToPrefix[file] = rel
	return rel, true
}

// isFunctionLikeNode is true for graph node types that contribute a
// function-shaped entry to the registry. Free functions and method
// declarations are both function-like; class declarations are not.
func isFunctionLikeNode(node *graph.Node) bool {
	switch node.Type {
	case cNodeFunctionDefinition, cNodeMethodDeclaration:
		return true
	}
	return false
}

// isProjectInclude reports whether node is a quoted `#include "..."`
// for the given language. System includes (`#include <...>`) and nodes
// of other types are excluded.
func isProjectInclude(node *graph.Node, language string) bool {
	if node == nil || node.Language != language || node.Type != cNodeIncludeStatement {
		return false
	}
	if node.Name == "" || node.File == "" {
		return false
	}
	if v, ok := node.Metadata[metaSystemInclude].(bool); ok && v {
		return false
	}
	return true
}

// resolveLocalInclude searches the conventional project directories for
// a header named headerName and returns the first absolute path that
// exists, or "" when none match. Search order is documented on
// BuildCIncludeMap.
func resolveLocalInclude(projectRoot, sourceFile, headerName string) string {
	if headerName == "" {
		return ""
	}
	searchDirs := []string{
		filepath.Dir(sourceFile),
		filepath.Join(projectRoot, includeDirInclude),
		filepath.Join(projectRoot, includeDirSrc),
		projectRoot,
	}
	for _, dir := range searchDirs {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, headerName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

// cppClassEntry caches the byte range of a single C++ class_declaration
// node so methods declared inside it can be associated by containment.
// Caching avoids quadratic re-scans of the graph during method indexing.
type cppClassEntry struct {
	name        string
	packageName string
	startByte   uint32
	endByte     uint32
	fqn         string
}

// collectCppClasses walks the graph for class_declaration nodes,
// records each in registry.ClassIndex, and returns a per-file index
// used to associate methods with their enclosing class.
//
// Anonymous classes (Name=="") and classes whose source location is
// missing are skipped — neither can provide a meaningful FQN component.
func collectCppClasses(
	codeGraph *graph.CodeGraph,
	projectPath string,
	registry *core.CppModuleRegistry,
) map[string][]cppClassEntry {
	classes := make(map[string][]cppClassEntry)
	for _, node := range codeGraph.Nodes {
		if node == nil || node.Language != languageCpp || node.Type != cNodeClassDeclaration {
			continue
		}
		if node.Name == "" || node.File == "" || node.SourceLocation == nil {
			continue
		}
		prefix, ok := ensureFilePrefix(&registry.CModuleRegistry, node.File, projectPath)
		if !ok {
			continue
		}
		fqn := joinFQN(prefix, joinScope(node.PackageName, node.Name))
		registry.ClassIndex[node.Name] = appendUnique(registry.ClassIndex[node.Name], fqn)
		classes[node.File] = append(classes[node.File], cppClassEntry{
			name:        node.Name,
			packageName: node.PackageName,
			startByte:   node.SourceLocation.StartByte,
			endByte:     node.SourceLocation.EndByte,
			fqn:         fqn,
		})
	}
	return classes
}

// indexCppFreeFunction records a free C++ function in NamespaceIndex when
// it carries a namespace, OR associates it with an enclosing class when
// the function is defined inside one (out-of-line `Class::method` bodies
// land here as function_definition).
//
// The prefix argument is the file's project-relative path; the function
// composes the qualified FQN as "prefix::ns::[Class::]name", omitting
// empty scope components.
func indexCppFreeFunction(
	node *graph.Node,
	classes map[string][]cppClassEntry,
	prefix string,
	registry *core.CppModuleRegistry,
) {
	if cls := enclosingClass(node, classes); cls != nil {
		key := joinScope(cls.packageName, cls.name, node.Name)
		fqn := joinFQN(prefix, key)
		registry.NamespaceIndex[key] = fqn
		return
	}
	if node.PackageName == "" {
		return
	}
	key := joinScope(node.PackageName, node.Name)
	registry.NamespaceIndex[key] = joinFQN(prefix, key)
}

// indexCppMethod records a method_declaration in NamespaceIndex under
// its qualified key. Methods are always emitted inside a class context
// (either inline-in-class or via the destructor handler), so we look up
// the enclosing class by byte-range containment and fall back to
// PackageName-only qualification if no class is found (defensive — this
// should not happen for well-formed input).
func indexCppMethod(
	node *graph.Node,
	classes map[string][]cppClassEntry,
	prefix string,
	registry *core.CppModuleRegistry,
) {
	if cls := enclosingClass(node, classes); cls != nil {
		key := joinScope(cls.packageName, cls.name, node.Name)
		registry.NamespaceIndex[key] = joinFQN(prefix, key)
		return
	}
	if node.PackageName == "" {
		return
	}
	key := joinScope(node.PackageName, node.Name)
	registry.NamespaceIndex[key] = joinFQN(prefix, key)
}

// enclosingClass returns the cppClassEntry whose byte range encloses
// node.SourceLocation.StartByte within the same file, picking the
// innermost (smallest range) class when classes nest. Returns nil when
// no class contains the node — used by free-function indexing.
func enclosingClass(node *graph.Node, classes map[string][]cppClassEntry) *cppClassEntry {
	if node == nil || node.SourceLocation == nil {
		return nil
	}
	candidates := classes[node.File]
	if len(candidates) == 0 {
		return nil
	}
	pos := node.SourceLocation.StartByte
	var best *cppClassEntry
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

// joinFQN composes a top-level FQN from a file prefix and a tail scope.
// The tail is expected to already be `::`-joined when it represents
// nested scopes (e.g. "mylib::Socket::connect"). Both arguments are
// non-empty in every call site within this package — see
// indexFilesAndFunctions and indexCppMethod.
func joinFQN(prefix, tail string) string {
	return prefix + fqnSeparator + tail
}

// joinScope joins non-empty scope components with `::`. Empty entries
// are dropped so callers can pass node.PackageName unconditionally
// without worrying about double-separators when there is no namespace.
func joinScope(parts ...string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, fqnSeparator)
}

// relativeProjectPath returns filepath.Rel(projectPath, file) with
// outside-project paths (`../...`) treated as absent. The boolean
// signals whether the result is project-relative; callers that get
// false should drop the node.
//
// On Linux the comparison is case-sensitive, matching the underlying
// filesystem; on Windows/macOS the OS-level case-insensitivity of
// filepath.Rel still applies.
func relativeProjectPath(projectPath, file string) (string, bool) {
	rel, err := filepath.Rel(projectPath, file)
	if err != nil {
		return "", false
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// appendUnique adds value to slice unless it is already present. O(n)
// per call — acceptable for the small per-key slice sizes we expect
// (most function names map to a single FQN).
func appendUnique(slice []string, value string) []string {
	if slices.Contains(slice, value) {
		return slice
	}
	return append(slice, value)
}
