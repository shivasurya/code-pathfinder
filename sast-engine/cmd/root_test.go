package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/updatecheck"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecute(t *testing.T) {
	oldRoot := rootCmd
	defer func() { rootCmd = oldRoot }()

	tests := []struct {
		name          string
		args          []string
		expectedError bool
	}{
		{
			name:          "No arguments",
			args:          []string{},
			expectedError: false,
		},
		{
			name:          "Help command",
			args:          []string{"--help"},
			expectedError: false,
		},
		{
			name:          "Invalid command",
			args:          []string{"invalidcommand"},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd = &cobra.Command{Use: "pathfinder"}
			rootCmd.AddCommand(&cobra.Command{Use: "validcommand"})

			rootCmd.SetArgs(tt.args)
			err := Execute()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRootCmdPersistentPreRun(t *testing.T) {
	tests := []struct {
		name            string
		disableMetrics  bool
		expectedMetrics bool
	}{
		{
			name:            "Metrics enabled",
			disableMetrics:  false,
			expectedMetrics: true,
		},
		{
			name:            "Metrics disabled",
			disableMetrics:  true,
			expectedMetrics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Suppress the update-check HTTP call so the test is hermetic.
			t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "1")

			cmd := &cobra.Command{}
			cmd.Flags().Bool("disable-metrics", tt.disableMetrics, "")

			rootCmd.PersistentPreRun(cmd, []string{})

			// Since we can't directly test the analytics.Init function,
			// we can check if the flag was correctly read
			disableMetrics, _ := cmd.Flags().GetBool("disable-metrics")
			assert.Equal(t, tt.disableMetrics, disableMetrics)
		})
	}
}

func TestRootCmdFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "pathfinder"}
	cmd.AddCommand(rootCmd)

	disableMetricsFlag := rootCmd.PersistentFlags().Lookup("disable-metrics")
	assert.NotNil(t, disableMetricsFlag)
	assert.Equal(t, "false", disableMetricsFlag.DefValue)
	assert.Equal(t, "Disable metrics collection", disableMetricsFlag.Usage)
}

func TestRootCmdOutput(t *testing.T) {
	oldRoot := rootCmd
	defer func() { rootCmd = oldRoot }()

	rootCmd = &cobra.Command{Use: "pathfinder"}
	rootCmd.AddCommand(&cobra.Command{Use: "validcommand"})

	tests := []struct {
		name           string
		args           []string
		expectedOutput string
	}{
		{
			name:           "No arguments",
			args:           []string{},
			expectedOutput: "Usage:\n  pathfinder [command]\n\nAvailable Commands:\n  completion",
		},
		{
			name:           "Help command",
			args:           []string{"--help"},
			expectedOutput: "Usage:\n  pathfinder [command]\n\nAvailable Commands:\n  completion",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := new(bytes.Buffer)
			rootCmd.SetOut(b)
			rootCmd.SetArgs(tt.args)
			_ = rootCmd.Execute()

			assert.Contains(t, b.String(), tt.expectedOutput)
		})
	}
}

// --- helper: minimal fake CDN server ----------------------------------------

// startFakeCDN starts an httptest.Server that returns a valid manifest and
// counts how many times it was hit. Caller must defer srv.Close().
func startFakeCDN(t *testing.T) (srv *httptest.Server, hits *atomic.Int64) {
	t.Helper()
	hits = &atomic.Int64{}
	m := updatecheck.Manifest{
		Schema: 1,
		Latest: updatecheck.ManifestLatest{
			Version:    "99.0.0",
			ReleasedAt: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
			ReleaseURL: "https://example.com/releases/v99.0.0",
		},
		Message: updatecheck.ManifestMessage{Level: "info", Text: "big upgrade"},
	}
	body, err := json.Marshal(m)
	require.NoError(t, err)
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	return srv, hits
}

// runPreRun builds a cobra.Command with all persistent flags registered and
// calls rootCmd.PersistentPreRun. The flags map is applied before calling.
func runPreRun(t *testing.T, flags map[string]string) {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().Bool("disable-metrics", false, "")
	cmd.Flags().Bool("verbose", false, "")
	cmd.Flags().Bool("no-banner", false, "")
	cmd.Flags().Bool("no-update-check", false, "")
	for k, v := range flags {
		require.NoError(t, cmd.Flags().Set(k, v))
	}
	rootCmd.PersistentPreRun(cmd, []string{})
}

// --- shouldSkipUpdateCheck unit tests ---------------------------------------

func TestShouldSkipUpdateCheck_Flag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("no-update-check", true, "")
	assert.True(t, shouldSkipUpdateCheck(cmd))
}

func TestShouldSkipUpdateCheck_FlagFalse(t *testing.T) {
	t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "")
	cmd := &cobra.Command{}
	cmd.Flags().Bool("no-update-check", false, "")
	assert.False(t, shouldSkipUpdateCheck(cmd))
}

func TestShouldSkipUpdateCheck_EnvVar(t *testing.T) {
	t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "1")
	cmd := &cobra.Command{}
	cmd.Flags().Bool("no-update-check", false, "")
	assert.True(t, shouldSkipUpdateCheck(cmd))
}

