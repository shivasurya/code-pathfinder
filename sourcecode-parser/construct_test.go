package main

import (
	"testing"
)

func TestNewCodeGraph(t *testing.T) {
	graph := NewCodeGraph()
	if graph == nil {
		t.Error("NewCodeGraph() returned nil")
	}
	if graph != nil && graph.Nodes == nil {
		t.Error("NewCodeGraph() returned graph with nil Nodes")
	}
	if graph != nil && graph.Edges == nil {
		t.Error("NewCodeGraph() returned graph with nil Edges")
	}
	if graph != nil && len(graph.Nodes) != 0 {
		t.Errorf("NewCodeGraph() returned graph with non-empty Nodes, got %d nodes", len(graph.Nodes))
	}
	if graph != nil && len(graph.Edges) != 0 {
		t.Errorf("NewCodeGraph() returned graph with non-empty Edges, got %d edges", len(graph.Edges))
	}
}

func TestAddNode(t *testing.T) {
	graph := NewCodeGraph()
	node := &GraphNode{ID: "test_node"}
	graph.AddNode(node)

	if len(graph.Nodes) != 1 {
		t.Errorf("AddNode() failed to add node, expected 1 node, got %d", len(graph.Nodes))
	}
	if graph.Nodes["test_node"] != node {
		t.Error("AddNode() failed to add node with correct ID")
	}
}

func TestAddEdge(t *testing.T) {
	graph := NewCodeGraph()
	node1 := &GraphNode{ID: "node1"}
	node2 := &GraphNode{ID: "node2"}
	graph.AddNode(node1)
	graph.AddNode(node2)

	graph.AddEdge(node1, node2)

	if len(graph.Edges) != 1 {
		t.Errorf("AddEdge() failed to add edge, expected 1 edge, got %d", len(graph.Edges))
	}
	if graph.Edges[0].From != node1 || graph.Edges[0].To != node2 {
		t.Error("AddEdge() failed to add edge with correct From and To nodes")
	}
	if len(node1.OutgoingEdges) != 1 {
		t.Errorf("AddEdge() failed to add outgoing edge to From node, expected 1 edge, got %d", len(node1.OutgoingEdges))
	}
	if node1.OutgoingEdges[0].To != node2 {
		t.Error("AddEdge() failed to add correct outgoing edge to From node")
	}
}

func TestAddMultipleNodesAndEdges(t *testing.T) {
	graph := NewCodeGraph()
	node1 := &GraphNode{ID: "node1"}
	node2 := &GraphNode{ID: "node2"}
	node3 := &GraphNode{ID: "node3"}

	graph.AddNode(node1)
	graph.AddNode(node2)
	graph.AddNode(node3)

	graph.AddEdge(node1, node2)
	graph.AddEdge(node2, node3)
	graph.AddEdge(node1, node3)

	if len(graph.Nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 3 {
		t.Errorf("Expected 3 edges, got %d", len(graph.Edges))
	}
	if len(node1.OutgoingEdges) != 2 {
		t.Errorf("Expected 2 outgoing edges for node1, got %d", len(node1.OutgoingEdges))
	}
	if len(node2.OutgoingEdges) != 1 {
		t.Errorf("Expected 1 outgoing edge for node2, got %d", len(node2.OutgoingEdges))
	}
	if len(node3.OutgoingEdges) != 0 {
		t.Errorf("Expected 0 outgoing edges for node3, got %d", len(node3.OutgoingEdges))
	}
}

func TestIsJavaSourceFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"Valid Java file", "Example.java", true},
		{"Lowercase extension", "example.java", true},
		{"Non-Java file", "example.txt", false},
		{"No extension", "javafile", false},
		{"Empty string", "", false},
		{"Java file with path", "/path/to/Example.java", true},
		{"Java file with Windows path", "C:\\path\\to\\Example.java", true},
		{"File with multiple dots", "my.test.file.java", true},
		{"Hidden Java file", ".hidden.java", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isJavaSourceFile(tt.filename); got != tt.want {
				t.Errorf("isJavaSourceFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}
