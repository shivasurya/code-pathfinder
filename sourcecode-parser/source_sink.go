// source_sink.go
package main

import (
	"fmt"
)

type Result struct {
	IsConnected  bool   `json:"is_connected"`
	SourceMethod string `json:"source_method"`
	SourceLine   uint32 `json:"source_line"`
	SinkMethod   string `json:"sink_method"`
	SinkLine     uint32 `json:"sink_line"`
}

func DFS(graph *CodeGraph, currentNode, targetNode *GraphNode, visited map[string]bool) bool {
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
