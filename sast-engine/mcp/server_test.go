package mcp

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestServer creates a Server with test fixture data for testing.
func createTestServer() *Server {
	callGraph := core.NewCallGraph()

	// Add test function nodes.
	callGraph.Functions["myapp.auth.validate_user"] = &graph.Node{
		ID:         "1",
		Type:       "function_definition",
		Name:       "validate_user",
		File:       "/path/to/myapp/auth.py",
		LineNumber: 45,
		ReturnType: "User",
	}

	callGraph.Functions["myapp.views.login"] = &graph.Node{
		ID:         "2",
		Type:       "function_definition",
		Name:       "login",
		File:       "/path/to/myapp/views.py",
		LineNumber: 10,
	}

	callGraph.Functions["myapp.views.logout"] = &graph.Node{
		ID:         "3",
		Type:       "function_definition",
		Name:       "logout",
		File:       "/path/to/myapp/views.py",
		LineNumber: 50,
	}

	// Add call edges: login calls validate_user.
	callGraph.Edges["myapp.views.login"] = []string{"myapp.auth.validate_user"}
	callGraph.ReverseEdges["myapp.auth.validate_user"] = []string{"myapp.views.login"}

	// Add call site details.
	callGraph.CallSites["myapp.views.login"] = []core.CallSite{
		{
			Target:    "validate_user",
			TargetFQN: "myapp.auth.validate_user",
			Location:  core.Location{File: "/path/to/myapp/views.py", Line: 15, Column: 8},
			Resolved:  true,
		},
	}

	moduleRegistry := &core.ModuleRegistry{
		Modules:      map[string]string{"myapp.auth": "/path/to/myapp/auth.py", "myapp.views": "/path/to/myapp/views.py"},
		FileToModule: map[string]string{"/path/to/myapp/auth.py": "myapp.auth", "/path/to/myapp/views.py": "myapp.views"},
		ShortNames:   map[string][]string{"auth": {"/path/to/myapp/auth.py"}, "views": {"/path/to/myapp/views.py"}},
	}

	return NewServer("/test/project", "3.11", callGraph, moduleRegistry, nil, time.Second)
}

// createExtendedTestServer creates a Server with extended fixture data for coverage testing.
func createExtendedTestServer() *Server {
	callGraph := core.NewCallGraph()

	// Function with all optional fields.
	callGraph.Functions["myapp.auth.validate_user"] = &graph.Node{
		ID:                  "1",
		Type:                "function_definition",
		Name:                "validate_user",
		File:                "/path/to/myapp/auth.py",
		LineNumber:          45,
		ReturnType:          "User",
		MethodArgumentsType: []string{"username: str", "password: str"},
		Modifier:            "public",
		Annotation:          []string{"@login_required"},
		SuperClass:          "BaseValidator",
	}

	callGraph.Functions["myapp.views.login"] = &graph.Node{
		ID:         "2",
		Type:       "function_definition",
		Name:       "login",
		File:       "/path/to/myapp/views.py",
		LineNumber: 10,
	}

	callGraph.Functions["myapp.views.logout"] = &graph.Node{
		ID:         "3",
		Type:       "function_definition",
		Name:       "logout",
		File:       "/path/to/myapp/views.py",
		LineNumber: 50,
	}

	callGraph.Functions["myapp.utils.process"] = &graph.Node{
		ID:         "4",
		Type:       "function_definition",
		Name:       "process",
		File:       "/path/to/myapp/utils.py",
		LineNumber: 1,
	}

	// Add call edges.
	callGraph.Edges["myapp.views.login"] = []string{"myapp.auth.validate_user", "external.unknown"}
	callGraph.ReverseEdges["myapp.auth.validate_user"] = []string{"myapp.views.login"}

	// Add call sites with various properties.
	callGraph.CallSites["myapp.views.login"] = []core.CallSite{
		{
			Target:    "validate_user",
			TargetFQN: "myapp.auth.validate_user",
			Location:  core.Location{File: "/path/to/myapp/views.py", Line: 15, Column: 8},
			Resolved:  true,
			Arguments: []core.Argument{
				{Position: 0, Value: "request.username"},
				{Position: 1, Value: "request.password"},
			},
		},
		{
			Target:        "external_call",
			TargetFQN:     "",
			Location:      core.Location{File: "/path/to/myapp/views.py", Line: 20, Column: 4},
			Resolved:      false,
			FailureReason: "Module 'external' not found in project",
		},
		{
			Target:                   "inferred_method",
			TargetFQN:                "myapp.utils.inferred_method",
			Location:                 core.Location{File: "/path/to/myapp/views.py", Line: 25, Column: 4},
			Resolved:                 true,
			ResolvedViaTypeInference: true,
			InferredType:             "MyClass",
			TypeConfidence:           0.85,
			TypeSource:               "assignment",
		},
	}

	moduleRegistry := &core.ModuleRegistry{
		Modules: map[string]string{
			"myapp.auth":   "/path/to/myapp/auth.py",
			"myapp.views":  "/path/to/myapp/views.py",
			"myapp.utils":  "/path/to/myapp/utils.py",
			"other.utils":  "/path/to/other/utils.py",
			"myapp.models": "/path/to/myapp/models.py",
		},
		FileToModule: map[string]string{
			"/path/to/myapp/auth.py":   "myapp.auth",
			"/path/to/myapp/views.py":  "myapp.views",
			"/path/to/myapp/utils.py":  "myapp.utils",
			"/path/to/other/utils.py":  "other.utils",
			"/path/to/myapp/models.py": "myapp.models",
		},
		ShortNames: map[string][]string{
			"auth":   {"/path/to/myapp/auth.py"},
			"views":  {"/path/to/myapp/views.py"},
			"utils":  {"/path/to/myapp/utils.py", "/path/to/other/utils.py"}, // Ambiguous
			"models": {"/path/to/myapp/models.py"},
		},
	}

	return NewServer("/test/project", "3.11", callGraph, moduleRegistry, nil, time.Second)
}

