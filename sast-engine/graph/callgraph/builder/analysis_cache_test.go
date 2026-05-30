package builder

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMain points default-location index resolution at a throwaway directory so
// the package's tests never write SQLite files into the developer's real
// $HOME/.codepathfinder. It deliberately does not touch $HOME itself: other
// resolvers in this package read $HOME to locate GOMODCACHE.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "cpf-index-tests")
	if err != nil {
		panic(err)
	}
	indexParentDirOverride = dir
	code := m.Run()
	indexParentDirOverride = ""
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// ---- helpers ----

func openTempCache(t *testing.T) *AnalysisCache {
	t.Helper()
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cache.Close() })
	return cache
}

func writeTempGoFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(p, []byte(content), 0o644))
	return p
}

// ---- OpenAnalysisCache / Close ----

func TestAnalysisCache_OpenClose(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	require.NotNil(t, cache)
	require.NoError(t, cache.Close())
}

// TestAnalysisCache_OpenClose_InvalidDir verifies that an index path whose
// parent cannot be created returns an error instead of panicking. A regular
// file is planted where the index directory would go, so os.MkdirAll fails.
func TestAnalysisCache_OpenClose_InvalidDir(t *testing.T) {
	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	// IndexPath points inside the blocking file, so MkdirAll on its parent fails.
	_, err := OpenAnalysisCacheWithOptions(CacheOptions{
		ProjectRoot: tmp,
		IndexPath:   filepath.Join(blocker, "sub", "index.sqlite"),
	})
	assert.Error(t, err)
}

// ---- DBPath ----

func TestAnalysisCache_DBPath(t *testing.T) {
	cache := openTempCache(t)
	// DBPath returns the resolved index file. With the test override active it
	// lives under the isolated temp dir and ends in .sqlite.
	path := cache.DBPath()
	assert.Equal(t, indexParentDirOverride, filepath.Dir(path))
	assert.True(t, strings.HasSuffix(path, ".sqlite"), "got %q", path)
}

// ---- Per-table version wipe tests ----

// TestAnalysisCache_PerTableVersionWipe verifies that bumping an individual
// table's version wipes only that table while leaving others intact.
func TestAnalysisCache_PerTableVersionWipe(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)

	goFile := writeTempGoFile(t, dir, "wipe.go", "package main\n")
	cs := []CachedCallSite{{CallerFQN: "pkg.Fn", FunctionName: "f", CallerFile: goFile, CallLine: 1}}
	sc := &CachedScope{FunctionScopes: make(map[string]CachedFunctionScope)}
	require.NoError(t, cache.PutFileCached(goFile, cs, sc))
	require.NoError(t, cache.SaveFunctionIndex(map[string][]string{goFile: {"pkg.Fn"}}))

	// Simulate a future upgrade to pass4 schema only.
	_, err = cache.db.ExecContext(context.Background(),
		"INSERT OR REPLACE INTO meta(key,value) VALUES('pass4_version','999')")
	require.NoError(t, err)
	require.NoError(t, cache.Close())

	cache2, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	defer cache2.Close()

	// file_cache should still be warm.
	_, hit := cache2.GetFileCached(goFile)
	assert.True(t, hit, "file_cache should survive a pass4_version bump")

	// function_index should still have data.
	idx := cache2.LoadFunctionIndex()
	assert.NotEmpty(t, idx, "function_index should survive a pass4_version bump")
}

// TestAnalysisCache_FileCacheVersionWipe verifies that bumping file_cache_version
// wipes only file_cache, leaving function_index intact.
func TestAnalysisCache_FileCacheVersionWipe(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)

	goFile := writeTempGoFile(t, dir, "fc_wipe.go", "package main\n")
	cs := []CachedCallSite{{CallerFQN: "pkg.Fn", FunctionName: "f", CallerFile: goFile, CallLine: 1}}
	sc := &CachedScope{FunctionScopes: make(map[string]CachedFunctionScope)}
	require.NoError(t, cache.PutFileCached(goFile, cs, sc))
	require.NoError(t, cache.SaveFunctionIndex(map[string][]string{goFile: {"pkg.Fn"}}))

	// Bump file_cache_version to simulate a future schema upgrade.
	_, err = cache.db.ExecContext(context.Background(),
		"INSERT OR REPLACE INTO meta(key,value) VALUES('file_cache_version','999')")
	require.NoError(t, err)
	require.NoError(t, cache.Close())

	cache2, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	defer cache2.Close()

	// file_cache wiped.
	_, hit := cache2.GetFileCached(goFile)
	assert.False(t, hit, "file_cache should be wiped on file_cache_version bump")

	// function_index intact.
	idx := cache2.LoadFunctionIndex()
	assert.NotEmpty(t, idx, "function_index should survive a file_cache_version bump")
}

