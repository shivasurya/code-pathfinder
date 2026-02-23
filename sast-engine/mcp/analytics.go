package mcp

import (
	"runtime"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/analytics"
)

// Analytics provides MCP-specific telemetry helpers.
// All events are anonymous and contain no PII.
type Analytics struct {
	transport string // "stdio" or "http"
	startTime time.Time
	disabled  bool
}

// NewAnalytics creates a new analytics instance.
func NewAnalytics(transport string, disabled bool) *Analytics {
	return &Analytics{
		transport: transport,
		startTime: time.Now(),
		disabled:  disabled,
	}
}

// ReportServerStarted reports that the MCP server has started.
func (a *Analytics) ReportServerStarted() {
	if a.disabled {
		return
	}
	analytics.ReportEventWithProperties(analytics.MCPServerStarted, map[string]any{
		"transport": a.transport,
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
	})
}

// ReportServerStopped reports that the MCP server has stopped.
func (a *Analytics) ReportServerStopped() {
	if a.disabled {
		return
	}
	analytics.ReportEventWithProperties(analytics.MCPServerStopped, map[string]any{
		"transport":      a.transport,
		"uptime_seconds": time.Since(a.startTime).Seconds(),
	})
}

// ReportToolCall reports a tool invocation with timing and success info.
// No file paths or code content is included.
func (a *Analytics) ReportToolCall(toolName string, durationMs int64, success bool) {
	if a.disabled {
		return
	}
	analytics.ReportEventWithProperties(analytics.MCPToolCall, map[string]any{
		"tool":        toolName,
		"duration_ms": durationMs,
		"success":     success,
		"transport":   a.transport,
	})
}

// ReportIndexingStarted reports that indexing has begun.
func (a *Analytics) ReportIndexingStarted() {
	if a.disabled {
		return
	}
	analytics.ReportEventWithProperties(analytics.MCPIndexingStarted, map[string]any{
		"transport": a.transport,
	})
}

// ReportIndexingComplete reports successful indexing completion.
// Only aggregate counts are reported, no file paths.
func (a *Analytics) ReportIndexingComplete(stats *IndexingStats) {
	if a.disabled {
		return
	}
	props := map[string]any{
		"transport":        a.transport,
		"duration_seconds": stats.BuildDuration.Seconds(),
		"function_count":   stats.Functions,
		"call_edge_count":  stats.CallEdges,
		"module_count":     stats.Modules,
		"file_count":       stats.Files,
	}
	analytics.ReportEventWithProperties(analytics.MCPIndexingComplete, props)
}

// ReportIndexingFailed reports indexing failure.
// Error messages are not included to avoid potential PII.
func (a *Analytics) ReportIndexingFailed(phase string) {
	if a.disabled {
		return
	}
	analytics.ReportEventWithProperties(analytics.MCPIndexingFailed, map[string]any{
		"transport": a.transport,
		"phase":     phase,
	})
}

// ReportClientConnected reports a client connection with client info.
// Only client name/version (from MCP protocol) is reported.
func (a *Analytics) ReportClientConnected(clientName, clientVersion string) {
	if a.disabled {
		return
	}
	analytics.ReportEventWithProperties(analytics.MCPClientConnected, map[string]any{
		"transport":      a.transport,
		"client_name":    clientName,
		"client_version": clientVersion,
	})
}

// ToolCallMetrics holds metrics for a tool call.
type ToolCallMetrics struct {
	StartTime time.Time
	ToolName  string
}

// StartToolCall begins tracking a tool call.
func (a *Analytics) StartToolCall(toolName string) *ToolCallMetrics {
	return &ToolCallMetrics{
		StartTime: time.Now(),
		ToolName:  toolName,
	}
}

// EndToolCall completes tracking and reports the metric.
func (a *Analytics) EndToolCall(m *ToolCallMetrics, success bool) {
	if m == nil {
		return
	}
	durationMs := time.Since(m.StartTime).Milliseconds()
	a.ReportToolCall(m.ToolName, durationMs, success)
}
