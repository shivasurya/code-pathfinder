package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeCppRegistry(t *testing.T, dir string) {
	t.Helper()

	vec := &core.CStdlibHeader{
		SchemaVersion: "1.0.0",
		Header:        "vector",
		ModuleID:      "std::vector",
		Language:      core.LanguageCpp,
		Classes: map[string]*core.CppStdlibClass{
			"std::vector": {
				FQN:        "std::vector",
				TypeParams: []string{"T", "Allocator"},
				Methods: map[string]*core.CStdlibFunction{
					"push_back": {FQN: "std::vector::push_back", ReturnType: "void", Source: core.SourceOverlay, Confidence: 1.0},
					"size":      {FQN: "std::vector::size", ReturnType: "size_t", Source: core.SourceHeader, Confidence: 1.0},
				},
			},
		},
		FreeFunctions: map[string]*core.CStdlibFunction{
			"std::swap": {FQN: "std::swap", ReturnType: "void", Source: core.SourceHeader, Confidence: 1.0},
		},
	}
	vBytes, err := json.Marshal(vec)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "vector_stdlib.json"), vBytes, 0o644))

	utility := &core.CStdlibHeader{
		SchemaVersion: "1.0.0",
		Header:        "utility",
		ModuleID:      "std::utility",
		Language:      core.LanguageCpp,
		FreeFunctions: map[string]*core.CStdlibFunction{
			"std::move":    {FQN: "std::move", ReturnType: "T&&", Source: core.SourceOverlay, Confidence: 1.0},
			"std::forward": {FQN: "std::forward", ReturnType: "T&&", Source: core.SourceOverlay, Confidence: 1.0},
		},
	}
	uBytes, err := json.Marshal(utility)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "utility_stdlib.json"), uBytes, 0o644))

	manifest := &core.CStdlibManifest{
		SchemaVersion:   "1.0.0",
		RegistryVersion: "v1",
		Platform:        core.PlatformLinux,
		Language:        core.LanguageCpp,
		Headers: []*core.CStdlibHeaderEntry{
			{Header: "vector", ModuleID: "std::vector", File: "vector_stdlib.json"},
			{Header: "utility", ModuleID: "std::utility", File: "utility_stdlib.json"},
		},
		Statistics: &core.CStdlibStatistics{TotalHeaders: 2, TotalClasses: 1, TotalFunctions: 5},
	}
	mBytes, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), mBytes, 0o644))
}

func TestCppStdlibRegistry_FileMode_LoadAndAccessors(t *testing.T) {
	dir := t.TempDir()
	writeCppRegistry(t, dir)
	r := NewCppStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))
	assert.Equal(t, 2, r.HeaderCount())
	assert.Equal(t, core.PlatformLinux, r.Platform())
}

func TestCppStdlibRegistry_LoadManifestNilLogger(t *testing.T) {
	dir := t.TempDir()
	writeCppRegistry(t, dir)
	r := NewCppStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(nil))
}

func TestCppStdlibRegistry_GetClassAndMethod(t *testing.T) {
	dir := t.TempDir()
	writeCppRegistry(t, dir)
	r := NewCppStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))

	cls, err := r.GetClass("vector", "std::vector")
	require.NoError(t, err)
	assert.Equal(t, []string{"T", "Allocator"}, cls.TypeParams)

	pb, err := r.GetMethod("vector", "std::vector", "push_back")
	require.NoError(t, err)
	assert.Equal(t, "void", pb.ReturnType)

	// Missing method.
	_, err = r.GetMethod("vector", "std::vector", "iterate")
	require.Error(t, err)

	// Missing class.
	_, err = r.GetClass("vector", "std::list")
	require.Error(t, err)

	// Missing class via GetMethod.
	_, err = r.GetMethod("vector", "std::list", "size")
	require.Error(t, err)
}

func TestCppStdlibRegistry_GetFreeFunction(t *testing.T) {
	dir := t.TempDir()
	writeCppRegistry(t, dir)
	r := NewCppStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))

	mv, err := r.GetFreeFunction("utility", "std::move")
	require.NoError(t, err)
	assert.Equal(t, "T&&", mv.ReturnType)

	// Wrong header.
	_, err = r.GetFreeFunction("vector", "std::move")
	require.Error(t, err)
}

