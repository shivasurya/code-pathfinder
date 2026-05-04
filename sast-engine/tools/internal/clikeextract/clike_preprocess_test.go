package clikeextract

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPreprocessGlibcAttributes_StripsBareIdentifiers confirms the no-op
// glibc identifiers are blanked out without altering surrounding tokens
// or shifting byte offsets / line numbers.
func TestPreprocessGlibcAttributes_StripsBareIdentifiers(t *testing.T) {
	in := []byte(`extern int foo(void) __THROW __attribute_pure__;
extern int bar(int x) __wur;`)
	out := preprocessGlibcAttributes(in)

	got := string(out)
	assert.NotContains(t, got, "__THROW")
	assert.NotContains(t, got, "__attribute_pure__")
	assert.NotContains(t, got, "__wur")
	assert.Contains(t, got, "int foo(void)")
	assert.Contains(t, got, "int bar(int x)")
	assert.Equal(t, len(in), len(out), "preprocessing must be length-preserving")
	assert.Equal(t, strings.Count(string(in), "\n"), strings.Count(got, "\n"),
		"newline count must be preserved so line numbers stay accurate")
}

// TestPreprocessGlibcAttributes_StripsParenthesizedMacros covers the
// __attribute__((...)) / __nonnull((...)) / __attr_access((...)) forms.
// Balanced-paren tracking must consume the full argument list, including
// nested parens.
func TestPreprocessGlibcAttributes_StripsParenthesizedMacros(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"attribute double paren", `__attribute__((__format__(__printf__, 1, 2)))`},
		{"nonnull double paren", `__nonnull((1, 2, 3))`},
		{"attr_access double paren", `__attr_access((__read_only__, 1, 2))`},
		{"attr_dealloc single paren", `__attr_dealloc(free, 1)`},
		{"attribute newline before parens", "__attribute__\n((nonnull))"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := preprocessGlibcAttributes([]byte("int foo(void) " + tc.in + ";"))
			got := string(out)
			assert.NotContains(t, got, "__attribute__", "%s: head must be stripped", tc.in)
			assert.NotContains(t, got, "__nonnull")
			assert.NotContains(t, got, "__attr_access")
			assert.NotContains(t, got, "__attr_dealloc")
			assert.Contains(t, got, "int foo(void)")
		})
	}
}

// TestPreprocessGlibcAttributes_LeavesNormalCodeAlone is the negative
// sanity check — strings that don't carry any of the recognised tokens
// must come back byte-for-byte identical.
func TestPreprocessGlibcAttributes_LeavesNormalCodeAlone(t *testing.T) {
	in := []byte(`int main(int argc, char **argv) {
    return 0;
}`)
	out := preprocessGlibcAttributes(in)
	assert.Equal(t, in, out)
}

// TestPreprocessGlibcAttributes_PreservesByteOffsets pins the
// length-preserving guarantee: the returned slice has the same length as
// the input, and the strlen identifier sits at the same byte position.
func TestPreprocessGlibcAttributes_PreservesByteOffsets(t *testing.T) {
	in := []byte(`extern size_t strlen (const char *__s)
     __THROW __attribute_pure__ __nonnull ((1));
`)
	out := preprocessGlibcAttributes(in)
	require.Equal(t, len(in), len(out))

	// strlen sits at the same byte offset before and after.
	want := strings.Index(string(in), "strlen")
	got := strings.Index(string(out), "strlen")
	assert.Equal(t, want, got)
}

// TestPreprocessGlibcAttributes_DoesNotMatchInsideIdentifiers verifies the
// word-boundary guard — we don't want to mangle a project identifier that
// happens to contain "__THROW" as a substring.
func TestPreprocessGlibcAttributes_DoesNotMatchInsideIdentifiers(t *testing.T) {
	in := []byte(`int my__THROWN_var = 0;`)
	out := preprocessGlibcAttributes(in)
	assert.Equal(t, in, out)
}

// TestPreprocessGlibcAttributes_HandlesMissingClosingParen tolerates a
// truncated header. Without a closing paren the macro head is left alone
// so we don't mistakenly blank a half-recognized region.
func TestPreprocessGlibcAttributes_HandlesMissingClosingParen(t *testing.T) {
	in := []byte(`int foo() __attribute__((unbalanced
`)
	out := preprocessGlibcAttributes(in)
	// Should NOT blank the macro head when the parens don't balance.
	assert.Contains(t, string(out), "__attribute__")
}

// TestPreprocessGlibcAttributes_Idempotent verifies running the
// preprocessor twice produces the same result as running it once. Cheap
// invariant that catches regressions where a stripped span re-introduces
// matchable text.
func TestPreprocessGlibcAttributes_Idempotent(t *testing.T) {
	in := []byte(`extern void *memmem (const void *__h, size_t __hlen,
                     const void *__n, size_t __nlen)
    __THROW __attribute_pure__ __nonnull ((1, 3))
    __attr_access ((__read_only__, 1, 2))
    __attr_access ((__read_only__, 3, 4));`)
	once := preprocessGlibcAttributes(in)
	twice := preprocessGlibcAttributes(once)
	assert.Equal(t, once, twice)
}

// TestConsumeBalancedParens covers the helper independently of the
// regex so the brace-counting logic is easy to reason about.
func TestConsumeBalancedParens(t *testing.T) {
	tests := []struct {
		name string
		src  string
		from int
		want int
	}{
		{"plain", "(a)", 0, 3},
		{"nested", "((a, (b, c)))", 0, 13},
		{"whitespace before paren", "  (a)", 0, 5},
		{"newline before paren", "\n  (a)", 0, 6},
		{"no opening paren", "abc", 0, -1},
		{"unbalanced", "(a", 0, -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := consumeBalancedParens([]byte(tt.src), tt.from)
			assert.Equal(t, tt.want, got)
		})
	}
}
