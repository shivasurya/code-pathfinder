package docker

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/stretchr/testify/assert"
)

func TestDependencyGraph_AddNode(t *testing.T) {
	g := NewDependencyGraph()

	node := &DependencyNode{
		Name:       "web",
		Type:       "compose_service",
		File:       "docker-compose.yml",
		LineNumber: 5,
	}

	g.AddNode(node)

	assert.Len(t, g.Nodes, 1)
	assert.Equal(t, "web", g.Nodes["web"].Name)
}

func TestDependencyGraph_AddEdge(t *testing.T) {
	g := NewDependencyGraph()

	g.AddEdge("web", "api")
	g.AddEdge("web", "db")
	g.AddEdge("web", "api") // Duplicate should be ignored

	assert.Len(t, g.Edges["web"], 2)
	assert.Contains(t, g.Edges["web"], "api")
	assert.Contains(t, g.Edges["web"], "db")
}

func TestDependencyGraph_GetDependencies(t *testing.T) {
	g := NewDependencyGraph()

	g.AddEdge("web", "api")
	g.AddEdge("web", "db")

	deps := g.GetDependencies("web")
	assert.Len(t, deps, 2)
	assert.Contains(t, deps, "api")
	assert.Contains(t, deps, "db")

	// Non-existent node
	empty := g.GetDependencies("nonexistent")
	assert.Empty(t, empty)
}

func TestDependencyGraph_GetDependents(t *testing.T) {
	g := NewDependencyGraph()

	g.AddEdge("web", "api")
	g.AddEdge("worker", "api")

	dependents := g.GetDependents("api")
	assert.Len(t, dependents, 2)
	assert.Contains(t, dependents, "web")
	assert.Contains(t, dependents, "worker")
}

func TestBuildComposeGraph(t *testing.T) {
	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
	}

	// Add compose service nodes with dependencies
	webNode := &graph.Node{
		ID:         "web-id",
		Type:       "compose_service",
		Name:       "web",
		File:       "docker-compose.yml",
		LineNumber: 5,
		Metadata: map[string]any{
			"depends_on": []any{"api", "redis"},
		},
	}
	codeGraph.Nodes["web-id"] = webNode

	apiNode := &graph.Node{
		ID:         "api-id",
		Type:       "compose_service",
		Name:       "api",
		File:       "docker-compose.yml",
		LineNumber: 12,
		Metadata: map[string]any{
			"depends_on": []any{"db"},
		},
	}
	codeGraph.Nodes["api-id"] = apiNode

	dbNode := &graph.Node{
		ID:         "db-id",
		Type:       "compose_service",
		Name:       "db",
		File:       "docker-compose.yml",
		LineNumber: 20,
		Metadata:   map[string]any{},
	}
	codeGraph.Nodes["db-id"] = dbNode

	// Build dependency graph
	depGraph := BuildComposeGraph(codeGraph)

	// Verify nodes
	assert.Len(t, depGraph.Nodes, 3)
	assert.Contains(t, depGraph.Nodes, "web")
	assert.Contains(t, depGraph.Nodes, "api")
	assert.Contains(t, depGraph.Nodes, "db")

	// Verify edges
	assert.ElementsMatch(t, []string{"api", "redis"}, depGraph.GetDependencies("web"))
	assert.ElementsMatch(t, []string{"db"}, depGraph.GetDependencies("api"))
	assert.Empty(t, depGraph.GetDependencies("db"))
}

func TestBuildDockerfileGraph(t *testing.T) {
	codeGraph := &graph.CodeGraph{
		Nodes: make(map[string]*graph.Node),
	}

	// Add FROM instruction (stage 0: builder)
	fromNode1 := &graph.Node{
		ID:         "from-1",
		Type:       "dockerfile_instruction",
		Name:       "FROM",
		File:       "Dockerfile",
		LineNumber: 1,
		Metadata: map[string]any{
			"stage_index": 0,
			"stage_name":  "builder",
		},
	}
	codeGraph.Nodes["from-1"] = fromNode1

	// Add FROM instruction (stage 1: final)
	fromNode2 := &graph.Node{
		ID:         "from-2",
		Type:       "dockerfile_instruction",
		Name:       "FROM",
		File:       "Dockerfile",
		LineNumber: 10,
		Metadata: map[string]any{
			"stage_index": 1,
		},
	}
	codeGraph.Nodes["from-2"] = fromNode2

	// Add COPY --from instruction in stage 1
	copyNode := &graph.Node{
		ID:         "copy-1",
		Type:       "dockerfile_instruction",
		Name:       "COPY",
		File:       "Dockerfile",
		LineNumber: 11,
		Metadata: map[string]any{
			"copy_from":   "builder",
			"stage_index": 1,
		},
	}
	codeGraph.Nodes["copy-1"] = copyNode

	// Build dependency graph
	depGraph := BuildDockerfileGraph(codeGraph, "")

	// Verify nodes
	assert.Len(t, depGraph.Nodes, 2)
	assert.Contains(t, depGraph.Nodes, "builder")
	assert.Contains(t, depGraph.Nodes, "stage-1")

	// Verify edges: stage-1 depends on builder
	deps := depGraph.GetDependencies("stage-1")
	assert.Contains(t, deps, "builder")
}

func TestGetStageName(t *testing.T) {
	tests := []struct {
		name     string
		node     *graph.Node
		expected string
	}{
		{
			name: "Named stage",
			node: &graph.Node{
				Metadata: map[string]any{
					"stage_name":  "builder",
					"stage_index": 0,
				},
			},
			expected: "builder",
		},
		{
			name: "Unnamed stage",
			node: &graph.Node{
				Metadata: map[string]any{
					"stage_index": 2,
				},
			},
			expected: "stage-2",
		},
		{
			name: "No metadata",
			node: &graph.Node{
				Metadata: nil,
			},
			expected: "stage-0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStageName(tt.node)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatStageIndex(t *testing.T) {
	tests := []struct {
		index    int
		expected string
	}{
		{0, "stage-0"},
		{1, "stage-1"},
		{5, "stage-5"},
		{10, "stage-10"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatStageIndex(tt.index)
			assert.Equal(t, tt.expected, result)
		})
	}
}
