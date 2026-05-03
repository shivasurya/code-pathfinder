package updatecheck

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- selectUpgrade ---

func TestSelectUpgrade_NewerAvailable_InfoLevel(t *testing.T) {
	m := &Manifest{
		Latest:       ManifestLatest{Version: "2.1.1", ReleaseURL: "https://example.com/v2.1.1"},
		Message:      ManifestMessage{Text: "New features"},
		MinSupported: "1.9.0",
	}
	got := selectUpgrade(m, "2.0.0")
	require.NotNil(t, got)
	assert.Equal(t, "2.0.0", got.Current)
	assert.Equal(t, "2.1.1", got.Latest)
	assert.Equal(t, "info", got.Level)
	assert.Equal(t, "New features", got.Message)
	assert.Equal(t, "https://example.com/v2.1.1", got.ReleaseURL)
}

func TestSelectUpgrade_NewerAvailable_WarnLevel_BelowMinSupported(t *testing.T) {
	m := &Manifest{
		Latest:       ManifestLatest{Version: "2.1.1"},
		MinSupported: "2.0.0",
	}
	// current = 1.8.0 < min_supported = 2.0.0 → warn
	got := selectUpgrade(m, "1.8.0")
	require.NotNil(t, got)
	assert.Equal(t, "warn", got.Level)
}

func TestSelectUpgrade_AlreadyAtLatest(t *testing.T) {
	m := &Manifest{Latest: ManifestLatest{Version: "2.1.1"}}
	assert.Nil(t, selectUpgrade(m, "2.1.1"))
}

func TestSelectUpgrade_AheadOfLatest(t *testing.T) {
	// Dev build or pre-release scenario where current > latest.
	m := &Manifest{Latest: ManifestLatest{Version: "2.1.1"}}
	assert.Nil(t, selectUpgrade(m, "2.2.0"))
}

func TestSelectUpgrade_MalformedCurrentVersion(t *testing.T) {
	// "dev" is not valid semver; Compare returns 0 → treated as equal → nil.
	m := &Manifest{Latest: ManifestLatest{Version: "2.1.1"}}
	assert.Nil(t, selectUpgrade(m, "dev"))
}

func TestSelectUpgrade_NoMinSupported_InfoLevel(t *testing.T) {
	// min_supported is empty → always "info" regardless of how old current is.
	m := &Manifest{
		Latest:       ManifestLatest{Version: "3.0.0"},
		MinSupported: "",
	}
	got := selectUpgrade(m, "1.0.0")
	require.NotNil(t, got)
	assert.Equal(t, "info", got.Level)
}

func TestSelectUpgrade_ReleasedAtPropagated(t *testing.T) {
	ts := time.Date(2026, 4, 10, 18, 22, 0, 0, time.UTC)
	m := &Manifest{
		Latest: ManifestLatest{
			Version:    "2.1.1",
			ReleasedAt: ts,
		},
	}
	got := selectUpgrade(m, "2.0.0")
	require.NotNil(t, got)
	assert.Equal(t, ts, got.ReleasedAt)
}

// --- selectAnnouncement ---

// now is a fixed reference time used throughout announcement tests.
var now = time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)

func TestSelectAnnouncement_EmptyList(t *testing.T) {
	m := &Manifest{Announcements: []ManifestAnnouncement{}}
	assert.Nil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
}

func TestSelectAnnouncement_NilList(t *testing.T) {
	m := &Manifest{}
	assert.Nil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
}

func TestSelectAnnouncement_AudienceFilter_All(t *testing.T) {
	// audience="all" in the announcement → visible to both "cli" and "mcp" callers.
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "info", Title: "T", Text: "X", Audience: "all"},
		},
	}
	assert.NotNil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
	assert.NotNil(t, selectAnnouncement(m, "2.0.0", "mcp", now, nil))
}

func TestSelectAnnouncement_AudienceFilter_EmptyDefaultsToAll(t *testing.T) {
	// Empty Audience field defaults to "all".
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "info", Title: "T", Text: "X", Audience: ""},
		},
	}
	assert.NotNil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
	assert.NotNil(t, selectAnnouncement(m, "2.0.0", "mcp", now, nil))
}

