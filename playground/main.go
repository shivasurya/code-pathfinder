// Package main implements a web server for analyzing Java source code and executing CodeQL queries.
// It provides endpoints for code analysis, AST parsing, and visualization.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	parser "github.com/shivasurya/code-pathfinder/sourcecode-parser/antlr"
	"github.com/shivasurya/code-pathfinder/sourcecode-parser/graph"
)

const (
	// QueryTimeout is the maximum time allowed for query execution
	QueryTimeout = 60 * time.Second

	// FilePermissions for created files
	FilePermissions = 0644
)

// Request/Response Types
type (
	// AnalyzeRequest represents the input for code analysis
	AnalyzeRequest struct {
		JavaSource string `json:"javaSource"`
		Query      string `json:"query"`
	}

	// QueryResult represents a single result from code analysis
	QueryResult struct {
		File    string `json:"file"`
		Line    int    `json:"line"`
		Snippet string `json:"snippet"`
	}

	// AnalyzeResponse represents the response from code analysis
	AnalyzeResponse struct {
		Results []QueryResult `json:"results"`
		Error   string        `json:"error,omitempty"`
	}

	// ParseRequest represents the input for AST parsing
	ParseRequest struct {
		JavaSource string `json:"javaSource"`
	}

	// ASTNode represents a node in the Abstract Syntax Tree
	ASTNode struct {
		Type     string    `json:"type"`
		Name     string    `json:"name,omitempty"`
		Value    string    `json:"value,omitempty"`
		Line     int       `json:"line"`
		Children []ASTNode `json:"children,omitempty"`
	}

	// ParseResponse represents the response from AST parsing
	ParseResponse struct {
		AST   *ASTNode `json:"ast"`
		Error string   `json:"error,omitempty"`
	}
)

// Channel types
type resultChannel chan []QueryResult

// HTTP Handlers

// analyzeHandler processes POST requests to /analyze endpoint.
// It accepts Java source code and a CodeQL query, executes the query using code-pathfinder,
// and returns the query results. The execution is done with a 60-second timeout.
func analyzeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer logRequestDuration("analyzeHandler", start)

	if !validateMethod(w, r, http.MethodPost) {
		return
	}

	var req AnalyzeRequest
	if err := decodeJSONRequest(w, r, &req); err != nil {
		return
	}

	tmpDir, err := createTempWorkspace("code-analysis-*")
	if err != nil {
		sendErrorResponse(w, "Failed to create temporary directory", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	if err := writeSourceAndQueryFiles(tmpDir, req.JavaSource, req.Query); err != nil {
		sendErrorResponse(w, "Failed to write files", err)
		return
	}

	results, err := executeQueryWithTimeout(tmpDir, req.Query)
	if err != nil {
		sendErrorResponse(w, "Query execution failed", err)
		return
	}

	sendJSONResponse(w, AnalyzeResponse{Results: results})
}

// parseHandler processes POST requests to /parse endpoint.
// It accepts Java source code, parses it into an AST using code-pathfinder,
// and returns the AST structure for visualization.
func parseHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer logRequestDuration("parseHandler", start)

	if !validateMethod(w, r, http.MethodPost) {
		return
	}

	var req ParseRequest
	if err := decodeJSONRequest(w, r, &req); err != nil {
		return
	}

	tmpDir, err := createTempWorkspace("ast-parse-*")
	if err != nil {
		sendErrorResponse(w, "Failed to create temporary directory", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	if err := writeSourceFile(tmpDir, req.JavaSource); err != nil {
		sendErrorResponse(w, "Failed to write source file", err)
		return
	}

	codeGraph := graph.Initialize(tmpDir)
	if codeGraph == nil {
		sendErrorResponse(w, "Failed to initialize code graph", nil)
		return
	}

	ast := buildAST(codeGraph)
	sendJSONResponse(w, ParseResponse{AST: ast})
}

// Helper Functions

// logRequestDuration logs the duration of a request
func logRequestDuration(handler string, start time.Time) {
	log.Printf("[%s] Completed in %v", handler, time.Since(start))
}

// validateMethod checks if the request method matches the expected method
func validateMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		log.Printf("[%s] Invalid method %s", r.URL.Path, r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return false
	}
	return true
}

// decodeJSONRequest decodes the JSON request body into the target struct
func decodeJSONRequest(w http.ResponseWriter, r *http.Request, target interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(target); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return err
	}
	return nil
}

