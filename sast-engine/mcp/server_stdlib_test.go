package mcp

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errMockStdlibNotFound is the sentinel error returned by the MCP test mock.
var errMockStdlibNotFound = errors.New("not found in mock")

// mockMCPStdlibLoader implements core.GoStdlibLoader for MCP-layer testing.
// Avoids network access and the extraction-layer mock re-declaration.
type mockMCPStdlibLoader struct {
	stdlibPkgs map[string]bool
	functions  map[string]*core.GoStdlibFunction // key: "importPath.funcName"
}

func (m *mockMCPStdlibLoader) ValidateStdlibImport(importPath string) bool {
	return m.stdlibPkgs[importPath]
}

func (m *mockMCPStdlibLoader) GetFunction(importPath, funcName string) (*core.GoStdlibFunction, error) {
	key := importPath + "." + funcName
	fn, ok := m.functions[key]
	if !ok {
		return nil, errMockStdlibNotFound
	}
	return fn, nil
}

func (m *mockMCPStdlibLoader) GetType(_, _ string) (*core.GoStdlibType, error) {
	return nil, errMockStdlibNotFound
}

func (m *mockMCPStdlibLoader) PackageCount() int {
	return len(m.stdlibPkgs)
}

// createGoTestServer builds a ready Server with a Go call graph containing
// both local and stdlib call sites for stdlib-enrichment tests.
func createGoTestServer() *Server {
	callGraph := core.NewCallGraph()

	callGraph.Functions["myapp.handler.Handle"] = &graph.Node{
		ID:         "1",
		Type:       "function_declaration",
		Name:       "Handle",
		File:       "/proj/handler/handler.go",
		LineNumber: 10,
	}

	callGraph.Functions["myapp.util.Helper"] = &graph.Node{
		ID:         "2",
		Type:       "function_declaration",
		Name:       "Helper",
		File:       "/proj/util/util.go",
		LineNumber: 5,
	}

	// Handle calls: net/http.Get (stdlib) and myapp.util.Helper (local).
	callGraph.Edges["myapp.handler.Handle"] = []string{
		"net/http.Get",
		"myapp.util.Helper",
	}
	callGraph.ReverseEdges["myapp.util.Helper"] = []string{"myapp.handler.Handle"}

	callGraph.CallSites["myapp.handler.Handle"] = []core.CallSite{
		{
			Target:    "Get",
			TargetFQN: "net/http.Get",
			Location:  core.Location{File: "/proj/handler/handler.go", Line: 15, Column: 4},
			Resolved:  true,
			IsStdlib:  true,
		},
		{
			Target:    "Helper",
			TargetFQN: "myapp.util.Helper",
			Location:  core.Location{File: "/proj/handler/handler.go", Line: 20, Column: 4},
			Resolved:  true,
			IsStdlib:  false,
		},
	}

	moduleRegistry := &core.ModuleRegistry{
		Modules:      map[string]string{"myapp.handler": "/proj/handler/handler.go", "myapp.util": "/proj/util/util.go"},
		FileToModule: map[string]string{"/proj/handler/handler.go": "myapp.handler", "/proj/util/util.go": "myapp.util"},
		ShortNames:   map[string][]string{"handler": {"/proj/handler/handler.go"}, "util": {"/proj/util/util.go"}},
	}

	return NewServer("/proj", "", callGraph, moduleRegistry, nil, time.Second, false)
}

// withStdlibLoader attaches a stdlib loader to the server's goModuleRegistry (mutates in place).
func withStdlibLoader(s *Server, loader core.GoStdlibLoader) {
	reg := core.NewGoModuleRegistry()
	reg.StdlibLoader = loader
	s.goModuleRegistry = reg
}

// =============================================================================
// SetGoContext
// =============================================================================

