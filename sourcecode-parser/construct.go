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
	"sync"
	"time"

	"github.com/shivasurya/code-pathfinder/sourcecode-parser/model"
	"github.com/smacker/go-tree-sitter/java"

	sitter "github.com/smacker/go-tree-sitter"
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
	isJavaSourceFile     bool
	ThrowsExceptions     []string
	Annotation           []string
	JavaDoc              *model.Javadoc
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

func isJavaSourceFile(filename string) bool {
	return filepath.Ext(filename) == ".java"
}

//nolint:all
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

func parseJavadocTags(commentContent string) *model.Javadoc {
	javaDoc := &model.Javadoc{}
	var javadocTags []*model.JavadocTag

	commentLines := strings.Split(commentContent, "\n")
	for _, line := range commentLines {
		line = strings.TrimSpace(line)
		// line may start with /** or *
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "@") {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				tagName := strings.TrimPrefix(parts[0], "@")
				tagText := strings.TrimSpace(parts[1])

				var javadocTag *model.JavadocTag
				switch tagName {
				case "author":
					javadocTag = model.NewJavadocTag(tagName, tagText, "author")
					javaDoc.Author = tagText
				case "param":
					javadocTag = model.NewJavadocTag(tagName, tagText, "param")
				case "see":
					javadocTag = model.NewJavadocTag(tagName, tagText, "see")
				case "throws":
					javadocTag = model.NewJavadocTag(tagName, tagText, "throws")
				case "version":
					javadocTag = model.NewJavadocTag(tagName, tagText, "version")
					javaDoc.Version = tagText
				case "since":
					javadocTag = model.NewJavadocTag(tagName, tagText, "since")
				default:
					javadocTag = model.NewJavadocTag(tagName, tagText, "unknown")
				}
				javadocTags = append(javadocTags, javadocTag)
			}
		}
	}

	javaDoc.Tags = javadocTags
	javaDoc.NumberOfCommentLines = len(commentLines)
	javaDoc.CommentedCodeElements = commentContent

	return javaDoc
}

