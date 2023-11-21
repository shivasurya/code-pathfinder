// api.go

package main

import (
    "encoding/json"
    "net/http"
)


func startServer(graph *CodeGraph) {
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
        json.NewEncoder(w).Encode(result)
    })

    http.ListenAndServe(":8080", nil)
}
