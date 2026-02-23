package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultHTTPConfig(t *testing.T) {
	config := DefaultHTTPConfig()

	assert.Equal(t, ":8080", config.Address)
	assert.Equal(t, 30*time.Second, config.ReadTimeout)
	assert.Equal(t, 30*time.Second, config.WriteTimeout)
	assert.Equal(t, 5*time.Second, config.ShutdownTimeout)
	assert.Equal(t, []string{"*"}, config.AllowedOrigins)
}

func TestNewHTTPServer(t *testing.T) {
	mcpServer := createTestServer()

	t.Run("with config", func(t *testing.T) {
		config := &HTTPConfig{Address: ":9090"}
		httpServer := NewHTTPServer(mcpServer, config)

		assert.NotNil(t, httpServer)
		assert.Equal(t, ":9090", httpServer.Address())
	})

	t.Run("with nil config uses defaults", func(t *testing.T) {
		httpServer := NewHTTPServer(mcpServer, nil)

		assert.NotNil(t, httpServer)
		assert.Equal(t, ":8080", httpServer.Address())
	})
}

func TestHTTPServer_ServeHTTP_Initialize(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)

	params := map[string]any{
		"protocolVersion": "2024-11-05",
		"clientInfo": map[string]any{
			"name":    "test-client",
			"version": "1.0",
		},
	}
	paramsBytes, _ := json.Marshal(params)

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  paramsBytes,
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	httpServer.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response JSONRPCResponse
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "2.0", response.JSONRPC)
	assert.NotNil(t, response.Result)
	assert.Nil(t, response.Error)
}

func TestHTTPServer_ServeHTTP_ToolsCall(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)

	params := map[string]any{
		"name":      "get_index_info",
		"arguments": map[string]any{},
	}
	paramsBytes, _ := json.Marshal(params)

	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  paramsBytes,
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	httpServer.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "project_path")
}

func TestHTTPServer_ServeHTTP_MethodNotAllowed(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/", nil)
			rec := httptest.NewRecorder()
			httpServer.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
			assert.Contains(t, rec.Body.String(), "Only POST method is allowed")
		})
	}
}

func TestHTTPServer_ServeHTTP_WrongContentType(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "text/plain")

	rec := httptest.NewRecorder()
	httpServer.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
	assert.Contains(t, rec.Body.String(), "Content-Type must be application/json")
}

func TestHTTPServer_ServeHTTP_InvalidJSON(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	httpServer.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response JSONRPCResponse
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.NotNil(t, response.Error)
	assert.Equal(t, ErrCodeParseError, response.Error.Code)
}

