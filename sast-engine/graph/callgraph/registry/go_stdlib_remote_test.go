package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Helpers ---

// buildFmtPackage returns a minimal GoStdlibPackage for "fmt" used in tests.
func buildFmtPackage() *core.GoStdlibPackage {
	return &core.GoStdlibPackage{
		ImportPath: "fmt",
		GoVersion:  "1.26",
		Functions: map[string]*core.GoStdlibFunction{
			"Println": {
				Name:       "Println",
				Signature:  "func Println(a ...any) (n int, err error)",
				IsVariadic: true,
				Confidence: 1.0,
			},
			"Sprintf": {
				Name:       "Sprintf",
				Signature:  "func Sprintf(format string, a ...any) string",
				IsVariadic: true,
				Confidence: 1.0,
			},
		},
		Types: map[string]*core.GoStdlibType{
			"Stringer": {
				Name: "Stringer",
				Kind: "interface",
			},
		},
		Constants: map[string]*core.GoStdlibConstant{},
		Variables: map[string]*core.GoStdlibVariable{},
	}
}

// packageChecksum computes the "sha256:<hex>" checksum of a marshalled package.
func packageChecksum(pkg *core.GoStdlibPackage) string {
	data, _ := json.Marshal(pkg)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// buildManifest returns a GoManifest with a single "fmt" entry.
func buildManifest(checksum string) *core.GoManifest {
	return &core.GoManifest{
		SchemaVersion:   "1.0.0",
		RegistryVersion: "v1",
		GoVersion: core.GoVersionInfo{
			Major: 1, Minor: 26, Patch: 0, Full: "1.26.0",
		},
		Packages: []*core.GoPackageEntry{
			{
				ImportPath:    "fmt",
				Checksum:      checksum,
				FunctionCount: 2,
				TypeCount:     1,
			},
		},
		Statistics: &core.GoRegistryStats{
			TotalPackages:  1,
			TotalFunctions: 2,
			TotalTypes:     1,
		},
	}
}

// --- Constructor tests ---

func TestNewGoStdlibRegistryRemote(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com/registries", "1.26")

	assert.Equal(t, "https://example.com/registries", r.baseURL)
	assert.Equal(t, "1.26", r.goVersion)
	assert.NotNil(t, r.packageCache)
	assert.NotNil(t, r.httpClient)
	assert.Equal(t, 30*time.Second, r.httpClient.Timeout)
	assert.False(t, r.IsManifestLoaded())
}

func TestNewGoStdlibRegistryRemote_StripsTrailingSlash(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com/registries/", "1.21")
	assert.Equal(t, "https://example.com/registries", r.baseURL)
}

func TestGoVersion(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	assert.Equal(t, "1.26", r.GoVersion())
}

// --- LoadManifest tests ---

func TestLoadManifest_Success(t *testing.T) {
	fmtPkg := buildFmtPackage()
	checksum := packageChecksum(fmtPkg)
	manifest := buildManifest(checksum)
	manifestJSON, err := json.Marshal(manifest)
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/go1.26/stdlib/v1/manifest.json", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(manifestJSON)
	}))
	defer server.Close()

	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")
	err = remote.LoadManifest(newTestLogger())

	require.NoError(t, err)
	assert.True(t, remote.IsManifestLoaded())
	assert.Equal(t, 1, remote.PackageCount())
}

func TestLoadManifest_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")
	err := remote.LoadManifest(newTestLogger())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestLoadManifest_NetworkError(t *testing.T) {
	// Point to a server that's not listening.
	remote := NewGoStdlibRegistryRemote("http://127.0.0.1:1", "1.26")
	remote.httpClient = &http.Client{Timeout: 100 * time.Millisecond}

	err := remote.LoadManifest(newTestLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "downloading manifest")
}

func TestLoadManifest_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer server.Close()

	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")
	err := remote.LoadManifest(newTestLogger())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing manifest JSON")
}

// --- ValidateStdlibImport tests ---

