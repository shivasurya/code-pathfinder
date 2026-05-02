package clike

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsCSourceFile(t *testing.T) {
	defer headerLanguageCache.Delete("/tmp/cache_h_c.h")
	defer headerLanguageCache.Delete("/tmp/cache_h_cpp.h")

	CacheHeaderLanguage("/tmp/cache_h_c.h", false)
	CacheHeaderLanguage("/tmp/cache_h_cpp.h", true)

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{".c file", "main.c", true},
		{".cpp file", "main.cpp", false},
		{".cc file", "main.cc", false},
		{".cxx file", "main.cxx", false},
		{".hpp file", "main.hpp", false},
		{".hh file", "main.hh", false},
		{".hxx file", "main.hxx", false},
		{".java file", "Main.java", false},
		{".py file", "main.py", false},
		{".go file", "main.go", false},
		{"no extension", "Makefile", false},
		{".h cached as C", "/tmp/cache_h_c.h", true},
		{".h cached as C++", "/tmp/cache_h_cpp.h", false},
		{".h not cached defaults to C", "/tmp/cache_h_unknown.h", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCSourceFile(tt.filename); got != tt.want {
				t.Errorf("IsCSourceFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsCppSourceFile(t *testing.T) {
	defer headerLanguageCache.Delete("/tmp/cppcache_c.h")
	defer headerLanguageCache.Delete("/tmp/cppcache_cpp.h")

	CacheHeaderLanguage("/tmp/cppcache_c.h", false)
	CacheHeaderLanguage("/tmp/cppcache_cpp.h", true)

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{".cpp file", "main.cpp", true},
		{".cc file", "main.cc", true},
		{".cxx file", "main.cxx", true},
		{".hpp file", "main.hpp", true},
		{".hh file", "main.hh", true},
		{".hxx file", "main.hxx", true},
		{".c file", "main.c", false},
		{".java file", "Main.java", false},
		{".py file", "main.py", false},
		{".go file", "main.go", false},
		{".h cached as C", "/tmp/cppcache_c.h", false},
		{".h cached as C++", "/tmp/cppcache_cpp.h", true},
		{".h not cached defaults to C", "/tmp/cppcache_unknown.h", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCppSourceFile(tt.filename); got != tt.want {
				t.Errorf("IsCppSourceFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestDetectCppInHeader(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name: "pure C header with typedef and struct",
			content: `#ifndef UTILS_H
#define UTILS_H
typedef struct Point { int x; int y; } Point;
int add(int a, int b);
#endif
`,
			want: false,
		},
		{
			name: "C++ class header",
			content: `#pragma once
class Foo {
public:
    int bar();
};
`,
			want: true,
		},
		{
			name:    "namespace header",
			content: "namespace mylib {\nint compute();\n}\n",
			want:    true,
		},
		{
			name:    "template header",
			content: "template<typename T>\nT identity(T v) { return v; }\n",
			want:    true,
		},
		{
			name:    "qualified call uses ::",
			content: "void f() { std::cout << 1; }\n",
			want:    true,
		},
		{
			name:    "empty file",
			content: "",
			want:    false,
		},
		{
			name:    "extern C block (no C++ indicator on first lines)",
			content: "#ifdef __cplusplus\nextern \"C\" {\n#endif\nint plain_c(void);\n",
			want:    false,
		},
	}

	dir, err := os.MkdirTemp("", "detect_cpp_header")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, "h_"+tt.name+".h")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("write: %v", err)
			}
			if got := DetectCppInHeader(path); got != tt.want {
				t.Errorf("DetectCppInHeader(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}

	t.Run("missing file returns false", func(t *testing.T) {
		if DetectCppInHeader(filepath.Join(dir, "does_not_exist.h")) {
			t.Error("expected false for missing file")
		}
	})
}
