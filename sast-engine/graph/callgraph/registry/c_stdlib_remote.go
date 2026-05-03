package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// CStdlibRegistryRemote is the loader behind core.CStdlibLoader for C stdlib
// registries. It supports two source modes — file:// (PR-02; used by tests
// and local-validation) and HTTP (stub here, full implementation in PR-03).
//
// The struct intentionally serves both modes through one type: file:// vs
// HTTP differs only in fetch (network vs filesystem read), and everything
// else (manifest validation, in-memory cache, double-check locking) is the
// same. This mirrors the existing GoStdlibRegistryRemote pattern.
type CStdlibRegistryRemote struct {
	baseURL  string // for HTTP path; empty in file:// mode
	platform string // "linux" | "windows" | "darwin"

	// fileBase is set on construction via NewCStdlibRegistryFile; non-empty
	// indicates file:// mode (skip HTTP machinery, read from disk).
	fileBase string

	manifest    *core.CStdlibManifest
	headerCache map[string]*core.CStdlibHeader
	cacheMutex  sync.RWMutex

	httpClient *http.Client    // unused in file:// mode
	diskCache  *diskCacheStore // unused in file:// mode (no point caching local reads)
}

// NewCStdlibRegistryFile constructs a file:// loader rooted at localPath.
// localPath should point at the directory that contains manifest.json plus
// the per-header *_stdlib.json files (the output of the PR-01 generator).
//
// PR-02 uses this constructor exclusively. PR-03 will switch the production
// default to NewCStdlibRegistryRemote.
func NewCStdlibRegistryFile(localPath, platform string) *CStdlibRegistryRemote {
	return &CStdlibRegistryRemote{
		platform:    platform,
		fileBase:    localPath,
		headerCache: make(map[string]*core.CStdlibHeader),
	}
}

// NewCStdlibRegistryRemote constructs an HTTP loader. The path under baseURL
// is `{platform}/c/v1/{file}` — matching the URL layout the manifest entries
// embed. The HTTP fetch implementation is stubbed in PR-02 and lands in PR-03.
func NewCStdlibRegistryRemote(baseURL, platform string) *CStdlibRegistryRemote {
	cacheRoot := getStdlibCacheRoot()
	var dc *diskCacheStore
	if cacheRoot != "" {
		dc = newDiskCacheStore(filepath.Join(cacheRoot, platform, "c", "v1"))
	}
	return &CStdlibRegistryRemote{
		baseURL:     strings.TrimSuffix(baseURL, "/"),
		platform:    platform,
		headerCache: make(map[string]*core.CStdlibHeader),
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		diskCache:   dc,
	}
}

// LoadManifest is the one-shot startup operation. It reads (or fetches) the
// top-level manifest.json and populates the in-memory pointer; subsequent
// per-header GetHeader calls consult that manifest for the URL/path of each
// per-header JSON.
//
// Logger is optional but expected — pass nil only when wiring the loader
// from tests where logging would be noise.
func (r *CStdlibRegistryRemote) LoadManifest(logger core.CStdlibLogger) error {
	if r.fileBase != "" {
		return r.loadManifestFromFile(logger)
	}
	return r.loadManifestFromHTTP(logger)
}

func (r *CStdlibRegistryRemote) loadManifestFromFile(logger core.CStdlibLogger) error {
	path := filepath.Join(r.fileBase, "manifest.json")
	data, err := os.ReadFile(path) //nolint:gosec // path is operator-supplied via CLI flag
	if err != nil {
		return fmt.Errorf("loadManifestFromFile: reading %s: %w", path, err)
	}

	var manifest core.CStdlibManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("loadManifestFromFile: parsing %s: %w", path, err)
	}

	r.cacheMutex.Lock()
	r.manifest = &manifest
	r.cacheMutex.Unlock()

	if logger != nil {
		logger.Statistic("Loaded C stdlib manifest from file: %d headers for %s",
			len(manifest.Headers), r.platform)
	}
	return nil
}

// loadManifestFromHTTP is the PR-03 hook. PR-02 ships it as a deliberate stub
// so the type satisfies CStdlibLoader without any half-built network code
// shipping early.
func (r *CStdlibRegistryRemote) loadManifestFromHTTP(_ core.CStdlibLogger) error {
	return errors.New("CStdlibRegistryRemote: HTTP loader not yet implemented; tracked in PR-03")
}

