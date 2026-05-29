package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stdlibFixture is the canonical small registry the HTTP tests reuse. Each
// test that needs a server can call serveFixture(t, fixture) to get a live
// httptest.Server and a baseURL that the loader recognizes.
type stdlibFixture struct {
	manifestC   *core.CStdlibManifest
	headerC     *core.CStdlibHeader
	manifestCpp *core.CStdlibManifest
	headerCpp   *core.CStdlibHeader
}

func newCFixture() *stdlibFixture {
	return &stdlibFixture{
		manifestC: &core.CStdlibManifest{
			SchemaVersion:   "1.0.0",
			RegistryVersion: "v1",
			Platform:        core.PlatformLinux,
			Language:        core.LanguageC,
			Headers: []*core.CStdlibHeaderEntry{
				{Header: "stdio.h", ModuleID: "c::stdio", File: "stdio_stdlib.json"},
			},
			Statistics: &core.CStdlibStatistics{TotalHeaders: 1, TotalFunctions: 1},
		},
		headerC: &core.CStdlibHeader{
			SchemaVersion: "1.0.0",
			Header:        "stdio.h",
			ModuleID:      "c::stdio",
			Language:      core.LanguageC,
			Functions: map[string]*core.CStdlibFunction{
				"printf": {FQN: "c::stdio::printf", ReturnType: "int", Source: core.SourceOverlay, Confidence: 1.0},
			},
		},
		manifestCpp: &core.CStdlibManifest{
			SchemaVersion:   "1.0.0",
			RegistryVersion: "v1",
			Platform:        core.PlatformLinux,
			Language:        core.LanguageCpp,
			Headers: []*core.CStdlibHeaderEntry{
				{Header: "vector", ModuleID: "std::vector", File: "vector_stdlib.json"},
			},
			Statistics: &core.CStdlibStatistics{TotalHeaders: 1, TotalClasses: 1, TotalFunctions: 1},
		},
		headerCpp: &core.CStdlibHeader{
			SchemaVersion: "1.0.0",
			Header:        "vector",
			ModuleID:      "std::vector",
			Language:      core.LanguageCpp,
			Classes: map[string]*core.CppStdlibClass{
				"std::vector": {
					FQN: "std::vector", TypeParams: []string{"T"},
					Methods: map[string]*core.CStdlibFunction{
						"push_back": {FQN: "std::vector::push_back", ReturnType: "void", Source: core.SourceOverlay, Confidence: 1.0},
					},
				},
			},
		},
	}
}

// serveFixture stands up a test server that mimics the CDN's URL layout:
//
//	GET /registries/linux/c/v1/manifest.json
//	GET /registries/linux/c/v1/stdio_stdlib.json
//	GET /registries/linux/cpp/v1/manifest.json
//	GET /registries/linux/cpp/v1/vector_stdlib.json
//
// The returned baseURL ends in "/registries" so the loader's joinURL builds
// exactly the paths above.
func serveFixture(t *testing.T, f *stdlibFixture) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/registries/linux/c/v1/manifest.json":
			writeJSON(t, w, f.manifestC)
		case "/registries/linux/c/v1/stdio_stdlib.json":
			writeJSON(t, w, f.headerC)
		case "/registries/linux/cpp/v1/manifest.json":
			writeJSON(t, w, f.manifestCpp)
		case "/registries/linux/cpp/v1/vector_stdlib.json":
			writeJSON(t, w, f.headerCpp)
		default:
			http.NotFound(w, req)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

// withTempCacheRoot points the loader's disk cache at a t.TempDir for the
// duration of the test. The default $HOME/.cache root is unsuitable because
// tests must not pollute the developer's real cache.
func withTempCacheRoot(t *testing.T, r *CStdlibRegistryRemote) {
	t.Helper()
	r.diskCache = newDiskCacheStore(t.TempDir())
}

func withTempCacheRootCpp(t *testing.T, r *CppStdlibRegistryRemote) {
	t.Helper()
	r.diskCache = newDiskCacheStore(t.TempDir())
}

func TestCStdlibRegistry_HTTP_LoadManifestAndHeader(t *testing.T) {
	srv := serveFixture(t, newCFixture())
	r := NewCStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	withTempCacheRoot(t, r)

	require.NoError(t, r.LoadManifest(noopLogger{}))
	assert.Equal(t, 1, r.HeaderCount())

	h, err := r.GetHeader("stdio.h")
	require.NoError(t, err)
	require.Contains(t, h.Functions, "printf")
	assert.Equal(t, "c::stdio::printf", h.Functions["printf"].FQN)

	// Second call must hit the in-memory cache (same pointer).
	h2, err := r.GetHeader("stdio.h")
	require.NoError(t, err)
	assert.Same(t, h, h2)
}

