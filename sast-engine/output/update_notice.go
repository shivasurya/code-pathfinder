package output

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/shivasurya/code-pathfinder/sast-engine/updatecheck"
)

// Notice box geometry.
// Total visual line width = noticeBoxDashes + 2 (for ╭/╮ or ╰/╯ corner chars).
// Content line visual width = noticeContentW + 4 (│·text·│ where · is a space).
const (
	noticeBoxDashes = 64 // number of ─ chars on top/bottom border
	noticeContentW  = 62 // max visible chars per content line (between │ and │)
)

// ANSI escape codes — only written to TTY outputs.
const (
	ansiReset   = "\033[0m"
	ansiCyan    = "\033[36m"
	ansiYellow  = "\033[33m"
	ansiBoldRed = "\033[1;31m"
)

// levelColor maps an update-check level string to its ANSI color code.
func levelColor(level string) string {
	switch level {
	case "warn":
		return ansiYellow
	case "critical":
		return ansiBoldRed
	default: // "info" and anything else
		return ansiCyan
	}
}

// levelIcon maps a level string to a Unicode status icon.
func levelIcon(level string) string {
	switch level {
	case "warn":
		return "⚠"
	case "critical":
		return "✖"
	default: // "info" and anything else
		return "ℹ"
	}
}

// PrintUpdateNotice renders an upgrade banner to w.
//
// TTY layout (info, cyan border):
//
//	╭────────────────────────────────────────────────────────────────╮
//	│ ⬆  Update available: 2.0.2 → 2.1.1  (released 2026-04-10)      │
//	│    Upgrade now.                                                 │
//	│    https://github.com/shivasurya/.../releases/tag/v2.1.1       │
//	│    Disable: --no-update-check or PATHFINDER_NO_UPDATE_CHECK    │
//	╰────────────────────────────────────────────────────────────────╯
//
// warn uses a yellow border; critical uses bold red.
// Non-TTY: single plain "[update] current → latest" line.
func PrintUpdateNotice(w io.Writer, n *updatecheck.UpgradeNotice) {
	if n == nil || w == nil {
		return
	}
	printUpdateNotice(w, n, IsTTY(w))
}

// printUpdateNotice is the testable implementation that takes an explicit tty flag.
func printUpdateNotice(w io.Writer, n *updatecheck.UpgradeNotice, tty bool) {
	relDate := ""
	if !n.ReleasedAt.IsZero() {
		relDate = fmt.Sprintf("  (released %s)", n.ReleasedAt.Format("2006-01-02"))
	}
	heading := fmt.Sprintf("⬆  Update available: %s → %s%s", n.Current, n.Latest, relDate)

	var lines []string
	if n.Message != "" {
		lines = append(lines, wordWrapNotice(n.Message, noticeContentW-3)...)
	}
	if n.ReleaseURL != "" {
		lines = append(lines, n.ReleaseURL)
	}
	lines = append(lines, "Disable: --no-update-check or PATHFINDER_NO_UPDATE_CHECK")

	if tty {
		printNoticeBox(w, levelColor(n.Level), heading, lines)
		return
	}

	// Non-TTY: single plain line.
	fmt.Fprintf(w, "[update] %s → %s", n.Current, n.Latest)
	if n.Message != "" {
		fmt.Fprintf(w, "  %s", n.Message)
	}
	fmt.Fprintln(w)
}

// PrintAnnouncement renders an operator notice to w.
//
// TTY layout (level-colored border, title as headline):
//
//	╭────────────────────────────────────────────────────────────────╮
//	│ ℹ  Pathfinder Workshop — May 8                                  │
//	│    Join us for a live session on writing custom rules.         │
//	│    https://codepathfinder.dev/workshop                         │
//	╰────────────────────────────────────────────────────────────────╯
//
// Non-TTY: single plain "[notice] title: text" line.
func PrintAnnouncement(w io.Writer, a *updatecheck.Announcement) {
	if a == nil || w == nil {
		return
	}
	printAnnouncement(w, a, IsTTY(w))
}

// printAnnouncement is the testable implementation that takes an explicit tty flag.
func printAnnouncement(w io.Writer, a *updatecheck.Announcement, tty bool) {
	heading := fmt.Sprintf("%s  %s", levelIcon(a.Level), a.Title)

	var lines []string
	lines = append(lines, wordWrapNotice(a.Text, noticeContentW-3)...)
	if a.URL != "" {
		lines = append(lines, a.URL)
	}

	if tty {
		printNoticeBox(w, levelColor(a.Level), heading, lines)
		return
	}

	// Non-TTY: single plain line.
	fmt.Fprintf(w, "[notice] %s: %s", a.Title, a.Text)
	if a.URL != "" {
		fmt.Fprintf(w, "  %s", a.URL)
	}
	fmt.Fprintln(w)
}

// printNoticeBox renders the complete colored Unicode box to w.
func printNoticeBox(w io.Writer, color, heading string, lines []string) {
	horz := strings.Repeat("─", noticeBoxDashes)

	// Top border.
	fmt.Fprintf(w, "%s╭%s╮%s\n", color, horz, ansiReset)

	// Heading line (first content line, visually distinct by its icon prefix).
	printBoxLine(w, color, heading)

	// Body lines indented by 3 spaces to align text under the heading.
	for _, line := range lines {
		printBoxLine(w, color, "   "+line)
	}

	// Bottom border.
	fmt.Fprintf(w, "%s╰%s╯%s\n", color, horz, ansiReset)
}

// printBoxLine writes a single content line inside the notice box:
//
//	{color}│{reset} {text}{pad} {color}│{reset}
//
// text is truncated with … if it exceeds noticeContentW visible characters.
func printBoxLine(w io.Writer, color, text string) {
	visLen := utf8.RuneCountInString(text)
	if visLen > noticeContentW {
		runes := []rune(text)
		text = string(runes[:noticeContentW-1]) + "…"
		visLen = noticeContentW
	}
	pad := strings.Repeat(" ", noticeContentW-visLen)
	fmt.Fprintf(w, "%s│%s %s%s %s│%s\n", color, ansiReset, text, pad, color, ansiReset)
}

// wordWrapNotice splits text into lines of at most width visible characters,
// breaking only at word (whitespace) boundaries. Returns nil for empty or
// whitespace-only text. Leading and trailing whitespace is stripped before
// wrapping; internal runs of whitespace are collapsed to a single space.
func wordWrapNotice(text string, width int) []string {
	words := strings.Fields(text) // strips whitespace and splits on runs
	if len(words) == 0 {
		return nil
	}

	// Fast path: joined text fits on a single line.
	joined := strings.Join(words, " ")
	if utf8.RuneCountInString(joined) <= width {
		return []string{joined}
	}

	var lines []string
	var current strings.Builder
	currentLen := 0

	for _, word := range words {
		wordLen := utf8.RuneCountInString(word)
		switch {
		case current.Len() == 0:
			current.WriteString(word)
			currentLen = wordLen
		case currentLen+1+wordLen <= width:
			current.WriteByte(' ')
			current.WriteString(word)
			currentLen += 1 + wordLen
		default:
			lines = append(lines, current.String())
			current.Reset()
			current.WriteString(word)
			currentLen = wordLen
		}
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}
