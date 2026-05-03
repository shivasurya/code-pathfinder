// Package updatecheck performs a non-blocking freshness check for the
// pathfinder binary against a CDN-hosted manifest. It surfaces upgrade
// notices and operator announcements (workshops, security advisories,
// deprecation notices) through a single entry point, [Check].
//
// The manifest is fetched at most once per invocation (CLI) or once at
// server startup (MCP). No state is ever persisted to disk.
//
// Usage:
//
//	result := updatecheck.Check(ctx, version, "cli", updatecheck.Options{
//	    Logger: logger,
//	})
//	if result != nil {
//	    // render result.Upgrade and/or result.Announcement
//	}
//
// Opt-out is supported via the PATHFINDER_NO_UPDATE_CHECK=1 environment
// variable, the Options.DisableCheck field, or the "noupdatecheck" build tag.
package updatecheck