func buildGraphFromAST(node *sitter.Node, sourceCode []byte, graph *CodeGraph, currentContext *GraphNode, file string) {
	isJavaSourceFile := isJavaSourceFile(file)
	switch node.Type() {
	case "method_declaration":
		var javadoc *model.Javadoc
		if node.PrevSibling() != nil && node.PrevSibling().Type() == "block_comment" {
			commentContent := node.PrevSibling().Content(sourceCode)
			if strings.HasPrefix(commentContent, "/*") {
				javadoc = parseJavadocTags(commentContent)
			}
		}
		methodName, methodID := extractMethodName(node, sourceCode, file)
		modifiers := ""
		returnType := ""
		throws := []string{}
		methodArgumentType := []string{}
		methodArgumentValue := []string{}
		annotationMarkers := []string{}

		for i := 0; i < int(node.ChildCount()); i++ {
			childNode := node.Child(i)
			childType := childNode.Type()

			switch childType {
			case "throws":
				// namedChild
				for j := 0; j < int(childNode.NamedChildCount()); j++ {
					namedChild := childNode.NamedChild(j)
					if namedChild.Type() == "type_identifier" {
						throws = append(throws, namedChild.Content(sourceCode))
					}
				}
			case "modifiers":
				modifiers = childNode.Content(sourceCode)
				for j := 0; j < int(childNode.ChildCount()); j++ {
					if childNode.Child(j).Type() == "marker_annotation" {
						annotationMarkers = append(annotationMarkers, childNode.Child(j).Content(sourceCode))
					}
				}
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

		invokedNode := &GraphNode{
			ID:                   methodID, // In a real scenario, you would construct a unique ID, possibly using the method signature
			Type:                 "method_declaration",
			Name:                 methodName,
			CodeSnippet:          node.Content(sourceCode),
			LineNumber:           node.StartPoint().Row + 1, // Lines start from 0 in the AST
			Modifier:             extractVisibilityModifier(modifiers),
			ReturnType:           returnType,
			MethodArgumentsType:  methodArgumentType,
			MethodArgumentsValue: methodArgumentValue,
			File:                 file,
			isJavaSourceFile:     isJavaSourceFile,
			ThrowsExceptions:     throws,
			Annotation:           annotationMarkers,
			JavaDoc:              javadoc,
		}
		graph.AddNode(invokedNode)
		currentContext = invokedNode // Update context to the new method

	case "method_invocation":
		methodName, methodID := extractMethodName(node, sourceCode, file)
		arguments := []string{}
		// get argument list from arguments node iterate for child node
		for i := 0; i < int(node.ChildCount()); i++ {
			if node.Child(i).Type() == "argument_list" {
				argumentsNode := node.Child(i)
				for j := 0; j < int(argumentsNode.ChildCount()); j++ {
					argument := argumentsNode.Child(j)
					switch argument.Type() {
					case "identifier":
						arguments = append(arguments, argument.Content(sourceCode))
					case "string_literal":
						stringliteral := argument.Content(sourceCode)
						stringliteral = strings.TrimPrefix(stringliteral, "\"")
						stringliteral = strings.TrimSuffix(stringliteral, "\"")
						arguments = append(arguments, stringliteral)
					default:
						arguments = append(arguments, argument.Content(sourceCode))
					}
				}
			}
		}

		invokedNode := &GraphNode{
			ID:                   methodID,
			Type:                 "method_invocation",
			Name:                 methodName,
			IsExternal:           true,
			CodeSnippet:          node.Content(sourceCode),
			LineNumber:           node.StartPoint().Row + 1, // Lines start from 0 in the AST
			MethodArgumentsValue: arguments,
			File:                 file,
			isJavaSourceFile:     isJavaSourceFile,
		}
		graph.AddNode(invokedNode)

		if currentContext != nil {
			graph.AddEdge(currentContext, invokedNode)
		}
	case "class_declaration":
		var javadoc *model.Javadoc
		if node.PrevSibling() != nil && node.PrevSibling().Type() == "block_comment" {
			commentContent := node.PrevSibling().Content(sourceCode)
			if strings.HasPrefix(commentContent, "/*") {
				javadoc = parseJavadocTags(commentContent)
			}
		}
		className := node.ChildByFieldName("name").Content(sourceCode)
		packageName := ""
		accessModifier := ""
		superClass := ""
		annotationMarkers := []string{}
		implementedInterface := []string{}
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "modifiers" {
				accessModifier = child.Content(sourceCode)
				for j := 0; j < int(child.ChildCount()); j++ {
					if child.Child(j).Type() == "marker_annotation" {
						annotationMarkers = append(annotationMarkers, child.Child(j).Content(sourceCode))
					}
				}
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
			ID:               generateMethodID(className, []string{}, file),
			Type:             "class_declaration",
			Name:             className,
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1,
			PackageName:      packageName,
			Modifier:         extractVisibilityModifier(accessModifier),
			SuperClass:       superClass,
			Interface:        implementedInterface,
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
			JavaDoc:          javadoc,
			Annotation:       annotationMarkers,
		}
		graph.AddNode(classNode)
	case "block_comment":
		// Parse block comments
		if strings.HasPrefix(node.Content(sourceCode), "/*") {
			commentContent := node.Content(sourceCode)
			javadocTags := parseJavadocTags(commentContent)

			commentNode := &GraphNode{
				ID:               generateMethodID(node.Content(sourceCode), []string{}, file),
				Type:             "block_comment",
				CodeSnippet:      commentContent,
				LineNumber:       node.StartPoint().Row + 1,
				File:             file,
				isJavaSourceFile: isJavaSourceFile,
				JavaDoc:          javadocTags,
			}
			graph.AddNode(commentNode)
		}
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
			//nolint:all
			// hasAccessValue = hasAccess(node.NextSibling(), variableName, sourceCode)
		} else {
			scope = "field"
		}
		// Create a new node for the variable
		variableNode := &GraphNode{
			ID:               generateMethodID(variableName, []string{}, file),
			Type:             "variable_declaration",
			Name:             variableName,
			CodeSnippet:      node.Content(sourceCode),
			LineNumber:       node.StartPoint().Row + 1,
			Modifier:         extractVisibilityModifier(variableModifier),
			DataType:         variableType,
			Scope:            scope,
			VariableValue:    variableValue,
			hasAccess:        hasAccessValue,
			File:             file,
			isJavaSourceFile: isJavaSourceFile,
		}
		graph.AddNode(variableNode)
	}

	// Recursively process child nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		buildGraphFromAST(child, sourceCode, graph, currentContext, file)
	}

	// iterate through method declaration from graph node
	for _, node := range graph.Nodes {
		if node.Type == "method_declaration" {
			// iterate through method method_invocation from graph node
			for _, invokedNode := range graph.Nodes {
				if invokedNode.Type == "method_invocation" {
					if invokedNode.Name == node.Name {
						// check argument list count is same
						if len(invokedNode.MethodArgumentsValue) == len(node.MethodArgumentsType) {
							node.hasAccess = true
						}
					}
				}
			}
		}
	}
}

