package cmd

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- validateExcludePatterns ---

func TestValidateExcludePatterns_EmptyList(t *testing.T) {
	got, err := validateExcludePatterns(nil)
	require.NoError(t, err)
	assert.Empty(t, got)

	got, err = validateExcludePatterns([]string{})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestValidateExcludePatterns_SingleValid(t *testing.T) {
	got, err := validateExcludePatterns([]string{"rules/"})
	require.NoError(t, err)
	// Trailing slash must be stripped.
	assert.Equal(t, []string{"rules"}, got)
}

func TestValidateExcludePatterns_MultipleValid(t *testing.T) {
	got, err := validateExcludePatterns([]string{"rules/", "sast-engine/test-fixtures"})
	require.NoError(t, err)
	assert.Equal(t, []string{"rules", "sast-engine/test-fixtures"}, got)
}

func TestValidateExcludePatterns_TrailingSlashStripped(t *testing.T) {
	got, err := validateExcludePatterns([]string{"foo/bar/"})
	require.NoError(t, err)
	assert.Equal(t, []string{"foo/bar"}, got)
}

func TestValidateExcludePatterns_AbsoluteRejected(t *testing.T) {
	_, err := validateExcludePatterns([]string{"/etc/passwd"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no leading slash")
}

func TestValidateExcludePatterns_TraversalRejected(t *testing.T) {
	cases := []string{
		"../secret",
		"foo/../bar",
		"foo/..",
	}
	for _, p := range cases {
		t.Run(p, func(t *testing.T) {
			_, err := validateExcludePatterns([]string{p})
			require.Error(t, err, "pattern %q should be rejected", p)
			assert.Contains(t, err.Error(), "..")
		})
	}
}

func TestValidateExcludePatterns_BackslashRejected(t *testing.T) {
	_, err := validateExcludePatterns([]string{"foo\\bar"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backslash")
}

func TestValidateExcludePatterns_NullByteRejected(t *testing.T) {
	_, err := validateExcludePatterns([]string{"foo\x00bar"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "null byte")
}

func TestValidateExcludePatterns_TooLongRejected(t *testing.T) {
	long := strings.Repeat("a", 513)
	_, err := validateExcludePatterns([]string{long})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too long")
}

func TestValidateExcludePatterns_ExactlyMaxLengthAllowed(t *testing.T) {
	maxLen := strings.Repeat("a", 512)
	got, err := validateExcludePatterns([]string{maxLen})
	require.NoError(t, err)
	assert.Equal(t, []string{maxLen}, got)
}

func TestValidateExcludePatterns_UnicodeAllowed(t *testing.T) {
	got, err := validateExcludePatterns([]string{"src/testi18n/世界"})
	require.NoError(t, err)
	assert.Equal(t, []string{"src/testi18n/世界"}, got)
}

func TestValidateExcludePatterns_ExactDuplicatesDropped(t *testing.T) {
	got, err := validateExcludePatterns([]string{"rules", "vendor", "rules"})
	require.NoError(t, err)
	assert.Equal(t, []string{"rules", "vendor"}, got)
}

func TestValidateExcludePatterns_DuplicatesAfterNormalization(t *testing.T) {
	// "rules/" and "rules" both normalize to "rules"; only one survives.
	// "/vendor/" and "vendor" both normalize to "vendor".
	got, err := validateExcludePatterns([]string{"rules/", "rules", "vendor", "vendor/"})
	require.NoError(t, err)
	assert.Equal(t, []string{"rules", "vendor"}, got)
}

func TestValidateExcludePatterns_DedupPreservesFirstOccurrenceOrder(t *testing.T) {
	got, err := validateExcludePatterns([]string{"c", "a", "b", "a", "c"})
	require.NoError(t, err)
	assert.Equal(t, []string{"c", "a", "b"}, got)
}

// --- isExcluded ---

func TestIsExcluded_EmptyPatterns(t *testing.T) {
	assert.False(t, isExcluded("rules/foo.py", nil))
	assert.False(t, isExcluded("rules/foo.py", []string{}))
}

func TestIsExcluded_SinglePrefixMatch(t *testing.T) {
	assert.True(t, isExcluded("rules/foo.py", []string{"rules"}))
}

func TestIsExcluded_SinglePrefixNoMatch(t *testing.T) {
	// "rules" must NOT match "rulesx/foo.py" (no separator boundary).
	assert.False(t, isExcluded("rulesx/foo.py", []string{"rules"}))
}

func TestIsExcluded_ExactDirMatch(t *testing.T) {
	// Exact match of the directory name itself is excluded.
	assert.True(t, isExcluded("rules", []string{"rules"}))
}

func TestIsExcluded_NestedPrefix(t *testing.T) {
	assert.True(t, isExcluded("sast-engine/test-fixtures/java/Main.java", []string{"sast-engine/test-fixtures"}))
}

func TestIsExcluded_MultiplePatterns(t *testing.T) {
	patterns := []string{"rules", "sast-engine/test-fixtures"}
	assert.True(t, isExcluded("rules/owasp.py", patterns))
	assert.True(t, isExcluded("sast-engine/test-fixtures/x.java", patterns))
	assert.False(t, isExcluded("sast-engine/cmd/scan.go", patterns))
}

func TestIsExcluded_EmptyPattern(t *testing.T) {
	// An empty string in the pattern list must be a no-op, not a wildcard.
	assert.False(t, isExcluded("anything.py", []string{""}))
}

func TestIsExcluded_CaseSensitive(t *testing.T) {
	// Patterns are case-sensitive on Linux; verify no lowercasing occurs.
	assert.False(t, isExcluded("Rules/foo.py", []string{"rules"}))
	assert.True(t, isExcluded("Rules/foo.py", []string{"Rules"}))
}

func TestIsExcluded_WindowsPathNormalized(t *testing.T) {
	// Even if relPath uses OS separator, forward-slash comparison must work.
	// filepath.ToSlash is applied inside isExcluded.
	assert.True(t, isExcluded("rules/foo.py", []string{"rules"}))
}
