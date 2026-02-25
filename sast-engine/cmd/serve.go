package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/builder"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
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

	// Auto-detect Python version
	pythonVersion := builder.DetectPythonVersion(projectPath)
	if pythonVersionOverride != "" {
		pythonVersion = pythonVersionOverride
	}

	fmt.Fprintln(os.Stderr, "Starting MCP server...")

	// Create server with empty index (will be populated by background indexing)
	server := mcp.NewServerWithBackgroundIndexing(projectPath, pythonVersion, disableAnalytics)

	// Start indexing in background goroutine
	go func() {
		fmt.Fprintln(os.Stderr, "Building index in background...")
		server.UpdateIndexingStatus(mcp.StateIndexing, mcp.PhaseParsing, "Parsing AST...", 0.1)
		start := time.Now()

		logger := output.NewLogger(output.VerbosityVerbose)

		// 1. Initialize code graph (AST parsing)
		server.UpdateIndexingStatus(mcp.StateIndexing, mcp.PhaseParsing, "Parsing source files...", 0.2)
		codeGraph := graph.Initialize(projectPath, nil)
		if codeGraph == nil {
			server.SetIndexingError(fmt.Errorf("failed to initialize code graph"))
			return
		}

		// 2. Build module registry
		server.UpdateIndexingStatus(mcp.StateIndexing, mcp.PhaseModuleRegistry, "Building module registry...", 0.3)
		moduleRegistry, err := registry.BuildModuleRegistry(projectPath, true)
		if err != nil {
			server.SetIndexingError(fmt.Errorf("failed to build module registry: %w", err))
			return
		}
		fmt.Fprintf(os.Stderr, "Loaded manifest: %d modules\n", len(moduleRegistry.Modules))

		// 3. Build call graph (5-pass algorithm)
		server.UpdateIndexingStatus(mcp.StateIndexing, mcp.PhaseCallGraph, "Building Python call graph...", 0.5)
		callGraph, err := builder.BuildCallGraph(codeGraph, moduleRegistry, projectPath, logger)
		if err != nil {
			server.SetIndexingError(fmt.Errorf("failed to build call graph: %w", err))
			return
		}

		// 4. Build Go call graph if go.mod exists
		goModPath := filepath.Join(projectPath, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			server.UpdateIndexingStatus(mcp.StateIndexing, mcp.PhaseCallGraph, "Building Go call graph...", 0.7)
			fmt.Fprintf(os.Stderr, "Detected go.mod, building Go call graph...\n")

			goRegistry, err := resolution.BuildGoModuleRegistry(projectPath)
			if err != nil {
				logger.Warning("Failed to build Go module registry: %v", err)
			} else {
				if goRegistry.GoVersion != "" {
					fmt.Fprintf(os.Stderr, "Detected Go version: %s\n", goRegistry.GoVersion)
				}
				fmt.Fprintf(os.Stderr, "Go module: %s\n", goRegistry.ModulePath)

				builder.InitGoStdlibLoader(goRegistry, projectPath, logger)
				goTypeEngine := resolution.NewGoTypeInferenceEngine(goRegistry)
				goCG, err := builder.BuildGoCallGraph(codeGraph, goRegistry, goTypeEngine)
				if err != nil {
					logger.Warning("Failed to build Go call graph: %v", err)
				} else {
					builder.MergeCallGraphs(callGraph, goCG)
					fmt.Fprintf(os.Stderr, "Go call graph merged: %d functions, %d call sites\n",
						len(goCG.Functions), len(goCG.CallSites))
				}
			}
		}

		buildTime := time.Since(start)
		fmt.Fprintf(os.Stderr, "Index built in %v\n", buildTime)
		fmt.Fprintf(os.Stderr, "  Total functions: %d\n", len(callGraph.Functions))
		fmt.Fprintf(os.Stderr, "  Call edges: %d\n", len(callGraph.Edges))
		fmt.Fprintf(os.Stderr, "  Modules: %d\n", len(moduleRegistry.Modules))

		// Mark indexing as complete and update server with data
		server.SetIndexReady(callGraph, moduleRegistry, codeGraph, buildTime)
		fmt.Fprintln(os.Stderr, "Indexing complete - server ready!")
	}()

	// Start serving immediately (before indexing completes)
	fmt.Fprintln(os.Stderr, "MCP server ready (indexing in background)...")

	if useHTTP {
		return runHTTPServer(server, address)
	}

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
