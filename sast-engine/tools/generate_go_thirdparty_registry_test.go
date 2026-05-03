//go:build cpf_generate_thirdparty_registry

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/tools/internal/goextract"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// readPackageList
// ---------------------------------------------------------------------------

func TestReadPackageList_Valid(t *testing.T) {
	content := `# Web frameworks
github.com/gin-gonic/gin@v1.10.0
gorm.io/gorm@v1.25.12
`
	f := writeTempFile(t, content)
	modules, err := readPackageList(f)
	require.NoError(t, err)
	require.Len(t, modules, 2)
	assert.Equal(t, "github.com/gin-gonic/gin", modules[0].Path)
	assert.Equal(t, "v1.10.0", modules[0].Version)
	assert.Equal(t, "gorm.io/gorm", modules[1].Path)
	assert.Equal(t, "v1.25.12", modules[1].Version)
}

func TestReadPackageList_CommentsAndBlanksSkipped(t *testing.T) {
	content := `
# comment line

  # indented comment
github.com/pkg/errors@v0.9.1

`
	f := writeTempFile(t, content)
	modules, err := readPackageList(f)
	require.NoError(t, err)
	require.Len(t, modules, 1)
	assert.Equal(t, "github.com/pkg/errors", modules[0].Path)
	assert.Equal(t, "v0.9.1", modules[0].Version)
}

func TestReadPackageList_EmptyFile(t *testing.T) {
	f := writeTempFile(t, "")
	modules, err := readPackageList(f)
	require.NoError(t, err)
	assert.Empty(t, modules)
}

func TestReadPackageList_MalformedLine(t *testing.T) {
	content := "github.com/gin-gonic/gin\n" // missing @version
	f := writeTempFile(t, content)
	_, err := readPackageList(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid line")
}

func TestReadPackageList_NonExistentFile(t *testing.T) {
	_, err := readPackageList("/nonexistent/path/packages.txt")
	require.Error(t, err)
}

func TestReadPackageList_VersionWithAt(t *testing.T) {
	// Versions can technically contain @ in pseudo-versions; SplitN(2) handles this.
	content := "github.com/pkg/errors@v0.9.1-0.20210430015257-a9b15e44dba1\n"
	f := writeTempFile(t, content)
	modules, err := readPackageList(f)
	require.NoError(t, err)
	require.Len(t, modules, 1)
	assert.Equal(t, "github.com/pkg/errors", modules[0].Path)
	// Version includes everything after the first @.
	assert.Equal(t, "v0.9.1-0.20210430015257-a9b15e44dba1", modules[0].Version)
}

// ---------------------------------------------------------------------------
// encodeModulePath
// ---------------------------------------------------------------------------

func TestEncodeModulePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"gorm.io/gorm", "gorm.io_gorm"},
		{"github.com/gin-gonic/gin", "github.com_gin-gonic_gin"},
		{"github.com/jackc/pgx/v5", "github.com_jackc_pgx_v5"},
		{"google.golang.org/grpc", "google.golang.org_grpc"},
		{"gopkg.in/yaml.v3", "gopkg.in_yaml.v3"},
		{"k8s.io/client-go", "k8s.io_client-go"},
		{"go.uber.org/zap", "go.uber.org_zap"},
		{"fmt", "fmt"},                   // single segment — no slashes
		{"net/http", "net_http"},         // two segments
		{"", ""},                         // empty
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, encodeModulePath(tt.input))
		})
	}
}

// ---------------------------------------------------------------------------
// downloadModule
// ---------------------------------------------------------------------------

