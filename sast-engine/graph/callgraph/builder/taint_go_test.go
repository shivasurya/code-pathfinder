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

func TestGenerateGoTaintSummaries_BasicFunction(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

import "fmt"

func hello(name string) {
	msg := "Hello, " + name
	fmt.Println(msg)
}
`), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)
	callGraph := core.NewCallGraph()

	// Find and index the Go function node
	for _, node := range codeGraph.Nodes {
		if node.Type == "function_declaration" && node.Name == "hello" && node.Language == "go" {
			callGraph.Functions["testapp.hello"] = node
			break
		}
	}
	require.NotEmpty(t, callGraph.Functions, "Should have indexed the hello function")

	GenerateGoTaintSummaries(callGraph, codeGraph, nil, nil, nil)

	// Verify statements were extracted
	stmts, ok := callGraph.Statements["testapp.hello"]
	require.True(t, ok, "Statements should be populated for testapp.hello")
	require.NotEmpty(t, stmts, "Should have extracted at least one statement")

	// Verify: msg := "Hello, " + name → Def:"msg", Uses contains "name"
	foundMsgAssign := false
	for _, stmt := range stmts {
		if stmt.Def == "msg" {
			foundMsgAssign = true
			assert.Equal(t, core.StatementTypeAssignment, stmt.Type)
			assert.Contains(t, stmt.Uses, "name")
		}
	}
	assert.True(t, foundMsgAssign, "Should find 'msg' assignment statement")

	// Verify summary was generated
	summary, ok := callGraph.Summaries["testapp.hello"]
	require.True(t, ok, "TaintSummary should be populated")
	assert.NotNil(t, summary)
}

func TestGenerateGoTaintSummaries_SkipsPythonFunctions(t *testing.T) {
	callGraph := core.NewCallGraph()

	// Add a Python function — should be skipped
	callGraph.Functions["myapp.handler"] = &graph.Node{
		ID:       "py1",
		Type:     "function_definition",
		Name:     "handler",
		Language: "python",
		File:     "/nonexistent/handler.py",
	}

	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
		Edges: make([]*graph.Edge, 0),
	}

	// Should not panic, should not populate statements for Python function
	GenerateGoTaintSummaries(callGraph, codeGraph, nil, nil, nil)

	assert.Empty(t, callGraph.Statements, "Should not extract statements for Python functions")
}

func TestGenerateGoTaintSummaries_EmptyCallGraph(t *testing.T) {
	callGraph := core.NewCallGraph()
	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
		Edges: make([]*graph.Edge, 0),
	}

	// Should not panic with empty inputs
	GenerateGoTaintSummaries(callGraph, codeGraph, nil, nil, nil)

	assert.Empty(t, callGraph.Statements)
	assert.Empty(t, callGraph.Summaries)
}

func TestGenerateGoTaintSummaries_MultipleStatements(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

func handler() {
	query := getInput()
	sql := "SELECT * FROM " + query
	result := execute(sql)
	_ = result
}
`), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)
	callGraph := core.NewCallGraph()

	for _, node := range codeGraph.Nodes {
		if node.Type == "function_declaration" && node.Name == "handler" && node.Language == "go" {
			callGraph.Functions["testapp.handler"] = node
			break
		}
	}
	require.NotEmpty(t, callGraph.Functions)

	GenerateGoTaintSummaries(callGraph, codeGraph, nil, nil, nil)

	stmts := callGraph.Statements["testapp.handler"]
	require.NotEmpty(t, stmts)

	// Verify def-use chain is extractable
	defs := map[string]bool{}
	for _, stmt := range stmts {
		if stmt.Def != "" {
			defs[stmt.Def] = true
		}
	}
	assert.True(t, defs["query"], "Should extract query assignment")
	assert.True(t, defs["sql"], "Should extract sql assignment")
	assert.True(t, defs["result"], "Should extract result assignment")
}

