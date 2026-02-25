package resolution

import (
	"errors"
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errMockResolutionNotFound = errors.New("not found in mock")

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
				StdlibPackages: make(map[string]bool),
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

// ============================================================================
// mockResolutionStdlibLoader — in-package mock for GoImportResolver tests
// ============================================================================

type mockResolutionStdlibLoader struct {
	packages map[string]bool
}

func (m *mockResolutionStdlibLoader) ValidateStdlibImport(importPath string) bool {
	return m.packages[importPath]
}

func (m *mockResolutionStdlibLoader) GetFunction(_, _ string) (*core.GoStdlibFunction, error) {
	return nil, errMockResolutionNotFound
}

func (m *mockResolutionStdlibLoader) GetType(_, _ string) (*core.GoStdlibType, error) {
	return nil, errMockResolutionNotFound
}

func (m *mockResolutionStdlibLoader) PackageCount() int {
	return len(m.packages)
}

func newMockResolutionLoader(pkgs ...string) *mockResolutionStdlibLoader {
	m := &mockResolutionStdlibLoader{packages: make(map[string]bool, len(pkgs))}
	for _, p := range pkgs {
		m.packages[p] = true
	}
	return m
}

// ============================================================================
// TestGoImportResolver tests
// ============================================================================

func TestGoImportResolver_NilRegistry(t *testing.T) {
	r := NewGoImportResolver(nil)
	// Should not panic
	assert.Equal(t, ImportThirdParty, r.ClassifyImport("github.com/gorilla/mux"))
	assert.Equal(t, ImportStdlib, r.ClassifyImport("fmt"))
	assert.Equal(t, ImportLocal, r.ClassifyImport("./utils"))
}

func TestGoImportResolver_isStdlibImportFallback(t *testing.T) {
	r := NewGoImportResolver(nil)
	tests := []struct {
		importPath string
		want       bool
	}{
		{"fmt", true},
		{"os", true},
		{"net/http", true},
		{"encoding/json", true},
		{"database/sql", true},
		{"github.com/gorilla/mux", false}, // has dot
		{"gopkg.in/yaml.v2", false},       // has dot
		{"internal/foo", false},           // internal prefix
		{"internal/trace", false},
	}

	for _, tt := range tests {
		t.Run(tt.importPath, func(t *testing.T) {
			got := r.isStdlibImportFallback(tt.importPath)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGoImportResolver_isStdlibImport_WithLoader(t *testing.T) {
	reg := core.NewGoModuleRegistry()
	reg.StdlibLoader = newMockResolutionLoader("fmt", "net/http")
	r := NewGoImportResolver(reg)

	assert.True(t, r.isStdlibImport("fmt"))
	assert.True(t, r.isStdlibImport("net/http"))
	// "os" not in mock → false (loader is authoritative)
	assert.False(t, r.isStdlibImport("os"))
	assert.False(t, r.isStdlibImport("github.com/gorilla/mux"))
}

func TestGoImportResolver_isStdlibImport_NilLoader_FallsBackToHeuristic(t *testing.T) {
	reg := core.NewGoModuleRegistry()
	reg.StdlibLoader = nil // no loader → fallback
	r := NewGoImportResolver(reg)

	// Heuristic: no dot in path → stdlib
	assert.True(t, r.isStdlibImport("fmt"))
	assert.True(t, r.isStdlibImport("net/http"))
	assert.False(t, r.isStdlibImport("github.com/gorilla/mux"))
}

func TestGoImportResolver_ClassifyImport_Stdlib(t *testing.T) {
	reg := core.NewGoModuleRegistry()
	reg.ModulePath = "github.com/example/myapp"
	reg.StdlibLoader = newMockResolutionLoader("fmt", "net/http", "encoding/json")
	r := NewGoImportResolver(reg)

	assert.Equal(t, ImportStdlib, r.ClassifyImport("fmt"))
	assert.Equal(t, ImportStdlib, r.ClassifyImport("net/http"))
	assert.Equal(t, ImportStdlib, r.ClassifyImport("encoding/json"))
}

func TestGoImportResolver_ClassifyImport_ThirdParty(t *testing.T) {
	reg := core.NewGoModuleRegistry()
	reg.ModulePath = "github.com/example/myapp"
	reg.StdlibLoader = newMockResolutionLoader("fmt")
	r := NewGoImportResolver(reg)

	assert.Equal(t, ImportThirdParty, r.ClassifyImport("github.com/gorilla/mux"))
	assert.Equal(t, ImportThirdParty, r.ClassifyImport("gopkg.in/yaml.v2"))
	assert.Equal(t, ImportThirdParty, r.ClassifyImport("github.com/lib/pq"))
}

func TestGoImportResolver_ClassifyImport_Local_RelativePath(t *testing.T) {
	reg := core.NewGoModuleRegistry()
	reg.ModulePath = "github.com/example/myapp"
	r := NewGoImportResolver(reg)

	assert.Equal(t, ImportLocal, r.ClassifyImport("./utils"))
	assert.Equal(t, ImportLocal, r.ClassifyImport("../models"))
	assert.Equal(t, ImportLocal, r.ClassifyImport("."))
}

func TestGoImportResolver_ClassifyImport_Local_SameModule(t *testing.T) {
	reg := core.NewGoModuleRegistry()
	reg.ModulePath = "github.com/example/myapp"
	reg.StdlibLoader = newMockResolutionLoader("fmt")
	r := NewGoImportResolver(reg)

	assert.Equal(t, ImportLocal, r.ClassifyImport("github.com/example/myapp/handlers"))
	assert.Equal(t, ImportLocal, r.ClassifyImport("github.com/example/myapp/utils/validation"))
	// Same prefix but different module
	assert.Equal(t, ImportThirdParty, r.ClassifyImport("github.com/example/otherapp/handlers"))
}

func TestGoImportResolver_ResolveImports(t *testing.T) {
	reg := core.NewGoModuleRegistry()
	reg.ModulePath = "github.com/example/myapp"
	reg.StdlibLoader = newMockResolutionLoader("fmt", "net/http")
	r := NewGoImportResolver(reg)

	imports := []string{
		"fmt",
		"net/http",
		"github.com/gorilla/mux",
		"github.com/example/myapp/handlers",
		"./utils",
	}

	result := r.ResolveImports(imports)

	assert.Equal(t, ImportStdlib, result["fmt"])
	assert.Equal(t, ImportStdlib, result["net/http"])
	assert.Equal(t, ImportThirdParty, result["github.com/gorilla/mux"])
	assert.Equal(t, ImportLocal, result["github.com/example/myapp/handlers"])
	assert.Equal(t, ImportLocal, result["./utils"])
}

func TestGoImportResolver_ResolveImports_Empty(t *testing.T) {
	r := NewGoImportResolver(nil)
	result := r.ResolveImports([]string{})
	assert.Empty(t, result)
}