func TestDownloadModule_CommandError(t *testing.T) {
	// Replace cmdRunner with one that always fails.
	orig := cmdRunner
	defer func() { cmdRunner = orig }()
	cmdRunner = func(_ string, _ ...string) ([]byte, error) {
		return nil, os.ErrNotExist
	}

	_, err := downloadModule("github.com/gin-gonic/gin", "v1.10.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "go mod download")
}

func TestDownloadModule_InvalidJSON(t *testing.T) {
	orig := cmdRunner
	defer func() { cmdRunner = orig }()
	cmdRunner = func(_ string, _ ...string) ([]byte, error) {
		return []byte("not-json"), nil
	}

	_, err := downloadModule("github.com/gin-gonic/gin", "v1.10.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing go mod download output")
}

func TestDownloadModule_EmptyDir(t *testing.T) {
	orig := cmdRunner
	defer func() { cmdRunner = orig }()
	result, _ := json.Marshal(goModDownloadResult{Dir: ""})
	cmdRunner = func(_ string, _ ...string) ([]byte, error) {
		return result, nil
	}

	_, err := downloadModule("github.com/gin-gonic/gin", "v1.10.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Dir")
}

func TestDownloadModule_Success(t *testing.T) {
	fakeDir := t.TempDir()
	orig := cmdRunner
	defer func() { cmdRunner = orig }()
	result, _ := json.Marshal(goModDownloadResult{Dir: fakeDir})
	cmdRunner = func(_ string, _ ...string) ([]byte, error) {
		return result, nil
	}

	dir, err := downloadModule("github.com/gin-gonic/gin", "v1.10.0")
	require.NoError(t, err)
	assert.Equal(t, fakeDir, dir)
}

// ---------------------------------------------------------------------------
// Integration: full pipeline with a synthetic package directory
// ---------------------------------------------------------------------------

func TestMain_EndToEnd(t *testing.T) {
	// Build a fake module directory that looks like a downloaded module.
	modDir := t.TempDir()
	src := `package mypkg

// Client is an HTTP client.
type Client struct{}

// Get performs a GET request.
func (c *Client) Get(url string) (string, error) { return "", nil }

// New creates a new Client.
func New() *Client { return &Client{} }
`
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "client.go"), []byte(src), 0o644))

	// Wire up cmdRunner to return the fake module dir.
	orig := cmdRunner
	defer func() { cmdRunner = orig }()
	result, _ := json.Marshal(goModDownloadResult{Dir: modDir})
	cmdRunner = func(_ string, _ ...string) ([]byte, error) {
		return result, nil
	}

	// Build a packages file.
	pkgFile := writeTempFile(t, "example.com/myhttp@v1.0.0\n")

	// Set output dir.
	outDir := t.TempDir()

	// Run main-equivalent logic.
	modules, err := readPackageList(pkgFile)
	require.NoError(t, err)

	dir, err := downloadModule(modules[0].Path, modules[0].Version)
	require.NoError(t, err)

	extractor := newTestExtractor()
	pkg, err := extractor.ExtractSinglePackage(dir, modules[0].Path)
	require.NoError(t, err)
	assert.Contains(t, pkg.Types, "Client")
	assert.Contains(t, pkg.Functions, "New")

	// Verify the encoded file name.
	encoded := encodeModulePath(modules[0].Path)
	assert.Equal(t, "example.com_myhttp", encoded)

	// Write the JSON.
	jsonBytes, err := json.MarshalIndent(pkg, "", "  ")
	require.NoError(t, err)
	outFile := filepath.Join(outDir, encoded+".json")
	require.NoError(t, os.WriteFile(outFile, jsonBytes, 0o644))

	// Verify it can be read back.
	data, err := os.ReadFile(outFile)
	require.NoError(t, err)
	// Verify import_path field is present (snake_case matches goextract.Package JSON tags).
	//nolint:tagliatelle // import_path matches the goextract.Package JSON schema (snake_case).
	var parsed struct {
		ImportPath string `json:"import_path"`
	}
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.Equal(t, "example.com/myhttp", parsed.ImportPath)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// goModDownloadResult mirrors the JSON output of `go mod download -json`.
//
//nolint:tagliatelle // "Dir" matches the literal field in `go mod download -json` output.
type goModDownloadResult struct {
	Dir string `json:"Dir"`
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "pkglist-*.txt")
	require.NoError(t, err)
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func newTestExtractor() *goextract.Extractor {
	return goextract.NewExtractor(goextract.Config{})
}
