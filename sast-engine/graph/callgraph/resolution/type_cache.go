// Package resolution provides type caching for inference performance.
package resolution

import (
	"container/list"
	"sync"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
)

// TypeCache provides LRU-based caching for inferred types.
// Thread-safe for concurrent access during parallel file processing.
type TypeCache struct {
	capacity int
	cache    map[string]*list.Element
	lru      *list.List
	mutex    sync.RWMutex

	// Metrics
	hits   int64
	misses int64
}

// cacheEntry stores a cached type with metadata.
type cacheEntry struct {
	key  string
	typ  core.Type
	file string // Source file for invalidation
}

// NewTypeCache creates a new TypeCache with the given capacity.
func NewTypeCache(capacity int) *TypeCache {
	if capacity <= 0 {
		capacity = 10000 // Default
	}
	return &TypeCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		lru:      list.New(),
	}
}

// Get retrieves a type from the cache.
// Returns the type and true if found, nil and false otherwise.
func (tc *TypeCache) Get(key string) (core.Type, bool) {
	tc.mutex.RLock()
	elem, found := tc.cache[key]
	tc.mutex.RUnlock()

	if !found {
		tc.mutex.Lock()
		tc.misses++
		tc.mutex.Unlock()
		return nil, false
	}

	tc.mutex.Lock()
	tc.lru.MoveToFront(elem)
	tc.hits++
	tc.mutex.Unlock()

	return elem.Value.(*cacheEntry).typ, true
}

// Put adds a type to the cache.
// If the cache is at capacity, evicts the least recently used entry.
func (tc *TypeCache) Put(key string, typ core.Type, file string) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	// Check if already exists
	if elem, found := tc.cache[key]; found {
		tc.lru.MoveToFront(elem)
		elem.Value.(*cacheEntry).typ = typ
		elem.Value.(*cacheEntry).file = file
		return
	}

	// Evict if at capacity
	if tc.lru.Len() >= tc.capacity {
		tc.evictOldest()
	}

	// Add new entry
	entry := &cacheEntry{key: key, typ: typ, file: file}
	elem := tc.lru.PushFront(entry)
	tc.cache[key] = elem
}

// evictOldest removes the least recently used entry.
// Must be called with mutex held.
func (tc *TypeCache) evictOldest() {
	elem := tc.lru.Back()
	if elem != nil {
		tc.lru.Remove(elem)
		delete(tc.cache, elem.Value.(*cacheEntry).key)
	}
}

// InvalidateFile removes all entries associated with a file.
// Used when a file is modified.
func (tc *TypeCache) InvalidateFile(file string) int {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	count := 0
	var toRemove []*list.Element

	for _, elem := range tc.cache {
		if elem.Value.(*cacheEntry).file == file {
			toRemove = append(toRemove, elem)
		}
	}

	for _, elem := range toRemove {
		tc.lru.Remove(elem)
		delete(tc.cache, elem.Value.(*cacheEntry).key)
		count++
	}

	return count
}

// Clear removes all entries from the cache.
func (tc *TypeCache) Clear() {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()

	tc.cache = make(map[string]*list.Element)
	tc.lru = list.New()
}

// Stats returns cache statistics.
func (tc *TypeCache) Stats() (hits, misses int64, size int) {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()

	return tc.hits, tc.misses, tc.lru.Len()
}

// HitRate returns the cache hit rate as a percentage.
func (tc *TypeCache) HitRate() float64 {
	tc.mutex.RLock()
	defer tc.mutex.RUnlock()

	total := tc.hits + tc.misses
	if total == 0 {
		return 0.0
	}
	return float64(tc.hits) / float64(total) * 100.0
}

// MakeCacheKey creates a cache key for a variable at a location.
func MakeCacheKey(file string, line, col int, varName string) string {
	return file + ":" + varName + "@" + itoa(line) + ":" + itoa(col)
}

// Simple int to string (avoid fmt import for performance).
func itoa(i int) string {
	if i == 0 {
		return "0"
	}

	neg := false
	if i < 0 {
		neg = true
		i = -i
	}

	buf := make([]byte, 20)
	pos := len(buf)

	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}

	if neg {
		pos--
		buf[pos] = '-'
	}

	return string(buf[pos:])
}
