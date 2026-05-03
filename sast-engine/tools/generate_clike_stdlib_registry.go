//go:build cpf_generate_stdlib_registry

// generate_clike_stdlib_registry is a standalone tool that walks installed
// system headers (Linux glibc + libstdc++ in PR-01; Windows + Darwin in PR-03)
// and emits per-header JSON registry files describing functions, classes,
// methods, typedefs, and constants — the input the loader (PR-02) consumes
// when resolving stdlib calls during analysis.
//
// Usage:
//
//	go run -tags cpf_generate_stdlib_registry tools/generate_clike_stdlib_registry.go \
//	    --target=linux --language=c --output-dir=./out/linux/c/v1
//
// Flags:
//
//	--target      "linux" (PR-01); "windows" / "darwin" land in PR-03
//	--language    "c" or "cpp"
//	--output-dir  Directory to write manifest.json + per-header JSON files
//	--overlay     Path to YAML overlay; defaults to tools/<lang>_stdlib_overlay.yaml
//	--base-url    Override the URL stamped into manifest entries (default
//	              https://assets.codepathfinder.dev/registries — useful for
//	              file:// in local development)
//
// Output: <output-dir>/manifest.json + <output-dir>/<header>_stdlib.json files.
//
// The build tag `cpf_generate_stdlib_registry` excludes this file from the
// regular `go build` / `go test` invocations — matches the existing pattern
// used by tools/generate_go_stdlib_registry.go.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/tools/internal/clikeextract"
)

func main() {
	target := flag.String("target", core.PlatformLinux,
		"target platform: linux | windows | darwin (windows/darwin land in PR-03)")
	language := flag.String("language", "",
		"language: c or cpp (required)")
	outputDir := flag.String("output-dir", "",
		"output directory (required)")
	overlayPath := flag.String("overlay", "",
		"path to YAML overlay (default: tools/<language>_stdlib_overlay.yaml)")
	baseURL := flag.String("base-url", "",
		"override base URL stamped into manifest entries (default: assets.codepathfinder.dev/registries)")
	flag.Parse()

	if *language == "" {
		fail("--language is required (c or cpp)")
	}
	if *outputDir == "" {
		fail("--output-dir is required")
	}

	if *overlayPath == "" {
		// Auto-detect overlay alongside this binary's source directory.
		// Resolve relative to the cwd; callers running `go run` from the
		// sast-engine directory will hit `tools/<lang>_stdlib_overlay.yaml`.
		guess := filepath.Join("tools", *language+"_stdlib_overlay.yaml")
		if _, err := os.Stat(guess); err == nil {
			*overlayPath = guess
		}
	}

	cfg := clikeextract.Config{
		Target:      *target,
		Language:    *language,
		OutputDir:   *outputDir,
		OverlayPath: *overlayPath,
		BaseURL:     *baseURL,
	}
	if err := cfg.Validate(); err != nil {
		fail(err.Error())
	}

	if err := clikeextract.NewExtractor(cfg).Run(); err != nil {
		fail("generator failed: %v", err)
	}

	fmt.Fprintf(os.Stderr, "wrote registry to %s (target=%s language=%s overlay=%s)\n",
		*outputDir, *target, *language, *overlayPath)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	flag.Usage()
	os.Exit(2)
}
