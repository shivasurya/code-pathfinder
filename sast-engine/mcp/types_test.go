package mcp

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// JSON-RPC 2.0 Types Tests
// ============================================================================

func TestJSONRPCRequest_Serialization(t *testing.T) {
	tests := []struct {
		name     string
		request  JSONRPCRequest
		expected string
	}{
		{
			name: "basic request",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      1,
				Method:  "initialize",
			},
			expected: `{"jsonrpc":"2.0","id":1,"method":"initialize"}`,
		},
		{
			name: "request with params",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      "abc-123",
				Method:  "tools/call",
				Params:  json.RawMessage(`{"name":"get_index_info"}`),
			},
			expected: `{"jsonrpc":"2.0","id":"abc-123","method":"tools/call","params":{"name":"get_index_info"}}`,
		},
		{
			name: "request with null id",
			request: JSONRPCRequest{
				JSONRPC: "2.0",
				ID:      nil,
				Method:  "notification",
			},
			expected: `{"jsonrpc":"2.0","id":null,"method":"notification"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.request)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

func TestJSONRPCRequest_Deserialization(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectedMethod string
		expectedID     any
	}{
		{
			name:           "integer id",
			input:          `{"jsonrpc":"2.0","id":42,"method":"ping"}`,
			expectedMethod: "ping",
			expectedID:     float64(42), // JSON numbers decode as float64.
		},
		{
			name:           "string id",
			input:          `{"jsonrpc":"2.0","id":"request-1","method":"tools/list"}`,
			expectedMethod: "tools/list",
			expectedID:     "request-1",
		},
		{
			name:           "with params",
			input:          `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"find_symbol","arguments":{"name":"test"}}}`,
			expectedMethod: "tools/call",
			expectedID:     float64(1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req JSONRPCRequest
			err := json.Unmarshal([]byte(tt.input), &req)
			require.NoError(t, err)
			assert.Equal(t, "2.0", req.JSONRPC)
			assert.Equal(t, tt.expectedMethod, req.Method)
			assert.Equal(t, tt.expectedID, req.ID)
		})
	}
}

func TestJSONRPCResponse_Serialization(t *testing.T) {
	tests := []struct {
		name     string
		response JSONRPCResponse
	}{
		{
			name: "success response",
			response: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  map[string]string{"status": "ok"},
			},
		},
		{
			name: "error response",
			response: JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error:   &RPCError{Code: -32600, Message: "Invalid Request"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.response)
			require.NoError(t, err)

			// Verify it's valid JSON.
			var parsed map[string]any
			err = json.Unmarshal(data, &parsed)
			require.NoError(t, err)
			assert.Equal(t, "2.0", parsed["jsonrpc"])
		})
	}
}

func TestJSONRPCResponse_Deserialization(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		input := `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`
		var resp JSONRPCResponse
		err := json.Unmarshal([]byte(input), &resp)
		require.NoError(t, err)
		assert.Equal(t, "2.0", resp.JSONRPC)
		assert.Equal(t, float64(1), resp.ID)
		assert.NotNil(t, resp.Result)
		assert.Nil(t, resp.Error)
	})

	t.Run("error response", func(t *testing.T) {
		input := `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`
		var resp JSONRPCResponse
		err := json.Unmarshal([]byte(input), &resp)
		require.NoError(t, err)
		assert.Equal(t, "2.0", resp.JSONRPC)
		assert.Nil(t, resp.Result)
		require.NotNil(t, resp.Error)
		assert.Equal(t, -32601, resp.Error.Code)
		assert.Equal(t, "Method not found", resp.Error.Message)
	})
}

func TestRPCError_Serialization(t *testing.T) {
	tests := []struct {
		name     string
		error    RPCError
		expected string
	}{
		{
			name:     "simple error",
			error:    RPCError{Code: -32600, Message: "Invalid Request"},
			expected: `{"code":-32600,"message":"Invalid Request"}`,
		},
		{
			name:     "error with data",
			error:    RPCError{Code: -32602, Message: "Invalid params", Data: "name is required"},
			expected: `{"code":-32602,"message":"Invalid params","data":"name is required"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.error)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}

// ============================================================================
// MCP Protocol Types Tests
// ============================================================================

func TestInitializeParams_Serialization(t *testing.T) {
	params := InitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo: ClientInfo{
			Name:    "claude-code",
			Version: "1.0.0",
		},
	}

	data, err := json.Marshal(params)
	require.NoError(t, err)

	expected := `{"protocolVersion":"2024-11-05","clientInfo":{"name":"claude-code","version":"1.0.0"}}`
	assert.JSONEq(t, expected, string(data))
}

func TestInitializeParams_Deserialization(t *testing.T) {
	input := `{"protocolVersion":"2024-11-05","clientInfo":{"name":"test-client","version":"0.1.0"}}`

	var params InitializeParams
	err := json.Unmarshal([]byte(input), &params)
	require.NoError(t, err)
	assert.Equal(t, "2024-11-05", params.ProtocolVersion)
	assert.Equal(t, "test-client", params.ClientInfo.Name)
	assert.Equal(t, "0.1.0", params.ClientInfo.Version)
}

