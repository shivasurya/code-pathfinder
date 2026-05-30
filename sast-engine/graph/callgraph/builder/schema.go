package builder

import (
	"context"
	"database/sql"
	"fmt"
)

// Table DDL for the persisted analysis index.
//
// All statements use CREATE TABLE/INDEX IF NOT EXISTS so applySchema is
// idempotent: running it against an up-to-date database is a no-op, and adding
// a brand-new table to an existing database needs no version bump (the new
// table is simply created empty for downstream PRs to populate).
//
// The first three tables (meta, file_cache, function_index, pass4_results)
// predate this index generalisation and back the Go callgraph cache. The
// fqn_index and call_sites tables are the language-agnostic FQN surface added
// here; PR-02 populates fqn_index, PR-03 populates call_sites.
const (
	ddlMeta = `CREATE TABLE IF NOT EXISTS meta (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`

	ddlFileCache = `CREATE TABLE IF NOT EXISTS file_cache (
		file_path        TEXT    PRIMARY KEY,
		content_hash     TEXT    NOT NULL,
		updated_at       INTEGER NOT NULL,
		call_sites_json  TEXT    NOT NULL,
		scope_json       TEXT    NOT NULL
	)`

	ddlFunctionIndex = `CREATE TABLE IF NOT EXISTS function_index (
		file_path TEXT NOT NULL,
		fqn       TEXT NOT NULL,
		PRIMARY KEY (file_path, fqn)
	)`

	ddlPass4Results = `CREATE TABLE IF NOT EXISTS pass4_results (
		file_path       TEXT    PRIMARY KEY,
		content_hash    TEXT    NOT NULL,
		updated_at      INTEGER NOT NULL,
		edges_json      TEXT    NOT NULL,
		unresolved_json TEXT    NOT NULL
	)`

	// fqn_index: every FQN the engine knows about, with its definition site if
	// any. file/start_*/end_* are NULL for synthetic stdlib entries.
	ddlFqnIndex = `CREATE TABLE IF NOT EXISTS fqn_index (
		fqn             TEXT    NOT NULL,
		language        TEXT    NOT NULL,
		kind            TEXT    NOT NULL,
		source          TEXT    NOT NULL,
		file            TEXT,
		start_line      INTEGER,
		start_col       INTEGER,
		end_line        INTEGER,
		end_col         INTEGER,
		parent_fqn      TEXT,
		signature       TEXT,
		schema_version  INTEGER NOT NULL
	)`

	// call_sites: every call site, resolved or not, with enough breadcrumbs to
	// explain failures. diagnostics holds a JSON array of frames, populated only
	// when resolved=0.
	ddlCallSites = `CREATE TABLE IF NOT EXISTS call_sites (
		file            TEXT    NOT NULL,
		line            INTEGER NOT NULL,
		col             INTEGER NOT NULL,
		function_fqn    TEXT,
		target_name     TEXT    NOT NULL,
		target_fqn      TEXT,
		resolved        INTEGER NOT NULL,
		diagnostics     TEXT,
		schema_version  INTEGER NOT NULL
	)`

	// indexed_files: per-file mtime stamp recorded when a file's definitions were
	// last written into fqn_index. A later query compares the file's current
	// mtime against indexed_at_mtime to decide whether the file is stale and its
	// rows must be refreshed before the query answers.
	ddlIndexedFiles = `CREATE TABLE IF NOT EXISTS indexed_files (
		file_path          TEXT    PRIMARY KEY,
		indexed_at_mtime   INTEGER NOT NULL,
		indexed_at         INTEGER NOT NULL
	)`
)

// schemaStmts is the ordered list of DDL applied at open time. Tables first,
// then their indices.
var schemaStmts = []string{
	ddlMeta,
	ddlFileCache,
	ddlFunctionIndex,
	ddlPass4Results,
	ddlFqnIndex,
	ddlCallSites,
	ddlIndexedFiles,
	`CREATE INDEX IF NOT EXISTS idx_fqn_exact    ON fqn_index(fqn)`,
	`CREATE INDEX IF NOT EXISTS idx_fqn_prefix   ON fqn_index(fqn COLLATE NOCASE)`,
	`CREATE INDEX IF NOT EXISTS idx_location     ON fqn_index(file, start_line)`,
	`CREATE INDEX IF NOT EXISTS idx_parent_fqn   ON fqn_index(parent_fqn)`,
	`CREATE INDEX IF NOT EXISTS idx_callsite_loc ON call_sites(file, line, col)`,
	`CREATE INDEX IF NOT EXISTS idx_callsite_fqn ON call_sites(target_fqn)`,
}

// dataTables are every table holding cached analysis output (everything except
// meta). A full rebuild (schema/engine version mismatch, or --rebuild-index)
// empties these while leaving meta to carry the new version stamps.
var dataTables = []string{
	"file_cache",
	"function_index",
	"pass4_results",
	"fqn_index",
	"call_sites",
	"indexed_files",
}

// applySchema creates every table and index if absent. It is wrapped in a
// single transaction so a partial DDL failure rolls back and never leaves a
// half-applied schema on disk.
func applySchema(db *sql.DB) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("analysis cache: begin schema tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, stmt := range schemaStmts {
		if _, err := tx.ExecContext(context.Background(), stmt); err != nil {
			return fmt.Errorf("analysis cache: apply schema: %w", err)
		}
	}
	return tx.Commit()
}

// wipeDataTables empties every data table in a single transaction, leaving meta
// intact. Used by the full-rebuild paths.
func wipeDataTables(db *sql.DB) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("analysis cache: begin wipe tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, tbl := range dataTables {
		if _, err := tx.ExecContext(context.Background(), `DELETE FROM `+tbl); err != nil {
			return fmt.Errorf("analysis cache: wipe %s: %w", tbl, err)
		}
	}
	return tx.Commit()
}
