package builder

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStdlibReturnTypeChaining_EndToEnd verifies that the full pipeline
// resolves stdlib return-type chains correctly.  For the fixture:
//
//	conn   = sqlite3.connect("test.db")   → type sqlite3.Connection
//	cursor = conn.cursor()                → type sqlite3.Cursor
//
// Requires CDN data regenerated with typeshed overlay for C builtin return types.
//	cursor.execute(...)                   → FQN contains "sqlite3"
func TestStdlibReturnTypeChaining_EndToEnd(t *testing.T) {
	t.Skip("Requires CDN data regenerated with typeshed overlay — sqlite3.connect() return type is 'unknown' in current CDN")
	// Use absolute path to ensure graph.Initialize and BuildCallGraphFromPath
	// use consistent file paths (module registry converts to absolute internally).
	projectPath, err := filepath.Abs("../../../../test-fixtures/querytype-poc")
	require.NoError(t, err)

	// Build code graph from the fixture
	codeGraph := graph.Initialize(projectPath, nil)
	require.NotNil(t, codeGraph)

	// Build call graph (includes stdlib registry loading, type inference, etc.)
	logger := output.NewLogger(output.VerbosityDefault)
	callGraph, _, err := BuildCallGraphFromPath(codeGraph, projectPath, logger)
	require.NoError(t, err)
	require.NotNil(t, callGraph)

	// Search all call sites for a target matching "execute" whose
	// TargetFQN contains "sqlite3" — this proves the return-type chain
	// was resolved through connect → Connection, cursor → Cursor.
	foundSqlite3Execute := false
	for caller, sites := range callGraph.CallSites {
		for _, cs := range sites {
			if strings.Contains(cs.Target, "execute") &&
				strings.Contains(cs.TargetFQN, "sqlite3") {
				t.Logf("Found sqlite3 execute: caller=%s target=%s fqn=%s",
					caller, cs.Target, cs.TargetFQN)
				foundSqlite3Execute = true
			}
		}
	}

	assert.True(t, foundSqlite3Execute,
		"Expected at least one call site with Target containing 'execute' "+
			"and TargetFQN containing 'sqlite3', proving stdlib return-type chaining works")
}
