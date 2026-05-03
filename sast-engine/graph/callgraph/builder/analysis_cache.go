package builder

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, no CGO conflict
)

// Per-table schema versions.
//
// Upgrade policy:
//   - Adding new JSON fields to cached structs → NO version bump needed.
//     Go's JSON decoder sets missing fields to their zero value, so old rows
//     deserialize cleanly into newer structs.
//   - Adding a new SQL column to a table → bump only that table's version.
//     Only that table is wiped; all other tables remain warm.
//   - Breaking SQL change (rename/drop column) → bump that table's version.
//   - Adding a new SQL table → no version bump needed (CREATE TABLE IF NOT EXISTS).
//
// At worst, an affected table is wiped and rebuilt from a single full analysis
// run; unaffected tables stay warm.
const (
	fileCacheVersion    = "1"
	functionIndexVersion = "1"
	pass4Version        = "1"
)

// CachedCallSite is the minimal data needed to reconstruct a CallSiteInternal.
type CachedCallSite struct {
	CallerFQN    string   `json:"callerFqn"`
	CallerFile   string   `json:"callerFile"`
	CallLine     uint32   `json:"callLine"`
	FunctionName string   `json:"functionName"`
	ObjectName   string   `json:"objectName,omitempty"`
	Arguments    []string `json:"arguments,omitempty"`
}

// CachedScope holds the variable bindings for all function scopes within one file.
type CachedScope struct {
	// Map from function FQN to its per-variable bindings.
	FunctionScopes map[string]CachedFunctionScope `json:"functionScopes"`
}

// CachedFunctionScope holds the variable bindings for one function.
type CachedFunctionScope struct {
	FunctionFQN string                   `json:"functionFqn"`
	Variables   map[string]CachedBinding `json:"variables"`
}

// CachedBinding is the serialisable form of the latest GoVariableBinding for a variable.
type CachedBinding struct {
	TypeFQN      string  `json:"typeFqn"`
	Confidence   float32 `json:"confidence"`
	Source       string  `json:"source"`
	AssignedFrom string  `json:"assignedFrom,omitempty"`
}

// CachedFilePResult holds everything recovered from a warm cache entry for one file.
type CachedFilePResult struct {
	CallSites []CachedCallSite
	Scope     *CachedScope
}

// CacheStats tracks how many files were served from cache vs re-analysed.
type CacheStats struct {
	HitFiles  int // files whose hash matched — hot-loaded from cache
	MissFiles int // files that were re-analysed (new or changed)
}

// --- Pass 4 cache types ---

// CachedArgument is the serialisable form of core.Argument.
type CachedArgument struct {
	Value      string `json:"value"`
	IsVariable bool   `json:"isVariable,omitempty"`
	Position   int    `json:"position"`
}

// CachedPass4Edge represents one call site result from Pass 4.
// Both resolved and unresolved edges are stored so we can replay Stage 2 exactly.
type CachedPass4Edge struct {
	CallerFQN string `json:"callerFqn"`
	TargetFQN string `json:"targetFqn,omitempty"` // empty when unresolved
	Resolved  bool   `json:"resolved"`
	// core.CallSite fields — stored flat to avoid importing core in this file.
	Target                   string           `json:"target"`
	File                     string           `json:"file"`
	Line                     int              `json:"line"`
	Arguments                []CachedArgument `json:"arguments,omitempty"`
	IsStdlib                 bool             `json:"isStdlib,omitempty"`
	ResolvedViaTypeInference bool             `json:"resolvedViaTypeInference,omitempty"`
	InferredType             string           `json:"inferredType,omitempty"`
	TypeConfidence           float32          `json:"typeConfidence,omitempty"`
	TypeSource               string           `json:"typeSource,omitempty"`
	FailureReason            string           `json:"failureReason,omitempty"`
}

// CachedPass4Result holds the cached Pass 4 output for one source file.
type CachedPass4Result struct {
	ContentHash     string            `json:"contentHash"`
	Edges           []CachedPass4Edge `json:"edges"`
	// UnresolvedNames are the FunctionName values of unresolved call sites in this
	// file.  They are stored so that, when new functions are added to the index,
	// we can detect that a previously-failing call site might now resolve and mark
	// the file as dirty again.
	UnresolvedNames []string `json:"unresolvedNames"`
}

// AnalysisCache is a per-project SQLite database that caches:
//   - Pass 2b variable scopes and Pass 3 call sites (file_cache table)
//   - Pass 4 resolved edges (pass4_results table)
//   - Pass 1 function index snapshot (function_index table)
//
// Thread-safety: the DB connection serialises writes; parallel goroutines should
// only call Get* (reads) and flush with Put* sequentially afterwards.
type AnalysisCache struct {
	db *sql.DB
}

