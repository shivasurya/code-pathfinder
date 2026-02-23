package builder

import (
	"sync"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewImportMapCache(t *testing.T) {
	cache := NewImportMapCache()
	assert.NotNil(t, cache)
	assert.NotNil(t, cache.cache)
	assert.Empty(t, cache.cache)
}

func TestImportMapCache_GetPut(t *testing.T) {
	cache := NewImportMapCache()
	filePath := "/test/file.py"

	// Initially should not exist
	importMap, ok := cache.Get(filePath)
	assert.False(t, ok)
	assert.Nil(t, importMap)

	// Put an ImportMap
	expectedImportMap := core.NewImportMap(filePath)
	expectedImportMap.AddImport("os", "os")
	cache.Put(filePath, expectedImportMap)

	// Should now exist
	importMap, ok = cache.Get(filePath)
	assert.True(t, ok)
	assert.Equal(t, expectedImportMap, importMap)
}

func TestImportMapCache_GetOrExtract_CacheHit(t *testing.T) {
	cache := NewImportMapCache()
	filePath := "/test/file.py"

	// Pre-populate cache
	expectedImportMap := core.NewImportMap(filePath)
	expectedImportMap.AddImport("sys", "sys")
	cache.Put(filePath, expectedImportMap)

	// GetOrExtract should return cached value without calling ExtractImports
	importMap, err := cache.GetOrExtract(filePath, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, expectedImportMap, importMap)
}

func TestImportMapCache_GetOrExtract_CacheMiss(t *testing.T) {
	cache := NewImportMapCache()
	filePath := "/test/file.py"

	// Simple Python code with imports
	sourceCode := []byte(`import os
import sys
from pathlib import Path
`)

	registry := core.NewModuleRegistry()

	// GetOrExtract should extract and cache
	importMap, err := cache.GetOrExtract(filePath, sourceCode, registry)
	require.NoError(t, err)
	assert.NotNil(t, importMap)

	// Should now be in cache
	cachedImportMap, ok := cache.Get(filePath)
	assert.True(t, ok)
	assert.Equal(t, importMap, cachedImportMap)
}

func TestImportMapCache_ConcurrentAccess(t *testing.T) {
	cache := NewImportMapCache()
	filePath := "/test/file.py"

	sourceCode := []byte(`import os`)
	registry := core.NewModuleRegistry()

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	results := make([]*core.ImportMap, numGoroutines)

	// Multiple goroutines try to get/extract concurrently
	for i := range numGoroutines {
		go func(index int) {
			defer wg.Done()
			importMap, err := cache.GetOrExtract(filePath, sourceCode, registry)
			assert.NoError(t, err)
			results[index] = importMap
		}(i)
	}

	wg.Wait()

	// All results should be non-nil
	for i := range numGoroutines {
		assert.NotNil(t, results[i])
	}

	// All should point to the same cached instance
	firstResult := results[0]
	for i := 1; i < numGoroutines; i++ {
		assert.Equal(t, firstResult, results[i], "Result %d should match first result", i)
	}
}

func TestImportMapCache_ConcurrentPut(t *testing.T) {
	cache := NewImportMapCache()

	const numGoroutines = 50
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Multiple goroutines put different files concurrently
	for i := range numGoroutines {
		go func(index int) {
			defer wg.Done()
			filePath := "/test/file" + string(rune('0'+index)) + ".py"
			importMap := core.NewImportMap(filePath)
			cache.Put(filePath, importMap)
		}(i)
	}

	wg.Wait()

	// All files should be in cache
	for i := range numGoroutines {
		filePath := "/test/file" + string(rune('0'+i)) + ".py"
		importMap, ok := cache.Get(filePath)
		assert.True(t, ok, "File %d should be in cache", i)
		assert.NotNil(t, importMap)
	}
}

func TestImportMapCache_MultiplePutsForSameFile(t *testing.T) {
	cache := NewImportMapCache()
	filePath := "/test/file.py"

	// First put
	importMap1 := core.NewImportMap(filePath)
	importMap1.AddImport("os", "os")
	cache.Put(filePath, importMap1)

	// Second put should replace
	importMap2 := core.NewImportMap(filePath)
	importMap2.AddImport("sys", "sys")
	cache.Put(filePath, importMap2)

	// Should get the second one
	retrieved, ok := cache.Get(filePath)
	assert.True(t, ok)
	assert.Equal(t, importMap2, retrieved)
	assert.NotEqual(t, importMap1, retrieved)
}

func TestImportMapCache_EmptyFilePath(t *testing.T) {
	cache := NewImportMapCache()

	// Empty file path should work (edge case)
	importMap := core.NewImportMap("")
	cache.Put("", importMap)

	retrieved, ok := cache.Get("")
	assert.True(t, ok)
	assert.Equal(t, importMap, retrieved)
}
