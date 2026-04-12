package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/shivasurya/code-pathfinder/sast-engine/graph"
	"github.com/shivasurya/code-pathfinder/sast-engine/graph/callgraph/core"
	"github.com/shivasurya/code-pathfinder/sast-engine/updatecheck"
)

// mcpManifestURL overrides the CDN URL used by fetchUpdateInfo. Empty string
// means use the updatecheck package default. Overridable in tests.
var mcpManifestURL string

// Server handles MCP protocol communication.
type Server struct {
	projectPath      string
	pythonVersion    string
	callGraph        *core.CallGraph
	moduleRegistry   *core.ModuleRegistry
	codeGraph        *graph.CodeGraph
	indexedAt        time.Time
	buildTime        time.Duration
	statusTracker    *StatusTracker
	degradation      *GracefulDegradation
	analytics        *Analytics
	disableAnalytics bool
	version          string

	// Go-specific context for stdlib metadata in MCP tool responses.
	// Set via SetGoContext after the Go call graph is built.
	goVersion        string
	goModuleRegistry *core.GoModuleRegistry

	// updateInfo is populated once at server construction via a synchronous
	// updatecheck.Check call (5 s timeout). It is immutable for the lifetime
	// of the process — no goroutine, no locking, no Close() needed.
	updateInfo *updatecheck.Result

	// reachReporter deduplicates analytics reach events within a 24-hour
	// window. Initialized in both constructors alongside updateInfo.
	reachReporter *updatecheck.ReachReporter
}

// SetVersion sets the server version reported in MCP initialize responses.
// Should be called with cmd.Version (injected via ldflags at build time).
func (s *Server) SetVersion(version string) {
	s.version = version
}

// SetGoContext stores the Go version and module registry so that MCP tool
// responses can include stdlib metadata (is_stdlib, signature, return_type,
// etc.) for Go standard library calls.
//
// Must be called after InitGoStdlibLoader has populated reg.StdlibLoader.
// Safe to skip — tools degrade gracefully when goModuleRegistry is nil.
func (s *Server) SetGoContext(version string, reg *core.GoModuleRegistry) {
	s.goVersion = version
	s.goModuleRegistry = reg
}

// NewServer creates a new MCP server with the given index data.
func NewServer(
	projectPath string,
	pythonVersion string,
	callGraph *core.CallGraph,
	moduleRegistry *core.ModuleRegistry,
	codeGraph *graph.CodeGraph,
	buildTime time.Duration,
	disableAnalytics bool,
) *Server {
	tracker := NewStatusTracker()

	// Mark as ready since we're being created with complete data.
	tracker.StartIndexing()
	stats := &IndexingStats{
		Functions:     len(callGraph.Functions),
		CallEdges:     len(callGraph.Edges),
		Modules:       len(moduleRegistry.Modules),
		Files:         len(moduleRegistry.FileToModule),
		BuildDuration: buildTime,
	}
	tracker.CompleteIndexing(stats)

	// Initialize analytics with stdio transport (default).
	mcpAnalytics := NewAnalytics("stdio", disableAnalytics)
	mcpAnalytics.ReportIndexingComplete(stats)

	s := &Server{
		projectPath:      projectPath,
		pythonVersion:    pythonVersion,
		callGraph:        callGraph,
		moduleRegistry:   moduleRegistry,
		codeGraph:        codeGraph,
		indexedAt:        time.Now(),
		buildTime:        buildTime,
		statusTracker:    tracker,
		degradation:      NewGracefulDegradation(tracker),
		analytics:        mcpAnalytics,
		disableAnalytics: disableAnalytics,
		reachReporter:    updatecheck.NewReachReporter(),
	}
	s.fetchUpdateInfo()
	return s
}

// NewServerWithBackgroundIndexing creates a server that will be populated via background indexing.
func NewServerWithBackgroundIndexing(projectPath, pythonVersion string, disableAnalytics bool) *Server {
	tracker := NewStatusTracker()
	// Starts in StateUninitialized

	s := &Server{
		projectPath:      projectPath,
		pythonVersion:    pythonVersion,
		callGraph:        nil, // Will be set later
		moduleRegistry:   nil, // Will be set later
		codeGraph:        nil, // Will be set later
		statusTracker:    tracker,
		degradation:      NewGracefulDegradation(tracker),
		analytics:        NewAnalytics("stdio", disableAnalytics),
		disableAnalytics: disableAnalytics,
		reachReporter:    updatecheck.NewReachReporter(),
	}
	s.fetchUpdateInfo()
	return s
}

