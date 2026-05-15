package logging_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

// testHandler is a simple handler that captures log records for testing.
// mu makes records-append safe under TestRedactingHandler_Handle_MultipleWriters,
// which deliberately exercises concurrent Handle calls.
type testHandler struct {
	mu      sync.Mutex
	records []slog.Record
	attrs   []slog.Attr
	groups  []string
}

func (h *testHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *testHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	return nil
}

func (h *testHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := &testHandler{
		records: h.records,
		attrs:   append(h.attrs, attrs...),
		groups:  h.groups,
	}
	return newHandler
}

func (h *testHandler) WithGroup(name string) slog.Handler {
	return &testHandler{
		records: h.records,
		attrs:   h.attrs,
		groups:  append(h.groups, name),
	}
}

func TestNewRedactingHandler(t *testing.T) {
	inner := &testHandler{}
	rh := logging.NewRedactingHandler(inner)

	if rh == nil {
		t.Fatal("NewRedactingHandler() returned nil")
	}

	if rh.Inner() != inner {
		t.Error("NewRedactingHandler() did not set inner handler correctly")
	}
}

func TestRedactingHandler_Enabled(t *testing.T) {
	inner := &testHandler{}
	rh := logging.NewRedactingHandler(inner)

	// Should delegate to inner handler
	if !rh.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Enabled() returned false, expected true")
	}
}

func TestRedactingHandler_Handle(t *testing.T) {
	tests := []struct {
		name           string
		message        string
		attrs          []slog.Attr
		expectRedacted bool
		checkMessage   string
	}{
		{
			name:           "message with password",
			message:        "login failed password=secret123",
			expectRedacted: true,
			checkMessage:   "[REDACTED]",
		},
		{
			name:           "clean message",
			message:        "request completed successfully",
			expectRedacted: false,
			checkMessage:   "request completed successfully",
		},
		{
			name:    "attribute with sensitive key",
			message: "user authentication",
			attrs: []slog.Attr{
				slog.String("password", "mysecretpass"),
			},
			expectRedacted: true,
		},
		{
			name:    "attribute with token in value",
			message: "making request",
			attrs: []slog.Attr{
				slog.String("header", "Bearer abc123token"),
			},
			expectRedacted: true,
		},
		{
			name:    "attribute with clean value",
			message: "processing",
			attrs: []slog.Attr{
				slog.String("user", "john"),
				slog.Int("count", 42),
			},
			expectRedacted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := &testHandler{}
			rh := logging.NewRedactingHandler(inner)

			record := slog.NewRecord(time.Now(), slog.LevelInfo, tt.message, 0)
			for _, attr := range tt.attrs {
				record.AddAttrs(attr)
			}

			err := rh.Handle(context.Background(), record)
			if err != nil {
				t.Errorf("Handle() returned error: %v", err)
			}

			if len(inner.records) == 0 {
				t.Fatal("Handle() did not pass record to inner handler")
			}

			result := inner.records[0]

			// Check message redaction
			if tt.expectRedacted && tt.checkMessage != "" {
				if !strings.Contains(result.Message, tt.checkMessage) {
					t.Errorf("Message not properly redacted: got %q", result.Message)
				}
			}
		})
	}
}

func TestRedactingHandler_Handle_Errors(t *testing.T) {
	inner := &testHandler{}
	rh := logging.NewRedactingHandler(inner)

	// Test with error containing sensitive data
	testErr := errors.New("connection failed: password=secret123")
	record := slog.NewRecord(time.Now(), slog.LevelError, "operation failed", 0)
	record.AddAttrs(slog.Any("error", testErr))

	err := rh.Handle(context.Background(), record)
	if err != nil {
		t.Errorf("Handle() returned error: %v", err)
	}

	if len(inner.records) == 0 {
		t.Fatal("Handle() did not pass record to inner handler")
	}

	// Verify the error was redacted
	var foundError bool
	inner.records[0].Attrs(func(a slog.Attr) bool {
		if a.Key == "error" {
			foundError = true
			val := a.Value.String()
			if strings.Contains(val, "secret123") {
				t.Error("Error value was not redacted")
			}
		}
		return true
	})

	if !foundError {
		t.Error("Error attribute not found in record")
	}
}

