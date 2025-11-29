package cfg

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewControlFlowGraph(t *testing.T) {
	cfg := NewControlFlowGraph("myapp.views.get_user")

	assert.NotNil(t, cfg)
	assert.Equal(t, "myapp.views.get_user", cfg.FunctionFQN)
	assert.NotNil(t, cfg.Blocks)
	assert.Len(t, cfg.Blocks, 2) // Entry and exit blocks

	// Verify entry block
	entryBlock, exists := cfg.Blocks[cfg.EntryBlockID]
	require.True(t, exists)
	assert.Equal(t, BlockTypeEntry, entryBlock.Type)
	assert.Equal(t, "myapp.views.get_user:entry", entryBlock.ID)

	// Verify exit block
	exitBlock, exists := cfg.Blocks[cfg.ExitBlockID]
	require.True(t, exists)
	assert.Equal(t, BlockTypeExit, exitBlock.Type)
	assert.Equal(t, "myapp.views.get_user:exit", exitBlock.ID)
}

func TestBasicBlock_Creation(t *testing.T) {
	block := &BasicBlock{
		ID:           "block1",
		Type:         BlockTypeNormal,
		StartLine:    10,
		EndLine:      15,
		Instructions: []core.CallSite{},
		Successors:   []string{"block2"},
		Predecessors: []string{"entry"},
	}

	assert.Equal(t, "block1", block.ID)
	assert.Equal(t, BlockTypeNormal, block.Type)
	assert.Equal(t, 10, block.StartLine)
	assert.Equal(t, 15, block.EndLine)
	assert.Len(t, block.Successors, 1)
	assert.Len(t, block.Predecessors, 1)
}

func TestCFG_AddBlock(t *testing.T) {
	cfg := NewControlFlowGraph("myapp.test")

	block := &BasicBlock{
		ID:   "block1",
		Type: BlockTypeNormal,
	}

	cfg.AddBlock(block)

	assert.Len(t, cfg.Blocks, 3) // Entry, exit, and new block
	retrievedBlock, exists := cfg.GetBlock("block1")
	assert.True(t, exists)
	assert.Equal(t, block, retrievedBlock)
}

func TestCFG_AddEdge(t *testing.T) {
	cfg := NewControlFlowGraph("myapp.test")

	block1 := &BasicBlock{ID: "block1", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}
	block2 := &BasicBlock{ID: "block2", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}

	cfg.AddBlock(block1)
	cfg.AddBlock(block2)

	cfg.AddEdge("block1", "block2")

	// Verify successors
	assert.Contains(t, block1.Successors, "block2")

	// Verify predecessors
	assert.Contains(t, block2.Predecessors, "block1")
}

func TestCFG_AddEdge_Duplicate(t *testing.T) {
	cfg := NewControlFlowGraph("myapp.test")

	block1 := &BasicBlock{ID: "block1", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}
	block2 := &BasicBlock{ID: "block2", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}

	cfg.AddBlock(block1)
	cfg.AddBlock(block2)

	// Add edge twice
	cfg.AddEdge("block1", "block2")
	cfg.AddEdge("block1", "block2")

	// Should only appear once
	assert.Len(t, block1.Successors, 1)
	assert.Len(t, block2.Predecessors, 1)
}

func TestCFG_AddEdge_NonExistentBlocks(t *testing.T) {
	cfg := NewControlFlowGraph("myapp.test")

	// Try to add edge between non-existent blocks
	cfg.AddEdge("nonexistent1", "nonexistent2")

	// Should not crash, just silently ignore
	assert.Len(t, cfg.Blocks, 2) // Only entry and exit
}

func TestCFG_GetBlock(t *testing.T) {
	cfg := NewControlFlowGraph("myapp.test")

	block := &BasicBlock{ID: "block1", Type: BlockTypeNormal}
	cfg.AddBlock(block)

	// Existing block
	retrieved, exists := cfg.GetBlock("block1")
	assert.True(t, exists)
	assert.Equal(t, block, retrieved)

	// Non-existent block
	_, exists = cfg.GetBlock("nonexistent")
	assert.False(t, exists)
}