func TestCStdlibRegistry_HTTP_NetworkFailureFallsBackToCachedManifest(t *testing.T) {
	// First successful fetch writes the manifest to disk cache.
	srv := serveFixture(t, newCFixture())
	cacheDir := t.TempDir()
	r := NewCStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	r.diskCache = newDiskCacheStore(cacheDir)
	require.NoError(t, r.LoadManifest(noopLogger{}))

	// Spin up a second loader pointed at a dead URL but sharing the same cache.
	r2 := NewCStdlibRegistryRemote("http://127.0.0.1:1/registries", core.PlatformLinux)
	r2.diskCache = newDiskCacheStore(cacheDir)
	require.NoError(t, r2.LoadManifest(noopLogger{}), "stale cache must serve when network is down")
	assert.Equal(t, 1, r2.HeaderCount())
}

func TestCStdlibRegistry_HTTP_HeaderFallsBackToCachedOnNetworkFailure(t *testing.T) {
	srv := serveFixture(t, newCFixture())
	cacheDir := t.TempDir()
	r := NewCStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	r.diskCache = newDiskCacheStore(cacheDir)
	require.NoError(t, r.LoadManifest(noopLogger{}))
	_, err := r.GetHeader("stdio.h") // populates header cache on disk
	require.NoError(t, err)

	// Tear the server down — subsequent loaders must serve from cache only.
	srv.Close()

	r2 := NewCStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	r2.diskCache = newDiskCacheStore(cacheDir)
	require.NoError(t, r2.LoadManifest(noopLogger{}), "loader should pick up cached manifest")
	h, err := r2.GetHeader("stdio.h")
	require.NoError(t, err)
	require.Contains(t, h.Functions, "printf")
}