func TestRedactingHandler_Handle_HTTPHeaders(t *testing.T) {
	inner := &testHandler{}
	rh := logging.NewRedactingHandler(inner)

	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer secrettoken123"},
		"X-Custom":      []string{"safe-value"},
	}

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "http request", 0)
	record.AddAttrs(slog.Any("headers", headers))

	err := rh.Handle(context.Background(), record)
	if err != nil {
		t.Errorf("Handle() returned error: %v", err)
	}

	// Verify headers were redacted
	if len(inner.records) == 0 {
		t.Fatal("Handle() did not pass record to inner handler")
	}

	var foundHeaders bool
	inner.records[0].Attrs(func(a slog.Attr) bool {
		if a.Key == "headers" {
			foundHeaders = true
			// The Authorization header should be redacted
			val := a.Value.String()
			if strings.Contains(val, "secrettoken123") {
				t.Error("Authorization header was not redacted")
			}
		}
		return true
	})

	if !foundHeaders {
		t.Error("Headers attribute not found in record")
	}
}

func TestRedactingHandler_Handle_Maps(t *testing.T) {
	inner := &testHandler{}
	rh := logging.NewRedactingHandler(inner)

	// Test map[string]interface{}
	data := map[string]any{
		"username": "john",
		"password": "secret123",
		"token":    "abc123",
	}

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "processing data", 0)
	record.AddAttrs(slog.Any("data", data))

	err := rh.Handle(context.Background(), record)
	if err != nil {
		t.Errorf("Handle() returned error: %v", err)
	}

	// Test map[string]string
	inner2 := &testHandler{}
	rh2 := logging.NewRedactingHandler(inner2)

	stringData := map[string]string{
		"username": "john",
		"password": "secret456",
	}

	record2 := slog.NewRecord(time.Now(), slog.LevelInfo, "processing", 0)
	record2.AddAttrs(slog.Any("config", stringData))

	err = rh2.Handle(context.Background(), record2)
	if err != nil {
		t.Errorf("Handle() with string map returned error: %v", err)
	}
}

func TestRedactingHandler_WithAttrs(t *testing.T) {
	inner := &testHandler{}
	rh := logging.NewRedactingHandler(inner)

	attrs := []slog.Attr{
		slog.String("password", "secret123"),
		slog.String("user", "john"),
	}

	newHandler := rh.WithAttrs(attrs)

	// Verify new handler is returned
	if newHandler == nil {
		t.Fatal("WithAttrs() returned nil")
	}

	// Verify it's a RedactingHandler
	_, ok := newHandler.(*logging.RedactingHandler)
	if !ok {
		t.Error("WithAttrs() did not return a RedactingHandler")
	}

	// Verify attributes were redacted
	rhNew := newHandler.(*logging.RedactingHandler)
	innerNew := rhNew.Inner().(*testHandler)

	var passwordRedacted bool
	for _, attr := range innerNew.attrs {
		if attr.Key == "password" && attr.Value.String() == "[REDACTED]" {
			passwordRedacted = true
		}
	}

	if !passwordRedacted {
		t.Error("WithAttrs() did not redact sensitive attribute")
	}
}

func TestRedactingHandler_WithGroup(t *testing.T) {
	inner := &testHandler{}
	rh := logging.NewRedactingHandler(inner)

	newHandler := rh.WithGroup("request")

	// Verify new handler is returned
	if newHandler == nil {
		t.Fatal("WithGroup() returned nil")
	}

	// Verify it's a RedactingHandler
	rhNew, ok := newHandler.(*logging.RedactingHandler)
	if !ok {
		t.Error("WithGroup() did not return a RedactingHandler")
	}

	// Verify group was added to inner handler
	innerNew := rhNew.Inner().(*testHandler)
	if len(innerNew.groups) == 0 || innerNew.groups[0] != "request" {
		t.Error("WithGroup() did not add group to inner handler")
	}
}