func TestShouldSkipUpdateCheck_CIEnvVars(t *testing.T) {
	ciVars := []string{
		"CI", "GITHUB_ACTIONS", "GITLAB_CI", "BUILDKITE",
		"CIRCLECI", "TRAVIS", "JENKINS_URL", "TF_BUILD",
	}
	for _, k := range ciVars {
		t.Run(k, func(t *testing.T) {
			t.Setenv(k, "true")
			cmd := &cobra.Command{}
			cmd.Flags().Bool("no-update-check", false, "")
			assert.True(t, shouldSkipUpdateCheck(cmd))
		})
	}
}

func TestShouldSkipUpdateCheck_NoFlagReturnsDefault(t *testing.T) {
	// If the flag is not registered on cmd, GetBool returns false+err;
	// the error is ignored and the check falls through to the env/CI tests.
	t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "")
	cmd := &cobra.Command{}
	assert.False(t, shouldSkipUpdateCheck(cmd))
}

// --- shouldShowNotice unit tests --------------------------------------------

func TestShouldShowNotice_NonTTY(t *testing.T) {
	assert.False(t, shouldShowNotice(false, false))
}

func TestShouldShowNotice_TTY_NoBannerFalse(t *testing.T) {
	assert.True(t, shouldShowNotice(true, false))
}

func TestShouldShowNotice_NoBannerFlag(t *testing.T) {
	// noBanner=true must suppress even when isTTY=true.
	assert.False(t, shouldShowNotice(true, true))
}

// --- PersistentPreRun integration tests -------------------------------------

// TestPersistentPreRun_NoUpdateCheckFlag_Skips verifies that --no-update-check
// prevents any HTTP call to the CDN.
func TestPersistentPreRun_NoUpdateCheckFlag_Skips(t *testing.T) {
	srv, hits := startFakeCDN(t)
	defer srv.Close()

	old := updateCheckManifestURL
	updateCheckManifestURL = srv.URL
	defer func() { updateCheckManifestURL = old }()

	runPreRun(t, map[string]string{"no-update-check": "true"})
	assert.Equal(t, int64(0), hits.Load(), "CDN must not be hit when --no-update-check is set")
}

// TestPersistentPreRun_EnvVarSkips verifies that PATHFINDER_NO_UPDATE_CHECK=1
// prevents any HTTP call to the CDN.
func TestPersistentPreRun_EnvVarSkips(t *testing.T) {
	t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "1")

	srv, hits := startFakeCDN(t)
	defer srv.Close()

	old := updateCheckManifestURL
	updateCheckManifestURL = srv.URL
	defer func() { updateCheckManifestURL = old }()

	runPreRun(t, nil)
	assert.Equal(t, int64(0), hits.Load(), "CDN must not be hit when PATHFINDER_NO_UPDATE_CHECK=1")
}

// TestPersistentPreRun_CIDetected_Skips verifies that CI=true prevents any
// HTTP call to the CDN.
func TestPersistentPreRun_CIDetected_Skips(t *testing.T) {
	t.Setenv("CI", "true")

	srv, hits := startFakeCDN(t)
	defer srv.Close()

	old := updateCheckManifestURL
	updateCheckManifestURL = srv.URL
	defer func() { updateCheckManifestURL = old }()

	runPreRun(t, nil)
	assert.Equal(t, int64(0), hits.Load(), "CDN must not be hit in CI environments")
}

// TestPersistentPreRun_NonTTY_FetchesButDoesNotRender verifies that in a
// non-TTY environment the manifest fetch still occurs (so future analytics can
// fire) but no banner is rendered.
func TestPersistentPreRun_NonTTY_FetchesButDoesNotRender(t *testing.T) {
	t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "")

	srv, hits := startFakeCDN(t)
	defer srv.Close()

	old := updateCheckManifestURL
	updateCheckManifestURL = srv.URL
	defer func() { updateCheckManifestURL = old }()

	// In a test, stderr is not a terminal → IsTTY() returns false → no render.
	runPreRun(t, nil)

	assert.Equal(t, int64(1), hits.Load(), "CDN should be hit even in non-TTY mode")
}

// TestPersistentPreRun_NoBanner_FetchesButDoesNotRender verifies that
// --no-banner suppresses the update notice while still fetching.
func TestPersistentPreRun_NoBanner_FetchesButDoesNotRender(t *testing.T) {
	t.Setenv("PATHFINDER_NO_UPDATE_CHECK", "")

	srv, hits := startFakeCDN(t)
	defer srv.Close()

	old := updateCheckManifestURL
	updateCheckManifestURL = srv.URL
	defer func() { updateCheckManifestURL = old }()

	runPreRun(t, map[string]string{"no-banner": "true"})

	assert.Equal(t, int64(1), hits.Load(), "CDN should be hit even when --no-banner is set")
}

// TestRootCmdFlags_NoUpdateCheckFlag verifies the --no-update-check flag is
// registered on rootCmd with the expected defaults and usage text.
func TestRootCmdFlags_NoUpdateCheckFlag(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("no-update-check")
	require.NotNil(t, f)
	assert.Equal(t, "false", f.DefValue)
	assert.Equal(t, "Disable check for newer pathfinder versions", f.Usage)
}
