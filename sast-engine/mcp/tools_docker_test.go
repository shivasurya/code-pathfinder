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

// createDockerTestServer creates a Server with Docker test fixtures.
func createDockerTestServer() *Server {
	callGraph := core.NewCallGraph()

	// Add a few regular function nodes for baseline.
	callGraph.Functions["myapp.main"] = &graph.Node{
		ID:         "func1",
		Type:       "function_definition",
		Name:       "main",
		File:       "/test/app.py",
		LineNumber: 10,
	}

	moduleRegistry := &core.ModuleRegistry{
		Modules:      map[string]string{"myapp": "/test/app.py"},
		FileToModule: map[string]string{"/test/app.py": "myapp"},
		ShortNames:   map[string][]string{"myapp": {"/test/app.py"}},
	}

	// Create CodeGraph with Docker nodes.
	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{
			"docker1": {
				ID:         "docker1",
				Type:       "dockerfile_instruction",
				Name:       "FROM",
				File:       "/test/Dockerfile",
				LineNumber: 1,
				MethodArgumentsValue: []string{
					"FROM python:3.11-slim",
					"python:3.11-slim",
				},
			},
			"docker2": {
				ID:         "docker2",
				Type:       "dockerfile_instruction",
				Name:       "RUN",
				File:       "/test/Dockerfile",
				LineNumber: 3,
				MethodArgumentsValue: []string{
					"RUN pip install -r requirements.txt",
				},
			},
			"docker3": {
				ID:         "docker3",
				Type:       "dockerfile_instruction",
				Name:       "USER",
				File:       "/test/Dockerfile",
				LineNumber: 8,
				MethodArgumentsValue: []string{
					"USER appuser:appgroup",
					"appuser",
					"appgroup",
				},
			},
			"docker4": {
				ID:         "docker4",
				Type:       "dockerfile_instruction",
				Name:       "EXPOSE",
				File:       "/test/Dockerfile",
				LineNumber: 9,
				MethodArgumentsValue: []string{
					"EXPOSE 8080/tcp",
					"8080",
				},
			},
			"compose1": {
				ID:         "compose1",
				Type:       "compose_service",
				Name:       "web",
				File:       "/test/docker-compose.yml",
				LineNumber: 3,
			},
			"compose2": {
				ID:         "compose2",
				Type:       "compose_service",
				Name:       "db",
				File:       "/test/docker-compose.yml",
				LineNumber: 10,
			},
		},
		Edges: []*graph.Edge{},
	}

	return NewServer("/test/project", "3.11", callGraph, moduleRegistry, codeGraph, time.Second, false)
}

// TestFindSymbol_DockerfileInstruction tests that Dockerfile instructions are queryable.
func TestFindSymbol_DockerfileInstruction(t *testing.T) {
	server := createDockerTestServer()

	result, isError := server.toolFindSymbol(map[string]any{
		"type": "dockerfile_instruction",
	})

	require.False(t, isError, "Expected no error")

	var parsed map[string]any
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err, "Expected valid JSON")

	matches, ok := parsed["matches"].([]any)
	require.True(t, ok, "Expected matches array")
	assert.Equal(t, 4, len(matches), "Expected 4 Dockerfile instructions")

	// Verify first match structure.
	match := matches[0].(map[string]any)
	assert.Contains(t, match, "fqn")
	assert.Contains(t, match, "file")
	assert.Contains(t, match, "line")
	assert.Equal(t, "dockerfile_instruction", match["type"])
	assert.Equal(t, float64(14), match["symbol_kind"]) // SymbolKindConstant
	assert.Equal(t, "DockerInstruction", match["symbol_kind_name"])
}

// TestFindSymbol_ComposeService tests that docker-compose services are queryable.
func TestFindSymbol_ComposeService(t *testing.T) {
	server := createDockerTestServer()

	result, isError := server.toolFindSymbol(map[string]any{
		"type": "compose_service",
	})

	require.False(t, isError, "Expected no error")

	var parsed map[string]any
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err, "Expected valid JSON")

	matches, ok := parsed["matches"].([]any)
	require.True(t, ok, "Expected matches array")
	assert.Equal(t, 2, len(matches), "Expected 2 compose services")

	// Verify first match structure.
	match := matches[0].(map[string]any)
	assert.Equal(t, "compose_service", match["type"])
	assert.Equal(t, float64(2), match["symbol_kind"]) // SymbolKindModule
	assert.Equal(t, "ComposeService", match["symbol_kind_name"])
}

// TestFindSymbol_DockerByName tests filtering Docker nodes by name.
func TestFindSymbol_DockerByName(t *testing.T) {
	server := createDockerTestServer()

	result, isError := server.toolFindSymbol(map[string]any{
		"type": "dockerfile_instruction",
		"name": "FROM",
	})

	require.False(t, isError, "Expected no error")

	var parsed map[string]any
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err, "Expected valid JSON")

	matches, ok := parsed["matches"].([]any)
	require.True(t, ok, "Expected matches array")
	assert.Equal(t, 1, len(matches), "Expected 1 FROM instruction")

	match := matches[0].(map[string]any)
	assert.Contains(t, match["fqn"], "FROM")
}