func TestValidateStdlibImport_ManifestNotLoaded(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	assert.False(t, r.ValidateStdlibImport("fmt"))
}

func TestValidateStdlibImport_Found(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	r.manifest = buildManifest("sha256:abc")
	assert.True(t, r.ValidateStdlibImport("fmt"))
}

func TestValidateStdlibImport_NotFound(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	r.manifest = buildManifest("sha256:abc")
	assert.False(t, r.ValidateStdlibImport("github.com/user/pkg"))
}

// --- PackageCount tests ---

func TestPackageCount_ManifestNotLoaded(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	assert.Equal(t, 0, r.PackageCount())
}

func TestPackageCount_WithStatistics(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	r.manifest = buildManifest("sha256:abc")
	assert.Equal(t, 1, r.PackageCount())
}

func TestPackageCount_NilStatistics(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	r.manifest = &core.GoManifest{
		Packages:   []*core.GoPackageEntry{{ImportPath: "fmt"}, {ImportPath: "os"}},
		Statistics: nil, // deliberately nil
	}
	assert.Equal(t, 2, r.PackageCount())
}

// --- GetPackage tests ---

func TestGetPackage_CacheHit(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	expected := buildFmtPackage()
	r.packageCache["fmt"] = expected

	got, err := r.GetPackage("fmt")
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestGetPackage_Download_Success(t *testing.T) {
	fmtPkg := buildFmtPackage()
	pkgJSON, _ := json.Marshal(fmtPkg)
	checksum := "sha256:" + func() string {
		sum := sha256.Sum256(pkgJSON)
		return hex.EncodeToString(sum[:])
	}()
	manifest := buildManifest(checksum)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/go1.26/stdlib/v1/manifest.json":
			data, _ := json.Marshal(manifest)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		case "/go1.26/stdlib/v1/fmt_stdlib.json":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(pkgJSON)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")
	require.NoError(t, remote.LoadManifest(newTestLogger()))

	pkg, err := remote.GetPackage("fmt")
	require.NoError(t, err)
	assert.Equal(t, "fmt", pkg.ImportPath)
	assert.Contains(t, pkg.Functions, "Println")

	// Second call must come from cache (no extra HTTP request needed).
	assert.Equal(t, 1, remote.CacheSize())
}

func TestGetPackage_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")
	_, err := remote.GetPackage("nonexistent/pkg")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetPackage_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")
	_, err := remote.GetPackage("fmt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 503")
}

func TestGetPackage_NetworkError(t *testing.T) {
	remote := NewGoStdlibRegistryRemote("http://127.0.0.1:1", "1.26")
	remote.httpClient = &http.Client{Timeout: 100 * time.Millisecond}

	_, err := remote.GetPackage("fmt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "downloading package fmt")
}

func TestGetPackage_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer server.Close()

	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")
	_, err := remote.GetPackage("fmt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parsing package fmt JSON")
}

func TestGetPackage_ChecksumMismatch(t *testing.T) {
	fmtPkg := buildFmtPackage()
	pkgJSON, _ := json.Marshal(fmtPkg)
	badChecksum := "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	manifest := buildManifest(badChecksum)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/go1.26/stdlib/v1/manifest.json":
			data, _ := json.Marshal(manifest)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		case "/go1.26/stdlib/v1/fmt_stdlib.json":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(pkgJSON)
		}
	}))
	defer server.Close()

	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")
	require.NoError(t, remote.LoadManifest(newTestLogger()))

	_, err := remote.GetPackage("fmt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "checksum mismatch")
}

func TestGetPackage_SkipsChecksumWhenManifestNil(t *testing.T) {
	fmtPkg := buildFmtPackage()
	pkgJSON, _ := json.Marshal(fmtPkg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pkgJSON)
	}))
	defer server.Close()

	// No manifest loaded — checksum is skipped.
	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")
	pkg, err := remote.GetPackage("fmt")
	require.NoError(t, err)
	assert.Equal(t, "fmt", pkg.ImportPath)
}

