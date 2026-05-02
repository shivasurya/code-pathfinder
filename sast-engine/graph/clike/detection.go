package clike

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// headerLanguageCache stores the C/C++ classification of .h files keyed by
// absolute path. Values are bool: true => C++, false => C.
//
// .h is shared between C and C++ grammars and the correct grammar can only be
// chosen by inspecting the file contents. Reading the file once per AST node
// would be unacceptable, so the worker that drives the parsing pipeline calls
// CacheHeaderLanguage exactly once per .h file before AST traversal begins;
// IsCSourceFile and IsCppSourceFile then read the cache without performing I/O.
var headerLanguageCache sync.Map

// CacheHeaderLanguage records that filename is a C++ header (isCpp == true) or
// a C header (isCpp == false). Must be called once per .h file in the parsing
// worker before any IsCSourceFile / IsCppSourceFile lookup for that file.
func CacheHeaderLanguage(filename string, isCpp bool) {
	headerLanguageCache.Store(filename, isCpp)
}

// IsCSourceFile reports whether filename should be parsed with the C grammar.
//
// Source-extension cases (.c) are answered directly. For .h files the answer
// comes from the header cache populated by CacheHeaderLanguage; an uncached
// .h falls back to C as the safe default (the grammar overlap means a
// misclassified C++ header still parses as a structurally-valid translation
// unit, just with reduced fidelity).
func IsCSourceFile(filename string) bool {
	ext := filepath.Ext(filename)
	if ext == ".c" {
		return true
	}
	if ext == ".h" {
		if v, ok := headerLanguageCache.Load(filename); ok {
			return !v.(bool)
		}
		return true
	}
	return false
}

// IsCppSourceFile reports whether filename should be parsed with the C++
// grammar.
//
// .cpp/.cc/.cxx and the C++-only header extensions (.hpp/.hh/.hxx) are
// answered directly. For .h files the answer comes from the header cache
// populated by CacheHeaderLanguage; an uncached .h falls back to "not C++"
// so that IsCSourceFile and IsCppSourceFile remain mutually exclusive.
func IsCppSourceFile(filename string) bool {
	ext := filepath.Ext(filename)
	switch ext {
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx":
		return true
	case ".h":
		if v, ok := headerLanguageCache.Load(filename); ok {
			return v.(bool)
		}
		return false
	}
	return false
}

// cppHeaderIndicators are byte sequences whose presence in a header file is a
// strong signal that the header should be parsed as C++ rather than C. The
// list is intentionally small and conservative — every entry is invalid in
// pure C, so a positive match is high-confidence.
var cppHeaderIndicators = []string{
	"class ", "namespace ", "template<", "template <",
	"public:", "private:", "protected:", "::",
}

// detectCppHeaderScanLines bounds how far DetectCppInHeader reads into a file.
// 100 lines covers the include guards, license header, and the first few
// declarations of every real-world header surveyed during the design phase.
const detectCppHeaderScanLines = 100

// DetectCppInHeader scans the first detectCppHeaderScanLines lines of filename
// for C++-only indicators (see cppHeaderIndicators).
//
// This is a best-effort heuristic, not a full preprocessor or parser. It is
// invoked from the worker exactly once per file and its result is stored via
// CacheHeaderLanguage; AST-traversal code reads the cache and never calls
// this function on the hot path.
//
// A missing or unreadable file returns false (treated as C) so that downstream
// parsing fails on an obviously-broken input rather than mislabeling the
// language.
func DetectCppInHeader(filename string) bool {
	f, err := os.Open(filename)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for lineCount := 0; lineCount < detectCppHeaderScanLines && scanner.Scan(); lineCount++ {
		line := scanner.Text()
		for _, ind := range cppHeaderIndicators {
			if strings.Contains(line, ind) {
				return true
			}
		}
	}
	return false
}