// TestAnalysisCache_ProjectRootChange verifies that a different project_root
// stored in meta causes all tables to be wiped.
func TestAnalysisCache_ProjectRootChange(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)

	goFile := writeTempGoFile(t, dir, "root_change.go", "package main\n")
	cs := []CachedCallSite{{CallerFQN: "pkg.Fn", FunctionName: "f", CallerFile: goFile, CallLine: 1}}
	sc := &CachedScope{FunctionScopes: make(map[string]CachedFunctionScope)}
	require.NoError(t, cache.PutFileCached(goFile, cs, sc))

	// Simulate a different project_root stored in meta.
	_, err = cache.db.ExecContext(context.Background(),
		"UPDATE meta SET value='/other/project' WHERE key='project_root'")
	require.NoError(t, err)
	require.NoError(t, cache.Close())

	cache2, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	defer cache2.Close()

	_, hit := cache2.GetFileCached(goFile)
	assert.False(t, hit, "file_cache should be wiped on project_root change")
}

// ---- GetFileCached / PutFileCached ----

func TestAnalysisCache_GetFileCached_Miss(t *testing.T) {
	cache := openTempCache(t)
	dir := t.TempDir()
	f := writeTempGoFile(t, dir, "foo.go", "package main\n")

	result, hit := cache.GetFileCached(f)
	assert.False(t, hit)
	assert.Nil(t, result)
}

func TestAnalysisCache_GetFileCached_NonExistentFile(t *testing.T) {
	cache := openTempCache(t)
	result, hit := cache.GetFileCached("/nonexistent/path/file.go")
	assert.False(t, hit)
	assert.Nil(t, result)
}

func TestAnalysisCache_PutGet_RoundTrip(t *testing.T) {
	cache := openTempCache(t)
	dir := t.TempDir()
	goFile := writeTempGoFile(t, dir, "sample.go", "package main\nfunc Foo(){}\n")

	callSites := []CachedCallSite{
		{
			CallerFQN:    "mypkg.Foo",
			CallerFile:   goFile,
			CallLine:     5,
			FunctionName: "Bar",
			ObjectName:   "baz",
			Arguments:    []string{"a", "b"},
		},
	}
	scope := &CachedScope{
		FunctionScopes: map[string]CachedFunctionScope{
			"mypkg.Foo": {
				FunctionFQN: "mypkg.Foo",
				Variables: map[string]CachedBinding{
					"x": {TypeFQN: "mypkg.MyType", Confidence: 0.9, Source: "literal", AssignedFrom: "mypkg.NewMyType"},
				},
			},
		},
	}

	require.NoError(t, cache.PutFileCached(goFile, callSites, scope))

	result, hit := cache.GetFileCached(goFile)
	require.True(t, hit)
	require.NotNil(t, result)

	require.Len(t, result.CallSites, 1)
	cs := result.CallSites[0]
	assert.Equal(t, "mypkg.Foo", cs.CallerFQN)
	assert.Equal(t, goFile, cs.CallerFile)
	assert.Equal(t, uint32(5), cs.CallLine)
	assert.Equal(t, "Bar", cs.FunctionName)
	assert.Equal(t, "baz", cs.ObjectName)
	assert.Equal(t, []string{"a", "b"}, cs.Arguments)

	require.NotNil(t, result.Scope)
	fs, ok := result.Scope.FunctionScopes["mypkg.Foo"]
	require.True(t, ok)
	binding, ok := fs.Variables["x"]
	require.True(t, ok)
	assert.Equal(t, "mypkg.MyType", binding.TypeFQN)
	assert.InDelta(t, 0.9, binding.Confidence, 0.001)
	assert.Equal(t, "literal", binding.Source)
	assert.Equal(t, "mypkg.NewMyType", binding.AssignedFrom)
}

func TestAnalysisCache_ContentChange_Invalidates(t *testing.T) {
	cache := openTempCache(t)
	dir := t.TempDir()
	goFile := writeTempGoFile(t, dir, "change.go", "package main\n")

	cs := []CachedCallSite{{CallerFQN: "pkg.Fn", FunctionName: "bar", CallerFile: goFile, CallLine: 1}}
	sc := &CachedScope{FunctionScopes: make(map[string]CachedFunctionScope)}
	require.NoError(t, cache.PutFileCached(goFile, cs, sc))

	_, hit := cache.GetFileCached(goFile)
	require.True(t, hit)

	require.NoError(t, os.WriteFile(goFile, []byte("package main\nfunc Extra(){}\n"), 0o644))

	result, hit := cache.GetFileCached(goFile)
	assert.False(t, hit)
	assert.Nil(t, result)
}

func TestAnalysisCache_PutFileCached_NonExistentFile(t *testing.T) {
	cache := openTempCache(t)
	err := cache.PutFileCached("/no/such/file.go", nil, &CachedScope{FunctionScopes: map[string]CachedFunctionScope{}})
	assert.Error(t, err, "hashing a non-existent file should fail")
}

