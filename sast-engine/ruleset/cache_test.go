package ruleset

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")

	cache, err := NewCache(cacheDir)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	if cache.dir != cacheDir {
		t.Errorf("expected cache dir %s, got %s", cacheDir, cache.dir)
	}

	// Verify directory was created
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Errorf("cache directory was not created")
	}
}

func TestCacheGetSet(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := NewCache(tempDir)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	spec := &RulesetSpec{
		Category: "docker",
		Bundle:   "security",
	}

	extractedPath := filepath.Join(tempDir, "extracted")
	checksum := "abc123"
	ttl := 1 * time.Hour

	// Set cache entry
	err = cache.Set(spec, extractedPath, checksum, ttl)
	if err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}

	// Create the extracted directory so Get doesn't fail
	if err := os.MkdirAll(extractedPath, 0755); err != nil {
		t.Fatalf("failed to create extracted dir: %v", err)
	}

	// Get cache entry
	cachedPath, err := cache.Get(spec, checksum)
	if err != nil {
		t.Fatalf("failed to get cache: %v", err)
	}

	if cachedPath != extractedPath {
		t.Errorf("expected path %s, got %s", extractedPath, cachedPath)
	}
}

func TestCacheGetMiss(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := NewCache(tempDir)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	spec := &RulesetSpec{
		Category: "docker",
		Bundle:   "security",
	}

	// Try to get non-existent entry
	_, err = cache.Get(spec, "abc123")
	if err == nil {
		t.Errorf("expected error for cache miss, got nil")
	}
}

func TestCacheExpiration(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := NewCache(tempDir)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	spec := &RulesetSpec{
		Category: "docker",
		Bundle:   "security",
	}

	extractedPath := filepath.Join(tempDir, "extracted")
	checksum := "abc123"
	ttl := 1 * time.Millisecond // Very short TTL

	// Set cache entry
	err = cache.Set(spec, extractedPath, checksum, ttl)
	if err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}

	// Create the extracted directory
	if err := os.MkdirAll(extractedPath, 0755); err != nil {
		t.Fatalf("failed to create extracted dir: %v", err)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Try to get expired entry
	_, err = cache.Get(spec, checksum)
	if err == nil {
		t.Errorf("expected error for expired cache, got nil")
	}
}

func TestCacheChecksumMismatch(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := NewCache(tempDir)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	spec := &RulesetSpec{
		Category: "docker",
		Bundle:   "security",
	}

	extractedPath := filepath.Join(tempDir, "extracted")
	checksum := "abc123"
	ttl := 1 * time.Hour

	// Set cache entry
	err = cache.Set(spec, extractedPath, checksum, ttl)
	if err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}

	// Create the extracted directory
	if err := os.MkdirAll(extractedPath, 0755); err != nil {
		t.Fatalf("failed to create extracted dir: %v", err)
	}

	// Try to get with different checksum
	_, err = cache.Get(spec, "different123")
	if err == nil {
		t.Errorf("expected error for checksum mismatch, got nil")
	}
}

func TestCacheInvalidate(t *testing.T) {
	tempDir := t.TempDir()
	cache, err := NewCache(tempDir)
	if err != nil {
		t.Fatalf("failed to create cache: %v", err)
	}

	spec := &RulesetSpec{
		Category: "docker",
		Bundle:   "security",
	}

	// Use the path that cache expects
	extractedPath := filepath.Join(tempDir, "docker", "security")
	checksum := "abc123"
	ttl := 1 * time.Hour

	// Set cache entry
	err = cache.Set(spec, extractedPath, checksum, ttl)
	if err != nil {
		t.Fatalf("failed to set cache: %v", err)
	}

	// Create the extracted directory
	if err := os.MkdirAll(extractedPath, 0755); err != nil {
		t.Fatalf("failed to create extracted dir: %v", err)
	}

	// Invalidate cache
	err = cache.Invalidate(spec)
	if err != nil {
		t.Fatalf("failed to invalidate cache: %v", err)
	}

	// Verify entry is gone
	_, err = cache.Get(spec, checksum)
	if err == nil {
		t.Errorf("expected error after invalidation, got nil")
	}

	// Verify directory is removed
	if _, err := os.Stat(extractedPath); err == nil {
		t.Errorf("extracted directory should be removed")
	}
}

func TestVerifyChecksum(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create test file
	content := []byte("test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Calculate expected checksum (sha256 of "test content")
	expectedChecksum := "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"

	// Verify correct checksum
	err := VerifyChecksum(testFile, expectedChecksum)
	if err != nil {
		t.Errorf("checksum verification failed: %v", err)
	}

	// Verify incorrect checksum
	err = VerifyChecksum(testFile, "wrongchecksum")
	if err == nil {
		t.Errorf("expected error for incorrect checksum, got nil")
	}

	// Verify non-existent file
	err = VerifyChecksum(filepath.Join(tempDir, "nonexistent.txt"), expectedChecksum)
	if err == nil {
		t.Errorf("expected error for non-existent file, got nil")
	}
}
