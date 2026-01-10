package mcp

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAnalytics(t *testing.T) {
	a := NewAnalytics("stdio")

	assert.NotNil(t, a)
	assert.Equal(t, "stdio", a.transport)
	assert.False(t, a.startTime.IsZero())
}

func TestNewAnalytics_HTTP(t *testing.T) {
	a := NewAnalytics("http")

	assert.Equal(t, "http", a.transport)
}

func TestAnalytics_ReportServerStarted(t *testing.T) {
	a := NewAnalytics("stdio")

	// Should not panic even with analytics disabled.
	a.ReportServerStarted()
}

func TestAnalytics_ReportServerStopped(t *testing.T) {
	a := NewAnalytics("http")

	// Should not panic.
	a.ReportServerStopped()
}

func TestAnalytics_ReportToolCall(t *testing.T) {
	a := NewAnalytics("stdio")

	// Should not panic.
	a.ReportToolCall("find_symbol", 150, true)
	a.ReportToolCall("get_callers", 50, false)
}

func TestAnalytics_ReportIndexingStarted(t *testing.T) {
	a := NewAnalytics("stdio")

	// Should not panic.
	a.ReportIndexingStarted()
}

func TestAnalytics_ReportIndexingComplete(t *testing.T) {
	a := NewAnalytics("http")

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
	a := NewAnalytics("stdio")

	// Should not panic.
	a.ReportIndexingFailed("parsing")
	a.ReportIndexingFailed("call_graph")
}

func TestAnalytics_ReportClientConnected(t *testing.T) {
	a := NewAnalytics("http")

	// Should not panic.
	a.ReportClientConnected("claude-code", "1.0.0")
	a.ReportClientConnected("", "")
}

func TestAnalytics_StartToolCall(t *testing.T) {
	a := NewAnalytics("stdio")

	metrics := a.StartToolCall("find_symbol")

	assert.NotNil(t, metrics)
	assert.Equal(t, "find_symbol", metrics.ToolName)
	assert.False(t, metrics.StartTime.IsZero())
}

func TestAnalytics_EndToolCall(t *testing.T) {
	a := NewAnalytics("stdio")

	metrics := a.StartToolCall("get_callers")
	time.Sleep(1 * time.Millisecond) // Small delay to ensure duration > 0

	// Should not panic.
	a.EndToolCall(metrics, true)
}

func TestAnalytics_EndToolCall_Nil(t *testing.T) {
	a := NewAnalytics("stdio")

	// Should not panic with nil metrics.
	a.EndToolCall(nil, true)
}

func TestAnalytics_EndToolCall_Failed(t *testing.T) {
	a := NewAnalytics("http")

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
