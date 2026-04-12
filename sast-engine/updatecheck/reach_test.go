package updatecheck

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReachReporter_FirstCallReturnsTrue(t *testing.T) {
	r := NewReachReporter()
	assert.True(t, r.ShouldReport("upgrade:2.0.0"))
}

func TestReachReporter_SecondCallWithinWindowReturnsFalse(t *testing.T) {
	r := NewReachReporter()
	assert.True(t, r.ShouldReport("upgrade:2.0.0"))
	assert.False(t, r.ShouldReport("upgrade:2.0.0"), "duplicate within window should be suppressed")
}

func TestReachReporter_DistinctKeysAreIndependent(t *testing.T) {
	r := NewReachReporter()
	assert.True(t, r.ShouldReport("upgrade:2.0.0"))
	assert.True(t, r.ShouldReport("upgrade:3.0.0"), "different key must not be affected by previous")
	assert.True(t, r.ShouldReport("announcement:ann-1"))
}

func TestReachReporter_KeyAfterWindowExpiresReturnsTrue(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	r := &ReachReporter{
		seen:   make(map[string]time.Time),
		window: 24 * time.Hour,
		now:    func() time.Time { return now },
	}

	assert.True(t, r.ShouldReport("upgrade:2.0.0"))
	assert.False(t, r.ShouldReport("upgrade:2.0.0"))

	// Advance clock past the 24-hour window.
	now = now.Add(25 * time.Hour)
	assert.True(t, r.ShouldReport("upgrade:2.0.0"), "key should be eligible again after window expires")
}

func TestReachReporter_KeyAtWindowBoundaryReturnsFalse(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	r := &ReachReporter{
		seen:   make(map[string]time.Time),
		window: 24 * time.Hour,
		now:    func() time.Time { return now },
	}

	r.ShouldReport("key")
	now = now.Add(23 * time.Hour)
	assert.False(t, r.ShouldReport("key"), "key at 23h should still be within the 24h window")
}

func TestReachReporter_ExactWindowBoundaryIsEligible(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	r := &ReachReporter{
		seen:   make(map[string]time.Time),
		window: 24 * time.Hour,
		now:    func() time.Time { return now },
	}

	r.ShouldReport("key")
	// Advance to exactly the window boundary — the condition is `< window`,
	// so at exactly 24h the key is eligible again.
	now = now.Add(24 * time.Hour)
	assert.True(t, r.ShouldReport("key"), "key at exactly 24h is outside the exclusive window and should be eligible")
}

func TestReachReporter_MultipleKeysTrackedIndependently(t *testing.T) {
	r := NewReachReporter()
	keys := []string{"upgrade:1.0.0", "upgrade:2.0.0", "announcement:ann-1", "announcement:ann-2"}
	for _, k := range keys {
		assert.True(t, r.ShouldReport(k), "first call for %q should return true", k)
	}
	for _, k := range keys {
		assert.False(t, r.ShouldReport(k), "second call for %q should return false", k)
	}
}

func TestReachReporter_ConcurrentSafety(t *testing.T) {
	r := NewReachReporter()
	done := make(chan bool, 10)
	for range 10 {
		go func() {
			_ = r.ShouldReport("key")
			done <- true
		}()
	}
	for range 10 {
		<-done
	}
	// No race — the important assertion is that we got here without a panic.
}