func TestNewServer(t *testing.T) {
	server := createTestServer()
	assert.NotNil(t, server)
	assert.Equal(t, "/test/project", server.projectPath)
	assert.Equal(t, "3.11", server.pythonVersion)
	assert.NotNil(t, server.callGraph)
	assert.NotNil(t, server.moduleRegistry)
}

func TestHandleInitialize(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1.0"}}`),
	}

	resp := server.handleInitialize(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(InitializeResult)
	require.True(t, ok)
	assert.Equal(t, "2024-11-05", result.ProtocolVersion)
	assert.Equal(t, "pathfinder", result.ServerInfo.Name)
	assert.Equal(t, "0.1.0-poc", result.ServerInfo.Version)
	assert.NotNil(t, result.Capabilities.Tools)
}

func TestHandleInitialize_NoParams(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}

	resp := server.handleInitialize(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
}

func TestHandleToolsList(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	resp := server.handleToolsList(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(ToolsListResult)
	require.True(t, ok)
	assert.Equal(t, 6, len(result.Tools)) // Exactly 6 tools from PoC
}

func TestHandleToolsCall_GetIndexInfo(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"get_index_info","arguments":{}}`),
	}

	resp := server.handleToolsCall(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(ToolResult)
	require.True(t, ok)
	assert.False(t, result.IsError)
	assert.Len(t, result.Content, 1)
	assert.Equal(t, "text", result.Content[0].Type)
	assert.Contains(t, result.Content[0].Text, "project_path")
	assert.Contains(t, result.Content[0].Text, "/test/project")
}

func TestHandleToolsCall_FindSymbol(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"find_symbol","arguments":{"name":"validate_user"}}`),
	}

	resp := server.handleToolsCall(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(ToolResult)
	require.True(t, ok)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "validate_user")
	assert.Contains(t, result.Content[0].Text, "myapp.auth.validate_user")
}

func TestHandleToolsCall_FindSymbol_NotFound(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"find_symbol","arguments":{"name":"nonexistent_xyz"}}`),
	}

	resp := server.handleToolsCall(req)

	assert.NotNil(t, resp)

	result, ok := resp.Result.(ToolResult)
	require.True(t, ok)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "not found")
}

func TestHandleToolsCall_GetCallers(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"get_callers","arguments":{"function":"validate_user"}}`),
	}

	resp := server.handleToolsCall(req)

	assert.NotNil(t, resp)

	result, ok := resp.Result.(ToolResult)
	require.True(t, ok)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "login")
	assert.Contains(t, result.Content[0].Text, "callers")
}

func TestHandleToolsCall_GetCallees(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"get_callees","arguments":{"function":"login"}}`),
	}

	resp := server.handleToolsCall(req)

	assert.NotNil(t, resp)

	result, ok := resp.Result.(ToolResult)
	require.True(t, ok)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "validate_user")
	assert.Contains(t, result.Content[0].Text, "callees")
}

func TestHandleToolsCall_GetCallDetails(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"get_call_details","arguments":{"caller":"login","callee":"validate_user"}}`),
	}

	resp := server.handleToolsCall(req)

	assert.NotNil(t, resp)

	result, ok := resp.Result.(ToolResult)
	require.True(t, ok)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "call_site")
}