func TestInitializeResult_Serialization(t *testing.T) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo: ServerInfo{
			Name:    "pathfinder",
			Version: "0.1.0-poc",
		},
		Capabilities: Capabilities{
			Tools: &ToolsCapability{ListChanged: true},
		},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "2024-11-05", parsed["protocolVersion"])
	assert.NotNil(t, parsed["serverInfo"])
	assert.NotNil(t, parsed["capabilities"])
}

func TestCapabilities_EmptyTools(t *testing.T) {
	caps := Capabilities{}

	data, err := json.Marshal(caps)
	require.NoError(t, err)
	// tools should be omitted when nil.
	assert.Equal(t, `{}`, string(data))
}

func TestCapabilities_WithTools(t *testing.T) {
	caps := Capabilities{
		Tools: &ToolsCapability{ListChanged: false},
	}

	data, err := json.Marshal(caps)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.NotNil(t, parsed["tools"])
}

// ============================================================================
// Tool Types Tests
// ============================================================================

func TestTool_Serialization(t *testing.T) {
	tool := Tool{
		Name:        "get_index_info",
		Description: "Get index information",
		InputSchema: InputSchema{
			Type:       "object",
			Properties: map[string]Property{},
			Required:   []string{},
		},
	}

	data, err := json.Marshal(tool)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "get_index_info", parsed["name"])
	assert.Equal(t, "Get index information", parsed["description"])
	assert.NotNil(t, parsed["inputSchema"])
}

func TestTool_WithProperties(t *testing.T) {
	tool := Tool{
		Name:        "find_symbol",
		Description: "Find a symbol by name",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"name": {Type: "string", Description: "Symbol name to search for"},
			},
			Required: []string{"name"},
		},
	}

	data, err := json.Marshal(tool)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	schema := parsed["inputSchema"].(map[string]any)
	props := schema["properties"].(map[string]any)
	assert.NotNil(t, props["name"])

	required := schema["required"].([]any)
	assert.Equal(t, "name", required[0])
}

func TestToolsListResult_Serialization(t *testing.T) {
	result := ToolsListResult{
		Tools: []Tool{
			{Name: "tool1", Description: "First tool", InputSchema: InputSchema{Type: "object"}},
			{Name: "tool2", Description: "Second tool", InputSchema: InputSchema{Type: "object"}},
		},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	tools := parsed["tools"].([]any)
	assert.Len(t, tools, 2)
}

func TestToolsListResult_Empty(t *testing.T) {
	result := ToolsListResult{Tools: []Tool{}}

	data, err := json.Marshal(result)
	require.NoError(t, err)
	assert.JSONEq(t, `{"tools":[]}`, string(data))
}

func TestToolCallParams_Serialization(t *testing.T) {
	params := ToolCallParams{
		Name: "find_symbol",
		Arguments: map[string]any{
			"name": "validate_user",
		},
	}

	data, err := json.Marshal(params)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "find_symbol", parsed["name"])
	assert.NotNil(t, parsed["arguments"])
}

func TestToolCallParams_Deserialization(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedName string
		expectedArgs map[string]any
	}{
		{
			name:         "with arguments",
			input:        `{"name":"get_callers","arguments":{"function":"login"}}`,
			expectedName: "get_callers",
			expectedArgs: map[string]any{"function": "login"},
		},
		{
			name:         "without arguments",
			input:        `{"name":"get_index_info"}`,
			expectedName: "get_index_info",
			expectedArgs: nil,
		},
		{
			name:         "empty arguments",
			input:        `{"name":"get_index_info","arguments":{}}`,
			expectedName: "get_index_info",
			expectedArgs: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var params ToolCallParams
			err := json.Unmarshal([]byte(tt.input), &params)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedName, params.Name)
			if tt.expectedArgs == nil {
				assert.Nil(t, params.Arguments)
			} else {
				assert.Equal(t, tt.expectedArgs, params.Arguments)
			}
		})
	}
}

func TestToolResult_Serialization(t *testing.T) {
	tests := []struct {
		name   string
		result ToolResult
	}{
		{
			name: "success result",
			result: ToolResult{
				Content: []ContentBlock{{Type: "text", Text: "Success"}},
				IsError: false,
			},
		},
		{
			name: "error result",
			result: ToolResult{
				Content: []ContentBlock{{Type: "text", Text: "Error occurred"}},
				IsError: true,
			},
		},
		{
			name: "multiple content blocks",
			result: ToolResult{
				Content: []ContentBlock{
					{Type: "text", Text: "Line 1"},
					{Type: "text", Text: "Line 2"},
				},
				IsError: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			require.NoError(t, err)

			var parsed map[string]any
			err = json.Unmarshal(data, &parsed)
			require.NoError(t, err)

			content := parsed["content"].([]any)
			assert.Len(t, content, len(tt.result.Content))
		})
	}
}

