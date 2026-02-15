package handlers

import (
	"fmt"
	"net/http"

	"github.com/example/callgraph/utils"
)

// HandleRequest processes HTTP requests with validation.
func HandleRequest(w http.ResponseWriter, r *http.Request) {
	userInput := r.URL.Query().Get("input")

	// Call to utils package function
	if !utils.Helper(userInput) {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	// Closure example
	process := func(data string) string {
		// Closure calls stdlib
		return fmt.Sprintf("Processed: %s", data)
	}

	result := process(userInput)
	fmt.Fprintf(w, result)
}

// ProcessData validates and processes data.
func ProcessData(data []string) []string {
	var results []string

	for _, item := range data {
		if utils.ValidateLength(item, 3) {
			results = append(results, item)
		}
	}

	return results
}
