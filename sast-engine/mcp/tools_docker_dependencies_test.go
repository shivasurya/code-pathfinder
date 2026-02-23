package mcp

import (
	"encoding/json"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	dockerpkg "github.com/shivasurya/code-pathfinder/sast-engine/mcp/docker"
	"github.com/stretchr/testify/assert"
)

func setupDockerDependenciesTest() *Server {
	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
	}

	// Add compose services
	codeGraph.AddNode(&graph.Node{
		ID:         "web-id",
		Type:       "compose_service",
		Name:       "web",
		File:       "docker-compose.yml",
		LineNumber: 5,
		Metadata: map[string]any{
			"depends_on": []any{"api", "redis"},
		},
	})

	codeGraph.AddNode(&graph.Node{
		ID:         "api-id",
		Type:       "compose_service",
		Name:       "api",
		File:       "docker-compose.yml",
		LineNumber: 12,
		Metadata: map[string]any{
			"depends_on": []any{"db"},
		},
	})

	codeGraph.AddNode(&graph.Node{
		ID:         "db-id",
		Type:       "compose_service",
		Name:       "db",
		File:       "docker-compose.yml",
		LineNumber: 20,
		Metadata:   map[string]any{},
	})

	codeGraph.AddNode(&graph.Node{
		ID:         "redis-id",
		Type:       "compose_service",
		Name:       "redis",
		File:       "docker-compose.yml",
		LineNumber: 25,
		Metadata:   map[string]any{},
	})

	// Add Dockerfile stages
	codeGraph.AddNode(&graph.Node{
		ID:         "from-builder",
		Type:       "dockerfile_instruction",
		Name:       "FROM",
		File:       "Dockerfile",
		LineNumber: 1,
		Metadata: map[string]any{
			"stage_index": 0,
			"stage_name":  "builder",
		},
	})

	codeGraph.AddNode(&graph.Node{
		ID:         "from-final",
		Type:       "dockerfile_instruction",
		Name:       "FROM",
		File:       "Dockerfile",
		LineNumber: 10,
		Metadata: map[string]any{
			"stage_index": 1,
		},
	})

	codeGraph.AddNode(&graph.Node{
		ID:         "copy-from-builder",
		Type:       "dockerfile_instruction",
		Name:       "COPY",
		File:       "Dockerfile",
		LineNumber: 11,
		Metadata: map[string]any{
			"copy_from":   "builder",
			"stage_index": 1,
		},
	})

	return &Server{
		codeGraph: codeGraph,
	}
}

func TestGetDockerDependencies_ComposeBasic(t *testing.T) {
	server := setupDockerDependenciesTest()

	args := map[string]any{
		"type": "compose",
		"name": "web",
	}

	result, success := server.toolGetDockerDependencies(args)
	assert.False(t, success) // Tool returns false for success

	var response dockerpkg.TraversalResult
	err := json.Unmarshal([]byte(result), &response)
	assert.NoError(t, err)

	assert.Equal(t, "web", response.Target)
	assert.Equal(t, "compose_service", response.Type)
	assert.Len(t, response.Upstream, 3) // api, db, redis

	// Verify upstream contains expected services
	upstreamNames := make([]string, len(response.Upstream))
	for i, dep := range response.Upstream {
		upstreamNames[i] = dep.Name
	}
	assert.Contains(t, upstreamNames, "api")
	assert.Contains(t, upstreamNames, "db")
	assert.Contains(t, upstreamNames, "redis")
}

func TestGetDockerDependencies_ComposeUpstream(t *testing.T) {
	server := setupDockerDependenciesTest()

	args := map[string]any{
		"type":      "compose",
		"name":      "web",
		"direction": "upstream",
	}

	result, _ := server.toolGetDockerDependencies(args)

	var response dockerpkg.TraversalResult
	err := json.Unmarshal([]byte(result), &response)
	assert.NoError(t, err)

	assert.Equal(t, "upstream", response.Direction)
	assert.NotEmpty(t, response.Upstream)
	assert.Empty(t, response.Downstream)
}

func TestGetDockerDependencies_ComposeDownstream(t *testing.T) {
	server := setupDockerDependenciesTest()

	args := map[string]any{
		"type":      "compose",
		"name":      "db",
		"direction": "downstream",
	}

	result, _ := server.toolGetDockerDependencies(args)

	var response dockerpkg.TraversalResult
	err := json.Unmarshal([]byte(result), &response)
	assert.NoError(t, err)

	assert.Equal(t, "downstream", response.Direction)
	assert.Empty(t, response.Upstream)
	assert.Len(t, response.Downstream, 2) // api, web

	downstreamNames := make([]string, len(response.Downstream))
	for i, dep := range response.Downstream {
		downstreamNames[i] = dep.Name
	}
	assert.Contains(t, downstreamNames, "api")
	assert.Contains(t, downstreamNames, "web")
}

