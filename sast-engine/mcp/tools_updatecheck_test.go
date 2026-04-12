package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/shivasurya/code-pathfinder/sast-engine/updatecheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- handleToolsList update-check tests -------------------------------------

func TestHandleToolsList_StatusDescriptionUnchangedWithoutUpdateInfo(t *testing.T) {
	server := createTestServer()
	server.updateInfo = nil

	req := makeJSONRPCRequest("tools/list", nil)
	resp := server.handleToolsList(req)

	tools := extractToolsListResult(t, resp)
	statusDesc := findToolDescription(t, tools, "status")

	// When updateInfo is nil the description must not start with an upgrade hint.
	assert.False(t, strings.HasPrefix(statusDesc, "⚠"), "no upgrade prefix expected without updateInfo")
	assert.False(t, strings.HasPrefix(statusDesc, "ℹ"), "no announcement prefix expected without updateInfo")
}

func TestHandleToolsList_StatusDescriptionInjectedWithUpgrade(t *testing.T) {
	server := createTestServer()
	server.updateInfo = &updatecheck.Result{
		Upgrade: &updatecheck.UpgradeNotice{
			Current:    "1.0.0",
			Latest:     "2.0.0",
			ReleaseURL: "https://example.com/releases/v2.0.0",
		},
	}

	req := makeJSONRPCRequest("tools/list", nil)
	resp := server.handleToolsList(req)

	tools := extractToolsListResult(t, resp)
	statusDesc := findToolDescription(t, tools, "status")

	assert.Contains(t, statusDesc, "Upgrade available")
	assert.Contains(t, statusDesc, "2.0.0")
	assert.Contains(t, statusDesc, "https://example.com/releases/v2.0.0")
}

func TestHandleToolsList_StatusDescriptionInjectedWithAnnouncement(t *testing.T) {
	server := createTestServer()
	server.updateInfo = &updatecheck.Result{
		Announcement: &updatecheck.Announcement{
			Title: "Workshop",
			Text:  "Join us for a live walkthrough.",
			URL:   "https://example.com/workshop",
		},
	}

	req := makeJSONRPCRequest("tools/list", nil)
	resp := server.handleToolsList(req)

	tools := extractToolsListResult(t, resp)
	statusDesc := findToolDescription(t, tools, "status")

	assert.Contains(t, statusDesc, "Workshop")
	assert.Contains(t, statusDesc, "Join us for a live walkthrough.")
}

func TestHandleToolsList_DescriptionTruncatedAt200(t *testing.T) {
	server := createTestServer()
	longURL := "https://example.com/releases/v2.0.0/" + strings.Repeat("x", 200)
	server.updateInfo = &updatecheck.Result{
		Upgrade: &updatecheck.UpgradeNotice{
			Current:    "1.0.0",
			Latest:     "2.0.0",
			ReleaseURL: longURL,
		},
	}

	req := makeJSONRPCRequest("tools/list", nil)
	resp := server.handleToolsList(req)

	tools := extractToolsListResult(t, resp)
	statusDesc := findToolDescription(t, tools, "status")

	// Extract the first line (before \n\n).
	parts := strings.SplitN(statusDesc, "\n\n", 2)
	require.Len(t, parts, 2, "expected separator between prefix and base")
	assert.LessOrEqual(t, len(parts[0]), 200, "prefix must be ≤200 bytes")
}

func TestHandleToolsList_OtherToolDescriptionsUntouched(t *testing.T) {
	server := createTestServer()

	// Capture descriptions without upgrade info.
	server.updateInfo = nil
	req := makeJSONRPCRequest("tools/list", nil)
	toolsWithout := extractToolsListResult(t, server.handleToolsList(req))

	// Now set an upgrade and re-fetch.
	server.updateInfo = &updatecheck.Result{
		Upgrade: &updatecheck.UpgradeNotice{
			Current: "1.0.0", Latest: "2.0.0",
			ReleaseURL: "https://example.com/releases/v2.0.0",
		},
	}
	toolsWith := extractToolsListResult(t, server.handleToolsList(req))

	require.Len(t, toolsWith, len(toolsWithout))

	for i, with := range toolsWith {
		without := toolsWithout[i]
		if with.Name == "status" {
			// Only the status tool should differ.
			assert.NotEqual(t, with.Description, without.Description, "status description should be mutated")
		} else {
			assert.Equal(t, with.Description, without.Description,
				"description for %q must not change", with.Name)
		}
	}
}

// --- status tool structured fields test -------------------------------------

func TestStatusTool_IncludesStructuredUpgradeFields(t *testing.T) {
	server := createTestServer()
	server.statusTracker.StartIndexing()
	server.statusTracker.CompleteIndexing(&IndexingStats{Functions: 10})

	server.updateInfo = &updatecheck.Result{
		Upgrade: &updatecheck.UpgradeNotice{
			Current:    "1.0.0",
			Latest:     "2.0.0",
			ReleaseURL: "https://example.com/releases/v2.0.0",
			Message:    "big release",
		},
		Announcement: &updatecheck.Announcement{
			ID:    "ann-1",
			Level: "info",
			Title: "Workshop",
			Text:  "Join us.",
			URL:   "https://example.com/workshop",
		},
	}

	req := makeJSONRPCRequest("status", nil)
	resp := server.handleStatus(req)

	require.NotNil(t, resp)
	b, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, json.Unmarshal(b, &result))

	assert.Equal(t, "2.0.0", result["latest_version"])
	assert.Equal(t, "big release", result["update_message"])
	assert.Equal(t, "https://example.com/releases/v2.0.0", result["release_url"])

	ann, ok := result["announcement"].(map[string]any)
	require.True(t, ok, "announcement should be a map")
	assert.Equal(t, "ann-1", ann["id"])
	assert.Equal(t, "info", ann["level"])
	assert.Equal(t, "Workshop", ann["title"])
}

func TestStatusTool_NoUpdateFieldsWhenUpdateInfoNil(t *testing.T) {
	server := createTestServer()
	server.statusTracker.StartIndexing()
	server.statusTracker.CompleteIndexing(&IndexingStats{})
	server.updateInfo = nil

	req := makeJSONRPCRequest("status", nil)
	resp := server.handleStatus(req)

	b, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, json.Unmarshal(b, &result))

	assert.NotContains(t, result, "latest_version")
	assert.NotContains(t, result, "update_message")
	assert.NotContains(t, result, "announcement")
}

// --- helpers ----------------------------------------------------------------

func extractToolsListResult(t *testing.T, resp *JSONRPCResponse) []Tool {
	t.Helper()
	b, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var r ToolsListResult
	require.NoError(t, json.Unmarshal(b, &r))
	return r.Tools
}

func findToolDescription(t *testing.T, tools []Tool, name string) string { //nolint:unparam
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool.Description
		}
	}
	t.Fatalf("tool %q not found in tools list", name)
	return ""
}
