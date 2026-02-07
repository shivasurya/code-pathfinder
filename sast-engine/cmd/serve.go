package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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
	Long: `Builds code index and starts MCP server.

Designed for integration with Claude Code, Codex CLI, and other AI assistants
that support the Model Context Protocol (MCP).

The server indexes the codebase once at startup, then responds to queries
about symbols, call graphs, and code relationships.

Transport modes:
  - stdio (default): Standard input/output for direct integration
  - http: HTTP server for network access`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().StringP("project", "p", ".", "Project path to index")
	serveCmd.Flags().String("python-version", "", "Python version override (auto-detected from .python-version or pyproject.toml)")
	serveCmd.Flags().Bool("http", false, "Use HTTP transport instead of stdio")
	serveCmd.Flags().String("address", ":8080", "HTTP server address (only with --http)")
}

func runServe(cmd *cobra.Command, _ []string) error {
	projectPath, _ := cmd.Flags().GetString("project")
	pythonVersionOverride, _ := cmd.Flags().GetString("python-version")
	useHTTP, _ := cmd.Flags().GetBool("http")
	address, _ := cmd.Flags().GetString("address")
	disableAnalytics, _ := cmd.Flags().GetBool("disable-metrics")

	fmt.Fprintln(os.Stderr, "Building index...")
	start := time.Now()

	// Auto-detect Python version (same logic as BuildCallGraph).
	pythonVersion := builder.DetectPythonVersion(projectPath)
	if pythonVersionOverride != "" {
		pythonVersion = pythonVersionOverride
		fmt.Fprintf(os.Stderr, "Using Python version override: %s\n", pythonVersion)
	} else {
		fmt.Fprintf(os.Stderr, "Detected Python version: %s\n", pythonVersion)
	}

	// Create logger for build process (verbose to stderr)
	logger := output.NewLogger(output.VerbosityVerbose)

	// 1. Initialize code graph (AST parsing)
	codeGraph := graph.Initialize(projectPath, nil)
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

	// 4. Create MCP server
	server := mcp.NewServer(projectPath, pythonVersion, callGraph, moduleRegistry, codeGraph, buildTime, disableAnalytics)

	// 5. Start appropriate transport
	if useHTTP {
		return runHTTPServer(server, address)
	}

	fmt.Fprintln(os.Stderr, "Starting MCP server on stdio...")
	return server.ServeStdio()
}

func runHTTPServer(mcpServer *mcp.Server, address string) error {
	// Set transport type for analytics.
	mcpServer.SetTransport("http")

	config := &mcp.HTTPConfig{
		Address:         address,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		AllowedOrigins:  []string{"*"},
	}

	httpServer := mcp.NewHTTPServer(mcpServer, config)

	// Handle graceful shutdown.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		errChan <- httpServer.Start()
	}()

	select {
	case err := <-errChan:
		return err
	case sig := <-sigChan:
		fmt.Fprintf(os.Stderr, "\nReceived %v, shutting down...\n", sig)
		ctx, cancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
		defer cancel()
		return httpServer.Shutdown(ctx)
	}
}
