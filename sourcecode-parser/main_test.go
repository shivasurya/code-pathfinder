package main

import (
	"testing"
)

func TestProcessQuery(t *testing.T) {
	graph := NewCodeGraph()
	output := "json"
	input := "FIND variable_declaration WHERE visibility = 'private'"

	result, err := processQuery(input, graph, output)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Here you can add assertions based on what you expect the result to be.
	// This will depend on the specifics of your processQuery function.
	if result == "" {
		t.Errorf("Expected result to be non-empty")
	}
}

func TestExecuteProject(t *testing.T) {
	project := "/Users/shiva/src/Stirling-PDF"
	query := "FIND variable_declaration WHERE visibility = 'private'"
	output := "json"

	result, err := executeProject(project, query, output, false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Here you can add assertions based on what you expect the result to be.
	// This will depend on the specifics of your executeProject function.
	if result == "" {
		t.Errorf("Expected result to be non-empty")
	}
}
