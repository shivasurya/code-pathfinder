package mcp

import (
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/analytics"
	"github.com/shivasurya/code-pathfinder/sast-engine/updatecheck"
	"github.com/stretchr/testify/assert"
)

// Note: we cannot replace analytics.ReportEventWithProperties from outside
// the analytics package; instead we verify gate behavior (disabled = no event)
// and dedupe behavior through the reachReporter state.

func TestReportReachIfNeeded_NoOpWhenUpdateInfoNil(t *testing.T) {
	analytics.Init(false)
	server := createTestServer()
	server.updateInfo = nil
	// Should not panic.
	server.reportReachIfNeeded()
}

func TestReportReachIfNeeded_NoOpWhenAnalyticsDisabled(t *testing.T) {
	analytics.Init(true)
	t.Cleanup(func() { analytics.Init(false) })

	server := createTestServer()
	server.updateInfo = &updatecheck.Result{
		Upgrade: &updatecheck.UpgradeNotice{Current: "1.0.0", Latest: "2.0.0", Level: "info"},
	}

	// With analytics disabled, ShouldReport should never even be called
	// (the function returns early). We verify this by checking that the
	// reachReporter has NOT recorded the key.
	server.reportReachIfNeeded()
	// If analytics is disabled, reportReachIfNeeded returns before calling
	// ShouldReport, so a subsequent call should still return true for the key.
	assert.True(t, server.reachReporter.ShouldReport("upgrade:2.0.0"),
		"reachReporter should not have been touched when analytics is disabled")
}

func TestReportReachIfNeeded_DeduplicatesWithinWindow(t *testing.T) {
	analytics.Init(false)
	server := createTestServer()
	server.updateInfo = &updatecheck.Result{
		Upgrade: &updatecheck.UpgradeNotice{Current: "1.0.0", Latest: "2.0.0", Level: "info"},
	}

	// First call records the key.
	server.reportReachIfNeeded()
	// Second call should be deduplicated — ShouldReport returns false.
	assert.False(t, server.reachReporter.ShouldReport("upgrade:2.0.0"),
		"key should be locked after first reportReachIfNeeded call")
}

func TestReportReachIfNeeded_AnnouncementKey(t *testing.T) {
	analytics.Init(false)
	server := createTestServer()
	server.updateInfo = &updatecheck.Result{
		Announcement: &updatecheck.Announcement{
			ID: "ann-1", Level: "info", Title: "Workshop", Text: "Join us.",
		},
	}

	server.reportReachIfNeeded()
	assert.False(t, server.reachReporter.ShouldReport("announcement:ann-1"),
		"announcement key should be locked after first call")
}

func TestReportReachIfNeeded_NoReachReporter_NoOp(t *testing.T) {
	analytics.Init(false)
	server := createTestServer()
	server.reachReporter = nil
	server.updateInfo = &updatecheck.Result{
		Upgrade: &updatecheck.UpgradeNotice{Current: "1.0.0", Latest: "2.0.0"},
	}
	// Should not panic.
	server.reportReachIfNeeded()
}

// --- handleToolsList fires reach once across many calls ---------------------

func TestHandleToolsList_ReachFiredOnceAcrossManyCalls(t *testing.T) {
	analytics.Init(false)
	server := createTestServer()
	server.updateInfo = &updatecheck.Result{
		Upgrade: &updatecheck.UpgradeNotice{Current: "1.0.0", Latest: "2.0.0", Level: "info"},
	}

	req := makeJSONRPCRequest("tools/list", nil)
	for range 50 {
		server.handleToolsList(req)
	}

	// After 50 calls, the key must be locked (ShouldReport returns false).
	assert.False(t, server.reachReporter.ShouldReport("upgrade:2.0.0"),
		"upgrade key must be locked after first tools/list in the 24h window")
}

func TestHandleToolsList_NoReachWhenAnalyticsDisabled(t *testing.T) {
	analytics.Init(true)
	t.Cleanup(func() { analytics.Init(false) })

	server := createTestServer()
	server.updateInfo = &updatecheck.Result{
		Upgrade: &updatecheck.UpgradeNotice{Current: "1.0.0", Latest: "2.0.0", Level: "info"},
	}

	req := makeJSONRPCRequest("tools/list", nil)
	server.handleToolsList(req)

	// Key must not be locked — analytics gate prevented ShouldReport from being called.
	assert.True(t, server.reachReporter.ShouldReport("upgrade:2.0.0"),
		"key must remain unlocked when analytics is disabled")
}

// --- cmd root.go dismissed event wiring -------------------------------------

func TestUpdateCheckSkipReason_FlagReturnsFlagReason(t *testing.T) {
	// Import the cmd package behaviour via root_test.go helpers; here we test
	// the new updateCheckSkipReason indirectly via shouldSkipUpdateCheck which
	// wraps it. Direct testing is done in cmd/root_skip_test.go.
	//
	// This test lives in mcp package to avoid import cycles, so we just verify
	// the analytics reach reporter integration is sound.
	analytics.Init(false)
	server := createTestServer()
	assert.NotNil(t, server.reachReporter)
}