func TestSelectAnnouncement_AudienceFilter_CLIOnly(t *testing.T) {
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "info", Title: "T", Text: "X", Audience: "cli"},
		},
	}
	assert.NotNil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
	assert.Nil(t, selectAnnouncement(m, "2.0.0", "mcp", now, nil))
}

func TestSelectAnnouncement_AudienceFilter_MCPOnly(t *testing.T) {
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "info", Title: "T", Text: "X", Audience: "mcp"},
		},
	}
	assert.Nil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
	assert.NotNil(t, selectAnnouncement(m, "2.0.0", "mcp", now, nil))
}

func TestSelectAnnouncement_TimeWindow_StartsInFuture(t *testing.T) {
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "info", Title: "T", Text: "X", StartsAt: now.Add(time.Hour)},
		},
	}
	assert.Nil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
}

func TestSelectAnnouncement_TimeWindow_StartsNowOrPast(t *testing.T) {
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			// Starts exactly at now — time.Before(starts_at) is false → included.
			{ID: "a1", Level: "info", Title: "T", Text: "X", StartsAt: now},
		},
	}
	assert.NotNil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
}

func TestSelectAnnouncement_TimeWindow_Expired(t *testing.T) {
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "info", Title: "T", Text: "X", ExpiresAt: now.Add(-time.Minute)},
		},
	}
	assert.Nil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
}

func TestSelectAnnouncement_TimeWindow_NotYetExpired(t *testing.T) {
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "info", Title: "T", Text: "X", ExpiresAt: now.Add(time.Hour)},
		},
	}
	assert.NotNil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
}

func TestSelectAnnouncement_TimeWindow_ZeroValuesAlwaysPass(t *testing.T) {
	// Zero StartsAt and ExpiresAt mean "no window restriction".
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "info", Title: "T", Text: "X"},
		},
	}
	assert.NotNil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
}

func TestSelectAnnouncement_DismissedIDsFilter_Nil(t *testing.T) {
	// nil dismissed map → no filtering, announcement is visible.
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "info", Title: "T", Text: "X"},
		},
	}
	assert.NotNil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
}

func TestSelectAnnouncement_DismissedIDsFilter_IDPresent(t *testing.T) {
	// ID is in the dismissed set → announcement is skipped.
	dismissed := map[string]struct{}{"a1": {}}
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "info", Title: "T", Text: "X"},
		},
	}
	assert.Nil(t, selectAnnouncement(m, "2.0.0", "cli", now, dismissed))
}

func TestSelectAnnouncement_DismissedIDsFilter_IDAbsent(t *testing.T) {
	// Dismissed map is non-nil but doesn't contain this ID → announcement passes.
	dismissed := map[string]struct{}{"other": {}}
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "info", Title: "T", Text: "X"},
		},
	}
	assert.NotNil(t, selectAnnouncement(m, "2.0.0", "cli", now, dismissed))
}

func TestSelectAnnouncement_VersionRange_Generic(t *testing.T) {
	// Empty version_range → generic, shown to every version.
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "info", Title: "T", Text: "X", VersionRange: ""},
		},
	}
	assert.NotNil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
}

func TestSelectAnnouncement_VersionRange_Matches(t *testing.T) {
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "warn", Title: "T", Text: "X", VersionRange: "<2.0.0"},
		},
	}
	assert.NotNil(t, selectAnnouncement(m, "1.9.9", "cli", now, nil))
}

func TestSelectAnnouncement_VersionRange_DoesNotMatch(t *testing.T) {
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "warn", Title: "T", Text: "X", VersionRange: "<2.0.0"},
		},
	}
	assert.Nil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
}

func TestSelectAnnouncement_VersionRange_MalformedSilentlySkipped(t *testing.T) {
	// A malformed version_range must never crash and must be silently skipped.
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "a1", Level: "critical", Title: "T", Text: "X", VersionRange: "bogus-range"},
		},
	}
	assert.Nil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
}

