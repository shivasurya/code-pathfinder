package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPythonNodeKind(t *testing.T) {
	cases := map[string]string{
		"function_definition": "function",
		"method":              "method",
		"constructor":         "method",
		"property":            "method",
		"special_method":      "method",
		"class_definition":    "class",
		"dataclass":           "class",
		"method_declaration":  "", // Java; not indexed here
		"call":                "",
	}
	for nodeType, want := range cases {
		assert.Equal(t, want, pythonNodeKind(nodeType), "kind for %q", nodeType)
	}
}

func TestIsPythonFile(t *testing.T) {
	assert.True(t, isPythonFile("/a/b/models.py"))
	assert.False(t, isPythonFile("/a/b/main.go"))
	assert.False(t, isPythonFile("/a/b/py"))
}

func TestParentFqn(t *testing.T) {
	assert.Equal(t, "app.models.User", parentFqn("app.models.User.save"))
	assert.Equal(t, "app.models", parentFqn("app.models.User"))
	assert.Equal(t, "app", parentFqn("app.helper"))
	assert.Equal(t, "", parentFqn("toplevel"))
}

func TestPythonSignature(t *testing.T) {
	assert.Equal(t, "(user: User, force: bool) -> None",
		pythonSignature(&graph.Node{MethodArgumentsType: []string{"user: User", "force: bool"}, ReturnType: "None"}))
	assert.Equal(t, "(x: int)",
		pythonSignature(&graph.Node{MethodArgumentsType: []string{"x: int"}}))
	assert.Equal(t, "() -> str",
		pythonSignature(&graph.Node{ReturnType: "str"}))
	assert.Equal(t, "", pythonSignature(&graph.Node{}), "no args and no return type yields no signature")
}

func TestCollectPythonFqnEntries(t *testing.T) {
	pyFile := "/proj/app/models.py"
	goFile := "/proj/main.go"

	fn := &graph.Node{Type: "function_definition", Name: "sanitize", File: pyFile, LineNumber: 10, MethodArgumentsType: []string{"s: str"}, ReturnType: "str"}
	method := &graph.Node{Type: "method", Name: "save", File: pyFile, LineNumber: 20}
	goFn := &graph.Node{Type: "function_definition", Name: "Main", File: goFile, LineNumber: 3}

	cg := core.NewCallGraph()
	cg.Functions["app.models.sanitize"] = fn
	cg.Functions["app.models.User.save"] = method
	cg.Functions["main.Main"] = goFn // non-python: must be skipped

	classNode := &graph.Node{Type: "class_definition", Name: "User", File: pyFile, LineNumber: 15}
	codeGraph := &graph.CodeGraph{Nodes: map[string]*graph.Node{"c1": classNode}}

	registry := core.NewModuleRegistry()
	registry.FileToModule[pyFile] = "app.models"
	registry.FileToModule[goFile] = "main"

	entries, files := collectPythonFqnEntries(cg, codeGraph, registry)

	byFqn := make(map[string]FqnEntry, len(entries))
	for _, e := range entries {
		byFqn[e.Fqn] = e
		assert.Equal(t, "python", e.Language)
		assert.Equal(t, "project", e.Source)
	}

	require.Contains(t, byFqn, "app.models.sanitize")
	require.Contains(t, byFqn, "app.models.User.save")
	require.Contains(t, byFqn, "app.models.User")
	assert.NotContains(t, byFqn, "main.Main", "Go functions must not be indexed as python")
	assert.Len(t, entries, 3)

	fnEntry := byFqn["app.models.sanitize"]
	assert.Equal(t, "function", fnEntry.Kind)
	assert.Equal(t, "app.models", fnEntry.ParentFqn.String)
	assert.Equal(t, "(s: str) -> str", fnEntry.Signature.String)
	assert.Equal(t, int64(10), fnEntry.StartLine.Int64)
	assert.Equal(t, pyFile, fnEntry.File.String)

	methodEntry := byFqn["app.models.User.save"]
	assert.Equal(t, "method", methodEntry.Kind)
	assert.Equal(t, "app.models.User", methodEntry.ParentFqn.String)
	assert.False(t, methodEntry.Signature.Valid, "no args/return -> NULL signature")

	classEntry := byFqn["app.models.User"]
	assert.Equal(t, "class", classEntry.Kind)
	assert.Equal(t, "app.models", classEntry.ParentFqn.String)

	// Only the python file is stamped for staleness; the .go file is excluded.
	require.Len(t, files, 0, "files with no on-disk presence are skipped by stat; none exist in this unit test")
	_ = files
}

// TestCollectPythonFqnEntries_NilAndUnknownNodes ensures defensive skips:
// a nil node, a node in a non-registered file, and an unindexed node type.
func TestCollectPythonFqnEntries_SkipsNonDefinitions(t *testing.T) {
	pyFile := "/proj/app/x.py"
	cg := core.NewCallGraph()
	cg.Functions["app.x.fn"] = &graph.Node{Type: "function_definition", Name: "fn", File: pyFile, LineNumber: 1}
	cg.Functions["app.x.nilnode"] = nil
	cg.Functions["app.x.var"] = &graph.Node{Type: "assignment", Name: "var", File: pyFile, LineNumber: 2} // not a def

	// A class node whose file is not in the registry must be skipped.
	codeGraph := &graph.CodeGraph{Nodes: map[string]*graph.Node{
		"c1": {Type: "class_definition", Name: "Orphan", File: "/unregistered.py", LineNumber: 3},
		"c2": nil,
	}}

	registry := core.NewModuleRegistry()
	registry.FileToModule[pyFile] = "app.x"

	entries, _ := collectPythonFqnEntries(cg, codeGraph, registry)
	require.Len(t, entries, 1)
	assert.Equal(t, "app.x.fn", entries[0].Fqn)
}

// TestCollectPythonFqnEntries_DedupesRepeatedClassNodes verifies the seen-set:
// two class nodes resolving to the same FQN produce a single entry.
func TestCollectPythonFqnEntries_DedupesRepeatedClassNodes(t *testing.T) {
	pyFile := "/proj/app/d.py"
	codeGraph := &graph.CodeGraph{Nodes: map[string]*graph.Node{
		"c1": {Type: "class_definition", Name: "Dup", File: pyFile, LineNumber: 1},
		"c2": {Type: "class_definition", Name: "Dup", File: pyFile, LineNumber: 1},
	}}
	registry := core.NewModuleRegistry()
	registry.FileToModule[pyFile] = "app.d"

	entries, _ := collectPythonFqnEntries(core.NewCallGraph(), codeGraph, registry)
	require.Len(t, entries, 1, "the same class FQN must be indexed once")
	assert.Equal(t, "app.d.Dup", entries[0].Fqn)
}

// TestCollectPythonIndexedFiles_StampsRealFilesSkipsMissing verifies that an
// on-disk python file is stamped while a registered-but-missing path is skipped.
func TestCollectPythonIndexedFiles_StampsRealFilesSkipsMissing(t *testing.T) {
	dir := t.TempDir()
	present := filepath.Join(dir, "real.py")
	require.NoError(t, os.WriteFile(present, []byte("x = 1\n"), 0o644))

	registry := core.NewModuleRegistry()
	registry.FileToModule[present] = "real"
	registry.FileToModule["/does/not/exist.py"] = "ghost"
	registry.FileToModule[filepath.Join(dir, "main.go")] = "main" // non-python, skipped

	files := collectPythonIndexedFiles(registry)
	require.Len(t, files, 1, "only the existing python file is stamped")
	assert.Equal(t, present, files[0].Path)
	assert.Positive(t, files[0].ModTimeUnix)
}