// fetchUpdateInfo performs a single synchronous update-check fetch with a
// 5-second ceiling. Failures are silently swallowed — s.updateInfo stays nil
// and every downstream consumer handles nil gracefully.
// No goroutine is spawned; this is the entire lifecycle for update-check state.
func (s *Server) fetchUpdateInfo() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.updateInfo = updatecheck.Check(ctx, s.version, "mcp", updatecheck.Options{
		HTTPTimeout: 5 * time.Second,
		ManifestURL: mcpManifestURL,
	})
}

// UpdateIndexingStatus updates the indexing progress.
func (s *Server) UpdateIndexingStatus(state IndexingState, phase IndexingPhase, message string, progress float64) {
	s.statusTracker.SetPhase(phase, message)
}

// SetIndexReady marks indexing as complete and updates with indexed data.
func (s *Server) SetIndexReady(callGraph *core.CallGraph, moduleReg *core.ModuleRegistry,
	codeGraph *graph.CodeGraph, buildTime time.Duration) {
	s.callGraph = callGraph
	s.moduleRegistry = moduleReg
	s.codeGraph = codeGraph
	s.buildTime = buildTime
	s.indexedAt = time.Now()

	stats := &IndexingStats{
		Functions:     len(callGraph.Functions),
		CallEdges:     len(callGraph.Edges),
		Modules:       len(moduleReg.Modules),
		Files:         len(moduleReg.FileToModule),
		BuildDuration: buildTime,
	}
	s.statusTracker.CompleteIndexing(stats)
	s.analytics.ReportIndexingComplete(stats)
}

// SetIndexingError marks indexing as failed.
func (s *Server) SetIndexingError(err error) {
	s.statusTracker.FailIndexing(err)
}

// SetTransport updates the analytics transport type (e.g., "http").
func (s *Server) SetTransport(transport string) {
	s.analytics = NewAnalytics(transport, s.disableAnalytics)
}

// ServeStdio starts the MCP server on stdin/stdout.
func (s *Server) ServeStdio() error {
	reader := bufio.NewReader(os.Stdin)

	// Report server started.
	s.analytics.ReportServerStarted()
	defer s.analytics.ReportServerStopped()

	for {
		// Read line from stdin.
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Fprintln(os.Stderr, "Client disconnected")
				return nil // Clean shutdown
			}
			return fmt.Errorf("read error: %w", err)
		}

		// Skip empty lines.
		if len(line) <= 1 {
			continue
		}

		// Parse JSON-RPC request.
		var request JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			s.sendResponse(MakeErrorResponse(nil, ParseError(err.Error())))
			continue
		}

		// Handle request and send response.
		response := s.handleRequest(&request)
		if response != nil {
			s.sendResponse(response)
		}
	}
}

// sendResponse writes a JSON-RPC response to stdout.
func (s *Server) sendResponse(resp *JSONRPCResponse) {
	bytes, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal response: %v\n", err)
		return
	}
	fmt.Println(string(bytes))
}

// handleRequest dispatches to the appropriate handler.
func (s *Server) handleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	startTime := time.Now()

	// Validate JSON-RPC version.
	if req.JSONRPC != "2.0" {
		return MakeErrorResponse(req.ID, InvalidRequestError("jsonrpc must be '2.0'"))
	}

	// Validate method exists.
	if req.Method == "" {
		return MakeErrorResponse(req.ID, InvalidRequestError("method is required"))
	}

	var response *JSONRPCResponse

	switch req.Method {
	case "initialize":
		response = s.handleInitialize(req)
	case "initialized":
		// Acknowledgment notification - no response needed.
		fmt.Fprintln(os.Stderr, "Client initialized")
		return nil
	case "notifications/initialized":
		// Alternative notification format.
		return nil
	case "tools/list":
		response = s.handleToolsList(req)
	case "tools/call":
		response = s.handleToolsCall(req)
	case "status":
		response = s.handleStatus(req)
	case "ping":
		response = SuccessResponse(req.ID, map[string]string{"status": "ok"})
	default:
		response = MakeErrorResponse(req.ID, MethodNotFoundError(req.Method))
	}

	// Log request timing.
	elapsed := time.Since(startTime)
	fmt.Fprintf(os.Stderr, "[%s] %s (%v)\n", req.Method, "completed", elapsed)

	return response
}

