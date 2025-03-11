package handlers

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/shivasurya/code-pathfinder/playground/pkg/models"
	"github.com/shivasurya/code-pathfinder/playground/pkg/utils"
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

	utils.SendJSONResponse(w, models.AnalyzeResponse{Results: results})
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
	// TODO: This is a placeholder implementation
	// In a real implementation, this would use the code-pathfinder library
	// to execute the query and analyze the code

	// For now, we'll look for WebView security issues in the source code
	var results []models.QueryResult

	// Add a dummy result for testing
	results = append(results, models.QueryResult{
		File:    "MainActivity.java",
		Line:    42,
		Snippet: "setJavaScriptEnabled(true)",
	})

	resultsChan <- results
}
