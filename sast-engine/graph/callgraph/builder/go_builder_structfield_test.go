package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildStructFieldIndex_Basic verifies that struct fields are indexed
// with the correct "TypeFQN.FieldName" → resolved field type FQN mapping.
func TestBuildStructFieldIndex_Basic(t *testing.T) {
	cg := graph.NewCodeGraph()
	// Simulate: type Store struct { db *sql.DB }
	cg.Nodes["store_node"] = &graph.Node{
		ID:        "store_node",
		Type:      "struct_definition",
		Name:      "Store",
		Interface: []string{"db: *sql.DB"},
		File:      "/project/store.go",
		Language:  "go",
	}

	registry := core.NewGoModuleRegistry()
	registry.DirToImport["/project"] = "myapp"

	importMaps := map[string]*core.GoImportMap{
		"/project/store.go": {Imports: map[string]string{"sql": "database/sql"}},
	}

	idx := buildStructFieldIndex(cg, registry, importMaps)

	typeFQN, ok := idx["myapp.Store.db"]
	assert.True(t, ok, "expected entry for myapp.Store.db")
	assert.Equal(t, "database/sql.DB", typeFQN)
}

// TestBuildStructFieldIndex_SamePackageField verifies unqualified field types
// (same package) are qualified with the struct's package path.
func TestBuildStructFieldIndex_SamePackageField(t *testing.T) {
	cg := graph.NewCodeGraph()
	cg.Nodes["model_node"] = &graph.Node{
		ID:        "model_node",
		Type:      "struct_definition",
		Name:      "Attention",
		Interface: []string{"KNorm: *Linear"},
		File:      "/project/model.go",
		Language:  "go",
	}

	registry := core.NewGoModuleRegistry()
	registry.DirToImport["/project"] = "myapp/model"

	importMaps := map[string]*core.GoImportMap{
		"/project/model.go": {Imports: map[string]string{}},
	}

	idx := buildStructFieldIndex(cg, registry, importMaps)

	typeFQN, ok := idx["myapp/model.Attention.KNorm"]
	assert.True(t, ok, "expected entry for KNorm field")
	assert.Equal(t, "myapp/model.Linear", typeFQN)
}

// TestBuildStructFieldIndex_SkipsNonStruct verifies that non-struct nodes
// (function_declaration, module_variable, etc.) are ignored.
func TestBuildStructFieldIndex_SkipsNonStruct(t *testing.T) {
	cg := graph.NewCodeGraph()
	cg.Nodes["fn_node"] = &graph.Node{
		ID:       "fn_node",
		Type:     "function_declaration",
		Name:     "handler",
		File:     "/project/main.go",
		Language: "go",
	}

	registry := core.NewGoModuleRegistry()
	registry.DirToImport["/project"] = "myapp"
	importMaps := map[string]*core.GoImportMap{"/project/main.go": {Imports: map[string]string{}}}

	idx := buildStructFieldIndex(cg, registry, importMaps)
	assert.Empty(t, idx)
}

// TestBuildStructFieldIndex_EmbeddedFieldSkipped verifies that embedded type
// entries (no ": " separator) are skipped without panicking.
func TestBuildStructFieldIndex_EmbeddedFieldSkipped(t *testing.T) {
	cg := graph.NewCodeGraph()
	cg.Nodes["base_node"] = &graph.Node{
		ID:        "base_node",
		Type:      "struct_definition",
		Name:      "Handler",
		Interface: []string{"http.Handler", "name: string"}, // embedded + named
		File:      "/project/main.go",
		Language:  "go",
	}

	registry := core.NewGoModuleRegistry()
	registry.DirToImport["/project"] = "myapp"
	importMaps := map[string]*core.GoImportMap{"/project/main.go": {Imports: map[string]string{}}}

	idx := buildStructFieldIndex(cg, registry, importMaps)
	// Only the named field should appear; embedded type skipped.
	_, hasEmbedded := idx["myapp.Handler.http.Handler"]
	assert.False(t, hasEmbedded, "embedded type should not be indexed")
	_, hasNamed := idx["myapp.Handler.name"]
	assert.True(t, hasNamed, "named field should be indexed")
}

// TestSource4_StructFieldResolution_Integration verifies end-to-end resolution
// of a.Field.Method() where 'a' is a receiver variable and Field is a struct field.
func TestSource4_StructFieldResolution_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"),
		[]byte("module testapp\n\ngo 1.21\n"), 0644))

	// Attention has a field KNorm of type *Linear.
	// Linear has a method Forward defined in user code.
	// Attention.Forward calls a.KNorm.Forward() — chained field method call.
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "model.go"), []byte(`package main

type Linear struct{}

func (l *Linear) Forward(x string) string { return x }

type Attention struct {
	KNorm *Linear
}

func (a *Attention) Forward(x string) string {
	return a.KNorm.Forward(x)
}
`), 0644))

	codeGraph := graph.Initialize(tmpDir, nil)
	goRegistry, err := resolution.BuildGoModuleRegistry(tmpDir)
	require.NoError(t, err)

	goTypeEngine := resolution.NewGoTypeInferenceEngine(goRegistry)
	callGraph, err := BuildGoCallGraph(codeGraph, goRegistry, goTypeEngine, nil, nil)
	require.NoError(t, err)

	// Verify that Attention.Forward's call to a.KNorm.Forward is resolved.
	sites := callGraph.CallSites["testapp.Attention.Forward"]
	var found bool
	for _, cs := range sites {
		if cs.Target == "Forward" && cs.Resolved && cs.TargetFQN == "testapp.Linear.Forward" {
			found = true
			break
		}
	}
	assert.True(t, found, "a.KNorm.Forward() should resolve to testapp.Linear.Forward via Source 4")

	// Verify the struct field index was populated.
	assert.Contains(t, callGraph.GoStructFieldIndex, "testapp.Attention.KNorm",
		"GoStructFieldIndex should contain Attention.KNorm entry")
}

// TestSource4_ReceiverField_Database verifies s.db.QueryRow() where s is a
// receiver of type *Store and db is a field of type *sql.DB.
func TestSource4_ReceiverField_Database(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "go.mod"),
		[]byte("module testapp\n\ngo 1.21\n"), 0644))

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "store.go"), []byte(`package main

import "database/sql"

type Store struct {
	db *sql.DB
}

func (s *Store) GetUser(id int) {
	s.db.QueryRow("SELECT * FROM users WHERE id = ?", id)
}
`), 0644))

	codeGraph := graph.Initialize(tmpDir, nil)
	goRegistry, err := resolution.BuildGoModuleRegistry(tmpDir)
	require.NoError(t, err)

	goTypeEngine := resolution.NewGoTypeInferenceEngine(goRegistry)
	callGraph, err := BuildGoCallGraph(codeGraph, goRegistry, goTypeEngine, nil, nil)
	require.NoError(t, err)

	// The index must have the db field mapped.
	assert.Equal(t, "database/sql.DB",
		callGraph.GoStructFieldIndex["testapp.Store.db"],
		"Store.db field should map to database/sql.DB")

	// s.db.QueryRow should be resolved.
	sites := callGraph.CallSites["testapp.Store.GetUser"]
	var found bool
	for _, cs := range sites {
		if cs.Target == "QueryRow" && cs.Resolved {
			found = true
			assert.Equal(t, "database/sql.DB.QueryRow", cs.TargetFQN)
			break
		}
	}
	assert.True(t, found, "s.db.QueryRow() should be resolved via Source 4 + stdlib check")
}
