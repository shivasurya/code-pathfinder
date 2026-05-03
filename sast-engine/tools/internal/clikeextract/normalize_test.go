package clikeextract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeType(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"plain int", "int", "int"},
		{"strip nonnull attribute", "int __attribute__((nonnull))", "int"},
		{"strip format attribute", "int __attribute__((format(printf, 1, 2)))", "int"},
		{"strip Nonnull tag", "const char* _Nonnull", "const char*"},
		{"strip cdecl", "void __cdecl printf(const char*)", "void printf(const char*)"},
		{"canonicalize cxx11 namespace", "std::__cxx11::basic_string<char>", "std::basic_string<char>"},
		{"canonicalize libc++ inline", "std::__1::vector<int>", "std::vector<int>"},
		{"strip GLIBCXX_CONSTEXPR", "_GLIBCXX_CONSTEXPR size_t size() const", "size_t size() const"},
		{"strip libcpp inline visibility", "_LIBCPP_INLINE_VISIBILITY const T& at(size_t)", "const T& at(size_t)"},
		{"strip THROW marker", "void* __THROW malloc(size_t)", "void* malloc(size_t)"},
		{"collapse whitespace", "int   *foo   ()", "int *foo ()"},
		{"trim trailing space", "int  ", "int"},
		{"unrecognised left alone", "MyCustomType<int>", "MyCustomType<int>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeType(tt.in)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCollapseWhitespace(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"  ", ""},
		{"abc", "abc"},
		{"a b", "a b"},
		{"a  b", "a b"},
		{"a\tb\nc", "a b c"},
		{"  leading", "leading"},
		{"trailing  ", "trailing"},
		{"  both  ", "both"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, collapseWhitespace(tt.in))
		})
	}
}

func TestIsPrivateSymbol(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"", false},
		{"printf", false},
		{"_Bool", false},          // C keyword — keep
		{"_Static_assert", false}, // C keyword — keep
		{"_Generic", false},       // C keyword — keep
		{"_exit", true},           // library-private alias
		{"_setjmp", true},
		{"__builtin_strlen", true},
		{"__GLIBC_INTERNAL", true},
		{"__cxxabiv1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsPrivateSymbol(tt.name))
		})
	}
}

func TestIsPrivateNamespace(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"", false},
		{"std", false},
		{"boost", false},
		{"__detail", true},
		{"__cxxabiv1", true},
		{"_GLIBCXX_DEBUG", true},
		{"_GLIBCXX_VERSION_NAMESPACE", true},
		{"_LIBCPP_INTERNAL", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsPrivateNamespace(tt.name))
		})
	}
}

func TestSanitizeHeaderName(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"vector", "vector"},
		{"<string>", "string"},
		{"<vector>", "vector"},
		{"stdio.h", "stdio"},
		{"string.h", "string"},
		{"<stdio.h>", "stdio"},
		{"sys/socket.h", "sys_socket"},
		{"bits/types.h", "bits_types"},
		{"foo.hpp", "foo"},
		{"foo.hxx", "foo"},
		{"foo.hh", "foo"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assert.Equal(t, tt.want, SanitizeHeaderName(tt.in))
		})
	}
}

func TestSanitizeHeaderName_Idempotent(t *testing.T) {
	for _, in := range []string{"vector", "<string>", "stdio.h", "sys/socket.h"} {
		once := SanitizeHeaderName(in)
		twice := SanitizeHeaderName(once)
		assert.Equal(t, once, twice, "input=%q", in)
	}
}
