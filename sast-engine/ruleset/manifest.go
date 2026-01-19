package ruleset

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"
)

// ManifestLoader handles manifest fetching and caching.
type ManifestLoader struct {
	baseURL    string
	httpClient *http.Client
	cacheDir   string
}

// NewManifestLoader creates a new loader.
func NewManifestLoader(baseURL, cacheDir string) *ManifestLoader {
	return &ManifestLoader{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cacheDir:   cacheDir,
	}
}

// LoadCategoryManifest loads manifest for a category.
func (m *ManifestLoader) LoadCategoryManifest(category string) (*Manifest, error) {
	url := fmt.Sprintf("%s/%s/manifest.json", m.baseURL, category)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add ETag support for cache validation
	// TODO: Implement ETag caching

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest fetch failed: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// GetBundle retrieves bundle metadata from manifest.
func (m *Manifest) GetBundle(bundleName string) (*Bundle, error) {
	bundle, exists := m.Bundles[bundleName]
	if !exists {
		return nil, fmt.Errorf("bundle not found: %s", bundleName)
	}
	return bundle, nil
}

// GetAllBundleNames returns a sorted list of all bundle names in this category.
// Used for expanding "category/all" specs to all available bundles.
func (m *Manifest) GetAllBundleNames() []string {
	names := make([]string, 0, len(m.Bundles))
	for name := range m.Bundles {
		names = append(names, name)
	}
	// Sort for consistent ordering across runs
	sort.Strings(names)
	return names
}
