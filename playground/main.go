// Package main implements a web server for analyzing Java source code and executing CodeQL queries.
// It provides endpoints for code analysis, AST parsing, and visualization.
package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/shivasurya/code-pathfinder/playground/pkg/handlers"
	"github.com/shivasurya/code-pathfinder/playground/pkg/middleware"
)

func main() {
	// Create a new mux for better control over middleware
	mux := http.NewServeMux()

	// Serve static files with security and logging middleware
	fs := http.FileServer(http.Dir("public/static"))
	mux.Handle("/", middleware.LoggingMiddleware(fs))

	// API endpoints with security, logging, and CORS middleware
	mux.Handle("/api/analyze", middleware.CorsMiddleware(middleware.LoggingMiddleware(http.HandlerFunc(handlers.AnalyzeHandler))))
	mux.Handle("/api/parse", middleware.CorsMiddleware(middleware.LoggingMiddleware(http.HandlerFunc(handlers.ParseHandler))))

	// Get port from environment variable or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Ensure port starts with :
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	log.Printf("Starting server on port %s", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
