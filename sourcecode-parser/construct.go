package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
	//nolint:all
)

type GraphNode struct {
	ID                   string
	Type                 string
	Name                 string
	CodeSnippet          string
	LineNumber           uint32
	OutgoingEdges        []*GraphEdge
	IsExternal           bool
	Modifier             string
	ReturnType           string
	MethodArgumentsType  []string
	MethodArgumentsValue []string
	PackageName          string
	ImportPackage        []string
	SuperClass           string
	Interface            []string
	DataType             string
	Scope                string
	VariableValue        string
	hasAccess            bool
	File                 string
}

type GraphEdge struct {
	From *GraphNode
	To   *GraphNode
}

type CodeGraph struct {
	Nodes map[string]*GraphNode
	Edges []*GraphEdge
}

func NewCodeGraph() *CodeGraph {
	return &CodeGraph{
		Nodes: make(map[string]*GraphNode),
		Edges: make([]*GraphEdge, 0),
	}
}

func (g *CodeGraph) AddNode(node *GraphNode) {
	g.Nodes[node.ID] = node
}

func (g *CodeGraph) AddEdge(from, to *GraphNode) {
	edge := &GraphEdge{From: from, To: to}
	g.Edges = append(g.Edges, edge)
	from.OutgoingEdges = append(from.OutgoingEdges, edge)
}

// Add to graph.go