// write a function to generate unique method id from method name, class name, and package name, parameters, and return type.
func generateMethodID(methodName string, parameters []string, sourceFile string) string {
	// Example: Use the node type and its start byte position in the source code to generate a unique ID
	hashInput := fmt.Sprintf("%s-%s-%s", methodName, parameters, sourceFile)
	hash := sha256.Sum256([]byte(hashInput))
	return hex.EncodeToString(hash[:])
}

//nolint:all
func extractMethodName(node *sitter.Node, sourceCode []byte, filepath string) (string, string) {
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
	methodID = generateMethodID(methodName, parameters, filepath)
	return methodName, methodID
}

func getFiles(directory string) ([]string, error) {
	var files []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
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
	codeGraph := NewCodeGraph()
	// record start time
	start := time.Now()

	files, err := getFiles(directory)
	if err != nil {
		//nolint:all
		log.Println("Directory not found:", err)
		return codeGraph
	}

	totalFiles := len(files)
	numWorkers := 5 // Number of concurrent workers
	fileChan := make(chan string, totalFiles)
	resultChan := make(chan *CodeGraph, totalFiles)
	statusChan := make(chan string, numWorkers)
	progressChan := make(chan int, totalFiles)
	var wg sync.WaitGroup

	// Worker function
	worker := func(workerID int) {
		// Initialize the parser for each worker
		parser := sitter.NewParser()
		defer parser.Close()

		// Set the language (Java in this case)
		parser.SetLanguage(java.GetLanguage())

		for file := range fileChan {
			fileName := filepath.Base(file)
			statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Reading and parsing code %s\033[0m", workerID, fileName)
			sourceCode, err := readFile(file)
			if err != nil {
				log.Println("File not found:", err)
				continue
			}
			// Parse the source code
			tree, err := parser.ParseCtx(context.TODO(), nil, sourceCode)
			if err != nil {
				log.Println("Error parsing file:", err)
				continue
			}
			//nolint:all
			defer tree.Close()

			rootNode := tree.RootNode()
			localGraph := NewCodeGraph()
			statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Building graph and traversing code %s\033[0m", workerID, fileName)
			buildGraphFromAST(rootNode, sourceCode, localGraph, nil, file)
			statusChan <- fmt.Sprintf("\033[32mWorker %d ....... Done processing file %s\033[0m", workerID, fileName)

			resultChan <- localGraph
			progressChan <- 1
		}
		wg.Done()
	}

	// Start workers
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker(i + 1)
	}

	// Send files to workers
	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)

	// Status updater
	go func() {
		statusLines := make([]string, numWorkers)
		progress := 0
		for {
			select {
			case status, ok := <-statusChan:
				if !ok {
					return
				}
				workerID := int(status[12] - '0')
				statusLines[workerID-1] = status
			case _, ok := <-progressChan:
				if !ok {
					return
				}
				progress++
			}
			fmt.Print("\033[H\033[J") // Clear the screen
			for _, line := range statusLines {
				fmt.Println(line)
			}
			fmt.Printf("Progress: %d%%\n", (progress*100)/totalFiles)
		}
	}()

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
		close(statusChan)
		close(progressChan)
	}()

	// Collect results
	for localGraph := range resultChan {
		for _, node := range localGraph.Nodes {
			codeGraph.AddNode(node)
		}
		for _, edge := range localGraph.Edges {
			codeGraph.AddEdge(edge.From, edge.To)
		}
	}

	end := time.Now()
	elapsed := end.Sub(start)
	log.Println("Elapsed time: ", elapsed)
	log.Println("Graph built successfully")

	return codeGraph
}