func TestToolResult_IsErrorOmitted(t *testing.T) {
	result := ToolResult{
		Content: []ContentBlock{{Type: "text", Text: "Success"}},
		IsError: false,
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	// isError should be omitted when false.
	assert.NotContains(t, string(data), "isError")
}

func TestToolResult_IsErrorPresent(t *testing.T) {
	result := ToolResult{
		Content: []ContentBlock{{Type: "text", Text: "Error"}},
		IsError: true,
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	// isError should be present when true.
	assert.Contains(t, string(data), "isError")
}

func TestContentBlock_Serialization(t *testing.T) {
	block := ContentBlock{
		Type: "text",
		Text: "Hello, world!",
	}

	data, err := json.Marshal(block)
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"text","text":"Hello, world!"}`, string(data))
}

func TestContentBlock_SpecialCharacters(t *testing.T) {
	block := ContentBlock{
		Type: "text",
		Text: `{"key": "value with \"quotes\" and \n newlines"}`,
	}

	data, err := json.Marshal(block)
	require.NoError(t, err)

	// Verify round-trip.
	var parsed ContentBlock
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, block.Text, parsed.Text)
}

// ============================================================================
// Helper Functions Tests
// ============================================================================

func TestSuccessResponse(t *testing.T) {
	tests := []struct {
		name   string
		id     any
		result any
	}{
		{
			name:   "integer id",
			id:     1,
			result: map[string]string{"status": "ok"},
		},
		{
			name:   "string id",
			id:     "request-1",
			result: []string{"item1", "item2"},
		},
		{
			name:   "nil result",
			id:     1,
			result: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := SuccessResponse(tt.id, tt.result)

			assert.Equal(t, "2.0", resp.JSONRPC)
			assert.Equal(t, tt.id, resp.ID)
			assert.Equal(t, tt.result, resp.Result)
			assert.Nil(t, resp.Error)
		})
	}
}

func TestErrorResponse(t *testing.T) {
	tests := []struct {
		name        string
		id          any
		code        int
		message     string
		expectedErr *RPCError
	}{
		{
			name:        "parse error",
			id:          nil,
			code:        -32700,
			message:     "Parse error",
			expectedErr: &RPCError{Code: -32700, Message: "Parse error"},
		},
		{
			name:        "invalid request",
			id:          1,
			code:        -32600,
			message:     "Invalid Request",
			expectedErr: &RPCError{Code: -32600, Message: "Invalid Request"},
		},
		{
			name:        "method not found",
			id:          "req-123",
			code:        -32601,
			message:     "Method not found",
			expectedErr: &RPCError{Code: -32601, Message: "Method not found"},
		},
		{
			name:        "invalid params",
			id:          1,
			code:        -32602,
			message:     "Invalid params",
			expectedErr: &RPCError{Code: -32602, Message: "Invalid params"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := ErrorResponse(tt.id, tt.code, tt.message)

			assert.Equal(t, "2.0", resp.JSONRPC)
			assert.Equal(t, tt.id, resp.ID)
			assert.Nil(t, resp.Result)
			require.NotNil(t, resp.Error)
			assert.Equal(t, tt.expectedErr.Code, resp.Error.Code)
			assert.Equal(t, tt.expectedErr.Message, resp.Error.Message)
		})
	}
}

func TestSuccessResponse_Serializable(t *testing.T) {
	resp := SuccessResponse(1, map[string]any{"tools": []Tool{}})

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "2.0", parsed["jsonrpc"])
	assert.Nil(t, parsed["error"])
}

func TestErrorResponse_Serializable(t *testing.T) {
	resp := ErrorResponse(1, -32601, "Method not found")

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Equal(t, "2.0", parsed["jsonrpc"])
	assert.NotNil(t, parsed["error"])

	errObj := parsed["error"].(map[string]any)
	assert.Equal(t, float64(-32601), errObj["code"])
	assert.Equal(t, "Method not found", errObj["message"])
}

// ============================================================================
// Round-Trip Tests
// ============================================================================

func TestJSONRPCRequest_RoundTrip(t *testing.T) {
	original := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "test-id-123",
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"find_symbol","arguments":{"name":"test"}}`),
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded JSONRPCRequest
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.JSONRPC, decoded.JSONRPC)
	assert.Equal(t, original.ID, decoded.ID)
	assert.Equal(t, original.Method, decoded.Method)
	assert.JSONEq(t, string(original.Params), string(decoded.Params))
}

func TestJSONRPCResponse_RoundTrip(t *testing.T) {
	original := SuccessResponse("id-456", InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo:      ServerInfo{Name: "pathfinder", Version: "0.1.0"},
		Capabilities:    Capabilities{Tools: &ToolsCapability{ListChanged: true}},
	})

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded JSONRPCResponse
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.JSONRPC, decoded.JSONRPC)
	assert.Equal(t, original.ID, decoded.ID)
	assert.Nil(t, decoded.Error)
}