func TestSetGoContext_SetsFields(t *testing.T) {
	server := createTestServer()

	reg := core.NewGoModuleRegistry()
	reg.ModulePath = "github.com/myapp"
	server.SetGoContext("1.22", reg)

	assert.Equal(t, "1.22", server.goVersion)
	assert.Equal(t, reg, server.goModuleRegistry)
}

func TestSetGoContext_NilRegistryAllowed(t *testing.T) {
	server := createTestServer()
	// Must not panic when nil registry is passed (graceful degradation).
	assert.NotPanics(t, func() {
		server.SetGoContext("1.21", nil)
	})
	assert.Equal(t, "1.21", server.goVersion)
	assert.Nil(t, server.goModuleRegistry)
}

// =============================================================================
// stdlibInfoForFQN
// =============================================================================

func TestStdlibInfoForFQN_NilRegistry(t *testing.T) {
	server := createTestServer()
	// goModuleRegistry is nil by default on createTestServer.
	result := server.stdlibInfoForFQN("net/http.Get")
	assert.Nil(t, result)
}

func TestStdlibInfoForFQN_NilLoader(t *testing.T) {
	server := createTestServer()
	// Registry without a loader.
	server.goModuleRegistry = core.NewGoModuleRegistry()
	// StdlibLoader is nil by default.
	result := server.stdlibInfoForFQN("net/http.Get")
	assert.Nil(t, result)
}

func TestStdlibInfoForFQN_NoDotInFQN(t *testing.T) {
	server := createGoTestServer()
	withStdlibLoader(server, &mockMCPStdlibLoader{
		stdlibPkgs: map[string]bool{"fmt": true},
	})
	// FQN with no dot — cannot split importPath/funcName.
	result := server.stdlibInfoForFQN("noDot")
	assert.Nil(t, result)
}

func TestStdlibInfoForFQN_NotStdlibPackage(t *testing.T) {
	server := createGoTestServer()
	withStdlibLoader(server, &mockMCPStdlibLoader{
		stdlibPkgs: map[string]bool{"fmt": true},
	})
	result := server.stdlibInfoForFQN("github.com/gin-gonic/gin.Context")
	assert.Nil(t, result)
}

func TestStdlibInfoForFQN_FunctionNotFound(t *testing.T) {
	server := createGoTestServer()
	withStdlibLoader(server, &mockMCPStdlibLoader{
		stdlibPkgs: map[string]bool{"fmt": true},
		functions:  map[string]*core.GoStdlibFunction{},
	})
	// Package is stdlib but function missing — still returns package info.
	result := server.stdlibInfoForFQN("fmt.NonExistent")
	require.NotNil(t, result)
	assert.Equal(t, "fmt", result["package"])
	assert.NotContains(t, result, "signature")
	assert.NotContains(t, result, "return_types")
}

func TestStdlibInfoForFQN_FunctionFoundWithSignature(t *testing.T) {
	server := createGoTestServer()
	withStdlibLoader(server, &mockMCPStdlibLoader{
		stdlibPkgs: map[string]bool{"net/http": true},
		functions: map[string]*core.GoStdlibFunction{
			"net/http.Get": {
				Name:      "Get",
				Signature: "func Get(url string) (resp *Response, err error)",
				Returns: []*core.GoReturnValue{
					{Type: "*Response"},
					{Type: "error"},
				},
			},
		},
	})
	result := server.stdlibInfoForFQN("net/http.Get")
	require.NotNil(t, result)
	assert.Equal(t, "net/http", result["package"])
	assert.Equal(t, "func Get(url string) (resp *Response, err error)", result["signature"])
	retTypes, ok := result["return_types"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"*Response", "error"}, retTypes)
}

