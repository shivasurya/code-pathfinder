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

func TestGenerateGoTaintSummaries_NilSourceLocation(t *testing.T) {
	callGraph := core.NewCallGraph()

	// Go function without SourceLocation — should be skipped gracefully
	callGraph.Functions["testapp.broken"] = &graph.Node{
		ID:             "go1",
		Type:           "function_declaration",
		Name:           "broken",
		Language:       "go",
		File:           "/nonexistent/file.go",
		SourceLocation: nil, // nil SourceLocation
	}

	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
		Edges: make([]*graph.Edge, 0),
	}

	// Should not panic — function is skipped due to nil SourceLocation
	GenerateGoTaintSummaries(callGraph, codeGraph, nil, nil, nil)
	assert.Empty(t, callGraph.Statements)
}

func TestGenerateGoTaintSummaries_NonexistentFile(t *testing.T) {
	callGraph := core.NewCallGraph()

	callGraph.Functions["testapp.gone"] = &graph.Node{
		ID:       "go1",
		Type:     "function_declaration",
		Name:     "gone",
		Language: "go",
		File:     "/definitely/not/a/real/file.go",
		SourceLocation: &graph.SourceLocation{
			StartByte: 0,
			EndByte:   100,
		},
	}

	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
		Edges: make([]*graph.Edge, 0),
	}

	// Should not panic — skips with warning
	GenerateGoTaintSummaries(callGraph, codeGraph, nil, nil, nil)
	assert.Empty(t, callGraph.Statements)
}

func TestGenerateGoTaintSummaries_WrongByteRange(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	mainFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(mainFile, []byte(`package main

func foo() {}
`), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)
	callGraph := core.NewCallGraph()

	// Function with wrong byte range — won't match any AST node
	callGraph.Functions["testapp.phantom"] = &graph.Node{
		ID:       "go1",
		Type:     "function_declaration",
		Name:     "phantom",
		Language: "go",
		File:     mainFile,
		SourceLocation: &graph.SourceLocation{
			StartByte: 9999,
			EndByte:   9999,
		},
	}

	GenerateGoTaintSummaries(callGraph, codeGraph, nil, nil, nil)
	assert.Empty(t, callGraph.Statements, "Wrong byte range should skip function")
}

func TestGenerateGoTaintSummaries_FileCacheReuse(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	// Two functions in the same file — should only parse once
	mainFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(mainFile, []byte(`package main

func first() {
	x := 1
	_ = x
}

func second() {
	y := 2
	_ = y
}
`), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)
	callGraph := core.NewCallGraph()

	for _, node := range codeGraph.Nodes {
		if node.Language == "go" && node.Type == "function_declaration" {
			callGraph.Functions["testapp."+node.Name] = node
		}
	}

	GenerateGoTaintSummaries(callGraph, codeGraph, nil, nil, nil)

	// Both functions should have statements
	foundFirst := false
	foundSecond := false
	for fqn := range callGraph.Statements {
		if fqn == "testapp.first" {
			foundFirst = true
		}
		if fqn == "testapp.second" {
			foundSecond = true
		}
	}
	assert.True(t, foundFirst, "Should extract statements from first()")
	assert.True(t, foundSecond, "Should extract statements from second()")
}

func TestGenerateGoTaintSummaries_PackageLevelVars_MultiFile(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	// Two files in same package with vars
	err = os.WriteFile(filepath.Join(tmpDir, "config_a.go"), []byte(`package main

import "os"

var HostA = os.Getenv("HOST_A")
`), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "config_b.go"), []byte(`package main

import "os"

var HostB = os.Getenv("HOST_B")
`), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

func main() {}
`), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)
	callGraph := core.NewCallGraph()

	for _, node := range codeGraph.Nodes {
		if node.Language == "go" && node.Type == "function_declaration" {
			callGraph.Functions["testapp."+node.Name] = node
		}
	}

	GenerateGoTaintSummaries(callGraph, codeGraph, nil, nil, nil)

	// Both HostA and HostB should be in init$vars scope
	var initFQN string
	for fqn := range callGraph.Statements {
		for _, stmt := range callGraph.Statements[fqn] {
			if stmt.Def == "HostA" || stmt.Def == "HostB" {
				initFQN = fqn
				break
			}
		}
	}
	require.NotEmpty(t, initFQN, "Should have synthetic scope")

	defs := map[string]bool{}
	for _, stmt := range callGraph.Statements[initFQN] {
		if stmt.Def != "" {
			defs[stmt.Def] = true
		}
	}
	assert.True(t, defs["HostA"], "Should have HostA from config_a.go")
	assert.True(t, defs["HostB"], "Should have HostB from config_b.go")
}

func TestExtractGoPackageLevelVars_NoVars(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	// File with no package-level vars
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

func main() {
	x := 1
	_ = x
}
`), 0644)
	require.NoError(t, err)

	codeGraph := graph.Initialize(tmpDir, nil)
	callGraph := core.NewCallGraph()

	for _, node := range codeGraph.Nodes {
		if node.Language == "go" && node.Type == "function_declaration" {
			callGraph.Functions["testapp."+node.Name] = node
		}
	}

	GenerateGoTaintSummaries(callGraph, codeGraph, nil, nil, nil)

	// Should not create synthetic init scope when no package vars exist
	for fqn := range callGraph.Statements {
		assert.NotContains(t, fqn, "init$vars", "Should not create init$vars scope when no package vars")
	}
}

func TestGenerateGoTaintSummaries_WithCFG(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0644)
	require.NoError(t, err)

	// Code with control flow — CFG captures taint through if branches
	err = os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

func handler(input string) {
	if len(input) > 0 {
		query := input
		execute(query)
	}
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

	// CFGs should be populated
	_, hasCFG := callGraph.CFGs["testapp.handler"]
	assert.True(t, hasCFG, "CFG should be populated for handler function")

	_, hasBlockStmts := callGraph.CFGBlockStatements["testapp.handler"]
	assert.True(t, hasBlockStmts, "CFGBlockStatements should be populated")

	// Statements key should exist (even if empty — all code is inside if block)
	_, hasStmts := callGraph.Statements["testapp.handler"]
	assert.True(t, hasStmts, "Statements key should exist")
}
