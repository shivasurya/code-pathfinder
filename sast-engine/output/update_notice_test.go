package output

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/updatecheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ----------------------------------------------------------------

func upgradeNotice(level string) *updatecheck.UpgradeNotice {
	return &updatecheck.UpgradeNotice{
		Current:    "2.0.2",
		Latest:     "2.1.1",
		ReleasedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
		ReleaseURL: "https://github.com/shivasurya/code-pathfinder/releases/tag/v2.1.1",
		Level:      level,
		Message:    "v2.1.1 ships Go third-party type resolution.",
	}
}

func announcement(level string) *updatecheck.Announcement {
	return &updatecheck.Announcement{
		ID:    "ann1",
		Level: level,
		Title: "Pathfinder Workshop — May 8",
		Text:  "Join us for a live session on writing custom rules.",
		URL:   "https://codepathfinder.dev/workshop",
	}
}

// --- levelColor -------------------------------------------------------------

func TestLevelColor(t *testing.T) {
	assert.Equal(t, ansiCyan, levelColor("info"))
	assert.Equal(t, ansiYellow, levelColor("warn"))
	assert.Equal(t, ansiBoldRed, levelColor("critical"))
	assert.Equal(t, ansiCyan, levelColor(""))
	assert.Equal(t, ansiCyan, levelColor("unknown"))
}

// --- levelIcon --------------------------------------------------------------

func TestLevelIcon(t *testing.T) {
	assert.Equal(t, "ℹ", levelIcon("info"))
	assert.Equal(t, "⚠", levelIcon("warn"))
	assert.Equal(t, "✖", levelIcon("critical"))
	assert.Equal(t, "ℹ", levelIcon(""))
	assert.Equal(t, "ℹ", levelIcon("unknown"))
}

// --- wordWrapNotice ---------------------------------------------------------

func TestWordWrapNotice_EmptyString(t *testing.T) {
	assert.Nil(t, wordWrapNotice("", 40))
}

func TestWordWrapNotice_ShortFitsOnOneLine(t *testing.T) {
	got := wordWrapNotice("Hello world.", 40)
	require.Len(t, got, 1)
	assert.Equal(t, "Hello world.", got[0])
}

func TestWordWrapNotice_ExactWidth(t *testing.T) {
	// 10-char string, width=10 → single line (<=, not <)
	got := wordWrapNotice("1234567890", 10)
	require.Len(t, got, 1)
	assert.Equal(t, "1234567890", got[0])
}

func TestWordWrapNotice_WrapsAtWordBoundary(t *testing.T) {
	// "hello world" (11 chars), width=8 → ["hello", "world"]
	got := wordWrapNotice("hello world", 8)
	require.Len(t, got, 2)
	assert.Equal(t, "hello", got[0])
	assert.Equal(t, "world", got[1])
}

func TestWordWrapNotice_MultipleWraps(t *testing.T) {
	text := "one two three four five"
	got := wordWrapNotice(text, 10)
	// "one two" (7) fits, "three" (5) would make 13 → wrap
	// "three four" (10) fits, "five" would make 15 → wrap
	require.True(t, len(got) >= 2)
	// Verify no line exceeds width
	for _, line := range got {
		assert.LessOrEqual(t, len(line), 10+5, // allow slight over for single long words
			"line %q exceeds wrap width", line)
	}
}

func TestWordWrapNotice_AllWhitespace(t *testing.T) {
	assert.Nil(t, wordWrapNotice("   ", 40))
}

func TestWordWrapNotice_SingleLongWord(t *testing.T) {
	// A single word longer than width cannot be wrapped; returned as-is.
	long := strings.Repeat("x", 80)
	got := wordWrapNotice(long, 40)
	require.Len(t, got, 1)
	assert.Equal(t, long, got[0])
}

// --- printBoxLine -----------------------------------------------------------

func TestPrintBoxLine_ShortText(t *testing.T) {
	var buf bytes.Buffer
	printBoxLine(&buf, ansiCyan, "hello")
	line := buf.String()
	assert.Contains(t, line, "hello")
	assert.Contains(t, line, "│")
}

func TestPrintBoxLine_ExactWidth(t *testing.T) {
	text := strings.Repeat("x", noticeContentW)
	var buf bytes.Buffer
	printBoxLine(&buf, ansiCyan, text)
	line := buf.String()
	assert.Contains(t, line, text)
	assert.NotContains(t, line, "…")
}