func TestCFG_GetSuccessors(t *testing.T) {
	cfg := NewControlFlowGraph("myapp.test")

	block1 := &BasicBlock{ID: "block1", Type: BlockTypeConditional, Successors: []string{}, Predecessors: []string{}}
	block2 := &BasicBlock{ID: "block2", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}
	block3 := &BasicBlock{ID: "block3", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}

	cfg.AddBlock(block1)
	cfg.AddBlock(block2)
	cfg.AddBlock(block3)

	cfg.AddEdge("block1", "block2")
	cfg.AddEdge("block1", "block3")

	successors := cfg.GetSuccessors("block1")
	assert.Len(t, successors, 2)

	successorIDs := []string{successors[0].ID, successors[1].ID}
	assert.Contains(t, successorIDs, "block2")
	assert.Contains(t, successorIDs, "block3")
}

func TestCFG_GetPredecessors(t *testing.T) {
	cfg := NewControlFlowGraph("myapp.test")

	block1 := &BasicBlock{ID: "block1", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}
	block2 := &BasicBlock{ID: "block2", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}
	block3 := &BasicBlock{ID: "block3", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}

	cfg.AddBlock(block1)
	cfg.AddBlock(block2)
	cfg.AddBlock(block3)

	cfg.AddEdge("block1", "block3")
	cfg.AddEdge("block2", "block3")

	predecessors := cfg.GetPredecessors("block3")
	assert.Len(t, predecessors, 2)

	predecessorIDs := []string{predecessors[0].ID, predecessors[1].ID}
	assert.Contains(t, predecessorIDs, "block1")
	assert.Contains(t, predecessorIDs, "block2")
}

func TestCFG_ComputeDominators_Linear(t *testing.T) {
	// Test linear CFG: Entry → Block1 → Block2 → Exit
	cfg := NewControlFlowGraph("myapp.test")

	block1 := &BasicBlock{ID: "block1", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}
	block2 := &BasicBlock{ID: "block2", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}

	cfg.AddBlock(block1)
	cfg.AddBlock(block2)

	cfg.AddEdge(cfg.EntryBlockID, "block1")
	cfg.AddEdge("block1", "block2")
	cfg.AddEdge("block2", cfg.ExitBlockID)

	cfg.ComputeDominators()

	// Entry dominates itself
	assert.Contains(t, cfg.Blocks[cfg.EntryBlockID].Dominators, cfg.EntryBlockID)
	assert.Len(t, cfg.Blocks[cfg.EntryBlockID].Dominators, 1)

	// Block1 dominated by entry and itself
	assert.Contains(t, block1.Dominators, cfg.EntryBlockID)
	assert.Contains(t, block1.Dominators, "block1")

	// Block2 dominated by entry, block1, and itself
	assert.Contains(t, block2.Dominators, cfg.EntryBlockID)
	assert.Contains(t, block2.Dominators, "block1")
	assert.Contains(t, block2.Dominators, "block2")

	// Exit dominated by all blocks
	assert.Contains(t, cfg.Blocks[cfg.ExitBlockID].Dominators, cfg.EntryBlockID)
	assert.Contains(t, cfg.Blocks[cfg.ExitBlockID].Dominators, "block1")
	assert.Contains(t, cfg.Blocks[cfg.ExitBlockID].Dominators, "block2")
	assert.Contains(t, cfg.Blocks[cfg.ExitBlockID].Dominators, cfg.ExitBlockID)
}

func TestCFG_ComputeDominators_Branch(t *testing.T) {
	// Test branching CFG:
	//   Entry → Block1 → Block2 → Block4 → Exit
	//                  → Block3 ↗
	cfg := NewControlFlowGraph("myapp.test")

	block1 := &BasicBlock{ID: "block1", Type: BlockTypeConditional, Successors: []string{}, Predecessors: []string{}}
	block2 := &BasicBlock{ID: "block2", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}
	block3 := &BasicBlock{ID: "block3", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}
	block4 := &BasicBlock{ID: "block4", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}

	cfg.AddBlock(block1)
	cfg.AddBlock(block2)
	cfg.AddBlock(block3)
	cfg.AddBlock(block4)

	cfg.AddEdge(cfg.EntryBlockID, "block1")
	cfg.AddEdge("block1", "block2")
	cfg.AddEdge("block1", "block3")
	cfg.AddEdge("block2", "block4")
	cfg.AddEdge("block3", "block4")
	cfg.AddEdge("block4", cfg.ExitBlockID)

	cfg.ComputeDominators()

	// Block1 dominates block2 and block3
	assert.Contains(t, block2.Dominators, "block1")
	assert.Contains(t, block3.Dominators, "block1")

	// Block4 dominated by entry, block1, and itself (NOT by block2 or block3)
	assert.Contains(t, block4.Dominators, cfg.EntryBlockID)
	assert.Contains(t, block4.Dominators, "block1")
	assert.Contains(t, block4.Dominators, "block4")
	// Block4 should NOT be dominated by block2 or block3 (can reach via either path)
	assert.NotContains(t, block4.Dominators, "block2")
	assert.NotContains(t, block4.Dominators, "block3")
}

