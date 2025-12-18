package logging

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"time"
)

const (
	// RequestIDHeader is the HTTP header used to pass request IDs.
	RequestIDHeader = "X-Request-ID"
)

// RequestIDMiddleware generates a unique request ID for each incoming request
// and adds it to the request context. If the client sends an X-Request-ID header,
// that value is used instead (useful for distributed tracing).
//
// The request ID is available via RequestIDFromContext(r.Context()) and is
// automatically included in logs when using FromContext(ctx) to get the logger.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for existing request ID from upstream proxy/client
		requestID := r.Header.Get(RequestIDHeader)

		// Generate a new one if not provided
		if requestID == "" {
			requestID = generateRequestID()
		}

		// Add request ID to response header for client correlation
		w.Header().Set(RequestIDHeader, requestID)

		// Add request ID to context
		ctx := WithRequestID(r.Context(), requestID)

		// Pass the modified request to the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// generateRequestID creates a unique request ID using random bytes.
// Format: 16 hex characters (8 bytes of randomness).
func generateRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000")))[:16]
	}
	return hex.EncodeToString(b)
}

// LoggingMiddleware logs HTTP requests with timing information.
// It captures the request method, path, status code, and duration.
//
// This middleware should be placed after RequestIDMiddleware so that
// the request ID is available in the logs.
//
//nolint:revive // stuttering: keeping name for backward compatibility.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log the request
		duration := time.Since(start)
		logger := FromContext(r.Context())

		// Skip logging for health checks and static assets
		if r.URL.Path == "/api/health" || r.URL.Path == "/health" {
			return
		}

		logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"duration_ms", duration.Milliseconds(),
			"client_ip", GetClientIP(r),
			"user_agent", r.UserAgent(),
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

// WriteHeader captures the status code before calling the underlying WriteHeader.
func (w *responseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

// Write calls the underlying Write and sets status to 200 if not already set.
func (w *responseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap returns the underlying ResponseWriter, supporting http.ResponseController.
func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// Hijack implements http.Hijacker for WebSocket support (fixes #ws-hijacker).
// This allows the logging middleware to be used with WebSocket endpoints.
func (w *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := w.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}