func TestPrintBoxLine_TruncatesLongText(t *testing.T) {
	text := strings.Repeat("x", noticeContentW+10)
	var buf bytes.Buffer
	printBoxLine(&buf, ansiCyan, text)
	line := buf.String()
	assert.Contains(t, line, "…")
	// Ensure the line length is bounded (ANSI codes don't count toward visual width)
	// Just verify truncation happened
	assert.NotContains(t, line, text) // truncated, so original long string not present
}

func TestPrintBoxLine_UnicodeText(t *testing.T) {
	// Unicode characters that are 1 visual column wide (multi-byte in UTF-8)
	text := "⬆  Update available: 2.0.2 → 2.1.1"
	var buf bytes.Buffer
	printBoxLine(&buf, ansiCyan, text)
	line := buf.String()
	assert.Contains(t, line, "⬆")
	assert.Contains(t, line, "→")
}

// --- printUpdateNotice (internal, explicit isTTY) ---------------------------

func TestPrintUpdateNotice_TTY_Info(t *testing.T) {
	var buf bytes.Buffer
	n := upgradeNotice("info")
	printUpdateNotice(&buf, n, true)

	out := buf.String()
	assert.Contains(t, out, "╭")
	assert.Contains(t, out, "╰")
	assert.Contains(t, out, "⬆")
	assert.Contains(t, out, "2.0.2 → 2.1.1")
	assert.Contains(t, out, "2026-04-10")
	assert.Contains(t, out, n.Message)
	// The URL is 68 chars; with the 3-char body indent it exceeds noticeContentW
	// (62) and gets truncated. Check for the longest prefix that fits.
	assert.Contains(t, out, "https://github.com/shivasurya/code-pathfinder/releases/tag")
	assert.Contains(t, out, "--no-update-check")
	assert.Contains(t, out, ansiCyan)
	assert.NotContains(t, out, "[update]") // TTY mode, not plain
}

func TestPrintUpdateNotice_TTY_Warn(t *testing.T) {
	var buf bytes.Buffer
	printUpdateNotice(&buf, upgradeNotice("warn"), true)

	out := buf.String()
	assert.Contains(t, out, ansiYellow)
	assert.Contains(t, out, "╭")
}

func TestPrintUpdateNotice_TTY_NoReleaseDate(t *testing.T) {
	var buf bytes.Buffer
	n := upgradeNotice("info")
	n.ReleasedAt = time.Time{} // zero = no date
	printUpdateNotice(&buf, n, true)

	out := buf.String()
	assert.NotContains(t, out, "released")
	assert.Contains(t, out, "2.0.2 → 2.1.1")
}

func TestPrintUpdateNotice_TTY_NoMessage(t *testing.T) {
	var buf bytes.Buffer
	n := upgradeNotice("info")
	n.Message = ""
	printUpdateNotice(&buf, n, true)

	out := buf.String()
	assert.Contains(t, out, "╭")
	// Still has disable hint even without message
	assert.Contains(t, out, "--no-update-check")
}

func TestPrintUpdateNotice_TTY_NoReleaseURL(t *testing.T) {
	var buf bytes.Buffer
	n := upgradeNotice("info")
	n.ReleaseURL = ""
	printUpdateNotice(&buf, n, true)

	out := buf.String()
	assert.Contains(t, out, "╭")
	assert.NotContains(t, out, "https://github.com")
}

func TestPrintUpdateNotice_TTY_LongMessageWraps(t *testing.T) {
	var buf bytes.Buffer
	n := upgradeNotice("info")
	n.Message = strings.Repeat("word ", 20) // very long message → must wrap
	printUpdateNotice(&buf, n, true)

	out := buf.String()
	lines := strings.Split(out, "\n")
	// Should have more lines than a single-line message would produce
	assert.Greater(t, len(lines), 5, "long message should produce multiple lines")
}

