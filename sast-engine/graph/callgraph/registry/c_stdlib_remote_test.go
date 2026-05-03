package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeRegistry materialises a manifest plus per-header JSON files in dir
// and returns the dir. Reused by both C and C++ loader tests via copies in
// each test file.
func writeCRegistry(t *testing.T, dir string) {
	t.Helper()

	stdio := &core.CStdlibHeader{
		SchemaVersion: "1.0.0",
		Header:        "stdio.h",
		ModuleID:      "c::stdio",
		Language:      core.LanguageC,
		Platform:      core.PlatformLinux,
		Functions: map[string]*core.CStdlibFunction{
			"printf": {
				FQN:        "c::stdio::printf",
				ReturnType: "int",
				Source:     core.SourceOverlay,
				Confidence: 1.0,
			},
		},
	}
	stdioBytes, err := json.Marshal(stdio)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stdio_stdlib.json"), stdioBytes, 0o644))

	stdlib := &core.CStdlibHeader{
		SchemaVersion: "1.0.0",
		Header:        "stdlib.h",
		ModuleID:      "c::stdlib",
		Language:      core.LanguageC,
		Functions: map[string]*core.CStdlibFunction{
			"malloc": {FQN: "c::stdlib::malloc", ReturnType: "void*", Source: core.SourceHeader, Confidence: 1.0},
			"system": {
				FQN: "c::stdlib::system", ReturnType: "int",
				SecurityTag: "command_injection_sink",
				Source:      core.SourceOverlay, Confidence: 1.0,
			},
		},
	}
	stdlibBytes, err := json.Marshal(stdlib)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stdlib_stdlib.json"), stdlibBytes, 0o644))

	manifest := &core.CStdlibManifest{
		SchemaVersion:   "1.0.0",
		RegistryVersion: "v1",
		Platform:        core.PlatformLinux,
		Language:        core.LanguageC,
		Headers: []*core.CStdlibHeaderEntry{
			{Header: "stdio.h", ModuleID: "c::stdio", File: "stdio_stdlib.json"},
			{Header: "stdlib.h", ModuleID: "c::stdlib", File: "stdlib_stdlib.json"},
		},
		Statistics: &core.CStdlibStatistics{TotalHeaders: 2, TotalFunctions: 3},
	}
	mBytes, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), mBytes, 0o644))
}

// noopLogger satisfies core.CStdlibLogger without writing anywhere.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any)     {}
func (noopLogger) Statistic(string, ...any) {}
func (noopLogger) Warning(string, ...any)   {}

func TestCStdlibRegistry_FileMode_LoadManifest(t *testing.T) {
	dir := t.TempDir()
	writeCRegistry(t, dir)

	r := NewCStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))

	assert.Equal(t, core.PlatformLinux, r.Platform())
	assert.Equal(t, 2, r.HeaderCount())
}

func TestCStdlibRegistry_FileMode_LoadManifest_NilLogger(t *testing.T) {
	dir := t.TempDir()
	writeCRegistry(t, dir)

	r := NewCStdlibRegistryFile(dir, core.PlatformLinux)
	// nil logger is allowed.
	require.NoError(t, r.LoadManifest(nil))
}

func TestCStdlibRegistry_FileMode_MissingManifest(t *testing.T) {
	r := NewCStdlibRegistryFile(filepath.Join(t.TempDir(), "absent"), core.PlatformLinux)
	err := r.LoadManifest(noopLogger{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading")
}

func TestCStdlibRegistry_FileMode_CorruptManifest(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("{not json"), 0o644))
	r := NewCStdlibRegistryFile(dir, core.PlatformLinux)
	err := r.LoadManifest(noopLogger{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing")
}

func TestCStdlibRegistry_FileMode_GetHeader(t *testing.T) {
	dir := t.TempDir()
	writeCRegistry(t, dir)

	r := NewCStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))

	h, err := r.GetHeader("stdio.h")
	require.NoError(t, err)
	assert.Equal(t, "stdio.h", h.Header)
	require.Contains(t, h.Functions, "printf")

	// Same header again — should hit cache.
	h2, err := r.GetHeader("stdio.h")
	require.NoError(t, err)
	assert.Same(t, h, h2, "second GetHeader should return cached pointer")
}