// TestFindSymbol_InvalidDockerType tests error handling for invalid Docker types.
func TestFindSymbol_InvalidDockerType(t *testing.T) {
	server := createDockerTestServer()

	result, isError := server.toolFindSymbol(map[string]any{
		"type": "invalid_docker_type",
	})

	require.True(t, isError, "Expected error for invalid type")
	assert.Contains(t, result, "Invalid symbol type")
	assert.Contains(t, result, "dockerfile_instruction")
	assert.Contains(t, result, "compose_service")
}

// TestGetSymbolKind_DockerTypes tests LSP Symbol Kind mapping for Docker types.
func TestGetSymbolKind_DockerTypes(t *testing.T) {
	tests := []struct {
		symbolType   string
		expectedKind int
		expectedName string
	}{
		{
			symbolType:   "dockerfile_instruction",
			expectedKind: SymbolKindConstant, // 14
			expectedName: "DockerInstruction",
		},
		{
			symbolType:   "compose_service",
			expectedKind: SymbolKindModule, // 2
			expectedName: "ComposeService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.symbolType, func(t *testing.T) {
			kind, name := getSymbolKind(tt.symbolType)
			assert.Equal(t, tt.expectedKind, kind, "Symbol kind mismatch")
			assert.Equal(t, tt.expectedName, name, "Symbol kind name mismatch")
		})
	}
}

// TestGetIndexInfo_DockerStats tests that Docker statistics are included in index info.
func TestGetIndexInfo_DockerStats(t *testing.T) {
	server := createDockerTestServer()

	result, isError := server.toolGetIndexInfo()

	require.False(t, isError, "Expected no error")

	var parsed map[string]any
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err, "Expected valid JSON")

	// Verify stats section includes Docker counts.
	stats, ok := parsed["stats"].(map[string]any)
	require.True(t, ok, "Expected stats object")

	// Check Docker instruction count.
	dockerInstructions, ok := stats["docker_instructions"].(float64)
	require.True(t, ok, "Expected docker_instructions field")
	assert.Equal(t, float64(4), dockerInstructions, "Expected 4 Dockerfile instructions")

	// Check compose service count.
	composeServices, ok := stats["compose_services"].(float64)
	require.True(t, ok, "Expected compose_services field")
	assert.Equal(t, float64(2), composeServices, "Expected 2 compose services")

	// Verify other stats are still present.
	assert.Contains(t, stats, "total_symbols")
	assert.Contains(t, stats, "call_edges")
	assert.Contains(t, stats, "modules")
	assert.Contains(t, stats, "files")
	assert.Contains(t, stats, "class_fields")
}

// TestGetIndexInfo_NoDockerNodes tests handling when no Docker nodes exist.
func TestGetIndexInfo_NoDockerNodes(t *testing.T) {
	// Create server without Docker nodes.
	callGraph := core.NewCallGraph()
	callGraph.Functions["test.func"] = &graph.Node{
		ID:         "1",
		Type:       "function_definition",
		Name:       "func",
		File:       "/test/app.py",
		LineNumber: 1,
	}

	moduleRegistry := &core.ModuleRegistry{
		Modules:      map[string]string{"test": "/test/app.py"},
		FileToModule: map[string]string{"/test/app.py": "test"},
	}

	// CodeGraph with no Docker nodes.
	codeGraph := &graph.CodeGraph{
		Nodes: map[string]*graph.Node{},
		Edges: []*graph.Edge{},
	}

	server := NewServer("/test/project", "3.11", callGraph, moduleRegistry, codeGraph, time.Second, false)

	result, isError := server.toolGetIndexInfo()

	require.False(t, isError, "Expected no error")

	var parsed map[string]any
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err, "Expected valid JSON")

	stats, ok := parsed["stats"].(map[string]any)
	require.True(t, ok, "Expected stats object")

	// Docker counts should be 0.
	assert.Equal(t, float64(0), stats["docker_instructions"])
	assert.Equal(t, float64(0), stats["compose_services"])
}

// TestGetIndexInfo_NilCodeGraph tests handling when CodeGraph is nil.
func TestGetIndexInfo_NilCodeGraph(t *testing.T) {
	callGraph := core.NewCallGraph()
	callGraph.Functions["test.func"] = &graph.Node{
		ID:         "1",
		Type:       "function_definition",
		Name:       "func",
		File:       "/test/app.py",
		LineNumber: 1,
	}

	moduleRegistry := &core.ModuleRegistry{
		Modules:      map[string]string{"test": "/test/app.py"},
		FileToModule: map[string]string{"/test/app.py": "test"},
	}

	// Pass nil CodeGraph.
	server := NewServer("/test/project", "3.11", callGraph, moduleRegistry, nil, time.Second, false)

	result, isError := server.toolGetIndexInfo()

	require.False(t, isError, "Expected no error")

	var parsed map[string]any
	err := json.Unmarshal([]byte(result), &parsed)
	require.NoError(t, err, "Expected valid JSON")

	stats, ok := parsed["stats"].(map[string]any)
	require.True(t, ok, "Expected stats object")

	// Docker counts should be 0 when CodeGraph is nil.
	assert.Equal(t, float64(0), stats["docker_instructions"])
	assert.Equal(t, float64(0), stats["compose_services"])
}
