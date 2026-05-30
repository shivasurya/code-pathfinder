package builder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Reserved meta keys. The meta table is a flat key/value store; these are the
// keys reconcileVersions reads and writes at open time.
const (
	metaProjectRoot   = "project_root"
	metaSchemaVersion = "schema_version"
	metaEngineVersion = "engine_version"
)

// errMetaKeyMissing is returned by getMetaValue when a key is absent. Callers
// that treat "absent" as "fresh database" check for it with errors.Is.
var errMetaKeyMissing = errors.New("analysis cache: meta key not found")

// getMetaValue reads a single meta key. It returns errMetaKeyMissing when the
// key is absent (distinct from a present-but-empty value or a SQL error) so the
// open-time check can tell a brand-new database apart from one stamped by an
// older binary.
func getMetaValue(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRowContext(context.Background(),
		`SELECT value FROM meta WHERE key=?`, key,
	).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", errMetaKeyMissing
	}
	if err != nil {
		return "", fmt.Errorf("analysis cache: read meta %q: %w", key, err)
	}
	return value, nil
}

// setMetaValue upserts a single meta key.
func setMetaValue(db *sql.DB, key, value string) error {
	if _, err := db.ExecContext(context.Background(),
		`INSERT OR REPLACE INTO meta(key,value) VALUES(?,?)`, key, value,
	); err != nil {
		return fmt.Errorf("analysis cache: write meta %q: %w", key, err)
	}
	return nil
}
