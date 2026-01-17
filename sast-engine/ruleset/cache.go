package ruleset

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Cache manages local ruleset cache.
type Cache struct {
	dir string
}

// NewCache creates a new cache instance.
func NewCache(cacheDir string) (*Cache, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %w", err)
	}
	return &Cache{dir: cacheDir}, nil
}

// Get retrieves cached ruleset if valid.
func (c *Cache) Get(spec *RulesetSpec, expectedChecksum string) (string, error) {
	entry, err := c.loadEntry(spec)
	if err != nil {
		return "", err // Cache miss
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		return "", fmt.Errorf("cache expired")
	}

	// Verify checksum matches
	if entry.Checksum != expectedChecksum {
		return "", fmt.Errorf("checksum mismatch")
	}

	// Verify directory exists
	if _, err := os.Stat(entry.Path); os.IsNotExist(err) {
		return "", fmt.Errorf("cached path missing")
	}

	return entry.Path, nil
}

// Set stores a ruleset in cache.
func (c *Cache) Set(spec *RulesetSpec, extractedPath, checksum string, ttl time.Duration) error {
	entry := &CacheEntry{
		Spec:      *spec,
		Path:      extractedPath,
		Checksum:  checksum,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	return c.saveEntry(entry)
}

// Invalidate removes a cached ruleset.
func (c *Cache) Invalidate(spec *RulesetSpec) error {
	entryPath := c.entryPath(spec)
	if err := os.Remove(entryPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove extracted directory
	extractedPath := c.extractedPath(spec)
	return os.RemoveAll(extractedPath)
}

// Helper methods.
func (c *Cache) entryPath(spec *RulesetSpec) string {
	return filepath.Join(c.dir, spec.Category, fmt.Sprintf("%s.json", spec.Bundle))
}

func (c *Cache) extractedPath(spec *RulesetSpec) string {
	return filepath.Join(c.dir, spec.Category, spec.Bundle)
}

func (c *Cache) loadEntry(spec *RulesetSpec) (*CacheEntry, error) {
	path := c.entryPath(spec)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (c *Cache) saveEntry(entry *CacheEntry) error {
	path := c.entryPath(&entry.Spec)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// VerifyChecksum calculates checksum of a file.
func VerifyChecksum(filePath, expectedChecksum string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actual := fmt.Sprintf("%x", h.Sum(nil))
	if actual != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actual)
	}
	return nil
}
