package builder

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestLogger returns a silent logger suitable for unit tests.
func newGoVersionTestLogger() *output.Logger {
	return output.NewLogger(output.VerbosityDefault)
}

// writeTempFile writes content to a file inside dir.
func writeTempFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
}

// -----------------------------------------------------------------------------
// normalizeGoVersion
// -----------------------------------------------------------------------------

func TestNormalizeGoVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1.21", "1.21"},
		{"1.21.4", "1.21"},
		{"1.26.0", "1.26"},
		{"2.0.0", "2.0"},
		{"1", "1"},   // single component — returned as-is
		{"", ""},     // empty — returned as-is
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, normalizeGoVersion(tt.input))
		})
	}
}

// -----------------------------------------------------------------------------
// parseGoVersionFromFile
// -----------------------------------------------------------------------------

func TestParseGoVersionFromFile_GoMod(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.22\n")
	assert.Equal(t, "1.22", parseGoVersionFromFile(filepath.Join(dir, "go.mod")))
}

func TestParseGoVersionFromFile_GoModWithPatch(t *testing.T) {
	dir := t.TempDir()
	// go.mod may contain the full toolchain line; the regex captures only X.Y.
	writeTempFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.23\n\ntoolchain go1.23.4\n")
	assert.Equal(t, "1.23", parseGoVersionFromFile(filepath.Join(dir, "go.mod")))
}

func TestParseGoVersionFromFile_MissingFile(t *testing.T) {
	assert.Equal(t, "", parseGoVersionFromFile("/nonexistent/go.mod"))
}

func TestParseGoVersionFromFile_NoGoDirective(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", "module example.com/app\n")
	assert.Equal(t, "", parseGoVersionFromFile(filepath.Join(dir, "go.mod")))
}

// -----------------------------------------------------------------------------
// readGoVersionFile
// -----------------------------------------------------------------------------

func TestReadGoVersionFile_SimpleVersion(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, ".go-version", "1.22.4\n")
	assert.Equal(t, "1.22", readGoVersionFile(dir))
}

func TestReadGoVersionFile_AlreadyNormalised(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, ".go-version", "1.21")
	assert.Equal(t, "1.21", readGoVersionFile(dir))
}

func TestReadGoVersionFile_MissingFile(t *testing.T) {
	assert.Equal(t, "", readGoVersionFile(t.TempDir()))
}

func TestReadGoVersionFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, ".go-version", "   \n")
	// normalizeGoVersion("") returns ""
	assert.Equal(t, "", readGoVersionFile(dir))
}

// -----------------------------------------------------------------------------
// DetectGoVersion — priority chain
// -----------------------------------------------------------------------------

func TestDetectGoVersion_GoMod(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.21\n")
	assert.Equal(t, "1.21", DetectGoVersion(dir))
}

func TestDetectGoVersion_GoModStripsPatch(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.26\n")
	assert.Equal(t, "1.26", DetectGoVersion(dir))
}

func TestDetectGoVersion_GoVersionFile(t *testing.T) {
	dir := t.TempDir()
	// No go.mod → falls through to .go-version
	writeTempFile(t, dir, ".go-version", "1.22.4\n")
	assert.Equal(t, "1.22", DetectGoVersion(dir))
}

func TestDetectGoVersion_GoWork(t *testing.T) {
	dir := t.TempDir()
	// No go.mod, no .go-version → falls through to go.work
	writeTempFile(t, dir, "go.work", "go 1.23\n\nuse .\n")
	assert.Equal(t, "1.23", DetectGoVersion(dir))
}

func TestDetectGoVersion_Default(t *testing.T) {
	// Empty directory — no detection files present.
	assert.Equal(t, defaultGoVersion, DetectGoVersion(t.TempDir()))
}

func TestDetectGoVersion_GoModTakesPriorityOverGoVersionFile(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.24\n")
	writeTempFile(t, dir, ".go-version", "1.22.0\n")
	// go.mod wins
	assert.Equal(t, "1.24", DetectGoVersion(dir))
}

func TestDetectGoVersion_GoVersionFileTakesPriorityOverGoWork(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, ".go-version", "1.22.0\n")
	writeTempFile(t, dir, "go.work", "go 1.23\n")
	// .go-version wins over go.work
	assert.Equal(t, "1.22", DetectGoVersion(dir))
}

