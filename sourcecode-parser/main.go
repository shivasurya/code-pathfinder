package main

import (
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
	var graphNode *GraphNode

	//fmt.Print(node.Type() + " - ")
	//fmt.Print(node.Content(sourceCode) + "\n")

	switch node.Type() {
	case "method_declaration":
		graphNode = createMethodNode(node, sourceCode)
		graph.AddNode(graphNode)
		currentContext = graphNode // Update context to the new method

	case "method_invocation":
		methodName := extractMethodName(node, sourceCode) // Implement this
		invokedNode, exists := graph.Nodes[methodName]
		if !exists {
			// Create a placeholder node for external or inbuilt method
			invokedNode = &GraphNode{
				ID:          methodName,
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
	methodName := extractMethodName(node, sourceCode) // Extract the method name

	return &GraphNode{
		ID:          methodName, // In a real scenario, you would construct a unique ID, possibly using the method signature
		Type:        "method_declaration",
		Name:        methodName,
		CodeSnippet: string(node.Content(sourceCode)),
		LineNumber:  node.StartPoint().Row + 1, // Lines start from 0 in the AST
		// CodeSnippet and LineNumber are skipped as per the requirement
	}
}

func extractMethodName(node *sitter.Node, sourceCode []byte) string {
	var methodName string

	// Loop through the child nodes to find the method name
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		// Check if the child node is an identifier (method name)
		fmt.Print(child.Content(sourceCode) + "\n")
		if child.Type() == "identifier" {
			// parse full method name
			methodName = child.Content(sourceCode)
			for j := i + 1; j < int(node.ChildCount()); j++ {
				child = node.Child(j)
				if child.Type() == "identifier" {
					methodName += "." + child.Content(sourceCode)
				}
			}
			break
		}

		// Recursively call this function if the child is 'method_declaration' or 'method_invocation'
		if child.Type() == "method_declaration" || child.Type() == "method_invocation" {
			methodName = extractMethodName(child, sourceCode)
			if methodName != "" {
				break
			}
		}
	}
	fmt.Println(methodName)
	return methodName
}

func getFiles(directory string) ([]string, error) {
	var files []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
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
	directory := "example-java-project"
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
		tree := parser.Parse(nil, sourceCode)
		defer tree.Close()

		//TODO: Merge the tree into a single root node
		//TODO: normalize the class name without duplication of class, method names
	}

	// Example Java source code
	sourceCode := `public class HelloWorld {
        public static void main(String[] args) {
            System.out.println("Hello, World!");
			int a = 1;
			Log.d("TAG", "Hello, World!");
        }
    }`

	sourceCodeBytes := []byte(sourceCode)

	// Parse the source code
	tree := parser.Parse(nil, []byte(sourceCode))
	defer tree.Close()

	// Get the root node of the AST
	rootNode := tree.RootNode()

	buildGraphFromAST(rootNode, sourceCodeBytes, codeGraph, nil)

	log.Println("Graph built successfully:", codeGraph)

	go startServer(codeGraph)

	select {}
}
