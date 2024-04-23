// source_sink.go
package main

import (
	"fmt"
	"queryparser"
)

type SourceSinkPath struct {
	Source *GraphNode
	Sink   *GraphNode
}

var MethodAttribute = map[string]int{
	"name":       0,
	"visibility": 1,
	"returntype": 2,
	// Add more attributes as needed
}

type Result struct {
	IsConnected  bool   `json:"isConnected"`
	SourceMethod string `json:"sourceMethod"`
	SourceLine   uint32 `json:"sourceLine"`
	SinkMethod   string `json:"sinkMethod"`
	SinkLine     uint32 `json:"sinkLine"`
}

func DFS(graph *CodeGraph, currentNode *GraphNode, targetNode *GraphNode, visited map[string]bool) bool {
	if currentNode.ID == targetNode.ID {
		return true // Target node found
	}

	visited[currentNode.ID] = true

	for _, edge := range currentNode.OutgoingEdges { // Assuming each node has a list of outgoing edges
		fmt.Println(edge.From.Name, "->--", edge.To.Name)
		fmt.Println(edge.To.OutgoingEdges)
		fmt.Println(edge.From.ID, "->", edge.To.ID)
		nextNode := edge.To
		fmt.Println(visited[nextNode.ID])
		if !visited[nextNode.ID] {
			fmt.Println(graph.Nodes[nextNode.ID])
			invokedNode := graph.Nodes[nextNode.ID]
			fmt.Println(invokedNode.ID)
			fmt.Println(targetNode.ID)
			if DFS(graph, invokedNode, targetNode, visited) {
				return true
			}

		}
	}
	return false
}

func AnalyzeSourceSinkPatterns(graph *CodeGraph, sourceMethodName, sinkMethodName string) Result {
	// Find source and sink nodes
	var sourceNode, sinkNode *GraphNode
	for _, node := range graph.Nodes {
		fmt.Println(node.Name)
		if node.Type == "method_declaration" && node.Name == sourceMethodName {
			sourceNode = node
		} else if node.Type == "method_invocation" && node.Name == sinkMethodName {
			sinkNode = node
		}
	}

	if sourceNode == nil || sinkNode == nil {
		// return false if either source or sink node is not found
		return Result{IsConnected: false, SourceMethod: sourceMethodName, SinkMethod: sinkMethodName}
	}

	// Use DFS to check if sourceNode is connected to sinkNode
	visited := make(map[string]bool)
	isConnected := DFS(graph, sourceNode, sinkNode, visited)
	// Return true if sourceNode is connected to sinkNode as a result of the DFS
	return Result{IsConnected: isConnected, SourceMethod: sourceNode.CodeSnippet, SinkMethod: sinkNode.CodeSnippet, SourceLine: sourceNode.LineNumber, SinkLine: sinkNode.LineNumber}
}

type GraphNodeContext struct {
	Node *GraphNode
}

// GetValue returns the value of a field in a GraphNode based on the key.
func (gnc *GraphNodeContext) GetValue(key string) string {
	switch key {
	case "visibility":
		return gnc.Node.Modifier
	case "returntype":
		return gnc.Node.ReturnType
	case "name":
		return gnc.Node.Name
	// add other cases as necessary for your application
	default:
		fmt.Printf("Unsupported attribute key: %s\n", key)
		return ""
	}
}

func QueryEntities(graph *CodeGraph, query *queryparser.Query) []*GraphNode {
	result := make([]*GraphNode, 0)

	for _, node := range graph.Nodes {
		if node.Type == query.Entity {
			ctx := GraphNodeContext{Node: node}  // Create a context for each node
			if query.Conditions.Evaluate(&ctx) { // Use the context in evaluation
				result = append(result, node)
			}
		}
	}
	return result
}
