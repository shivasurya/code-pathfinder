package resolution

import (
	"sync"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTypeCache_PutAndGet(t *testing.T) {
	cache := NewTypeCache(100)
	typ := core.NewConcreteType("myapp.User", 0.9)

	cache.Put("key1", typ, "file.py")

	retrieved, found := cache.Get("key1")
	require.True(t, found)
	assert.True(t, typ.Equals(retrieved))
}

func TestTypeCache_Miss(t *testing.T) {
	cache := NewTypeCache(100)

	retrieved, found := cache.Get("nonexistent")
	assert.False(t, found)
	assert.Nil(t, retrieved)
}

func TestTypeCache_LRUEviction(t *testing.T) {
	cache := NewTypeCache(3)

	cache.Put("key1", core.NewConcreteType("Type1", 0.9), "file.py")
	cache.Put("key2", core.NewConcreteType("Type2", 0.9), "file.py")
	cache.Put("key3", core.NewConcreteType("Type3", 0.9), "file.py")

	// Access key1 to make it recently used
	cache.Get("key1")

	// Add new entry, should evict key2 (least recently used)
	cache.Put("key4", core.NewConcreteType("Type4", 0.9), "file.py")

	_, found := cache.Get("key1")
	assert.True(t, found, "key1 should still be in cache")

	_, found = cache.Get("key2")
	assert.False(t, found, "key2 should be evicted")

	_, found = cache.Get("key3")
	assert.True(t, found, "key3 should still be in cache")

	_, found = cache.Get("key4")
	assert.True(t, found, "key4 should be in cache")
}

func TestTypeCache_Update(t *testing.T) {
	cache := NewTypeCache(100)

	cache.Put("key1", core.NewConcreteType("OldType", 0.9), "file.py")
	cache.Put("key1", core.NewConcreteType("NewType", 0.9), "file.py")

	retrieved, found := cache.Get("key1")
	require.True(t, found)

	ct, ok := core.ExtractConcreteType(retrieved)
	require.True(t, ok)
	assert.Equal(t, "NewType", ct.Name)
}

func TestTypeCache_InvalidateFile(t *testing.T) {
	cache := NewTypeCache(100)

	cache.Put("key1", core.NewConcreteType("Type1", 0.9), "file1.py")
	cache.Put("key2", core.NewConcreteType("Type2", 0.9), "file1.py")
	cache.Put("key3", core.NewConcreteType("Type3", 0.9), "file2.py")

	count := cache.InvalidateFile("file1.py")
	assert.Equal(t, 2, count)

	_, found := cache.Get("key1")
	assert.False(t, found)

	_, found = cache.Get("key2")
	assert.False(t, found)

	_, found = cache.Get("key3")
	assert.True(t, found, "key3 from different file should remain")
}

func TestTypeCache_Clear(t *testing.T) {
	cache := NewTypeCache(100)

	cache.Put("key1", core.NewConcreteType("Type1", 0.9), "file.py")
	cache.Put("key2", core.NewConcreteType("Type2", 0.9), "file.py")

	cache.Clear()

	_, found := cache.Get("key1")
	assert.False(t, found)

	_, found = cache.Get("key2")
	assert.False(t, found)
}

func TestTypeCache_Stats(t *testing.T) {
	cache := NewTypeCache(100)

	cache.Put("key1", core.NewConcreteType("Type1", 0.9), "file.py")

	cache.Get("key1") // Hit
	cache.Get("key1") // Hit
	cache.Get("key2") // Miss

	hits, misses, size := cache.Stats()
	assert.Equal(t, int64(2), hits)
	assert.Equal(t, int64(1), misses)
	assert.Equal(t, 1, size)
}

func TestTypeCache_HitRate(t *testing.T) {
	cache := NewTypeCache(100)

	// No operations
	assert.Equal(t, 0.0, cache.HitRate())

	cache.Put("key1", core.NewConcreteType("Type1", 0.9), "file.py")

	cache.Get("key1") // Hit
	cache.Get("key1") // Hit
	cache.Get("key2") // Miss
	cache.Get("key3") // Miss

	assert.Equal(t, 50.0, cache.HitRate())
}

func TestTypeCache_ConcurrentAccess(t *testing.T) {
	cache := NewTypeCache(1000)
	var wg sync.WaitGroup

	// Concurrent writes
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := MakeCacheKey("file.py", i, 0, "var")
			cache.Put(key, core.NewConcreteType("Type", 0.9), "file.py")
		}(i)
	}

	// Concurrent reads
	for i := range 100 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := MakeCacheKey("file.py", i, 0, "var")
			cache.Get(key)
		}(i)
	}

	wg.Wait()
	// No race condition = test passes
}

func TestMakeCacheKey(t *testing.T) {
	key := MakeCacheKey("myapp/models.py", 42, 10, "user")
	assert.Equal(t, "myapp/models.py:user@42:10", key)
}

func TestTypeCache_DefaultCapacity(t *testing.T) {
	cache := NewTypeCache(0) // Should default to 10000

	// Should not panic and work normally
	cache.Put("key1", core.NewConcreteType("Type1", 0.9), "file.py")
	_, found := cache.Get("key1")
	assert.True(t, found)
}

func TestItoa(t *testing.T) {
	assert.Equal(t, "0", itoa(0))
	assert.Equal(t, "42", itoa(42))
	assert.Equal(t, "123456", itoa(123456))
	assert.Equal(t, "-1", itoa(-1))
	assert.Equal(t, "-42", itoa(-42))
}
