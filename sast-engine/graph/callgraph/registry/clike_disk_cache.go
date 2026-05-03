package registry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// diskCacheStore is a thin on-disk cache for stdlib registry JSON files. It
// mirrors the cache shape used by the Python and Go stdlib loaders: per-file
// entries with a 24h TTL and a "fall back to stale data on network failure"
// rule (the latter is enforced by the loader, not the cache).
//
// The cache is filesystem-backed. Each cached file is stored as a regular
// JSON file inside rootDir; freshness is determined by file mtime. On Linux
// and macOS the canonical rootDir is `~/.cache/pathfinder/registries/...`;
// on Windows we fall back to `%LOCALAPPDATA%\pathfinder\registries\...`.
//
// PR-02 ships the structure but the file:// loader does NOT use it (no point
// caching a local-file read). The HTTP loader in PR-03 will exercise it.
type diskCacheStore struct {
	rootDir string
}

// newDiskCacheStore returns a store rooted at rootDir. The directory is not
// created eagerly — Save* methods MkdirAll on first write so cold-start
// scans without write permission still succeed (with cache silently
// disabled).
func newDiskCacheStore(rootDir string) *diskCacheStore {
	return &diskCacheStore{rootDir: rootDir}
}

// SaveManifest writes the manifest JSON bytes to disk. Errors are non-fatal:
// returning err lets the caller log a warning, but loaders should still serve
// the manifest from memory and continue scanning.
func (s *diskCacheStore) SaveManifest(data []byte) error {
	if s == nil || s.rootDir == "" {
		return errors.New("diskCacheStore: not configured")
	}
	if err := os.MkdirAll(s.rootDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.rootDir, "manifest.json"), data, 0o644)
}

// GetManifest reads the cached manifest JSON. Returns ErrCacheMiss when no
// cached copy exists.
func (s *diskCacheStore) GetManifest() (*core.CStdlibManifest, error) {
	if s == nil || s.rootDir == "" {
		return nil, ErrCacheMiss
	}
	path := filepath.Join(s.rootDir, "manifest.json")
	data, err := os.ReadFile(path) //nolint:gosec // path is under our managed cache dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}
	var m core.CStdlibManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// SaveHeader writes one per-header JSON file. headerName is the manifest's
// File field (e.g., "stdio_stdlib.json") — the cache uses the same filename
// to keep on-disk layout identical to the CDN layout.
func (s *diskCacheStore) SaveHeader(filename string, data []byte) error {
	if s == nil || s.rootDir == "" {
		return errors.New("diskCacheStore: not configured")
	}
	if err := os.MkdirAll(s.rootDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.rootDir, filename), data, 0o644)
}

// GetHeader reads a cached per-header JSON. Returns ErrCacheMiss when the
// file is absent.
func (s *diskCacheStore) GetHeader(filename string) (*core.CStdlibHeader, error) {
	if s == nil || s.rootDir == "" {
		return nil, ErrCacheMiss
	}
	path := filepath.Join(s.rootDir, filename)
	data, err := os.ReadFile(path) //nolint:gosec // path is under our managed cache dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrCacheMiss
		}
		return nil, err
	}
	var h core.CStdlibHeader
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, err
	}
	return &h, nil
}

// IsFresh reports whether the on-disk file at filename was written within
// the given TTL. A file that doesn't exist is always stale; the test is
// inclusive of the TTL boundary (file mtime == now-TTL is still fresh).
func (s *diskCacheStore) IsFresh(filename string, ttl time.Duration) bool {
	if s == nil || s.rootDir == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(s.rootDir, filename))
	if err != nil {
		return false
	}
	return time.Since(info.ModTime()) <= ttl
}

// ErrCacheMiss is returned by GetManifest / GetHeader when no cached copy is
// present. Loaders distinguish ErrCacheMiss from other errors so they can
// fall through to a network fetch without surfacing the miss as a warning.
var ErrCacheMiss = errors.New("cache miss")

// stdlibCacheTTL is the freshness window for cached registry files. Stdlib
// libraries change slowly so a 24h cache is a good balance between fresh
// content and avoiding network calls on every scan.
const stdlibCacheTTL = 24 * time.Hour

// getStdlibCacheRoot returns the platform-conventional cache root for stdlib
// registries. On Linux/macOS: $XDG_CACHE_HOME/pathfinder/registries (falling
// back to $HOME/.cache/pathfinder/registries). On Windows: $LOCALAPPDATA/
// pathfinder/registries.
//
// Returns "" if no usable directory can be discovered — callers should treat
// that as "cache disabled" and continue without one.
func getStdlibCacheRoot() string {
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("LOCALAPPDATA"); appdata != "" {
			return filepath.Join(appdata, "pathfinder", "registries")
		}
	}
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "pathfinder", "registries")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".cache", "pathfinder", "registries")
	}
	return ""
}
