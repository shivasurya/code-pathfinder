package clikeextract

import (
	"regexp"
	"strings"
)

// Normalisation helpers shared by the C and C++ extractors. System headers carry
// a lot of compiler-implementation noise — gcc attributes, libc++/libstdc++
// internal namespace prefixes, _GLIBCXX_NOEXCEPT macros — that makes mechanical
// type comparison fail. These helpers strip the noise so emitted entries match
// the canonical forms found in cppreference and developer-facing source.

// attributeRegex matches `__attribute__((...))` clauses, including nested parens
// (one level deep, which is sufficient for every libc/libstdc++ form we care
// about: format(printf,1,2), nonnull(1), warn_unused_result, deprecated("msg")).
var attributeRegex = regexp.MustCompile(`__attribute__\s*\(\([^)]*(\)[^)]*)*\)\)`)

// pragmaRegex matches `__cdecl` / `__stdcall` / `__fastcall` calling-convention
// markers used in mingw/MSVC headers.
var pragmaRegex = regexp.MustCompile(`__(cdecl|stdcall|fastcall|thiscall|vectorcall)\b`)

// libcppMacroRegex matches macros that expand to nothing in real builds but show
// up in the AST verbatim (we don't run a preprocessor). Each macro is treated as
// removable whitespace.
var libcppMacroRegex = regexp.MustCompile(strings.Join([]string{
	`_GLIBCXX_NOEXCEPT`,
	`_GLIBCXX_USE_NOEXCEPT`,
	`_GLIBCXX_CONSTEXPR`,
	`_GLIBCXX17_CONSTEXPR`,
	`_GLIBCXX20_CONSTEXPR`,
	`_LIBCPP_INLINE_VISIBILITY`,
	`_LIBCPP_HIDE_FROM_ABI`,
	`_LIBCPP_CONSTEXPR_SINCE_CXX\d+`,
	`_LIBCPP_NODISCARD`,
	`_NOEXCEPT`,
	`_Nonnull`,
	`_Nullable`,
	`_Null_unspecified`,
	`__THROW`,
	`__nonnull\s*\(\s*\(\s*[^)]*\)\s*\)`,
	`__wur`,
}, "|"))

// cxx11InlineNamespaceRegex matches the libstdc++ inline namespace `__cxx11`
// that wraps C++11-ABI-stable types (basic_string, list). Stripping it gives the
// canonical type used in user code: `std::__cxx11::basic_string` →
// `std::basic_string`. The same shape catches `std::__1::` (libc++) and
// `std::__detail::` for completeness — the latter should normally be filtered
// out higher up the stack as a private symbol, but the regex catches the cases
// that slip through (e.g. inside a typedef value).
var cxx11InlineNamespaceRegex = regexp.MustCompile(`std::__(cxx11|1|detail)::`)

// NormalizeType strips compiler decorations and canonicalises stdlib internal
// namespaces. Input may be empty; output is whitespace-collapsed but otherwise
// preserves the original form.
//
// Examples:
//
//	"int __attribute__((nonnull))"            → "int"
//	"const char* _Nonnull"                    → "const char*"
//	"std::__cxx11::basic_string<char>"        → "std::basic_string<char>"
//	"void __cdecl printf(const char*)"        → "void printf(const char*)"
//	"_GLIBCXX_CONSTEXPR size_t size() const"  → "size_t size() const"
//
// The function is intentionally conservative: anything it doesn't recognise is
// left alone. The overlay (overlay.go) is the escape hatch for anything the
// regex set doesn't cover.
func NormalizeType(s string) string {
	if s == "" {
		return s
	}
	s = attributeRegex.ReplaceAllString(s, "")
	s = pragmaRegex.ReplaceAllString(s, "")
	s = libcppMacroRegex.ReplaceAllString(s, "")
	s = cxx11InlineNamespaceRegex.ReplaceAllString(s, "std::")
	return collapseWhitespace(s)
}

// collapseWhitespace replaces runs of whitespace with a single space and trims
// leading/trailing space — the noise left over after the regex strips run.
func collapseWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	prevSpace := true // start as if preceded by whitespace, so leading is trimmed
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}

	out := b.String()
	return strings.TrimRight(out, " ")
}

// IsPrivateSymbol reports whether name is a compiler/library implementation
// detail that should not appear in the public registry. The rule is conservative:
//
//   - Names with a double-underscore prefix (`__builtin_*`, `__cxxabiv1::*`)
//     are private by every C and C++ convention.
//   - Names with a single-underscore prefix followed by a lowercase letter are
//     library private (`_exit` is the GNU underscore alias for `exit`; we want
//     the public spelling, so we skip the alias).
//   - Names starting with `_` followed by an uppercase letter or a digit are
//     standard-library KEEP forms (`_Bool`, `_Static_assert`, `_Generic`, the
//     `_LIBCPP_*` macros that get filtered separately) — these we keep.
//
// Edge case the keep-list rule above is designed for: `_Bool` is a C language
// keyword (since C99); skipping it would lose a real type. `_LIBCPP_*` is
// macro/internal but the macro is consumed before extraction reaches the name,
// so we should never see it here — but the keep rule means even if we do, we
// don't accidentally drop something that's just an oddly-named keyword.
func IsPrivateSymbol(name string) bool {
	if len(name) < 2 {
		return false
	}
	if !strings.HasPrefix(name, "_") {
		return false
	}
	if strings.HasPrefix(name, "__") {
		return true
	}
	// Single underscore + lowercase ascii letter → library-private (e.g. _exit).
	c := name[1]
	if c >= 'a' && c <= 'z' {
		return true
	}
	return false
}

// IsPrivateNamespace reports whether the given C++ namespace name is a
// compiler-implementation detail that should be skipped during extraction.
// Namespaces with `__` prefix or matching well-known internal patterns are
// excluded; user-facing namespaces (`std`, `boost`, `Qt`) pass through.
func IsPrivateNamespace(name string) bool {
	if name == "" {
		return false
	}
	if strings.HasPrefix(name, "__") {
		return true
	}
	if name == "_GLIBCXX_DEBUG" || strings.HasPrefix(name, "_GLIBCXX_") || strings.HasPrefix(name, "_LIBCPP_") {
		return true
	}
	return false
}

// SanitizeHeaderName converts a `#include`-form header name into a stable
// filesystem-safe identifier suitable for use as a JSON filename stem. The
// transformation is:
//
//   - Strip the leading `<` and trailing `>` if present.
//   - Strip the trailing `.h`, `.hpp`, or `.hxx` extension.
//   - Replace `/` with `_` so subdirectories (`sys/socket.h`, `bits/types.h`)
//     yield a flat filename.
//
// Examples:
//
//	"vector"          → "vector"
//	"<string>"        → "string"
//	"stdio.h"         → "stdio"
//	"sys/socket.h"    → "sys_socket"
//	"bits/types.h"    → "bits_types"
//
// Idempotent — running it twice on the same input yields the same output.
func SanitizeHeaderName(h string) string {
	h = strings.TrimPrefix(h, "<")
	h = strings.TrimSuffix(h, ">")
	for _, ext := range []string{".hpp", ".hxx", ".hh", ".h"} {
		if cut, ok := strings.CutSuffix(h, ext); ok {
			h = cut
			break
		}
	}
	h = strings.ReplaceAll(h, "/", "_")
	return h
}
