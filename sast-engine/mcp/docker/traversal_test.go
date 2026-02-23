package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTestGraph() *DependencyGraph {
	g := NewDependencyGraph()

	// Create a simple dependency chain: db → api → web → nginx
	g.AddNode(&DependencyNode{Name: "db", Type: "compose_service", File: "docker-compose.yml", LineNumber: 1})
	g.AddNode(&DependencyNode{Name: "api", Type: "compose_service", File: "docker-compose.yml", LineNumber: 5})
	g.AddNode(&DependencyNode{Name: "web", Type: "compose_service", File: "docker-compose.yml", LineNumber: 10})
	g.AddNode(&DependencyNode{Name: "nginx", Type: "compose_service", File: "docker-compose.yml", LineNumber: 15})

	g.AddEdge("api", "db")
	g.AddEdge("web", "api")
	g.AddEdge("nginx", "web")

	return g
}

func TestTraverse_Upstream(t *testing.T) {
	g := setupTestGraph()

	// Traverse upstream from nginx
	result := Traverse(g, "nginx", DirectionUpstream, 10)

	assert.Equal(t, "nginx", result.Target)
	assert.Equal(t, string(DirectionUpstream), result.Direction)
	assert.Len(t, result.Upstream, 3) // Should find web, api, db

	names := make([]string, 0, len(result.Upstream))
	for _, dep := range result.Upstream {
		names = append(names, dep.Name)
	}
	assert.Contains(t, names, "web")
	assert.Contains(t, names, "api")
	assert.Contains(t, names, "db")
}

func TestTraverse_Downstream(t *testing.T) {
	g := setupTestGraph()

	// Traverse downstream from db
	result := Traverse(g, "db", DirectionDownstream, 10)

	assert.Equal(t, "db", result.Target)
	assert.Equal(t, string(DirectionDownstream), result.Direction)
	assert.Len(t, result.Downstream, 3) // Should find api, web, nginx

	names := make([]string, 0, len(result.Downstream))
	for _, dep := range result.Downstream {
		names = append(names, dep.Name)
	}
	assert.Contains(t, names, "api")
	assert.Contains(t, names, "web")
	assert.Contains(t, names, "nginx")
}

func TestTraverse_Both(t *testing.T) {
	g := setupTestGraph()

	// Traverse both directions from web
	result := Traverse(g, "web", DirectionBoth, 10)

	assert.Equal(t, "web", result.Target)
	assert.Equal(t, string(DirectionBoth), result.Direction)
	assert.Len(t, result.Upstream, 2)   // Should find api, db
	assert.Len(t, result.Downstream, 1) // Should find nginx
}

func TestTraverse_MaxDepth(t *testing.T) {
	g := setupTestGraph()

	tests := []struct {
		name          string
		maxDepth      int
		expectedCount int
	}{
		{"Depth 1", 1, 1}, // Only web (direct dependency)
		{"Depth 2", 2, 2}, // web, api
		{"Depth 3", 3, 3}, // web, api, db
		{"Depth 0", 0, 0}, // No dependencies
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Traverse(g, "nginx", DirectionUpstream, tt.maxDepth)
			assert.Len(t, result.Upstream, tt.expectedCount)
		})
	}
}

func TestTraverse_NonExistent(t *testing.T) {
	g := setupTestGraph()

	// Try to traverse from non-existent node
	result := Traverse(g, "nonexistent", DirectionBoth, 10)

	assert.Equal(t, "nonexistent", result.Target)
	assert.Empty(t, result.Upstream)
	assert.Empty(t, result.Downstream)
	assert.Empty(t, result.File)
}

func TestTraverse_IsolatedNode(t *testing.T) {
	g := setupTestGraph()

	// Add isolated node
	g.AddNode(&DependencyNode{Name: "isolated", Type: "compose_service", File: "docker-compose.yml", LineNumber: 20})

	result := Traverse(g, "isolated", DirectionBoth, 10)

	assert.Equal(t, "isolated", result.Target)
	assert.Empty(t, result.Upstream)
	assert.Empty(t, result.Downstream)
}

func TestBuildDependencyChain(t *testing.T) {
	tests := []struct {
		name       string
		upstream   []DependencyInfo
		target     string
		downstream []DependencyInfo
		expected   string
	}{
		{
			name:       "Both directions",
			upstream:   []DependencyInfo{{Name: "db"}, {Name: "api"}},
			target:     "web",
			downstream: []DependencyInfo{{Name: "nginx"}},
			expected:   "api → db → web → nginx",
		},
		{
			name:       "Only upstream",
			upstream:   []DependencyInfo{{Name: "db"}},
			target:     "api",
			downstream: []DependencyInfo{},
			expected:   "db → api",
		},
		{
			name:       "Only downstream",
			upstream:   []DependencyInfo{},
			target:     "db",
			downstream: []DependencyInfo{{Name: "api"}},
			expected:   "db → api",
		},
		{
			name:       "Isolated node",
			upstream:   []DependencyInfo{},
			target:     "isolated",
			downstream: []DependencyInfo{},
			expected:   "isolated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDependencyChain(tt.upstream, tt.target, tt.downstream)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractRelevantMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata map[string]any
		expected map[string]any
	}{
		{
			name: "With stage info",
			metadata: map[string]any{
				"stage_name":  "builder",
				"stage_index": 0,
				"other_field": "ignored",
			},
			expected: map[string]any{
				"stage_name":  "builder",
				"stage_index": 0,
			},
		},
		{
			name: "Only stage index",
			metadata: map[string]any{
				"stage_index": 1,
			},
			expected: map[string]any{
				"stage_index": 1,
			},
		},
		{
			name: "No relevant metadata",
			metadata: map[string]any{
				"other_field": "value",
			},
			expected: nil,
		},
		{
			name:     "Nil metadata",
			metadata: nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRelevantMetadata(tt.metadata)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTraverseDirection_DepthTracking(t *testing.T) {
	g := setupTestGraph()

	// Traverse upstream from nginx
	result := traverseDirection(g, "nginx", true, 10)

	// Verify depth is tracked correctly
	depthMap := make(map[string]int)
	for _, dep := range result {
		depthMap[dep.Name] = dep.Depth
	}

	assert.Equal(t, 1, depthMap["web"]) // Direct dependency
	assert.Equal(t, 2, depthMap["api"]) // 2 hops away
	assert.Equal(t, 3, depthMap["db"])  // 3 hops away
}
