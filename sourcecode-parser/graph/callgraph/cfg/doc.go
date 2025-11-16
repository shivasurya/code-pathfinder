// Package cfg provides control flow graph (CFG) construction and analysis.
//
// This package builds CFGs from statement sequences for flow-sensitive analysis.
// CFGs are essential for advanced static analysis including:
//   - Data flow analysis
//   - Taint propagation
//   - Dead code detection
//   - Reachability analysis
//
// # Basic Blocks
//
// A BasicBlock represents a maximal sequence of instructions with:
//   - Single entry point (at the beginning)
//   - Single exit point (at the end)
//   - No internal branches
//
// # Control Flow Graph
//
// Build a CFG from a sequence of statements:
//
//	cfg := cfg.BuildCFG(statements)
//	for _, block := range cfg.Blocks {
//	    fmt.Printf("Block %d: %s\n", block.ID, block.Type)
//	    for _, successor := range block.Successors {
//	        fmt.Printf("  -> Block %d\n", successor.ID)
//	    }
//	}
//
// # Block Types
//
// The package defines several block types:
//   - BlockTypeEntry: Function entry point
//   - BlockTypeExit: Function exit point
//   - BlockTypeNormal: Straight-line code
//   - BlockTypeConditional: If statements, ternary operators
//   - BlockTypeLoop: While/for loop headers
//   - BlockTypeSwitch: Switch/match statements
//   - BlockTypeTry: Try blocks
//   - BlockTypeCatch: Exception handlers
//   - BlockTypeFinally: Finally blocks
//
// # Usage Example
//
//	// Build CFG for a function
//	cfg := cfg.NewControlFlowGraph("myapp.process_payment")
//
//	// Create basic blocks
//	entryBlock := &cfg.BasicBlock{
//	    ID:   0,
//	    Type: cfg.BlockTypeEntry,
//	}
//	cfg.Entry = entryBlock
//	cfg.Blocks = append(cfg.Blocks, entryBlock)
//
//	// Analyze control flow
//	for _, block := range cfg.Blocks {
//	    if block.Type == cfg.BlockTypeConditional {
//	        // Analyze both branches
//	    }
//	}
package cfg
