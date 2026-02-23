package mcp

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexingState_String(t *testing.T) {
	tests := []struct {
		state    IndexingState
		expected string
	}{
		{StateUninitialized, "uninitialized"},
		{StateIndexing, "indexing"},
		{StateReady, "ready"},
		{StateFailed, "failed"},
		{IndexingState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestIndexingPhase_String(t *testing.T) {
	tests := []struct {
		phase    IndexingPhase
		expected string
	}{
		{PhaseNone, "none"},
		{PhaseParsing, "parsing"},
		{PhaseModuleRegistry, "module_registry"},
		{PhaseCallGraph, "call_graph"},
		{PhaseComplete, "complete"},
		{IndexingPhase(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.phase.String())
		})
	}
}

func TestNewStatusTracker(t *testing.T) {
	tracker := NewStatusTracker()

	assert.NotNil(t, tracker)
	assert.Equal(t, StateUninitialized, tracker.GetState())
	assert.False(t, tracker.IsReady())
}

func TestStatusTracker_StartIndexing(t *testing.T) {
	tracker := NewStatusTracker()

	tracker.StartIndexing()

	status := tracker.GetStatus()
	assert.Equal(t, StateIndexing, status.State)
	assert.NotNil(t, status.StartedAt)
	assert.Nil(t, status.CompletedAt)
	assert.Equal(t, PhaseParsing, status.Progress.Phase)
	assert.Equal(t, 0.0, status.Progress.OverallProgress)
}

func TestStatusTracker_SetPhase(t *testing.T) {
	tracker := NewStatusTracker()
	tracker.StartIndexing()

	tests := []struct {
		phase           IndexingPhase
		expectedOverall float64
	}{
		{PhaseParsing, 0.0},
		{PhaseModuleRegistry, 0.33},
		{PhaseCallGraph, 0.66},
		{PhaseComplete, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.phase.String(), func(t *testing.T) {
			tracker.SetPhase(tt.phase, "test message")

			status := tracker.GetStatus()
			assert.Equal(t, tt.phase, status.Progress.Phase)
			assert.Equal(t, tt.expectedOverall, status.Progress.OverallProgress)
			assert.Equal(t, "test message", status.Progress.Message)
			assert.Equal(t, 0.0, status.Progress.PhaseProgress)
		})
	}
}

func TestStatusTracker_UpdateProgress(t *testing.T) {
	tracker := NewStatusTracker()
	tracker.StartIndexing()
	tracker.SetPhase(PhaseParsing, "Parsing files")

	tracker.UpdateProgress(50, 100, "file.py")

	status := tracker.GetStatus()
	assert.Equal(t, 50, status.Progress.FilesProcessed)
	assert.Equal(t, 100, status.Progress.TotalFiles)
	assert.Equal(t, "file.py", status.Progress.CurrentFile)
	assert.Equal(t, 0.5, status.Progress.PhaseProgress)
	// Overall progress: 0.0 (base) + 0.5 * 0.33 (phase weight) = 0.165
	assert.InDelta(t, 0.165, status.Progress.OverallProgress, 0.01)
}

func TestStatusTracker_UpdateProgress_ZeroTotal(t *testing.T) {
	tracker := NewStatusTracker()
	tracker.StartIndexing()

	tracker.UpdateProgress(0, 0, "")

	status := tracker.GetStatus()
	assert.Equal(t, 0.0, status.Progress.PhaseProgress)
}

func TestStatusTracker_CompleteIndexing(t *testing.T) {
	tracker := NewStatusTracker()
	tracker.StartIndexing()

	stats := &IndexingStats{
		Functions:     100,
		CallEdges:     500,
		Modules:       20,
		Files:         50,
		BuildDuration: 5 * time.Second,
	}

	tracker.CompleteIndexing(stats)

	status := tracker.GetStatus()
	assert.Equal(t, StateReady, status.State)
	assert.True(t, tracker.IsReady())
	assert.NotNil(t, status.CompletedAt)
	assert.Equal(t, PhaseComplete, status.Progress.Phase)
	assert.Equal(t, 1.0, status.Progress.OverallProgress)
	assert.Equal(t, stats, status.Stats)
}

func TestStatusTracker_FailIndexing(t *testing.T) {
	tracker := NewStatusTracker()
	tracker.StartIndexing()

	err := errors.New("something went wrong")
	tracker.FailIndexing(err)

	status := tracker.GetStatus()
	assert.Equal(t, StateFailed, status.State)
	assert.False(t, tracker.IsReady())
	assert.NotNil(t, status.CompletedAt)
	assert.Equal(t, "something went wrong", status.Error)
	assert.Equal(t, "Indexing failed", status.Progress.Message)
}

func TestStatusTracker_FailIndexing_NilError(t *testing.T) {
	tracker := NewStatusTracker()
	tracker.StartIndexing()

	tracker.FailIndexing(nil)

	status := tracker.GetStatus()
	assert.Equal(t, StateFailed, status.State)
	assert.Empty(t, status.Error)
}

func TestStatusTracker_Subscribe(t *testing.T) {
	tracker := NewStatusTracker()

	ch := tracker.Subscribe()
	require.NotNil(t, ch)

	// Should receive initial status.
	select {
	case status := <-ch:
		assert.Equal(t, StateUninitialized, status.State)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected initial status")
	}

	// Start indexing and receive update.
	tracker.StartIndexing()

	select {
	case status := <-ch:
		assert.Equal(t, StateIndexing, status.State)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected status update")
	}

	tracker.Unsubscribe(ch)
}

func TestStatusTracker_Unsubscribe(t *testing.T) {
	tracker := NewStatusTracker()

	ch := tracker.Subscribe()
	// Drain initial status.
	<-ch

	tracker.Unsubscribe(ch)

	// Channel should be closed.
	_, ok := <-ch
	assert.False(t, ok)
}

func TestStatusTracker_MultipleSubscribers(t *testing.T) {
	tracker := NewStatusTracker()

	ch1 := tracker.Subscribe()
	ch2 := tracker.Subscribe()

	// Drain initial statuses.
	<-ch1
	<-ch2

	tracker.StartIndexing()

	// Both should receive update.
	select {
	case s := <-ch1:
		assert.Equal(t, StateIndexing, s.State)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ch1 should receive update")
	}

	select {
	case s := <-ch2:
		assert.Equal(t, StateIndexing, s.State)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ch2 should receive update")
	}

	tracker.Unsubscribe(ch1)
	tracker.Unsubscribe(ch2)
}

func TestNewGracefulDegradation(t *testing.T) {
	tracker := NewStatusTracker()
	gd := NewGracefulDegradation(tracker)

	assert.NotNil(t, gd)
}

func TestGracefulDegradation_CheckReady_Ready(t *testing.T) {
	tracker := NewStatusTracker()
	tracker.StartIndexing()
	tracker.CompleteIndexing(&IndexingStats{})
	gd := NewGracefulDegradation(tracker)

	err := gd.CheckReady()
	assert.Nil(t, err)
}

func TestGracefulDegradation_CheckReady_Indexing(t *testing.T) {
	tracker := NewStatusTracker()
	tracker.StartIndexing()
	tracker.SetPhase(PhaseCallGraph, "Building call graph")
	tracker.UpdateProgress(50, 100, "")
	gd := NewGracefulDegradation(tracker)

	err := gd.CheckReady()
	require.NotNil(t, err)
	assert.Equal(t, ErrCodeIndexNotReady, err.Code)
	assert.Contains(t, err.Message, "call_graph")
}

func TestGracefulDegradation_CheckReady_Failed(t *testing.T) {
	tracker := NewStatusTracker()
	tracker.StartIndexing()
	tracker.FailIndexing(errors.New("oops"))
	gd := NewGracefulDegradation(tracker)

	err := gd.CheckReady()
	require.NotNil(t, err)
	assert.Equal(t, ErrCodeInternalError, err.Code)
	assert.Contains(t, err.Message, "oops")
}

func TestGracefulDegradation_CheckReady_Uninitialized(t *testing.T) {
	tracker := NewStatusTracker()
	gd := NewGracefulDegradation(tracker)

	err := gd.CheckReady()
	require.NotNil(t, err)
	assert.Equal(t, ErrCodeIndexNotReady, err.Code)
}

func TestGracefulDegradation_WrapToolCall_Ready(t *testing.T) {
	tracker := NewStatusTracker()
	tracker.StartIndexing()
	tracker.CompleteIndexing(&IndexingStats{})
	gd := NewGracefulDegradation(tracker)

	called := false
	result, isError := gd.WrapToolCall("test_tool", func() (string, bool) {
		called = true
		return `{"result": "ok"}`, false
	})

	assert.True(t, called)
	assert.False(t, isError)
	assert.Contains(t, result, "ok")
}

func TestGracefulDegradation_WrapToolCall_NotReady(t *testing.T) {
	tracker := NewStatusTracker()
	tracker.StartIndexing()
	gd := NewGracefulDegradation(tracker)

	called := false
	result, isError := gd.WrapToolCall("test_tool", func() (string, bool) {
		called = true
		return `{"result": "ok"}`, false
	})

	assert.False(t, called)
	assert.True(t, isError)
	assert.Contains(t, result, "test_tool")
	assert.Contains(t, result, "status")
}

func TestGracefulDegradation_GetStatusJSON(t *testing.T) {
	tracker := NewStatusTracker()
	tracker.StartIndexing()
	tracker.SetPhase(PhaseParsing, "Parsing source files")
	tracker.UpdateProgress(10, 50, "main.py")
	gd := NewGracefulDegradation(tracker)

	status := gd.GetStatusJSON()

	assert.Equal(t, "indexing", status["state"])
	assert.NotNil(t, status["progress"])
	assert.NotNil(t, status["startedAt"])

	progress := status["progress"].(map[string]any)
	assert.Equal(t, "parsing", progress["phase"])
	assert.Equal(t, 10, progress["filesProcessed"])
	assert.Equal(t, 50, progress["totalFiles"])
	assert.Equal(t, "main.py", progress["currentFile"])
	assert.Equal(t, "Parsing source files", progress["message"])
}

func TestGracefulDegradation_GetStatusJSON_Complete(t *testing.T) {
	tracker := NewStatusTracker()
	tracker.StartIndexing()
	tracker.CompleteIndexing(&IndexingStats{
		Functions:     100,
		CallEdges:     500,
		Modules:       20,
		Files:         50,
		BuildDuration: 5 * time.Second,
	})
	gd := NewGracefulDegradation(tracker)

	status := gd.GetStatusJSON()

	assert.Equal(t, "ready", status["state"])
	assert.NotNil(t, status["completedAt"])
	assert.NotNil(t, status["stats"])

	stats := status["stats"].(map[string]any)
	assert.Equal(t, 100, stats["functions"])
	assert.Equal(t, 500, stats["callEdges"])
}

func TestGracefulDegradation_GetStatusJSON_Failed(t *testing.T) {
	tracker := NewStatusTracker()
	tracker.StartIndexing()
	tracker.FailIndexing(errors.New("parse error"))
	gd := NewGracefulDegradation(tracker)

	status := gd.GetStatusJSON()

	assert.Equal(t, "failed", status["state"])
	assert.Equal(t, "parse error", status["error"])
}

func TestStatusTracker_FullWorkflow(t *testing.T) {
	tracker := NewStatusTracker()

	// Initial state.
	assert.Equal(t, StateUninitialized, tracker.GetState())

	// Start indexing.
	tracker.StartIndexing()
	assert.Equal(t, StateIndexing, tracker.GetState())

	// Phase 1: Parsing.
	tracker.SetPhase(PhaseParsing, "Parsing Python files")
	for i := 0; i <= 100; i += 10 {
		tracker.UpdateProgress(i, 100, "file.py")
	}

	// Phase 2: Module registry.
	tracker.SetPhase(PhaseModuleRegistry, "Building module registry")
	for i := 0; i <= 100; i += 20 {
		tracker.UpdateProgress(i, 100, "")
	}

	// Phase 3: Call graph.
	tracker.SetPhase(PhaseCallGraph, "Building call graph")
	for i := 0; i <= 100; i += 25 {
		tracker.UpdateProgress(i, 100, "")
	}

	// Complete.
	tracker.CompleteIndexing(&IndexingStats{
		Functions: 50,
		CallEdges: 100,
	})

	assert.Equal(t, StateReady, tracker.GetState())
	assert.True(t, tracker.IsReady())
}

func TestStatusTracker_ConcurrentAccess(t *testing.T) {
	tracker := NewStatusTracker()

	done := make(chan bool)

	// Reader goroutine.
	go func() {
		for range 100 {
			_ = tracker.GetStatus()
			_ = tracker.GetState()
			_ = tracker.IsReady()
		}
		done <- true
	}()

	// Writer goroutine.
	go func() {
		tracker.StartIndexing()
		for i := range 50 {
			tracker.UpdateProgress(i, 50, "file.py")
		}
		tracker.CompleteIndexing(&IndexingStats{})
		done <- true
	}()

	<-done
	<-done
}
