//go:build !noupdatecheck

package updatecheck

import (
	"context"
	"os"
	"time"
)

// DefaultManifestURL is the CDN URL for the latest.json manifest.
const DefaultManifestURL = "https://assets.codepathfinder.dev/pathfinder/latest.json"

// defaultManifestURL is the resolved URL used when Options.ManifestURL is
// empty. It is a package-level variable (not the constant) so tests can
// substitute a local httptest server without touching the live CDN.
var defaultManifestURL = DefaultManifestURL

// envOptOut reports whether the PATHFINDER_NO_UPDATE_CHECK environment
// variable requests that the update check be silenced.
func envOptOut() bool {
	v := os.Getenv("PATHFINDER_NO_UPDATE_CHECK")
	return v == "1" || v == "true"
}

// Check performs a non-blocking freshness check against the CDN manifest and
// returns whatever is worth surfacing to the user.
//
// Returns nil when:
//   - opts.DisableCheck is true, or PATHFINDER_NO_UPDATE_CHECK is set.
//   - The manifest fetch times out or returns an error (silent fallback).
//   - The running binary is already up to date and no announcements qualify.
//
// audience should be "cli" for the pathfinder binary or "mcp" for the MCP
// server; it controls which announcements are eligible.
func Check(ctx context.Context, current, audience string, opts Options) *Result {
	if opts.DisableCheck || envOptOut() {
		return nil
	}
	if opts.ManifestURL == "" {
		opts.ManifestURL = defaultManifestURL
	}
	if opts.HTTPTimeout == 0 {
		opts.HTTPTimeout = 800 * time.Millisecond
	}

	m, err := Fetch(ctx, opts.ManifestURL, opts.HTTPTimeout)
	if err != nil {
		if opts.Logger != nil {
			opts.Logger.Debug("updatecheck fetch failed: %v", err)
		}
		return nil
	}

	result := &Result{
		Upgrade:      selectUpgrade(m, current),
		Announcement: selectAnnouncement(m, current, audience, time.Now(), nil),
	}
	if result.Upgrade == nil && result.Announcement == nil {
		return nil
	}
	return result
}
