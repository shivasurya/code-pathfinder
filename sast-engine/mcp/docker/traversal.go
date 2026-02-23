package docker

import "strings"

// TraversalDirection specifies the direction for dependency traversal.
type TraversalDirection string

const (
	// DirectionUpstream traverses dependencies (what this entity depends on).
	DirectionUpstream TraversalDirection = "upstream"

	// DirectionDownstream traverses dependents (what depends on this entity).
	DirectionDownstream TraversalDirection = "downstream"

	// DirectionBoth traverses both upstream and downstream.
	DirectionBoth TraversalDirection = "both"
)

// TraversalResult contains the results of a dependency traversal.
type TraversalResult struct {
	Target          string
	Type            string
	File            string
	Line            uint32
	Direction       string
	MaxDepth        int
	Upstream        []DependencyInfo
	Downstream      []DependencyInfo
	DependencyChain string
	FiltersApplied  map[string]any
}

// DependencyInfo contains enriched information about a dependency.
type DependencyInfo struct {
	Name     string         `json:"name"`
	Type     string         `json:"type"`
	File     string         `json:"file"`
	Line     uint32         `json:"line"`
	Depth    int            `json:"depth"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Traverse performs dependency traversal from a starting node.
func Traverse(graph *DependencyGraph, startNode string, direction TraversalDirection, maxDepth int) *TraversalResult {
	result := &TraversalResult{
		Target:     startNode,
		MaxDepth:   maxDepth,
		Upstream:   []DependencyInfo{},
		Downstream: []DependencyInfo{},
	}

	// Find target node
	targetNode, exists := graph.Nodes[startNode]
	if !exists {
		return result
	}

	result.Type = targetNode.Type
	result.File = targetNode.File
	result.Line = targetNode.LineNumber

	// Traverse upstream if requested
	if direction == DirectionUpstream || direction == DirectionBoth {
		result.Upstream = traverseDirection(graph, startNode, true, maxDepth)
		result.Direction = string(DirectionUpstream)
	}

	// Traverse downstream if requested
	if direction == DirectionDownstream || direction == DirectionBoth {
		result.Downstream = traverseDirection(graph, startNode, false, maxDepth)
		result.Direction = string(DirectionDownstream)
	}

	// Set direction for "both" case
	if direction == DirectionBoth {
		result.Direction = string(DirectionBoth)
	}

	// Build dependency chain
	result.DependencyChain = buildDependencyChain(result.Upstream, startNode, result.Downstream)

	return result
}

// traverseDirection performs BFS traversal in the specified direction.
// If upstream is true, traverses dependencies; otherwise, traverses dependents.
func traverseDirection(graph *DependencyGraph, startNode string, upstream bool, maxDepth int) []DependencyInfo {
	if maxDepth <= 0 {
		return []DependencyInfo{}
	}

	visited := make(map[string]bool)
	result := []DependencyInfo{}

	// BFS queue: (node name, depth)
	type queueItem struct {
		name  string
		depth int
	}
	queue := []queueItem{{name: startNode, depth: 0}}
	visited[startNode] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		// Check depth limit
		if current.depth >= maxDepth {
			continue
		}

		// Get neighbors based on direction
		var neighbors []string
		if upstream {
			neighbors = graph.GetDependencies(current.name)
		} else {
			neighbors = graph.GetDependents(current.name)
		}

		// Process each neighbor
		for _, neighbor := range neighbors {
			if visited[neighbor] {
				continue
			}

			visited[neighbor] = true
			queue = append(queue, queueItem{name: neighbor, depth: current.depth + 1})

			// Add to result
			if node, exists := graph.Nodes[neighbor]; exists {
				info := DependencyInfo{
					Name:     node.Name,
					Type:     node.Type,
					File:     node.File,
					Line:     node.LineNumber,
					Depth:    current.depth + 1,
					Metadata: extractRelevantMetadata(node.Metadata),
				}
				result = append(result, info)
			}
		}
	}

	return result
}

// extractRelevantMetadata extracts relevant metadata for the response.
func extractRelevantMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return nil
	}

	relevant := make(map[string]any)

	// Include stage-related metadata for Dockerfile stages
	if stageName, ok := metadata["stage_name"].(string); ok {
		relevant["stage_name"] = stageName
	}
	if stageIndex, ok := metadata["stage_index"].(int); ok {
		relevant["stage_index"] = stageIndex
	}

	// Only return if we have relevant data
	if len(relevant) == 0 {
		return nil
	}

	return relevant
}

// buildDependencyChain creates a simple dependency chain string.
func buildDependencyChain(upstream []DependencyInfo, target string, downstream []DependencyInfo) string {
	if len(upstream) == 0 && len(downstream) == 0 {
		return target
	}

	chain := []string{}

	// Add upstream in reverse order (furthest first)
	for i := len(upstream) - 1; i >= 0; i-- {
		chain = append(chain, upstream[i].Name)
	}

	// Add target in the middle
	chain = append(chain, target)

	// Add downstream
	for _, dep := range downstream {
		chain = append(chain, dep.Name)
	}

	// Join with arrow
	var result strings.Builder
	for i, node := range chain {
		if i > 0 {
			result.WriteString(" → ")
		}
		result.WriteString(node)
	}

	return result.String()
}