func TestStdlibInfoForFQN_EmptyReturnTypes(t *testing.T) {
	// Function with no return types (e.g., fmt.Println minus simplification).
	server := createGoTestServer()
	withStdlibLoader(server, &mockMCPStdlibLoader{
		stdlibPkgs: map[string]bool{"fmt": true},
		functions: map[string]*core.GoStdlibFunction{
			"fmt.Println": {
				Name:      "Println",
				Signature: "func Println(a ...any) (n int, err error)",
				Returns:   []*core.GoReturnValue{},
			},
		},
	})
	result := server.stdlibInfoForFQN("fmt.Println")
	require.NotNil(t, result)
	assert.Equal(t, "fmt", result["package"])
	assert.Equal(t, "func Println(a ...any) (n int, err error)", result["signature"])
	// No return_types key because Returns is empty.
	assert.NotContains(t, result, "return_types")
}

func TestStdlibInfoForFQN_BlankReturnTypeSkipped(t *testing.T) {
	// Returns with blank Type string should be omitted from return_types.
	server := createGoTestServer()
	withStdlibLoader(server, &mockMCPStdlibLoader{
		stdlibPkgs: map[string]bool{"os": true},
		functions: map[string]*core.GoStdlibFunction{
			"os.Open": {
				Name: "Open",
				Returns: []*core.GoReturnValue{
					{Type: "*File"},
					{Type: ""}, // blank — should be skipped
				},
			},
		},
	})
	result := server.stdlibInfoForFQN("os.Open")
	require.NotNil(t, result)
	retTypes, ok := result["return_types"].([]string)
	require.True(t, ok)
	assert.Equal(t, []string{"*File"}, retTypes)
}

// =============================================================================
// toolGetCallees — is_stdlib + stdlib_info
// =============================================================================

func TestToolGetCallees_IsStdlibField(t *testing.T) {
	server := createGoTestServer()

	resultStr, isError := server.toolGetCallees(map[string]interface{}{"function": "Handle"})
	require.False(t, isError)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(resultStr), &result))

	callees, ok := result["callees"].([]interface{})
	require.True(t, ok)
	require.Len(t, callees, 2)

	// First callee is net/http.Get (stdlib).
	first := callees[0].(map[string]interface{})
	assert.Equal(t, true, first["is_stdlib"])

	// Second callee is myapp.util.Helper (not stdlib).
	second := callees[1].(map[string]interface{})
	assert.Equal(t, false, second["is_stdlib"])
}

func TestToolGetCallees_StdlibInfo_WithLoader(t *testing.T) {
	server := createGoTestServer()
	withStdlibLoader(server, &mockMCPStdlibLoader{
		stdlibPkgs: map[string]bool{"net/http": true},
		functions: map[string]*core.GoStdlibFunction{
			"net/http.Get": {
				Name:      "Get",
				Signature: "func Get(url string) (resp *Response, err error)",
				Returns: []*core.GoReturnValue{
					{Type: "*Response"},
					{Type: "error"},
				},
			},
		},
	})

	resultStr, isError := server.toolGetCallees(map[string]interface{}{"function": "Handle"})
	require.False(t, isError)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(resultStr), &result))

	callees := result["callees"].([]interface{})
	first := callees[0].(map[string]interface{})

	assert.Equal(t, true, first["is_stdlib"])
	stdlibInfo, ok := first["stdlib_info"].(map[string]interface{})
	require.True(t, ok, "stdlib_info should be present for stdlib callee")
	assert.Equal(t, "net/http", stdlibInfo["package"])
	assert.Contains(t, stdlibInfo["signature"], "Get")
}

func TestToolGetCallees_NoStdlibInfo_WhenNilLoader(t *testing.T) {
	// Without a StdlibLoader, is_stdlib is still surfaced but no stdlib_info block.
	server := createGoTestServer()
	// goModuleRegistry is nil — stdlib_info unavailable.

	resultStr, isError := server.toolGetCallees(map[string]interface{}{"function": "Handle"})
	require.False(t, isError)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(resultStr), &result))

	callees := result["callees"].([]interface{})
	first := callees[0].(map[string]interface{})

	assert.Equal(t, true, first["is_stdlib"])
	assert.NotContains(t, first, "stdlib_info")
}