// TestSelectAnnouncement_Priority_AllSixTiers verifies the full priority
// ordering: critical-targeted > critical-generic > warn-targeted > warn-generic >
// info-targeted > info-generic.
func TestSelectAnnouncement_Priority_AllSixTiers(t *testing.T) {
	// Feed all six tiers in reversed priority order; the selector must always
	// return the highest-priority one.
	makeAnn := func(id, level string, targeted bool) ManifestAnnouncement {
		vr := ""
		if targeted {
			vr = ">=1.0.0" // matches current = "2.0.0"
		}
		return ManifestAnnouncement{ID: id, Level: level, Title: id, Text: id, VersionRange: vr}
	}

	tiers := []ManifestAnnouncement{
		makeAnn("info-generic", "info", false),        // tier 6
		makeAnn("info-targeted", "info", true),        // tier 5
		makeAnn("warn-generic", "warn", false),        // tier 4
		makeAnn("warn-targeted", "warn", true),        // tier 3
		makeAnn("critical-generic", "critical", false), // tier 2
		makeAnn("critical-targeted", "critical", true), // tier 1 (highest)
	}
	m := &Manifest{Announcements: tiers}

	got := selectAnnouncement(m, "2.0.0", "cli", now, nil)
	require.NotNil(t, got)
	assert.Equal(t, "critical-targeted", got.ID)
}

func TestSelectAnnouncement_Priority_TargetedOverGeneric_SameLevel(t *testing.T) {
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "generic", Level: "warn", Title: "G", Text: "G"},
			{ID: "targeted", Level: "warn", Title: "T", Text: "T", VersionRange: ">=1.0.0"},
		},
	}
	got := selectAnnouncement(m, "2.0.0", "cli", now, nil)
	require.NotNil(t, got)
	assert.Equal(t, "targeted", got.ID)
}

func TestSelectAnnouncement_TieBreak_EarliestStartsAt(t *testing.T) {
	earlier := now.Add(-2 * time.Hour)
	later := now.Add(-time.Hour)
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "b", Level: "info", Title: "B", Text: "B", StartsAt: later},
			{ID: "a", Level: "info", Title: "A", Text: "A", StartsAt: earlier},
		},
	}
	got := selectAnnouncement(m, "2.0.0", "cli", now, nil)
	require.NotNil(t, got)
	assert.Equal(t, "a", got.ID)
}

func TestSelectAnnouncement_TieBreak_LexicalID(t *testing.T) {
	// Same level, same targeted-ness, same StartsAt → lexically first ID wins.
	ts := now.Add(-time.Hour)
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{ID: "z-ann", Level: "info", Title: "Z", Text: "Z", StartsAt: ts},
			{ID: "a-ann", Level: "info", Title: "A", Text: "A", StartsAt: ts},
		},
	}
	got := selectAnnouncement(m, "2.0.0", "cli", now, nil)
	require.NotNil(t, got)
	assert.Equal(t, "a-ann", got.ID)
}

func TestSelectAnnouncement_FieldsPropagate(t *testing.T) {
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			{
				ID:           "ann1",
				Level:        "critical",
				Title:        "Security advisory",
				Text:         "Please upgrade immediately.",
				URL:          "https://example.com/advisory",
				Audience:     "all",
				VersionRange: "<2.0.0",
			},
		},
	}
	got := selectAnnouncement(m, "1.9.9", "cli", now, nil)
	require.NotNil(t, got)
	assert.Equal(t, "ann1", got.ID)
	assert.Equal(t, "critical", got.Level)
	assert.Equal(t, "Security advisory", got.Title)
	assert.Equal(t, "Please upgrade immediately.", got.Text)
	assert.Equal(t, "https://example.com/advisory", got.URL)
	assert.Equal(t, "all", got.Audience)
	assert.Equal(t, "<2.0.0", got.VersionRange)
}

func TestSelectAnnouncement_AllFilteredOut(t *testing.T) {
	m := &Manifest{
		Announcements: []ManifestAnnouncement{
			// Wrong audience
			{ID: "a1", Level: "critical", Title: "T", Text: "X", Audience: "mcp"},
			// Not yet started
			{ID: "a2", Level: "critical", Title: "T", Text: "X", StartsAt: now.Add(time.Hour)},
			// Expired
			{ID: "a3", Level: "critical", Title: "T", Text: "X", ExpiresAt: now.Add(-time.Minute)},
		},
	}
	assert.Nil(t, selectAnnouncement(m, "2.0.0", "cli", now, nil))
}

func TestLevelIndex(t *testing.T) {
	assert.Equal(t, 0, levelIndex("critical"))
	assert.Equal(t, 1, levelIndex("warn"))
	assert.Equal(t, 2, levelIndex("info"))
	assert.Equal(t, 3, levelIndex("unknown"))
	assert.Equal(t, 3, levelIndex(""))
}
