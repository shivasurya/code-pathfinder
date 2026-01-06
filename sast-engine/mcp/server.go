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
func (s *Server) ServeStdio() error {
	reader := bufio.NewReader(os.Stdin)

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
			s.sendResponse(ErrorResponse(nil, -32700, "Parse error: "+err.Error()))
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
	case "ping":
		response = SuccessResponse(req.ID, map[string]string{"status": "ok"})
	default:
		response = ErrorResponse(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
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
		return ErrorResponse(req.ID, -32602, "Invalid params: "+err.Error())
	}

	fmt.Fprintf(os.Stderr, "Tool call: %s\n", params.Name)

	result, isError := s.executeTool(params.Name, params.Arguments)

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

// getToolDefinitions returns tool schemas.
// Stub implementation - full definitions in Commit 3.
func (s *Server) getToolDefinitions() []Tool {
	return []Tool{
		{
			Name:        "get_index_info",
			Description: "Get information about the indexed codebase",
			InputSchema: InputSchema{Type: "object", Properties: map[string]Property{}},
		},
		{
			Name:        "find_symbol",
			Description: "Find a function or class by name",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{"name": {Type: "string", Description: "Symbol name"}},
				Required:   []string{"name"},
			},
		},
		{
			Name:        "get_callers",
			Description: "Find all functions that call a given function",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{"function": {Type: "string", Description: "Function name"}},
				Required:   []string{"function"},
			},
		},
		{
			Name:        "get_callees",
			Description: "Find all functions called by a given function",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{"function": {Type: "string", Description: "Function name"}},
				Required:   []string{"function"},
			},
		},
		{
			Name:        "get_call_details",
			Description: "Get detailed information about a specific call site",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"caller": {Type: "string", Description: "Caller function"},
					"callee": {Type: "string", Description: "Callee function"},
				},
				Required: []string{"caller", "callee"},
			},
		},
		{
			Name:        "resolve_import",
			Description: "Resolve a Python import path to its file location",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{"import": {Type: "string", Description: "Import path"}},
				Required:   []string{"import"},
			},
		},
	}
}

// executeTool runs a tool and returns the result.
// Stub implementation - full logic in Commit 3.
func (s *Server) executeTool(name string, _ map[string]interface{}) (string, bool) {
	switch name {
	case "get_index_info":
		// Quick implementation for testing.
		result := map[string]interface{}{
			"project_path":       s.projectPath,
			"indexed_at":         s.indexedAt.Format(time.RFC3339),
			"build_time_seconds": s.buildTime.Seconds(),
			"stats": map[string]int{
				"functions":  len(s.callGraph.Functions),
				"call_edges": len(s.callGraph.Edges),
				"modules":    len(s.moduleRegistry.Modules),
			},
		}
		bytes, _ := json.MarshalIndent(result, "", "  ")
		return string(bytes), false
	default:
		return fmt.Sprintf(`{"error": "Tool not yet implemented: %s", "note": "Full implementation in Commit 3"}`, name), true
	}
}