// FindNodesByType finds all nodes of a given type.
func (g *CodeGraph) FindNodesByType(nodeType string) []*GraphNode {
	var nodes []*GraphNode
	for _, node := range g.Nodes {
		if node.Type == nodeType {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func extractVisibilityModifier(modifiers string) string {
	words := strings.Fields(modifiers)
	for _, word := range words {
		switch word {
		case "public", "private", "protected":
			return word
		}
	}
	return "" // return an empty string if no visibility modifier is found
}

func hasAccess(node *sitter.Node, variableName string, sourceCode []byte) bool {
	if node == nil {
		return false
	}
	if node.Type() == "identifier" && node.Content(sourceCode) == variableName {
		return true
	}

	// Recursively check all children of the current node
	for i := 0; i < int(node.ChildCount()); i++ {
		childNode := node.Child(i)
		if hasAccess(childNode, variableName, sourceCode) {
			return true
		}
	}

	// Continue checking in the next sibling
	return hasAccess(node.NextSibling(), variableName, sourceCode)
}

func buildGraphFromAST(node *sitter.Node, sourceCode []byte, graph *CodeGraph, currentContext *GraphNode, file string) {
	switch node.Type() {
	case "method_declaration":
		methodName, methodID := extractMethodName(node, sourceCode)
		invokedNode, exists := graph.Nodes[methodID]
		modifiers := ""
		returnType := ""
		methodArgumentType := []string{}
		methodArgumentValue := []string{}

		for i := 0; i < int(node.ChildCount()); i++ {
			childNode := node.Child(i)
			childType := childNode.Type()

			switch childType {
			case "modifiers":
				modifiers = childNode.Content(sourceCode)
			case "void_type", "type_identifier":
				// get return type of method
				returnType = childNode.Content(sourceCode)
			case "formal_parameters":
				// get method arguments
				for j := 0; j < int(childNode.NamedChildCount()); j++ {
					param := childNode.NamedChild(j)
					if param.Type() == "formal_parameter" {
						// get type of argument and add to method arguments
						paramType := param.Child(0).Content(sourceCode)
						paramValue := param.Child(1).Content(sourceCode)
						methodArgumentType = append(methodArgumentType, paramType)
						methodArgumentValue = append(methodArgumentValue, paramValue)
					}
				}
			}
		}

		if !exists || (exists && invokedNode.ID != methodID) {
			invokedNode = &GraphNode{
				ID:                   methodID, // In a real scenario, you would construct a unique ID, possibly using the method signature
				Type:                 "method_declaration",
				Name:                 methodName,
				CodeSnippet:          node.Content(sourceCode),
				LineNumber:           node.StartPoint().Row + 1, // Lines start from 0 in the AST
				Modifier:             extractVisibilityModifier(modifiers),
				ReturnType:           returnType,
				MethodArgumentsType:  methodArgumentType,
				MethodArgumentsValue: methodArgumentValue,
				// CodeSnippet and LineNumber are skipped as per the requirement
			}
		}
		graph.AddNode(invokedNode)
		currentContext = invokedNode // Update context to the new method

	case "method_invocation":
		methodName, methodID := extractMethodName(node, sourceCode) // Implement this
		invokedNode, exists := graph.Nodes[methodID]
		if !exists || (exists && invokedNode.ID != methodID) {
			// Create a placeholder node for external or inbuilt method
			invokedNode = &GraphNode{
				ID:          methodID,
				Type:        "method_invocation",
				Name:        methodName,
				IsExternal:  true,
				CodeSnippet: node.Content(sourceCode),
				LineNumber:  node.StartPoint().Row + 1, // Lines start from 0 in the AST
			}
			graph.AddNode(invokedNode)
		}

		if currentContext != nil {
			graph.AddEdge(currentContext, invokedNode)
		}
	case "class_declaration":
		className := node.ChildByFieldName("name").Content(sourceCode)
		packageName := ""
		accessModifier := ""
		superClass := ""
		implementedInterface := []string{}
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "modifiers" {
				accessModifier = child.Content(sourceCode)
			}
			if child.Type() == "superclass" {
				for j := 0; j < int(child.ChildCount()); j++ {
					if child.Child(j).Type() == "type_identifier" {
						superClass = child.Child(j).Content(sourceCode)
					}
				}
			}
			if child.Type() == "super_interfaces" {
				for j := 0; j < int(child.ChildCount()); j++ {
					// typelist node and then iterate through type_identifier node
					typeList := child.Child(j)
					for k := 0; k < int(typeList.ChildCount()); k++ {
						implementedInterface = append(implementedInterface, typeList.Child(k).Content(sourceCode))
					}
				}
			}
		}
		classNode := &GraphNode{
			ID:          generateMethodID(className, []string{}),
			Type:        "class_declaration",
			Name:        className,
			CodeSnippet: node.Content(sourceCode),
			LineNumber:  node.StartPoint().Row + 1,
			PackageName: packageName,
			Modifier:    extractVisibilityModifier(accessModifier),
			SuperClass:  superClass,
			Interface:   implementedInterface,
		}
		graph.AddNode(classNode)
	case "local_variable_declaration", "field_declaration":
		// Extract variable name, type, and modifiers
		variableName := ""
		variableType := ""
		variableModifier := ""
		variableValue := ""
		hasAccessValue := false
		var scope string
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			switch child.Type() {
			case "variable_declarator":
				variableName = child.Content(sourceCode)
				for j := 0; j < int(child.ChildCount()); j++ {
					if child.Child(j).Type() == "identifier" {
						variableName = child.Child(j).Content(sourceCode)
					}
					// if child type contains =, iterate through and get remaining content
					if child.Child(j).Type() == "=" {
						for k := j + 1; k < int(child.ChildCount()); k++ {
							variableValue += child.Child(k).Content(sourceCode)
						}
					}

				}
				// remove spaces from variable value
				variableValue = strings.ReplaceAll(variableValue, " ", "")
				// remove new line from variable value
				variableValue = strings.ReplaceAll(variableValue, "\n", "")
			case "modifiers":
				variableModifier = child.Content(sourceCode)
			}
			// if child type contains type, get the type of variable
			if strings.Contains(child.Type(), "type") {
				variableType = child.Content(sourceCode)
			}
		}
		if node.Type() == "local_variable_declaration" {
			scope = "local"
			hasAccessValue = hasAccess(node.NextSibling(), variableName, sourceCode)
		} else {
			scope = "field"
			hasAccessValue = false
		}
		// Create a new node for the variable
		variableNode := &GraphNode{
			ID:            generateMethodID(variableName, []string{}),
			Type:          "variable_declaration",
			Name:          variableName,
			CodeSnippet:   node.Content(sourceCode),
			LineNumber:    node.StartPoint().Row + 1,
			Modifier:      extractVisibilityModifier(variableModifier),
			DataType:      variableType,
			Scope:         scope,
			VariableValue: variableValue,
			hasAccess:     hasAccessValue,
			File:          file,
		}
		graph.AddNode(variableNode)
	}

	// Recursively process child nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		buildGraphFromAST(child, sourceCode, graph, currentContext, file)
	}
}

