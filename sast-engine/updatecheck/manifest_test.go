package updatecheck

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sampleManifest returns a valid Manifest for use in tests.
func sampleManifest() Manifest {
	return Manifest{
		Schema: 1,
		Latest: ManifestLatest{
			Version:    "2.1.1",
			ReleasedAt: time.Date(2026, 4, 10, 18, 22, 0, 0, time.UTC),
			ReleaseURL: "https://github.com/shivasurya/code-pathfinder/releases/tag/v2.1.1",
		},
		Channels: map[string]string{"stable": "2.1.1"},
		Message: ManifestMessage{
			Level: "info",
			Text:  "v2.1.1 ships Go third-party type resolution.",
		},
		Announcements: []ManifestAnnouncement{},
		MinSupported:  "1.9.0",
	}
}

func serveManifest(t *testing.T, m Manifest) *httptest.Server {
	t.Helper()
	body, err := json.Marshal(m)
	require.NoError(t, err)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
}

// TestFetch_Success verifies a round-trip fetch against a local httptest server.
func TestFetch_Success(t *testing.T) {
	m := sampleManifest()
	srv := serveManifest(t, m)
	defer srv.Close()

	got, err := Fetch(context.Background(), srv.URL, 5*time.Second)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 1, got.Schema)
	assert.Equal(t, "2.1.1", got.Latest.Version)
	assert.Equal(t, "v2.1.1 ships Go third-party type resolution.", got.Message.Text)
	assert.Equal(t, "1.9.0", got.MinSupported)
}

// TestFetch_UnknownFieldsIgnored verifies forward-compatible JSON parsing:
// unknown fields in the manifest must not cause an error.
func TestFetch_UnknownFieldsIgnored(t *testing.T) {
	payload := `{
		"schema": 1,
		"latest": {"version": "2.1.1", "released_at": "2026-04-10T18:22:00Z", "release_url": "https://example.com"},
		"future_field": "ignored",
		"another_new_field": 42
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	got, err := Fetch(context.Background(), srv.URL, 5*time.Second)
	require.NoError(t, err)
	assert.Equal(t, "2.1.1", got.Latest.Version)
}

// TestFetch_HTTP404 verifies that non-200 responses are surfaced as errors.
func TestFetch_HTTP404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := Fetch(context.Background(), srv.URL, 5*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

// TestFetch_MalformedJSON verifies that invalid JSON is surfaced as an error.
func TestFetch_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json at all {{{"))
	}))
	defer srv.Close()

	_, err := Fetch(context.Background(), srv.URL, 5*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updatecheck: decode:")
}

// TestFetch_UnsupportedSchema verifies that an unknown schema version is rejected.
func TestFetch_UnsupportedSchema(t *testing.T) {
	payload := `{"schema": 99, "latest": {"version": "9.0.0"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	_, err := Fetch(context.Background(), srv.URL, 5*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported schema 99")
}

// TestFetch_Timeout verifies that a slow server causes a timeout error.
func TestFetch_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never respond — let the client time out.
		<-r.Context().Done()
	}))
	defer srv.Close()

	_, err := Fetch(context.Background(), srv.URL, 50*time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updatecheck: fetch:")
}

// TestFetch_InvalidURL verifies that a malformed URL is surfaced as an error
// (covers the http.NewRequestWithContext error path).
func TestFetch_InvalidURL(t *testing.T) {
	_, err := Fetch(context.Background(), "://bad-url", 5*time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "updatecheck: build request:")
}

// TestFetch_UserAgentHeader verifies that the correct User-Agent header is sent.
func TestFetch_UserAgentHeader(t *testing.T) {
	var receivedUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		m := sampleManifest()
		body, _ := json.Marshal(m)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	_, err := Fetch(context.Background(), srv.URL, 5*time.Second)
	require.NoError(t, err)
	assert.Equal(t, "pathfinder-updatecheck/1", receivedUA)
}