func TestHandleToolsCall_ResolveImport(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"resolve_import","arguments":{"import":"myapp.auth"}}`),
	}

	resp := server.handleToolsCall(req)

	assert.NotNil(t, resp)

	result, ok := resp.Result.(ToolResult)
	require.True(t, ok)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "auth.py")
	assert.Contains(t, result.Content[0].Text, "resolved")
}

func TestHandleToolsCall_InvalidParams(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{invalid json}`),
	}

	resp := server.handleToolsCall(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32602, resp.Error.Code)
}

func TestHandleToolsCall_InvalidTool(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"invalid_tool","arguments":{}}`),
	}

	resp := server.handleToolsCall(req)

	assert.NotNil(t, resp)

	result, ok := resp.Result.(ToolResult)
	require.True(t, ok)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Content[0].Text, "Unknown tool")
}

func TestHandleRequest_Initialize(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}

	resp := server.handleRequest(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
}

func TestHandleRequest_Initialized(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialized",
	}

	resp := server.handleRequest(req)

	// initialized is a notification, no response expected.
	assert.Nil(t, resp)
}

func TestHandleRequest_NotificationsInitialized(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "notifications/initialized",
	}

	resp := server.handleRequest(req)

	// No response expected for notifications.
	assert.Nil(t, resp)
}

func TestHandleRequest_MethodNotFound(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "invalid/method",
	}

	resp := server.handleRequest(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32601, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "Method not found")
}

func TestHandleRequest_Ping(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "ping",
	}

	resp := server.handleRequest(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(map[string]string)
	require.True(t, ok)
	assert.Equal(t, "ok", result["status"])
}

func TestSendResponse(t *testing.T) {
	server := createTestServer()

	resp := SuccessResponse(1, map[string]string{"test": "value"})

	// sendResponse writes to stdout, just verify it doesn't panic.
	// In real tests, we'd capture stdout.
	assert.NotPanics(t, func() {
		server.sendResponse(resp)
	})
}

// ============================================================================
// Error Handling Tests (PR-02)
// ============================================================================

func TestHandleRequest_InvalidJSONRPCVersion(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "1.0", // Invalid version
		ID:      1,
		Method:  "ping",
	}

	resp := server.handleRequest(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeInvalidRequest, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "jsonrpc must be '2.0'")
}

func TestHandleRequest_MissingMethod(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "", // Missing method
	}

	resp := server.handleRequest(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeInvalidRequest, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "method is required")
}

func TestHandleToolsCall_MissingToolName(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"","arguments":{}}`),
	}

	resp := server.handleToolsCall(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeInvalidParams, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "tool name is required")
}

func TestHandleRequest_MethodNotFound_WithErrorData(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "unknown/method",
	}

	resp := server.handleRequest(req)

	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, ErrCodeMethodNotFound, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "unknown/method")

	// Verify error has method in data.
	data := resp.Error.Data.(map[string]string)
	assert.Equal(t, "unknown/method", data["method"])
}

// ============================================================================
// Status and Analytics Tests (PR-05)
// ============================================================================

func TestSetTransport(t *testing.T) {
	server := createTestServer()

	// Default is stdio.
	assert.NotNil(t, server.analytics)

	// Set to HTTP transport.
	server.SetTransport("http")

	// Analytics instance should be updated.
	assert.NotNil(t, server.analytics)
}

func TestHandleStatus(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "status",
	}

	resp := server.handleStatus(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)
}

func TestHandleRequest_Status(t *testing.T) {
	server := createTestServer()

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "status",
	}

	resp := server.handleRequest(req)

	assert.NotNil(t, resp)
	assert.Nil(t, resp.Error)
}

func TestGetStatusTracker(t *testing.T) {
	server := createTestServer()

	tracker := server.GetStatusTracker()

	assert.NotNil(t, tracker)
	assert.Equal(t, server.statusTracker, tracker)
}

func TestIsReady(t *testing.T) {
	server := createTestServer()

	// Server should be ready after NewServer is called with valid data.
	assert.True(t, server.IsReady())
}

func TestIsReady_NotReady(t *testing.T) {
	callGraph := core.NewCallGraph()
	moduleRegistry := &core.ModuleRegistry{
		Modules:      map[string]string{},
		FileToModule: map[string]string{},
		ShortNames:   map[string][]string{},
	}

	server := NewServer("/test", "3.11", callGraph, moduleRegistry, nil, time.Second)

	// Manually set status tracker to indexing to simulate not-ready state.
	server.statusTracker.StartIndexing()

	assert.False(t, server.IsReady())
}
