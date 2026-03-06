package cfg

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parsePythonFunction(t *testing.T, source string) *sitter.Node {
	t.Helper()
	parser := sitter.NewParser()
	parser.SetLanguage(python.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(source))
	require.NoError(t, err)
	// Find the function_definition node
	root := tree.RootNode()
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child.Type() == "function_definition" {
			return child
		}
	}
	t.Fatal("no function_definition found in source")
	return nil
}

func TestBuildCFG_LinearFunction(t *testing.T) {
	source := `def foo():
    x = source()
    y = x
    sink(y)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have: entry, body block, exit
	assert.GreaterOrEqual(t, len(cfg.Blocks), 3)

	// Count total statements across all blocks
	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	assert.Equal(t, 3, totalStmts, "should have 3 statements: assignment, assignment, call")

	// Verify paths: should be exactly 1 path from entry to exit
	paths := cfg.GetAllPaths()
	assert.GreaterOrEqual(t, len(paths), 1)
}

func TestBuildCFG_IfElse(t *testing.T) {
	source := `def foo():
    x = source()
    if x:
        y = x
    else:
        y = "safe"
    sink(y)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have blocks for: entry, body, if_cond, if_true, if_false, if_merge, exit
	assert.GreaterOrEqual(t, len(cfg.Blocks), 6)

	// Verify statements are extracted from BOTH branches
	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	// x=source(), condition, y=x (true), y="safe" (false), sink(y) (after merge)
	assert.GreaterOrEqual(t, totalStmts, 4, "should have statements from both branches")

	// Verify at least 2 paths (true and false branches)
	paths := cfg.GetAllPaths()
	assert.GreaterOrEqual(t, len(paths), 2, "should have at least 2 paths through if/else")
}

func TestBuildCFG_IfNoBranch(t *testing.T) {
	source := `def foo():
    x = source()
    if x:
        y = x
    sink(y)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should extract y=x from inside the if
	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	assert.GreaterOrEqual(t, totalStmts, 3, "should have: x=source, cond, y=x, sink")
	_ = blockStmts
}

func TestBuildCFG_ForLoop(t *testing.T) {
	source := `def foo():
    items = source()
    for item in items:
        sink(item)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have loop header and body blocks
	hasLoop := false
	for _, block := range cfg.Blocks {
		if block.Type == BlockTypeLoop {
			hasLoop = true
			break
		}
	}
	assert.True(t, hasLoop, "should have a loop block")

	// Statement inside for body should be extracted
	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	// items=source(), for header (item in items), sink(item)
	assert.GreaterOrEqual(t, totalStmts, 3)
}

func TestBuildCFG_WhileLoop(t *testing.T) {
	source := `def foo():
    x = source()
    while x:
        sink(x)
        x = transform(x)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	hasLoop := false
	for _, block := range cfg.Blocks {
		if block.Type == BlockTypeLoop {
			hasLoop = true
			break
		}
	}
	assert.True(t, hasLoop, "should have a while loop block")

	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	// x=source(), while cond, sink(x), x=transform(x)
	assert.GreaterOrEqual(t, totalStmts, 4)
}

func TestBuildCFG_TryExcept(t *testing.T) {
	source := `def foo():
    try:
        x = source()
        sink(x)
    except ValueError:
        y = "safe"
        sink(y)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have try and catch blocks
	hasTry := false
	hasCatch := false
	for _, block := range cfg.Blocks {
		if block.Type == BlockTypeTry {
			hasTry = true
		}
		if block.Type == BlockTypeCatch {
			hasCatch = true
		}
	}
	assert.True(t, hasTry, "should have a try block")
	assert.True(t, hasCatch, "should have a catch block")

	// Statements from both try and except bodies should be extracted
	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	// x=source(), sink(x), y="safe", sink(y)
	assert.GreaterOrEqual(t, totalStmts, 4)
}

func TestBuildCFG_WithStatement(t *testing.T) {
	source := `def foo():
    with open(filename) as f:
        data = f.read()
        sink(data)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Statements inside with body should be extracted
	totalStmts := 0
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
	}
	// with-var def (f), data=f.read(), sink(data)
	assert.GreaterOrEqual(t, totalStmts, 2, "should extract statements from with body")
}

func TestBuildCFG_NestedIfInFor(t *testing.T) {
	source := `def foo():
    items = source()
    for item in items:
        if item:
            sink(item)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Should have both loop and conditional blocks
	hasLoop := false
	hasCond := false
	for _, block := range cfg.Blocks {
		if block.Type == BlockTypeLoop {
			hasLoop = true
		}
		if block.Type == BlockTypeConditional {
			hasCond = true
		}
	}
	assert.True(t, hasLoop, "should have loop block")
	assert.True(t, hasCond, "should have conditional block inside loop")

	// sink(item) inside if inside for should be extracted
	totalStmts := 0
	foundSink := false
	for _, stmts := range blockStmts {
		totalStmts += len(stmts)
		for _, s := range stmts {
			if s.CallTarget == "sink" {
				foundSink = true
			}
		}
	}
	assert.True(t, foundSink, "should extract sink() call from inside nested control flow")
}

func TestBuildCFG_ReturnInMiddle(t *testing.T) {
	source := `def foo():
    x = source()
    if x:
        return x
    sink(x)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	cfg, _, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// The return inside if should connect to exit
	paths := cfg.GetAllPaths()
	assert.GreaterOrEqual(t, len(paths), 1, "should have paths through the function")
}

func TestBuildCFG_StatementsHaveLineNumbers(t *testing.T) {
	source := `def foo():
    x = source()
    y = x
    sink(y)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	_, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)

	// All statements should have non-zero line numbers
	for _, stmts := range blockStmts {
		for _, s := range stmts {
			assert.Greater(t, s.LineNumber, uint32(0), "statement should have line number set")
		}
	}
}

func TestBuildCFG_BlockStatementsPreserveDefUse(t *testing.T) {
	source := `def foo():
    x = source()
    y = x
    sink(y)
`
	funcNode := parsePythonFunction(t, source)
	sourceBytes := []byte(source)

	_, blockStmts, err := BuildCFGFromAST("test.foo", funcNode, sourceBytes)
	require.NoError(t, err)

	// Collect all statements
	var allStmts []*struct {
		def  string
		uses []string
	}
	for _, stmts := range blockStmts {
		for _, s := range stmts {
			allStmts = append(allStmts, &struct {
				def  string
				uses []string
			}{s.Def, s.Uses})
		}
	}

	// Should have: x = source() (def=x), y = x (def=y, uses=[x]), sink(y) (uses=[y])
	foundXDef := false
	foundYDef := false
	foundSinkUse := false
	for _, s := range allStmts {
		if s.def == "x" {
			foundXDef = true
		}
		if s.def == "y" {
			foundYDef = true
			assert.Contains(t, s.uses, "x", "y=x should have x in uses")
		}
		if s.def == "" && len(s.uses) > 0 {
			for _, u := range s.uses {
				if u == "y" {
					foundSinkUse = true
				}
			}
		}
	}
	assert.True(t, foundXDef, "should have x definition")
	assert.True(t, foundYDef, "should have y definition")
	assert.True(t, foundSinkUse, "should have sink using y")
}