func TestGetDockerDependencies_ComposeMaxDepth(t *testing.T) {
	server := setupDockerDependenciesTest()

	tests := []struct {
		name        string
		maxDepth    float64
		expectedMin int
		expectedMax int
	}{
		{"Depth 1", 1.0, 1, 2}, // Direct dependencies only (api, redis)
		{"Depth 2", 2.0, 2, 3}, // Up to 2 levels (api, redis, db)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]any{
				"type":      "compose",
				"name":      "web",
				"direction": "upstream",
				"max_depth": tt.maxDepth,
			}

			result, _ := server.toolGetDockerDependencies(args)

			var response dockerpkg.TraversalResult
			err := json.Unmarshal([]byte(result), &response)
			assert.NoError(t, err)

			assert.GreaterOrEqual(t, len(response.Upstream), tt.expectedMin)
			assert.LessOrEqual(t, len(response.Upstream), tt.expectedMax)
		})
	}
}

func TestGetDockerDependencies_DockerfileBasic(t *testing.T) {
	server := setupDockerDependenciesTest()

	args := map[string]any{
		"type":      "dockerfile",
		"name":      "stage-1",
		"file_path": "Dockerfile",
	}

	result, _ := server.toolGetDockerDependencies(args)

	var response dockerpkg.TraversalResult
	err := json.Unmarshal([]byte(result), &response)
	assert.NoError(t, err)

	assert.Equal(t, "stage-1", response.Target)
	assert.Equal(t, "dockerfile_stage", response.Type)
	assert.Len(t, response.Upstream, 1) // builder stage

	assert.Equal(t, "builder", response.Upstream[0].Name)
}

func TestGetDockerDependencies_DockerfileDownstream(t *testing.T) {
	server := setupDockerDependenciesTest()

	args := map[string]any{
		"type":      "dockerfile",
		"name":      "builder",
		"direction": "downstream",
	}

	result, _ := server.toolGetDockerDependencies(args)

	var response dockerpkg.TraversalResult
	err := json.Unmarshal([]byte(result), &response)
	assert.NoError(t, err)

	assert.Len(t, response.Downstream, 1) // stage-1 depends on builder
	assert.Equal(t, "stage-1", response.Downstream[0].Name)
}

func TestGetDockerDependencies_DependencyChain(t *testing.T) {
	server := setupDockerDependenciesTest()

	args := map[string]any{
		"type": "compose",
		"name": "web",
	}

	result, _ := server.toolGetDockerDependencies(args)

	var response dockerpkg.TraversalResult
	err := json.Unmarshal([]byte(result), &response)
	assert.NoError(t, err)

	assert.NotEmpty(t, response.DependencyChain)
	assert.Contains(t, response.DependencyChain, "web")
	assert.Contains(t, response.DependencyChain, "→") // Arrow separator
}

func TestGetDockerDependencies_NonExistent(t *testing.T) {
	server := setupDockerDependenciesTest()

	args := map[string]any{
		"type": "compose",
		"name": "nonexistent",
	}

	result, _ := server.toolGetDockerDependencies(args)

	var response dockerpkg.TraversalResult
	err := json.Unmarshal([]byte(result), &response)
	assert.NoError(t, err)

	assert.Equal(t, "nonexistent", response.Target)
	assert.Empty(t, response.Upstream)
	assert.Empty(t, response.Downstream)
}

func TestGetDockerDependencies_MissingType(t *testing.T) {
	server := setupDockerDependenciesTest()

	args := map[string]any{
		"name": "web",
	}

	result, success := server.toolGetDockerDependencies(args)
	assert.False(t, success)

	var response map[string]any
	err := json.Unmarshal([]byte(result), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}

func TestGetDockerDependencies_MissingName(t *testing.T) {
	server := setupDockerDependenciesTest()

	args := map[string]any{
		"type": "compose",
	}

	result, success := server.toolGetDockerDependencies(args)
	assert.False(t, success)

	var response map[string]any
	err := json.Unmarshal([]byte(result), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}

func TestGetDockerDependencies_InvalidType(t *testing.T) {
	server := setupDockerDependenciesTest()

	args := map[string]any{
		"type": "invalid",
		"name": "web",
	}

	result, success := server.toolGetDockerDependencies(args)
	assert.False(t, success)

	var response map[string]any
	err := json.Unmarshal([]byte(result), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}

func TestGetDockerDependencies_InvalidDirection(t *testing.T) {
	server := setupDockerDependenciesTest()

	args := map[string]any{
		"type":      "compose",
		"name":      "web",
		"direction": "invalid",
	}

	result, success := server.toolGetDockerDependencies(args)
	assert.False(t, success)

	var response map[string]any
	err := json.Unmarshal([]byte(result), &response)
	assert.NoError(t, err)
	assert.Contains(t, response, "error")
}

func TestGetDockerDependencies_FiltersApplied(t *testing.T) {
	server := setupDockerDependenciesTest()

	args := map[string]any{
		"type":      "compose",
		"name":      "web",
		"file_path": "docker-compose.yml",
		"direction": "upstream",
		"max_depth": 5.0,
	}

	result, _ := server.toolGetDockerDependencies(args)

	var response dockerpkg.TraversalResult
	err := json.Unmarshal([]byte(result), &response)
	assert.NoError(t, err)

	assert.NotNil(t, response.FiltersApplied)
	assert.Equal(t, "compose", response.FiltersApplied["type"])
	assert.Equal(t, "web", response.FiltersApplied["name"])
	assert.Equal(t, "docker-compose.yml", response.FiltersApplied["file_path"])
	assert.Equal(t, "upstream", response.FiltersApplied["direction"])
	assert.Equal(t, float64(5), response.FiltersApplied["max_depth"])
}