func TestCFG_IsDominator(t *testing.T) {
	// Linear CFG: Entry → Block1 → Block2 → Exit
	cfg := NewControlFlowGraph("myapp.test")

	block1 := &BasicBlock{ID: "block1", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}
	block2 := &BasicBlock{ID: "block2", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}

	cfg.AddBlock(block1)
	cfg.AddBlock(block2)

	cfg.AddEdge(cfg.EntryBlockID, "block1")
	cfg.AddEdge("block1", "block2")
	cfg.AddEdge("block2", cfg.ExitBlockID)

	cfg.ComputeDominators()

	// Block1 dominates block2
	assert.True(t, cfg.IsDominator("block1", "block2"))

	// Entry dominates block1
	assert.True(t, cfg.IsDominator(cfg.EntryBlockID, "block1"))

	// Block2 does NOT dominate block1
	assert.False(t, cfg.IsDominator("block2", "block1"))
}

func TestCFG_GetAllPaths_Linear(t *testing.T) {
	// Linear CFG: Entry → Block1 → Exit
	cfg := NewControlFlowGraph("myapp.test")

	block1 := &BasicBlock{ID: "block1", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}
	cfg.AddBlock(block1)

	cfg.AddEdge(cfg.EntryBlockID, "block1")
	cfg.AddEdge("block1", cfg.ExitBlockID)

	paths := cfg.GetAllPaths()

	require.Len(t, paths, 1)
	assert.Equal(t, []string{cfg.EntryBlockID, "block1", cfg.ExitBlockID}, paths[0])
}

func TestCFG_GetAllPaths_Branch(t *testing.T) {
	// Branching CFG:
	//   Entry → Block1 → Block2 → Exit
	//                  → Block3 ↗
	cfg := NewControlFlowGraph("myapp.test")

	block1 := &BasicBlock{ID: "block1", Type: BlockTypeConditional, Successors: []string{}, Predecessors: []string{}}
	block2 := &BasicBlock{ID: "block2", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}
	block3 := &BasicBlock{ID: "block3", Type: BlockTypeNormal, Successors: []string{}, Predecessors: []string{}}

	cfg.AddBlock(block1)
	cfg.AddBlock(block2)
	cfg.AddBlock(block3)

	cfg.AddEdge(cfg.EntryBlockID, "block1")
	cfg.AddEdge("block1", "block2")
	cfg.AddEdge("block1", "block3")
	cfg.AddEdge("block2", cfg.ExitBlockID)
	cfg.AddEdge("block3", cfg.ExitBlockID)

	paths := cfg.GetAllPaths()

	require.Len(t, paths, 2)

	// Convert paths to comparable format
	path1 := []string{cfg.EntryBlockID, "block1", "block2", cfg.ExitBlockID}
	path2 := []string{cfg.EntryBlockID, "block1", "block3", cfg.ExitBlockID}

	assert.Contains(t, paths, path1)
	assert.Contains(t, paths, path2)
}

func TestBlockType_Constants(t *testing.T) {
	assert.Equal(t, BlockType("entry"), BlockTypeEntry)
	assert.Equal(t, BlockType("exit"), BlockTypeExit)
	assert.Equal(t, BlockType("normal"), BlockTypeNormal)
	assert.Equal(t, BlockType("conditional"), BlockTypeConditional)
	assert.Equal(t, BlockType("loop"), BlockTypeLoop)
	assert.Equal(t, BlockType("switch"), BlockTypeSwitch)
	assert.Equal(t, BlockType("try"), BlockTypeTry)
	assert.Equal(t, BlockType("catch"), BlockTypeCatch)
	assert.Equal(t, BlockType("finally"), BlockTypeFinally)
}

func TestBasicBlock_WithInstructions(t *testing.T) {
	callSite := core.CallSite{
		Target: "sanitize",
		Location: core.Location{
			File:   "/test/file.py",
			Line:   10,
			Column: 5,
		},
		Arguments: []core.Argument{
			{Value: "data", IsVariable: true, Position: 0},
		},
		Resolved:  true,
		TargetFQN: "myapp.utils.sanitize",
	}

	block := &BasicBlock{
		ID:           "block1",
		Type:         BlockTypeNormal,
		StartLine:    10,
		EndLine:      12,
		Instructions: []core.CallSite{callSite},
	}

	assert.Len(t, block.Instructions, 1)
	assert.Equal(t, "sanitize", block.Instructions[0].Target)
	assert.Equal(t, "myapp.utils.sanitize", block.Instructions[0].TargetFQN)
}

