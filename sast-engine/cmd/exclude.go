package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validateExcludePatterns normalizes and validates a list of repo-relative path prefixes.
// It returns the cleaned slice on success, or an error listing the offending pattern.
//
// A pattern is rejected if it:
//   - is absolute (starts with "/")
//   - contains ".." as a path component
//   - contains a null byte
//   - contains a backslash (only forward slashes are allowed)
//   - exceeds 512 characters
//
// Valid patterns are normalized: leading slashes stripped, trailing slashes
// stripped. Exact duplicates (after normalization) are dropped silently so the
// caller can repeat --exclude flags without bloating the per-file check loop.
func validateExcludePatterns(patterns []string) ([]string, error) {
	cleaned := make([]string, 0, len(patterns))
	seen := make(map[string]struct{}, len(patterns))
	for _, p := range patterns {
		if len(p) > 512 {
			return nil, fmt.Errorf("--exclude pattern too long (>512 chars): %q", p)
		}
		if strings.HasPrefix(p, "/") {
			return nil, fmt.Errorf("--exclude pattern must be repo-relative (no leading slash): %q", p)
		}
		if strings.Contains(p, "\\") {
			return nil, fmt.Errorf("--exclude pattern must use forward slashes, not backslashes: %q", p)
		}
		if strings.ContainsRune(p, 0) {
			return nil, fmt.Errorf("--exclude pattern contains null byte: %q", p)
		}
		// Check every path component for ".."
		norm := filepath.ToSlash(strings.Trim(p, "/"))
		for _, seg := range strings.Split(norm, "/") {
			if seg == ".." {
				return nil, fmt.Errorf("--exclude pattern must not contain '..': %q", p)
			}
		}
		if _, dup := seen[norm]; dup {
			continue
		}
		seen[norm] = struct{}{}
		cleaned = append(cleaned, norm)
	}
	return cleaned, nil
}

// isExcluded reports whether relPath (forward-slash, repo-relative) is covered by
// any of the given patterns. A file is covered when its path starts with
// "<pattern>/", ensuring "rules" matches "rules/foo.py" but not "rulesx/foo.py".
// An exact match (relPath == pattern) is also considered covered.
func isExcluded(relPath string, patterns []string) bool {
	rel := filepath.ToSlash(relPath)
	for _, p := range patterns {
		if p == "" {
			continue
		}
		// Exact match or prefix match with a separator boundary.
		if rel == p || strings.HasPrefix(rel, p+"/") {
			return true
		}
	}
	return false
}
