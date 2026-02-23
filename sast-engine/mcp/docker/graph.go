package docker

import (
	"fmt"
	"slices"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
)

// DependencyGraph represents a dependency graph for Docker entities.
// It maps relationships between docker-compose services and Dockerfile stages.
type DependencyGraph struct {
	// Nodes maps entity names to their details
	Nodes map[string]*DependencyNode

	// Edges maps entity names to their dependencies (from → to)
	// For compose: service → services it depends_on
	// For dockerfile: stage → stages it copies from
	Edges map[string][]string
}

// DependencyNode represents a node in the dependency graph.
type DependencyNode struct {
	Name       string         // Service name or stage name
	Type       string         // "compose_service" or "dockerfile_stage"
	File       string         // Source file path
	LineNumber uint32         // Line number in source file
	Metadata   map[string]any // Additional metadata from Node
}

// NewDependencyGraph creates a new empty dependency graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		Nodes: make(map[string]*DependencyNode),
		Edges: make(map[string][]string),
	}
}

// AddNode adds a node to the graph.
func (g *DependencyGraph) AddNode(node *DependencyNode) {
	g.Nodes[node.Name] = node
}

// AddEdge adds a directed edge (from → to) representing a dependency.
// For compose: from service depends on to service.
// For dockerfile: from stage copies from to stage.
func (g *DependencyGraph) AddEdge(from, to string) {
	if g.Edges[from] == nil {
		g.Edges[from] = []string{}
	}

	// Avoid duplicate edges
	if slices.Contains(g.Edges[from], to) {
		return
	}

	g.Edges[from] = append(g.Edges[from], to)
}

// GetDependencies returns the dependencies for a node (upstream).
func (g *DependencyGraph) GetDependencies(name string) []string {
	if deps, exists := g.Edges[name]; exists {
		return deps
	}
	return []string{}
}

// GetDependents returns nodes that depend on this node (downstream).
func (g *DependencyGraph) GetDependents(name string) []string {
	dependents := []string{}
	for node, deps := range g.Edges {
		if slices.Contains(deps, name) {
			dependents = append(dependents, node)
		}
	}
	return dependents
}

// BuildComposeGraph builds a dependency graph from compose services.
func BuildComposeGraph(codeGraph *graph.CodeGraph) *DependencyGraph {
	depGraph := NewDependencyGraph()

	// Iterate compose_service nodes
	for _, node := range codeGraph.Nodes {
		if node.Type != "compose_service" {
			continue
		}

		// Add node to graph
		depNode := &DependencyNode{
			Name:       node.Name,
			Type:       "compose_service",
			File:       node.File,
			LineNumber: node.LineNumber,
			Metadata:   node.Metadata,
		}
		depGraph.AddNode(depNode)

		// Extract dependencies from Metadata
		if node.Metadata != nil {
			if deps, ok := node.Metadata["depends_on"].([]any); ok {
				for _, dep := range deps {
					if depStr, ok := dep.(string); ok {
						depGraph.AddEdge(node.Name, depStr)
					}
				}
			}
		}
	}

	return depGraph
}

// BuildDockerfileGraph builds a dependency graph for multi-stage Dockerfiles.
func BuildDockerfileGraph(codeGraph *graph.CodeGraph, filePath string) *DependencyGraph {
	depGraph := NewDependencyGraph()

	// Map stage names to nodes for quick lookup
	stageNodes := make(map[string]*graph.Node)

	// Pass 1: Collect all FROM instructions (stages)
	for _, node := range codeGraph.Nodes {
		if node.Type != "dockerfile_instruction" || node.Name != "FROM" {
			continue
		}
		if filePath != "" && node.File != filePath {
			continue
		}

		// Get stage name from Metadata
		stageName := getStageName(node)
		stageNodes[stageName] = node

		// Add to dependency graph
		depNode := &DependencyNode{
			Name:       stageName,
			Type:       "dockerfile_stage",
			File:       node.File,
			LineNumber: node.LineNumber,
			Metadata:   node.Metadata,
		}
		depGraph.AddNode(depNode)
	}

	// Pass 2: Collect COPY --from instructions (dependencies)
	for _, node := range codeGraph.Nodes {
		if node.Type != "dockerfile_instruction" || node.Name != "COPY" {
			continue
		}
		if filePath != "" && node.File != filePath {
			continue
		}

		if node.Metadata == nil {
			continue
		}

		// Get copy_from stage
		copyFrom, hasCopyFrom := node.Metadata["copy_from"].(string)
		if !hasCopyFrom {
			continue
		}

		// Determine current stage by finding the FROM instruction before this COPY
		currentStage := findStageForInstruction(node, stageNodes)
		if currentStage == "" {
			continue
		}

		// Add edge: current stage depends on copy_from stage
		depGraph.AddEdge(currentStage, copyFrom)
	}

	return depGraph
}

// getStageName extracts the stage name from a FROM instruction node.
func getStageName(node *graph.Node) string {
	if node.Metadata == nil {
		return "stage-0"
	}

	// Try to get stage_name (alias)
	if stageName, ok := node.Metadata["stage_name"].(string); ok && stageName != "" {
		return stageName
	}

	// Fall back to stage index
	if stageIndex, ok := node.Metadata["stage_index"].(int); ok {
		return formatStageIndex(stageIndex)
	}

	return "stage-0"
}

// formatStageIndex formats a stage index as "stage-0", "stage-1", etc.
func formatStageIndex(index int) string {
	return fmt.Sprintf("stage-%d", index)
}

// findStageForInstruction finds which stage a COPY instruction belongs to.
func findStageForInstruction(copyNode *graph.Node, stageNodes map[string]*graph.Node) string {
	if copyNode.Metadata == nil {
		return ""
	}

	// Get the stage index from the COPY instruction
	stageIndex, hasIndex := copyNode.Metadata["stage_index"].(int)
	if !hasIndex {
		return ""
	}

	// Find the FROM node with this stage index
	for stageName, stageNode := range stageNodes {
		if stageNode.Metadata == nil {
			continue
		}
		if idx, ok := stageNode.Metadata["stage_index"].(int); ok && idx == stageIndex {
			return stageName
		}
	}

	// Fall back to formatted index
	return formatStageIndex(stageIndex)
}
