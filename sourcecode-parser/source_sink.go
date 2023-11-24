// source_sink.go
package main

import (
    "fmt"
)

type SourceSinkPath struct {
    Source *GraphNode
    Sink   *GraphNode
}

type Result struct {
    IsConnected bool `json:"isConnected"`
    SourceMethod string `json:"sourceMethod"`
    SourceLine uint32 `json:"sourceLine"`
    SinkMethod string `json:"sinkMethod"`
    SinkLine uint32 `json:"sinkLine"`
}

func DFS(currentNode *GraphNode, targetNode *GraphNode, visited map[string]bool) bool {
    if currentNode.ID == targetNode.ID {
        return true // Target node found
    }

    visited[currentNode.ID] = true

    for _, edge := range currentNode.OutgoingEdges { // Assuming each node has a list of outgoing edges
        nextNode := edge.To
        if !visited[nextNode.ID] {
            if DFS(nextNode, targetNode, visited) {
                return true
            }
        }
    }
    return false
}

func AnalyzeSourceSinkPatterns(graph *CodeGraph, sourceMethodName, sinkMethodName string) Result  {
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
     isConnected := DFS(sourceNode, sinkNode, visited)
     // Return true if sourceNode is connected to sinkNode as a result of the DFS
    return Result{IsConnected: isConnected, SourceMethod: sourceNode.CodeSnippet, SinkMethod: sinkNode.CodeSnippet, SourceLine: sourceNode.LineNumber, SinkLine: sinkNode.LineNumber}
}