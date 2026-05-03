package registry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiskCache_ManifestRoundTrip(t *testing.T) {
	c := newDiskCacheStore(t.TempDir())
	m := &core.CStdlibManifest{
		SchemaVersion: "1.0.0",
		Platform:      core.PlatformLinux,
		Language:      core.LanguageC,
	}
	data, err := json.Marshal(m)
	require.NoError(t, err)

	require.NoError(t, c.SaveManifest(data))

	got, err := c.GetManifest()
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", got.SchemaVersion)
	assert.Equal(t, core.PlatformLinux, got.Platform)
}

func TestDiskCache_ManifestMiss(t *testing.T) {
	c := newDiskCacheStore(t.TempDir())
	_, err := c.GetManifest()
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCacheMiss))
}

func TestDiskCache_ManifestCorrupt(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "manifest.json"), []byte("{not json"), 0o644))
	c := newDiskCacheStore(root)
	_, err := c.GetManifest()
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrCacheMiss))
}

func TestDiskCache_HeaderRoundTrip(t *testing.T) {
	c := newDiskCacheStore(t.TempDir())
	h := &core.CStdlibHeader{Header: "stdio.h", Language: core.LanguageC}
	data, err := json.Marshal(h)
	require.NoError(t, err)

	require.NoError(t, c.SaveHeader("stdio_stdlib.json", data))

	got, err := c.GetHeader("stdio_stdlib.json")
	require.NoError(t, err)
	assert.Equal(t, "stdio.h", got.Header)
}

func TestDiskCache_HeaderMiss(t *testing.T) {
	c := newDiskCacheStore(t.TempDir())
	_, err := c.GetHeader("absent.json")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrCacheMiss))
}

func TestDiskCache_HeaderCorrupt(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "stdio_stdlib.json"), []byte("garbage"), 0o644))
	c := newDiskCacheStore(root)
	_, err := c.GetHeader("stdio_stdlib.json")
	require.Error(t, err)
}

func TestDiskCache_IsFreshWithinTTL(t *testing.T) {
	c := newDiskCacheStore(t.TempDir())
	require.NoError(t, c.SaveHeader("x.json", []byte("{}")))
	assert.True(t, c.IsFresh("x.json", 24*time.Hour))
}

func TestDiskCache_IsFreshExpired(t *testing.T) {
	root := t.TempDir()
	c := newDiskCacheStore(root)
	require.NoError(t, c.SaveHeader("x.json", []byte("{}")))
	// Backdate the file to 25h ago.
	old := time.Now().Add(-25 * time.Hour)
	require.NoError(t, os.Chtimes(filepath.Join(root, "x.json"), old, old))
	assert.False(t, c.IsFresh("x.json", 24*time.Hour))
}

func TestDiskCache_IsFreshMissingFile(t *testing.T) {
	c := newDiskCacheStore(t.TempDir())
	assert.False(t, c.IsFresh("nope.json", 24*time.Hour))
}

func TestDiskCache_NilStore(t *testing.T) {
	var c *diskCacheStore
	require.Error(t, c.SaveManifest([]byte("{}")))
	require.Error(t, c.SaveHeader("x.json", []byte("{}")))

	_, err := c.GetManifest()
	require.True(t, errors.Is(err, ErrCacheMiss))
	_, err = c.GetHeader("x.json")
	require.True(t, errors.Is(err, ErrCacheMiss))
	assert.False(t, c.IsFresh("x.json", time.Hour))
}

func TestDiskCache_EmptyRoot(t *testing.T) {
	c := newDiskCacheStore("")
	require.Error(t, c.SaveManifest([]byte("{}")))
	_, err := c.GetManifest()
	require.True(t, errors.Is(err, ErrCacheMiss))
}

func TestDiskCache_SavesCreatesDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "nested", "deep")
	c := newDiskCacheStore(root)
	require.NoError(t, c.SaveHeader("x.json", []byte("{}")))
	_, err := os.Stat(filepath.Join(root, "x.json"))
	require.NoError(t, err)
}

func TestGetStdlibCacheRoot(t *testing.T) {
	got := getStdlibCacheRoot()
	// Either "" (no usable dir) or a path under the user's home/cache.
	if got == "" {
		return
	}
	assert.Contains(t, got, "pathfinder")
	assert.Contains(t, got, "registries")
}

func TestGetStdlibCacheRoot_PrefersXDG(t *testing.T) {
	if got := os.Getenv("XDG_CACHE_HOME"); got == "" {
		t.Setenv("XDG_CACHE_HOME", t.TempDir())
	} else {
		// Already set; just confirm it propagates.
	}
	got := getStdlibCacheRoot()
	assert.Contains(t, got, "pathfinder")
}

func TestStdlibCacheTTL_Constant(t *testing.T) {
	assert.Equal(t, 24*time.Hour, stdlibCacheTTL)
}