func TestIsSensitiveKey(t *testing.T) {
	sensitiveKeys := []string{
		"password",
		"Password",
		"PASSWORD",
		"passwd",
		"pwd",
		"secret",
		"token",
		"api_key",
		"apikey",
		"auth",
		"authorization",
		"bearer",
		"credential",
		"credentials",
		"private_key",
		"privatekey",
		"jwt",
		"session",
		"cookie",
		// Keys containing sensitive substrings
		"user_password",
		"auth_token",
		"api_key_value",
		"session_id",
		"jwt_secret",
		"db_password",
	}

	for _, key := range sensitiveKeys {
		t.Run(key, func(t *testing.T) {
			if !logging.ExportIsSensitiveKey(key) {
				t.Errorf("isSensitiveKey(%q) = false, want true", key)
			}
		})
	}

	nonSensitiveKeys := []string{
		"username",
		"email",
		"id",
		"name",
		"count",
		"status",
		"path",
		"method",
		"duration",
		"timestamp",
	}

	for _, key := range nonSensitiveKeys {
		t.Run(key, func(t *testing.T) {
			if logging.ExportIsSensitiveKey(key) {
				t.Errorf("isSensitiveKey(%q) = true, want false", key)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HELLO", "hello"},
		{"Hello", "hello"},
		{"hello", "hello"},
		{"HeLLo WoRLD", "hello world"},
		{"123ABC", "123abc"},
		{"", ""},
		{"ABC123xyz", "abc123xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := logging.ExportToLower(tt.input)
			if result != tt.expected {
				t.Errorf("toLower(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "lo wo", true},
		{"hello", "hello", true},
		{"hello", "hello world", false},
		{"hello", "xyz", false},
		{"", "", true},
		{"hello", "", true},
		{"", "hello", false},
	}

	for _, tt := range tests {
		name := tt.s + "_" + tt.substr
		t.Run(name, func(t *testing.T) {
			result := logging.ExportContains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestRedactingHandler_Integration(t *testing.T) {
	// Integration test with real slog output
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	rh := logging.NewRedactingHandler(baseHandler)
	logger := slog.New(rh)

	// Log with sensitive data
	logger.Info("user login",
		"username", "john",
		"password", "secret123",
		"token", "bearer abc123",
	)

	output := buf.String()

	// Verify sensitive data is redacted
	if strings.Contains(output, "secret123") {
		t.Error("Password value was not redacted in output")
	}

	// Verify non-sensitive data is present
	if !strings.Contains(output, "john") {
		t.Error("Username was incorrectly redacted")
	}

	// Verify redaction marker is present for sensitive keys
	if !strings.Contains(output, "[REDACTED]") {
		t.Error("Redaction marker not found in output")
	}
}

func TestRedactAttr_NilError(t *testing.T) {
	inner := &testHandler{}
	rh := logging.NewRedactingHandler(inner)

	// Test with nil error
	attr := slog.Any("error", error(nil))
	result := rh.RedactAttr(attr)

	// Should return original attribute for nil error
	if result.Key != "error" {
		t.Errorf("redactAttr with nil error changed key: got %q", result.Key)
	}
}

func TestRedactAttr_NumericValues(t *testing.T) {
	inner := &testHandler{}
	rh := logging.NewRedactingHandler(inner)

	tests := []struct {
		name string
		attr slog.Attr
	}{
		{"int", slog.Int("count", 42)},
		{"int64", slog.Int64("size", 1024)},
		{"float64", slog.Float64("rate", 3.14)},
		{"bool", slog.Bool("enabled", true)},
		{"duration", slog.Duration("elapsed", time.Second)},
		{"time", slog.Time("timestamp", time.Now())},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rh.RedactAttr(tt.attr)
			// Numeric and other non-string types should pass through unchanged
			if result.Key != tt.attr.Key {
				t.Errorf("redactAttr changed key: got %q, want %q", result.Key, tt.attr.Key)
			}
		})
	}
}

func TestRedactAttr_NestedAttrs(t *testing.T) {
	inner := &testHandler{}
	rh := logging.NewRedactingHandler(inner)

	// Test nested []slog.Attr (group)
	nestedAttrs := []slog.Attr{
		slog.String("username", "john"),
		slog.String("password", "secret123"),
		slog.Int("attempts", 3),
	}

	groupAttr := slog.Any("credentials", nestedAttrs)
	result := rh.RedactAttr(groupAttr)

	if result.Key != "credentials" {
		t.Errorf("Group key = %q, want credentials", result.Key)
	}

	// The nested password should be redacted
	// The result is a group, so we need to check its contents
}

func TestRedactingHandler_Handle_NestedGroup(t *testing.T) {
	inner := &testHandler{}
	rh := logging.NewRedactingHandler(inner)

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "test nested", 0)

	// Create nested attributes as []slog.Attr
	nestedAttrs := []slog.Attr{
		slog.String("user", "alice"),
		slog.String("token", "secret-token"),
	}
	record.AddAttrs(slog.Any("auth_data", nestedAttrs))

	err := rh.Handle(context.Background(), record)
	if err != nil {
		t.Errorf("Handle() error = %v", err)
	}

	if len(inner.records) == 0 {
		t.Fatal("Handle() did not pass record to inner handler")
	}
}

func TestRedactingHandler_Handle_MultipleWriters(t *testing.T) {
	// Test with multiple concurrent Handle calls
	inner := &testHandler{}
	rh := logging.NewRedactingHandler(inner)

	done := make(chan bool, 10)
	for i := range 10 {
		go func(idx int) {
			record := slog.NewRecord(time.Now(), slog.LevelInfo, "concurrent test", 0)
			record.AddAttrs(slog.String("password", "secret"))
			record.AddAttrs(slog.Int("idx", idx))
			_ = rh.Handle(context.Background(), record)
			done <- true
		}(i)
	}

	for range 10 {
		<-done
	}

	// All records should have been processed
	if len(inner.records) != 10 {
		t.Errorf("Expected 10 records, got %d", len(inner.records))
	}
}

func TestRedactingHandler_ChainedWithAttrs(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	rh := logging.NewRedactingHandler(baseHandler)

	// Chain WithAttrs multiple times
	handler := rh.WithAttrs([]slog.Attr{
		slog.String("service", "test"),
	}).WithAttrs([]slog.Attr{
		slog.String("version", "1.0"),
	})

	logger := slog.New(handler)
	logger.Info("chained test", "key", "value")

	output := buf.String()
	if !strings.Contains(output, "service") {
		t.Error("Output should contain service attribute")
	}
	if !strings.Contains(output, "version") {
		t.Error("Output should contain version attribute")
	}
}

func TestRedactingHandler_ChainedWithGroup(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	rh := logging.NewRedactingHandler(baseHandler)

	// Chain WithGroup
	handler := rh.WithGroup("request").WithGroup("details")

	logger := slog.New(handler)
	logger.Info("grouped test", "path", "/api/test")

	output := buf.String()
	if output == "" {
		t.Error("Output should not be empty")
	}
}

func TestRedactingHandler_SensitiveKeyVariations(t *testing.T) {
	tests := []struct {
		key          string
		shouldRedact bool
	}{
		// Direct matches
		{"password", true},
		{"token", true},
		{"secret", true},
		{"api_key", true},
		{"apikey", true},
		{"auth", true},
		{"authorization", true},
		{"bearer", true},
		{"credential", true},
		{"credentials", true},
		{"private_key", true},
		{"privatekey", true},
		{"jwt", true},
		{"session", true},
		{"cookie", true},
		{"passwd", true},
		{"pwd", true},

		// Case variations
		{"PASSWORD", true},
		{"Token", true},
		{"API_KEY", true},

		// Substring matches
		{"db_password", true},
		{"user_token", true},
		{"session_token", true},
		{"oauth_token", true},

		// Should NOT be redacted
		{"username", false},
		{"email", false},
		{"path", false},
		{"method", false},
		{"status", false},
		{"duration", false},
		{"id", false},
		{"name", false},
	}

	inner := &testHandler{}
	rh := logging.NewRedactingHandler(inner)

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			attr := slog.String(tt.key, "test-value")
			result := rh.RedactAttr(attr)

			isRedacted := result.Value.String() == "[REDACTED]"
			if isRedacted != tt.shouldRedact {
				t.Errorf("Key %q: redacted=%v, want %v", tt.key, isRedacted, tt.shouldRedact)
			}
		})
	}
}
