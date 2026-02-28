package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// Server handles MCP protocol communication.
type Server struct {
	projectPath      string
	pythonVersion    string
	callGraph        *core.CallGraph
	moduleRegistry   *core.ModuleRegistry
	codeGraph        *graph.CodeGraph
	indexedAt        time.Time
	buildTime        time.Duration
	statusTracker    *StatusTracker
	degradation      *GracefulDegradation
	analytics        *Analytics
	disableAnalytics bool
	version          string

	// Go-specific context for stdlib metadata in MCP tool responses.
	// Set via SetGoContext after the Go call graph is built.
	goVersion        string
	goModuleRegistry *core.GoModuleRegistry
}

// SetVersion sets the server version reported in MCP initialize responses.
// Should be called with cmd.Version (injected via ldflags at build time).
func (s *Server) SetVersion(version string) {
	s.version = version
}

// SetGoContext stores the Go version and module registry so that MCP tool
// responses can include stdlib metadata (is_stdlib, signature, return_type,
// etc.) for Go standard library calls.
//
// Must be called after InitGoStdlibLoader has populated reg.StdlibLoader.
// Safe to skip — tools degrade gracefully when goModuleRegistry is nil.
func (s *Server) SetGoContext(version string, reg *core.GoModuleRegistry) {
	s.goVersion = version
	s.goModuleRegistry = reg
}

// NewServer creates a new MCP server with the given index data.
func NewServer(
	projectPath string,
	pythonVersion string,
	callGraph *core.CallGraph,
	moduleRegistry *core.ModuleRegistry,
	codeGraph *graph.CodeGraph,
	buildTime time.Duration,
	disableAnalytics bool,
) *Server {
	tracker := NewStatusTracker()

	// Mark as ready since we're being created with complete data.
	tracker.StartIndexing()
	stats := &IndexingStats{
		Functions:     len(callGraph.Functions),
		CallEdges:     len(callGraph.Edges),
		Modules:       len(moduleRegistry.Modules),
		Files:         len(moduleRegistry.FileToModule),
		BuildDuration: buildTime,
	}
	tracker.CompleteIndexing(stats)

	// Initialize analytics with stdio transport (default).
	mcpAnalytics := NewAnalytics("stdio", disableAnalytics)
	mcpAnalytics.ReportIndexingComplete(stats)

	return &Server{
		projectPath:      projectPath,
		pythonVersion:    pythonVersion,
		callGraph:        callGraph,
		moduleRegistry:   moduleRegistry,
		codeGraph:        codeGraph,
		indexedAt:        time.Now(),
		buildTime:        buildTime,
		statusTracker:    tracker,
		degradation:      NewGracefulDegradation(tracker),
		analytics:        mcpAnalytics,
		disableAnalytics: disableAnalytics,
	}
}

// NewServerWithBackgroundIndexing creates a server that will be populated via background indexing.
func NewServerWithBackgroundIndexing(projectPath, pythonVersion string, disableAnalytics bool) *Server {
	tracker := NewStatusTracker()
	// Starts in StateUninitialized

	return &Server{
		projectPath:      projectPath,
		pythonVersion:    pythonVersion,
		callGraph:        nil, // Will be set later
		moduleRegistry:   nil, // Will be set later
		codeGraph:        nil, // Will be set later
		statusTracker:    tracker,
		degradation:      NewGracefulDegradation(tracker),
		analytics:        NewAnalytics("stdio", disableAnalytics),
		disableAnalytics: disableAnalytics,
	}
}

// UpdateIndexingStatus updates the indexing progress.
func (s *Server) UpdateIndexingStatus(state IndexingState, phase IndexingPhase, message string, progress float64) {
	s.statusTracker.SetPhase(phase, message)
}

// SetIndexReady marks indexing as complete and updates with indexed data.
func (s *Server) SetIndexReady(callGraph *core.CallGraph, moduleReg *core.ModuleRegistry,
	codeGraph *graph.CodeGraph, buildTime time.Duration) {
	s.callGraph = callGraph
	s.moduleRegistry = moduleReg
	s.codeGraph = codeGraph
	s.buildTime = buildTime
	s.indexedAt = time.Now()

	stats := &IndexingStats{
		Functions:     len(callGraph.Functions),
		CallEdges:     len(callGraph.Edges),
		Modules:       len(moduleReg.Modules),
		Files:         len(moduleReg.FileToModule),
		BuildDuration: buildTime,
	}
	s.statusTracker.CompleteIndexing(stats)
	s.analytics.ReportIndexingComplete(stats)
}

// SetIndexingError marks indexing as failed.
func (s *Server) SetIndexingError(err error) {
	s.statusTracker.FailIndexing(err)
}

// SetTransport updates the analytics transport type (e.g., "http").
func (s *Server) SetTransport(transport string) {
	s.analytics = NewAnalytics(transport, s.disableAnalytics)
}

