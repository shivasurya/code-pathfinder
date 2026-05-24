// Package clikeextract walks installed C and C++ system headers and emits per-header
// JSON registry files describing functions, classes, methods, typedefs, and #define
// constants — the input the loader (PR-02) consumes when resolving stdlib calls
// during analysis.
//
// The package is the heavy lifter behind tools/generate_clike_stdlib_registry.go.
// The entry-point binary is a thin //go:build cpf_generate_stdlib_registry wrapper
// that flag-parses and calls Extractor.Run; everything else lives here so it stays
// testable under regular `go test ./...`.
//
// # Pipeline
//
//	discoverHeaderSources(target, lang)        →  []HeaderSource
//	walkHeaders(src)                           →  []HeaderFile
//	extractHeader(file, lang)                  →  *core.CStdlibHeader  (Source: "header")
//	mergeOverlay(extracted, overlay)           →  *core.CStdlibHeader  (Source: "merged" or unchanged)
//	emitter.WritePerHeader(...) + WriteManifest(...)
//
// # Why a separate package (not flat in tools/)
//
// The flat layout in the original PR-01 spec piles 1,500+ LoC into tools/. Sibling
// generator tools/internal/goextract demonstrates the alternative: thin entry-point
// + internal package with one file per concern. This package follows that pattern
// for consistency and so each concern (walker, extractor, overlay, emitter) is
// individually unit-testable.
//
// # Reuse of Phase 1 helpers
//
// AST extraction reuses graph/clike helpers wherever possible
// (ExtractFunctionInfo, ExtractStructFields, ExtractTypeString, ExtractParameters)
// — this package does not re-implement tree-sitter walking that already exists
// upstream. C++-specific concerns not yet covered by graph/clike (preproc_def
// macro extraction, template_parameter_list capture on classes, namespace tracking
// for free functions) are added here.
//
// # Output layout
//
//	<output-dir>/
//	  manifest.json                  CStdlibManifest — top-level index
//	  <sanitized-header>_stdlib.json CStdlibHeader   — one per header
//
// where sanitized-header strips the .h extension and replaces /'s with _'s.
package clikeextract
