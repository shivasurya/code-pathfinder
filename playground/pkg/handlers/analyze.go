package handlers

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/shivasurya/code-pathfinder/playground/pkg/models"
	"github.com/shivasurya/code-pathfinder/playground/pkg/utils"
	parser "github.com/shivasurya/code-pathfinder/sast-engine/antlr"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
)

const (
	// QueryTimeout is the maximum time allowed for query execution
	QueryTimeout = 60 * time.Second
)

// Error definitions
var (
	ErrQueryTimeout = errors.New("query execution timed out")
)

// Channel types
type resultChannel chan []models.QueryResult

// AnalyzeHandler processes POST requests to /analyze endpoint.
// It accepts Java source code and a CodeQL query, executes the query using code-pathfinder,
// and returns the query results. The execution is done with a 60-second timeout.
func AnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer utils.LogRequestDuration("analyzeHandler", start)

	if !utils.ValidateMethod(w, r, http.MethodPost) {
		return
	}

	var req models.AnalyzeRequest
	if err := utils.DecodeJSONRequest(w, r, &req); err != nil {
		return
	}

	tmpDir, err := utils.CreateTempWorkspace("code-analysis-*")
	if err != nil {
		utils.SendErrorResponse(w, "Failed to create temporary directory", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	if err := utils.WriteSourceAndQueryFiles(tmpDir, req.JavaSource, req.Query); err != nil {
		utils.SendErrorResponse(w, "Failed to write files", err)
		return
	}

	results, err := executeQueryWithTimeout(tmpDir, req.Query)
	if err != nil {
		utils.SendErrorResponse(w, "Query execution failed", err)
		return
	}

	utils.SendJSONResponse(w, models.AnalyzeResponse{
		Results: results,
	})
}

// executeQueryWithTimeout executes the query with a timeout
func executeQueryWithTimeout(tmpDir, queryStr string) ([]models.QueryResult, error) {
	resultsChan := make(resultChannel)
	errorChan := make(chan error)

	go executeQuery(tmpDir, queryStr, resultsChan, errorChan)

	select {
	case results := <-resultsChan:
		return results, nil
	case err := <-errorChan:
		return nil, err
	case <-time.After(QueryTimeout):
		return nil, ErrQueryTimeout
	}
}

// executeQuery performs the actual query execution
func executeQuery(tmpDir, queryStr string, resultsChan resultChannel, errorChan chan error) {
	// Initialize code graph with the temporary directory
	codeGraph := graph.Initialize(tmpDir)

	// Parse the query
	parsedQuery, err := parser.ParseQuery(queryStr)
	if err != nil {
		errorChan <- err
		return
	}

	// Extract WHERE clause
	parts := strings.SplitN(queryStr, "WHERE", 2)
	if len(parts) > 1 {
		parsedQuery.Expression = strings.SplitN(parts[1], "SELECT", 2)[0]
	}

	// Execute query on the graph
	entities, _ := graph.QueryEntities(codeGraph, parsedQuery)

	// Convert results to QueryResult format
	var results []models.QueryResult
	for _, entity := range entities {
		for _, entityObject := range entity {

			// Create QueryResult
			result := models.QueryResult{
				File:    "Main.java",
				Line:    int64(entityObject.LineNumber),
				Snippet: entityObject.CodeSnippet,
				Kind:    entityObject.Type, // Use the Type field from entityObject
			}

			// Add the result to the list
			results = append(results, result)
		}
	}

	resultsChan <- results
}
