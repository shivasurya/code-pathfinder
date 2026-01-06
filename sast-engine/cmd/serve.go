package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/builder"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/mcp"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server for AI coding assistants",
	Long: `Builds code index and starts MCP server on stdio.

Designed for integration with Claude Code, Codex CLI, and other AI assistants
that support the Model Context Protocol (MCP).

The server indexes the codebase once at startup, then responds to queries
about symbols, call graphs, and code relationships.`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("project", "p", ".", "Project path to index")
	serveCmd.Flags().String("python-version", "3.11", "Python version for stdlib resolution")
}

func runServe(cmd *cobra.Command, _ []string) error {
	projectPath, _ := cmd.Flags().GetString("project")
	pythonVersion, _ := cmd.Flags().GetString("python-version")

	fmt.Fprintln(os.Stderr, "Building index...")
	start := time.Now()

	// Create logger for build process (verbose to stderr)
	logger := output.NewLogger(output.VerbosityVerbose)

	// 1. Initialize code graph (AST parsing)
	codeGraph := graph.Initialize(projectPath)
	if codeGraph == nil {
		return fmt.Errorf("failed to initialize code graph")
	}

	// 2. Build module registry
	moduleRegistry, err := registry.BuildModuleRegistry(projectPath, true) // skip tests
	if err != nil {
		return fmt.Errorf("failed to build module registry: %w", err)
	}

	// 3. Build call graph (5-pass algorithm)
	callGraph, err := builder.BuildCallGraph(codeGraph, moduleRegistry, projectPath, logger)
	if err != nil {
		return fmt.Errorf("failed to build call graph: %w", err)
	}

	buildTime := time.Since(start)
	fmt.Fprintf(os.Stderr, "Index built in %v\n", buildTime)
	fmt.Fprintf(os.Stderr, "  Functions: %d\n", len(callGraph.Functions))
	fmt.Fprintf(os.Stderr, "  Call edges: %d\n", len(callGraph.Edges))
	fmt.Fprintf(os.Stderr, "  Modules: %d\n", len(moduleRegistry.Modules))

	// 4. Create and run MCP server
	server := mcp.NewServer(projectPath, pythonVersion, callGraph, moduleRegistry, codeGraph, buildTime)

	fmt.Fprintln(os.Stderr, "Starting MCP server on stdio...")
	return server.ServeStdio()
}
