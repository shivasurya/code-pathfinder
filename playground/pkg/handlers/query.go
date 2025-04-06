package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

type QueryRequest struct {
	Code     string `json:"code"`
	Rule     string `json:"rule"`
	Language string `json:"language"`
}

type QueryResponse struct {
	Results []QueryResult `json:"results"`
}

type QueryResult struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// QueryHandler handles requests to analyze code against a CodeQL query
func QueryHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual CodeQL analysis
	// For now, return mock results
	results := []QueryResult{
		{
			Line:    3,
			Message: "Found potential security issue",
		},
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(QueryResponse{
		Results: results,
	})
}
