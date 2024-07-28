package main

import (
	"testing"
)

func TestProcessQuery(t *testing.T) {
	graph := NewCodeGraph()
	output := "json"
	input := "FIND method_declaration AS md WHERE md.getVisibility() == \"public\""

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
	// get project from command line
	project := "../test-src/"
	query := "FIND method_declaration AS md WHERE md.getName() == \"onCreate\""
	output := "json"

	result, err := executeCLIQuery(project, query, output, false)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Here you can add assertions based on what you expect the result to be.
	// This will depend on the specifics of your executeProject function.
	if result == "" {
		t.Errorf("Expected result to be non-empty")
	}
}