func TestPrintUpdateNotice_NonTTY_Info(t *testing.T) {
	var buf bytes.Buffer
	printUpdateNotice(&buf, upgradeNotice("info"), false)

	out := buf.String()
	assert.Contains(t, out, "[update]")
	assert.Contains(t, out, "2.0.2 → 2.1.1")
	assert.Contains(t, out, "v2.1.1 ships")
	assert.NotContains(t, out, "\033[") // no ANSI codes
	assert.NotContains(t, out, "╭")    // no box
}

func TestPrintUpdateNotice_NonTTY_EmptyMessage(t *testing.T) {
	var buf bytes.Buffer
	n := upgradeNotice("info")
	n.Message = ""
	printUpdateNotice(&buf, n, false)

	out := buf.String()
	assert.Contains(t, out, "[update]")
	// No trailing double-space from empty message
	assert.NotContains(t, out, "  \n")
}

// --- PrintUpdateNotice (exported, nil guards) --------------------------------

func TestPrintUpdateNotice_NilNotice(t *testing.T) {
	var buf bytes.Buffer
	PrintUpdateNotice(&buf, nil) // must not panic, must produce no output
	assert.Empty(t, buf.String())
}

func TestPrintUpdateNotice_NilWriter(t *testing.T) {
	// Must not panic when w is nil.
	PrintUpdateNotice(nil, upgradeNotice("info"))
}

// --- printAnnouncement (internal, explicit isTTY) ---------------------------

func TestPrintAnnouncement_TTY_Info(t *testing.T) {
	var buf bytes.Buffer
	printAnnouncement(&buf, announcement("info"), true)

	out := buf.String()
	assert.Contains(t, out, "╭")
	assert.Contains(t, out, "ℹ")
	assert.Contains(t, out, "Pathfinder Workshop")
	assert.Contains(t, out, "Join us")
	assert.Contains(t, out, "https://codepathfinder.dev/workshop")
	assert.Contains(t, out, ansiCyan)
}

func TestPrintAnnouncement_TTY_Warn(t *testing.T) {
	var buf bytes.Buffer
	printAnnouncement(&buf, announcement("warn"), true)

	out := buf.String()
	assert.Contains(t, out, ansiYellow)
	assert.Contains(t, out, "⚠")
}

func TestPrintAnnouncement_TTY_Critical(t *testing.T) {
	var buf bytes.Buffer
	printAnnouncement(&buf, announcement("critical"), true)

	out := buf.String()
	assert.Contains(t, out, ansiBoldRed)
	assert.Contains(t, out, "✖")
}

func TestPrintAnnouncement_TTY_NoURL(t *testing.T) {
	var buf bytes.Buffer
	a := announcement("info")
	a.URL = ""
	printAnnouncement(&buf, a, true)

	out := buf.String()
	assert.Contains(t, out, "╭")
	assert.NotContains(t, out, "https://")
}

func TestPrintAnnouncement_TTY_LongTextWraps(t *testing.T) {
	var buf bytes.Buffer
	a := announcement("warn")
	a.Text = strings.Repeat("important notice ", 10) // long text → must wrap
	printAnnouncement(&buf, a, true)

	out := buf.String()
	lines := strings.Split(out, "\n")
	assert.Greater(t, len(lines), 5)
}

func TestPrintAnnouncement_NonTTY_WithURL(t *testing.T) {
	var buf bytes.Buffer
	printAnnouncement(&buf, announcement("info"), false)

	out := buf.String()
	assert.Contains(t, out, "[notice]")
	assert.Contains(t, out, "Pathfinder Workshop")
	assert.Contains(t, out, "Join us")
	assert.Contains(t, out, "https://codepathfinder.dev/workshop")
	assert.NotContains(t, out, "\033[")
}

func TestPrintAnnouncement_NonTTY_NoURL(t *testing.T) {
	var buf bytes.Buffer
	a := announcement("info")
	a.URL = ""
	printAnnouncement(&buf, a, false)

	out := buf.String()
	assert.Contains(t, out, "[notice]")
	assert.NotContains(t, out, "https://")
}

// --- PrintAnnouncement (exported, nil guards) --------------------------------

func TestPrintAnnouncement_NilAnnouncement(t *testing.T) {
	var buf bytes.Buffer
	PrintAnnouncement(&buf, nil)
	assert.Empty(t, buf.String())
}

func TestPrintAnnouncement_NilWriter(t *testing.T) {
	PrintAnnouncement(nil, announcement("info")) // must not panic
}
