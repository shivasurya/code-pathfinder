package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HTTPConfig holds configuration for the HTTP server.
type HTTPConfig struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	AllowedOrigins  []string
}

// DefaultHTTPConfig returns sensible defaults.
func DefaultHTTPConfig() *HTTPConfig {
	return &HTTPConfig{
		Address:         ":8080",
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		AllowedOrigins:  []string{"*"},
	}
}

// HTTPServer wraps the MCP server with HTTP transport.
type HTTPServer struct {
	server     *Server
	httpServer *http.Server
	config     *HTTPConfig
	mu         sync.RWMutex
	running    bool
}

// NewHTTPServer creates a new HTTP server wrapping the MCP server.
func NewHTTPServer(mcpServer *Server, config *HTTPConfig) *HTTPServer {
	if config == nil {
		config = DefaultHTTPConfig()
	}

	return &HTTPServer{
		server: mcpServer,
		config: config,
	}
}

// ServeHTTP implements http.Handler for JSON-RPC requests.
func (h *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers.
	h.setCORSHeaders(w, r)

	// Handle preflight requests.
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Only accept POST for JSON-RPC.
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	// Verify content type.
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		h.writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	// Read request body.
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	// Parse JSON-RPC request.
	var request JSONRPCRequest
	if err := json.Unmarshal(body, &request); err != nil {
		response := MakeErrorResponse(nil, ParseError(err.Error()))
		h.writeJSON(w, http.StatusOK, response)
		return
	}

	// Handle the request.
	response := h.server.handleRequest(&request)

	// Write response.
	h.writeJSON(w, http.StatusOK, response)
}

// Start starts the HTTP server.
func (h *HTTPServer) Start() error {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	mux := http.NewServeMux()
	mux.Handle("/", h)
	mux.HandleFunc("/health", h.healthHandler)

	h.httpServer = &http.Server{
		Addr:         h.config.Address,
		Handler:      mux,
		ReadTimeout:  h.config.ReadTimeout,
		WriteTimeout: h.config.WriteTimeout,
	}

	h.running = true
	h.mu.Unlock()

	// Start listening.
	lc := net.ListenConfig{}
	listener, err := lc.Listen(context.Background(), "tcp", h.config.Address)
	if err != nil {
		h.mu.Lock()
		h.running = false
		h.mu.Unlock()
		return fmt.Errorf("failed to listen on %s: %w", h.config.Address, err)
	}

	fmt.Printf("MCP HTTP server listening on %s\n", h.config.Address)
	return h.httpServer.Serve(listener)
}

// StartAsync starts the HTTP server in a goroutine and returns immediately.
func (h *HTTPServer) StartAsync() error {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return fmt.Errorf("server already running")
	}

	mux := http.NewServeMux()
	mux.Handle("/", h)
	mux.HandleFunc("/health", h.healthHandler)

	h.httpServer = &http.Server{
		Addr:         h.config.Address,
		Handler:      mux,
		ReadTimeout:  h.config.ReadTimeout,
		WriteTimeout: h.config.WriteTimeout,
	}

	h.running = true
	h.mu.Unlock()

	// Start listening.
	lc := net.ListenConfig{}
	listener, err := lc.Listen(context.Background(), "tcp", h.config.Address)
	if err != nil {
		h.mu.Lock()
		h.running = false
		h.mu.Unlock()
		return fmt.Errorf("failed to listen on %s: %w", h.config.Address, err)
	}

	go func() {
		_ = h.httpServer.Serve(listener)
	}()

	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (h *HTTPServer) Shutdown(ctx context.Context) error {
	h.mu.Lock()
	if !h.running {
		h.mu.Unlock()
		return nil
	}
	h.running = false
	h.mu.Unlock()

	if h.httpServer == nil {
		return nil
	}

	return h.httpServer.Shutdown(ctx)
}

// IsRunning returns whether the server is running.
func (h *HTTPServer) IsRunning() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.running
}

// Address returns the configured address.
func (h *HTTPServer) Address() string {
	return h.config.Address
}

// healthHandler returns server health status.
func (h *HTTPServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w, r)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	status := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	h.writeJSON(w, http.StatusOK, status)
}

// setCORSHeaders sets CORS headers based on configuration.
func (h *HTTPServer) setCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}

	// Check if origin is allowed.
	allowed := false
	for _, o := range h.config.AllowedOrigins {
		if o == "*" || o == origin {
			allowed = true
			break
		}
	}

	if allowed {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	}

	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// writeJSON writes a JSON response.
func (h *HTTPServer) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes an error response.
func (h *HTTPServer) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}

// SSEServer provides Server-Sent Events transport for streaming.
type SSEServer struct {
	httpServer *HTTPServer
}

// NewSSEServer creates a new SSE server.
func NewSSEServer(httpServer *HTTPServer) *SSEServer {
	return &SSEServer{httpServer: httpServer}
}

// ServeSSE handles SSE connections for streaming responses.
func (s *SSEServer) ServeSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	s.httpServer.setCORSHeaders(w, r)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection event.
	fmt.Fprintf(w, "event: connected\ndata: {\"status\": \"connected\"}\n\n")
	flusher.Flush()

	// Keep connection open until client disconnects.
	<-r.Context().Done()
}

// StreamingHTTPHandler provides a handler for streaming JSON-RPC over HTTP.
type StreamingHTTPHandler struct {
	server *Server
}

// NewStreamingHTTPHandler creates a new streaming handler.
func NewStreamingHTTPHandler(server *Server) *StreamingHTTPHandler {
	return &StreamingHTTPHandler{server: server}
}

// HandleStream processes a stream of JSON-RPC requests.
func (s *StreamingHTTPHandler) HandleStream(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	encoder := json.NewEncoder(w)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var request JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			response := MakeErrorResponse(nil, ParseError(err.Error()))
			if err := encoder.Encode(response); err != nil {
				return err
			}
			continue
		}

		response := s.server.handleRequest(&request)
		if err := encoder.Encode(response); err != nil {
			return err
		}
	}

	return scanner.Err()
}