func TestToolGetCallees_LocalCalleeNoStdlibInfo(t *testing.T) {
	server := createGoTestServer()
	withStdlibLoader(server, &mockMCPStdlibLoader{
		stdlibPkgs: map[string]bool{"net/http": true},
	})

	resultStr, isError := server.toolGetCallees(map[string]interface{}{"function": "Handle"})
	require.False(t, isError)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(resultStr), &result))

	callees := result["callees"].([]interface{})
	second := callees[1].(map[string]interface{})

	assert.Equal(t, false, second["is_stdlib"])
	assert.NotContains(t, second, "stdlib_info")
}

// =============================================================================
// toolGetCallDetails — is_stdlib + stdlib_info in resolution
// =============================================================================

func TestToolGetCallDetails_IsStdlib(t *testing.T) {
	server := createGoTestServer()

	resultStr, isError := server.toolGetCallDetails("Handle", "Get")
	require.False(t, isError)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(resultStr), &result))

	cs := result["call_site"].(map[string]interface{})
	resolution := cs["resolution"].(map[string]interface{})
	assert.Equal(t, true, resolution["is_stdlib"])
}

func TestToolGetCallDetails_StdlibInfo_WithLoader(t *testing.T) {
	server := createGoTestServer()
	withStdlibLoader(server, &mockMCPStdlibLoader{
		stdlibPkgs: map[string]bool{"net/http": true},
		functions: map[string]*core.GoStdlibFunction{
			"net/http.Get": {
				Name:      "Get",
				Signature: "func Get(url string) (resp *Response, err error)",
			},
		},
	})

	resultStr, isError := server.toolGetCallDetails("Handle", "Get")
	require.False(t, isError)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(resultStr), &result))

	cs := result["call_site"].(map[string]interface{})
	resolution := cs["resolution"].(map[string]interface{})
	assert.Equal(t, true, resolution["is_stdlib"])
	stdlibInfo, ok := resolution["stdlib_info"].(map[string]interface{})
	require.True(t, ok, "stdlib_info should be present when loader available")
	assert.Equal(t, "net/http", stdlibInfo["package"])
}

func TestToolGetCallDetails_NotStdlib_NoStdlibInfo(t *testing.T) {
	server := createGoTestServer()
	withStdlibLoader(server, &mockMCPStdlibLoader{
		stdlibPkgs: map[string]bool{"net/http": true},
	})

	resultStr, isError := server.toolGetCallDetails("Handle", "Helper")
	require.False(t, isError)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(resultStr), &result))

	cs := result["call_site"].(map[string]interface{})
	resolution := cs["resolution"].(map[string]interface{})
	assert.Equal(t, false, resolution["is_stdlib"])
	assert.NotContains(t, resolution, "stdlib_info")
}

// =============================================================================
// toolGetCallers — is_stdlib on call site
// =============================================================================

func TestToolGetCallers_IsStdlibOnCallSite(t *testing.T) {
	// Construct a server where a local function calls a "stdlib-tagged" function
	// and that same local function is in the callers list of another local function.
	// We test that is_stdlib surfaces in the call site metadata within callers.
	callGraph := core.NewCallGraph()

	callGraph.Functions["myapp.svc.Service"] = &graph.Node{
		ID: "1", Type: "function_declaration", Name: "Service",
		File: "/proj/svc.go", LineNumber: 1,
	}
	callGraph.Functions["myapp.svc.Sub"] = &graph.Node{
		ID: "2", Type: "function_declaration", Name: "Sub",
		File: "/proj/svc.go", LineNumber: 10,
	}

	// Service calls Sub (not stdlib) and Sub is the target we query callers for.
	callGraph.ReverseEdges["myapp.svc.Sub"] = []string{"myapp.svc.Service"}
	callGraph.CallSites["myapp.svc.Service"] = []core.CallSite{
		{
			Target:    "Sub",
			TargetFQN: "myapp.svc.Sub",
			Location:  core.Location{File: "/proj/svc.go", Line: 5},
			Resolved:  true,
			IsStdlib:  false,
		},
	}

	moduleRegistry := &core.ModuleRegistry{
		Modules:      map[string]string{"myapp.svc": "/proj/svc.go"},
		FileToModule: map[string]string{"/proj/svc.go": "myapp.svc"},
		ShortNames:   map[string][]string{"svc": {"/proj/svc.go"}},
	}

	server := NewServer("/proj", "", callGraph, moduleRegistry, nil, time.Second, false)

	resultStr, isError := server.toolGetCallers(map[string]interface{}{"function": "Sub"})
	require.False(t, isError)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(resultStr), &result))

	callers := result["callers"].([]interface{})
	require.Len(t, callers, 1)
	caller := callers[0].(map[string]interface{})
	// Non-stdlib call site: is_stdlib key should NOT be present (only added when true).
	assert.NotContains(t, caller, "is_stdlib")
}