// createTempWorkspace creates a temporary directory for analysis
func createTempWorkspace(prefix string) (string, error) {
	return os.MkdirTemp("", prefix)
}

// writeSourceAndQueryFiles writes the source and query files to the workspace
func writeSourceAndQueryFiles(dir, source, query string) error {
	if err := writeSourceFile(dir, source); err != nil {
		return err
	}
	return writeFile(filepath.Join(dir, "query.cql"), query)
}

// writeSourceFile writes the Java source code to a file
func writeSourceFile(dir, source string) error {
	return writeFile(filepath.Join(dir, "Source.java"), source)
}

// writeFile writes content to a file with proper permissions
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), FilePermissions)
}

// executeQueryWithTimeout executes the query with a timeout
func executeQueryWithTimeout(tmpDir, queryStr string) ([]QueryResult, error) {
	resultsChan := make(resultChannel, 1)
	errorChan := make(chan error, 1)

	go executeQuery(tmpDir, queryStr, resultsChan, errorChan)

	select {
	case results := <-resultsChan:
		return results, nil
	case err := <-errorChan:
		return nil, err
	case <-time.After(QueryTimeout):
		return nil, fmt.Errorf("query execution timed out after %v", QueryTimeout)
	}
}

// executeQuery performs the actual query execution
func executeQuery(tmpDir, queryStr string, resultsChan resultChannel, errorChan chan error) {
	codeGraph := graph.Initialize(tmpDir)
	if codeGraph == nil {
		errorChan <- fmt.Errorf("failed to initialize code graph")
		return
	}

	parsedQuery, err := parser.ParseQuery(queryStr)
	if err != nil {
		errorChan <- fmt.Errorf("failed to parse query: %v", err)
		return
	}

	entities, _ := graph.QueryEntities(codeGraph, parsedQuery)
	results := formatQueryResults(entities)
	resultsChan <- results
}

// formatQueryResults formats the query results into the response structure
func formatQueryResults(entities [][]*graph.Node) []QueryResult {
	results := make([]QueryResult, 0)
	for _, entity := range entities {
		for _, node := range entity {
			if node != nil {
				results = append(results, QueryResult{
					File:    node.File,
					Line:    int(node.LineNumber),
					Snippet: node.CodeSnippet,
				})
			}
		}
	}
	return results
}

// sendErrorResponse sends an error response to the client
func sendErrorResponse(w http.ResponseWriter, msg string, err error) {
	errMsg := msg
	if err != nil {
		errMsg += ": " + err.Error()
	}
	w.WriteHeader(http.StatusInternalServerError)
	sendJSONResponse(w, AnalyzeResponse{Error: errMsg})
}

// sendJSONResponse sends a JSON response to the client
func sendJSONResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
// buildAST converts a CodeGraph into an AST structure
func buildAST(codeGraph *graph.CodeGraph) *ASTNode {
	if codeGraph == nil {
		return nil
	}

	// Create root node for the compilation unit
	root := &ASTNode{
		Type:     "CompilationUnit",
		Children: make([]ASTNode, 0),
	}

	// Process all nodes
	for _, node := range codeGraph.Nodes {
		// Skip non-declaration nodes
		if !strings.Contains(node.Type, "Declaration") {
			continue
		}

		// Create node based on type
		astNode := ASTNode{
			Type:     node.Type,
			Name:     node.Name,
			Line:     int(node.LineNumber),
			Children: make([]ASTNode, 0),
		}

		// Add to root
		root.Children = append(root.Children, astNode)
	}

	return root
}

func main() {
	// Serve static files
	fs := http.FileServer(http.Dir("public/static"))
	http.Handle("/", fs)

	// API endpoints
	http.HandleFunc("/analyze", analyzeHandler)
	http.HandleFunc("/parse", parseHandler)

	port := ":8080"
	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