// ---- hashFile ----

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "hf.go")
	require.NoError(t, os.WriteFile(f, []byte("hello"), 0o644))
	h, err := hashFile(f)
	require.NoError(t, err)
	assert.Len(t, h, 64)
}

func TestHashFile_NonExistent(t *testing.T) {
	_, err := hashFile("/nonexistent/file.go")
	assert.Error(t, err)
}

// ---- LoadFunctionIndex / SaveFunctionIndex ----

func TestFunctionIndex_Empty(t *testing.T) {
	cache := openTempCache(t)
	idx := cache.LoadFunctionIndex()
	assert.Empty(t, idx)
}

func TestFunctionIndex_RoundTrip(t *testing.T) {
	cache := openTempCache(t)

	input := map[string][]string{
		"/project/pkg/a.go": {"pkg.FuncA", "pkg.FuncB"},
		"/project/pkg/b.go": {"pkg.FuncC"},
	}
	require.NoError(t, cache.SaveFunctionIndex(input))

	out := cache.LoadFunctionIndex()
	require.Len(t, out, 2)

	assert.ElementsMatch(t, []string{"pkg.FuncA", "pkg.FuncB"}, out["/project/pkg/a.go"])
	assert.ElementsMatch(t, []string{"pkg.FuncC"}, out["/project/pkg/b.go"])
}

func TestFunctionIndex_SaveReplacesPrevious(t *testing.T) {
	cache := openTempCache(t)

	first := map[string][]string{"/a.go": {"pkg.Old"}}
	require.NoError(t, cache.SaveFunctionIndex(first))

	second := map[string][]string{"/b.go": {"pkg.New1", "pkg.New2"}}
	require.NoError(t, cache.SaveFunctionIndex(second))

	out := cache.LoadFunctionIndex()
	// Old entry must be gone; only second snapshot remains.
	assert.Nil(t, out["/a.go"])
	assert.ElementsMatch(t, []string{"pkg.New1", "pkg.New2"}, out["/b.go"])
}

func TestFunctionIndex_EmptyIndex(t *testing.T) {
	cache := openTempCache(t)
	// Saving an empty map should not error.
	require.NoError(t, cache.SaveFunctionIndex(map[string][]string{}))
	assert.Empty(t, cache.LoadFunctionIndex())
}

// ---- ComputeFunctionIndexDelta ----

func TestComputeFunctionIndexDelta_BothEmpty(t *testing.T) {
	added, removed := ComputeFunctionIndexDelta(nil, nil)
	assert.Empty(t, added)
	assert.Empty(t, removed)
}

func TestComputeFunctionIndexDelta_AllAdded(t *testing.T) {
	cached := map[string][]string{}
	current := map[string][]string{"/a.go": {"pkg.New1", "pkg.New2"}}
	added, removed := ComputeFunctionIndexDelta(cached, current)
	assert.Equal(t, map[string]bool{"pkg.New1": true, "pkg.New2": true}, added)
	assert.Empty(t, removed)
}

func TestComputeFunctionIndexDelta_AllRemoved(t *testing.T) {
	cached := map[string][]string{"/a.go": {"pkg.Old1", "pkg.Old2"}}
	current := map[string][]string{}
	added, removed := ComputeFunctionIndexDelta(cached, current)
	assert.Empty(t, added)
	assert.Equal(t, map[string]bool{"pkg.Old1": true, "pkg.Old2": true}, removed)
}

func TestComputeFunctionIndexDelta_Mixed(t *testing.T) {
	cached := map[string][]string{
		"/a.go": {"pkg.Stable", "pkg.Removed"},
	}
	current := map[string][]string{
		"/a.go": {"pkg.Stable", "pkg.Added"},
	}
	added, removed := ComputeFunctionIndexDelta(cached, current)
	assert.Equal(t, map[string]bool{"pkg.Added": true}, added)
	assert.Equal(t, map[string]bool{"pkg.Removed": true}, removed)
}

func TestComputeFunctionIndexDelta_NoChange(t *testing.T) {
	cached := map[string][]string{"/a.go": {"pkg.Fn"}}
	current := map[string][]string{"/a.go": {"pkg.Fn"}}
	added, removed := ComputeFunctionIndexDelta(cached, current)
	assert.Empty(t, added)
	assert.Empty(t, removed)
}

// ---- LoadPass4Results / SavePass4Results ----

func TestPass4Results_EmptyInput(t *testing.T) {
	cache := openTempCache(t)
	out := cache.LoadPass4Results(nil)
	assert.Empty(t, out)
}