func TestGenerateGoTaintSummaries_MethodDeclaration(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

type Server struct{}

func (s *Server) Handle(input string) string {
	result := process(input)
	return result
}
`), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)
	callGraph := core.NewCallGraph()

	for _, node := range codeGraph.Nodes {
		if node.Type == "method" && node.Name == "Handle" && node.Language == "go" {
			callGraph.Functions["testapp.Server.Handle"] = node
			break
		}
	}
	require.NotEmpty(t, callGraph.Functions, "Should index the method")

	GenerateGoTaintSummaries(callGraph, codeGraph, nil, nil, nil)

	stmts := callGraph.Statements["testapp.Server.Handle"]
	require.NotEmpty(t, stmts, "Should extract statements from method")

	// result := process(input) → Def:"result", CallTarget:"process"
	foundResult := false
	for _, stmt := range stmts {
		if stmt.Def == "result" && stmt.CallTarget == "process" {
			foundResult = true
		}
	}
	assert.True(t, foundResult, "Should extract result := process(input)")
}

func TestGenerateGoTaintSummaries_Integration_BuildGoCallGraph(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

func process(input string) string {
	result := transform(input)
	return result
}

func transform(data string) string {
	return data
}
`), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)

	goRegistry, err := resolution.BuildGoModuleRegistry(tmpDir)
	require.NoError(t, err)

	// BuildGoCallGraph should call GenerateGoTaintSummaries internally
	callGraph, err := BuildGoCallGraph(codeGraph, goRegistry, nil)
	require.NoError(t, err)

	// After BuildGoCallGraph, Statements should be populated for Go functions
	foundStatements := false
	for funcFQN, stmts := range callGraph.Statements {
		if len(stmts) > 0 {
			foundStatements = true
			t.Logf("Function %s has %d statements", funcFQN, len(stmts))
		}
	}
	assert.True(t, foundStatements, "BuildGoCallGraph should populate Statements via GenerateGoTaintSummaries")

	// Summaries should also be populated
	foundSummary := false
	for funcFQN := range callGraph.Summaries {
		foundSummary = true
		t.Logf("Function %s has taint summary", funcFQN)
	}
	assert.True(t, foundSummary, "BuildGoCallGraph should populate Summaries")
}

func TestGenerateGoTaintSummaries_PackageLevelVars(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	// Package-level variable with a source call
	err = os.WriteFile(filepath.Join(tmpDir, "config.go"), []byte(`package main

import "os"

var DBHost = os.Getenv("DB_HOST")
var SafeVal = "hardcoded"
`), 0644)
	require.NoError(t, err)

	// Need a function so the file gets parsed into the file cache
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

func main() {}
`), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)
	callGraph := core.NewCallGraph()

	// Index the main function so config.go gets parsed via file cache
	for _, node := range codeGraph.Nodes {
		if node.Language == "go" && node.Type == "function_declaration" {
			callGraph.Functions["testapp."+node.Name] = node
		}
	}

	GenerateGoTaintSummaries(callGraph, codeGraph, nil, nil, nil)

	// Check for synthetic init scope with package-level var statements
	var initFQN string
	for fqn := range callGraph.Statements {
		for _, stmt := range callGraph.Statements[fqn] {
			if stmt.Def == "DBHost" && stmt.CallTarget == "Getenv" {
				initFQN = fqn
				break
			}
		}
	}

	require.NotEmpty(t, initFQN, "Should create synthetic scope with DBHost = os.Getenv(...)")

	// Verify the synthetic scope has both vars
	stmts := callGraph.Statements[initFQN]
	defs := map[string]bool{}
	for _, stmt := range stmts {
		if stmt.Def != "" {
			defs[stmt.Def] = true
		}
	}
	assert.True(t, defs["DBHost"], "Should have DBHost definition")
	assert.True(t, defs["SafeVal"], "Should have SafeVal definition")

	// Verify summary exists for synthetic scope
	assert.Contains(t, callGraph.Summaries, initFQN, "Should have taint summary for init scope")
}
