package clikeextract

import (
	"regexp"
)

// preprocessGlibcAttributes returns a copy of src with glibc's no-op
// attribute macros stripped to whitespace. Tree-sitter's C grammar can't
// parse declarations like:
//
//	extern size_t strlen (const char *__s)
//	     __THROW __attribute_pure__ __nonnull ((1));
//
// because the trailing macro chain doesn't match any grammar rule. The
// parser produces an ERROR node that swallows the whole declaration, so
// strlen / strcmp / snprintf / memcmp / strcasecmp and dozens of similar
// glibc-decorated functions never make it into the manifest.
//
// The fix is to remove the macro tokens BEFORE parsing. We replace each
// matched range with spaces of the same length so:
//   - Byte offsets and line numbers stay identical to the original (line
//     numbers are reported via tree-sitter's StartPoint(), which counts
//     newlines — preserved here).
//   - Tree-sitter sees clean C, can build a normal `declaration` node, and
//     the existing walker picks it up without further changes.
//
// The list of stripped tokens is intentionally narrow:
//   - Bare identifiers that expand to nothing in real glibc (`__THROW`,
//     `__attribute_pure__`, `__wur`, `__nothrow__`, `__leaf__`)
//   - Function-like macros that take a bracketed argument list — these
//     need balanced-paren tracking, not a regex (`__attribute__((...))`,
//     `__nonnull((...))`, `__attr_access((...))`, `__attr_dealloc(...)`)
//
// Anything not on the list is left alone — better to mis-parse a single
// declaration than silently rewrite tokens we don't fully understand.
//
// Idempotent. Safe to call on already-clean source.
func preprocessGlibcAttributes(src []byte) []byte {
	out := make([]byte, len(src))
	copy(out, src)

	out = stripBareAttributeIdentifiers(out)
	out = stripParenthesizedAttributeMacros(out)
	return out
}

// bareAttributeIdentifiers matches glibc no-op identifier macros. They
// appear as plain tokens (no parens) and expand to nothing in real glibc
// builds, so erasing them is a no-op semantically.
//
// The list is sourced from /usr/include/x86_64-linux-gnu/sys/cdefs.h on
// recent Debian/Ubuntu releases. The pattern uses a non-capturing group
// with alternation; word boundaries ensure we don't match inside
// identifiers (`__THROWN_HARD` stays intact).
//
// What is NOT on this list (intentionally):
//   - `__inline`, `__inline__`, `__restrict`, `__restrict_arr` — real GCC
//     keywords that tree-sitter's C grammar handles natively.
//   - `__extern_inline`, `__extern_always_inline`, `__always_inline`,
//     `__fortify_function` — inline qualifiers; stripping risks changing
//     how the parser classifies the function.
//   - `__flexarr` — expands to `[]` for flexible-array declarations.
//
// Kept as a compile-time package var so each call avoids the regex compile
// cost — hot path runs against thousands of headers.
var bareAttributeIdentifiers = regexp.MustCompile(
	`\b(?:__THROW|__THROWNL|__LEAF|__LEAF_ATTR|__COLD|__BEGIN_DECLS|__END_DECLS|` +
		`__attribute_pure__|__attribute_const__|__attribute_malloc__|` +
		`__attribute_warn_unused_result__|__attribute_used__|__attribute_unused__|` +
		`__attribute_noinline__|__attribute_artificial__|__attribute_returns_twice__|` +
		`__attribute_deprecated__|__attribute_maybe_unused__|__attribute_nonstring__|` +
		`__nothrow__|__leaf__|__wur|__pure__|__const__|__cold__|__hot__|` +
		`__nonnull__|__returns_nonnull|__attr_dealloc_free|__extension__)\b`)