func TestPass4Results_RoundTrip(t *testing.T) {
	cache := openTempCache(t)
	dir := t.TempDir()
	goFile := writeTempGoFile(t, dir, "p4.go", "package main\n")

	hash, err := hashFile(goFile)
	require.NoError(t, err)

	results := map[string]*CachedPass4Result{
		goFile: {
			ContentHash: hash,
			Edges: []CachedPass4Edge{
				{
					CallerFQN: "pkg.Caller",
					TargetFQN: "pkg.Target",
					Resolved:  true,
					Target:    "Target",
					File:      goFile,
					Line:      10,
					Arguments: []CachedArgument{
						{Value: "x", IsVariable: true, Position: 0},
					},
					IsStdlib:                 false,
					ResolvedViaTypeInference: true,
					InferredType:             "pkg.MyType",
					TypeConfidence:           0.9,
					TypeSource:               "go_variable_binding",
				},
				{
					CallerFQN:     "pkg.Caller",
					Resolved:      false,
					Target:        "Unknown",
					File:          goFile,
					Line:          20,
					FailureReason: "unresolved_go_call",
				},
			},
			UnresolvedNames: []string{"Unknown"},
		},
	}

	require.NoError(t, cache.SavePass4Results(results))

	out := cache.LoadPass4Results([]string{goFile})
	require.Len(t, out, 1)
	r := out[goFile]
	require.NotNil(t, r)

	assert.Equal(t, hash, r.ContentHash)
	require.Len(t, r.Edges, 2)

	e0 := r.Edges[0]
	assert.Equal(t, "pkg.Caller", e0.CallerFQN)
	assert.Equal(t, "pkg.Target", e0.TargetFQN)
	assert.True(t, e0.Resolved)
	assert.Equal(t, goFile, e0.File)
	assert.Equal(t, 10, e0.Line)
	require.Len(t, e0.Arguments, 1)
	assert.Equal(t, "x", e0.Arguments[0].Value)
	assert.True(t, e0.Arguments[0].IsVariable)
	assert.Equal(t, 0, e0.Arguments[0].Position)
	assert.True(t, e0.ResolvedViaTypeInference)
	assert.Equal(t, "pkg.MyType", e0.InferredType)
	assert.InDelta(t, 0.9, e0.TypeConfidence, 0.001)
	assert.Equal(t, "go_variable_binding", e0.TypeSource)

	e1 := r.Edges[1]
	assert.False(t, e1.Resolved)
	assert.Equal(t, "unresolved_go_call", e1.FailureReason)

	assert.Equal(t, []string{"Unknown"}, r.UnresolvedNames)
}

func TestPass4Results_Miss(t *testing.T) {
	cache := openTempCache(t)
	out := cache.LoadPass4Results([]string{"/no/such/file.go"})
	assert.Empty(t, out)
}

func TestPass4Results_MultipleFiles(t *testing.T) {
	cache := openTempCache(t)
	dir := t.TempDir()
	f1 := writeTempGoFile(t, dir, "a.go", "package a\n")
	f2 := writeTempGoFile(t, dir, "b.go", "package b\n")

	h1, _ := hashFile(f1)
	h2, _ := hashFile(f2)

	results := map[string]*CachedPass4Result{
		f1: {ContentHash: h1, Edges: []CachedPass4Edge{{CallerFQN: "a.F", Target: "G", Resolved: false}}, UnresolvedNames: []string{"G"}},
		f2: {ContentHash: h2, Edges: []CachedPass4Edge{{CallerFQN: "b.H", TargetFQN: "b.I", Target: "I", Resolved: true}}, UnresolvedNames: []string{}},
	}
	require.NoError(t, cache.SavePass4Results(results))

	out := cache.LoadPass4Results([]string{f1, f2})
	assert.Len(t, out, 2)
	assert.NotNil(t, out[f1])
	assert.NotNil(t, out[f2])
	assert.Equal(t, []string{"G"}, out[f1].UnresolvedNames)
	assert.Empty(t, out[f2].UnresolvedNames)
}

func TestPass4Results_EmptyResultsMap(t *testing.T) {
	cache := openTempCache(t)
	// Saving empty map should not error.
	require.NoError(t, cache.SavePass4Results(map[string]*CachedPass4Result{}))
}

// ---- NeedsPass4Rerun ----

func TestNeedsPass4Rerun_NilCached(t *testing.T) {
	assert.True(t, NeedsPass4Rerun(nil, "anyhash", nil, nil))
}

func TestNeedsPass4Rerun_HashMismatch(t *testing.T) {
	cached := &CachedPass4Result{ContentHash: "oldhash"}
	assert.True(t, NeedsPass4Rerun(cached, "newhash", nil, nil))
}

func TestNeedsPass4Rerun_RemovedCallee(t *testing.T) {
	cached := &CachedPass4Result{
		ContentHash: "abc",
		Edges: []CachedPass4Edge{
			{CallerFQN: "pkg.A", TargetFQN: "pkg.B", Resolved: true},
		},
	}
	removed := map[string]bool{"pkg.B": true}
	assert.True(t, NeedsPass4Rerun(cached, "abc", nil, removed))
}

