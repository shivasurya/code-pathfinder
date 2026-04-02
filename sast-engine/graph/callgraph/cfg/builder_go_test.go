package cfg

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	golang "github.com/smacker/go-tree-sitter/golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseGoFunction parses Go source and returns the first function_declaration node.
func parseGoFunction(t *testing.T, source string) (*sitter.Node, []byte) {
	t.Helper()
	sourceBytes := []byte(source)
	parser := sitter.NewParser()
	parser.SetLanguage(golang.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, sourceBytes)
	require.NoError(t, err)

	root := tree.RootNode()
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child.Type() == "function_declaration" || child.Type() == "method_declaration" {
			return child, sourceBytes
		}
	}
	t.Fatal("no function_declaration found")
	return nil, nil
}

func TestBuildGoCFG_LinearFunction(t *testing.T) {
	source := `package main

func foo() {
	x := source()
	y := x
	sink(y)
}
`
	funcNode, sourceBytes := parseGoFunction(t, source)

	cfg, blockStmts, err := BuildGoCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have: entry, body block, exit (minimum 3)
	assert.GreaterOrEqual(t, len(cfg.Blocks), 3)

	// Count total statements across all blocks
	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	assert.Equal(t, 3, totalStmts, "should have 3 statements: 2 assignments + 1 call")
}

func TestBuildGoCFG_IfElse(t *testing.T) {
	source := `package main

func foo(x int) {
	a := source()
	if x > 0 {
		b := a
		_ = b
	} else {
		c := safe()
		_ = c
	}
	sink(a)
}
`
	funcNode, sourceBytes := parseGoFunction(t, source)

	cfg, blockStmts, err := BuildGoCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have blocks for: entry, body, if_cond, if_true, if_false, if_merge, exit
	assert.GreaterOrEqual(t, len(cfg.Blocks), 6)

	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	assert.GreaterOrEqual(t, totalStmts, 4)

	// Should have at least 2 paths from entry to exit (true branch, false branch)
	paths := cfg.GetAllPaths()
	assert.GreaterOrEqual(t, len(paths), 2)
}

func TestBuildGoCFG_IfWithInit(t *testing.T) {
	source := `package main

func foo() {
	if err := validate(); err != nil {
		handleError(err)
	}
	proceed()
}
`
	funcNode, sourceBytes := parseGoFunction(t, source)

	cfg, blockStmts, err := BuildGoCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Init statement should be in the condition block
	totalStmts := 0
	foundInit := false
	for _, stmts := range blockStmts {
		for _, stmt := range stmts {
			totalStmts++
			if stmt.Def == "err" && stmt.CallTarget == "validate" {
				foundInit = true
			}
		}
	}
	assert.True(t, foundInit, "Should extract init statement: err := validate()")
}

func TestBuildGoCFG_IfNoElse(t *testing.T) {
	source := `package main

func foo(x int) {
	if x > 0 {
		doSomething()
	}
	after()
}
`
	funcNode, sourceBytes := parseGoFunction(t, source)

	cfg, _, err := BuildGoCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Without else, condition has edge to merge block directly
	assert.GreaterOrEqual(t, len(cfg.Blocks), 5)
}

func TestBuildGoCFG_ForRange(t *testing.T) {
	source := `package main

func foo() {
	items := getItems()
	for i, v := range items {
		process(v)
	}
	done()
}
`
	funcNode, sourceBytes := parseGoFunction(t, source)

	cfg, blockStmts, err := BuildGoCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	foundLoop := false
	for _, block := range cfg.Blocks {
		if block.Type == BlockTypeLoop {
			foundLoop = true
			break
		}
	}
	assert.True(t, foundLoop, "Should have a loop header block")

	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	assert.GreaterOrEqual(t, totalStmts, 3)
}

func TestBuildGoCFG_ForCStyle(t *testing.T) {
	source := `package main

func foo() {
	for i := 0; i < 10; i++ {
		work(i)
	}
}
`
	funcNode, sourceBytes := parseGoFunction(t, source)

	cfg, _, err := BuildGoCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	foundLoop := false
	for _, block := range cfg.Blocks {
		if block.Type == BlockTypeLoop {
			foundLoop = true
		}
	}
	assert.True(t, foundLoop)
}

func TestBuildGoCFG_ForBare(t *testing.T) {
	source := `package main

func foo() {
	for {
		work()
	}
}
`
	funcNode, sourceBytes := parseGoFunction(t, source)

	cfg, _, err := BuildGoCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	foundLoop := false
	for _, block := range cfg.Blocks {
		if block.Type == BlockTypeLoop {
			foundLoop = true
		}
	}
	assert.True(t, foundLoop, "Bare for{} should have loop block")
}

func TestBuildGoCFG_Switch(t *testing.T) {
	source := `package main

func foo(x int) {
	switch x {
	case 1:
		doOne()
	case 2:
		doTwo()
	default:
		doDefault()
	}
	after()
}
`
	funcNode, sourceBytes := parseGoFunction(t, source)

	cfg, blockStmts, err := BuildGoCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	foundSwitch := false
	for _, block := range cfg.Blocks {
		if block.Type == BlockTypeSwitch {
			foundSwitch = true
		}
	}
	assert.True(t, foundSwitch, "Should have a switch block")

	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	// switch value + doOne() + doTwo() + doDefault() + after()
	assert.GreaterOrEqual(t, totalStmts, 4)
}

func TestBuildGoCFG_SwitchNoDefault(t *testing.T) {
	source := `package main

func foo(x int) {
	switch x {
	case 1:
		doOne()
	}
	after()
}
`
	funcNode, sourceBytes := parseGoFunction(t, source)

	cfg, _, err := BuildGoCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Without default, switch block has edge to merge
	assert.GreaterOrEqual(t, len(cfg.Blocks), 5)
}

func TestBuildGoCFG_Select(t *testing.T) {
	source := `package main

func foo() {
	select {
	case msg := <-ch:
		process(msg)
	case <-done:
		cleanup()
	}
}
`
	funcNode, sourceBytes := parseGoFunction(t, source)

	cfg, _, err := BuildGoCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.GreaterOrEqual(t, len(cfg.Blocks), 5, "Should have entry, select, 2 cases, merge, exit")
}

func TestBuildGoCFG_Return(t *testing.T) {
	source := `package main

func foo(x int) int {
	if x > 0 {
		return x
	}
	return 0
}
`
	funcNode, sourceBytes := parseGoFunction(t, source)

	cfg, _, err := BuildGoCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Both returns should have edges to exit
	exitPreds := cfg.Blocks[cfg.ExitBlockID].Predecessors
	assert.GreaterOrEqual(t, len(exitPreds), 2, "Both return paths should reach exit")
}

func TestBuildGoCFG_NilNode(t *testing.T) {
	_, _, err := BuildGoCFGFromAST("test", nil, nil)
	assert.Error(t, err)
}

func TestBuildGoCFG_EmptyBody(t *testing.T) {
	source := `package main

func foo() {}
`
	funcNode, sourceBytes := parseGoFunction(t, source)

	cfg, blockStmts, err := BuildGoCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Empty function: entry → body → exit
	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	assert.Equal(t, 0, totalStmts)
}
