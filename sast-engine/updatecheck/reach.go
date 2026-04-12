package updatecheck

import (
	"sync"
	"time"
)

// ReachReporter deduplicates reach analytics events within a configurable
// window (default 24 hours) per process. Both the CLI banner renderer and
// the MCP server share this logic.
//
// For the CLI, the process dies after each invocation so the dedupe is mostly
// defensive (guards against tight-loop scripts). For the MCP server, the
// process is long-lived and the dedupe prevents per-tools/list event flooding.
//
// State lives entirely in process memory — no disk file, no goroutine.
type ReachReporter struct {
	mu     sync.Mutex
	seen   map[string]time.Time
	window time.Duration
	now    func() time.Time // injectable for tests; defaults to time.Now
}

// NewReachReporter returns a ReachReporter with a 24-hour dedup window.
func NewReachReporter() *ReachReporter {
	return &ReachReporter{
		seen:   make(map[string]time.Time),
		window: 24 * time.Hour,
		now:    time.Now,
	}
}

// ShouldReport returns true when no event for key has been recorded within
// the dedup window, and records the key. Subsequent calls for the same key
// within the window return false.
//
// Key construction convention:
//
//	upgrade notice:   "upgrade:" + latest_version
//	announcement:     "announcement:" + announcement_id
func (r *ReachReporter) ShouldReport(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	if last, ok := r.seen[key]; ok && now.Sub(last) < r.window {
		return false
	}
	r.seen[key] = now
	return true
}