// OpenAnalysisCache opens (or creates) the SQLite cache for the given project
// root. The database lives at ~/.cache/pathfinder/<hex(sha256(projectRoot))[:16]>.db.
func OpenAnalysisCache(projectRoot string) (*AnalysisCache, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	pfDir := filepath.Join(cacheDir, "pathfinder")
	if err := os.MkdirAll(pfDir, 0o755); err != nil {
		return nil, fmt.Errorf("analysis cache: cannot create cache dir %s: %w", pfDir, err)
	}

	// Derive a short, stable identifier from the project root path.
	h := sha256.Sum256([]byte(projectRoot))
	dbName := hex.EncodeToString(h[:])[:16] + ".db"
	dbPath := filepath.Join(pfDir, dbName)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("analysis cache: open %s: %w", dbPath, err)
	}

	// Enable WAL mode for better concurrency (readers don't block writers).
	if _, err := db.ExecContext(context.Background(), "PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("analysis cache: set WAL mode: %w", err)
	}

	if err := initSchema(db, projectRoot); err != nil {
		db.Close()
		return nil, err
	}

	return &AnalysisCache{db: db}, nil
}

// DBPath returns the filesystem path of the SQLite database file, for diagnostics.
func (c *AnalysisCache) DBPath() string {
	// Query sqlite_master for the database filename via the pragma.
	var path string
	_ = c.db.QueryRowContext(context.Background(), `PRAGMA database_list`).Scan(nil, nil, &path)
	return path
}

// initSchema creates tables and runs per-table version checks.
// Only tables whose stored version differs from the current version are wiped;
// unchanged tables keep their warm data.
func initSchema(db *sql.DB, projectRoot string) error {
	createStmts := []string{
		`CREATE TABLE IF NOT EXISTS meta (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS file_cache (
			file_path        TEXT    PRIMARY KEY,
			content_hash     TEXT    NOT NULL,
			updated_at       INTEGER NOT NULL,
			call_sites_json  TEXT    NOT NULL,
			scope_json       TEXT    NOT NULL
		)`,
		// Pass 1 function index snapshot — one row per (file, fqn) pair.
		`CREATE TABLE IF NOT EXISTS function_index (
			file_path TEXT NOT NULL,
			fqn       TEXT NOT NULL,
			PRIMARY KEY (file_path, fqn)
		)`,
		// Pass 4 resolved-edge cache — one row per source file.
		`CREATE TABLE IF NOT EXISTS pass4_results (
			file_path       TEXT    PRIMARY KEY,
			content_hash    TEXT    NOT NULL,
			updated_at      INTEGER NOT NULL,
			edges_json      TEXT    NOT NULL,
			unresolved_json TEXT    NOT NULL
		)`,
	}
	for _, stmt := range createStmts {
		if _, err := db.ExecContext(context.Background(), stmt); err != nil {
			return fmt.Errorf("analysis cache: create schema: %w", err)
		}
	}

	// Always upsert project_root so a moved project gets its own DB.
	var storedRoot string
	_ = db.QueryRowContext(context.Background(), `SELECT value FROM meta WHERE key='project_root'`).Scan(&storedRoot)
	if storedRoot != "" && storedRoot != projectRoot {
		// Project root changed — wipe everything; this is a different project.
		for _, tbl := range []string{"file_cache", "function_index", "pass4_results"} {
			if _, err := db.ExecContext(context.Background(), `DELETE FROM `+tbl); err != nil {
				return fmt.Errorf("analysis cache: wipe table %s on project root change: %w", tbl, err)
			}
		}
	}

	// Per-table version check: only wipe tables whose version changed.
	tableVersions := []struct {
		metaKey string
		current string
		table   string
	}{
		{"file_cache_version", fileCacheVersion, "file_cache"},
		{"function_index_version", functionIndexVersion, "function_index"},
		{"pass4_version", pass4Version, "pass4_results"},
	}
	for _, tv := range tableVersions {
		var stored string
		_ = db.QueryRowContext(context.Background(),
			`SELECT value FROM meta WHERE key=?`, tv.metaKey,
		).Scan(&stored)
		if stored != tv.current {
			if _, err := db.ExecContext(context.Background(), `DELETE FROM `+tv.table); err != nil {
				return fmt.Errorf("analysis cache: wipe %s on version change: %w", tv.table, err)
			}
		}
	}

	// Upsert all metadata.
	upserts := []struct{ k, v string }{
		{"project_root", projectRoot},
		{"file_cache_version", fileCacheVersion},
		{"function_index_version", functionIndexVersion},
		{"pass4_version", pass4Version},
	}
	for _, kv := range upserts {
		if _, err := db.ExecContext(context.Background(),
			`INSERT OR REPLACE INTO meta(key,value) VALUES(?,?)`, kv.k, kv.v,
		); err != nil {
			return fmt.Errorf("analysis cache: upsert meta %q: %w", kv.k, err)
		}
	}
	return nil
}

