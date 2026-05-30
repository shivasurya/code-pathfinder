package builder

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetaValue_RoundTrip(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))

	require.NoError(t, setMetaValue(db, metaSchemaVersion, "2"))
	got, err := getMetaValue(db, metaSchemaVersion)
	require.NoError(t, err)
	assert.Equal(t, "2", got)
}

func TestMetaValue_Upsert(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))

	require.NoError(t, setMetaValue(db, metaEngineVersion, "2.1.0"))
	require.NoError(t, setMetaValue(db, metaEngineVersion, "2.2.0"))
	got, err := getMetaValue(db, metaEngineVersion)
	require.NoError(t, err)
	assert.Equal(t, "2.2.0", got, "second write should overwrite the first")
}

func TestMetaValue_MissingKey(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))

	_, err := getMetaValue(db, "never-written")
	assert.True(t, errors.Is(err, errMetaKeyMissing), "absent key should return errMetaKeyMissing, got %v", err)
}

func TestMetaValue_EmptyStringIsNotMissing(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))

	require.NoError(t, setMetaValue(db, "present-but-empty", ""))
	got, err := getMetaValue(db, "present-but-empty")
	require.NoError(t, err, "a present empty value is not a missing key")
	assert.Equal(t, "", got)
}

func TestGetMetaValue_SQLErrorPropagates(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))
	require.NoError(t, db.Close()) // force a real SQL error, distinct from "missing"

	_, err := getMetaValue(db, metaSchemaVersion)
	require.Error(t, err)
	assert.False(t, errors.Is(err, errMetaKeyMissing), "a closed-DB error must not masquerade as a missing key")
}

func TestReadOptionalMeta_MissingReturnsEmpty(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))

	got, err := readOptionalMeta(db, "absent")
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

func TestReadOptionalMeta_SQLErrorPropagates(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))
	require.NoError(t, db.Close())

	_, err := readOptionalMeta(db, metaSchemaVersion)
	assert.Error(t, err, "a real SQL error must not be swallowed as empty")
}

func TestSetMetaValue_ClosedDBErrors(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))
	require.NoError(t, db.Close())

	assert.Error(t, setMetaValue(db, metaSchemaVersion, "2"))
}

// ---- init helpers error injection (closed DB) ----

func TestNeedsFullRebuild_SQLErrorPropagates(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))
	require.NoError(t, db.Close())

	_, _, err := needsFullRebuild(db, CacheOptions{ProjectRoot: "/p"})
	assert.Error(t, err)
}

func TestWipeStaleTables_SQLErrorPropagates(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))
	require.NoError(t, db.Close())

	assert.Error(t, wipeStaleTables(db))
}

func TestStampMeta_SQLErrorPropagates(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))
	require.NoError(t, db.Close())

	assert.Error(t, stampMeta(db, CacheOptions{ProjectRoot: "/p", EngineVersion: "1.0.0"}))
}

func TestReadRebuildStamps_ClosedDBErrors(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))
	require.NoError(t, db.Close())

	_, err := readRebuildStamps(db)
	assert.Error(t, err)
}

func TestReconcileVersions_NeedsRebuildErrorOnClosedDB(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, db.Close())

	assert.Error(t, reconcileVersions(db, CacheOptions{ProjectRoot: "/p"}))
}

func TestReconcileVersions_WipeDataErrorOnForceRebuild(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))
	// Drop a data table, then force a full rebuild so wipeDataTables hits it.
	_, err := db.ExecContext(context.Background(), `DROP TABLE fqn_index`)
	require.NoError(t, err)

	assert.Error(t, reconcileVersions(db, CacheOptions{ProjectRoot: "/p", ForceRebuild: true}))
}

func TestReconcileVersions_WipeStaleErrorOnDroppedTable(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))
	// No version stamps yet, so the per-table check will try to DELETE every
	// table; dropping one makes that DELETE fail (no full rebuild triggered).
	_, err := db.ExecContext(context.Background(), `DROP TABLE file_cache`)
	require.NoError(t, err)

	assert.Error(t, reconcileVersions(db, CacheOptions{ProjectRoot: "/p"}))
}

func TestWipeStaleTables_DeleteErrorOnDroppedTable(t *testing.T) {
	db := openRawDB(t)
	require.NoError(t, applySchema(db))
	// file_cache_version is unstamped (""), so wipeStaleTables will try to DELETE
	// file_cache; dropping the table makes that DELETE fail.
	_, err := db.ExecContext(context.Background(), `DROP TABLE file_cache`)
	require.NoError(t, err)

	assert.Error(t, wipeStaleTables(db))
}

// TestOpenAnalysisCacheWithOptions_WALErrorOnDirectory points the index path at
// an existing directory: sql.Open is lazy, so the failure surfaces at the WAL
// pragma rather than at open, exercising that error path.
func TestOpenAnalysisCacheWithOptions_WALErrorOnDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := OpenAnalysisCacheWithOptions(CacheOptions{ProjectRoot: dir, IndexPath: dir})
	assert.Error(t, err)
}

// TestOpenAnalysisCacheWithOptions_ApplySchemaError seeds a valid database with
// an object whose name collides with an index applySchema creates, so the open
// path fails at schema application (after a clean WAL pragma).
func TestOpenAnalysisCacheWithOptions_ApplySchemaError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "seeded.sqlite")
	seed, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	_, err = seed.ExecContext(context.Background(), `CREATE TABLE idx_fqn_exact (x INTEGER)`)
	require.NoError(t, err)
	require.NoError(t, seed.Close())

	_, err = OpenAnalysisCacheWithOptions(CacheOptions{ProjectRoot: t.TempDir(), IndexPath: path})
	assert.Error(t, err, "a name collision must surface during schema application")
}
