// source_sink.go
package main

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"

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

type Env struct {
	Node *GraphNode
}

func (env *Env) GetVisibility() string {
	return env.Node.Modifier
}

func (env *Env) GetAnnotation(annotationVal string) string {
	for _, annotation := range env.Node.Annotation {
		if annotation == annotationVal {
			return annotation
		}
	}
	return ""
}

func (env *Env) GetReturnType() string {
	return env.Node.ReturnType
}

func (env *Env) GetName() string {
	return env.Node.Name
}

func (env *Env) GetArgumentType(argVal string) string {
	for i, arg := range env.Node.MethodArgumentsType {
		if arg == argVal {
			return env.Node.MethodArgumentsType[i]
		}
	}
	return ""
}

func (env *Env) GetArgumentName(argVal string) string {
	for i, arg := range env.Node.MethodArgumentsValue {
		if arg == argVal {
			return env.Node.MethodArgumentsValue[i]
		}
	}
	return ""
}

func (env *Env) GetSuperClass() string {
	return env.Node.SuperClass
}

func (env *Env) GetInterface(interfaceVal string) string {
	for i, iface := range env.Node.Interface {
		if iface == interfaceVal {
			return env.Node.Interface[i]
		}
	}
	return ""
}

func (env *Env) GetScope() string {
	return env.Node.Scope
}

func (env *Env) GetVariableValue() string {
	return env.Node.VariableValue
}

func (env *Env) GetVariableDataType() string {
	return env.Node.DataType
}

func (env *Env) GetThrowsType(throwsVal string) string {
	for i, arg := range env.Node.ThrowsExceptions {
		if arg == throwsVal {
			return env.Node.ThrowsExceptions[i]
		}
	}
	return ""
}

func (env *Env) HasAccess() bool {
	return env.Node.hasAccess
}

func (env *Env) IsJavaSourceFile() bool {
	return env.Node.isJavaSourceFile
}

func (env *Env) GetCommentAuthor() string {
	if env.Node.JavaDoc != nil {
		if env.Node.JavaDoc.Author != "" {
			return env.Node.JavaDoc.Author
		}
	}
	return ""
}

func (env *Env) GetCommentSee() string {
	if env.Node.JavaDoc != nil {
		for _, docTag := range env.Node.JavaDoc.Tags {
			if docTag.TagName == "see" && docTag.Text != "" {
				return docTag.Text
			}
		}
	}
	return ""
}

func (env *Env) GetCommentVersion() string {
	if env.Node.JavaDoc != nil {
		for _, docTag := range env.Node.JavaDoc.Tags {
			if docTag.TagName == "version" && docTag.Text != "" {
				return docTag.Text
			}
		}
	}
	return ""
}

func (env *Env) GetCommentSince() string {
	if env.Node.JavaDoc != nil {
		for _, docTag := range env.Node.JavaDoc.Tags {
			if docTag.TagName == "since" && docTag.Text != "" {
				return docTag.Text
			}
		}
	}
	return ""
}

func (env *Env) GetCommentParam() string {
	if env.Node.JavaDoc != nil {
		for _, docTag := range env.Node.JavaDoc.Tags {
			if docTag.TagName == "param" && docTag.Text != "" {
				return docTag.Text
			}
		}
	}
	return ""
}

func (env *Env) GetCommentThrows() string {
	if env.Node.JavaDoc != nil {
		for _, docTag := range env.Node.JavaDoc.Tags {
			if docTag.TagName == "throws" && docTag.Text != "" {
				return docTag.Text
			}
		}
	}
	return ""
}

func (env *Env) GetCommentReturn() string {
	if env.Node.JavaDoc != nil {
		for _, docTag := range env.Node.JavaDoc.Tags {
			if docTag.TagName == "return" && docTag.Text != "" {
				return docTag.Text
			}
		}
	}
	return ""
}

func QueryEntities(graph *CodeGraph, query parser.Query) []*GraphNode {
	result := make([]*GraphNode, 0)

	for _, node := range graph.Nodes {
		for _, entity := range query.SelectList {
			if entity.Entity == node.Type && FilterEntities(node, query) {
				result = append(result, node)
			}
		}
	}
	return result
}

func FilterEntities(node *GraphNode, query parser.Query) bool {
	expression := query.Expression
	if expression == "" {
		return true
	}
	env := &Env{Node: node}
	expression = strings.ReplaceAll(expression, "md.", "")
	program, err := expr.Compile(expression, expr.Env(env))
	if err != nil {
		fmt.Println("Error compiling expression: ", err)
		return false
	}
	output, err := expr.Run(program, env)
	if err != nil {
		fmt.Println("Error evaluating expression: ", err)
		return false
	}
	if output.(bool) {
		return true
	}
	return false
}
