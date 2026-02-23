package mcp

import (
	"sync"
	"time"
)

// IndexingState represents the current state of the indexing process.
type IndexingState int

const (
	// StateUninitialized means indexing hasn't started yet.
	StateUninitialized IndexingState = iota
	// StateIndexing means indexing is in progress.
	StateIndexing
	// StateReady means indexing is complete and server is ready.
	StateReady
	// StateFailed means indexing failed.
	StateFailed
)

// String returns the string representation of the state.
func (s IndexingState) String() string {
	switch s {
	case StateUninitialized:
		return "uninitialized"
	case StateIndexing:
		return "indexing"
	case StateReady:
		return "ready"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// IndexingPhase represents the current phase of indexing.
type IndexingPhase int

const (
	PhaseNone IndexingPhase = iota
	PhaseParsing
	PhaseModuleRegistry
	PhaseCallGraph
	PhaseComplete
)

// String returns the string representation of the phase.
func (p IndexingPhase) String() string {
	switch p {
	case PhaseNone:
		return "none"
	case PhaseParsing:
		return "parsing"
	case PhaseModuleRegistry:
		return "module_registry"
	case PhaseCallGraph:
		return "call_graph"
	case PhaseComplete:
		return "complete"
	default:
		return "unknown"
	}
}

// IndexingProgress holds detailed progress information.
type IndexingProgress struct {
	Phase           IndexingPhase `json:"phase"`
	PhaseProgress   float64       `json:"phaseProgress"`   // 0.0 to 1.0
	OverallProgress float64       `json:"overallProgress"` // 0.0 to 1.0
	FilesProcessed  int           `json:"filesProcessed"`
	TotalFiles      int           `json:"totalFiles"`
	CurrentFile     string        `json:"currentFile,omitempty"`
	Message         string        `json:"message,omitempty"`
}

// IndexingStatus holds the complete indexing status.
type IndexingStatus struct {
	State       IndexingState    `json:"state"`
	Progress    IndexingProgress `json:"progress"`
	StartedAt   *time.Time       `json:"startedAt,omitempty"`
	CompletedAt *time.Time       `json:"completedAt,omitempty"`
	Error       string           `json:"error,omitempty"`
	Stats       *IndexingStats   `json:"stats,omitempty"`
}

// IndexingStats holds statistics after indexing completes.
type IndexingStats struct {
	Functions     int           `json:"functions"`
	CallEdges     int           `json:"callEdges"`
	Modules       int           `json:"modules"`
	Files         int           `json:"files"`
	BuildDuration time.Duration `json:"buildDuration"`
}

// StatusTracker tracks and reports indexing status.
type StatusTracker struct {
	mu          sync.RWMutex
	state       IndexingState
	progress    IndexingProgress
	startedAt   *time.Time
	completedAt *time.Time
	errorMsg    string
	stats       *IndexingStats
	subscribers []chan IndexingStatus
}

// NewStatusTracker creates a new status tracker.
func NewStatusTracker() *StatusTracker {
	return &StatusTracker{
		state: StateUninitialized,
		progress: IndexingProgress{
			Phase: PhaseNone,
		},
	}
}

// GetStatus returns the current indexing status.
func (t *StatusTracker) GetStatus() IndexingStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return IndexingStatus{
		State:       t.state,
		Progress:    t.progress,
		StartedAt:   t.startedAt,
		CompletedAt: t.completedAt,
		Error:       t.errorMsg,
		Stats:       t.stats,
	}
}

// GetState returns the current indexing state.
func (t *StatusTracker) GetState() IndexingState {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state
}

// IsReady returns true if the server is ready to handle requests.
func (t *StatusTracker) IsReady() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.state == StateReady
}

// StartIndexing marks the start of indexing.
func (t *StatusTracker) StartIndexing() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.state = StateIndexing
	t.startedAt = &now
	t.completedAt = nil
	t.errorMsg = ""
	t.progress = IndexingProgress{
		Phase:           PhaseParsing,
		PhaseProgress:   0,
		OverallProgress: 0,
	}

	t.notifySubscribers()
}

// SetPhase updates the current indexing phase.
func (t *StatusTracker) SetPhase(phase IndexingPhase, message string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.Phase = phase
	t.progress.PhaseProgress = 0
	t.progress.Message = message

	// Calculate overall progress based on phase.
	switch phase {
	case PhaseParsing:
		t.progress.OverallProgress = 0.0
	case PhaseModuleRegistry:
		t.progress.OverallProgress = 0.33
	case PhaseCallGraph:
		t.progress.OverallProgress = 0.66
	case PhaseComplete:
		t.progress.OverallProgress = 1.0
	}

	t.notifySubscribers()
}

// UpdateProgress updates progress within the current phase.
func (t *StatusTracker) UpdateProgress(processed, total int, currentFile string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.progress.FilesProcessed = processed
	t.progress.TotalFiles = total
	t.progress.CurrentFile = currentFile

	if total > 0 {
		t.progress.PhaseProgress = float64(processed) / float64(total)
	}

	// Update overall progress based on phase + phase progress.
	baseProgress := 0.0
	phaseWeight := 0.33

	switch t.progress.Phase {
	case PhaseParsing:
		baseProgress = 0.0
	case PhaseModuleRegistry:
		baseProgress = 0.33
	case PhaseCallGraph:
		baseProgress = 0.66
	}

	t.progress.OverallProgress = baseProgress + (t.progress.PhaseProgress * phaseWeight)

	t.notifySubscribers()
}

