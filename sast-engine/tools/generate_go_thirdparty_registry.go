//go:build cpf_generate_thirdparty_registry

// generate_go_thirdparty_registry is a standalone tool that downloads Go third-party
// modules and extracts their exported API surface into versioned JSON registry files
// compatible with the GoThirdPartyRegistryRemote CDN loader.
//
// Usage:
//
//	go run -tags cpf_generate_thirdparty_registry tools/generate_go_thirdparty_registry.go \
//	  --packages-file tools/top1000.txt \
//	  --output-dir ./out/go-thirdparty/v1/
//
// Flags:
//
//	--packages-file  File with "module@version" lines (default: tools/top1000.txt).
//	--output-dir     Directory to write registry JSON files (default: ./out/go-thirdparty/v1/).
//
// Output layout:
//
//	{output-dir}/manifest.json           — registry index with checksums
//	{output-dir}/{encoded-path}.json     — per-package type metadata
//
// Module path encoding: slashes are replaced with underscores.
//
//	"gorm.io/gorm"             → "gorm.io_gorm.json"
//	"github.com/gin-gonic/gin" → "github.com_gin-gonic_gin.json"
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/tools/internal/goextract"
)

// moduleSpec holds a parsed "module@version" entry from the packages file.
type moduleSpec struct {
	Path    string
	Version string
}

// cmdRunner executes an external command and returns its combined output.
// It is a package-level variable so it can be replaced in tests.
var cmdRunner = func(name string, arg ...string) ([]byte, error) {
	return exec.Command(name, arg...).Output() //nolint:wrapcheck
}

func main() {
	packagesFile := flag.String("packages-file", "tools/top1000.txt", "File with module@version lines")
	outputDir := flag.String("output-dir", "./out/go-thirdparty/v1/", "Output directory for JSON files")
	flag.Parse()

	modules, err := readPackageList(*packagesFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading packages file: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	extractor := goextract.NewExtractor(goextract.Config{})
	manifest := core.NewGoManifest()
	manifest.SchemaVersion = "1.0.0"
	manifest.RegistryVersion = "v1"
	manifest.GeneratedAt = time.Now().UTC().Format(time.RFC3339)

	var successCount int
	for _, mod := range modules {
		modDir, downloadErr := downloadModule(mod.Path, mod.Version)
		if downloadErr != nil {
			fmt.Fprintf(os.Stderr, "SKIP %s@%s: download failed: %v\n", mod.Path, mod.Version, downloadErr)
			continue
		}

		pkg, extractErr := extractor.ExtractSinglePackage(modDir, mod.Path)
		if extractErr != nil {
			fmt.Fprintf(os.Stderr, "SKIP %s@%s: extraction failed: %v\n", mod.Path, mod.Version, extractErr)
			continue
		}

		jsonBytes, marshalErr := json.MarshalIndent(pkg, "", "  ")
		if marshalErr != nil {
			fmt.Fprintf(os.Stderr, "SKIP %s: marshal failed: %v\n", mod.Path, marshalErr)
			continue
		}

		encoded := encodeModulePath(mod.Path)
		outputFile := filepath.Join(*outputDir, encoded+".json")
		if writeErr := os.WriteFile(outputFile, jsonBytes, 0o644); writeErr != nil {
			fmt.Fprintf(os.Stderr, "SKIP %s: write failed: %v\n", mod.Path, writeErr)
			continue
		}

		hash := sha256.Sum256(jsonBytes)
		checksum := "sha256:" + hex.EncodeToString(hash[:])

		manifest.Packages = append(manifest.Packages, &core.GoPackageEntry{
			ImportPath:    mod.Path,
			Checksum:      checksum,
			FileSize:      int64(len(jsonBytes)),
			FunctionCount: len(pkg.Functions),
			TypeCount:     len(pkg.Types),
			ConstantCount: len(pkg.Constants),
		})

		successCount++
		fmt.Printf("OK %s@%s: %d types, %d functions -> %s\n",
			mod.Path, mod.Version, len(pkg.Types), len(pkg.Functions), outputFile)
	}

	// Sort manifest packages alphabetically for deterministic output.
	sort.Slice(manifest.Packages, func(i, j int) bool {
		return manifest.Packages[i].ImportPath < manifest.Packages[j].ImportPath
	})

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling manifest: %v\n", err)
		os.Exit(1)
	}
	manifestFile := filepath.Join(*outputDir, "manifest.json")
	if err := os.WriteFile(manifestFile, manifestBytes, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing manifest: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nGenerated manifest: %d/%d packages succeeded -> %s\n",
		successCount, len(modules), manifestFile)
}

// downloadModule runs "go mod download -json module@version" and returns the
// local directory path where the module source is cached by the Go toolchain.
func downloadModule(modulePath, version string) (string, error) {
	output, err := cmdRunner("go", "mod", "download", "-json", modulePath+"@"+version)
	if err != nil {
		return "", fmt.Errorf("go mod download %s@%s: %w", modulePath, version, err)
	}

	//nolint:tagliatelle // "Dir" is the literal field name in `go mod download -json` output.
	var result struct {
		Dir string `json:"Dir"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return "", fmt.Errorf("parsing go mod download output for %s@%s: %w", modulePath, version, err)
	}

	if result.Dir == "" {
		return "", fmt.Errorf("no Dir in go mod download output for %s@%s", modulePath, version)
	}

	return result.Dir, nil
}

// readPackageList reads a file with "module@version" lines.
// Lines starting with "#" are treated as comments and skipped. Empty lines are skipped.
// Returns an error if any non-empty, non-comment line does not have the "module@version" format.
func readPackageList(filename string) ([]moduleSpec, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading packages file %s: %w", filename, err)
	}

	var modules []moduleSpec
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "@", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line (expected module@version): %q", line)
		}
		modules = append(modules, moduleSpec{
			Path:    parts[0],
			Version: parts[1],
		})
	}
	return modules, nil
}

// encodeModulePath encodes a Go module import path for use as a CDN filename.
// Slashes are replaced with underscores, consistent with PR-06's GoThirdPartyRegistryRemote.
//
// Examples:
//
//	"gorm.io/gorm"               → "gorm.io_gorm"
//	"github.com/gin-gonic/gin"   → "github.com_gin-gonic_gin"
//	"github.com/jackc/pgx/v5"    → "github.com_jackc_pgx_v5"
func encodeModulePath(modulePath string) string {
	return strings.ReplaceAll(modulePath, "/", "_")
}
