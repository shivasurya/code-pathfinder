//go:build !noupdatecheck

package updatecheck

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/output"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startManifestServer starts an httptest server that responds with m.
func startManifestServer(t *testing.T, m Manifest) *httptest.Server {
	t.Helper()
	body, err := json.Marshal(m)
	require.NoError(t, err)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
}

// manifestWithUpgrade returns a Manifest that will trigger an upgrade notice
// for any current version < "2.1.1".
func manifestWithUpgrade() Manifest {
	return Manifest{
		Schema: 1,
		Latest: ManifestLatest{
			Version:    "2.1.1",
			ReleasedAt: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
			ReleaseURL: "https://github.com/shivasurya/code-pathfinder/releases/tag/v2.1.1",
		},
		Message:       ManifestMessage{Level: "info", Text: "Upgrade now"},
		Announcements: []ManifestAnnouncement{},
		MinSupported:  "1.9.0",
	}
}

// manifestWithAnnouncement returns a Manifest that has an active announcement
// and no upgrade notice for current "2.1.1".
func manifestWithAnnouncement() Manifest {
	return Manifest{
		Schema: 1,
		Latest: ManifestLatest{Version: "2.1.1"},
		Message: ManifestMessage{},
		Announcements: []ManifestAnnouncement{
			{ID: "ws1", Level: "info", Title: "Workshop", Text: "Join us on May 8"},
		},
	}
}

// TestCheck_DisableCheck verifies that opts.DisableCheck short-circuits the fetch.
func TestCheck_DisableCheck(t *testing.T) {
	// Use a closed server to confirm no HTTP call is made.
	srv := startManifestServer(t, manifestWithUpgrade())
	srv.Close() // closed immediately

	result := Check(context.Background(), "2.0.0", "cli", Options{
		DisableCheck: true,
		ManifestURL:  srv.URL,
	})
	assert.Nil(t, result)
}

// TestCheck_EnvOptOut verifies that PATHFINDER_NO_UPDATE_CHECK=1 silences the check.
func TestCheck_EnvOptOut_One(t *testing.T) {
	t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "1")

	srv := startManifestServer(t, manifestWithUpgrade())
	defer srv.Close()

	result := Check(context.Background(), "2.0.0", "cli", Options{ManifestURL: srv.URL})
	assert.Nil(t, result)
}

// TestCheck_EnvOptOut_True verifies that PATHFINDER_NO_UPDATE_CHECK=true is also honoured.
func TestCheck_EnvOptOut_True(t *testing.T) {
	t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "true")

	srv := startManifestServer(t, manifestWithUpgrade())
	defer srv.Close()

	result := Check(context.Background(), "2.0.0", "cli", Options{ManifestURL: srv.URL})
	assert.Nil(t, result)
}

// TestCheck_DefaultManifestURLAndTimeout verifies that empty ManifestURL and
// zero HTTPTimeout are filled in with sensible defaults (covers those branches).
func TestCheck_DefaultManifestURLAndTimeout(t *testing.T) {
	srv := startManifestServer(t, manifestWithUpgrade())
	defer srv.Close()

	// Temporarily redirect defaultManifestURL to the local test server.
	saved := defaultManifestURL
	defaultManifestURL = srv.URL
	defer func() { defaultManifestURL = saved }()

	// Pass empty ManifestURL and zero HTTPTimeout → both defaults applied.
	result := Check(context.Background(), "2.0.0", "cli", Options{})
	require.NotNil(t, result)
	require.NotNil(t, result.Upgrade)
	assert.Equal(t, "2.1.1", result.Upgrade.Latest)
}

// TestCheck_FetchFails_SilentReturn verifies that a network error produces nil,
// not a panic or user-visible error.
func TestCheck_FetchFails_SilentReturn(t *testing.T) {
	srv := startManifestServer(t, manifestWithUpgrade())
	srv.Close() // closed before the check runs

	result := Check(context.Background(), "2.0.0", "cli", Options{
		ManifestURL: srv.URL,
		HTTPTimeout: 200 * time.Millisecond,
	})
	assert.Nil(t, result)
}

// TestCheck_FetchFails_LoggerDebugCalled verifies the debug log path when
// the fetch fails and a Logger is provided.
func TestCheck_FetchFails_LoggerDebugCalled(t *testing.T) {
	srv := startManifestServer(t, manifestWithUpgrade())
	srv.Close()

	var buf bytes.Buffer
	logger := output.NewLoggerWithWriter(output.VerbosityDebug, &buf)

	result := Check(context.Background(), "2.0.0", "cli", Options{
		ManifestURL: srv.URL,
		HTTPTimeout: 200 * time.Millisecond,
		Logger:      logger,
	})
	assert.Nil(t, result)
	assert.Contains(t, buf.String(), "updatecheck fetch failed")
}

