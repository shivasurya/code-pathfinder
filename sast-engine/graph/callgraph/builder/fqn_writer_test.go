package builder

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pyEntry(fqn, kind, file string, line int) FqnEntry {
	return FqnEntry{
		Fqn:       fqn,
		Language:  "python",
		Kind:      kind,
		Source:    "project",
		File:      nullStr(file),
		StartLine: nullInt(line),
		ParentFqn: nullStr(parentFqn(fqn)),
	}
}

func TestReplacePythonFqnIndex_WriteAndLookup(t *testing.T) {
	cache := openTempCache(t)
	entries := []FqnEntry{
		pyEntry("app.models.sanitize", "function", "/proj/app/models.py", 10),
		pyEntry("app.models.User", "class", "/proj/app/models.py", 15),
	}
	require.NoError(t, cache.ReplacePythonFqnIndex(entries, nil))

	got, ok, err := cache.LookupFqn("/proj/app/models.py", 10, 0)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "app.models.sanitize", got.Fqn)
	assert.Equal(t, "function", got.Kind)
	assert.Equal(t, "project", got.Source)
	assert.Equal(t, "app.models", got.ParentFqn.String)
}

func TestLookupFqn_Miss(t *testing.T) {
	cache := openTempCache(t)
	_, ok, err := cache.LookupFqn("/nope.py", 1, 0)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestReplacePythonFqnIndex_ReplacesPreviousPythonRows(t *testing.T) {
	cache := openTempCache(t)
	require.NoError(t, cache.ReplacePythonFqnIndex([]FqnEntry{
		pyEntry("app.old.gone", "function", "/proj/old.py", 1),
	}, nil))
	require.NoError(t, cache.ReplacePythonFqnIndex([]FqnEntry{
		pyEntry("app.new.fresh", "function", "/proj/new.py", 1),
	}, nil))

	_, ok, err := cache.LookupFqn("/proj/old.py", 1, 0)
	require.NoError(t, err)
	assert.False(t, ok, "the previous python slice must be gone after a replace")

	_, ok, err = cache.LookupFqn("/proj/new.py", 1, 0)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestReplacePythonFqnIndex_LeavesNonPythonRowsIntact(t *testing.T) {
	cache := openTempCache(t)
	// Seed a non-python row directly.
	_, err := cache.db.ExecContext(context.Background(),
		`INSERT INTO fqn_index(fqn, language, kind, source, file, start_line, schema_version)
		 VALUES('pkg.GoFn','go','function','project','/proj/main.go',3,?)`, currentSchemaVersion)
	require.NoError(t, err)

	require.NoError(t, cache.ReplacePythonFqnIndex([]FqnEntry{
		pyEntry("app.x.fn", "function", "/proj/x.py", 1),
	}, nil))

	var goRows int
	require.NoError(t, cache.db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM fqn_index WHERE language='go'`).Scan(&goRows))
	assert.Equal(t, 1, goRows, "a python replace must not touch go rows")
}

func TestReplacePythonFqnIndex_NullableColumns(t *testing.T) {
	cache := openTempCache(t)
	// An entry with no file and no signature should store NULLs, not empty strings.
	e := FqnEntry{Fqn: "synthetic.thing", Language: "python", Kind: "function", Source: "project"}
	require.NoError(t, cache.ReplacePythonFqnIndex([]FqnEntry{e}, nil))

	var file, signature sql.NullString
	err := cache.db.QueryRowContext(context.Background(),
		`SELECT file, signature FROM fqn_index WHERE fqn='synthetic.thing'`).Scan(&file, &signature)
	require.NoError(t, err)
	assert.False(t, file.Valid, "empty file must be NULL")
	assert.False(t, signature.Valid, "empty signature must be NULL")
}

func TestReplacePythonFqnIndex_StampsIndexedFiles(t *testing.T) {
	cache := openTempCache(t)
	files := []IndexedFile{{Path: "/proj/app/models.py", ModTimeUnix: 1700000000}}
	require.NoError(t, cache.ReplacePythonFqnIndex(nil, files))

	mtime, ok, err := cache.GetIndexedFileMtime("/proj/app/models.py")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, int64(1700000000), mtime)
}

func TestReplacePythonFqnIndex_ClosedDBErrors(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	require.NoError(t, cache.Close())

	err = cache.ReplacePythonFqnIndex([]FqnEntry{pyEntry("a.b", "function", "/a.py", 1)}, nil)
	assert.Error(t, err)
}

func TestLookupFqn_ClosedDBErrors(t *testing.T) {
	dir := t.TempDir()
	cache, err := OpenAnalysisCache(dir)
	require.NoError(t, err)
	require.NoError(t, cache.Close())

	_, _, err = cache.LookupFqn("/a.py", 1, 0)
	assert.Error(t, err)
}

func TestReplacePythonFqnIndex_DeleteErrorWhenTableDropped(t *testing.T) {
	cache := openTempCache(t)
	_, err := cache.db.ExecContext(context.Background(), `DROP TABLE fqn_index`)
	require.NoError(t, err)

	err = cache.ReplacePythonFqnIndex([]FqnEntry{pyEntry("a.b", "function", "/a.py", 1)}, nil)
	assert.Error(t, err, "DELETE on a dropped fqn_index must surface")
}

func TestReplacePythonFqnIndex_MarkErrorWhenIndexedFilesDropped(t *testing.T) {
	cache := openTempCache(t)
	_, err := cache.db.ExecContext(context.Background(), `DROP TABLE indexed_files`)
	require.NoError(t, err)

	err = cache.ReplacePythonFqnIndex(
		[]FqnEntry{pyEntry("a.b", "function", "/a.py", 1)},
		[]IndexedFile{{Path: "/a.py", ModTimeUnix: 1}},
	)
	assert.Error(t, err, "stamping indexed_files when the table is gone must surface")
}

// fqnIndexCols is the column list ReplacePythonFqnIndex inserts into. Tests that
// recreate the table to inject failures keep it in sync.
const fqnIndexCols = `fqn, language, kind, source, file, start_line, start_col, end_line, end_col, parent_fqn, signature, schema_version`

func TestReplacePythonFqnIndex_PrepareErrorOnColumnMismatch(t *testing.T) {
	cache := openTempCache(t)
	// Replace fqn_index with a table that has the wrong shape so the 12-column
	// INSERT cannot be prepared (the DELETE still succeeds first).
	_, err := cache.db.ExecContext(context.Background(), `DROP TABLE fqn_index`)
	require.NoError(t, err)
	_, err = cache.db.ExecContext(context.Background(), `CREATE TABLE fqn_index(fqn TEXT)`)
	require.NoError(t, err)

	err = cache.ReplacePythonFqnIndex([]FqnEntry{pyEntry("a.b", "function", "/a.py", 1)}, nil)
	assert.Error(t, err, "a shape mismatch must surface as a prepare error")
}

func TestReplacePythonFqnIndex_InsertErrorRollsBack(t *testing.T) {
	cache := openTempCache(t)
	// Seed a good row, then rebuild fqn_index with a CHECK that the next write
	// violates. The whole call must fail and roll back, leaving the seed intact.
	require.NoError(t, cache.ReplacePythonFqnIndex([]FqnEntry{pyEntry("keep.me", "function", "/keep.py", 1)}, nil))
	_, err := cache.db.ExecContext(context.Background(), `DROP TABLE fqn_index`)
	require.NoError(t, err)
	_, err = cache.db.ExecContext(context.Background(),
		`CREATE TABLE fqn_index(`+fqnIndexCols+`, CHECK(kind <> 'class'))`)
	require.NoError(t, err)
	require.NoError(t, cache.ReplacePythonFqnIndex([]FqnEntry{pyEntry("keep.me", "function", "/keep.py", 1)}, nil))

	err = cache.ReplacePythonFqnIndex([]FqnEntry{pyEntry("bad.cls", "class", "/bad.py", 1)}, nil)
	require.Error(t, err, "a CHECK violation mid-insert must surface")

	// Rollback: the prior good row survives, the bad row is absent.
	_, ok, err := cache.LookupFqn("/keep.py", 1, 0)
	require.NoError(t, err)
	assert.True(t, ok, "the transaction must roll back, preserving the previous slice")
}

func TestReplacePythonFqnIndex_MarkExecErrorRollsBack(t *testing.T) {
	cache := openTempCache(t)
	// Rebuild indexed_files with a CHECK the stamp violates, so the insert phase
	// succeeds but the mark phase fails.
	_, err := cache.db.ExecContext(context.Background(), `DROP TABLE indexed_files`)
	require.NoError(t, err)
	_, err = cache.db.ExecContext(context.Background(),
		`CREATE TABLE indexed_files(file_path TEXT PRIMARY KEY, indexed_at_mtime INTEGER NOT NULL, indexed_at INTEGER NOT NULL, CHECK(file_path <> '/bad.py'))`)
	require.NoError(t, err)

	err = cache.ReplacePythonFqnIndex(
		[]FqnEntry{pyEntry("a.b", "function", "/a.py", 1)},
		[]IndexedFile{{Path: "/bad.py", ModTimeUnix: 1}},
	)
	require.Error(t, err, "a CHECK violation while stamping must surface")

	// Rollback: the entry written before the failing stamp must not persist.
	_, ok, err := cache.LookupFqn("/a.py", 1, 0)
	require.NoError(t, err)
	assert.False(t, ok, "the whole transaction must roll back")
}
