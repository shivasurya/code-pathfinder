package cfg

import (
	"slices"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// BlockType represents the type of basic block in a control flow graph.
// Different block types enable different security analysis patterns.
type BlockType string

const (
	// BlockTypeEntry represents the entry point of a function.
	// Every function has exactly one entry block.
	BlockTypeEntry BlockType = "entry"

	// BlockTypeExit represents the exit point of a function.
	// Every function has exactly one exit block where all return paths converge.
	BlockTypeExit BlockType = "exit"

	// BlockTypeNormal represents a regular basic block with sequential execution.
	// Contains straight-line code with no branches.
	BlockTypeNormal BlockType = "normal"

	// BlockTypeConditional represents a conditional branch block.
	// Has multiple successor blocks (true/false branches).
	// Examples: if statements, ternary operators, short-circuit logic.
	BlockTypeConditional BlockType = "conditional"

	// BlockTypeLoop represents a loop header block.
	// Has back-edges for loop iteration.
	// Examples: while loops, for loops, do-while loops.
	BlockTypeLoop BlockType = "loop"

	// BlockTypeSwitch represents a switch/match statement block.
	// Has multiple successor blocks (one per case).
	BlockTypeSwitch BlockType = "switch"

	// BlockTypeTry represents a try block in exception handling.
	// Has normal successor and exception handler successors.
	BlockTypeTry BlockType = "try"

	// BlockTypeCatch represents a catch/except block in exception handling.
	// Handles exceptions from try blocks.
	BlockTypeCatch BlockType = "catch"

	// BlockTypeFinally represents a finally block in exception handling.
	// Always executes regardless of exceptions.
	BlockTypeFinally BlockType = "finally"
)

// BasicBlock represents a basic block in a control flow graph.
// A basic block is a maximal sequence of instructions with:
//   - Single entry point (at the beginning)
//   - Single exit point (at the end)
//   - No internal branches
//
// Basic blocks are the nodes in a CFG, connected by edges representing
// control flow between blocks.
type BasicBlock struct {
	// ID uniquely identifies this block within the CFG
	ID string

	// Type categorizes the block for analysis purposes
	Type BlockType

	// StartLine is the first line of code in this block (1-indexed)
	StartLine int

	// EndLine is the last line of code in this block (1-indexed)
	EndLine int

	// Instructions contains the call sites within this block.
	// Call sites represent function/method invocations that occur
	// during execution of this block.
	Instructions []core.CallSite

	// Successors are the blocks that can execute after this block.
	// For normal blocks: single successor
	// For conditional blocks: two successors (true/false branches)
	// For switch blocks: multiple successors (one per case)
	// For exit blocks: empty (no successors)
	Successors []string

	// Predecessors are the blocks that can execute before this block.
	// Used for backward analysis and dominance calculations.
	Predecessors []string

	// Condition stores the condition expression for conditional blocks.
	// Empty for non-conditional blocks.
	// Examples: "x > 0", "user.is_admin()", "data is not None"
	Condition string

	// Dominators are the blocks that always execute before this block
	// on any path from entry. Used for security analysis to determine
	// if sanitization always occurs before usage.
	Dominators []string
}

// ControlFlowGraph represents the control flow graph of a function.
// A CFG models all possible execution paths through a function, enabling
// data flow and taint analysis for security vulnerabilities.
//
// Example:
//
//	def process_user(user_id):
//	    user = get_user(user_id)        # Block 1 (entry)
//	    if user.is_admin():              # Block 2 (conditional)
//	        grant_access()               # Block 3 (true branch)
//	    else:
//	        deny_access()                # Block 4 (false branch)
//	    log_action(user)                 # Block 5 (merge point)
//	    return                           # Block 6 (exit)
//
// CFG Structure:
//
//	Entry → Block1 → Block2 → Block3 → Block5 → Exit
//	                       → Block4 ↗
type ControlFlowGraph struct {
	// FunctionFQN is the fully qualified name of the function this CFG represents
	FunctionFQN string

	// Blocks maps block IDs to BasicBlock objects
	Blocks map[string]*BasicBlock

	// EntryBlockID identifies the entry block
	EntryBlockID string

	// ExitBlockID identifies the exit block
	ExitBlockID string

	// CallGraph reference for resolving inter-procedural flows
	CallGraph *core.CallGraph
}

// NewControlFlowGraph creates and initializes a new CFG for a function.
func NewControlFlowGraph(functionFQN string) *ControlFlowGraph {
	cfg := &ControlFlowGraph{
		FunctionFQN: functionFQN,
		Blocks:      make(map[string]*BasicBlock),
	}

	// Create entry and exit blocks
	entryBlock := &BasicBlock{
		ID:           functionFQN + ":entry",
		Type:         BlockTypeEntry,
		Successors:   []string{},
		Predecessors: []string{},
		Instructions: []core.CallSite{},
	}

	exitBlock := &BasicBlock{
		ID:           functionFQN + ":exit",
		Type:         BlockTypeExit,
		Successors:   []string{},
		Predecessors: []string{},
		Instructions: []core.CallSite{},
	}

	cfg.Blocks[entryBlock.ID] = entryBlock
	cfg.Blocks[exitBlock.ID] = exitBlock
	cfg.EntryBlockID = entryBlock.ID
	cfg.ExitBlockID = exitBlock.ID

	return cfg
}

// AddBlock adds a basic block to the CFG.
func (cfg *ControlFlowGraph) AddBlock(block *BasicBlock) {
	cfg.Blocks[block.ID] = block
}

// AddEdge adds a control flow edge from one block to another.
// Automatically updates both successors and predecessors.
func (cfg *ControlFlowGraph) AddEdge(fromBlockID, toBlockID string) {
	fromBlock, fromExists := cfg.Blocks[fromBlockID]
	toBlock, toExists := cfg.Blocks[toBlockID]

	if !fromExists || !toExists {
		return
	}

	// Add to successors if not already present
	if !containsString(fromBlock.Successors, toBlockID) {
		fromBlock.Successors = append(fromBlock.Successors, toBlockID)
	}

	// Add to predecessors if not already present
	if !containsString(toBlock.Predecessors, fromBlockID) {
		toBlock.Predecessors = append(toBlock.Predecessors, fromBlockID)
	}
}

// GetBlock retrieves a block by ID.
func (cfg *ControlFlowGraph) GetBlock(blockID string) (*BasicBlock, bool) {
	block, exists := cfg.Blocks[blockID]
	return block, exists
}

// GetSuccessors returns the successor blocks of a given block.
func (cfg *ControlFlowGraph) GetSuccessors(blockID string) []*BasicBlock {
	block, exists := cfg.Blocks[blockID]
	if !exists {
		return nil
	}

	successors := make([]*BasicBlock, 0, len(block.Successors))
	for _, succID := range block.Successors {
		if succBlock, ok := cfg.Blocks[succID]; ok {
			successors = append(successors, succBlock)
		}
	}
	return successors
}

// GetPredecessors returns the predecessor blocks of a given block.
func (cfg *ControlFlowGraph) GetPredecessors(blockID string) []*BasicBlock {
	block, exists := cfg.Blocks[blockID]
	if !exists {
		return nil
	}

	predecessors := make([]*BasicBlock, 0, len(block.Predecessors))
	for _, predID := range block.Predecessors {
		if predBlock, ok := cfg.Blocks[predID]; ok {
			predecessors = append(predecessors, predBlock)
		}
	}
	return predecessors
}

// ComputeDominators calculates dominator sets for all blocks.
// A block X dominates block Y if every path from entry to Y must go through X.
// This is essential for determining if sanitization always occurs before usage.
//
// Algorithm: Iterative data flow analysis
//  1. Initialize: Entry dominates only itself, all others dominated by all blocks
//  2. Iterate until fixed point:
//     For each block B (except entry):
//     Dom(B) = {B} ∪ (intersection of Dom(P) for all predecessors P of B)
func (cfg *ControlFlowGraph) ComputeDominators() {
	// Initialize dominator sets
	allBlockIDs := make([]string, 0, len(cfg.Blocks))
	for blockID := range cfg.Blocks {
		allBlockIDs = append(allBlockIDs, blockID)
	}

	// Entry block dominates only itself
	entryBlock := cfg.Blocks[cfg.EntryBlockID]
	entryBlock.Dominators = []string{cfg.EntryBlockID}

	// All other blocks initially dominated by all blocks
	for blockID, block := range cfg.Blocks {
		if blockID != cfg.EntryBlockID {
			block.Dominators = append([]string{}, allBlockIDs...)
		}
	}

	// Iterate until no changes
	changed := true
	for changed {
		changed = false

		for blockID, block := range cfg.Blocks {
			if blockID == cfg.EntryBlockID {
				continue
			}

			// Compute intersection of predecessors' dominators
			var newDominators []string
			if len(block.Predecessors) > 0 {
				// Start with first predecessor's dominators
				firstPred := cfg.Blocks[block.Predecessors[0]]
				newDominators = append([]string{}, firstPred.Dominators...)

				// Intersect with other predecessors
				for i := 1; i < len(block.Predecessors); i++ {
					pred := cfg.Blocks[block.Predecessors[i]]
					newDominators = intersect(newDominators, pred.Dominators)
				}
			}

			// Add block itself to dominator set
			if !containsString(newDominators, blockID) {
				newDominators = append(newDominators, blockID)
			}

			// Check if dominators changed
			if !slicesEqual(block.Dominators, newDominators) {
				block.Dominators = newDominators
				changed = true
			}
		}
	}
}

// IsDominator returns true if dominator dominates dominated.
// Used to check if sanitization (in dominator) always occurs before usage (in dominated).
func (cfg *ControlFlowGraph) IsDominator(dominator, dominated string) bool {
	block, exists := cfg.Blocks[dominated]
	if !exists {
		return false
	}
	return containsString(block.Dominators, dominator)
}

// GetAllPaths returns all execution paths from entry to exit.
// Used for exhaustive security analysis.
// WARNING: Can be exponential in size for complex CFGs with loops.
func (cfg *ControlFlowGraph) GetAllPaths() [][]string {
	var paths [][]string
	var currentPath []string
	visited := make(map[string]bool)

	cfg.dfsAllPaths(cfg.EntryBlockID, currentPath, visited, &paths)
	return paths
}

// dfsAllPaths performs depth-first search to enumerate all paths.
func (cfg *ControlFlowGraph) dfsAllPaths(blockID string, currentPath []string, visited map[string]bool, paths *[][]string) {
	// Avoid infinite loops in cyclic CFGs
	if visited[blockID] {
		return
	}

	// Add current block to path
	currentPath = append(currentPath, blockID)
	visited[blockID] = true

	// If we reached exit, save this path
	if blockID == cfg.ExitBlockID {
		pathCopy := make([]string, len(currentPath))
		copy(pathCopy, currentPath)
		*paths = append(*paths, pathCopy)
	} else {
		// Recurse on successors
		block := cfg.Blocks[blockID]
		for _, succID := range block.Successors {
			cfg.dfsAllPaths(succID, currentPath, visited, paths)
		}
	}

	// Backtrack
	visited[blockID] = false
}

// Helper function to compute intersection of two string slices.
func intersect(a, b []string) []string {
	result := []string{}
	for _, item := range a {
		if containsString(b, item) {
			result = append(result, item)
		}
	}
	return result
}

// Helper function to check if two string slices are equal.
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Helper function to check if a string slice contains a specific string.
func containsString(slice []string, item string) bool {
	return slices.Contains(slice, item)
}