func TestNeedsPass4Rerun_AddedCalleeMatchesUnresolved(t *testing.T) {
	cached := &CachedPass4Result{
		ContentHash:     "abc",
		Edges:           []CachedPass4Edge{{CallerFQN: "pkg.A", Resolved: false, Target: "NewFunc"}},
		UnresolvedNames: []string{"NewFunc"},
	}
	added := map[string]bool{"github.com/foo/bar.NewFunc": true}
	assert.True(t, NeedsPass4Rerun(cached, "abc", added, nil))
}

func TestNeedsPass4Rerun_AddedCalleeNoMatch(t *testing.T) {
	// Added FQN suffix doesn't match any unresolved name → still warm.
	cached := &CachedPass4Result{
		ContentHash:     "abc",
		UnresolvedNames: []string{"OtherFunc"},
	}
	added := map[string]bool{"github.com/foo/bar.DifferentFunc": true}
	assert.False(t, NeedsPass4Rerun(cached, "abc", added, nil))
}

func TestNeedsPass4Rerun_Warm(t *testing.T) {
	// Hash matches, no removed callees, no new callees matching unresolved → warm.
	cached := &CachedPass4Result{
		ContentHash: "abc",
		Edges: []CachedPass4Edge{
			{CallerFQN: "pkg.A", TargetFQN: "pkg.B", Resolved: true},
		},
		UnresolvedNames: []string{"MissingFn"},
	}
	assert.False(t, NeedsPass4Rerun(cached, "abc", nil, nil))
}

func TestNeedsPass4Rerun_UnresolvedEdgeNotTriggeredByRemoved(t *testing.T) {
	// Unresolved edges (Resolved=false) do NOT trigger dirty on removedFQNs check.
	cached := &CachedPass4Result{
		ContentHash: "abc",
		Edges: []CachedPass4Edge{
			{CallerFQN: "pkg.A", Resolved: false, Target: "Ghost"},
		},
		UnresolvedNames: []string{"Ghost"},
	}
	removed := map[string]bool{"pkg.B": true} // removed != unresolved target
	assert.False(t, NeedsPass4Rerun(cached, "abc", nil, removed))
}

func TestNeedsPass4Rerun_EmptyUnresolvedWithAddedFQNs(t *testing.T) {
	// No unresolved names → added FQNs can never match → warm.
	cached := &CachedPass4Result{
		ContentHash:     "abc",
		UnresolvedNames: []string{},
	}
	added := map[string]bool{"github.com/foo.SomeFn": true}
	assert.False(t, NeedsPass4Rerun(cached, "abc", added, nil))
}

// ---- Error-path coverage via corrupt DB state ----

// TestGetFileCached_CorruptCallSitesJSON verifies that a cache entry with
// invalid call_sites_json JSON is treated as a miss.
func TestGetFileCached_CorruptCallSitesJSON(t *testing.T) {
	cache := openTempCache(t)
	dir := t.TempDir()
	goFile := writeTempGoFile(t, dir, "corrupt_cs.go", "package main\n")

	hash, err := hashFile(goFile)
	require.NoError(t, err)

	// Insert row with correct hash but invalid call_sites_json.
	_, err = cache.db.ExecContext(context.Background(),
		`INSERT OR REPLACE INTO file_cache(file_path, content_hash, updated_at, call_sites_json, scope_json)
		 VALUES(?,?,?,?,?)`,
		goFile, hash, 1, "NOT_VALID_JSON", `{"function_scopes":{}}`,
	)
	require.NoError(t, err)

	result, hit := cache.GetFileCached(goFile)
	assert.False(t, hit)
	assert.Nil(t, result)
}

// TestGetFileCached_CorruptScopeJSON verifies that invalid scope_json causes a miss.
func TestGetFileCached_CorruptScopeJSON(t *testing.T) {
	cache := openTempCache(t)
	dir := t.TempDir()
	goFile := writeTempGoFile(t, dir, "corrupt_scope.go", "package main\n")

	hash, err := hashFile(goFile)
	require.NoError(t, err)

	_, err = cache.db.ExecContext(context.Background(),
		`INSERT OR REPLACE INTO file_cache(file_path, content_hash, updated_at, call_sites_json, scope_json)
		 VALUES(?,?,?,?,?)`,
		goFile, hash, 1, `[]`, `NOT_VALID_JSON`,
	)
	require.NoError(t, err)

	result, hit := cache.GetFileCached(goFile)
	assert.False(t, hit)
	assert.Nil(t, result)
}

// TestLoadPass4Results_CorruptEdgesJSON verifies that invalid edges_json causes a miss.
func TestLoadPass4Results_CorruptEdgesJSON(t *testing.T) {
	cache := openTempCache(t)
	dir := t.TempDir()
	goFile := writeTempGoFile(t, dir, "corrupt_edges.go", "package main\n")

	hash, _ := hashFile(goFile)
	_, err := cache.db.ExecContext(context.Background(),
		`INSERT OR REPLACE INTO pass4_results(file_path, content_hash, updated_at, edges_json, unresolved_json)
		 VALUES(?,?,?,?,?)`,
		goFile, hash, 1, "NOT_VALID_JSON", `[]`,
	)
	require.NoError(t, err)

	out := cache.LoadPass4Results([]string{goFile})
	assert.Empty(t, out, "corrupt edges_json should produce a cache miss")
}

