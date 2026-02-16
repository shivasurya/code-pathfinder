package resolution

import (
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildGoModuleRegistry(t *testing.T) {
	tests := []struct {
		name         string
		projectRoot  string
		wantModule   string
		wantMappings map[string]string // relative dir → import path
		wantErr      bool
	}{
		{
			name:        "simple project with go.mod",
			projectRoot: "../../../test-fixtures/golang/module_project",
			wantModule:  "github.com/example/testapp",
			wantMappings: map[string]string{
				".":                    "github.com/example/testapp",
				"handlers":             "github.com/example/testapp/handlers",
				"models":               "github.com/example/testapp/models",
				"utils":                "github.com/example/testapp/utils",
				"utils/validation":     "github.com/example/testapp/utils/validation",
			},
			wantErr: false,
		},
		{
			name:        "missing go.mod",
			projectRoot: "/nonexistent/path",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry, err := BuildGoModuleRegistry(tt.projectRoot)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantModule, registry.ModulePath)

			// Verify key mappings exist
			for relDir, expectedImport := range tt.wantMappings {
				// Build check: find directory with suffix matching relDir
				found := false
				for dir, importPath := range registry.DirToImport {
					// Handle both "." and subdirectories
					if relDir == "." {
						// Root directory: import path should equal module path
						if importPath == expectedImport {
							found = true
							break
						}
					} else {
						// Subdirectory: check if path ends with relDir (with path separator)
						if (strings.HasSuffix(dir, "/"+relDir) || strings.HasSuffix(dir, relDir)) && importPath == expectedImport {
							found = true
							break
						}
					}
				}
				assert.True(t, found, "Expected mapping for %s -> %s", relDir, expectedImport)

				// Verify reverse mapping exists
				_, ok := registry.ImportToDir[expectedImport]
				assert.True(t, ok, "Expected reverse mapping for %s", expectedImport)
			}

			// Verify stdlib packages loaded
			assert.True(t, registry.StdlibPackages["fmt"])
			assert.True(t, registry.StdlibPackages["net/http"])
			assert.True(t, registry.StdlibPackages["encoding/json"])
			assert.False(t, registry.StdlibPackages["nonexistent"])
		})
	}
}

func TestExtractGoImports(t *testing.T) {
	tests := []struct {
		name            string
		sourceCode      string
		wantPackageName string
		wantImports     map[string]string
	}{
		{
			name: "simple imports",
			sourceCode: `package main

import "fmt"
import "os"

func main() {}
`,
			wantPackageName: "main",
			wantImports: map[string]string{
				"fmt": "fmt",
				"os":  "os",
			},
		},
		{
			name: "grouped imports with aliases",
			sourceCode: `package handlers

import (
	"fmt"
	h "net/http"
	. "github.com/example/utils"
	_ "github.com/lib/pq"
)

func Handle() {}
`,
			wantPackageName: "handlers",
			wantImports: map[string]string{
				"fmt": "fmt",
				"h":   "net/http",
				".":   "github.com/example/utils",
				"_":   "github.com/lib/pq",
			},
		},
		{
			name: "nested package paths",
			sourceCode: `package models

import (
	"github.com/example/testapp/handlers"
	"github.com/example/testapp/utils/validation"
)
`,
			wantPackageName: "models",
			wantImports: map[string]string{
				"handlers":   "github.com/example/testapp/handlers",
				"validation": "github.com/example/testapp/utils/validation",
			},
		},
		{
			name: "mixed stdlib and third-party",
			sourceCode: `package server

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)
`,
			wantPackageName: "server",
			wantImports: map[string]string{
				"context": "context",
				"sql":     "database/sql",
				"http":    "net/http",
				"mux":     "github.com/gorilla/mux",
				"_":       "github.com/lib/pq",
			},
		},
		{
			name: "no imports",
			sourceCode: `package types

type User struct {
	ID   int
	Name string
}
`,
			wantPackageName: "types",
			wantImports:     map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create dummy registry (not used in extraction but required by signature)
			registry := &core.GoModuleRegistry{
				ModulePath:     "github.com/example/test",
				DirToImport:    make(map[string]string),
				ImportToDir:    make(map[string]string),
				StdlibPackages: goStdlibSet(),
			}

			importMap, err := ExtractGoImports("/tmp/test.go", []byte(tt.sourceCode), registry)

			require.NoError(t, err)
			assert.Equal(t, tt.wantPackageName, importMap.PackageName)
			assert.Equal(t, tt.wantImports, importMap.Imports)
		})
	}
}

