// Package logging exports internal functions for testing.
package logging

import (
	"log/slog"
	"net/http"
)

// ExportIsSensitiveKey exposes isSensitiveKey for testing.
func ExportIsSensitiveKey(key string) bool {
	return isSensitiveKey(key)
}

// ExportToLower exposes toLower for testing.
func ExportToLower(s string) string {
	return toLower(s)
}

// ExportContains exposes contains for testing.
func ExportContains(s, substr string) bool {
	return contains(s, substr)
}

// ExportParseLevel exposes parseLevel for testing.
func ExportParseLevel(level string) slog.Level {
	return parseLevel(level)
}

// ExportGenerateRequestID exposes generateRequestID for testing.
func ExportGenerateRequestID() string {
	return generateRequestID()
}

// ExportSetGlobalLogger sets the global logger for testing.
func ExportSetGlobalLogger(l *slog.Logger) {
	setLogger(l)
}

// ExportClearGlobalLogger clears the global logger for testing.
func ExportClearGlobalLogger() {
	clearLogger()
}

// ExportRequestIDKeyValue returns the requestIDKey for testing.
func ExportRequestIDKeyValue() any {
	return requestIDKey
}

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