// TestLoadPass4Results_CorruptUnresolvedJSON verifies that invalid unresolved_json causes a miss.
func TestLoadPass4Results_CorruptUnresolvedJSON(t *testing.T) {
	cache := openTempCache(t)
	dir := t.TempDir()
	goFile := writeTempGoFile(t, dir, "corrupt_unresolved.go", "package main\n")

	hash, _ := hashFile(goFile)
	_, err := cache.db.ExecContext(context.Background(),
		`INSERT OR REPLACE INTO pass4_results(file_path, content_hash, updated_at, edges_json, unresolved_json)
		 VALUES(?,?,?,?,?)`,
		goFile, hash, 1, `[]`, `NOT_VALID_JSON`,
	)
	require.NoError(t, err)

	out := cache.LoadPass4Results([]string{goFile})
	assert.Empty(t, out, "corrupt unresolved_json should produce a cache miss")
}

// TestSaveFunctionIndex_ClosedDB verifies that SaveFunctionIndex returns an
// error when the database connection is already closed.
func TestSaveFunctionIndex_ClosedDB(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	require.NoError(t, cache.Close()) // close before use

	err = cache.SaveFunctionIndex(map[string][]string{"/a.go": {"pkg.Fn"}})
	assert.Error(t, err)
}

// TestSavePass4Results_ClosedDB verifies that SavePass4Results returns an
// error when the database connection is already closed.
func TestSavePass4Results_ClosedDB(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	require.NoError(t, cache.Close())

	err = cache.SavePass4Results(map[string]*CachedPass4Result{
		"/a.go": {ContentHash: "abc", Edges: nil, UnresolvedNames: nil},
	})
	assert.Error(t, err)
}

// TestLoadFunctionIndex_ClosedDB verifies that LoadFunctionIndex returns an
// empty map (not a panic) when the database connection is already closed.
func TestLoadFunctionIndex_ClosedDB(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	require.NoError(t, cache.Close())

	// Should return empty map, not panic, on closed DB.
	idx := cache.LoadFunctionIndex()
	assert.Empty(t, idx)
}

// TestSaveFunctionIndex_TableDropped verifies that SaveFunctionIndex returns an
// error when the function_index table has been dropped.
func TestSaveFunctionIndex_TableDropped(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	defer cache.Close()

	// Drop the table so the DELETE inside SaveFunctionIndex fails.
	_, err = cache.db.ExecContext(context.Background(), `DROP TABLE function_index`)
	require.NoError(t, err)

	err = cache.SaveFunctionIndex(map[string][]string{"/a.go": {"pkg.Fn"}})
	assert.Error(t, err)
}

// TestSavePass4Results_TableDropped verifies that SavePass4Results returns an
// error when the pass4_results table has been dropped.
func TestSavePass4Results_TableDropped(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	defer cache.Close()

	_, err = cache.db.ExecContext(context.Background(), `DROP TABLE pass4_results`)
	require.NoError(t, err)

	err = cache.SavePass4Results(map[string]*CachedPass4Result{
		"/a.go": {ContentHash: "abc", Edges: []CachedPass4Edge{}, UnresolvedNames: []string{}},
	})
	assert.Error(t, err)
}

// TestOpenAnalysisCache_ProjectRootMismatch_WipesAll verifies that when the
// stored project_root differs from the new one, all data tables are emptied and
// the cache opens successfully with the new project root stored.
func TestOpenAnalysisCache_ProjectRootMismatch_WipesAll(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)

	goFile := writeTempGoFile(t, dir, "wipe_all.go", "package main\n")
	cs := []CachedCallSite{{CallerFQN: "pkg.Fn", FunctionName: "f", CallerFile: goFile, CallLine: 1}}
	sc := &CachedScope{FunctionScopes: make(map[string]CachedFunctionScope)}
	require.NoError(t, cache.PutFileCached(goFile, cs, sc))
	require.NoError(t, cache.SaveFunctionIndex(map[string][]string{goFile: {"pkg.Fn"}}))

	// Store a different project_root to simulate opening the same DB for a new project.
	_, err = cache.db.ExecContext(context.Background(),
		"UPDATE meta SET value='/different/project' WHERE key='project_root'")
	require.NoError(t, err)
	require.NoError(t, cache.Close())

	// Re-opening should succeed and wipe all data tables.
	cache2, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	defer cache2.Close()

	// All data tables should be wiped (file was in old project).
	_, hit := cache2.GetFileCached(goFile)
	assert.False(t, hit, "file_cache should be wiped on project_root change")

	idx := cache2.LoadFunctionIndex()
	assert.Empty(t, idx, "function_index should be wiped on project_root change")
}