func TestDetectGoVersion_GoModNoGoDirective(t *testing.T) {
	dir := t.TempDir()
	// go.mod present but has no "go" directive → falls through to .go-version
	writeTempFile(t, dir, "go.mod", "module example.com/app\n")
	writeTempFile(t, dir, ".go-version", "1.20\n")
	assert.Equal(t, "1.20", DetectGoVersion(dir))
}

// -----------------------------------------------------------------------------
// InitGoStdlibLoader / initGoStdlibLoaderWithBase
// -----------------------------------------------------------------------------

// minimalManifest returns the JSON body of a minimal valid manifest.
func minimalManifest() []byte {
	manifest := core.GoManifest{
		SchemaVersion:   "1.0.0",
		RegistryVersion: "v1",
		GoVersion:       core.GoVersionInfo{Major: 1, Minor: 21},
		Packages: []*core.GoPackageEntry{
			{ImportPath: "fmt"},
			{ImportPath: "os"},
		},
	}
	data, _ := json.Marshal(manifest)
	return data
}

func TestInitGoStdlibLoader_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(minimalManifest())
	}))
	defer server.Close()

	reg := core.NewGoModuleRegistry()
	reg.GoVersion = "1.21"
	logger := newGoVersionTestLogger()

	initGoStdlibLoaderWithBase(reg, t.TempDir(), logger, server.URL)

	require.NotNil(t, reg.StdlibLoader)
	assert.Equal(t, 2, reg.StdlibLoader.PackageCount())
	assert.True(t, reg.StdlibLoader.ValidateStdlibImport("fmt"))
}

func TestInitGoStdlibLoader_ManifestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer server.Close()

	reg := core.NewGoModuleRegistry()
	reg.GoVersion = "1.21"
	logger := newGoVersionTestLogger()

	initGoStdlibLoaderWithBase(reg, t.TempDir(), logger, server.URL)

	// Graceful degradation: StdlibLoader must remain nil.
	assert.Nil(t, reg.StdlibLoader)
}

func TestInitGoStdlibLoader_EmptyRegistryVersion_FallsBackToDetect(t *testing.T) {
	// reg.GoVersion is empty → InitGoStdlibLoader must call DetectGoVersion.
	dir := t.TempDir()
	writeTempFile(t, dir, "go.mod", "module example.com/app\n\ngo 1.23\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that the detected version (1.23) is used in the URL.
		assert.Contains(t, r.URL.Path, "go1.23")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(minimalManifest())
	}))
	defer server.Close()

	reg := core.NewGoModuleRegistry()
	// GoVersion intentionally empty — must be detected from go.mod.
	logger := newGoVersionTestLogger()

	initGoStdlibLoaderWithBase(reg, dir, logger, server.URL)

	require.NotNil(t, reg.StdlibLoader)
}

func TestInitGoStdlibLoader_VersionNormalized(t *testing.T) {
	// reg.GoVersion has a patch suffix ("1.21.4") — must be stripped to "1.21".
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(minimalManifest())
	}))
	defer server.Close()

	reg := core.NewGoModuleRegistry()
	reg.GoVersion = "1.21.4"
	logger := newGoVersionTestLogger()

	initGoStdlibLoaderWithBase(reg, t.TempDir(), logger, server.URL)

	assert.Contains(t, capturedPath, "go1.21")
	assert.NotContains(t, capturedPath, "1.21.4")
	require.NotNil(t, reg.StdlibLoader)
}

func TestInitGoStdlibLoader_NetworkError(t *testing.T) {
	// Point at a URL that refuses connections.
	reg := core.NewGoModuleRegistry()
	reg.GoVersion = "1.21"
	logger := newGoVersionTestLogger()

	initGoStdlibLoaderWithBase(reg, t.TempDir(), logger, "http://127.0.0.1:0")

	assert.Nil(t, reg.StdlibLoader)
}

func TestInitGoStdlibLoader_PublicAPI_CallsInner(t *testing.T) {
	// Verify that the public InitGoStdlibLoader function is reachable.
	// We override stdlibRegistryBaseURL so it hits a local server instead of CDN.
	original := stdlibRegistryBaseURL
	t.Cleanup(func() { stdlibRegistryBaseURL = original })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(minimalManifest())
	}))
	defer server.Close()

	stdlibRegistryBaseURL = server.URL

	reg := core.NewGoModuleRegistry()
	reg.GoVersion = "1.21"
	logger := newGoVersionTestLogger()

	InitGoStdlibLoader(reg, t.TempDir(), logger)

	require.NotNil(t, reg.StdlibLoader)
}