func TestExtractLocalName(t *testing.T) {
	tests := []struct {
		importPath string
		want       string
	}{
		{"fmt", "fmt"},
		{"net/http", "http"},
		{"github.com/example/myapp/handlers", "handlers"},
		{"github.com/gorilla/mux", "mux"},
		{"database/sql", "sql"},
		{"encoding/json", "json"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.importPath, func(t *testing.T) {
			got := extractLocalName(tt.importPath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGoStdlibSet(t *testing.T) {
	stdlib := goStdlibSet()

	// Verify common packages
	assert.True(t, stdlib["fmt"])
	assert.True(t, stdlib["os"])
	assert.True(t, stdlib["io"])
	assert.True(t, stdlib["net/http"])
	assert.True(t, stdlib["encoding/json"])
	assert.True(t, stdlib["database/sql"])
	assert.True(t, stdlib["context"])

	// Verify Go 1.21+ packages
	assert.True(t, stdlib["slices"])
	assert.True(t, stdlib["maps"])
	assert.True(t, stdlib["cmp"])
	assert.True(t, stdlib["log/slog"])

	// Verify crypto packages
	assert.True(t, stdlib["crypto/sha256"])
	assert.True(t, stdlib["crypto/tls"])

	// Verify non-stdlib is not included
	assert.False(t, stdlib["github.com/example/myapp"])
	assert.False(t, stdlib["github.com/gorilla/mux"])
	assert.False(t, stdlib["gopkg.in/yaml.v2"])

	// Verify minimum count
	assert.GreaterOrEqual(t, len(stdlib), 100, "Should have at least 100 stdlib packages")
}

func TestShouldSkipGoDirectory(t *testing.T) {
	tests := []struct {
		dirName string
		want    bool
	}{
		{"vendor", true},
		{"testdata", true},
		{".git", true},
		{".svn", true},
		{"node_modules", true},
		{"dist", true},
		{"build", true},
		{".vscode", true},
		{".idea", true},
		{"tmp", true},
		{"handlers", false},
		{"models", false},
		{"utils", false},
		{"internal", false},
	}

	for _, tt := range tests {
		t.Run(tt.dirName, func(t *testing.T) {
			got := shouldSkipGoDirectory(tt.dirName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseGoMod(t *testing.T) {
	tests := []struct {
		name        string
		projectRoot string
		wantModule  string
		wantErr     bool
	}{
		{
			name:        "valid go.mod",
			projectRoot: "../../../test-fixtures/golang/module_project",
			wantModule:  "github.com/example/testapp",
			wantErr:     false,
		},
		{
			name:        "missing go.mod",
			projectRoot: "/nonexistent/path",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modulePath, _, err := parseGoMod(tt.projectRoot)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantModule, modulePath)
		})
	}
}

func TestGoImportMapMethods(t *testing.T) {
	importMap := core.NewGoImportMap("/tmp/test.go")

	// Test initial state
	assert.Equal(t, "/tmp/test.go", importMap.FilePath)
	assert.Empty(t, importMap.Imports)
	assert.Empty(t, importMap.PackageName)

	// Test AddImport
	importMap.AddImport("fmt", "fmt")
	importMap.AddImport("h", "net/http")

	assert.Equal(t, "fmt", importMap.Imports["fmt"])
	assert.Equal(t, "net/http", importMap.Imports["h"])

	// Test Resolve - existing
	path, ok := importMap.Resolve("h")
	assert.True(t, ok)
	assert.Equal(t, "net/http", path)

	// Test Resolve - non-existing
	path, ok = importMap.Resolve("nonexistent")
	assert.False(t, ok)
	assert.Empty(t, path)
}

func TestGoModuleRegistryMethods(t *testing.T) {
	registry := core.NewGoModuleRegistry()

	// Test initial state
	assert.Empty(t, registry.ModulePath)
	assert.Empty(t, registry.DirToImport)
	assert.Empty(t, registry.ImportToDir)
	assert.Empty(t, registry.StdlibPackages)

	// Set module path
	registry.ModulePath = "github.com/example/test"

	// Add directory mappings
	registry.DirToImport["/project/handlers"] = "github.com/example/test/handlers"
	registry.ImportToDir["github.com/example/test/handlers"] = "/project/handlers"

	// Verify mappings
	importPath, ok := registry.DirToImport["/project/handlers"]
	assert.True(t, ok)
	assert.Equal(t, "github.com/example/test/handlers", importPath)

	dirPath, ok := registry.ImportToDir["github.com/example/test/handlers"]
	assert.True(t, ok)
	assert.Equal(t, "/project/handlers", dirPath)
}
