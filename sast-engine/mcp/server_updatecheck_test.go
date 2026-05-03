package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/updatecheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startFakeManifestCDN starts an httptest.Server that returns a manifest with a
// stale version (so the running binary will always appear to need an upgrade).
func startFakeManifestCDN(t *testing.T) *httptest.Server {
	t.Helper()
	m := updatecheck.Manifest{
		Schema: 1,
		Latest: updatecheck.ManifestLatest{
			Version:    "99.0.0",
			ReleasedAt: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
			ReleaseURL: "https://example.com/releases/v99.0.0",
		},
	}
	body, err := json.Marshal(m)
	require.NoError(t, err)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
}

// withMCPManifestURL temporarily overrides the package-level manifest URL and
// restores it on test cleanup.
func withMCPManifestURL(t *testing.T, url string) {
	t.Helper()
	old := mcpManifestURL
	mcpManifestURL = url
	t.Cleanup(func() { mcpManifestURL = old })
}

// --- NewServer update-check tests -------------------------------------------

func TestNewServer_FetchesUpdateInfo(t *testing.T) {
	srv := startFakeManifestCDN(t)
	defer srv.Close()
	withMCPManifestURL(t, srv.URL)
	t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "")

	server := createTestServer()
	// Set a real semver so the Compare logic sees current < 99.0.0.
	server.version = "1.0.0"
	// Re-run fetchUpdateInfo now that version is set.
	server.fetchUpdateInfo()

	require.NotNil(t, server.updateInfo, "updateInfo should be non-nil when CDN returns a stale manifest")
	require.NotNil(t, server.updateInfo.Upgrade)
	assert.Equal(t, "99.0.0", server.updateInfo.Upgrade.Latest)
}

func TestNewServer_FetchError_UpdateInfoNil(t *testing.T) {
	// Point at an unreachable URL — the fetch returns an error immediately
	// (connection refused), so fetchUpdateInfo sets s.updateInfo = nil.
	withMCPManifestURL(t, "http://127.0.0.1:1") // port 1 is never listening
	t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "")

	server := createTestServer()
	server.version = "1.0.0"
	server.fetchUpdateInfo()

	// On connection refused, Check returns nil; server must still construct.
	assert.NotNil(t, server)
	assert.Nil(t, server.updateInfo, "updateInfo should be nil when the fetch fails")
}

// --- handleInitialize metadata tests ----------------------------------------

func TestHandleInitialize_NoMetadataWhenUpdateInfoNil(t *testing.T) {
	server := createTestServer()
	server.updateInfo = nil

	req := makeJSONRPCRequest("initialize", nil)
	resp := server.handleInitialize(req)

	require.NotNil(t, resp)
	result := extractInitResult(t, resp)
	assert.Nil(t, result.ServerInfo.Metadata, "Metadata should be absent when updateInfo is nil")
}

func TestHandleInitialize_PopulatesMetadata(t *testing.T) {
	server := createTestServer()
	server.updateInfo = &updatecheck.Result{
		Upgrade: &updatecheck.UpgradeNotice{
			Current:    "1.0.0",
			Latest:     "2.0.0",
			ReleaseURL: "https://example.com/releases/v2.0.0",
			Message:    "big release",
		},
		Announcement: &updatecheck.Announcement{
			ID:    "ann-1",
			Level: "info",
			Title: "New docs",
			Text:  "docs updated",
			URL:   "https://example.com/docs",
		},
	}

	req := makeJSONRPCRequest("initialize", nil)
	resp := server.handleInitialize(req)

	require.NotNil(t, resp)
	result := extractInitResult(t, resp)
	require.NotNil(t, result.ServerInfo.Metadata, "Metadata should be populated")

	md := result.ServerInfo.Metadata
	assert.Equal(t, "2.0.0", md.LatestVersion)
	assert.Equal(t, "big release", md.UpdateMessage)
	assert.Equal(t, "https://example.com/releases/v2.0.0", md.ReleaseURL)

	require.NotNil(t, md.Announcement)
	assert.Equal(t, "ann-1", md.Announcement.ID)
	assert.Equal(t, "info", md.Announcement.Level)
	assert.Equal(t, "New docs", md.Announcement.Title)
}

func TestHandleInitialize_MetadataOmittedWhenEmpty(t *testing.T) {
	// A Result with neither Upgrade nor Announcement yields an empty ServerMetadata
	// — the Metadata field should remain nil (omitempty in JSON, not set at all).
	server := createTestServer()
	server.updateInfo = &updatecheck.Result{} // both fields nil

	req := makeJSONRPCRequest("initialize", nil)
	resp := server.handleInitialize(req)

	result := extractInitResult(t, resp)
	assert.Nil(t, result.ServerInfo.Metadata)
}

// --- helpers used by these tests --------------------------------------------

func makeJSONRPCRequest(method string, params any) *JSONRPCRequest { //nolint:unparam
	var raw json.RawMessage
	if params != nil {
		b, _ := json.Marshal(params)
		raw = b
	}
	return &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  raw,
	}
}

func extractInitResult(t *testing.T, resp *JSONRPCResponse) InitializeResult {
	t.Helper()
	b, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var r InitializeResult
	require.NoError(t, json.Unmarshal(b, &r))
	return r
}
