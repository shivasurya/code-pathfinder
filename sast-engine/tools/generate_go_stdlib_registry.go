//go:build cpf_generate_stdlib_registry

// generate_go_stdlib_registry is a standalone tool that extracts the exported API
// surface of Go standard library packages and writes versioned JSON registry files.
//
// Usage:
//
//	go run tools/generate_go_stdlib_registry.go --go-version 1.21 --output-dir ./out
//
// Flags:
//
//	--go-version   Go version to tag in the registry (e.g., "1.21", "1.26.0").
//	               Defaults to the version of the running Go toolchain.
//	--output-dir   Directory to write registry files (required).
//	--goroot       GOROOT path. Defaults to runtime.GOROOT().
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/tools/internal/goextract"
)

func main() {
	goVersion := flag.String("go-version", "", "Go version tag (e.g., 1.21, 1.26.0). Defaults to current runtime version.")
	outputDir := flag.String("output-dir", "", "Output directory for registry JSON files (required).")
	goroot := flag.String("goroot", "", "GOROOT path. Defaults to runtime.GOROOT().")
	flag.Parse()

	if *goVersion == "" {
		// Auto-detect from the running toolchain: "go1.26.0" → "1.26.0".
		*goVersion = strings.TrimPrefix(runtime.Version(), "go")
	}

	if *outputDir == "" {
		fmt.Fprintln(os.Stderr, "error: --output-dir is required")
		flag.Usage()
		os.Exit(1)
	}

	if *goroot == "" {
		*goroot = runtime.GOROOT()
	}

	cfg := goextract.Config{
		GoVersion: *goVersion,
		GOROOT:    *goroot,
		OutputDir: *outputDir,
	}

	extractor := goextract.NewExtractor(cfg)
	if err := extractor.Run(); err != nil {
		log.Fatalf("extraction failed: %v", err)
	}

	fmt.Printf("Successfully generated stdlib registry for Go %s in %s\n", *goVersion, *outputDir)
}
