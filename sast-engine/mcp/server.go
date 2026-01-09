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
	projectPath    string
	pythonVersion  string
	callGraph      *core.CallGraph
	moduleRegistry *core.ModuleRegistry
	codeGraph      *graph.CodeGraph
	indexedAt      time.Time
	buildTime      time.Duration
	statusTracker  *StatusTracker
	degradation    *GracefulDegradation
	analytics      *Analytics
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
	mcpAnalytics := NewAnalytics("stdio")
	mcpAnalytics.ReportIndexingComplete(stats)

	return &Server{
		projectPath:    projectPath,
		pythonVersion:  pythonVersion,
		callGraph:      callGraph,
		moduleRegistry: moduleRegistry,
		codeGraph:      codeGraph,
		indexedAt:      time.Now(),
		buildTime:      buildTime,
		statusTracker:  tracker,
		degradation:    NewGracefulDegradation(tracker),
		analytics:      mcpAnalytics,
	}
}

// SetTransport updates the analytics transport type (e.g., "http").
func (s *Server) SetTransport(transport string) {
	s.analytics = NewAnalytics(transport)
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

	return SuccessResponse(req.ID, InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: ServerInfo{
			Name:    "pathfinder",
			Version: "0.1.0-poc",
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