func TestBasicBlock_ConditionalWithCondition(t *testing.T) {
	block := &BasicBlock{
		ID:        "block1",
		Type:      BlockTypeConditional,
		Condition: "user.is_admin()",
		Successors: []string{"true_branch", "false_branch"},
	}

	assert.Equal(t, BlockTypeConditional, block.Type)
	assert.Equal(t, "user.is_admin()", block.Condition)
	assert.Len(t, block.Successors, 2)
}

// TestIntersect and TestSlicesEqual are not included because intersect and slicesEqual
// are private functions in the callgraph package. These helper functions are tested
// indirectly through the dominator computation tests above.

func TestCFG_ComplexExample(t *testing.T) {
	// Test a more realistic CFG structure representing:
	// def process_user(user_id):
	//     user = get_user(user_id)        # Block 1
	//     if user.is_admin():              # Block 2 (conditional)
	//         grant_access()               # Block 3 (true branch)
	//     else:
	//         deny_access()                # Block 4 (false branch)
	//     log_action(user)                 # Block 5 (merge point)
	//     return                           # Exit

	cfg := NewControlFlowGraph("myapp.process_user")

	block1 := &BasicBlock{
		ID:        "block1",
		Type:      BlockTypeNormal,
		StartLine: 2,
		EndLine:   2,
		Instructions: []core.CallSite{
			{Target: "get_user", TargetFQN: "myapp.db.get_user"},
		},
		Successors:   []string{},
		Predecessors: []string{},
	}

	block2 := &BasicBlock{
		ID:           "block2",
		Type:         BlockTypeConditional,
		StartLine:    3,
		EndLine:      3,
		Condition:    "user.is_admin()",
		Successors:   []string{},
		Predecessors: []string{},
	}

	block3 := &BasicBlock{
		ID:        "block3",
		Type:      BlockTypeNormal,
		StartLine: 4,
		EndLine:   4,
		Instructions: []core.CallSite{
			{Target: "grant_access", TargetFQN: "myapp.auth.grant_access"},
		},
		Successors:   []string{},
		Predecessors: []string{},
	}

	block4 := &BasicBlock{
		ID:        "block4",
		Type:      BlockTypeNormal,
		StartLine: 6,
		EndLine:   6,
		Instructions: []core.CallSite{
			{Target: "deny_access", TargetFQN: "myapp.auth.deny_access"},
		},
		Successors:   []string{},
		Predecessors: []string{},
	}

	block5 := &BasicBlock{
		ID:        "block5",
		Type:      BlockTypeNormal,
		StartLine: 7,
		EndLine:   7,
		Instructions: []core.CallSite{
			{Target: "log_action", TargetFQN: "myapp.logging.log_action"},
		},
		Successors:   []string{},
		Predecessors: []string{},
	}

	cfg.AddBlock(block1)
	cfg.AddBlock(block2)
	cfg.AddBlock(block3)
	cfg.AddBlock(block4)
	cfg.AddBlock(block5)

	// Build edges
	cfg.AddEdge(cfg.EntryBlockID, "block1")
	cfg.AddEdge("block1", "block2")
	cfg.AddEdge("block2", "block3") // True branch
	cfg.AddEdge("block2", "block4") // False branch
	cfg.AddEdge("block3", "block5") // Merge
	cfg.AddEdge("block4", "block5") // Merge
	cfg.AddEdge("block5", cfg.ExitBlockID)

	// Compute dominators
	cfg.ComputeDominators()

	// Verify structure
	assert.Len(t, cfg.Blocks, 7) // Entry, 5 blocks, Exit

	// Verify paths
	paths := cfg.GetAllPaths()
	assert.Len(t, paths, 2) // Two paths (admin and non-admin)

	// Verify dominators
	// Block1 should dominate block5 (always executed before block5)
	assert.True(t, cfg.IsDominator("block1", "block5"))

	// Block2 should dominate block5 (always executed before block5)
	assert.True(t, cfg.IsDominator("block2", "block5"))

	// Block3 should NOT dominate block5 (only on true path)
	assert.False(t, cfg.IsDominator("block3", "block5"))

	// Block4 should NOT dominate block5 (only on false path)
	assert.False(t, cfg.IsDominator("block4", "block5"))
}
