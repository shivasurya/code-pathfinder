package graph

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// ProjectStats summarises the file population that a single getFiles walk
// observed. It is exposed on CodeGraph so callers (cmd/scan, cmd/ci) can
// render a meaningful empty-state message when the graph ended up with zero
// parseable nodes, instead of exiting with a generic "no source files" error.
//
// Counts only include files that survived the always-skipped directories
// (vendor, node_modules, .git, etc.) and the user's --exclude prefixes; the
// goal is files the user expected pathfinder to look at, not every regular
// file on disk.
type ProjectStats struct {
	// TotalFiles is the count of every regular file walked, after applying
	// the skip-directory list and --exclude patterns.
	TotalFiles int
	// ScannedFiles is the count of files routed to one of the supported
	// tree-sitter parsers. Equals len(files) returned from getFiles.
	ScannedFiles int
	// ByLanguage is a language-display-name → file-count map covering every
	// file walked, both supported and unsupported. Files in formats
	// pathfinder cannot classify (binaries, lock files, dotfiles) are
	// omitted entirely rather than bucketed as "Other"; this keeps the
	// downstream "Detected: …" line honest about what's actually source.
	ByLanguage map[string]int
}

// supportedLanguageNames lists the display names of the languages this build
// of pathfinder can analyse, in stable presentation order. The values must
// match the keys produced by languageOf for supported files so the
// "Detected:" / "Supported:" split renders consistently.
var supportedLanguageNames = []string{
	"Java", "Python", "Go", "C/C++", "Dockerfile", "docker-compose",
}

// SupportedLanguages returns a copy of the supported-language display names
// for use in user-facing messages.
func SupportedLanguages() []string {
	out := make([]string, len(supportedLanguageNames))
	copy(out, supportedLanguageNames)
	return out
}

// recordFile increments TotalFiles, and ScannedFiles when supported is true,
// and bumps the per-language count if the file maps to a known language.
func (s *ProjectStats) recordFile(path string, supported bool) {
	if s.ByLanguage == nil {
		s.ByLanguage = make(map[string]int)
	}
	s.TotalFiles++
	if supported {
		s.ScannedFiles++
	}
	if lang := languageOf(path); lang != "" {
		s.ByLanguage[lang]++
	}
}

// languageOf returns the human-readable language name for path, or "" if the
// file should not be counted at all (binary blobs, lock files, anything we
// have no opinion about). Both supported and unsupported languages return a
// non-empty name so the "Detected:" line can show every language present.
func languageOf(path string) string {
	base := strings.ToLower(filepath.Base(path))
	ext := filepath.Ext(base)

	if strings.HasPrefix(base, "dockerfile") {
		return "Dockerfile"
	}
	if strings.Contains(base, "docker-compose") && (ext == ".yml" || ext == ".yaml") {
		return "docker-compose"
	}

	switch ext {
	case ".java":
		return "Java"
	case ".py", ".pyi":
		return "Python"
	case ".go":
		return "Go"
	case ".c", ".h":
		return "C/C++"
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx":
		return "C/C++"
	case ".ts", ".tsx":
		return "TypeScript"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "JavaScript"
	case ".rb":
		return "Ruby"
	case ".php":
		return "PHP"
	case ".rs":
		return "Rust"
	case ".kt", ".kts":
		return "Kotlin"
	case ".swift":
		return "Swift"
	case ".scala":
		return "Scala"
	case ".cs":
		return "C#"
	case ".sh", ".bash", ".zsh":
		return "Shell"
	case ".yml", ".yaml":
		return "YAML"
	case ".json":
		return "JSON"
	case ".xml":
		return "XML"
	case ".toml":
		return "TOML"
	case ".md", ".markdown":
		return "Markdown"
	case ".html", ".htm":
		return "HTML"
	case ".css", ".scss", ".sass", ".less":
		return "CSS"
	case ".sql":
		return "SQL"
	case ".proto":
		return "Protobuf"
	case ".tf":
		return "Terraform"
	}
	return ""
}

// UnsupportedFileCount returns the total number of detected files whose
// language is not in the supported set. Used for the headline number in the
// "Scanned 0 of N files (M unsupported)" line.
func (s ProjectStats) UnsupportedFileCount() int {
	if len(s.ByLanguage) == 0 {
		return 0
	}
	supported := supportedSet()
	var n int
	for name, count := range s.ByLanguage {
		if _, ok := supported[name]; !ok {
			n += count
		}
	}
	return n
}

// UnsupportedSummary returns the top-N unsupported languages formatted as
// "TypeScript (32), JavaScript (8), JSON (5), Markdown (2)". Returns "" if
// no unsupported files were detected. A topN of 0 or less means unbounded.
// Ties are broken by language name (alphabetical) for deterministic output.
func (s ProjectStats) UnsupportedSummary(topN int) string {
	if len(s.ByLanguage) == 0 {
		return ""
	}
	supported := supportedSet()
	type pair struct {
		name  string
		count int
	}
	rows := make([]pair, 0, len(s.ByLanguage))
	for name, count := range s.ByLanguage {
		if _, ok := supported[name]; ok {
			continue
		}
		rows = append(rows, pair{name, count})
	}
	if len(rows) == 0 {
		return ""
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].count != rows[j].count {
			return rows[i].count > rows[j].count
		}
		return rows[i].name < rows[j].name
	})
	if topN > 0 && len(rows) > topN {
		rows = rows[:topN]
	}
	parts := make([]string, 0, len(rows))
	for _, r := range rows {
		parts = append(parts, fmt.Sprintf("%s (%d)", r.name, r.count))
	}
	return strings.Join(parts, ", ")
}

func supportedSet() map[string]struct{} {
	m := make(map[string]struct{}, len(supportedLanguageNames))
	for _, n := range supportedLanguageNames {
		m[n] = struct{}{}
	}
	return m
}
