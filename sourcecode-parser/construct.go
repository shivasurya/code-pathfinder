package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
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

func buildGraphFromAST(node *sitter.Node, sourceCode []byte, graph *CodeGraph, currentContext *GraphNode) {
	packageName, className := extractPackageAndClassName(node, sourceCode)

	switch node.Type() {
	case "method_declaration":
		methodName, methodID := extractMethodName(node, sourceCode, packageName, className)
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
					// get type of argument and add to method arguments
					paramType := param.Child(0).Content(sourceCode)
					paramValue := param.Child(1).Content(sourceCode)
					methodArgumentType = append(methodArgumentType, paramType)
					methodArgumentValue = append(methodArgumentValue, paramValue)
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
		methodName, methodID := extractMethodName(node, sourceCode, packageName, className) // Implement this
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
			ID:          generateMethodID(node, sourceCode, className, []string{}),
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
	}

	// Recursively process child nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		buildGraphFromAST(child, sourceCode, graph, currentContext)
	}
}

// write a function to generate unique method id from method name, class name, and package name, parameters, and return type.
func generateMethodID(node *sitter.Node, sourceCode []byte, methodName string, parameters []string) string {
	// Example: Use the node type and its start byte position in the source code to generate a unique ID
	hashInput := fmt.Sprintf("%s-%s", methodName, parameters)
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

// write a function to get package name and class name from the AST.
func extractPackageAndClassName(node *sitter.Node, sourceCode []byte) (string, string) {
	var packageName, className string

	// Loop through the child nodes to find the package name and class name
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		// Check if the child node is a package_declaration
		if child.Type() == "package_declaration" {
			packageName = child.Content(sourceCode)
			packageName = strings.TrimSpace(packageName)
		}

		// Check if the child node is a class_declaration
		if child.Type() == "class_declaration" {
			// find if child has identifier node child and get class name
			className = child.ChildByFieldName("name").Content(sourceCode)
			className = strings.TrimSpace(className)
		}
	}
	return packageName, className
}

func extractMethodName(node *sitter.Node, sourceCode []byte, packageName string, className string) (string, string) {
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
				modifiers = append(modifiers, child.Content(sourceCode))
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
	methodID = generateMethodID(node, sourceCode, methodName, parameters)
	return methodName, methodID
}

func getFiles(directory string) ([]string, error) {
	var files []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
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
	content, err := ioutil.ReadFile(path)
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
		log.Fatal(err)
	}
	for _, file := range files {
		sourceCode, err := readFile(file)
		if err != nil {
			log.Fatal(err)
		}
		// Parse the source code
		tree, err := parser.ParseCtx(context.TODO(), nil, []byte(sourceCode))
		if err != nil {
			log.Fatal(err)
		}
		defer tree.Close()

		// TODO: Merge the tree into a single root node
		// TODO: normalize the class name without duplication of class, method names

		rootNode := tree.RootNode()

		buildGraphFromAST(rootNode, []byte(sourceCode), codeGraph, nil)
	}

	// log.Println("Graph built successfully:", codeGraph)
	log.Println("Graph built successfully")
	// go StartServer(codeGraph)

	// select {}
	return codeGraph
}