// Close releases the database connection.
func (c *AnalysisCache) Close() error {
	return c.db.Close()
}

// ---- Pass 2b / Pass 3 cache (file_cache table) ----

// GetFileCached returns the cached analysis result for a file if the stored
// sha256 matches the current file content. Returns (nil, false) on any miss.
func (c *AnalysisCache) GetFileCached(filePath string) (*CachedFilePResult, bool) {
	currentHash, err := hashFile(filePath)
	if err != nil {
		return nil, false
	}

	var storedHash, callSitesJSON, scopeJSON string
	err = c.db.QueryRowContext(context.Background(),
		`SELECT content_hash, call_sites_json, scope_json FROM file_cache WHERE file_path=?`,
		filePath,
	).Scan(&storedHash, &callSitesJSON, &scopeJSON)
	if err != nil {
		return nil, false
	}

	if storedHash != currentHash {
		return nil, false // content changed
	}

	var callSites []CachedCallSite
	if err := json.Unmarshal([]byte(callSitesJSON), &callSites); err != nil {
		return nil, false
	}

	var scope CachedScope
	if err := json.Unmarshal([]byte(scopeJSON), &scope); err != nil {
		return nil, false
	}

	return &CachedFilePResult{
		CallSites: callSites,
		Scope:     &scope,
	}, true
}

// PutFileCached stores the analysis results for a file.
func (c *AnalysisCache) PutFileCached(filePath string, callSites []CachedCallSite, scope *CachedScope) error {
	hash, err := hashFile(filePath)
	if err != nil {
		return fmt.Errorf("analysis cache: hash %s: %w", filePath, err)
	}

	csJSON, err := json.Marshal(callSites)
	if err != nil {
		return fmt.Errorf("analysis cache: marshal call sites for %s: %w", filePath, err)
	}

	sJSON, err := json.Marshal(scope)
	if err != nil {
		return fmt.Errorf("analysis cache: marshal scope for %s: %w", filePath, err)
	}

	_, err = c.db.ExecContext(context.Background(),
		`INSERT OR REPLACE INTO file_cache(file_path, content_hash, updated_at, call_sites_json, scope_json)
		 VALUES(?,?,?,?,?)`,
		filePath, hash, time.Now().Unix(), string(csJSON), string(sJSON),
	)
	return err
}

// ---- Pass 1 function index (function_index table) ----

// LoadFunctionIndex loads the cached function index.
// Returns a map from file path to the list of FQNs defined in that file.
// Returns an empty map (not nil) if the table is empty.
func (c *AnalysisCache) LoadFunctionIndex() map[string][]string {
	result := make(map[string][]string)
	rows, err := c.db.QueryContext(context.Background(), `SELECT file_path, fqn FROM function_index`)
	if err != nil {
		return result
	}
	defer rows.Close()
	for rows.Next() {
		var file, fqn string
		if err := rows.Scan(&file, &fqn); err != nil {
			continue
		}
		result[file] = append(result[file], fqn)
	}
	if err := rows.Err(); err != nil {
		return make(map[string][]string) // return empty on iteration error
	}
	return result
}

// SaveFunctionIndex replaces the stored function index with the current one.
// It deletes all existing rows and inserts the new snapshot in a single transaction.
func (c *AnalysisCache) SaveFunctionIndex(index map[string][]string) error {
	tx, err := c.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("analysis cache: begin tx for function_index: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(context.Background(), `DELETE FROM function_index`); err != nil {
		return fmt.Errorf("analysis cache: clear function_index: %w", err)
	}

	stmt, err := tx.PrepareContext(context.Background(), `INSERT INTO function_index(file_path, fqn) VALUES(?,?)`)
	if err != nil {
		return fmt.Errorf("analysis cache: prepare function_index insert: %w", err)
	}
	defer stmt.Close()

	for file, fqns := range index {
		for _, fqn := range fqns {
			if _, err := stmt.ExecContext(context.Background(), file, fqn); err != nil {
				return fmt.Errorf("analysis cache: insert function_index (%s, %s): %w", file, fqn, err)
			}
		}
	}

	return tx.Commit()
}