// TestCheck_FetchFails_NilLogger verifies that nil Logger does not panic on error.
func TestCheck_FetchFails_NilLogger(t *testing.T) {
	srv := startManifestServer(t, manifestWithUpgrade())
	srv.Close()

	result := Check(context.Background(), "2.0.0", "cli", Options{
		ManifestURL: srv.URL,
		HTTPTimeout: 200 * time.Millisecond,
		Logger:      nil,
	})
	assert.Nil(t, result)
}

// TestCheck_UpToDate_NoAnnouncements returns nil when current == latest and
// there are no announcements (nothing to surface).
func TestCheck_UpToDate_NoAnnouncements(t *testing.T) {
	srv := startManifestServer(t, manifestWithUpgrade())
	defer srv.Close()

	result := Check(context.Background(), "2.1.1", "cli", Options{ManifestURL: srv.URL})
	assert.Nil(t, result)
}

// TestCheck_UpgradeOnly returns a Result with only Upgrade set.
func TestCheck_UpgradeOnly(t *testing.T) {
	srv := startManifestServer(t, manifestWithUpgrade())
	defer srv.Close()

	result := Check(context.Background(), "2.0.0", "cli", Options{ManifestURL: srv.URL})
	require.NotNil(t, result)
	require.NotNil(t, result.Upgrade)
	assert.Nil(t, result.Announcement)
	assert.Equal(t, "2.0.0", result.Upgrade.Current)
	assert.Equal(t, "2.1.1", result.Upgrade.Latest)
}

// TestCheck_AnnouncementOnly returns a Result with only Announcement set.
func TestCheck_AnnouncementOnly(t *testing.T) {
	srv := startManifestServer(t, manifestWithAnnouncement())
	defer srv.Close()

	// current == latest → no upgrade; announcement present → Result returned.
	result := Check(context.Background(), "2.1.1", "cli", Options{ManifestURL: srv.URL})
	require.NotNil(t, result)
	assert.Nil(t, result.Upgrade)
	require.NotNil(t, result.Announcement)
	assert.Equal(t, "ws1", result.Announcement.ID)
}

// TestCheck_BothUpgradeAndAnnouncement returns both fields non-nil.
func TestCheck_BothUpgradeAndAnnouncement(t *testing.T) {
	m := Manifest{
		Schema: 1,
		Latest: ManifestLatest{Version: "2.1.1"},
		Announcements: []ManifestAnnouncement{
			{ID: "ann1", Level: "info", Title: "T", Text: "X"},
		},
	}
	srv := startManifestServer(t, m)
	defer srv.Close()

	result := Check(context.Background(), "2.0.0", "cli", Options{ManifestURL: srv.URL})
	require.NotNil(t, result)
	assert.NotNil(t, result.Upgrade)
	assert.NotNil(t, result.Announcement)
}

// TestCheck_AudienceRouting verifies that the audience parameter is forwarded
// to selectAnnouncement.
func TestCheck_AudienceRouting(t *testing.T) {
	m := Manifest{
		Schema: 1,
		Latest: ManifestLatest{Version: "2.1.1"},
		Announcements: []ManifestAnnouncement{
			{ID: "cli-only", Level: "info", Title: "T", Text: "X", Audience: "cli"},
		},
	}
	srv := startManifestServer(t, m)
	defer srv.Close()

	// CLI audience → announcement visible.
	cliResult := Check(context.Background(), "2.1.1", "cli", Options{ManifestURL: srv.URL})
	require.NotNil(t, cliResult)
	assert.Equal(t, "cli-only", cliResult.Announcement.ID)

	// MCP audience → announcement filtered out → nil (up-to-date, no eligible ann).
	mcpResult := Check(context.Background(), "2.1.1", "mcp", Options{ManifestURL: srv.URL})
	assert.Nil(t, mcpResult)
}

// TestEnvOptOut covers the three cases of envOptOut.
func TestEnvOptOut(t *testing.T) {
	t.Run("unset", func(t *testing.T) {
		t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "")
		assert.False(t, envOptOut())
	})
	t.Run("set_to_1", func(t *testing.T) {
		t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "1")
		assert.True(t, envOptOut())
	})
	t.Run("set_to_true", func(t *testing.T) {
		t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "true")
		assert.True(t, envOptOut())
	})
	t.Run("other_value", func(t *testing.T) {
		t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "yes")
		assert.False(t, envOptOut())
	})
}
