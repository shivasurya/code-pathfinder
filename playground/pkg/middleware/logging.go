package middleware

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// LoggingResponseWriter is a custom response writer that captures the status code
type LoggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (lrw *LoggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware logs request details with security context
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Get client IP with security checks
		clientIP := r.Header.Get("X-Real-IP")
		if clientIP == "" {
			clientIP = r.Header.Get("X-Forwarded-For")
			if clientIP == "" {
				// Extract IP without port
				clientIP = strings.Split(r.RemoteAddr, ":")[0]
			}
		}

		// Get user agent (sanitized)
		userAgent := strings.ReplaceAll(r.Header.Get("User-Agent"), "\n", "")

		// Get request ID or generate one
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add request ID to response headers for tracing
		w.Header().Set("X-Request-ID", requestID)

		// Create a response writer that captures the status code
		lrw := &LoggingResponseWriter{
			ResponseWriter: w,
			statusCode:    http.StatusOK,
		}

		// Call the next handler
		next.ServeHTTP(lrw, r)

		// Calculate request duration
		duration := time.Since(start)

		// Get content length
		contentLength := r.Header.Get("Content-Length")

		// Log request details with security context
		log.Printf("[%s] [%s] Method=%s Path=%s IP=%s Status=%d Duration=%v Size=%s UA=%s Referer=%s Protocol=%s Host=%s",
			time.Now().Format(time.RFC3339),
			requestID,
			r.Method,
			r.URL.Path,
			clientIP,
			lrw.statusCode,
			duration,
			contentLength,
			userAgent,
			r.Referer(),
			r.Proto,
			r.Host,
		)
	})
}