func TestCStdlibRegistry_FileMode_GetHeaderUnknown(t *testing.T) {
	dir := t.TempDir()
	writeCRegistry(t, dir)
	r := NewCStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))

	_, err := r.GetHeader("missing.h")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in stdlib manifest")
}

func TestCStdlibRegistry_FileMode_GetHeader_BeforeLoad(t *testing.T) {
	r := NewCStdlibRegistryFile(t.TempDir(), core.PlatformLinux)
	_, err := r.GetHeader("stdio.h")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest not loaded")
}

func TestCStdlibRegistry_FileMode_GetHeaderMissingFile(t *testing.T) {
	dir := t.TempDir()
	writeCRegistry(t, dir)
	// Delete the per-header file; manifest still references it.
	require.NoError(t, os.Remove(filepath.Join(dir, "stdio_stdlib.json")))

	r := NewCStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))
	_, err := r.GetHeader("stdio.h")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading")
}

func TestCStdlibRegistry_FileMode_GetHeaderCorruptFile(t *testing.T) {
	dir := t.TempDir()
	writeCRegistry(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stdio_stdlib.json"), []byte("garbage"), 0o644))

	r := NewCStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))
	_, err := r.GetHeader("stdio.h")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing")
}

func TestCStdlibRegistry_FileMode_GetFunction(t *testing.T) {
	dir := t.TempDir()
	writeCRegistry(t, dir)
	r := NewCStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))

	got, err := r.GetFunction("stdio.h", "printf")
	require.NoError(t, err)
	assert.Equal(t, "c::stdio::printf", got.FQN)

	// Function in a different header.
	got, err = r.GetFunction("stdlib.h", "system")
	require.NoError(t, err)
	assert.Equal(t, "command_injection_sink", got.SecurityTag)

	// Missing function in present header.
	_, err = r.GetFunction("stdio.h", "scanf")
	require.Error(t, err)

	// Missing header.
	_, err = r.GetFunction("missing.h", "x")
	require.Error(t, err)
}

func TestCStdlibRegistry_HeaderCount_BeforeLoad(t *testing.T) {
	r := NewCStdlibRegistryFile(t.TempDir(), core.PlatformLinux)
	assert.Equal(t, 0, r.HeaderCount())
}

func TestCStdlibRegistry_DoubleCheckLocking(t *testing.T) {
	dir := t.TempDir()
	writeCRegistry(t, dir)
	r := NewCStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))

	// Hammer the loader from many goroutines and confirm no panic / no
	// data corruption.
	var wg sync.WaitGroup
	const workers = 100
	results := make([]*core.CStdlibHeader, workers)
	for i := range workers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			h, err := r.GetHeader("stdio.h")
			if err == nil {
				results[idx] = h
			}
		}(i)
	}
	wg.Wait()

	// All goroutines should see the same cached pointer.
	for i := 1; i < workers; i++ {
		assert.Same(t, results[0], results[i], "worker %d saw different pointer", i)
	}
}

func TestCStdlibRegistry_HTTPMode_Stub(t *testing.T) {
	r := NewCStdlibRegistryRemote("https://example.com/registries", core.PlatformLinux)
	err := r.LoadManifest(noopLogger{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PR-03")
}

func TestCStdlibRegistry_HTTPMode_FetchHeaderStub(t *testing.T) {
	// Construct the HTTP loader with an in-memory manifest by going through
	// a file:// loader first, then forcing the fetch path to HTTP.
	dir := t.TempDir()
	writeCRegistry(t, dir)
	r := NewCStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))

	// Switch to HTTP mode mid-flight by clearing fileBase. Tests-only —
	// production code never does this.
	r.fileBase = ""
	_, err := r.GetHeader("stdio.h") // fresh header (not yet cached)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "PR-03")
}

func TestCStdlibRegistry_RemoteCtor_TrimsTrailingSlash(t *testing.T) {
	r := NewCStdlibRegistryRemote("https://example.com/registries/", core.PlatformLinux)
	assert.Equal(t, "https://example.com/registries", r.baseURL)
}

func TestCStdlibRegistry_ImplementsInterface(t *testing.T) {
	var _ core.CStdlibLoader = NewCStdlibRegistryFile(t.TempDir(), core.PlatformLinux)
	var _ core.CStdlibLoader = NewCStdlibRegistryRemote("https://x", core.PlatformLinux)
}
