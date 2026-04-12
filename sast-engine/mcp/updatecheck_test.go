package mcp

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/updatecheck"
	"github.com/stretchr/testify/assert"
)

// --- formatStatusDescription -------------------------------------------------

func makeUpgradeResult(current, latest, releaseURL, message string) *updatecheck.Result {
	return &updatecheck.Result{
		Upgrade: &updatecheck.UpgradeNotice{
			Current:    current,
			Latest:     latest,
			ReleaseURL: releaseURL,
			Message:    message,
		},
	}
}

func makeAnnouncementResult(id, level, title, text, url string) *updatecheck.Result {
	return &updatecheck.Result{
		Announcement: &updatecheck.Announcement{
			ID:    id,
			Level: level,
			Title: title,
			Text:  text,
			URL:   url,
		},
	}
}

func TestFormatStatusDescription_Nil_NoOp(t *testing.T) {
	base := "some base description"
	got := formatStatusDescription(base, nil)
	assert.Equal(t, base, got)
}

func TestFormatStatusDescription_NeitherPresent(t *testing.T) {
	base := "some base description"
	got := formatStatusDescription(base, &updatecheck.Result{})
	assert.Equal(t, base, got)
}

func TestFormatStatusDescription_UpgradeOnly(t *testing.T) {
	base := "base desc"
	r := makeUpgradeResult("1.0.0", "2.0.0", "https://example.com/releases/v2.0.0", "")
	got := formatStatusDescription(base, r)

	assert.Contains(t, got, "Upgrade available")
	assert.Contains(t, got, "1.0.0")
	assert.Contains(t, got, "2.0.0")
	assert.Contains(t, got, "https://example.com/releases/v2.0.0")
	// Base is appended after a blank line.
	assert.Contains(t, got, "\n\n"+base)
}

func TestFormatStatusDescription_AnnouncementOnly(t *testing.T) {
	base := "base desc"
	r := makeAnnouncementResult("ann-1", "info", "Q2 Release", "New features landed.", "https://example.com/blog/q2")
	got := formatStatusDescription(base, r)

	assert.Contains(t, got, "ℹ")
	assert.Contains(t, got, "Q2 Release")
	assert.Contains(t, got, "New features landed.")
	assert.Contains(t, got, "https://example.com/blog/q2")
	assert.Contains(t, got, "\n\n"+base)
}

func TestFormatStatusDescription_BothPresent_UpgradeWins(t *testing.T) {
	base := "base desc"
	r := makeUpgradeResult("1.0.0", "2.0.0", "https://example.com/releases/v2.0.0", "")
	r.Announcement = &updatecheck.Announcement{
		Title: "Should not appear in prefix",
		Text:  "announcement text",
	}
	got := formatStatusDescription(base, r)

	// Upgrade prefix wins.
	assert.Contains(t, got, "Upgrade available")
	assert.Contains(t, got, "2.0.0")
	// Announcement title must NOT appear as the leading prefix (upgrade wins).
	lines := got[:len(got)-len("\n\n"+base)]
	assert.NotContains(t, lines, "Should not appear in prefix")
}

func TestFormatStatusDescription_TruncatedAt200(t *testing.T) {
	// Build a release URL long enough to push the formatted string past 200 bytes.
	longURL := "https://example.com/releases/" + string(make([]byte, 200))
	for i := range longURL {
		_ = i // just to silence the linter; longURL is deliberately long
	}
	longURL = "https://example.com/releases/v9.9.9/" + repeatStr("x", 180)
	r := makeUpgradeResult("1.0.0", "9.9.9", longURL, "")
	got := formatStatusDescription("base", r)

	// The prefix portion (everything before "\n\n") must be ≤ 200 bytes.
	parts := splitOnce(got, "\n\n")
	assert.LessOrEqual(t, len(parts[0]), 200)
	// Must end with the ellipsis rune (UTF-8: 3 bytes).
	assert.True(t, len(parts[0]) > 0)
	last := []rune(parts[0])
	assert.Equal(t, '…', last[len(last)-1])
}

// --- TruncateNotice ----------------------------------------------------------

func TestTruncateNotice_NoOpWhenShort(t *testing.T) {
	s := "short string"
	assert.Equal(t, s, TruncateNotice(s, 200))
}

func TestTruncateNotice_NoOpExactLength(t *testing.T) {
	s := repeatStr("a", 200)
	assert.Equal(t, s, TruncateNotice(s, 200))
}

func TestTruncateNotice_AppendsEllipsis(t *testing.T) {
	s := repeatStr("a", 201)
	got := TruncateNotice(s, 200)
	assert.LessOrEqual(t, len(got), 200)
	assert.True(t, len(got) > 0)
	last := []rune(got)
	assert.Equal(t, '…', last[len(last)-1])
}

// --- helpers -----------------------------------------------------------------

func repeatStr(s string, n int) string {
	out := make([]byte, n*len(s))
	for i := range n {
		copy(out[i*len(s):], s)
	}
	return string(out)
}

// splitOnce splits s on the first occurrence of sep into at most 2 parts.
func splitOnce(s, sep string) []string {
	idx := indexOf(s, sep)
	if idx < 0 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+len(sep):]}
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