func TestGetPackage_SkipsChecksumWhenEntryMissing(t *testing.T) {
	// Manifest exists but doesn't have an entry for the requested package.
	fmtPkg := buildFmtPackage()
	pkgJSON, _ := json.Marshal(fmtPkg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pkgJSON)
	}))
	defer server.Close()

	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")
	// Manifest with NO entry for "fmt".
	remote.manifest = &core.GoManifest{
		Packages:   []*core.GoPackageEntry{},
		Statistics: &core.GoRegistryStats{},
	}

	pkg, err := remote.GetPackage("fmt")
	require.NoError(t, err)
	assert.Equal(t, "fmt", pkg.ImportPath)
}

func TestGetPackage_SkipsChecksumWhenEntryChecksumEmpty(t *testing.T) {
	fmtPkg := buildFmtPackage()
	pkgJSON, _ := json.Marshal(fmtPkg)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pkgJSON)
	}))
	defer server.Close()

	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")
	// Entry with empty checksum — verification is skipped.
	remote.manifest = &core.GoManifest{
		Packages:   []*core.GoPackageEntry{{ImportPath: "fmt", Checksum: ""}},
		Statistics: &core.GoRegistryStats{},
	}

	pkg, err := remote.GetPackage("fmt")
	require.NoError(t, err)
	assert.Equal(t, "fmt", pkg.ImportPath)
}

// --- GetFunction tests ---

func TestGetFunction_Found(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	r.packageCache["fmt"] = buildFmtPackage()

	fn, err := r.GetFunction("fmt", "Println")
	require.NoError(t, err)
	assert.Equal(t, "Println", fn.Name)
	assert.True(t, fn.IsVariadic)
}

func TestGetFunction_NotFound(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	r.packageCache["fmt"] = buildFmtPackage()

	_, err := r.GetFunction("fmt", "NoSuchFunc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NoSuchFunc")
	assert.Contains(t, err.Error(), "fmt")
}

func TestGetFunction_PackageError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	r := NewGoStdlibRegistryRemote(server.URL, "1.26")
	_, err := r.GetFunction("os", "Open")
	assert.Error(t, err)
}

// --- GetType tests ---

func TestGetType_Found(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	r.packageCache["fmt"] = buildFmtPackage()

	typ, err := r.GetType("fmt", "Stringer")
	require.NoError(t, err)
	assert.Equal(t, "Stringer", typ.Name)
}

func TestGetType_NotFound(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	r.packageCache["fmt"] = buildFmtPackage()

	_, err := r.GetType("fmt", "NoSuchType")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NoSuchType")
}

func TestGetType_PackageError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	r := NewGoStdlibRegistryRemote(server.URL, "1.26")
	_, err := r.GetType("os", "File")
	assert.Error(t, err)
}

// --- Cache management tests ---

func TestCacheSize(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	assert.Equal(t, 0, r.CacheSize())

	r.packageCache["fmt"] = buildFmtPackage()
	assert.Equal(t, 1, r.CacheSize())
}

func TestClearCache(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	r.packageCache["fmt"] = buildFmtPackage()
	r.packageCache["os"] = &core.GoStdlibPackage{ImportPath: "os"}

	assert.Equal(t, 2, r.CacheSize())
	r.ClearCache()
	assert.Equal(t, 0, r.CacheSize())
}

// --- goPackageToFilename tests ---

func TestGoPackageToFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"fmt", "fmt_stdlib.json"},
		{"net/http", "net_http_stdlib.json"},
		{"encoding/json", "encoding_json_stdlib.json"},
		{"net/http/httputil", "net_http_httputil_stdlib.json"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, goPackageToFilename(tt.input))
		})
	}
}

// --- Concurrency tests ---

