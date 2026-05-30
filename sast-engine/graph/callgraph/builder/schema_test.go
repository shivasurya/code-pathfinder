package builder

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// openRawDB opens a file-backed SQLite database with no schema applied, for
// exercising schema.go in isolation.
func openRawDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "raw.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var got string
	err := db.QueryRowContext(context.Background(),
		`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, name,
	).Scan(&got)
	if err == sql.ErrNoRows {
		return false
	}
	require.NoError(t, err)
	return got == name
}

func indexExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var got string
	err := db.QueryRowContext(context.Background(),
		`SELECT name FROM sqlite_master WHERE type='index' AND name=?`, name,
	).Scan(&got)
	if err == sql.ErrNoRows {
		return false
	}
	require.NoError(t, err)
	return got == name
}

func TestApplySchema_CreatesAllTablesAndIndices(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))

	for _, tbl := range []string{"meta", "file_cache", "function_index", "pass4_results", "fqn_index", "call_sites"} {
		assert.True(t, tableExists(t, db, tbl), "table %s should exist", tbl)
	}
	for _, idx := range []string{
		"idx_fqn_exact", "idx_fqn_prefix", "idx_location", "idx_parent_fqn",
		"idx_callsite_loc", "idx_callsite_fqn",
	} {
		assert.True(t, indexExists(t, db, idx), "index %s should exist", idx)
	}
}

func TestApplySchema_Idempotent(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))
	// A second apply against an up-to-date schema is a no-op, not an error.
	require.NoError(t, applySchema(db))
	assert.True(t, tableExists(t, db, "fqn_index"))
}

func TestWipeDataTables_EmptiesDataKeepsMeta(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))

	// Seed one row in a data table and one meta row.
	_, err := db.ExecContext(context.Background(),
		`INSERT INTO fqn_index(fqn, language, kind, source, schema_version) VALUES('pkg.Fn','python','function','project',2)`)
	require.NoError(t, err)
	require.NoError(t, setMetaValue(db, "sentinel", "keep"))

	require.NoError(t, wipeDataTables(db))

	var fqnCount int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM fqn_index`).Scan(&fqnCount))
	assert.Zero(t, fqnCount, "fqn_index should be emptied")

	got, err := getMetaValue(db, "sentinel")
	require.NoError(t, err)
	assert.Equal(t, "keep", got, "meta must survive a data wipe")
}

func TestApplySchema_RollsBackOnError(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, db.Close()) // a closed DB makes BeginTx fail
	assert.Error(t, applySchema(db))
}

func TestWipeDataTables_ErrorsOnClosedDB(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))
	require.NoError(t, db.Close())
	assert.Error(t, wipeDataTables(db))
}

// TestWipeDataTables_ErrorsOnMissingTable covers the in-loop DELETE failure:
// the DB is open but one data table has been dropped.
func TestWipeDataTables_ErrorsOnMissingTable(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))
	_, err := db.ExecContext(context.Background(), `DROP TABLE fqn_index`)
	require.NoError(t, err)

	assert.Error(t, wipeDataTables(db))
}

// TestApplySchema_ErrorsOnNameCollision covers the in-loop DDL failure: an
// object already exists under a name applySchema needs for an index, so the
// CREATE INDEX cannot be satisfied even with IF NOT EXISTS.
func TestApplySchema_ErrorsOnNameCollision(t *testing.T) {
	db := openRawDB(t)
	_, err := db.ExecContext(context.Background(), `CREATE TABLE idx_fqn_exact (x INTEGER)`)
	require.NoError(t, err)

	assert.Error(t, applySchema(db))
}