// ServeStdio starts the MCP server on stdin/stdout.
func (s *Server) ServeStdio() error {
	reader := bufio.NewReader(os.Stdin)

	// Report server started.
	s.analytics.ReportServerStarted()
	defer s.analytics.ReportServerStopped()

	for {
		// Read line from stdin.
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Fprintln(os.Stderr, "Client disconnected")
				return nil // Clean shutdown
			}
			return fmt.Errorf("read error: %w", err)
		}

		// Skip empty lines.
		if len(line) <= 1 {
			continue
		}

		// Parse JSON-RPC request.
		var request JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			s.sendResponse(MakeErrorResponse(nil, ParseError(err.Error())))
			continue
		}

		// Handle request and send response.
		response := s.handleRequest(&request)
		if response != nil {
			s.sendResponse(response)
		}
	}
}

// sendResponse writes a JSON-RPC response to stdout.
func (s *Server) sendResponse(resp *JSONRPCResponse) {
	bytes, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal response: %v\n", err)
		return
	}
	fmt.Println(string(bytes))
}

// handleRequest dispatches to the appropriate handler.
func (s *Server) handleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	startTime := time.Now()

	// Validate JSON-RPC version.
	if req.JSONRPC != "2.0" {
		return MakeErrorResponse(req.ID, InvalidRequestError("jsonrpc must be '2.0'"))
	}

	// Validate method exists.
	if req.Method == "" {
		return MakeErrorResponse(req.ID, InvalidRequestError("method is required"))
	}

	var response *JSONRPCResponse

	switch req.Method {
	case "initialize":
		response = s.handleInitialize(req)
	case "initialized":
		// Acknowledgment notification - no response needed.
		fmt.Fprintln(os.Stderr, "Client initialized")
		return nil
	case "notifications/initialized":
		// Alternative notification format.
		return nil
	case "tools/list":
		response = s.handleToolsList(req)
	case "tools/call":
		response = s.handleToolsCall(req)
	case "status":
		response = s.handleStatus(req)
	case "ping":
		response = SuccessResponse(req.ID, map[string]string{"status": "ok"})
	default:
		response = MakeErrorResponse(req.ID, MethodNotFoundError(req.Method))
	}

	// Log request timing.
	elapsed := time.Since(startTime)
	fmt.Fprintf(os.Stderr, "[%s] %s (%v)\n", req.Method, "completed", elapsed)

	return response
}

// handleInitialize responds to the initialize request.
func (s *Server) handleInitialize(req *JSONRPCRequest) *JSONRPCResponse {
	// Parse client info if needed.
	var params InitializeParams
	if req.Params != nil {
		_ = json.Unmarshal(req.Params, &params)
		fmt.Fprintf(os.Stderr, "Client: %s %s\n", params.ClientInfo.Name, params.ClientInfo.Version)

		// Report client connection (only name/version, no PII).
		s.analytics.ReportClientConnected(params.ClientInfo.Name, params.ClientInfo.Version)
	}

	version := s.version
	if version == "" {
		version = "dev"
	}

	return SuccessResponse(req.ID, InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: ServerInfo{
			Name:    "dev.codepathfinder/pathfinder",
			Version: version,
		},
		Capabilities: Capabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
		},
	})
}

// handleToolsList returns the list of available tools.
func (s *Server) handleToolsList(req *JSONRPCRequest) *JSONRPCResponse {
	tools := s.getToolDefinitions()
	return SuccessResponse(req.ID, ToolsListResult{
		Tools: tools,
	})
}

// handleToolsCall executes a tool.
func (s *Server) handleToolsCall(req *JSONRPCRequest) *JSONRPCResponse {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return MakeErrorResponse(req.ID, InvalidParamsError(err.Error()))
	}

	if params.Name == "" {
		return MakeErrorResponse(req.ID, InvalidParamsError("tool name is required"))
	}

	fmt.Fprintf(os.Stderr, "Tool call: %s\n", params.Name)

	// Track tool call metrics.
	metrics := s.analytics.StartToolCall(params.Name)
	result, isError := s.executeTool(params.Name, params.Arguments)
	s.analytics.EndToolCall(metrics, !isError)

	return SuccessResponse(req.ID, ToolResult{
		Content: []ContentBlock{
			{
				Type: "text",
				Text: result,
			},
		},
		IsError: isError,
	})
}

// handleStatus returns the current indexing status.
func (s *Server) handleStatus(req *JSONRPCRequest) *JSONRPCResponse {
	return SuccessResponse(req.ID, s.degradation.GetStatusJSON())
}

// GetStatusTracker returns the status tracker for external use.
func (s *Server) GetStatusTracker() *StatusTracker {
	return s.statusTracker
}

// IsReady returns true if the server is ready to handle tool requests.
func (s *Server) IsReady() bool {
	return s.statusTracker.IsReady()
}

