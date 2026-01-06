package mcp

import (
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// Server handles MCP protocol communication.
type Server struct {
	projectPath    string
	pythonVersion  string
	callGraph      *core.CallGraph
	moduleRegistry *core.ModuleRegistry
	codeGraph      *graph.CodeGraph
	indexedAt      time.Time
	buildTime      time.Duration
}

// NewServer creates a new MCP server with the given index data.
func NewServer(
	projectPath string,
	pythonVersion string,
	callGraph *core.CallGraph,
	moduleRegistry *core.ModuleRegistry,
	codeGraph *graph.CodeGraph,
	buildTime time.Duration,
) *Server {
	return &Server{
		projectPath:    projectPath,
		pythonVersion:  pythonVersion,
		callGraph:      callGraph,
		moduleRegistry: moduleRegistry,
		codeGraph:      codeGraph,
		indexedAt:      time.Now(),
		buildTime:      buildTime,
	}
}

// ServeStdio starts the MCP server on stdin/stdout.
// This is a skeleton - full implementation in Commit 2.
func (s *Server) ServeStdio() error {
	// TODO: Implement in Commit 2
	// For now, just return nil to verify build
	return nil
}