// GetHeader retrieves the per-header content, fetching on first reference and
// caching afterward. Concurrent access is safe — the double-check pattern
// guarantees at most one fetch per header even under heavy parallelism. The
// implementation mirrors GoStdlibRegistryRemote.GetPackage at
// graph/callgraph/registry/go_stdlib_remote.go:96-160.
func (r *CStdlibRegistryRemote) GetHeader(name string) (*core.CStdlibHeader, error) {
	// Fast path: read lock + cache hit.
	r.cacheMutex.RLock()
	if h, ok := r.headerCache[name]; ok {
		r.cacheMutex.RUnlock()
		return h, nil
	}
	r.cacheMutex.RUnlock()

	// Slow path: write lock, double-check, then fetch.
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()

	if h, ok := r.headerCache[name]; ok {
		return h, nil
	}

	h, err := r.fetchHeaderLocked(name)
	if err != nil {
		return nil, err
	}
	r.headerCache[name] = h
	return h, nil
}

// fetchHeaderLocked is the inner fetch routine — caller already holds the
// write lock. Re-checks the cache (in case another goroutine populated it),
// then dispatches to file:// or HTTP based on mode.
func (r *CStdlibRegistryRemote) fetchHeaderLocked(name string) (*core.CStdlibHeader, error) {
	if h, ok := r.headerCache[name]; ok {
		return h, nil
	}
	if r.manifest == nil {
		return nil, errors.New("fetchHeaderLocked: manifest not loaded; call LoadManifest first")
	}

	entry := r.manifest.GetHeaderEntry(name)
	if entry == nil {
		return nil, fmt.Errorf("fetchHeaderLocked: header %q not in stdlib manifest", name)
	}

	if r.fileBase != "" {
		return r.fetchHeaderFromFile(entry)
	}
	return r.fetchHeaderFromHTTP(entry)
}

func (r *CStdlibRegistryRemote) fetchHeaderFromFile(entry *core.CStdlibHeaderEntry) (*core.CStdlibHeader, error) {
	path := filepath.Join(r.fileBase, entry.File)
	data, err := os.ReadFile(path) //nolint:gosec // path is operator-supplied via CLI flag
	if err != nil {
		return nil, fmt.Errorf("fetchHeaderFromFile: reading %s: %w", path, err)
	}
	var h core.CStdlibHeader
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("fetchHeaderFromFile: parsing %s: %w", path, err)
	}
	return &h, nil
}

// fetchHeaderFromHTTP is the PR-03 hook. PR-02 stub keeps the type
// satisfying its interface contract without shipping half-built network code.
func (r *CStdlibRegistryRemote) fetchHeaderFromHTTP(_ *core.CStdlibHeaderEntry) (*core.CStdlibHeader, error) {
	return nil, errors.New("CStdlibRegistryRemote: HTTP fetch not yet implemented; tracked in PR-03")
}

// GetFunction is a convenience accessor: GetHeader followed by a function
// lookup. Returns the same error shapes the underlying calls produce.
func (r *CStdlibRegistryRemote) GetFunction(headerName, funcName string) (*core.CStdlibFunction, error) {
	h, err := r.GetHeader(headerName)
	if err != nil {
		return nil, err
	}
	if f, ok := h.Functions[funcName]; ok {
		return f, nil
	}
	if f, ok := h.FreeFunctions[funcName]; ok {
		return f, nil
	}
	return nil, fmt.Errorf("GetFunction: %q not in header %q", funcName, headerName)
}

// Platform returns the platform tag this registry was configured for.
// Used by the resolver to log diagnostics; not part of the cache key.
func (r *CStdlibRegistryRemote) Platform() string {
	return r.platform
}

// HeaderCount returns the total number of headers listed in the loaded
// manifest. Returns 0 before LoadManifest has been called.
func (r *CStdlibRegistryRemote) HeaderCount() int {
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()
	if r.manifest == nil {
		return 0
	}
	return len(r.manifest.Headers)
}

// Compile-time interface checks — fail at build time if the struct ever
// drifts from the contract.
var _ core.CStdlibLoader = (*CStdlibRegistryRemote)(nil)
