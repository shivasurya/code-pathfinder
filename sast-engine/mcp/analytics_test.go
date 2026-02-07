package mcp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAnalytics(t *testing.T) {
	a := NewAnalytics("stdio", false)

	assert.NotNil(t, a)
	assert.Equal(t, "stdio", a.transport)
	assert.False(t, a.startTime.IsZero())
}

func TestNewAnalytics_HTTP(t *testing.T) {
	a := NewAnalytics("http", false)

	assert.Equal(t, "http", a.transport)
}

func TestAnalytics_ReportServerStarted(t *testing.T) {
	a := NewAnalytics("stdio", false)

	// Should not panic even with analytics disabled.
	a.ReportServerStarted()
}

func TestAnalytics_ReportServerStopped(t *testing.T) {
	a := NewAnalytics("http", false)

	// Should not panic.
	a.ReportServerStopped()
}

func TestAnalytics_ReportToolCall(t *testing.T) {
	a := NewAnalytics("stdio", false)

	// Should not panic.
	a.ReportToolCall("find_symbol", 150, true)
	a.ReportToolCall("get_callers", 50, false)
}

func TestAnalytics_ReportIndexingStarted(t *testing.T) {
	a := NewAnalytics("stdio", false)

	// Should not panic.
	a.ReportIndexingStarted()
}

func TestAnalytics_ReportIndexingComplete(t *testing.T) {
	a := NewAnalytics("http", false)

	stats := &IndexingStats{
		Functions:     100,
		CallEdges:     500,
		Modules:       20,
		Files:         50,
		BuildDuration: 5 * time.Second,
	}

	// Should not panic.
	a.ReportIndexingComplete(stats)
}

func TestAnalytics_ReportIndexingFailed(t *testing.T) {
	a := NewAnalytics("stdio", false)

	// Should not panic.
	a.ReportIndexingFailed("parsing")
	a.ReportIndexingFailed("call_graph")
}

func TestAnalytics_ReportClientConnected(t *testing.T) {
	a := NewAnalytics("http", false)

	// Should not panic.
	a.ReportClientConnected("claude-code", "1.0.0")
	a.ReportClientConnected("", "")
}

func TestAnalytics_StartToolCall(t *testing.T) {
	a := NewAnalytics("stdio", false)

	metrics := a.StartToolCall("find_symbol")

	assert.NotNil(t, metrics)
	assert.Equal(t, "find_symbol", metrics.ToolName)
	assert.False(t, metrics.StartTime.IsZero())
}

func TestAnalytics_EndToolCall(t *testing.T) {
	a := NewAnalytics("stdio", false)

	metrics := a.StartToolCall("get_callers")
	time.Sleep(1 * time.Millisecond) // Small delay to ensure duration > 0

	// Should not panic.
	a.EndToolCall(metrics, true)
}

func TestAnalytics_EndToolCall_Nil(t *testing.T) {
	a := NewAnalytics("stdio", false)

	// Should not panic with nil metrics.
	a.EndToolCall(nil, true)
}

func TestAnalytics_EndToolCall_Failed(t *testing.T) {
	a := NewAnalytics("http", false)

	metrics := a.StartToolCall("resolve_import")

	// Should not panic.
	a.EndToolCall(metrics, false)
}

func TestToolCallMetrics(t *testing.T) {
	metrics := &ToolCallMetrics{
		StartTime: time.Now(),
		ToolName:  "test_tool",
	}

	assert.Equal(t, "test_tool", metrics.ToolName)
	assert.False(t, metrics.StartTime.IsZero())
}

func TestNewAnalytics_Disabled(t *testing.T) {
	a := NewAnalytics("stdio", true)

	assert.NotNil(t, a)
	assert.Equal(t, "stdio", a.transport)
	assert.True(t, a.disabled)
	assert.False(t, a.startTime.IsZero())
}

func TestNewAnalytics_Enabled(t *testing.T) {
	a := NewAnalytics("http", false)

	assert.NotNil(t, a)
	assert.Equal(t, "http", a.transport)
	assert.False(t, a.disabled)
}

func TestAnalytics_Disabled_ReportServerStarted(t *testing.T) {
	a := NewAnalytics("stdio", true)

	// Should not panic and should return early when disabled.
	a.ReportServerStarted()
}

func TestAnalytics_Disabled_ReportServerStopped(t *testing.T) {
	a := NewAnalytics("http", true)

	// Should not panic and should return early when disabled.
	a.ReportServerStopped()
}

func TestAnalytics_Disabled_ReportToolCall(t *testing.T) {
	a := NewAnalytics("stdio", true)

	// Should not panic and should return early when disabled.
	a.ReportToolCall("find_symbol", 150, true)
}

func TestAnalytics_Disabled_ReportIndexingStarted(t *testing.T) {
	a := NewAnalytics("stdio", true)

	// Should not panic and should return early when disabled.
	a.ReportIndexingStarted()
}

func TestAnalytics_Disabled_ReportIndexingComplete(t *testing.T) {
	a := NewAnalytics("http", true)

	stats := &IndexingStats{
		Functions:     100,
		CallEdges:     500,
		Modules:       20,
		Files:         50,
		BuildDuration: 5 * time.Second,
	}

	// Should not panic and should return early when disabled.
	a.ReportIndexingComplete(stats)
}

func TestAnalytics_Disabled_ReportIndexingFailed(t *testing.T) {
	a := NewAnalytics("stdio", true)

	// Should not panic and should return early when disabled.
	a.ReportIndexingFailed("parsing")
}

func TestAnalytics_Disabled_ReportClientConnected(t *testing.T) {
	a := NewAnalytics("http", true)

	// Should not panic and should return early when disabled.
	a.ReportClientConnected("claude-code", "1.0.0")
}

func TestAnalytics_Disabled_EndToolCall(t *testing.T) {
	a := NewAnalytics("stdio", true)

	metrics := a.StartToolCall("get_callers")

	// EndToolCall still works (calls ReportToolCall which returns early).
	a.EndToolCall(metrics, true)
}
