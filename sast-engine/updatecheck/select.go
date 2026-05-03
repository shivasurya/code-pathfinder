package updatecheck

import (
	"sort"
	"time"
)

// levelIndex converts a level string to a numeric priority.
// Lower index = higher priority.
func levelIndex(level string) int {
	switch level {
	case "critical":
		return 0
	case "warn":
		return 1
	case "info":
		return 2
	default:
		return 3
	}
}

// selectUpgrade returns an UpgradeNotice when current is older than
// m.Latest.Version, or nil when current is already up to date.
// Level is set to "warn" when current falls below m.MinSupported.
func selectUpgrade(m *Manifest, current string) *UpgradeNotice {
	if Compare(current, m.Latest.Version) >= 0 {
		return nil
	}
	level := "info"
	if m.MinSupported != "" && Compare(current, m.MinSupported) < 0 {
		level = "warn"
	}
	return &UpgradeNotice{
		Current:    current,
		Latest:     m.Latest.Version,
		ReleasedAt: m.Latest.ReleasedAt,
		ReleaseURL: m.Latest.ReleaseURL,
		Level:      level,
		Message:    m.Message.Text,
	}
}

// selectAnnouncement picks the single highest-priority announcement from
// m.Announcements that passes all filters.
//
// Priority order (lower = higher priority):
//  1. critical + version-targeted
//  2. critical + generic
//  3. warn + version-targeted
//  4. warn + generic
//  5. info + version-targeted
//  6. info + generic
//
// Within a tier, the announcement with the earliest StartsAt wins; ties are
// broken by lexical ID. Returns nil when no announcement qualifies.
//
// dismissed is an optional set of IDs the caller has already shown; entries
// whose IDs are in dismissed are skipped. Pass nil to disable this filter
// (used by the CLI, which does not track dismissed IDs across invocations).
func selectAnnouncement(m *Manifest, current, audience string, now time.Time, dismissed map[string]struct{}) *Announcement {
	type candidate struct {
		ann      *ManifestAnnouncement
		targeted bool
	}

	var pool []candidate

	for i := range m.Announcements {
		ma := &m.Announcements[i]

		// Audience filter: empty defaults to "all".
		aud := ma.Audience
		if aud == "" {
			aud = "all"
		}
		if aud != "all" && aud != audience {
			continue
		}

		// Time-window filter.
		if !ma.StartsAt.IsZero() && now.Before(ma.StartsAt) {
			continue
		}
		if !ma.ExpiresAt.IsZero() && now.After(ma.ExpiresAt) {
			continue
		}

		// Dismissed-IDs filter.
		if dismissed != nil {
			if _, ok := dismissed[ma.ID]; ok {
				continue
			}
		}

		// Version-range filter.
		targeted := ma.VersionRange != ""
		if targeted {
			ok, err := Match(ma.VersionRange, current)
			if err != nil || !ok {
				continue
			}
		}

		pool = append(pool, candidate{ann: ma, targeted: targeted})
	}

	if len(pool) == 0 {
		return nil
	}

	// Sort by (levelIndex asc, targeted desc, StartsAt asc, ID asc).
	sort.Slice(pool, func(i, j int) bool {
		ci, cj := pool[i], pool[j]
		li, lj := levelIndex(ci.ann.Level), levelIndex(cj.ann.Level)
		if li != lj {
			return li < lj
		}
		// targeted outranks generic at the same level
		if ci.targeted != cj.targeted {
			return ci.targeted && !cj.targeted
		}
		if !ci.ann.StartsAt.Equal(cj.ann.StartsAt) {
			return ci.ann.StartsAt.Before(cj.ann.StartsAt)
		}
		return ci.ann.ID < cj.ann.ID
	})

	best := pool[0].ann
	return &Announcement{
		ID:           best.ID,
		Level:        best.Level,
		Title:        best.Title,
		Text:         best.Text,
		URL:          best.URL,
		Audience:     best.Audience,
		VersionRange: best.VersionRange,
	}
}
