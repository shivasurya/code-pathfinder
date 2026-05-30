package builder

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetIndexedFileMtime_NeverIndexed(t *testing.T) {
	cache := openTempCache(t)
	_, ok, err := cache.GetIndexedFileMtime("/never/seen.py")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestIsStale_NeverIndexed(t *testing.T) {
	cache := openTempCache(t)
	dir := t.TempDir()
	f := filepath.Join(dir, "fresh.py")
	require.NoError(t, os.WriteFile(f, []byte("x = 1\n"), 0o644))

	stale, err := cache.IsStale(f)
	require.NoError(t, err)
	assert.True(t, stale, "a file never indexed is stale")
}

func TestIsStale_FreshThenTouched(t *testing.T) {
	cache := openTempCache(t)
	dir := t.TempDir()
	f := filepath.Join(dir, "m.py")
	require.NoError(t, os.WriteFile(f, []byte("x = 1\n"), 0o644))

	info, err := os.Stat(f)
	require.NoError(t, err)
	require.NoError(t, cache.ReplacePythonFqnIndex(nil, []IndexedFile{{Path: f, ModTimeUnix: info.ModTime().Unix()}}))

	stale, err := cache.IsStale(f)
	require.NoError(t, err)
	assert.False(t, stale, "a file indexed at its current mtime is fresh")

	// Advance mtime past the stamp.
	future := time.Now().Add(2 * time.Second)
	require.NoError(t, os.Chtimes(f, future, future))

	stale, err = cache.IsStale(f)
	require.NoError(t, err)
	assert.True(t, stale, "a file touched after indexing is stale")
}

func TestIsStale_DeletedFile(t *testing.T) {
	cache := openTempCache(t)
	stale, err := cache.IsStale("/proj/deleted.py")
	assert.True(t, stale, "a missing file is reported stale")
	assert.Error(t, err, "stat failure is surfaced so the caller can drop the rows")
}

func TestGetIndexedFileMtime_ClosedDBErrors(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	require.NoError(t, cache.Close())

	_, _, err = cache.GetIndexedFileMtime("/a.py")
	assert.Error(t, err)
}

func TestIsStale_DBErrorPropagates(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)

	f := filepath.Join(dir, "live.py")
	require.NoError(t, os.WriteFile(f, []byte("x = 1\n"), 0o644))
	require.NoError(t, cache.Close()) // close so the mtime lookup errors, but stat still succeeds

	stale, err := cache.IsStale(f)
	assert.True(t, stale)
	assert.Error(t, err, "a DB error during staleness check must surface, defaulting to stale")
}
