package mcp

import (
	"fmt"

	"github.com/shivasurya/code-pathfinder/sast-engine/analytics"
	"github.com/shivasurya/code-pathfinder/sast-engine/updatecheck"
)

// noticeMaxLen is the maximum byte length of the upgrade/announcement prefix
// prepended to the status tool description. MCP clients truncate long
// descriptions aggressively; 200 bytes keeps the hint well within limits.
const noticeMaxLen = 200

// formatStatusDescription prepends a one-line upgrade or announcement hint to
// the existing status tool description. The hint is capped at 200 visible
// characters (including the trailing URL) to stay within MCP client truncation
// limits.
//
// When both an upgrade and an announcement are present the upgrade wins —
// it is the higher-leverage nudge. The announcement still appears in
// serverInfo.metadata and in the structured status tool result.
//
// Returns base unchanged when r is nil or neither Upgrade nor Announcement is
// set.
func formatStatusDescription(base string, r *updatecheck.Result) string {
	if r == nil {
		return base
	}

	var prefix string
	switch {
	case r.Upgrade != nil:
		prefix = fmt.Sprintf("⚠ Upgrade available: pathfinder %s → %s. %s",
			r.Upgrade.Current, r.Upgrade.Latest, r.Upgrade.ReleaseURL)
	case r.Announcement != nil:
		prefix = fmt.Sprintf("ℹ %s — %s %s",
			r.Announcement.Title, r.Announcement.Text, r.Announcement.URL)
	default:
		return base
	}

	return TruncateNotice(prefix, noticeMaxLen) + "\n\n" + base
}

// TruncateNotice caps s at n bytes, appending "…" (a single UTF-8 ellipsis
// character, 3 bytes) when truncation is needed. The total result is always
// ≤ n bytes. Exported for testing.
func TruncateNotice(s string, n int) string {
	if len(s) <= n {
		return s
	}
	const ellipsis = "…" // 3 UTF-8 bytes
	if n <= 3 {
		return ellipsis
	}
	return s[:n-3] + ellipsis
}

// reportReachIfNeeded fires analytics reach events for upgrade notices and
// announcements at most once per 24-hour window per process. It is a no-op
// when analytics are disabled, updateInfo is nil, or the dedup window has
// not yet expired for the given key.
//
// Called from handleToolsList after description injection — fired every
// tools/list, but the ReachReporter deduplicates to at most one event per day.
func (s *Server) reportReachIfNeeded() {
	if analytics.IsDisabled() {
		return
	}
	r := s.updateInfo
	if r == nil || s.reachReporter == nil {
		return
	}
	if r.Upgrade != nil && s.reachReporter.ShouldReport("upgrade:"+r.Upgrade.Latest) {
		analytics.ReportEventWithProperties("update_notice_shown", map[string]any{
			"current":                 r.Upgrade.Current,
			"latest":                  r.Upgrade.Latest,
			"level":                   r.Upgrade.Level,
			"surface":                 "mcp",
			"triggered_min_supported": r.Upgrade.Level == "warn",
		})
	}
	if r.Announcement != nil && s.reachReporter.ShouldReport("announcement:"+r.Announcement.ID) {
		kind := "generic"
		if r.Announcement.VersionRange != "" {
			kind = "version_targeted"
		}
		props := map[string]any{
			"id":      r.Announcement.ID,
			"level":   r.Announcement.Level,
			"kind":    kind,
			"surface": "mcp",
			"current": s.version,
		}
		if kind == "version_targeted" {
			props["version_range"] = r.Announcement.VersionRange
		}
		analytics.ReportEventWithProperties("announcement_shown", props)
	}
}
