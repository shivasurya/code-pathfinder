package builder

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// FqnEntry is one row of the fqn_index table: a single definition the engine
// discovered, with its location and classification. Nullable columns use
// sql.Null* so "absent" (a synthetic entry with no file, a variable with no
// signature) is distinct from an empty string or a zero line.
type FqnEntry struct {
	Fqn       string
	Language  string
	Kind      string // function | class | method | module | variable | import_alias
	Source    string // project | stdlib | thirdparty
	File      sql.NullString
	StartLine sql.NullInt64
	StartCol  sql.NullInt64
	EndLine   sql.NullInt64
	EndCol    sql.NullInt64
	ParentFqn sql.NullString
	Signature sql.NullString
}

// nullStr returns a valid sql.NullString only for a non-empty value.
func nullStr(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

// nullInt returns a valid sql.NullInt64 only for a positive value. Lines and
// columns are 1-indexed in the engine, so 0 means "unknown".
func nullInt(v int) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(v), Valid: v > 0}
}

// ReplacePythonFqnIndex atomically refreshes the Python slice of the index:
// it deletes every existing python row in fqn_index, inserts the supplied
// entries, and stamps each file's mtime in indexed_files, all in one
// transaction. A full BuildCallGraph run rediscovers every Python definition,
// so replacing the whole python slice keeps the index free of rows for deleted
// or renamed symbols without a per-file diff.
//
// Only python rows are touched; Go's function_index/pass4 caches are unaffected.
func (c *AnalysisCache) ReplacePythonFqnIndex(entries []FqnEntry, files []IndexedFile) error {
	ctx := context.Background()
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("analysis cache: begin fqn tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM fqn_index WHERE language='python'`); err != nil {
		return fmt.Errorf("analysis cache: clear python fqn_index: %w", err)
	}

	insert, err := tx.PrepareContext(ctx, `INSERT INTO fqn_index
		(fqn, language, kind, source, file, start_line, start_col, end_line, end_col, parent_fqn, signature, schema_version)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return fmt.Errorf("analysis cache: prepare fqn insert: %w", err)
	}
	defer insert.Close()

	for _, e := range entries {
		if _, err := insert.ExecContext(ctx,
			e.Fqn, e.Language, e.Kind, e.Source, e.File,
			e.StartLine, e.StartCol, e.EndLine, e.EndCol,
			e.ParentFqn, e.Signature, currentSchemaVersion,
		); err != nil {
			return fmt.Errorf("analysis cache: insert fqn %q: %w", e.Fqn, err)
		}
	}

	mark, err := tx.PrepareContext(ctx,
		`INSERT OR REPLACE INTO indexed_files(file_path, indexed_at_mtime, indexed_at) VALUES(?,?,?)`)
	if err != nil {
		return fmt.Errorf("analysis cache: prepare indexed_files insert: %w", err)
	}
	defer mark.Close()

	now := time.Now().Unix()
	for _, f := range files {
		if _, err := mark.ExecContext(ctx, f.Path, f.ModTimeUnix, now); err != nil {
			return fmt.Errorf("analysis cache: mark indexed %s: %w", f.Path, err)
		}
	}

	return tx.Commit()
}

// LookupFqn returns the indexed definition whose recorded start location is the
// given file and line. The bool is false when no definition starts there. It is
// the reader PR-04's `pathfinder fqn` CLI consumes; col is accepted for forward
// compatibility but definition rows currently index on (file, start_line).
func (c *AnalysisCache) LookupFqn(file string, line, _ int) (FqnEntry, bool, error) {
	var e FqnEntry
	err := c.db.QueryRowContext(context.Background(),
		`SELECT fqn, language, kind, source, file, start_line, start_col, end_line, end_col, parent_fqn, signature
		 FROM fqn_index WHERE file=? AND start_line=? LIMIT 1`, file, line,
	).Scan(&e.Fqn, &e.Language, &e.Kind, &e.Source, &e.File,
		&e.StartLine, &e.StartCol, &e.EndLine, &e.EndCol, &e.ParentFqn, &e.Signature)
	if err == sql.ErrNoRows {
		return FqnEntry{}, false, nil
	}
	if err != nil {
		return FqnEntry{}, false, fmt.Errorf("analysis cache: lookup fqn at %s:%d: %w", file, line, err)
	}
	return e, true, nil
}