// handleInitialize responds to the initialize request.
func (s *Server) handleInitialize(req *JSONRPCRequest) *JSONRPCResponse {
	// Parse client info if needed.
	var params InitializeParams
	if req.Params != nil {
		_ = json.Unmarshal(req.Params, &params)
		fmt.Fprintf(os.Stderr, "Client: %s %s\n", params.ClientInfo.Name, params.ClientInfo.Version)

		// Report client connection (only name/version, no PII).
		s.analytics.ReportClientConnected(params.ClientInfo.Name, params.ClientInfo.Version)
	}

	version := s.version
	if version == "" {
		version = "dev"
	}

	si := ServerInfo{
		Name:    "dev.codepathfinder/pathfinder",
		Version: version,
	}

	// Populate metadata from the update-check result (nil-safe).
	if r := s.updateInfo; r != nil {
		md := &ServerMetadata{}
		if r.Upgrade != nil {
			md.LatestVersion = r.Upgrade.Latest
			md.UpdateMessage = r.Upgrade.Message
			md.ReleaseURL = r.Upgrade.ReleaseURL
		}
		if r.Announcement != nil {
			md.Announcement = &AnnouncementInfo{
				ID:    r.Announcement.ID,
				Level: r.Announcement.Level,
				Title: r.Announcement.Title,
				Text:  r.Announcement.Text,
				URL:   r.Announcement.URL,
			}
		}
		// Only set Metadata when at least one field is non-empty.
		if *md != (ServerMetadata{}) {
			si.Metadata = md
		}
	}

	return SuccessResponse(req.ID, InitializeResult{
		ProtocolVersion: "2024-11-05",
		ServerInfo:      si,
		Capabilities: Capabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
		},
	})
}

// handleToolsList returns the list of available tools, with the status tool
// description enriched with an upgrade or announcement hint when available.
func (s *Server) handleToolsList(req *JSONRPCRequest) *JSONRPCResponse {
	tools := s.getToolDefinitions()
	if s.updateInfo != nil {
		for i := range tools {
			if tools[i].Name == "status" {
				tools[i].Description = formatStatusDescription(tools[i].Description, s.updateInfo)
				break
			}
		}
	}
	// Fire reach analytics after description injection (once per 24-hour window).
	s.reportReachIfNeeded()
	return SuccessResponse(req.ID, ToolsListResult{
		Tools: tools,
	})
}

// handleToolsCall executes a tool.
func (s *Server) handleToolsCall(req *JSONRPCRequest) *JSONRPCResponse {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return MakeErrorResponse(req.ID, InvalidParamsError(err.Error()))
	}

	if params.Name == "" {
		return MakeErrorResponse(req.ID, InvalidParamsError("tool name is required"))
	}

	fmt.Fprintf(os.Stderr, "Tool call: %s\n", params.Name)

	// Track tool call metrics.
	metrics := s.analytics.StartToolCall(params.Name)
	result, isError := s.executeTool(params.Name, params.Arguments)
	s.analytics.EndToolCall(metrics, !isError)

	return SuccessResponse(req.ID, ToolResult{
		Content: []ContentBlock{
			{
				Type: "text",
				Text: result,
			},
		},
		IsError: isError,
	})
}

// handleStatus returns the current indexing status, enriched with any
// available update-check fields.
func (s *Server) handleStatus(req *JSONRPCRequest) *JSONRPCResponse {
	result := s.degradation.GetStatusJSON()
	s.injectUpdateInfo(result)
	return SuccessResponse(req.ID, result)
}

// injectUpdateInfo merges update-check fields into a status map in-place.
// It is a no-op when s.updateInfo is nil.
func (s *Server) injectUpdateInfo(result map[string]any) {
	r := s.updateInfo
	if r == nil {
		return
	}
	if r.Upgrade != nil {
		result["latest_version"] = r.Upgrade.Latest  //nolint:tagliatelle
		result["update_message"] = r.Upgrade.Message //nolint:tagliatelle
		result["release_url"] = r.Upgrade.ReleaseURL //nolint:tagliatelle
	}
	if r.Announcement != nil {
		result["announcement"] = map[string]any{
			"id":    r.Announcement.ID,
			"level": r.Announcement.Level,
			"title": r.Announcement.Title,
			"text":  r.Announcement.Text,
			"url":   r.Announcement.URL,
		}
	}
}

// GetStatusTracker returns the status tracker for external use.
func (s *Server) GetStatusTracker() *StatusTracker {
	return s.statusTracker
}

// IsReady returns true if the server is ready to handle tool requests.
func (s *Server) IsReady() bool {
	return s.statusTracker.IsReady()
}

