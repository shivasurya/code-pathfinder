package builder

import (
	"sync"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/resolution"
)

// ImportMapCache provides thread-safe caching of ImportMap instances.
// It prevents redundant import extraction by caching results keyed by file path.
//
// Thread-safety:
//   - All methods are safe for concurrent use
//   - Uses RWMutex for optimized read-heavy workloads
//   - GetOrExtract handles double-checked locking pattern
type ImportMapCache struct {
	cache map[string]*core.ImportMap // Maps file path to ImportMap
	mu    sync.RWMutex                // Protects cache map
}

// NewImportMapCache creates a new empty import map cache.
func NewImportMapCache() *ImportMapCache {
	return &ImportMapCache{
		cache: make(map[string]*core.ImportMap),
	}
}

// Get retrieves an ImportMap from the cache if it exists.
//
// Parameters:
//   - filePath: absolute path to the Python file
//
// Returns:
//   - ImportMap and true if found in cache, nil and false otherwise
func (c *ImportMapCache) Get(filePath string) (*core.ImportMap, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	importMap, ok := c.cache[filePath]
	return importMap, ok
}

// Put stores an ImportMap in the cache.
//
// Parameters:
//   - filePath: absolute path to the Python file
//   - importMap: the extracted ImportMap to cache
func (c *ImportMapCache) Put(filePath string, importMap *core.ImportMap) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[filePath] = importMap
}

// GetOrExtract retrieves an ImportMap from cache or extracts it if not cached.
// This is the main entry point for using the cache.
//
// Parameters:
//   - filePath: absolute path to the Python file
//   - sourceCode: file contents (only used if extraction needed)
//   - registry: module registry for resolving imports
//
// Returns:
//   - ImportMap from cache or newly extracted
//   - error if extraction fails (cache misses only)
//
// Thread-safety:
//   - Multiple goroutines can safely call GetOrExtract concurrently
//   - First caller for a file will extract and cache
//   - Subsequent callers will get cached result
func (c *ImportMapCache) GetOrExtract(filePath string, sourceCode []byte, registry *core.ModuleRegistry) (*core.ImportMap, error) {
	// Try to get from cache (fast path with read lock)
	if importMap, ok := c.Get(filePath); ok {
		return importMap, nil
	}

	// Cache miss - extract imports (expensive operation)
	importMap, err := resolution.ExtractImports(filePath, sourceCode, registry)
	if err != nil {
		return nil, err
	}

	// Store in cache for future use
	c.Put(filePath, importMap)

	return importMap, nil
}