// TestOpenAnalysisCache_VersionMismatch_OnlyWipesAffectedTable verifies the
// per-table version check by bumping only pass4_version: pass4_results gets
// wiped while file_cache remains intact.
func TestOpenAnalysisCache_VersionMismatch_OnlyWipesAffectedTable(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)

	goFile := writeTempGoFile(t, dir, "ver_mismatch.go", "package main\n")
	cs := []CachedCallSite{{CallerFQN: "pkg.Fn", FunctionName: "f", CallerFile: goFile, CallLine: 1}}
	sc := &CachedScope{FunctionScopes: make(map[string]CachedFunctionScope)}
	require.NoError(t, cache.PutFileCached(goFile, cs, sc))

	h, _ := hashFile(goFile)
	p4 := map[string]*CachedPass4Result{
		goFile: {ContentHash: h, Edges: []CachedPass4Edge{}, UnresolvedNames: []string{}},
	}
	require.NoError(t, cache.SavePass4Results(p4))

	// Bump pass4_version to simulate a future upgrade.
	_, err = cache.db.ExecContext(context.Background(),
		"INSERT OR REPLACE INTO meta(key,value) VALUES('pass4_version','999')")
	require.NoError(t, err)
	require.NoError(t, cache.Close())

	cache2, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	defer cache2.Close()

	// pass4_results should be wiped.
	out := cache2.LoadPass4Results([]string{goFile})
	assert.Empty(t, out, "pass4_results should be wiped on pass4_version bump")

	// file_cache should still be warm.
	_, hit := cache2.GetFileCached(goFile)
	assert.True(t, hit, "file_cache should survive a pass4_version bump")
}

// ---- Global schema / engine version + rebuild (PR-01) ----

// seedFqnRow inserts one row directly into fqn_index (no writer exists yet).
func seedFqnRow(t *testing.T, cache *AnalysisCache) {
	t.Helper()
	_, err := cache.db.ExecContext(context.Background(),
		`INSERT INTO fqn_index(fqn, language, kind, source, schema_version) VALUES(?,?,?,?,?)`,
		"pkg.Fn", "python", "function", "project", currentSchemaVersion)
	require.NoError(t, err)
}

func countFqnRows(t *testing.T, cache *AnalysisCache) int {
	t.Helper()
	var n int
	require.NoError(t, cache.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM fqn_index`).Scan(&n))
	return n
}

func TestInitSchema_FreshDB_StampsVersions(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCacheWithOptions(CacheOptions{ProjectRoot: dir, EngineVersion: "9.9.9"})
	require.NoError(t, err)
	defer cache.Close()

	for key, want := range map[string]string{
		metaSchemaVersion:        currentSchemaVersion,
		metaEngineVersion:        "9.9.9",
		"fqn_index_version":      fqnIndexVersion,
		"call_sites_version":     callSitesVersion,
		"file_cache_version":     fileCacheVersion,
		"pass4_version":          pass4Version,
		"function_index_version": functionIndexVersion,
	} {
		got, err := getMetaValue(cache.db, key)
		require.NoError(t, err, "meta key %s should be stamped", key)
		assert.Equal(t, want, got, "meta key %s", key)
	}
}

func TestInitSchema_NewTablesQueryable(t *testing.T) {
	cache := openTempCache(t)
	// Both new tables should accept a query without error.
	assert.Zero(t, countFqnRows(t, cache))
	var cs int
	require.NoError(t, cache.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM call_sites`).Scan(&cs))
	assert.Zero(t, cs)
}

func TestInitSchema_SchemaVersionMismatch_FullRebuild(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)

	goFile := writeTempGoFile(t, dir, "schema_bump.go", "package main\n")
	cs := []CachedCallSite{{CallerFQN: "pkg.Fn", FunctionName: "f", CallerFile: goFile, CallLine: 1}}
	require.NoError(t, cache.PutFileCached(goFile, cs, &CachedScope{FunctionScopes: map[string]CachedFunctionScope{}}))
	seedFqnRow(t, cache)

	// Stamp an older schema version to trigger the open-time full rebuild.
	require.NoError(t, setMetaValue(cache.db, metaSchemaVersion, "1"))
	require.NoError(t, cache.Close())

	cache2, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	defer cache2.Close()

	_, hit := cache2.GetFileCached(goFile)
	assert.False(t, hit, "file_cache should be wiped on schema version mismatch")
	assert.Zero(t, countFqnRows(t, cache2), "fqn_index should be wiped on schema version mismatch")

	got, err := getMetaValue(cache2.db, metaSchemaVersion)
	require.NoError(t, err)
	assert.Equal(t, currentSchemaVersion, got, "schema version should be re-stamped after rebuild")
}

