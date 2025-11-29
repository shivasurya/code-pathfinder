// Package core provides foundational type definitions for the callgraph analyzer.
//
// This package contains pure data structures with minimal dependencies that form
// the contract for all other callgraph packages. Types in this package should:
//
//   - Have zero circular dependencies
//   - Contain minimal business logic
//   - Be stable and rarely change
//
// # Core Types
//
// CallGraph represents the complete call graph with edges between functions.
//
// Statement represents individual program statements for def-use analysis.
//
// TaintSummary stores results of taint analysis for a function.
//
// # Usage
//
//	import "github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
//
//	cg := core.NewCallGraph()
//	cg.AddEdge("main.foo", "main.bar")
package core