func TestToolGetCallers_IsStdlibTrueOnCallSite(t *testing.T) {
	// Build a graph where the caller's call site to the target is flagged IsStdlib.
	// This is an unusual but valid case (target could be misclassified or future stdlib).
	callGraph := core.NewCallGraph()

	callGraph.Functions["myapp.svc.Target"] = &graph.Node{
		ID: "t", Type: "function_declaration", Name: "Target",
		File: "/proj/svc.go", LineNumber: 1,
	}
	callGraph.Functions["myapp.svc.Caller"] = &graph.Node{
		ID: "c", Type: "function_declaration", Name: "Caller",
		File: "/proj/svc.go", LineNumber: 10,
	}

	callGraph.ReverseEdges["myapp.svc.Target"] = []string{"myapp.svc.Caller"}
	callGraph.CallSites["myapp.svc.Caller"] = []core.CallSite{
		{
			Target:    "Target",
			TargetFQN: "myapp.svc.Target",
			Location:  core.Location{File: "/proj/svc.go", Line: 15},
			Resolved:  true,
			IsStdlib:  true,
		},
	}

	moduleRegistry := &core.ModuleRegistry{
		Modules:      map[string]string{"myapp.svc": "/proj/svc.go"},
		FileToModule: map[string]string{"/proj/svc.go": "myapp.svc"},
		ShortNames:   map[string][]string{},
	}

	server := NewServer("/proj", "", callGraph, moduleRegistry, nil, time.Second, false)

	resultStr, isError := server.toolGetCallers(map[string]interface{}{"function": "Target"})
	require.False(t, isError)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(resultStr), &result))

	callers := result["callers"].([]interface{})
	require.Len(t, callers, 1)
	caller := callers[0].(map[string]interface{})
	assert.Equal(t, true, caller["is_stdlib"])
}

// =============================================================================
// Integration: MCP JSON-RPC round-trip with stdlib metadata
// =============================================================================

func TestHandleToolsCall_GetCallees_StdlibRoundTrip(t *testing.T) {
	server := createGoTestServer()
	withStdlibLoader(server, &mockMCPStdlibLoader{
		stdlibPkgs: map[string]bool{"net/http": true},
		functions: map[string]*core.GoStdlibFunction{
			"net/http.Get": {
				Name:      "Get",
				Signature: "func Get(url string) (resp *Response, err error)",
			},
		},
	})

	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"get_callees","arguments":{"function":"Handle"}}`),
	}

	resp := server.handleToolsCall(req)
	require.NotNil(t, resp)

	toolResult, ok := resp.Result.(ToolResult)
	require.True(t, ok)
	assert.False(t, toolResult.IsError)
	assert.Contains(t, toolResult.Content[0].Text, "is_stdlib")
	assert.Contains(t, toolResult.Content[0].Text, "stdlib_info")
	assert.Contains(t, toolResult.Content[0].Text, "net/http")
}
