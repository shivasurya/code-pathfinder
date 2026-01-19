package ruleset

import "time"

// RulesetSpec represents a parsed ruleset specification.
type RulesetSpec struct {
	Category string // "docker"
	Bundle   string // "security"
}

// RuleSpec represents a parsed individual rule specification.
type RuleSpec struct {
	Language string // "docker"
	RuleID   string // "DOCKER-BP-007"
}

// Manifest represents the global or category manifest.
//
//nolint:tagliatelle // JSON uses snake_case to match external manifest format from R2
type Manifest struct {
	Version        string             `json:"version,omitempty"`
	Categories     []string           `json:"categories,omitempty"`
	Category       string             `json:"category,omitempty"`
	Language       string             `json:"language,omitempty"`
	Description    string             `json:"description,omitempty"`
	Bundles        map[string]*Bundle `json:"bundles"`
	BaseURL        string             `json:"base_url,omitempty"`
	CategoriesInfo []CategoryInfo     `json:"categories_info,omitempty"`
}

// CategoryInfo represents category metadata in global manifest.
//
//nolint:tagliatelle // JSON uses snake_case to match external manifest format from R2
type CategoryInfo struct {
	Category    string `json:"category"`
	BundleCount int    `json:"bundle_count"`
	ManifestURL string `json:"manifest_url"`
}

// Bundle represents metadata for a rule bundle.
//
//nolint:tagliatelle // JSON uses snake_case to match external manifest format from R2
type Bundle struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	SeverityFilter []string `json:"severity_filter"`
	Recommended    bool     `json:"recommended"`
	Tags           []string `json:"tags"`
	// Computed fields (from PR-02).
	FileCount   int    `json:"file_count,omitempty"`
	ZipSize     int64  `json:"zip_size,omitempty"`
	Checksum    string `json:"checksum,omitempty"`
	DownloadURL string `json:"download_url,omitempty"`
}

// CacheEntry tracks cached rulesets.
//
//nolint:tagliatelle // JSON uses snake_case for consistency with manifest format
type CacheEntry struct {
	Spec      RulesetSpec `json:"spec"`
	Path      string      `json:"path"`
	Checksum  string      `json:"checksum"`
	CachedAt  time.Time   `json:"cached_at"`
	ExpiresAt time.Time   `json:"expires_at"`
}

// DownloadConfig configures the downloader.
type DownloadConfig struct {
	BaseURL       string
	CacheDir      string
	CacheTTL      time.Duration
	ManifestTTL   time.Duration
	HTTPTimeout   time.Duration
	RetryAttempts int
}

// ManifestProvider defines the interface for loading manifests.
// This interface enables testing with mock implementations.
type ManifestProvider interface {
	LoadCategoryManifest(category string) (*Manifest, error)
}