func TestGetPackage_ConcurrentAccess(t *testing.T) {
	fmtPkg := buildFmtPackage()
	pkgJSON, _ := json.Marshal(fmtPkg)

	var requestCount int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(pkgJSON)
	}))
	defer server.Close()

	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")

	const goroutines = 20
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pkg, err := remote.GetPackage("fmt")
			if err != nil {
				errs <- err
				return
			}
			if pkg.ImportPath != "fmt" {
				errs <- fmt.Errorf("unexpected import path: %s", pkg.ImportPath)
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("goroutine error: %v", err)
	}

	// The double-check locking ensures the package is downloaded at most once.
	assert.Equal(t, 1, remote.CacheSize())
	mu.Lock()
	assert.Equal(t, 1, requestCount, "package should be fetched exactly once")
	mu.Unlock()
}

func TestValidateStdlibImport_ConcurrentAccess(t *testing.T) {
	r := NewGoStdlibRegistryRemote("https://example.com", "1.26")
	r.manifest = buildManifest("sha256:abc")

	const goroutines = 50
	var wg sync.WaitGroup
	results := make([]bool, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = r.ValidateStdlibImport("fmt")
		}(i)
	}

	wg.Wait()
	for _, result := range results {
		assert.True(t, result)
	}
}

// --- GoStdlibLoader interface compliance ---

func TestGoStdlibRegistryRemote_ImplementsInterface(t *testing.T) {
	// Compile-time check: GoStdlibRegistryRemote must implement GoStdlibLoader.
	var _ core.GoStdlibLoader = (*GoStdlibRegistryRemote)(nil)
	t.Log("GoStdlibRegistryRemote correctly implements core.GoStdlibLoader")
}

// --- Request creation error (invalid URL) ---

func TestLoadManifest_BadURL(t *testing.T) {
	// Use a URL with an invalid character to force request creation failure.
	remote := NewGoStdlibRegistryRemote("http://\x00bad", "1.26")
	err := remote.LoadManifest(newTestLogger())
	assert.Error(t, err)
}

func TestGetPackage_BadURL(t *testing.T) {
	remote := NewGoStdlibRegistryRemote("http://\x00bad", "1.26")
	_, err := remote.GetPackage("fmt")
	assert.Error(t, err)
}

// --- io.ReadAll error tests (connection closed mid-response) ---

func TestLoadManifest_ReadBodyError(t *testing.T) {
	// Server writes Content-Length: 1000 but closes after sending a few bytes,
	// causing io.ReadAll to fail with an unexpected EOF error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Skip("server does not support hijacking")
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("hijack failed: %v", err)
			return
		}
		_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 1000\r\n\r\npartial"))
		conn.Close()
	}))
	defer server.Close()

	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")
	err := remote.LoadManifest(newTestLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading manifest body")
}

func TestGetPackage_ReadBodyError(t *testing.T) {
	// Same technique: close the connection after writing partial body data.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Skip("server does not support hijacking")
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("hijack failed: %v", err)
			return
		}
		_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 5000\r\n\r\npartial"))
		conn.Close()
	}))
	defer server.Close()

	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")
	_, err := remote.GetPackage("fmt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reading package fmt body")
}

// TestFetchPackageLocked_DoubleCheckCacheHit verifies that fetchPackageLocked
// returns the cached entry immediately when the cache is already populated,
// without making any HTTP request. This exercises the double-check path that
// guards against duplicate downloads when multiple goroutines race on the
// write lock.
func TestFetchPackageLocked_DoubleCheckCacheHit(t *testing.T) {
	fmtPkg := buildFmtPackage()

	// Server must NOT be called — any request means the double-check failed.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("unexpected HTTP request: double-check should have returned cached value")
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}))
	defer server.Close()

	remote := NewGoStdlibRegistryRemote(server.URL, "1.26")

	// Simulate the state where another goroutine already populated the cache
	// before we acquired the write lock.
	remote.cacheMutex.Lock()
	remote.packageCache["fmt"] = fmtPkg

	// Call fetchPackageLocked while holding the write lock, as GetPackage does.
	// The double-check must return the cached entry without hitting the network.
	result, err := remote.fetchPackageLocked("fmt")
	remote.cacheMutex.Unlock()

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, fmtPkg.ImportPath, result.ImportPath)
	assert.Equal(t, 1, remote.CacheSize())
}