// CompleteIndexing marks indexing as complete.
func (t *StatusTracker) CompleteIndexing(stats *IndexingStats) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.state = StateReady
	t.completedAt = &now
	t.stats = stats
	t.progress = IndexingProgress{
		Phase:           PhaseComplete,
		PhaseProgress:   1.0,
		OverallProgress: 1.0,
		Message:         "Indexing complete",
	}

	t.notifySubscribers()
}

// FailIndexing marks indexing as failed.
func (t *StatusTracker) FailIndexing(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.state = StateFailed
	t.completedAt = &now
	if err != nil {
		t.errorMsg = err.Error()
	}
	t.progress.Message = "Indexing failed"

	t.notifySubscribers()
}

// Subscribe returns a channel that receives status updates.
func (t *StatusTracker) Subscribe() chan IndexingStatus {
	t.mu.Lock()
	defer t.mu.Unlock()

	ch := make(chan IndexingStatus, 10)
	t.subscribers = append(t.subscribers, ch)

	// Send current status immediately.
	select {
	case ch <- t.buildStatus():
	default:
	}

	return ch
}

// Unsubscribe removes a subscription channel.
func (t *StatusTracker) Unsubscribe(ch chan IndexingStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, sub := range t.subscribers {
		if sub == ch {
			t.subscribers = append(t.subscribers[:i], t.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// notifySubscribers sends status to all subscribers (must be called with lock held).
func (t *StatusTracker) notifySubscribers() {
	status := t.buildStatus()
	for _, ch := range t.subscribers {
		select {
		case ch <- status:
		default:
			// Channel full, skip.
		}
	}
}

// buildStatus creates status struct (must be called with lock held).
func (t *StatusTracker) buildStatus() IndexingStatus {
	return IndexingStatus{
		State:       t.state,
		Progress:    t.progress,
		StartedAt:   t.startedAt,
		CompletedAt: t.completedAt,
		Error:       t.errorMsg,
		Stats:       t.stats,
	}
}

// GracefulDegradation provides methods for handling requests during indexing.
type GracefulDegradation struct {
	tracker *StatusTracker
}

// NewGracefulDegradation creates a new graceful degradation handler.
func NewGracefulDegradation(tracker *StatusTracker) *GracefulDegradation {
	return &GracefulDegradation{tracker: tracker}
}

// CheckReady checks if the server is ready and returns an appropriate error if not.
func (g *GracefulDegradation) CheckReady() *RPCError {
	status := g.tracker.GetStatus()

	switch status.State {
	case StateReady:
		return nil
	case StateIndexing:
		return IndexNotReadyError(
			status.Progress.Phase.String(),
			status.Progress.OverallProgress,
		)
	case StateFailed:
		return InternalError("Indexing failed: " + status.Error)
	default:
		return IndexNotReadyError("uninitialized", 0)
	}
}

// WrapToolCall wraps a tool call with readiness checking.
func (g *GracefulDegradation) WrapToolCall(toolName string, fn func() (string, bool)) (string, bool) {
	if err := g.CheckReady(); err != nil {
		return NewToolError(err.Message, err.Code, map[string]any{
			"tool":   toolName,
			"status": g.tracker.GetStatus(),
		}), true
	}
	return fn()
}

// GetStatusJSON returns the current status as a JSON-friendly map.
func (g *GracefulDegradation) GetStatusJSON() map[string]any {
	status := g.tracker.GetStatus()

	result := map[string]any{
		"state": status.State.String(),
		"progress": map[string]any{
			"phase":           status.Progress.Phase.String(),
			"phaseProgress":   status.Progress.PhaseProgress,
			"overallProgress": status.Progress.OverallProgress,
			"filesProcessed":  status.Progress.FilesProcessed,
			"totalFiles":      status.Progress.TotalFiles,
		},
	}

	if status.Progress.CurrentFile != "" {
		result["progress"].(map[string]any)["currentFile"] = status.Progress.CurrentFile
	}

	if status.Progress.Message != "" {
		result["progress"].(map[string]any)["message"] = status.Progress.Message
	}

	if status.StartedAt != nil {
		result["startedAt"] = status.StartedAt.Format(time.RFC3339)
	}

	if status.CompletedAt != nil {
		result["completedAt"] = status.CompletedAt.Format(time.RFC3339)
	}

	if status.Error != "" {
		result["error"] = status.Error
	}

	if status.Stats != nil {
		result["stats"] = map[string]any{
			"functions":     status.Stats.Functions,
			"callEdges":     status.Stats.CallEdges,
			"modules":       status.Stats.Modules,
			"files":         status.Stats.Files,
			"buildDuration": status.Stats.BuildDuration.String(),
		}
	}

	return result
}
