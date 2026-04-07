package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallSiteArguments_Populated(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

import "fmt"

func handler() {
	name := "world"
	fmt.Println(name)
}
`), 0644)

	codeGraph := graph.Initialize(tmpDir, nil)
	goRegistry, _ := resolution.BuildGoModuleRegistry(tmpDir)
	goTypeEngine := resolution.NewGoTypeInferenceEngine(goRegistry)

	callGraph, err := BuildGoCallGraph(codeGraph, goRegistry, goTypeEngine, nil)
	require.NoError(t, err)

	foundPrintln := false
	for _, sites := range callGraph.CallSites {
		for _, cs := range sites {
			if cs.Target == "Println" && cs.Resolved {
				foundPrintln = true
				require.NotEmpty(t, cs.Arguments, "Arguments should be populated")
				assert.Equal(t, 0, cs.Arguments[0].Position)
				assert.Equal(t, "name", cs.Arguments[0].Value)
				assert.True(t, cs.Arguments[0].IsVariable, "name is a variable")
			}
		}
	}
	assert.True(t, foundPrintln, "Should find Println call site")
}

func TestCallSiteArguments_MultipleArgs(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module testapp\n\ngo 1.21\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(`package main

import "fmt"

func handler() {
	x := "hello"
	fmt.Printf("%s %d", x, 42)
}
`), 0644)

	codeGraph := graph.Initialize(tmpDir, nil)
	goRegistry, _ := resolution.BuildGoModuleRegistry(tmpDir)

	callGraph, err := BuildGoCallGraph(codeGraph, goRegistry, nil, nil)
	require.NoError(t, err)

	for _, sites := range callGraph.CallSites {
		for _, cs := range sites {
			if cs.Target == "Printf" {
				t.Logf("Printf args: %v", cs.Arguments)
				// Should have arguments populated
				if len(cs.Arguments) > 0 {
					for _, arg := range cs.Arguments {
						t.Logf("  arg[%d]: %q isVar=%v", arg.Position, arg.Value, arg.IsVariable)
					}
				}
			}
		}
	}
}

func TestBuildCallSiteArguments(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantLen  int
		wantVars []bool
	}{
		{"nil args", nil, 0, nil},
		{"empty args", []string{}, 0, nil},
		{"single variable", []string{"x"}, 1, []bool{true}},
		{"string literal", []string{`"hello"`}, 1, []bool{false}},
		{"number literal", []string{"42"}, 1, []bool{false}},
		{"bool literal", []string{"true"}, 1, []bool{false}},
		{"nil literal", []string{"nil"}, 1, []bool{false}},
		{"mixed", []string{"x", `"hello"`, "42", "y"}, 4, []bool{true, false, false, true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildCallSiteArguments(tt.args)
			if tt.wantLen == 0 {
				assert.Nil(t, args)
				return
			}
			require.Len(t, args, tt.wantLen)
			for i, arg := range args {
				assert.Equal(t, i, arg.Position)
				assert.Equal(t, tt.args[i], arg.Value)
				if i < len(tt.wantVars) {
					assert.Equal(t, tt.wantVars[i], arg.IsVariable, "arg %d IsVariable", i)
				}
			}
		})
	}
}

func TestIsGoLiteral(t *testing.T) {
	tests := []struct {
		value    string
		expected bool
	}{
		{`"hello"`, true},
		{"`raw`", true},
		{"'c'", true},
		{"42", true},
		{"3.14", true},
		{"0xFF", true},
		{"true", true},
		{"false", true},
		{"nil", true},
		{"", true},
		{"x", false},
		{"myVar", false},
		{"db", false},
		{"r", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			assert.Equal(t, tt.expected, isGoLiteral(tt.value))
		})
	}
}
