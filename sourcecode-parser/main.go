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

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

type GraphNode struct {
	ID            string
	Type          string
	Name          string
	CodeSnippet   string
	LineNumber    uint32
	OutgoingEdges []*GraphEdge
	IsExternal    bool
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

func generateUniqueID(node *sitter.Node, sourceCode []byte) string {
	// Example: Use the node type and its start byte position in the source code to generate a unique ID
	hashInput := fmt.Sprintf("%s-%d-%d", node.Type(), node.StartByte(), node.EndByte())
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
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

func buildGraphFromAST(node *sitter.Node, sourceCode []byte, graph *CodeGraph, currentContext *GraphNode) {
	// fmt.Println("Node Type: ", node.Type(), " - ", node.Content(sourceCode))

	//fmt.Print(node.Type() + " - ")
	//fmt.Print(node.Content(sourceCode) + "\n")

	switch node.Type() {
	case "method_declaration":
		methodName, methodId := extractMethodName(node, sourceCode)
		invokedNode, exists := graph.Nodes[methodId]
		fmt.Println(graph.Nodes[methodName], methodName, methodId, exists)
		if !exists || (exists && invokedNode.ID != methodId) {
			invokedNode = createMethodNode(node, sourceCode)
		}
		graph.AddNode(invokedNode)
		currentContext = invokedNode // Update context to the new method

	case "method_invocation":
		methodName, methodId := extractMethodName(node, sourceCode) // Implement this
		invokedNode, exists := graph.Nodes[methodId]
		fmt.Println(graph.Nodes[methodName], methodName, methodId, exists)
		if !exists || (exists && invokedNode.ID != methodId) {
			// Create a placeholder node for external or inbuilt method
			invokedNode = &GraphNode{
				ID:          methodId,
				Type:        "method_invocation",
				Name:        methodName,
				IsExternal:  true,
				CodeSnippet: string(node.Content(sourceCode)),
				LineNumber:  node.StartPoint().Row + 1, // Lines start from 0 in the AST
			}
			graph.AddNode(invokedNode)
		}

		if currentContext != nil {
			graph.AddEdge(currentContext, invokedNode)
		}
	}

	// Recursively process child nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		buildGraphFromAST(child, sourceCode, graph, currentContext)
	}
}

func createMethodNode(node *sitter.Node, sourceCode []byte) *GraphNode {
	methodName, methodId := extractMethodName(node, sourceCode) // Extract the method name

	return &GraphNode{
		ID:          methodId, // In a real scenario, you would construct a unique ID, possibly using the method signature
		Type:        "method_declaration",
		Name:        methodName,
		CodeSnippet: string(node.Content(sourceCode)),
		LineNumber:  node.StartPoint().Row + 1, // Lines start from 0 in the AST
		// CodeSnippet and LineNumber are skipped as per the requirement
	}
}

// write a function to generate unique method id from method name, class name, and package name, parameters, and return type
func generateMethodID(node *sitter.Node, sourceCode []byte, methodName string, parameters []string, returnType string) string {
	packageName, className := extractPackageAndClassName(node, sourceCode)
	// Example: Use the node type and its start byte position in the source code to generate a unique ID
	hashInput := fmt.Sprintf("%s-%s-%s-%s-%s", packageName, className, methodName, parameters, returnType)
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

// write a function to get package name and class name from the AST
func extractPackageAndClassName(node *sitter.Node, sourceCode []byte) (string, string) {
	var packageName, className string

	// Loop through the child nodes to find the package name and class name
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		// Check if the child node is a package_declaration
		if child.Type() == "package_declaration" {
			packageName = child.Content(sourceCode)
		}

		// Check if the child node is a class_declaration
		if child.Type() == "class_declaration" {
			className = child.Content(sourceCode)
		}
	}
	return packageName, className
}

func extractMethodName(node *sitter.Node, sourceCode []byte) (string, string) {
	var methodId string

	// if the child node is method_declaration, extract method name, modifiers, parameters, and return type
	var returnType, methodName string
	var modifiers, parameters []string

	if node.Type() == "method_declaration" {
		// Iterate over all children of the method_declaration node
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			switch child.Type() {
			case "modifiers", "marker_annotation", "annotation":
				// This child is a modifier or annotation, add its content to modifiers
				modifiers = append(modifiers, child.Content(sourceCode))
			case "void_type", "type_identifier", "primitive_type":
				// This child is the return type
				returnType = child.Content(sourceCode)
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

	// check if type is method_invokacaion
	// if the child node is method_invocation, extract method name
	if node.Type() == "method_invocation" {

		for j := 0; j < int(node.ChildCount()); j++ {
			child := node.Child(j)
			fmt.Println(child.Type())
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
	//TODO:
	// the declaration method and invoked method should have same unique ID
	fmt.Println(returnType)
	methodId = generateMethodID(node, sourceCode, methodName, parameters, "")
	fmt.Println(methodName)
	fmt.Println(parameters)
	fmt.Println(methodId)
	fmt.Println("---------")

	return methodName, methodId
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

func main() {
	// Initialize the parser
	parser := sitter.NewParser()
	defer parser.Close()

	// Set the language (Java in this case)
	parser.SetLanguage(java.GetLanguage())

	codeGraph := NewCodeGraph()

	// iterate example-java-project directory for java code files
	// and parse each file
	directory := "example-java-project/android-demo"
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

		//TODO: Merge the tree into a single root node
		//TODO: normalize the class name without duplication of class, method names

		rootNode := tree.RootNode()

		buildGraphFromAST(rootNode, []byte(sourceCode), codeGraph, nil)
	}

	//log.Println("Graph built successfully:", codeGraph)
	log.Println("Graph built successfully")
	go StartServer(codeGraph)

	select {}
}
