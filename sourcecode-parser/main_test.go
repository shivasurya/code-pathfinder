package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

func TestProcessQuery(t *testing.T) {
	graph := NewCodeGraph()
	output := "json"
	input := "FIND method_declaration AS md WHERE md.GetVisibility() == \"public\""

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

func TestExecuteCLIQuery(t *testing.T) {
	// get project from command line
	project := "../test-src/"
	queries := []string{
		`FIND method_declaration AS md WHERE md.getName() == "onCreate"`,
		`FIND variable_declaration AS vd WHERE vd.getVisibility() == "private"`,
	}
	output := "json"

	for _, query := range queries {
		fmt.Println(query)
		result, err := executeCLIQuery(project, query, output, false)
		if err != nil {
			t.Errorf("Expected no error for query %s, got %v", query, err)
		}
		var resultMap []map[string]interface{}
		err = json.Unmarshal([]byte(result), &resultMap)
		if len(resultMap) == 0 {
			t.Errorf("Expected result to be non-empty for query %s", query)
		}
		if err != nil {
			t.Errorf("Expected no error for query %s, got %v", query, err)
		}

		// Here you can add assertions based on what you expect the result to be.
		// This will depend on the specifics of your executeProject function.
		if result == "" {
			t.Errorf("Expected result to be non-empty for query %s", query)
		}
	}
}

func TestInitializeProject(t *testing.T) {
	tests := []struct {
		name    string
		project string
		want    *CodeGraph
	}{
		{
			name:    "Empty project",
			project: "",
			want:    NewCodeGraph(),
		},
		{
			name:    "Valid project",
			project: "../test-src/",
			want:    Initialize("../test-src/"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := InitializeProject(tt.project)
			if got == nil {
				t.Errorf("InitializeProject() returned nil")
			}
			if tt.project == "" && len(got.Nodes) != 0 {
				t.Errorf("InitializeProject() with empty project should return empty graph")
			}
			if tt.project != "" && len(got.Nodes) == 0 {
				t.Errorf("InitializeProject() with valid project should return non-empty graph")
			}
		})
	}
}

func TestInitializeProjectWithInvalidPath(t *testing.T) {
	invalidProject := "/path/to/nonexistent/project"
	got := InitializeProject(invalidProject)
	if got == nil || got.Nodes == nil {
		t.Errorf("InitializeProject() with invalid path should return empty graph, not nil")
	} else if len(got.Nodes) != 0 {
		t.Errorf("InitializeProject() with invalid path should return empty graph")
	}
}

func TestInitializeProjectConsistency(t *testing.T) {
	project := "../test-src/"
	graph1 := InitializeProject(project)
	graph2 := InitializeProject(project)

	if len(graph1.Nodes) != len(graph2.Nodes) {
		t.Errorf("InitializeProject() should return consistent results for the same project")
	}
}

func TestMainFunctionality(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedExit int
		expectedOut  string
	}{
		{
			name:         "Version flag",
			args:         []string{"--version"},
			expectedExit: 0,
			expectedOut:  "Version: " + Version + "\nGit Commit: " + GitCommit + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()

			os.Args = append([]string{"cmd"}, tt.args...)

			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			main()

			err := w.Close()
			if err != nil {
				return
			}
			os.Stdout = oldStdout
			out, _ := io.ReadAll(r)

			if !strings.Contains(string(out), tt.expectedOut) {
				t.Errorf("Expected output to contain %q, got %q", tt.expectedOut, string(out))
			}

		})
	}
}
