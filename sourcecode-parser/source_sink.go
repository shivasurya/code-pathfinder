// source_sink.go
package main

import (
	"fmt"

	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
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

type GraphNodeContext struct {
	Node *GraphNode
}

// GetValue returns the value of a field in a GraphNode based on the key.
func (gnc *GraphNodeContext) GetValue(key, val string) string {
	switch key {
	case "visibility":
		return gnc.Node.Modifier
	case "annotation":
		for _, annotation := range gnc.Node.Annotation {
			if annotation == val {
				return annotation
			}
		}
		return ""
	case "returntype":
		return gnc.Node.ReturnType
	case "name":
		return gnc.Node.Name
	case "argumentype":
		// check value in MethodArgumentsType array
		for i, arg := range gnc.Node.MethodArgumentsType {
			if arg == val {
				return gnc.Node.MethodArgumentsType[i]
			}
		}
		return ""
	case "argumentname":
		// check value in MethodArgumentsValue array
		for i, arg := range gnc.Node.MethodArgumentsValue {
			if arg == val {
				return gnc.Node.MethodArgumentsValue[i]
			}
		}
		return ""
	case "superclass":
		return gnc.Node.SuperClass
	case "interface":
		// check value in Interface array
		for i, iface := range gnc.Node.Interface {
			if iface == val {
				return gnc.Node.Interface[i]
			}
		}
		return ""
	case "scope":
		return gnc.Node.Scope
	case "variablevalue":
		return gnc.Node.VariableValue
	case "variabledatatype":
		return gnc.Node.DataType
	case "throwstype":
		for i, arg := range gnc.Node.ThrowsExceptions {
			if arg == val {
				return gnc.Node.ThrowsExceptions[i]
			}
		}
		return ""
	case "has_access":
		if gnc.Node.hasAccess {
			return "true"
		}
		return "false"
	case "is_java_source":
		if gnc.Node.isJavaSourceFile {
			return "true"
		}
		return "false"
	case "comment_author":
		if gnc.Node.JavaDoc != nil {
			if gnc.Node.JavaDoc.Author != "" {
				return gnc.Node.JavaDoc.Author
			}
		}
		return ""
	case "comment_see":
		if gnc.Node.JavaDoc != nil {
			for _, docTag := range gnc.Node.JavaDoc.Tags {
				if docTag.TagName == "see" && docTag.Text != "" {
					if docTag.Text == val {
						return docTag.Text
					}
				}
			}
		}
		return ""
	case "comment_version":
		if gnc.Node.JavaDoc != nil {
			for _, docTag := range gnc.Node.JavaDoc.Tags {
				if docTag.TagName == "version" && docTag.Text != "" {
					if docTag.Text == val {
						return docTag.Text
					}
				}
			}
		}
		return ""
	case "comment_since":
		if gnc.Node.JavaDoc != nil {
			for _, docTag := range gnc.Node.JavaDoc.Tags {
				if docTag.TagName == "since" && docTag.Text != "" {
					if docTag.Text == val {
						return docTag.Text
					}
				}
			}
		}
		return ""
	case "comment_param":
		if gnc.Node.JavaDoc != nil {
			for _, docTag := range gnc.Node.JavaDoc.Tags {
				if docTag.TagName == "param" && docTag.Text != "" {
					if docTag.Text == val {
						return docTag.Text
					}
				}
			}
		}
		return ""
	case "comment_throws":
		if gnc.Node.JavaDoc != nil {
			for _, docTag := range gnc.Node.JavaDoc.Tags {
				if docTag.TagName == "throws" && docTag.Text != "" {
					if docTag.Text == val {
						return docTag.Text
					}
				}
			}
		}
		return ""
	default:
		fmt.Printf("Unsupported attribute key: %s\n", key)
		return ""
	}
}

func QueryEntities(graph *CodeGraph, query parser.Query) []*GraphNode {
	result := make([]*GraphNode, 0)

	for _, node := range graph.Nodes {
		for _, entity := range query.SelectList {
			if entity.Entity == node.Type {
				result = append(result, node)
			}
		}
	}
	return result
}