func TestCppStdlibRegistry_GetFunctionFreeFunctionFallback(t *testing.T) {
	dir := t.TempDir()
	writeCppRegistry(t, dir)
	r := NewCppStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))

	// std::swap is a free function under "vector"; GetFunction must find it
	// via the free-function fallback.
	got, err := r.GetFunction("vector", "std::swap")
	require.NoError(t, err)
	assert.Equal(t, "std::swap", got.FQN)
}

func TestCppStdlibRegistry_GetFunctionMissing(t *testing.T) {
	dir := t.TempDir()
	writeCppRegistry(t, dir)
	r := NewCppStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))

	_, err := r.GetFunction("vector", "nonexistent")
	require.Error(t, err)
}

// TestCppStdlibRegistry_HTTPMode_NetworkFailureNoCacheSurfacesError mirrors
// the C-loader test: with the disk cache explicitly disabled, an HTTP-only
// loader pointed at an unreachable port must surface the error.
func TestCppStdlibRegistry_HTTPMode_NetworkFailureNoCacheSurfacesError(t *testing.T) {
	r := NewCppStdlibRegistryRemote("http://127.0.0.1:1/registries", core.PlatformLinux)
	r.diskCache = nil
	err := r.LoadManifest(noopLogger{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loadManifestFromHTTP")
}

func TestCppStdlibRegistry_HeaderCountBeforeLoad(t *testing.T) {
	r := NewCppStdlibRegistryFile(t.TempDir(), core.PlatformLinux)
	assert.Equal(t, 0, r.HeaderCount())
}

func TestCppStdlibRegistry_GetHeaderBeforeLoad(t *testing.T) {
	r := NewCppStdlibRegistryFile(t.TempDir(), core.PlatformLinux)
	_, err := r.GetHeader("vector")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest not loaded")
}

func TestCppStdlibRegistry_GetHeaderUnknown(t *testing.T) {
	dir := t.TempDir()
	writeCppRegistry(t, dir)
	r := NewCppStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))
	_, err := r.GetHeader("absent")
	require.Error(t, err)
}

func TestCppStdlibRegistry_GetHeaderCorrupt(t *testing.T) {
	dir := t.TempDir()
	writeCppRegistry(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "vector_stdlib.json"), []byte("trash"), 0o644))

	r := NewCppStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))
	_, err := r.GetHeader("vector")
	require.Error(t, err)
}

func TestCppStdlibRegistry_HeaderCacheHit(t *testing.T) {
	dir := t.TempDir()
	writeCppRegistry(t, dir)
	r := NewCppStdlibRegistryFile(dir, core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))

	a, err := r.GetHeader("vector")
	require.NoError(t, err)
	b, err := r.GetHeader("vector")
	require.NoError(t, err)
	assert.Same(t, a, b)
}

func TestCppStdlibRegistry_LoadManifestMissing(t *testing.T) {
	r := NewCppStdlibRegistryFile(filepath.Join(t.TempDir(), "absent"), core.PlatformLinux)
	err := r.LoadManifest(noopLogger{})
	require.Error(t, err)
}

func TestCppStdlibRegistry_LoadManifestCorrupt(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte("garbage"), 0o644))
	r := NewCppStdlibRegistryFile(dir, core.PlatformLinux)
	err := r.LoadManifest(noopLogger{})
	require.Error(t, err)
}

func TestCppStdlibRegistry_RemoteCtorTrimsSlash(t *testing.T) {
	r := NewCppStdlibRegistryRemote("https://x/registries/", core.PlatformLinux)
	assert.Equal(t, "https://x/registries", r.baseURL)
}

func TestCppStdlibRegistry_ImplementsInterface(t *testing.T) {
	var _ core.CppStdlibLoader = NewCppStdlibRegistryFile(t.TempDir(), core.PlatformLinux)
	var _ core.CppStdlibLoader = NewCppStdlibRegistryRemote("https://x", core.PlatformLinux)
}
