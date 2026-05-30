package builder

import (
	"os"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// pythonNodeKind maps a tree-sitter Python definition node type to the kind
// stored in fqn_index. The empty string means "not a Python definition we index
// here" (PR-02 indexes callables and classes; module-level variables and import
// aliases are a follow-up).
func pythonNodeKind(nodeType string) string {
	switch nodeType {
	case "function_definition":
		return "function"
	case "method", "constructor", "property", "special_method":
		return "method"
	case "class_definition", "dataclass":
		return "class"
	default:
		return ""
	}
}

// isPythonFile reports whether a path is a Python source file.
func isPythonFile(path string) bool {
	return strings.HasSuffix(path, ".py")
}

// parentFqn returns the enclosing module/class FQN by dropping the last
// dot-segment. "app.models.User.save" -> "app.models.User"; "app.models.User"
// -> "app.models"; a name with no dot -> "".
func parentFqn(fqn string) string {
	if i := strings.LastIndex(fqn, "."); i >= 0 {
		return fqn[:i]
	}
	return ""
}

// pythonSignature renders a callable's signature from its typed parameters and
// return annotation, e.g. "(user: User, force: bool) -> None". Classes and
// nodes without parameters or a return type yield "".
func pythonSignature(node *graph.Node) string {
	if len(node.MethodArgumentsType) == 0 && node.ReturnType == "" {
		return ""
	}
	sig := "(" + strings.Join(node.MethodArgumentsType, ", ") + ")"
	if node.ReturnType != "" {
		sig += " -> " + node.ReturnType
	}
	return sig
}

// collectPythonFqnEntries enumerates every Python function, method, and class
// the build discovered and returns them as fqn_index rows plus the set of
// Python source files to stamp in indexed_files.
//
// Every entry is source="project": these are definitions written in the
// scanned project's own files. Stdlib/thirdparty symbols are resolution
// targets, not project definitions, and are out of scope here.
func collectPythonFqnEntries(
	callGraph *core.CallGraph,
	codeGraph *graph.CodeGraph,
	registry *core.ModuleRegistry,
) ([]FqnEntry, []IndexedFile) {
	entries := make([]FqnEntry, 0, len(callGraph.Functions))
	seen := make(map[string]bool, len(callGraph.Functions))

	add := func(fqn string, node *graph.Node, kind, signature string) {
		if seen[fqn] {
			return
		}
		seen[fqn] = true
		entries = append(entries, FqnEntry{
			Fqn:       fqn,
			Language:  "python",
			Kind:      kind,
			Source:    "project",
			File:      nullStr(node.File),
			StartLine: nullInt(int(node.LineNumber)),
			ParentFqn: nullStr(parentFqn(fqn)),
			Signature: nullStr(signature),
		})
	}

	// Functions and methods are keyed by FQN in the call graph.
	for fqn, node := range callGraph.Functions {
		if node == nil || !isPythonFile(node.File) {
			continue
		}
		kind := pythonNodeKind(node.Type)
		if kind == "" {
			continue
		}
		add(fqn, node, kind, pythonSignature(node))
	}

	// Classes are not call-graph functions; enumerate them from the code graph.
	for _, node := range codeGraph.Nodes {
		if node == nil || !isPythonFile(node.File) || pythonNodeKind(node.Type) != "class" {
			continue
		}
		modulePath, ok := registry.FileToModule[node.File]
		if !ok {
			continue
		}
		add(modulePath+"."+node.Name, node, "class", "")
	}

	return entries, collectPythonIndexedFiles(registry)
}

// collectPythonIndexedFiles returns every Python source file the build knows
// about, with its current mtime, so each is stamped in indexed_files. Files
// that cannot be stat'd (race with a delete) are skipped: a missing stamp just
// makes the file look stale to a later query, which is the safe default.
func collectPythonIndexedFiles(registry *core.ModuleRegistry) []IndexedFile {
	files := make([]IndexedFile, 0, len(registry.FileToModule))
	for file := range registry.FileToModule {
		if !isPythonFile(file) {
			continue
		}
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		files = append(files, IndexedFile{Path: file, ModTimeUnix: info.ModTime().Unix()})
	}
	return files
}