// ComputeFunctionIndexDelta compares a cached function index against the current one.
// Returns the set of FQNs that were added (new functions) and removed (deleted/renamed).
func ComputeFunctionIndexDelta(cached, current map[string][]string) (added, removed map[string]bool) {
	added = make(map[string]bool)
	removed = make(map[string]bool)

	cachedSet := make(map[string]bool)
	for _, fqns := range cached {
		for _, fqn := range fqns {
			cachedSet[fqn] = true
		}
	}

	currentSet := make(map[string]bool)
	for _, fqns := range current {
		for _, fqn := range fqns {
			currentSet[fqn] = true
		}
	}

	for fqn := range currentSet {
		if !cachedSet[fqn] {
			added[fqn] = true
		}
	}
	for fqn := range cachedSet {
		if !currentSet[fqn] {
			removed[fqn] = true
		}
	}
	return added, removed
}

// ---- Pass 4 cache (pass4_results table) ----

// LoadPass4Results loads cached Pass 4 results for the given file paths.
// Returns a map from file path to its cached result; missing files are absent from the map.
func (c *AnalysisCache) LoadPass4Results(filePaths []string) map[string]*CachedPass4Result {
	result := make(map[string]*CachedPass4Result, len(filePaths))
	if len(filePaths) == 0 {
		return result
	}

	for _, fp := range filePaths {
		var contentHash, edgesJSON, unresolvedJSON string
		err := c.db.QueryRowContext(context.Background(),
			`SELECT content_hash, edges_json, unresolved_json FROM pass4_results WHERE file_path=?`,
			fp,
		).Scan(&contentHash, &edgesJSON, &unresolvedJSON)
		if err != nil {
			continue // cache miss
		}

		var edges []CachedPass4Edge
		if err := json.Unmarshal([]byte(edgesJSON), &edges); err != nil {
			continue
		}

		var unresolvedNames []string
		if err := json.Unmarshal([]byte(unresolvedJSON), &unresolvedNames); err != nil {
			continue
		}

		result[fp] = &CachedPass4Result{
			ContentHash:     contentHash,
			Edges:           edges,
			UnresolvedNames: unresolvedNames,
		}
	}
	return result
}

// SavePass4Results persists Pass 4 results for a set of files in a single transaction.
func (c *AnalysisCache) SavePass4Results(results map[string]*CachedPass4Result) error {
	if len(results) == 0 {
		return nil
	}

	tx, err := c.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("analysis cache: begin tx for pass4_results: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(context.Background(),
		`INSERT OR REPLACE INTO pass4_results(file_path, content_hash, updated_at, edges_json, unresolved_json)
		 VALUES(?,?,?,?,?)`)
	if err != nil {
		return fmt.Errorf("analysis cache: prepare pass4_results insert: %w", err)
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for fp, r := range results {
		edgesJSON, err := json.Marshal(r.Edges)
		if err != nil {
			return fmt.Errorf("analysis cache: marshal pass4 edges for %s: %w", fp, err)
		}
		unresolvedJSON, err := json.Marshal(r.UnresolvedNames)
		if err != nil {
			return fmt.Errorf("analysis cache: marshal unresolved names for %s: %w", fp, err)
		}
		if _, err := stmt.ExecContext(context.Background(), fp, r.ContentHash, now, string(edgesJSON), string(unresolvedJSON)); err != nil {
			return fmt.Errorf("analysis cache: insert pass4_results for %s: %w", fp, err)
		}
	}
	return tx.Commit()
}

// NeedsPass4Rerun reports whether a file's Pass 4 results must be recomputed.
//
// A file is dirty for Pass 4 when any of the following is true:
//  1. No cached result exists (first run).
//  2. The file's content hash has changed (call sites inside may differ).
//  3. Any resolved target FQN was removed from the function index (callee renamed/deleted).
//  4. Any previously-unresolved call name now matches a newly-added FQN (new callee appeared).
func NeedsPass4Rerun(cached *CachedPass4Result, currentHash string, addedFQNs, removedFQNs map[string]bool) bool {
	if cached == nil {
		return true
	}
	if cached.ContentHash != currentHash {
		return true
	}
	// Check 3: a resolved callee disappeared.
	for _, edge := range cached.Edges {
		if edge.Resolved && removedFQNs[edge.TargetFQN] {
			return true
		}
	}
	// Check 4: a previously-unresolved name might now resolve.
	if len(addedFQNs) > 0 {
		for _, name := range cached.UnresolvedNames {
			for fqn := range addedFQNs {
				// Match on the simple name suffix: "pkg.FuncName" → "FuncName"
				if strings.HasSuffix(fqn, "."+name) {
					return true
				}
			}
		}
	}
	return false
}

// hashFile returns the hex-encoded SHA-256 digest of a file's contents.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
