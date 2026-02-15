package main

import (
	"fmt"
	"net/http"

	"github.com/example/callgraph/handlers"
)

func main() {
	// Stdlib call
	fmt.Println("Starting server...")

	// Setup routes
	http.HandleFunc("/process", handlers.HandleRequest)

	// Same-package call
	testData := prepareTestData()

	// Builtin usage
	moreData := append(testData, "extra")

	fmt.Printf("Test data: %v\n", moreData)

	// Start server
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// prepareTestData creates sample data for testing.
func prepareTestData() []string {
	data := []string{"test1", "test2"}

	// Process through handlers
	processed := handlers.ProcessData(data)

	// Builtin usage
	return append(processed, "test3")
}