// write a function to generate unique method id from method name, class name, and package name, parameters, and return type.
func generateMethodID(methodName string, parameters []string) string {
	// Example: Use the node type and its start byte position in the source code to generate a unique ID
	hashInput := fmt.Sprintf("%s-%s", methodName, parameters)
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

//nolint:all
func extractMethodName(node *sitter.Node, sourceCode []byte) (string, string) {
	var methodID string

	// if the child node is method_declaration, extract method name, modifiers, parameters, and return type
	var methodName string
	var modifiers, parameters []string

	if node.Type() == "method_declaration" {
		// Iterate over all children of the method_declaration node
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			switch child.Type() {
			case "modifiers", "marker_annotation", "annotation":
				// This child is a modifier or annotation, add its content to modifiers
				modifiers = append(modifiers, child.Content(sourceCode)) //nolint:all
			case "identifier":
				// This child is the method name
				methodName = child.Content(sourceCode)
			case "formal_parameters":
				// This child represents formal parameters; iterate through its children
				for j := 0; j < int(child.NamedChildCount()); j++ {
					param := child.NamedChild(j)
					parameters = append(parameters, param.Content(sourceCode))
				}
			}
		}
	}

	// check if type is method_invocation
	// if the child node is method_invocation, extract method name
	if node.Type() == "method_invocation" {
		for j := 0; j < int(node.ChildCount()); j++ {
			child := node.Child(j)
			if child.Type() == "identifier" {
				if methodName == "" {
					methodName = child.Content(sourceCode)
				} else {
					methodName = methodName + "." + child.Content(sourceCode)
				}
			}

			argumentsNode := node.ChildByFieldName("argument_list")
			// add data type of arguments list
			if argumentsNode != nil {
				for k := 0; k < int(argumentsNode.ChildCount()); k++ {
					argument := argumentsNode.Child(k)
					parameters = append(parameters, argument.Child(0).Content(sourceCode))
				}
			}

		}
	}
	methodID = generateMethodID(methodName, parameters)
	return methodName, methodID
}

func getFiles(directory string) ([]string, error) {
	var files []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, _ error) error {
		if !info.IsDir() {
			// append only java files
			if filepath.Ext(path) == ".java" {
				files = append(files, path)
			}
		}
		return nil
	})
	return files, err
}

func readFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func Initialize(directory string) *CodeGraph {
	// Initialize the parser
	parser := sitter.NewParser()
	defer parser.Close()

	// Set the language (Java in this case)
	parser.SetLanguage(java.GetLanguage())

	codeGraph := NewCodeGraph()

	files, err := getFiles(directory)
	if err != nil {
		//nolint:all
		log.Fatal(err)
	}
	for _, file := range files {
		sourceCode, err := readFile(file)
		if err != nil {
			log.Fatal(err)
		}
		// Parse the source code
		tree, err := parser.ParseCtx(context.TODO(), nil, sourceCode)
		if err != nil {
			log.Fatal(err)
		}
		//nolint:all
		defer tree.Close()

		// TODO: Merge the tree into a single root node
		// TODO: normalize the class name without duplication of class, method names

		rootNode := tree.RootNode()

		buildGraphFromAST(rootNode, sourceCode, codeGraph, nil, file)
	}
	//nolint:all
	// log.Println("Graph built successfully:", codeGraph)
	log.Println("Graph built successfully")
	//nolint:all
	// go StartServer(codeGraph)
	// select {}
	return codeGraph
}
