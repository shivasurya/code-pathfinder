package builder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
)

// IndexedFile records that a file's definitions were written into fqn_index at a
// known modification time. ModTimeUnix is the file's mtime at index time, used
// to detect staleness on a later query.
type IndexedFile struct {
	Path        string
	ModTimeUnix int64
}

// GetIndexedFileMtime returns the stored index-time mtime for a file. The bool
// is false when the file has never been indexed (no row), which is distinct
// from a SQL error.
func (c *AnalysisCache) GetIndexedFileMtime(path string) (int64, bool, error) {
	var mtime int64
	err := c.db.QueryRowContext(context.Background(),
		`SELECT indexed_at_mtime FROM indexed_files WHERE file_path=?`, path,
	).Scan(&mtime)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("analysis cache: read indexed mtime for %s: %w", path, err)
	}
	return mtime, true, nil
}

// IsStale reports whether a file must be re-indexed before a query trusts its
// fqn_index rows. A file is stale when it has never been indexed or when its
// current mtime is newer than the indexed mtime.
//
// When os.Stat fails (the file was deleted or is unreadable) IsStale returns
// (true, err): the stored rows can no longer be trusted, but the caller decides
// whether to re-parse (it cannot, for a deleted file) or drop the rows.
func (c *AnalysisCache) IsStale(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return true, fmt.Errorf("analysis cache: stat %s: %w", path, err)
	}
	indexedMtime, exists, err := c.GetIndexedFileMtime(path)
	if err != nil {
		return true, err
	}
	if !exists {
		return true, nil
	}
	return info.ModTime().Unix() > indexedMtime, nil
}
