// Package logging exports internal functions for testing.
package logging

import (
	"log/slog"
	"net/http"
	"sync"
)

// IsSensitiveKey exports isSensitiveKey for testing.
var IsSensitiveKey = isSensitiveKey

// ToLower exports toLower for testing.
var ToLower = toLower

// Contains exports contains for testing.
var Contains = contains

// ParseLevel exports parseLevel for testing.
var ParseLevel = parseLevel

// GenerateRequestID exports generateRequestID for testing.
var GenerateRequestID = generateRequestID

// LoggerMu returns a reference to loggerMu for testing.
func LoggerMu() *sync.RWMutex {
	return &loggerMu
}

// SetGlobalLogger sets the global logger for testing.
func SetGlobalLogger(l *slog.Logger) {
	loggerMu.Lock()
	globalLogger = l
	loggerMu.Unlock()
}

// ClearGlobalLogger clears the global logger for testing.
func ClearGlobalLogger() {
	loggerMu.Lock()
	globalLogger = nil
	loggerMu.Unlock()
}

// RequestIDKeyValue returns the requestIDKey for testing.
var RequestIDKeyValue = requestIDKey

// TestResponseWriter is a wrapper for responseWriter for testing.
type TestResponseWriter struct {
	rw *responseWriter
}

// NewTestResponseWriter creates a responseWriter for testing.
func NewTestResponseWriter(w http.ResponseWriter, initialStatus int) *TestResponseWriter {
	return &TestResponseWriter{
		rw: &responseWriter{
			ResponseWriter: w,
			status:         initialStatus,
		},
	}
}

// WriteHeader wraps the internal WriteHeader.
func (t *TestResponseWriter) WriteHeader(code int) {
	t.rw.WriteHeader(code)
}

// Write wraps the internal Write.
func (t *TestResponseWriter) Write(data []byte) (int, error) {
	return t.rw.Write(data)
}

// Unwrap wraps the internal Unwrap.
func (t *TestResponseWriter) Unwrap() http.ResponseWriter {
	return t.rw.Unwrap()
}

// Header wraps the internal Header.
func (t *TestResponseWriter) Header() http.Header {
	return t.rw.Header()
}

// Status returns the response status code.
func (t *TestResponseWriter) Status() int {
	return t.rw.status
}

// WroteHeader returns whether WriteHeader was called.
func (t *TestResponseWriter) WroteHeader() bool {
	return t.rw.wroteHeader
}

// RedactAttr exports redactAttr for testing.
func (h *RedactingHandler) RedactAttr(attr slog.Attr) slog.Attr {
	return h.redactAttr(attr)
}

// Inner returns the inner handler for testing.
func (h *RedactingHandler) Inner() slog.Handler {
	return h.inner
}
