package updatecheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Result bundles whatever is worth surfacing to the user from the manifest.
// Both fields are independently optional — Upgrade can be set without
// Announcement, Announcement can be set without Upgrade, both can be set, or
// neither (in which case Check returns nil).
type Result struct {
	Upgrade      *UpgradeNotice
	Announcement *Announcement
}

// UpgradeNotice is set when a newer version is available.
type UpgradeNotice struct {
	// Current is the running binary's version (e.g. "2.0.2").
	Current string
	// Latest is the newest published version (e.g. "2.1.1").
	Latest string
	// ReleasedAt is the release timestamp of Latest.
	ReleasedAt time.Time
	// ReleaseURL is the GitHub release page for Latest.
	ReleaseURL string
	// Level is "info" normally, "warn" when Current is below min_supported.
	Level string
	// Message is the operator-supplied short description of the new release.
	Message string
}

// Announcement is an operator message: workshop, advisory, deprecation, etc.
// Selected from the manifest's announcements array after filtering by time
// window, audience, dismissed-IDs, and (when set) version_range.
//
// VersionRange is empty for generic announcements (shown to all versions) and
// a semver constraint for version-targeted announcements (shown only when the
// running version matches). Both flow through the same selection and rendering
// path; the only difference is the filter step in selectAnnouncement.
type Announcement struct {
	// ID is a stable identifier for this announcement.
	ID string
	// Level is one of "info", "warn", or "critical".
	Level string
	// Title is a short headline (≤60 chars).
	Title string
	// Text is the body (≤240 chars, plain text).
	Text string
	// URL is an optional click-through link.
	URL string
	// Audience is "all", "cli", or "mcp".
	Audience string
	// VersionRange is a semver constraint, or "" for a generic announcement.
	VersionRange string
}

// DebugLogger is a minimal interface for receiving debug-level diagnostics.
// *output.Logger satisfies this interface, but the updatecheck package does
// not import output — callers pass in whatever logger they have.
type DebugLogger interface {
	Debug(format string, args ...any)
}

// Options configures the behavior of Check.
type Options struct {
	// DisableCheck silences the check entirely (equivalent to --no-update-check).
	DisableCheck bool
	// ManifestURL overrides the default CDN URL. Primarily for testing.
	ManifestURL string
	// HTTPTimeout caps the manifest fetch. Default: 800 ms (CLI), 5 s (MCP).
	HTTPTimeout time.Duration
	// Logger receives debug-level diagnostics when the fetch fails.
	// Pass nil to suppress diagnostics entirely.
	Logger DebugLogger
}

// Manifest is the wire-format representation of latest.json.
// It is exported for testing; production callers should use Check rather than
// constructing or fetching a Manifest directly.
type Manifest struct {
	// Schema is the manifest schema version; currently only 1 is supported.
	Schema        int                   `json:"schema"`
	Latest        ManifestLatest        `json:"latest"`
	Channels      map[string]string     `json:"channels"`
	Message       ManifestMessage       `json:"message"`
	Announcements []ManifestAnnouncement `json:"announcements"`
	// MinSupported is the oldest version that is still fully supported.
	// Running below this escalates the upgrade notice to "warn".
	MinSupported string `json:"min_supported"` //nolint:tagliatelle // CDN manifest uses snake_case
}

// ManifestLatest holds information about the latest published version.
type ManifestLatest struct {
	Version    string    `json:"version"`
	ReleasedAt time.Time `json:"released_at"` //nolint:tagliatelle // CDN manifest uses snake_case
	ReleaseURL string    `json:"release_url"` //nolint:tagliatelle // CDN manifest uses snake_case
}

// ManifestMessage is the operator-supplied short description shown alongside
// the upgrade notice.
type ManifestMessage struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

// ManifestAnnouncement is a single entry in the manifest's announcements array.
type ManifestAnnouncement struct {
	ID           string    `json:"id"`
	Level        string    `json:"level"`
	Title        string    `json:"title"`
	Text         string    `json:"text"`
	URL          string    `json:"url"`
	StartsAt     time.Time `json:"starts_at"`     //nolint:tagliatelle // CDN manifest uses snake_case
	ExpiresAt    time.Time `json:"expires_at"`    //nolint:tagliatelle // CDN manifest uses snake_case
	Audience     string    `json:"audience"`
	VersionRange string    `json:"version_range"` //nolint:tagliatelle // CDN manifest uses snake_case
}

// Fetch downloads and decodes the update manifest from url.
// It enforces timeout on both the context and the HTTP client (belt + braces).
// No disk I/O is performed at any point.
func Fetch(ctx context.Context, url string, timeout time.Duration) (*Manifest, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("updatecheck: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "pathfinder-updatecheck/1")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("updatecheck: fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("updatecheck: HTTP %d", resp.StatusCode)
	}

	var m Manifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("updatecheck: decode: %w", err)
	}
	if m.Schema != 1 {
		return nil, fmt.Errorf("updatecheck: unsupported schema %d", m.Schema)
	}
	return &m, nil
}
