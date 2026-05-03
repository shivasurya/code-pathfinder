package output

import (
	"bytes"
	"testing"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/analytics"
	"github.com/shivasurya/code-pathfinder/sast-engine/updatecheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturedEvent records a single analytics.ReportEventWithProperties call.
type capturedEvent struct {
	event string
	props map[string]any
}

// withAnalyticsStub replaces the package-level reportEvent and defaultReporter
// with test doubles, returning the captured events slice. Caller must defer the
// cleanup returned.
func withAnalyticsStub(t *testing.T) (*[]capturedEvent, func()) {
	t.Helper()

	var captured []capturedEvent
	oldReportEvent := reportEvent
	oldReporter := defaultReporter

	reportEvent = func(event string, props map[string]any) {
		captured = append(captured, capturedEvent{event: event, props: props})
	}
	// Fresh reporter per test so dedupe state doesn't bleed between cases.
	defaultReporter = updatecheck.NewReachReporter()

	cleanup := func() {
		reportEvent = oldReportEvent
		defaultReporter = oldReporter
	}
	return &captured, cleanup
}

// --- PrintUpdateNotice analytics tests --------------------------------------

func TestPrintUpdateNotice_FiresAnalyticsEvent(t *testing.T) {
	analytics.Init(false) // metrics enabled
	captured, cleanup := withAnalyticsStub(t)
	defer cleanup()

	n := &updatecheck.UpgradeNotice{
		Current: "1.0.0", Latest: "2.0.0",
		ReleasedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ReleaseURL: "https://example.com/releases/v2.0.0",
		Level:      "info",
	}
	printUpdateNotice(&bytes.Buffer{}, n, false)

	require.Len(t, *captured, 1)
	ev := (*captured)[0]
	assert.Equal(t, "update_notice_shown", ev.event)
	assert.Equal(t, "1.0.0", ev.props["current"])
	assert.Equal(t, "2.0.0", ev.props["latest"])
	assert.Equal(t, "info", ev.props["level"])
	assert.Equal(t, "cli", ev.props["surface"])
	assert.Equal(t, false, ev.props["triggered_min_supported"])
}

func TestPrintUpdateNotice_WarnLevelSetsTriggeredMinSupported(t *testing.T) {
	analytics.Init(false)
	captured, cleanup := withAnalyticsStub(t)
	defer cleanup()

	n := &updatecheck.UpgradeNotice{Current: "1.0.0", Latest: "2.0.0", Level: "warn"}
	printUpdateNotice(&bytes.Buffer{}, n, false)

	require.Len(t, *captured, 1)
	assert.Equal(t, true, (*captured)[0].props["triggered_min_supported"])
}

func TestPrintUpdateNotice_NoAnalyticsWhenDisabled(t *testing.T) {
	analytics.Init(true) // metrics disabled
	t.Cleanup(func() { analytics.Init(false) })
	captured, cleanup := withAnalyticsStub(t)
	defer cleanup()

	n := &updatecheck.UpgradeNotice{Current: "1.0.0", Latest: "2.0.0", Level: "info"}
	printUpdateNotice(&bytes.Buffer{}, n, false)

	assert.Empty(t, *captured, "no analytics call expected when metrics are disabled")
}

func TestPrintUpdateNotice_DedupeWithinWindow(t *testing.T) {
	analytics.Init(false)
	captured, cleanup := withAnalyticsStub(t)
	defer cleanup()

	n := &updatecheck.UpgradeNotice{Current: "1.0.0", Latest: "2.0.0", Level: "info"}
	printUpdateNotice(&bytes.Buffer{}, n, false)
	printUpdateNotice(&bytes.Buffer{}, n, false) // second call — same latest version

	assert.Len(t, *captured, 1, "second render of same upgrade should be deduplicated")
}

func TestPrintUpdateNotice_DifferentVersionsFireSeparateEvents(t *testing.T) {
	analytics.Init(false)
	captured, cleanup := withAnalyticsStub(t)
	defer cleanup()

	printUpdateNotice(&bytes.Buffer{}, &updatecheck.UpgradeNotice{Current: "1.0.0", Latest: "2.0.0", Level: "info"}, false)
	printUpdateNotice(&bytes.Buffer{}, &updatecheck.UpgradeNotice{Current: "1.0.0", Latest: "3.0.0", Level: "info"}, false)

	assert.Len(t, *captured, 2, "distinct latest versions should each fire once")
}

// --- PrintAnnouncement analytics tests --------------------------------------

func TestPrintAnnouncement_FiresAnalyticsEvent(t *testing.T) {
	analytics.Init(false)
	captured, cleanup := withAnalyticsStub(t)
	defer cleanup()

	a := &updatecheck.Announcement{
		ID: "ann-1", Level: "info", Title: "Workshop", Text: "Join us.",
		URL: "https://example.com/workshop",
	}
	printAnnouncement(&bytes.Buffer{}, a, false)

	require.Len(t, *captured, 1)
	ev := (*captured)[0]
	assert.Equal(t, "announcement_shown", ev.event)
	assert.Equal(t, "ann-1", ev.props["id"])
	assert.Equal(t, "info", ev.props["level"])
	assert.Equal(t, "generic", ev.props["kind"])
	assert.Equal(t, "cli", ev.props["surface"])
	assert.NotContains(t, ev.props, "version_range", "generic announcement must not include version_range")
}

func TestPrintAnnouncement_VersionTargetedIncludesRange(t *testing.T) {
	analytics.Init(false)
	captured, cleanup := withAnalyticsStub(t)
	defer cleanup()

	a := &updatecheck.Announcement{
		ID: "sec-1", Level: "warn", Title: "CVE", Text: "Patch now.",
		VersionRange: ">=1.0.0 <2.0.0",
	}
	printAnnouncement(&bytes.Buffer{}, a, false)

	require.Len(t, *captured, 1)
	ev := (*captured)[0]
	assert.Equal(t, "version_targeted", ev.props["kind"])
	assert.Equal(t, ">=1.0.0 <2.0.0", ev.props["version_range"])
}

func TestPrintAnnouncement_NoAnalyticsWhenDisabled(t *testing.T) {
	analytics.Init(true)
	t.Cleanup(func() { analytics.Init(false) })
	captured, cleanup := withAnalyticsStub(t)
	defer cleanup()

	a := &updatecheck.Announcement{ID: "ann-1", Level: "info", Title: "T", Text: "B"}
	printAnnouncement(&bytes.Buffer{}, a, false)

	assert.Empty(t, *captured)
}

func TestPrintAnnouncement_DedupeWithinWindow(t *testing.T) {
	analytics.Init(false)
	captured, cleanup := withAnalyticsStub(t)
	defer cleanup()

	a := &updatecheck.Announcement{ID: "ann-1", Level: "info", Title: "T", Text: "B"}
	printAnnouncement(&bytes.Buffer{}, a, false)
	printAnnouncement(&bytes.Buffer{}, a, false)

	assert.Len(t, *captured, 1, "second render of same announcement ID should be deduplicated")
}

// --- cmd/root.go dismissed event — tested indirectly via integration ---------
// The dismissed sampling is tested in cmd/root_test.go; here we just verify
// that the analytics package gate (IsDisabled) works as expected.

func TestAnalyticsIsDisabled_ReflectsInitState(t *testing.T) {
	analytics.Init(true)
	assert.True(t, analytics.IsDisabled())
	analytics.Init(false)
	assert.False(t, analytics.IsDisabled())
	t.Cleanup(func() { analytics.Init(false) })
}

// --- nil-safety check -------------------------------------------------------

func TestPrintUpdateNotice_NilUpgrade_NoAnalytics(t *testing.T) {
	analytics.Init(false)
	captured, cleanup := withAnalyticsStub(t)
	defer cleanup()

	// PrintUpdateNotice (exported) guards against nil.
	PrintUpdateNotice(&bytes.Buffer{}, nil)
	assert.Empty(t, *captured)
}

func TestPrintAnnouncement_NilAnnouncement_NoAnalytics(t *testing.T) {
	analytics.Init(false)
	captured, cleanup := withAnalyticsStub(t)
	defer cleanup()

	PrintAnnouncement(&bytes.Buffer{}, nil)
	assert.Empty(t, *captured)
}

var _ = require.New // ensure require import is used.
