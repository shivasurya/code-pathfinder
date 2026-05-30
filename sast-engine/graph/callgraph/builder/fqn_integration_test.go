package builder

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countPythonFqns returns how many python rows fqn_index holds.
func countPythonFqns(t *testing.T, cache *AnalysisCache) int {
	t.Helper()
	var n int
	require.NoError(t, cache.db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM fqn_index WHERE language='python'`).Scan(&n))
	return n
}

// TestBuildCallGraphWithCache_PersistsPythonFqnIndex drives the whole PR-02 path
// in-process: parse a real Python project, build the call graph with a cache,
// and assert the discovered definitions landed in fqn_index with their files
// stamped for staleness.
func TestBuildCallGraphWithCache_PersistsPythonFqnIndex(t *testing.T) {
	dir := t.TempDir()
	src := `GLOBAL = "x"

def sanitize(value):
    return value.strip()

class User:
    def __init__(self, name):
        self.name = name

    def save(self):
        return sanitize(self.name)
`
	modelsPy := filepath.Join(dir, "models.py")
	require.NoError(t, os.WriteFile(modelsPy, []byte(src), 0o644))

	codeGraph := graph.Initialize(dir, nil)
	require.NotNil(t, codeGraph)
	moduleRegistry, err := registry.BuildModuleRegistry(dir, false)
	require.NoError(t, err)

	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	defer cache.Close()

	logger := output.NewLogger(output.VerbosityDefault)
	cg, err := BuildCallGraphWithCache(codeGraph, moduleRegistry, dir, logger, cache)
	require.NoError(t, err)
	require.NotNil(t, cg)

	// The function, the class, and both methods should be indexed.
	assert.GreaterOrEqual(t, countPythonFqns(t, cache), 4, "function + class + two methods expected")

	for _, fqn := range []string{"models.sanitize", "models.User", "models.User.save", "models.User.__init__"} {
		var got string
		err := cache.db.QueryRowContext(context.Background(),
			`SELECT fqn FROM fqn_index WHERE fqn=? AND language='python'`, fqn).Scan(&got)
		require.NoError(t, err, "expected %s in fqn_index", fqn)
	}

	// Every indexed entry is project source.
	var nonProject int
	require.NoError(t, cache.db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM fqn_index WHERE language='python' AND source<>'project'`).Scan(&nonProject))
	assert.Zero(t, nonProject)

	// models.py is stamped for staleness, and a fresh check sees it as current.
	stale, err := cache.IsStale(modelsPy)
	require.NoError(t, err)
	assert.False(t, stale, "the file just indexed should not be stale")
}

// TestBuildCallGraph_NilCacheWritesNothing verifies the cache-disabled branch:
// the no-cache wrapper builds an identical graph and persists nothing.
func TestBuildCallGraph_NilCacheWritesNothing(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "m.py"), []byte("def f():\n    return 1\n"), 0o644))

	codeGraph := graph.Initialize(dir, nil)
	moduleRegistry, err := registry.BuildModuleRegistry(dir, false)
	require.NoError(t, err)

	logger := output.NewLogger(output.VerbosityDefault)
	cg, err := BuildCallGraph(codeGraph, moduleRegistry, dir, logger)
	require.NoError(t, err)
	assert.NotEmpty(t, cg.Functions, "the graph is still built without a cache")

	// A separately opened cache for the same project has no python rows.
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	defer cache.Close()
	assert.Zero(t, countPythonFqns(t, cache), "no cache passed -> nothing persisted")
}
