package callgraph

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewImportMapCache(t *testing.T) {
	cache := NewImportMapCache()
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.cache)
	assert.Equal(t, 0, len(cache.cache))
}

func TestImportMapCache_GetEmpty(t *testing.T) {
	cache := NewImportMapCache()

	importMap, ok := cache.Get("/nonexistent/file.py")
	assert.False(t, ok)
	assert.Nil(t, importMap)
}

func TestImportMapCache_PutAndGet(t *testing.T) {
	cache := NewImportMapCache()
	filePath := "/test/file.py"

	// Create a test ImportMap
	testImportMap := NewImportMap(filePath)
	testImportMap.AddImport("os", "os")
	testImportMap.AddImport("json", "json")

	// Put in cache
	cache.Put(filePath, testImportMap)

	// Get from cache
	retrieved, ok := cache.Get(filePath)
	assert.True(t, ok)
	assert.NotNil(t, retrieved)
	assert.Equal(t, filePath, retrieved.FilePath)
	assert.Equal(t, "os", retrieved.Imports["os"])
	assert.Equal(t, "json", retrieved.Imports["json"])
}

func TestImportMapCache_GetOrExtract_CacheHit(t *testing.T) {
	cache := NewImportMapCache()
	registry := NewModuleRegistry()
	filePath := "/test/file.py"

	// Pre-populate cache
	cachedImportMap := NewImportMap(filePath)
	cachedImportMap.AddImport("cached", "cached.module")
	cache.Put(filePath, cachedImportMap)

	// GetOrExtract should return cached version (sourceCode won't be used)
	result, err := cache.GetOrExtract(filePath, []byte("# dummy code"), registry)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "cached.module", result.Imports["cached"])
}

func TestImportMapCache_GetOrExtract_CacheMiss(t *testing.T) {
	cache := NewImportMapCache()
	registry := NewModuleRegistry()
	filePath := "../../../test-src/python/imports_test/simple_imports.py"

	// Read test file
	sourceCode, err := readFileBytes(filePath)
	assert.NoError(t, err)

	// GetOrExtract should extract and cache
	result, err := cache.GetOrExtract(filePath, sourceCode, registry)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify it's now in cache
	cached, ok := cache.Get(filePath)
	assert.True(t, ok)
	assert.Equal(t, result, cached)
}

func TestImportMapCache_Concurrent(t *testing.T) {
	cache := NewImportMapCache()
	registry := NewModuleRegistry()
	filePath := "../../../test-src/python/imports_test/simple_imports.py"

	sourceCode, err := readFileBytes(filePath)
	assert.NoError(t, err)

	// Launch multiple goroutines to access cache concurrently
	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make([]error, numGoroutines)
	results := make([]*ImportMap, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()
			result, getErr := cache.GetOrExtract(filePath, sourceCode, registry)
			errors[index] = getErr
			results[index] = result
		}(i)
	}

	wg.Wait()

	// All goroutines should succeed
	for i := 0; i < numGoroutines; i++ {
		assert.NoError(t, errors[i], "Goroutine %d should not error", i)
		assert.NotNil(t, results[i], "Goroutine %d should return a result", i)
	}

	// All results should be identical (same cached instance or semantically equal)
	for i := 1; i < numGoroutines; i++ {
		assert.Equal(t, results[0].FilePath, results[i].FilePath)
		assert.Equal(t, len(results[0].Imports), len(results[i].Imports))
	}

	// Cache should only contain one entry
	assert.Equal(t, 1, len(cache.cache))
}

func TestImportMapCache_MultipleFiles(t *testing.T) {
	cache := NewImportMapCache()

	file1 := "/test/file1.py"
	file2 := "/test/file2.py"
	file3 := "/test/file3.py"

	// Add multiple entries
	cache.Put(file1, NewImportMap(file1))
	cache.Put(file2, NewImportMap(file2))
	cache.Put(file3, NewImportMap(file3))

	// Verify all are cached
	_, ok1 := cache.Get(file1)
	_, ok2 := cache.Get(file2)
	_, ok3 := cache.Get(file3)

	assert.True(t, ok1)
	assert.True(t, ok2)
	assert.True(t, ok3)
	assert.Equal(t, 3, len(cache.cache))
}

func TestImportMapCache_OverwriteExisting(t *testing.T) {
	cache := NewImportMapCache()
	filePath := "/test/file.py"

	// Add first version
	firstMap := NewImportMap(filePath)
	firstMap.AddImport("first", "first.module")
	cache.Put(filePath, firstMap)

	// Overwrite with second version
	secondMap := NewImportMap(filePath)
	secondMap.AddImport("second", "second.module")
	cache.Put(filePath, secondMap)

	// Should have second version
	result, ok := cache.Get(filePath)
	assert.True(t, ok)
	assert.Equal(t, "second.module", result.Imports["second"])
	assert.NotContains(t, result.Imports, "first")
}

func BenchmarkImportMapCache_Get(b *testing.B) {
	cache := NewImportMapCache()
	filePath := "/test/file.py"
	testMap := NewImportMap(filePath)
	cache.Put(filePath, testMap)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(filePath)
	}
}

func BenchmarkImportMapCache_Put(b *testing.B) {
	cache := NewImportMapCache()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		filePath := "/test/file.py"
		testMap := NewImportMap(filePath)
		cache.Put(filePath, testMap)
	}
}

func BenchmarkImportMapCache_ConcurrentGet(b *testing.B) {
	cache := NewImportMapCache()
	filePath := "/test/file.py"
	testMap := NewImportMap(filePath)
	cache.Put(filePath, testMap)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cache.Get(filePath)
		}
	})
}