func TestInitSchema_EngineVersionMismatch_FullRebuild(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCacheWithOptions(CacheOptions{ProjectRoot: dir, EngineVersion: "2.1.0"})
	require.NoError(t, err)
	seedFqnRow(t, cache)
	require.NoError(t, cache.Close())

	cache2, err := OpenAnalysisCacheWithOptions(CacheOptions{ProjectRoot: dir, EngineVersion: "2.2.0"})
	require.NoError(t, err)
	defer cache2.Close()

	assert.Zero(t, countFqnRows(t, cache2), "fqn_index should be wiped on engine version mismatch")
	got, err := getMetaValue(cache2.db, metaEngineVersion)
	require.NoError(t, err)
	assert.Equal(t, "2.2.0", got)
}

// TestInitSchema_EngineVersionUnknown_DoesNotWipeOrClobber verifies that a
// command that does not know its version (EngineVersion == "") neither wipes the
// cache nor overwrites the stamp a versioned command wrote.
func TestInitSchema_EngineVersionUnknown_DoesNotWipeOrClobber(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCacheWithOptions(CacheOptions{ProjectRoot: dir, EngineVersion: "2.1.0"})
	require.NoError(t, err)
	seedFqnRow(t, cache)
	require.NoError(t, cache.Close())

	// Reopen with no engine version (e.g. an auxiliary command or a test).
	cache2, err := OpenAnalysisCacheWithOptions(CacheOptions{ProjectRoot: dir})
	require.NoError(t, err)
	defer cache2.Close()

	assert.Equal(t, 1, countFqnRows(t, cache2), "unknown engine version must not wipe the cache")
	got, err := getMetaValue(cache2.db, metaEngineVersion)
	require.NoError(t, err)
	assert.Equal(t, "2.1.0", got, "unknown engine version must not clobber the existing stamp")
}

// TestInitSchema_LegacyDBNoSchemaStamp_Survives simulates a database written by
// a binary that predates the schema_version stamp: it must be upgraded in place
// (new tables added, stamp written) without wiping the warm data.
func TestInitSchema_LegacyDBNoSchemaStamp_Survives(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)

	goFile := writeTempGoFile(t, dir, "legacy.go", "package main\n")
	cs := []CachedCallSite{{CallerFQN: "pkg.Fn", FunctionName: "f", CallerFile: goFile, CallLine: 1}}
	require.NoError(t, cache.PutFileCached(goFile, cs, &CachedScope{FunctionScopes: map[string]CachedFunctionScope{}}))

	// Remove the schema_version stamp to look like a pre-stamp database.
	_, err = cache.db.ExecContext(context.Background(), `DELETE FROM meta WHERE key=?`, metaSchemaVersion)
	require.NoError(t, err)
	require.NoError(t, cache.Close())

	cache2, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	defer cache2.Close()

	_, hit := cache2.GetFileCached(goFile)
	assert.True(t, hit, "a legacy DB without a schema stamp must not be wiped on upgrade")
	got, err := getMetaValue(cache2.db, metaSchemaVersion)
	require.NoError(t, err)
	assert.Equal(t, currentSchemaVersion, got, "schema version should be stamped on upgrade")
}

func TestInitSchema_ForceRebuild_WipesAll(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)

	goFile := writeTempGoFile(t, dir, "force.go", "package main\n")
	cs := []CachedCallSite{{CallerFQN: "pkg.Fn", FunctionName: "f", CallerFile: goFile, CallLine: 1}}
	require.NoError(t, cache.PutFileCached(goFile, cs, &CachedScope{FunctionScopes: map[string]CachedFunctionScope{}}))
	seedFqnRow(t, cache)
	require.NoError(t, cache.Close())

	cache2, err := OpenAnalysisCacheWithOptions(CacheOptions{ProjectRoot: dir, ForceRebuild: true})
	require.NoError(t, err)
	defer cache2.Close()

	_, hit := cache2.GetFileCached(goFile)
	assert.False(t, hit, "--rebuild-index should wipe file_cache")
	assert.Zero(t, countFqnRows(t, cache2), "--rebuild-index should wipe fqn_index")
}

func TestInitSchema_FqnIndexVersionBump_WipesOnlyFqnIndex(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)

	goFile := writeTempGoFile(t, dir, "fqn_bump.go", "package main\n")
	cs := []CachedCallSite{{CallerFQN: "pkg.Fn", FunctionName: "f", CallerFile: goFile, CallLine: 1}}
	require.NoError(t, cache.PutFileCached(goFile, cs, &CachedScope{FunctionScopes: map[string]CachedFunctionScope{}}))
	seedFqnRow(t, cache)

	// Bump only the fqn_index per-table version.
	require.NoError(t, setMetaValue(cache.db, "fqn_index_version", "999"))
	require.NoError(t, cache.Close())

	cache2, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	defer cache2.Close()

	assert.Zero(t, countFqnRows(t, cache2), "fqn_index should be wiped on its own version bump")
	_, hit := cache2.GetFileCached(goFile)
	assert.True(t, hit, "file_cache should survive an fqn_index_version bump")
}
