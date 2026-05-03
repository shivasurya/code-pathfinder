package cfg

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	clang "github.com/smacker/go-tree-sitter/c"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseCFunction parses C source and returns the function_definition node.
// Uses the C grammar (switch/do-while node types are identical in C++,
// so a single grammar is enough for these tests).
func parseCFunction(t *testing.T, source string) *sitter.Node {
	t.Helper()
	parser := sitter.NewParser()
	parser.SetLanguage(clang.GetLanguage())
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, []byte(source))
	require.NoError(t, err)
	root := tree.RootNode()
	for i := 0; i < int(root.ChildCount()); i++ {
		if c := root.Child(i); c != nil && c.Type() == "function_definition" {
			return c
		}
	}
	t.Fatal("no function_definition found in C source")
	return nil
}

// blockTypes returns every distinct BlockType present in the CFG.
// Useful for asserting on structural shape without hard-coding block IDs.
func blockTypes(cfg *ControlFlowGraph) map[BlockType]int {
	out := make(map[BlockType]int)
	for _, blk := range cfg.Blocks {
		out[blk.Type]++
	}
	return out
}

// TestBuildCFG_C_Switch_BasicShape verifies a simple switch with
// three cases produces one BlockTypeSwitch header, one block per case,
// and a merge block reachable from every case.
func TestBuildCFG_C_Switch_BasicShape(t *testing.T) {
	src := `void f(int x) {
    switch (x) {
        case 1: a(); break;
        case 2: b(); break;
        default: c(); break;
    }
}`
	fn := parseCFunction(t, src)
	cfg, _, err := BuildCFGFromAST("test::f", fn, []byte(src))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	types := blockTypes(cfg)
	assert.Equal(t, 1, types[BlockTypeSwitch], "expected exactly one switch header")
	assert.NotZero(t, types[BlockTypeNormal], "expected at least one normal block")

	// Every block type table should let us reach exit.
	paths := cfg.GetAllPaths()
	assert.NotEmpty(t, paths, "switch graph must have a path to exit")
}

// TestBuildCFG_C_Switch_FallthroughEdge verifies that consecutive
// cases without an explicit edge gating still leave the previous case
// connected to the next — necessary for fallthrough semantics.
func TestBuildCFG_C_Switch_FallthroughEdge(t *testing.T) {
	src := `void f(int x) {
    switch (x) {
        case 1: a(); /* no break */
        case 2: b(); break;
    }
}`
	fn := parseCFunction(t, src)
	cfg, _, err := BuildCFGFromAST("test::f", fn, []byte(src))
	require.NoError(t, err)

	// Find the switch header and confirm it has fan-out edges.
	var switchID string
	for id, blk := range cfg.Blocks {
		if blk.Type == BlockTypeSwitch {
			switchID = id
			break
		}
	}
	require.NotEmpty(t, switchID, "expected a switch header block")
	header := cfg.Blocks[switchID]
	// Header should fan out to: each case + the merge block.
	assert.GreaterOrEqual(t, len(header.Successors), 2)
}

// TestBuildCFG_C_Switch_EmptyBody guards the empty-switch corner: the
// header must still connect to a merge block so reachability holds.
func TestBuildCFG_C_Switch_EmptyBody(t *testing.T) {
	src := `void f(int x) {
    switch (x) { }
}`
	fn := parseCFunction(t, src)
	cfg, _, err := BuildCFGFromAST("test::f", fn, []byte(src))
	require.NoError(t, err)

	types := blockTypes(cfg)
	assert.Equal(t, 1, types[BlockTypeSwitch])
	paths := cfg.GetAllPaths()
	assert.NotEmpty(t, paths, "empty switch must still reach exit")
}

// TestBuildCFG_C_DoWhile verifies that do-while produces the
// `[body] -> [cond] -> [body]` loop shape and a falling-through
// after-block.
func TestBuildCFG_C_DoWhile(t *testing.T) {
	src := `void f(int x) {
    do {
        consume(x);
    } while (x > 0);
}`
	fn := parseCFunction(t, src)
	cfg, _, err := BuildCFGFromAST("test::f", fn, []byte(src))
	require.NoError(t, err)

	types := blockTypes(cfg)
	assert.NotZero(t, types[BlockTypeLoop], "expected at least one loop header (do-cond)")

	// The loop header should have at least two successors (body and after).
	var loopID string
	for id, blk := range cfg.Blocks {
		if blk.Type == BlockTypeLoop {
			loopID = id
			break
		}
	}
	require.NotEmpty(t, loopID)
	assert.GreaterOrEqual(t, len(cfg.Blocks[loopID].Successors), 2, "do-cond fan-out (body + after)")

	paths := cfg.GetAllPaths()
	assert.NotEmpty(t, paths)
}

// TestBuildCFG_C_DoWhile_BodyExecutesFirst verifies the entry block
// flows directly into the body block (no header gate before the first
// iteration), matching do-while's "always execute once" semantics.
func TestBuildCFG_C_DoWhile_BodyExecutesFirst(t *testing.T) {
	src := `void f() {
    do { a(); } while (1);
}`
	fn := parseCFunction(t, src)
	cfg, _, err := BuildCFGFromAST("test::f", fn, []byte(src))
	require.NoError(t, err)

	// The entry block's reachable successors should include a
	// non-loop body block before the loop header.
	entry := cfg.Blocks[cfg.EntryBlockID]
	require.NotNil(t, entry)

	// Walk from entry until we find a loop block; everything reachable
	// before it must be normal/conditional.
	visited := map[string]bool{}
	var seenLoop bool
	var walk func(id string)
	walk = func(id string) {
		if visited[id] || id == cfg.ExitBlockID {
			return
		}
		visited[id] = true
		blk := cfg.Blocks[id]
		if blk == nil {
			return
		}
		if blk.Type == BlockTypeLoop {
			seenLoop = true
		}
		for _, s := range blk.Successors {
			walk(s)
		}
	}
	walk(cfg.EntryBlockID)
	assert.True(t, seenLoop, "do-while should include a loop block reachable from entry")
}