// parenthesizedAttributeMacros matches the leading token of a function-like
// macro expansion. The trailing argument list is consumed by the
// balanced-paren walker in stripParenthesizedAttributeMacros — regex alone
// would mis-handle nested parens.
//
// Notable additions over the bare-token list: `__attribute__`,
// `__nonnull`, `__attr_access`, `__attr_access_none`, `__attr_dealloc`,
// and the size/align/deprecation/format helpers in cdefs.h.
var parenthesizedAttributeMacros = regexp.MustCompile(
	`\b(?:__attribute__|__attribute|__nonnull|__attr_access|__attr_access_none|` +
		`__attr_dealloc|__attr_alloc_size|__attribute_alloc_size__|` +
		`__attribute_alloc_align__|__attribute_format__|__attribute_format_arg__|` +
		`__attribute_format_strfmon__|__attribute_deprecated_msg__|` +
		`__attribute_warn_unused_result__|__nothrow_leaf__|__attribute_artificial__|` +
		`__REDIRECT|__REDIRECT_NTH|__REDIRECT_NTHNL|__REDIRECT_LDBL|` +
		`__REDIRECT_NTH_LDBL|__REDIRECT_FORTIFY|__REDIRECT_FORTIFY_NTH)\b`)

// stripBareAttributeIdentifiers replaces every match of
// bareAttributeIdentifiers with spaces of the same length. Done in-place
// to avoid extra allocations.
func stripBareAttributeIdentifiers(src []byte) []byte {
	matches := bareAttributeIdentifiers.FindAllIndex(src, -1)
	for _, m := range matches {
		blank(src, m[0], m[1])
	}
	return src
}

// stripParenthesizedAttributeMacros replaces every parenthesized-macro
// invocation with spaces. The macro head is found with a regex; the
// trailing argument list is consumed by walking forward and counting
// parens. The whole span (head + argument list) is overwritten.
//
// Handles two real-world shapes:
//
//	__attribute__((__format__(printf, 1, 2)))    // double-paren wrapper
//	__nonnull((1, 2))                            // double-paren wrapper
//	__attr_dealloc(free, 1)                      // single-paren call
//
// Stops at the first balanced `)` after the macro head — that boundary
// always closes the argument list because none of these macros appear
// inside expressions where paren depth would otherwise matter.
func stripParenthesizedAttributeMacros(src []byte) []byte {
	for {
		loc := parenthesizedAttributeMacros.FindIndex(src)
		if loc == nil {
			return src
		}
		end := consumeBalancedParens(src, loc[1])
		if end < 0 {
			// No paren followed (or no close paren found) — leave the
			// match alone rather than blank a half-recognized region.
			// Move past the match so the next iteration doesn't re-match.
			//
			// Replacing only the macro head would risk turning
			// `__attribute_format_arg__(1)` (function-like) into a bare
			// `(1)` that tree-sitter then mis-classifies. Skipping is
			// safer.
			break
		}
		blank(src, loc[0], end)
	}
	return src
}

// consumeBalancedParens scans forward from start to find the closing `)`
// that balances the first `(` after start. Returns the index just past the
// closing `)`, or -1 when no opening `(` is found before the next
// non-whitespace, non-paren token, or the closing paren is missing.
//
// Whitespace between the macro head and its `(` is allowed — glibc commonly
// formats long attribute lines with line breaks before the argument list.
func consumeBalancedParens(src []byte, start int) int {
	i := start
	// Skip whitespace until we find the opening paren.
	for i < len(src) && isSpace(src[i]) {
		i++
	}
	if i >= len(src) || src[i] != '(' {
		return -1
	}
	depth := 0
	for ; i < len(src); i++ {
		switch src[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}

// blank overwrites src[start:end] with spaces, leaving newlines intact so
// tree-sitter's reported line numbers continue to match the original
// source. Idempotent on already-blank ranges.
func blank(src []byte, start, end int) {
	for i := start; i < end; i++ {
		if src[i] != '\n' {
			src[i] = ' '
		}
	}
}

func isSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\v', '\f':
		return true
	}
	return false
}

