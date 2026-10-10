package logging

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"
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

// ExportSetGlobalLogger sets the global logger for testing.
func ExportSetGlobalLogger(l *slog.Logger) {
	setLogger(l)
}

// ExportClearGlobalLogger clears the global logger for testing.
func ExportClearGlobalLogger() {
	clearLogger()
}

// RedactAttr exports redactAttr for testing.
func (h *RedactingHandler) RedactAttr(attr slog.Attr) slog.Attr {
	return h.redactAttr(attr)
}

// Inner returns the inner handler for testing.
func (h *RedactingHandler) Inner() slog.Handler {
	return h.inner
}

// ExportUserIDKeyValue returns the userIDKey for testing.
func ExportUserIDKeyValue() any {
	return userIDKey
}

// ContextWithRequestID returns parent carrying an ID the route Registrar
// assigned, and that ID. The Registrar is the only producer of request IDs, so
// a test obtains one the way production does: by serving a request through it.
func ContextWithRequestID(parent context.Context, tb testing.TB) (context.Context, string) {
	tb.Helper()
	reg := route.New(route.Config{
		Error:        func(http.ResponseWriter, *http.Request, int, string, string) {},
		MaxBodyBytes: 1,
		Logger:       slog.New(slog.DiscardHandler),
	})
	var served *http.Request
	reg.Register(route.Route{Path: "/", Handler: func(_ http.ResponseWriter, r *http.Request) { served = r }})
	reg.Handler().ServeHTTP(httptest.NewRecorder(),
		httptest.NewRequestWithContext(parent, http.MethodGet, "/", http.NoBody))
	id := route.RequestID(served.Context())
	if id == "" {
		tb.Fatal("the Registrar assigned no request ID")
	}
	return served.Context(), id
}
