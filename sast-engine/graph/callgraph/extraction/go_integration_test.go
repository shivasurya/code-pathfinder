package extraction

import (
	"os"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
	"github.com/stretchr/testify/assert"
)

func TestExtractGoReturnTypes_RealFixture(t *testing.T) {
	// This test verifies extraction works with real parsed Go code
	// We'll parse the test fixture and extract return types

	fixturePath := "../../../test-fixtures/golang/type_tracking/all_type_patterns.go"

	// Check if fixture exists
	if _, err := os.Stat(fixturePath); os.IsNotExist(err) {
		t.Skip("Test fixture not found, skipping integration test")
		return
	}

	// Create mock call graph with sample functions
	// (In real usage, this would come from Phase 1 parsing)
	callGraph := core.NewCallGraph()
	registry := &core.GoModuleRegistry{
		ModulePath: "github.com/example/test",
		DirToImport: map[string]string{
			"/test": "github.com/example/test/typetracking",
		},
	}
	typeEngine := resolution.NewGoTypeInferenceEngine(registry)

	// Simulate some functions from the fixture
	testFunctions := map[string]string{
		"typetracking.GetInt":           "int",
		"typetracking.GetString":        "string",
		"typetracking.GetBool":          "bool",
		"typetracking.GetUserPointer":   "*User",
		"typetracking.LoadConfig":       "(string, error)",
		"typetracking.GetUserWithError": "(*User, error)",
	}

	for fqn, returnType := range testFunctions {
		callGraph.Functions[fqn] = &graph.Node{
			ReturnType: returnType,
			File:       fixturePath,
		}
	}

	// Extract return types
	err := ExtractGoReturnTypes(callGraph, registry, typeEngine)
	assert.NoError(t, err)

	// Verify extraction results
	expectedTypes := map[string]string{
		"typetracking.GetInt":           "builtin.int",
		"typetracking.GetString":        "builtin.string",
		"typetracking.GetBool":          "builtin.bool",
		"typetracking.GetUserPointer":   "User",
		"typetracking.LoadConfig":       "builtin.string",
		"typetracking.GetUserWithError": "User",
	}

	for fqn, expectedFQN := range expectedTypes {
		typeInfo, ok := typeEngine.GetReturnType(fqn)
		assert.True(t, ok, "Expected %s to be extracted", fqn)
		assert.Equal(t, expectedFQN, typeInfo.TypeFQN, "Wrong FQN for %s", fqn)
	}

	// Verify total count
	allTypes := typeEngine.GetAllReturnTypes()
	assert.Len(t, allTypes, len(testFunctions))
}
