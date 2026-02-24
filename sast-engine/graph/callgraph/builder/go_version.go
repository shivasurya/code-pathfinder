package builder

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/registry"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
)

const defaultGoVersion = "1.21"

// stdlibRegistryBaseURL is the CDN root for Go stdlib registries.
// Overridden in tests to point at a local httptest.Server.
var stdlibRegistryBaseURL = "https://assets.codepathfinder.dev/registries"

// goVersionRegex matches the "go X.Y" directive in go.mod and go.work files.
// The minor version suffix (e.g. ".4" in "1.21.4") is intentionally not captured;
// normalizeGoVersion handles stripping the patch component from raw go.mod values.
var goVersionRegex = regexp.MustCompile(`(?m)^go\s+(\d+\.\d+)`)

// DetectGoVersion determines the Go toolchain version targeted by a project.
//
// Detection priority:
//  1. go.mod  — "go X.Y" directive (most authoritative for module-aware projects)
//  2. .go-version — explicit version pin file used by tools such as goenv/asdf
//  3. go.work  — workspace go directive (multi-module projects)
//  4. Default  — "1.21" (the most widely deployed minor version as of 2024)
//
// All returned values are normalised to "X.Y" form (patch component stripped).
func DetectGoVersion(projectPath string) string {
	// Priority 1: go.mod
	if v := parseGoVersionFromFile(filepath.Join(projectPath, "go.mod")); v != "" {
		return normalizeGoVersion(v)
	}

	// Priority 2: .go-version
	if v := readGoVersionFile(projectPath); v != "" {
		return v
	}

	// Priority 3: go.work
	if v := parseGoVersionFromFile(filepath.Join(projectPath, "go.work")); v != "" {
		return normalizeGoVersion(v)
	}

	return defaultGoVersion
}

// normalizeGoVersion strips the patch component from a Go version string.
//
//	"1.21"   → "1.21"
//	"1.21.4" → "1.21"
//	"1.26.0" → "1.26"
func normalizeGoVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return version
}

// parseGoVersionFromFile reads a go.mod or go.work file and extracts the
// "go X.Y" directive using goVersionRegex.  Returns "" on any error.
func parseGoVersionFromFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	if match := goVersionRegex.FindSubmatch(data); match != nil {
		return string(match[1])
	}
	return ""
}

// readGoVersionFile reads a .go-version file and returns the normalised version.
// Returns "" when the file does not exist or is empty.
func readGoVersionFile(projectPath string) string {
	data, err := os.ReadFile(filepath.Join(projectPath, ".go-version"))
	if err != nil {
		return ""
	}
	return normalizeGoVersion(strings.TrimSpace(string(data)))
}

// InitGoStdlibLoader creates a GoStdlibRegistryRemote for the project's Go version
// and loads its manifest from the CDN.  On success it stores the loader in
// reg.StdlibLoader so that downstream phases can query stdlib function metadata.
//
// If the manifest cannot be fetched (network unavailable, CDN error, etc.) the
// function logs a warning and returns without setting reg.StdlibLoader, allowing
// the rest of the call-graph construction to proceed without stdlib metadata
// (graceful degradation).
//
// Version resolution:
//  1. reg.GoVersion (set by BuildGoModuleRegistry from go.mod) — normalised to "X.Y"
//  2. DetectGoVersion(projectPath) — full detection chain as fallback
func InitGoStdlibLoader(reg *core.GoModuleRegistry, projectPath string, logger *output.Logger) {
	initGoStdlibLoaderWithBase(reg, projectPath, logger, stdlibRegistryBaseURL)
}

// initGoStdlibLoaderWithBase is the testable inner implementation of InitGoStdlibLoader.
// It accepts an explicit baseURL so that tests can point at a local httptest.Server.
func initGoStdlibLoaderWithBase(reg *core.GoModuleRegistry, projectPath string, logger *output.Logger, baseURL string) {
	version := normalizeGoVersion(reg.GoVersion)
	if version == "" {
		version = DetectGoVersion(projectPath)
	}

	remote := registry.NewGoStdlibRegistryRemote(baseURL, version)
	if err := remote.LoadManifest(logger); err != nil {
		logger.Warning("Failed to load Go %s stdlib manifest: %v", version, err)
		return
	}

	logger.Progress("Loaded Go %s stdlib manifest (%d packages)", version, remote.PackageCount())
	reg.StdlibLoader = remote
}
