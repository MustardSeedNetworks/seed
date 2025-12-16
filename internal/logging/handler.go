package logging

import (
	"context"
	"log/slog"
	"net/http"
)

// RedactingHandler is a slog.Handler that automatically redacts sensitive data
// from log messages and attributes before passing them to the underlying handler.
//
// It uses the existing redaction functions (RedactString, RedactMap, RedactHeaders)
// to ensure no sensitive data like passwords, tokens, or API keys appear in logs.
type RedactingHandler struct {
	inner slog.Handler
}

// NewRedactingHandler creates a new RedactingHandler wrapping the given handler.
func NewRedactingHandler(inner slog.Handler) *RedactingHandler {
	return &RedactingHandler{inner: inner}
}

// Enabled reports whether the handler handles records at the given level.
func (h *RedactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle processes the log record, redacting sensitive data from the message
// and all attributes before passing to the inner handler.
//
//nolint:gocritic // hugeParam: slog.Handler interface requires value receiver.
func (h *RedactingHandler) Handle(ctx context.Context, r slog.Record) error {
	// Redact the message
	r.Message = RedactString(r.Message)

	// Create a new record with redacted attributes
	newRecord := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

	// Redact each attribute
	r.Attrs(func(a slog.Attr) bool {
		newRecord.AddAttrs(h.redactAttr(a))
		return true
	})

	return h.inner.Handle(ctx, newRecord)
}

// WithAttrs returns a new handler with the given attributes redacted and added.
func (h *RedactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	redacted := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		redacted[i] = h.redactAttr(a)
	}
	return &RedactingHandler{
		inner: h.inner.WithAttrs(redacted),
	}
}

// WithGroup returns a new handler with the given group name.
func (h *RedactingHandler) WithGroup(name string) slog.Handler {
	return &RedactingHandler{
		inner: h.inner.WithGroup(name),
	}
}

// redactAttr redacts sensitive data from a single attribute.
func (h *RedactingHandler) redactAttr(a slog.Attr) slog.Attr {
	key := a.Key

	// Check if the key itself indicates sensitive data
	if isSensitiveKey(key) {
		return slog.String(key, "[REDACTED]")
	}

	// Handle different value types
	switch v := a.Value.Any().(type) {
	case string:
		return slog.String(key, RedactString(v))

	case error:
		if v != nil {
			return slog.String(key, RedactString(v.Error()))
		}
		return a

	case http.Header:
		// Use existing RedactHeaders for HTTP headers
		redacted := RedactHeaders(v)
		return slog.Any(key, redacted)

	case map[string]interface{}:
		// Use existing RedactMap for maps
		redacted := RedactMap(v)
		return slog.Any(key, redacted)

	case map[string]string:
		// Convert and redact string maps
		m := make(map[string]interface{}, len(v))
		for k, val := range v {
			m[k] = val
		}
		redacted := RedactMap(m)
		return slog.Any(key, redacted)

	case []slog.Attr:
		// Handle nested groups
		redactedAttrs := make([]any, 0, len(v)*2)
		for _, nested := range v {
			redactedNested := h.redactAttr(nested)
			redactedAttrs = append(redactedAttrs, redactedNested.Key, redactedNested.Value.Any())
		}
		return slog.Group(key, redactedAttrs...)

	default:
		// For other types, return as-is (numbers, bools, etc.)
		return a
	}
}

// isSensitiveKey checks if an attribute key indicates sensitive data.
func isSensitiveKey(key string) bool {
	// Check against common sensitive field names (case-insensitive)
	sensitiveKeys := map[string]bool{
		"password":      true,
		"passwd":        true,
		"pwd":           true,
		"secret":        true,
		"token":         true,
		"api_key":       true,
		"apikey":        true,
		"auth":          true,
		"authorization": true,
		"bearer":        true,
		"credential":    true,
		"credentials":   true,
		"private_key":   true,
		"privatekey":    true,
		"jwt":           true,
		"session":       true,
		"cookie":        true,
	}

	// Convert to lowercase for comparison
	lowerKey := toLower(key)
	if sensitiveKeys[lowerKey] {
		return true
	}

	// Also check if the key contains these substrings
	for sensitiveKey := range sensitiveKeys {
		if contains(lowerKey, sensitiveKey) {
			return true
		}
	}

	return false
}

// toLower is a simple lowercase conversion for ASCII strings.
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// contains checks if s contains substr (simple implementation for ASCII).
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
