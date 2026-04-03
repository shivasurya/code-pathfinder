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

func TestEnrichGoStatements_AttributeAccess(t *testing.T) {
	funcNode := &graph.Node{
		Name:                 "handler",
		Language:             "go",
		MethodArgumentsValue: []string{"w", "r"},
		MethodArgumentsType:  []string{"w: http.ResponseWriter", "r: *http.Request"},
		File:                 "handler.go",
	}

	stmts := []*core.Statement{
		{
			Type:            core.StatementTypeAssignment,
			Def:             "path",
			Uses:            []string{"r"},
			AttributeAccess: "r.URL.Path",
			LineNumber:      3,
		},
	}

	importMap := core.NewGoImportMap("handler.go")
	importMap.AddImport("http", "net/http")
	importMaps := map[string]*core.GoImportMap{"handler.go": importMap}

	enrichGoStatements("testapp.handler", funcNode, stmts, nil, nil, importMaps)

	assert.Equal(t, "net/http.Request.URL.Path", stmts[0].AttributeAccess)
}

func TestEnrichGoStatements_CallChain(t *testing.T) {
	funcNode := &graph.Node{
		Name:                 "handler",
		Language:             "go",
		MethodArgumentsValue: []string{"w", "r"},
		MethodArgumentsType:  []string{"w: http.ResponseWriter", "r: *http.Request"},
		File:                 "handler.go",
	}

	stmts := []*core.Statement{
		{
			Type:       core.StatementTypeAssignment,
			Def:        "q",
			Uses:       []string{"r"},
			CallTarget: "FormValue",
			CallChain:  "r.FormValue",
			LineNumber: 3,
		},
	}

	importMap := core.NewGoImportMap("handler.go")
	importMap.AddImport("http", "net/http")
	importMaps := map[string]*core.GoImportMap{"handler.go": importMap}

	enrichGoStatements("testapp.handler", funcNode, stmts, nil, nil, importMaps)

	assert.Equal(t, "net/http.Request.FormValue", stmts[0].CallChain)
}

func TestEnrichGoStatements_LocalVariable(t *testing.T) {
	funcNode := &graph.Node{
		Name:     "handler",
		Language: "go",
		File:     "handler.go",
	}

	stmts := []*core.Statement{
		{
			Type:       core.StatementTypeAssignment,
			Def:        "rows",
			Uses:       []string{"sql"},
			CallTarget: "Query",
			CallChain:  "db.Query",
			LineNumber: 5,
		},
	}

	typeEngine := resolution.NewGoTypeInferenceEngine(nil)
	scope := resolution.NewGoFunctionScope("testapp.handler")
	scope.AddVariable(&resolution.GoVariableBinding{
		VarName: "db",
		Type:    &core.TypeInfo{TypeFQN: "*database/sql.DB", Confidence: 0.9},
	})
	typeEngine.AddScope(scope)

	enrichGoStatements("testapp.handler", funcNode, stmts, typeEngine, nil, nil)

	assert.Equal(t, "database/sql.DB.Query", stmts[0].CallChain)
}

func TestEnrichGoStatements_DifferentVariableNames(t *testing.T) {
	variableNames := []string{"r", "req", "request"}

	for _, varName := range variableNames {
		t.Run("var_"+varName, func(t *testing.T) {
			funcNode := &graph.Node{
				Name:                 "handler",
				Language:             "go",
				MethodArgumentsValue: []string{"w", varName},
				MethodArgumentsType:  []string{"w: http.ResponseWriter", varName + ": *http.Request"},
				File:                 "handler.go",
			}

			stmts := []*core.Statement{
				{
					Type:            core.StatementTypeAssignment,
					Def:             "path",
					AttributeAccess: varName + ".URL.Path",
					LineNumber:      3,
				},
			}

			importMap := core.NewGoImportMap("handler.go")
			importMap.AddImport("http", "net/http")
			importMaps := map[string]*core.GoImportMap{"handler.go": importMap}

			enrichGoStatements("testapp.handler", funcNode, stmts, nil, nil, importMaps)

			assert.Equal(t, "net/http.Request.URL.Path", stmts[0].AttributeAccess,
				"Should resolve %s.URL.Path → net/http.Request.URL.Path", varName)
		})
	}
}

func TestEnrichGoStatements_UnknownVariable(t *testing.T) {
	funcNode := &graph.Node{Name: "handler", Language: "go", File: "handler.go"}

	stmts := []*core.Statement{
		{
			Type:            core.StatementTypeAssignment,
			Def:             "path",
			AttributeAccess: "unknown.Field",
			LineNumber:      3,
		},
	}

	enrichGoStatements("testapp.handler", funcNode, stmts, nil, nil, nil)

	assert.Equal(t, "unknown.Field", stmts[0].AttributeAccess)
}

func TestEnrichGoStatements_NoCallChainDot(t *testing.T) {
	funcNode := &graph.Node{Name: "handler", Language: "go", File: "handler.go"}

	stmts := []*core.Statement{
		{
			Type:       core.StatementTypeCall,
			CallTarget: "println",
			CallChain:  "println",
			LineNumber: 3,
		},
	}

	enrichGoStatements("testapp.handler", funcNode, stmts, nil, nil, nil)

	assert.Equal(t, "println", stmts[0].CallChain)
}

func TestEnrichGoStatements_Integration(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

import "net/http"

func handler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	query := r.FormValue("q")
	_ = path
	_ = query
}
`), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)
	goRegistry, err := resolution.BuildGoModuleRegistry(tmpDir)
	require.NoError(t, err)
	goTypeEngine := resolution.NewGoTypeInferenceEngine(goRegistry)

	callGraph, err := BuildGoCallGraph(codeGraph, goRegistry, goTypeEngine)
	require.NoError(t, err)

	for funcFQN, stmts := range callGraph.Statements {
		for _, stmt := range stmts {
			if stmt.AttributeAccess != "" {
				t.Logf("%s: AttributeAccess = %q", funcFQN, stmt.AttributeAccess)
				// Positive assertion — must be type-qualified
				assert.Equal(t, "net/http.Request.URL.Path", stmt.AttributeAccess,
					"AttributeAccess should be resolved to type FQN")
			}
			if stmt.CallChain != "" && stmt.CallTarget == "FormValue" {
				t.Logf("%s: CallChain = %q", funcFQN, stmt.CallChain)
				assert.Equal(t, "net/http.Request.FormValue", stmt.CallChain,
					"CallChain should be resolved to type FQN")
			}
		}
	}
}