func TestCStdlibRegistry_HTTP_404Surfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	t.Cleanup(srv.Close)

	r := NewCStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	r.diskCache = nil
	err := r.LoadManifest(noopLogger{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestCStdlibRegistry_HTTP_ChecksumValid(t *testing.T) {
	f := newCFixture()
	headerBytes, err := json.Marshal(f.headerC)
	require.NoError(t, err)
	sum := sha256.Sum256(headerBytes)
	f.manifestC.Headers[0].Checksum = "sha256:" + hex.EncodeToString(sum[:])

	srv := serveFixture(t, f)
	r := NewCStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	withTempCacheRoot(t, r)
	require.NoError(t, r.LoadManifest(noopLogger{}))
	_, err = r.GetHeader("stdio.h")
	require.NoError(t, err)
}

func TestCStdlibRegistry_HTTP_ChecksumMismatch(t *testing.T) {
	f := newCFixture()
	f.manifestC.Headers[0].Checksum = "sha256:deadbeef" // wrong hash

	srv := serveFixture(t, f)
	r := NewCStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	withTempCacheRoot(t, r)
	require.NoError(t, r.LoadManifest(noopLogger{}))
	_, err := r.GetHeader("stdio.h")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "digest mismatch")
}

func TestCStdlibRegistry_HTTP_ChecksumUnsupportedFormat(t *testing.T) {
	f := newCFixture()
	f.manifestC.Headers[0].Checksum = "md5:abcdef"

	srv := serveFixture(t, f)
	r := NewCStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	withTempCacheRoot(t, r)
	require.NoError(t, r.LoadManifest(noopLogger{}))
	_, err := r.GetHeader("stdio.h")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported checksum format")
}

// TestCStdlibRegistry_HTTP_ManifestEmbeddedURLIgnored pins the post-bugfix
// contract: even when the manifest's per-entry URL points elsewhere, the
// loader ignores it and constructs the fetch URL from its own baseURL.
//
// The bug this guards: the generator stamps every manifest with the
// production CDN URL by default, so any `--stdlib-base-url` override (or
// local HTTP server) would silently bypass the override if entry.URL won.
func TestCStdlibRegistry_HTTP_ManifestEmbeddedURLIgnored(t *testing.T) {
	f := newCFixture()
	srv := serveFixture(t, f)

	// Embed a URL on a host the test fixture doesn't even know about.
	// If the loader followed entry.URL, the fetch would fail (or hang).
	// The loader must ignore it and use baseURL + entry.File.
	f.manifestC.Headers[0].URL = "https://nowhere.test/this/would/not/work.json"

	r := NewCStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	withTempCacheRoot(t, r)
	require.NoError(t, r.LoadManifest(noopLogger{}))
	h, err := r.GetHeader("stdio.h")
	require.NoError(t, err, "loader must use its own baseURL, not entry.URL")
	require.Contains(t, h.Functions, "printf")
}

func TestCStdlibRegistry_HTTP_ParallelHeaderFetches(t *testing.T) {
	srv := serveFixture(t, newCFixture())

	r := NewCStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	withTempCacheRoot(t, r)
	require.NoError(t, r.LoadManifest(noopLogger{}))

	const workers = 50
	results := make([]*core.CStdlibHeader, workers)
	errs := make([]error, workers)
	done := make(chan struct{})
	for i := range workers {
		go func(idx int) {
			results[idx], errs[idx] = r.GetHeader("stdio.h")
			done <- struct{}{}
		}(i)
	}
	for range workers {
		<-done
	}
	for i := range workers {
		require.NoError(t, errs[i])
	}
	for i := 1; i < workers; i++ {
		assert.Same(t, results[0], results[i], "worker %d saw different pointer", i)
	}
}

func TestCStdlibRegistry_HTTP_NoCachePropagatesHeaderError(t *testing.T) {
	srv := serveFixture(t, newCFixture())
	r := NewCStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	require.NoError(t, r.LoadManifest(noopLogger{}))
	srv.Close() // cause subsequent fetches to fail
	r.diskCache = nil

	_, err := r.GetHeader("stdio.h")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetchHeaderFromHTTP")
}

func TestCStdlibRegistry_HTTP_ParseErrorOnGarbageBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(srv.Close)

	r := NewCStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	withTempCacheRoot(t, r)
	err := r.LoadManifest(noopLogger{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing manifest")
}

// --- C++ HTTP loader -------------------------------------------------------

func TestCppStdlibRegistry_HTTP_LoadManifestAndHeader(t *testing.T) {
	srv := serveFixture(t, newCFixture())
	r := NewCppStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	withTempCacheRootCpp(t, r)

	require.NoError(t, r.LoadManifest(noopLogger{}))
	assert.Equal(t, 1, r.HeaderCount())

	cls, err := r.GetClass("vector", "std::vector")
	require.NoError(t, err)
	assert.Equal(t, []string{"T"}, cls.TypeParams)
}

func TestCppStdlibRegistry_HTTP_NetworkFailureFallsBackToCachedHeader(t *testing.T) {
	srv := serveFixture(t, newCFixture())
	cacheDir := t.TempDir()
	r := NewCppStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	r.diskCache = newDiskCacheStore(cacheDir)
	require.NoError(t, r.LoadManifest(noopLogger{}))
	_, err := r.GetClass("vector", "std::vector")
	require.NoError(t, err)

	srv.Close()

	r2 := NewCppStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	r2.diskCache = newDiskCacheStore(cacheDir)
	require.NoError(t, r2.LoadManifest(noopLogger{}))
	cls, err := r2.GetClass("vector", "std::vector")
	require.NoError(t, err)
	assert.Equal(t, []string{"T"}, cls.TypeParams)
}

func TestCppStdlibRegistry_HTTP_ChecksumMismatch(t *testing.T) {
	f := newCFixture()
	f.manifestCpp.Headers[0].Checksum = "sha256:deadbeef"

	srv := serveFixture(t, f)
	r := NewCppStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	withTempCacheRootCpp(t, r)
	require.NoError(t, r.LoadManifest(noopLogger{}))
	_, err := r.GetClass("vector", "std::vector")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "digest mismatch")
}

// TestCppStdlibRegistry_HTTP_ManifestEmbeddedURLIgnored — C++ counterpart
// to the C-loader test. Same contract: entry.URL is ignored.
func TestCppStdlibRegistry_HTTP_ManifestEmbeddedURLIgnored(t *testing.T) {
	f := newCFixture()
	srv := serveFixture(t, f)
	f.manifestCpp.Headers[0].URL = "https://nowhere.test/would/not/work.json"

	r := NewCppStdlibRegistryRemote(srv.URL+"/registries", core.PlatformLinux)
	withTempCacheRootCpp(t, r)
	require.NoError(t, r.LoadManifest(noopLogger{}))
	_, err := r.GetClass("vector", "std::vector")
	require.NoError(t, err, "loader must use its own baseURL, not entry.URL")
}

// --- helpers --------------------------------------------------------------

func TestVerifyChecksum_EmptyExpectedSkips(t *testing.T) {
	require.NoError(t, verifyChecksum([]byte("anything"), ""))
}

func TestJoinURL(t *testing.T) {
	tests := []struct {
		name string
		base string
		segs []string
		want string
	}{
		{"trailing slash on base", "https://x/", []string{"a", "b"}, "https://x/a/b"},
		{"leading slash on segment", "https://x", []string{"/a", "/b"}, "https://x/a/b"},
		{"empty segments dropped", "https://x", []string{"", "a", ""}, "https://x/a"},
		{"no segments", "https://x", nil, "https://x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, joinURL(tt.base, tt.segs...))
		})
	}
}

func TestFetchURL_NilClient(t *testing.T) {
	_, err := fetchURL(nil, "http://x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil HTTP client")
}

func TestFetchURL_BadRequestURL(t *testing.T) {
	_, err := fetchURL(&http.Client{}, "http://[::1:bad")
	require.Error(t, err)
}
