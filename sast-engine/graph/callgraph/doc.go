// Package callgraph provides static call graph analysis for Python code.
//
// This package is organized into several sub-packages for better modularity:
//
// # Core Types
//
// The core package contains fundamental data structures:
//
//	import "github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
//
//	cg := core.NewCallGraph()
//	cg.AddEdge("main.foo", "main.bar")
//
// # Registry
//
// The registry package manages module, builtin, and stdlib registries:
//
//	import "github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
//
//	moduleRegistry := registry.BuildModuleRegistry("/path/to/project")
//	builtins := registry.NewBuiltinRegistry()
//
// # Resolution
//
// The resolution package handles import, type, and call resolution:
//
//	import "github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
//
//	engine := resolution.NewTypeInferenceEngine(moduleRegistry)
//	typeInfo := engine.InferType(expr, scope)
//
// # Extraction
//
// The extraction package extracts code elements from AST:
//
//	import "github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/extraction"
//
//	statements := extraction.ExtractStatements(sourceCode, functionName)
//
// # Patterns
//
// The patterns package detects security and framework patterns:
//
//	import "github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/patterns"
//
//	registry := patterns.NewPatternRegistry()
//	matched := patterns.MatchPattern(pattern, funcFQN, statements)
//
// # Analysis
//
// The analysis package provides taint analysis:
//
//	import "github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/analysis/taint"
//
//	summary := taint.AnalyzeIntraProceduralTaint(funcFQN, statements, ...)
//
// # CFG
//
// The cfg package provides control flow graph construction:
//
//	import "github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/cfg"
//
//	controlFlow := cfg.BuildCFG(statements)
//
// # Builder
//
// The builder package orchestrates call graph construction:
//
//	import "github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/builder"
//
//	callGraph, err := builder.BuildCallGraphFromPath(codeGraph, "/path/to/project")
//
// # Quick Start
//
// To build a call graph for a Python project:
//
//	import (
//	    "github.com/shivasurya/code-pathfinder/sast-engine/graph"
//	    "github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph"
//	)
//
//	// Parse project
//	codeGraph := graph.Initialize(projectPath)
//
//	// Build call graph with all features
//	callGraph, moduleRegistry, patternRegistry, err := callgraph.InitializeCallGraph(codeGraph, projectPath)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Analyze for security patterns
//	matches := callgraph.AnalyzePatterns(callGraph, patternRegistry)
package callgraph