func TestHTTPServer_ServeHTTP_CORS(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)

	t.Run("preflight request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "http://example.com")

		rec := httptest.NewRecorder()
		httpServer.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, "http://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "POST, OPTIONS", rec.Header().Get("Access-Control-Allow-Methods"))
	})

	t.Run("regular request has CORS headers", func(t *testing.T) {
		request := JSONRPCRequest{JSONRPC: "2.0", ID: 1, Method: "ping"}
		body, _ := json.Marshal(request)

		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://example.com")

		rec := httptest.NewRecorder()
		httpServer.ServeHTTP(rec, req)

		assert.Equal(t, "http://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestHTTPServer_ServeHTTP_CORSAllowedOrigins(t *testing.T) {
	mcpServer := createTestServer()
	config := &HTTPConfig{
		Address:        ":8080",
		AllowedOrigins: []string{"http://allowed.com"},
	}
	httpServer := NewHTTPServer(mcpServer, config)

	t.Run("allowed origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "http://allowed.com")

		rec := httptest.NewRecorder()
		httpServer.ServeHTTP(rec, req)

		assert.Equal(t, "http://allowed.com", rec.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("disallowed origin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		req.Header.Set("Origin", "http://notallowed.com")

		rec := httptest.NewRecorder()
		httpServer.ServeHTTP(rec, req)

		assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestHTTPServer_HealthHandler(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	httpServer.healthHandler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
	assert.Contains(t, response, "timestamp")
}

func TestHTTPServer_HealthHandler_Preflight(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)

	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	rec := httptest.NewRecorder()

	httpServer.healthHandler(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestHTTPServer_IsRunning(t *testing.T) {
	mcpServer := createTestServer()
	config := &HTTPConfig{Address: "127.0.0.1:0"} // Use port 0 for random available port
	httpServer := NewHTTPServer(mcpServer, config)

	assert.False(t, httpServer.IsRunning())

	err := httpServer.StartAsync()
	require.NoError(t, err)
	assert.True(t, httpServer.IsRunning())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = httpServer.Shutdown(ctx)
	require.NoError(t, err)
	assert.False(t, httpServer.IsRunning())
}

func TestHTTPServer_StartAsync_AlreadyRunning(t *testing.T) {
	mcpServer := createTestServer()
	config := &HTTPConfig{Address: "127.0.0.1:0"}
	httpServer := NewHTTPServer(mcpServer, config)

	err := httpServer.StartAsync()
	require.NoError(t, err)
	defer func() {
		ctx := context.Background()
		_ = httpServer.Shutdown(ctx)
	}()

	err = httpServer.StartAsync()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestHTTPServer_Shutdown_NotRunning(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)

	ctx := context.Background()
	err := httpServer.Shutdown(ctx)
	assert.NoError(t, err)
}

func TestNewSSEServer(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)
	sseServer := NewSSEServer(httpServer)

	assert.NotNil(t, sseServer)
}

func TestNewStreamingHTTPHandler(t *testing.T) {
	mcpServer := createTestServer()
	handler := NewStreamingHTTPHandler(mcpServer)

	assert.NotNil(t, handler)
}

func TestStreamingHTTPHandler_HandleStream(t *testing.T) {
	mcpServer := createTestServer()
	handler := NewStreamingHTTPHandler(mcpServer)

	t.Run("single request", func(t *testing.T) {
		input := `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n"
		reader := strings.NewReader(input)
		var output bytes.Buffer

		err := handler.HandleStream(reader, &output)
		require.NoError(t, err)

		var response JSONRPCResponse
		err = json.Unmarshal(output.Bytes(), &response)
		require.NoError(t, err)
		assert.Equal(t, "2.0", response.JSONRPC)
	})

	t.Run("multiple requests", func(t *testing.T) {
		input := `{"jsonrpc":"2.0","id":1,"method":"ping"}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
`
		reader := strings.NewReader(input)
		var output bytes.Buffer

		err := handler.HandleStream(reader, &output)
		require.NoError(t, err)

		// Should have two responses.
		lines := strings.Split(strings.TrimSpace(output.String()), "\n")
		assert.Len(t, lines, 2)
	})

	t.Run("empty lines skipped", func(t *testing.T) {
		input := `{"jsonrpc":"2.0","id":1,"method":"ping"}

{"jsonrpc":"2.0","id":2,"method":"ping"}
`
		reader := strings.NewReader(input)
		var output bytes.Buffer

		err := handler.HandleStream(reader, &output)
		require.NoError(t, err)

		lines := strings.Split(strings.TrimSpace(output.String()), "\n")
		assert.Len(t, lines, 2)
	})

	t.Run("invalid JSON returns error response", func(t *testing.T) {
		input := "not valid json\n"
		reader := strings.NewReader(input)
		var output bytes.Buffer

		err := handler.HandleStream(reader, &output)
		require.NoError(t, err)

		var response JSONRPCResponse
		err = json.Unmarshal(output.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response.Error)
		assert.Equal(t, ErrCodeParseError, response.Error.Code)
	})
}

func TestHTTPServer_Integration(t *testing.T) {
	mcpServer := createTestServer()
	config := &HTTPConfig{Address: "127.0.0.1:0"}
	httpServer := NewHTTPServer(mcpServer, config)

	err := httpServer.StartAsync()
	require.NoError(t, err)
	defer func() {
		ctx := context.Background()
		_ = httpServer.Shutdown(ctx)
	}()

	// Give server time to start.
	time.Sleep(50 * time.Millisecond)

	// The server is running, we can test via httptest.
	// Since we don't have the actual address easily, use ServeHTTP directly.
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}
	body, _ := json.Marshal(request)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	httpServer.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "tools")
}

func TestHTTPServer_LargeRequest(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)

	// Create a request within the 1MB limit.
	request := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "ping",
	}
	body, _ := json.Marshal(request)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	httpServer.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestHTTPServer_ReadBodyError(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)

	req := httptest.NewRequest(http.MethodPost, "/", &errorReader{})
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	httpServer.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "Failed to read request body")
}

func TestHTTPServer_NoOriginHeader(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)

	request := JSONRPCRequest{JSONRPC: "2.0", ID: 1, Method: "ping"}
	body, _ := json.Marshal(request)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No Origin header set.

	rec := httptest.NewRecorder()
	httpServer.ServeHTTP(rec, req)

	// Should use "*" as fallback.
	assert.Equal(t, "*", rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestHTTPServer_Start_AlreadyRunning(t *testing.T) {
	mcpServer := createTestServer()
	config := &HTTPConfig{Address: "127.0.0.1:0"}
	httpServer := NewHTTPServer(mcpServer, config)

	// Start async first.
	err := httpServer.StartAsync()
	require.NoError(t, err)
	defer func() {
		ctx := context.Background()
		_ = httpServer.Shutdown(ctx)
	}()

	// Try to start synchronously - should fail.
	err = httpServer.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
}

func TestHTTPServer_Start_Blocking(t *testing.T) {
	mcpServer := createTestServer()
	config := &HTTPConfig{Address: "127.0.0.1:0"}
	httpServer := NewHTTPServer(mcpServer, config)

	// Start in goroutine since it blocks.
	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.Start()
	}()

	// Give server time to start.
	time.Sleep(50 * time.Millisecond)

	// Verify it's running.
	assert.True(t, httpServer.IsRunning())

	// Shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := httpServer.Shutdown(ctx)
	require.NoError(t, err)

	// Wait for Start() to return.
	select {
	case err := <-errCh:
		// Server closed error is expected.
		assert.Contains(t, err.Error(), "Server closed")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for Start to return")
	}
}

func TestSSEServer_ServeSSE(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)
	sseServer := NewSSEServer(httpServer)

	// Create a request with a cancellable context.
	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/sse", nil).WithContext(ctx)
	req.Header.Set("Origin", "http://example.com")

	rec := httptest.NewRecorder()

	// Run ServeSSE in a goroutine since it blocks.
	done := make(chan struct{})
	go func() {
		sseServer.ServeSSE(rec, req)
		close(done)
	}()

	// Give it time to set headers and send connected event.
	time.Sleep(50 * time.Millisecond)

	// Cancel context to close connection.
	cancel()

	// Wait for ServeSSE to return.
	select {
	case <-done:
		// Good, it returned.
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for ServeSSE to return")
	}

	// Verify SSE headers.
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", rec.Header().Get("Connection"))

	// Verify connected event was sent.
	assert.Contains(t, rec.Body.String(), "event: connected")
	assert.Contains(t, rec.Body.String(), "status")
}

// mockResponseWriter doesn't implement http.Flusher.
type noFlushResponseWriter struct {
	http.ResponseWriter
}

func TestSSEServer_ServeSSE_NoFlusher(t *testing.T) {
	mcpServer := createTestServer()
	httpServer := NewHTTPServer(mcpServer, nil)
	sseServer := NewSSEServer(httpServer)

	req := httptest.NewRequest(http.MethodGet, "/sse", nil)

	// Use a writer that doesn't implement Flusher.
	rec := httptest.NewRecorder()
	noFlush := &noFlushResponseWriter{rec}

	sseServer.ServeSSE(noFlush, req)

	// Should return error response.
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "SSE not supported")
}

func TestHTTPServer_Shutdown_WithContext(t *testing.T) {
	mcpServer := createTestServer()
	config := &HTTPConfig{Address: "127.0.0.1:0"}
	httpServer := NewHTTPServer(mcpServer, config)

	err := httpServer.StartAsync()
	require.NoError(t, err)

	// Shutdown with cancelled context should still work.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = httpServer.Shutdown(ctx)
	assert.NoError(t, err)
	assert.False(t, httpServer.IsRunning())
}

func TestHTTPServer_Start_InvalidAddress(t *testing.T) {
	mcpServer := createTestServer()
	// Use an invalid address that should fail to listen.
	config := &HTTPConfig{Address: "invalid:address:format:99999"}
	httpServer := NewHTTPServer(mcpServer, config)

	err := httpServer.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to listen")
	assert.False(t, httpServer.IsRunning())
}

func TestHTTPServer_StartAsync_InvalidAddress(t *testing.T) {
	mcpServer := createTestServer()
	// Use an invalid address that should fail to listen.
	config := &HTTPConfig{Address: "invalid:address:format:99999"}
	httpServer := NewHTTPServer(mcpServer, config)

	err := httpServer.StartAsync()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to listen")
	assert.False(t, httpServer.IsRunning())
}

// errorWriter always returns an error.
type errorWriter struct{}

func (e *errorWriter) Write(p []byte) (n int, err error) {
	return 0, io.ErrClosedPipe
}

func TestStreamingHTTPHandler_HandleStream_WriteError(t *testing.T) {
	mcpServer := createTestServer()
	handler := NewStreamingHTTPHandler(mcpServer)

	input := `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n"
	reader := strings.NewReader(input)

	err := handler.HandleStream(reader, &errorWriter{})
	assert.Error(t, err)
}

func TestStreamingHTTPHandler_HandleStream_WriteErrorOnParseError(t *testing.T) {
	mcpServer := createTestServer()
	handler := NewStreamingHTTPHandler(mcpServer)

	// Invalid JSON to trigger parse error, which then tries to write.
	input := "not valid json\n"
	reader := strings.NewReader(input)

	err := handler.HandleStream(reader, &errorWriter{})
	assert.Error(t, err)
}
