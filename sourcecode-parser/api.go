// api.go

package main

import (
	"encoding/json"
	"net/http"
)

func StartServer(graph *CodeGraph) {
	http.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		// For simplicity, let's return all nodes. You can add query params to filter nodes.
		json.NewEncoder(w).Encode(graph.Nodes)
	})

	http.HandleFunc("/source-sink-analysis", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		sourceMethod := query.Get("sourceMethod")
		sinkMethod := query.Get("sinkMethod")

		if sourceMethod == "" || sinkMethod == "" {
			http.Error(w, "sinkMethod and sourceMethod query parameters are required", http.StatusBadRequest)
			return
		}

		result := AnalyzeSourceSinkPatterns(graph, sourceMethod, sinkMethod)
		// Return the result as JSON
		// set json content type
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})

	// create a http handler to respond with index.html file content
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Read the index.html file from html directory
		http.ServeFile(w, r, "html/index.html")
	})

	http.ListenAndServe(":8080", nil)
}
