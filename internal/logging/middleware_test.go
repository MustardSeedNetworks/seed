package logging_test

import (
	"bufio"
	"bytes"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

func TestRequestIDMiddleware(t *testing.T) {
	t.Run("generates request ID when not provided", func(t *testing.T) {
		handler := logging.RequestIDMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request ID is in context
				requestID := logging.RequestIDFromContext(r.Context())
				if requestID == "" {
					t.Error("RequestIDMiddleware did not add request ID to context")
				}
				// Verify it's a valid hex string (16 chars)
				if len(requestID) != 16 {
					t.Errorf("Request ID has wrong length: got %d, want 16", len(requestID))
				}
				w.WriteHeader(http.StatusOK)
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// Verify response header is set
		responseID := rr.Header().Get(logging.RequestIDHeader)
		if responseID == "" {
			t.Error("RequestIDMiddleware did not set response header")
		}
	})

	t.Run("uses provided request ID from header", func(t *testing.T) {
		expectedID := "client-provided-id"
		var contextID string

		handler := logging.RequestIDMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contextID = logging.RequestIDFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		req.Header.Set(logging.RequestIDHeader, expectedID)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if contextID != expectedID {
			t.Errorf("Context request ID = %q, want %q", contextID, expectedID)
		}

		// Verify response header matches
		responseID := rr.Header().Get(logging.RequestIDHeader)
		if responseID != expectedID {
			t.Errorf("Response header request ID = %q, want %q", responseID, expectedID)
		}
	})

	t.Run("replaces invalid or oversized client IDs", func(t *testing.T) {
		// 70 chars plus invalid symbol should be rejected
		badID := strings.Repeat("a", 70) + "@"
		var contextID string

		handler := logging.RequestIDMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				contextID = logging.RequestIDFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		req.Header.Set(logging.RequestIDHeader, badID)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if contextID == badID {
			t.Errorf("Expected invalid request ID to be replaced, still got %q", contextID)
		}
		if rr.Header().Get(logging.RequestIDHeader) == badID {
			t.Errorf(
				"Response header should not echo invalid ID, got %q",
				rr.Header().Get(logging.RequestIDHeader),
			)
		}
	})

	t.Run("sets response header", func(t *testing.T) {
		handler := logging.RequestIDMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Header().Get(logging.RequestIDHeader) == "" {
			t.Error("Response missing X-Request-ID header")
		}
	})
}

func TestGenerateRequestID(t *testing.T) {
	t.Run("generates unique IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for range 100 {
			id := logging.ExportGenerateRequestID()
			if ids[id] {
				t.Errorf("generateRequestID() produced duplicate ID: %s", id)
			}
			ids[id] = true
		}
	})

	t.Run("generates correct length", func(t *testing.T) {
		for range 10 {
			id := logging.ExportGenerateRequestID()
			if len(id) != 16 {
				t.Errorf("generateRequestID() length = %d, want 16", len(id))
			}
		}
	})

	t.Run("generates valid hex", func(t *testing.T) {
		id := logging.ExportGenerateRequestID()
		for _, c := range id {
			isDigit := c >= '0' && c <= '9'
			isHexLetter := c >= 'a' && c <= 'f'
			if !isDigit && !isHexLetter {
				t.Errorf("generateRequestID() contains invalid hex char: %c", c)
			}
		}
	})
}

func TestLoggingMiddleware(t *testing.T) {
	// Initialize logger to capture output
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	logging.ExportSetGlobalLogger(slog.New(baseHandler))

	defer logging.ExportClearGlobalLogger()

	t.Run("logs request details", func(t *testing.T) {
		buf.Reset()

		handler := logging.LoggingMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("OK"))
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
		req.Header.Set("User-Agent", "test-agent")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		output := buf.String()

		// Verify log contains expected fields
		if !strings.Contains(output, "http request") {
			t.Error("Log missing 'http request' message")
		}
		if !strings.Contains(output, "GET") {
			t.Error("Log missing method")
		}
		if !strings.Contains(output, "/api/test") {
			t.Error("Log missing path")
		}
		if !strings.Contains(output, "200") {
			t.Error("Log missing status code")
		}
		if !strings.Contains(output, "duration_ms") {
			t.Error("Log missing duration")
		}
	})

	t.Run("skips health check endpoints", func(t *testing.T) {
		buf.Reset()

		handler := logging.LoggingMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		)

		// Test /api/health
		req := httptest.NewRequest(http.MethodGet, "/api/health", http.NoBody)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if strings.Contains(buf.String(), "http request") {
			t.Error("LoggingMiddleware logged /api/health")
		}

		buf.Reset()

		// Test /health
		req = httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
		rr = httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if strings.Contains(buf.String(), "http request") {
			t.Error("LoggingMiddleware logged /health")
		}
	})

	t.Run("captures correct status code", func(t *testing.T) {
		buf.Reset()

		handler := logging.LoggingMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/api/notfound", http.NoBody)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if !strings.Contains(buf.String(), "404") {
			t.Error("Log does not contain status code 404")
		}
	})

	t.Run("handles Write without WriteHeader", func(t *testing.T) {
		buf.Reset()

		handler := logging.LoggingMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				// Write without explicitly calling WriteHeader
				_, _ = w.Write([]byte("OK"))
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		// Should default to 200
		if !strings.Contains(buf.String(), "200") {
			t.Error("Log does not contain default status code 200")
		}
	})

	t.Run("uses request ID from context", func(t *testing.T) {
		buf.Reset()

		// Wrap with RequestIDMiddleware first
		handler := logging.RequestIDMiddleware(
			logging.LoggingMiddleware(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
			),
		)

		req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
		req.Header.Set(logging.RequestIDHeader, "test-request-123")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if !strings.Contains(buf.String(), "test-request-123") {
			t.Error("Log does not contain request ID")
		}
	})
}

func TestResponseWriter(t *testing.T) {
	t.Run("captures status code from WriteHeader", func(t *testing.T) {
		rr := httptest.NewRecorder()
		wrapped := logging.NewTestResponseWriter(rr, http.StatusOK)

		wrapped.WriteHeader(http.StatusNotFound)

		if wrapped.Status() != http.StatusNotFound {
			t.Errorf("status = %d, want %d", wrapped.Status(), http.StatusNotFound)
		}
		if !wrapped.WroteHeader() {
			t.Error("wroteHeader should be true after WriteHeader")
		}
	})

	t.Run("ignores subsequent WriteHeader calls", func(t *testing.T) {
		rr := httptest.NewRecorder()
		wrapped := logging.NewTestResponseWriter(rr, http.StatusOK)

		wrapped.WriteHeader(http.StatusNotFound)
		wrapped.WriteHeader(http.StatusInternalServerError) // Should be ignored

		if wrapped.Status() != http.StatusNotFound {
			t.Errorf("status = %d, want %d (first call)", wrapped.Status(), http.StatusNotFound)
		}
	})

	t.Run("Write sets default status", func(t *testing.T) {
		rr := httptest.NewRecorder()
		wrapped := logging.NewTestResponseWriter(rr, http.StatusOK)

		_, err := wrapped.Write([]byte("test"))
		if err != nil {
			t.Fatalf("Write() error: %v", err)
		}

		if !wrapped.WroteHeader() {
			t.Error("wroteHeader should be true after Write")
		}
		if wrapped.Status() != http.StatusOK {
			t.Errorf("status = %d, want %d", wrapped.Status(), http.StatusOK)
		}
	})

	t.Run("Write returns correct byte count", func(t *testing.T) {
		rr := httptest.NewRecorder()
		wrapped := logging.NewTestResponseWriter(rr, http.StatusOK)

		data := []byte("hello world")
		n, err := wrapped.Write(data)
		if err != nil {
			t.Fatalf("Write() error: %v", err)
		}
		if n != len(data) {
			t.Errorf("Write() returned %d bytes, want %d", n, len(data))
		}
	})

	t.Run("Unwrap returns underlying ResponseWriter", func(t *testing.T) {
		rr := httptest.NewRecorder()
		wrapped := logging.NewTestResponseWriter(rr, http.StatusOK)

		unwrapped := wrapped.Unwrap()
		if unwrapped != rr {
			t.Error("Unwrap() did not return the underlying ResponseWriter")
		}
	})
}

func TestLoggingMiddleware_ClientIP(t *testing.T) {
	// Initialize logger to capture output
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	logging.ExportSetGlobalLogger(slog.New(baseHandler))

	defer logging.ExportClearGlobalLogger()

	t.Run("logs X-Forwarded-For IP", func(t *testing.T) {
		buf.Reset()

		handler := logging.LoggingMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
		req.Header.Set("X-Forwarded-For", "203.0.113.195, 70.41.3.18, 150.172.238.178")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		output := buf.String()
		// Should log the first IP from X-Forwarded-For
		if !strings.Contains(output, "203.0.113.195") {
			t.Error("Log does not contain forwarded IP")
		}
	})

	t.Run("logs X-Real-IP", func(t *testing.T) {
		buf.Reset()

		handler := logging.LoggingMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/api/test", http.NoBody)
		req.Header.Set("X-Real-IP", "192.168.1.100")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		output := buf.String()
		if !strings.Contains(output, "192.168.1.100") {
			t.Error("Log does not contain X-Real-IP")
		}
	})
}

func TestMiddlewareChain(t *testing.T) {
	// Test that middlewares work together correctly
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	logging.ExportSetGlobalLogger(slog.New(baseHandler))

	defer logging.ExportClearGlobalLogger()

	// Chain: RequestID -> Logging -> Handler
	handler := logging.RequestIDMiddleware(
		logging.LoggingMiddleware(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify context has request ID
				requestID := logging.RequestIDFromContext(r.Context())
				if requestID == "" {
					t.Error("Handler did not receive request ID in context")
				}
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, "OK")
			}),
		),
	)

	req := httptest.NewRequest(http.MethodPost, "/api/data", http.NoBody)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Verify response
	if rr.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusOK)
	}

	// Verify request ID in response header
	if rr.Header().Get(logging.RequestIDHeader) == "" {
		t.Error("Response missing X-Request-ID header")
	}

	// Verify log contains request ID
	output := buf.String()
	if !strings.Contains(output, "request_id") {
		t.Error("Log does not contain request_id field")
	}
}

func TestRequestIDHeader_Constant(t *testing.T) {
	if logging.RequestIDHeader != "X-Request-ID" {
		t.Errorf("RequestIDHeader = %q, want %q", logging.RequestIDHeader, "X-Request-ID")
	}
}

func TestIsValidRequestID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected bool
	}{
		{"empty string", "", false},
		{"valid alphanumeric", "abc123XYZ", true},
		{"valid with dash", "req-123-abc", true},
		{"valid with underscore", "req_123_abc", true},
		{"valid with dot", "req.123.abc", true},
		{"valid UUID", "550e8400-e29b-41d4-a716-446655440000", true},
		{"invalid with space", "abc 123", false},
		{"invalid with special char", "abc@123", false},
		{"invalid with newline", "abc\n123", false},
		{"invalid with colon", "abc:123", false},
		{"too long", strings.Repeat("a", 65), false},
		{"at max length", strings.Repeat("a", 64), true},
		{"just under max", strings.Repeat("a", 63), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logging.ExportIsValidRequestID(tt.id)
			if result != tt.expected {
				t.Errorf("isValidRequestID(%q) = %v, want %v", tt.id, result, tt.expected)
			}
		})
	}
}

func TestResponseWriter_Hijack(t *testing.T) {
	t.Run("returns error when underlying writer does not support hijack", func(t *testing.T) {
		// httptest.ResponseRecorder does not implement http.Hijacker
		rr := httptest.NewRecorder()
		wrapped := logging.NewTestHijackableResponseWriter(rr)

		conn, rw, err := wrapped.Hijack()
		if err == nil {
			t.Error("Hijack() should return error for non-hijackable writer")
		}
		if conn != nil {
			t.Error("Hijack() should return nil conn for non-hijackable writer")
		}
		if rw != nil {
			t.Error("Hijack() should return nil rw for non-hijackable writer")
		}
		if !strings.Contains(err.Error(), "http.Hijacker") {
			t.Errorf("Error message = %q, should mention Hijacker", err.Error())
		}
	})

	t.Run("succeeds when underlying writer supports hijack", func(t *testing.T) {
		// Use a mock hijackable writer
		mock := &mockHijackableWriter{}
		wrapped := logging.NewTestHijackableResponseWriter(mock)

		conn, rw, err := wrapped.Hijack()
		if err != nil {
			t.Errorf("Hijack() error = %v, want nil", err)
		}
		// Mock returns non-nil connection and readwriter
		if conn == nil {
			t.Error("Hijack() should return non-nil connection from underlying writer")
		}
		if rw == nil {
			t.Error("Hijack() should return non-nil readwriter from underlying writer")
		}
	})
}

// mockHijackableWriter implements [http.ResponseWriter] and [http.Hijacker].
type mockHijackableWriter struct {
	headers http.Header
}

func (m *mockHijackableWriter) Header() http.Header {
	if m.headers == nil {
		m.headers = make(http.Header)
	}
	return m.headers
}

func (m *mockHijackableWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

func (m *mockHijackableWriter) WriteHeader(_ int) {}

func (m *mockHijackableWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	// Return mock connection and bufio.ReadWriter
	mockConn := &mockNetConn{}
	mockRW := bufio.NewReadWriter(
		bufio.NewReader(strings.NewReader("")),
		bufio.NewWriter(io.Discard),
	)
	return mockConn, mockRW, nil
}

// mockNetConn implements [net.Conn] for testing.
type mockNetConn struct{}

func (m *mockNetConn) Read(_ []byte) (int, error)         { return 0, io.EOF }
func (m *mockNetConn) Write(_ []byte) (int, error)        { return 0, nil }
func (m *mockNetConn) Close() error                       { return nil }
func (m *mockNetConn) LocalAddr() net.Addr                { return nil }
func (m *mockNetConn) RemoteAddr() net.Addr               { return nil }
func (m *mockNetConn) SetDeadline(_ time.Time) error      { return nil }
func (m *mockNetConn) SetReadDeadline(_ time.Time) error  { return nil }
func (m *mockNetConn) SetWriteDeadline(_ time.Time) error { return nil }

func TestRequestIDMiddleware_EdgeCases(t *testing.T) {
	t.Run("handles special characters in invalid ID", func(t *testing.T) {
		invalidIDs := []string{
			"abc\x00def",             // null byte
			"abc\rdef",               // carriage return
			"<script>alert</script>", // HTML
			"'; DROP TABLE--",        // SQL injection attempt
		}

		for _, badID := range invalidIDs {
			var contextID string
			handler := logging.RequestIDMiddleware(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					contextID = logging.RequestIDFromContext(r.Context())
					w.WriteHeader(http.StatusOK)
				}),
			)

			req := httptest.NewRequest(http.MethodGet, "/test", http.NoBody)
			req.Header.Set(logging.RequestIDHeader, badID)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if contextID == badID {
				t.Errorf("Invalid request ID %q should be replaced", badID)
			}
		}
	})
}

func TestLoggingMiddleware_DifferentMethods(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logging.ExportSetGlobalLogger(slog.New(baseHandler))
	defer logging.ExportClearGlobalLogger()

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			buf.Reset()

			handler := logging.LoggingMiddleware(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				}),
			)

			req := httptest.NewRequest(method, "/api/test", http.NoBody)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			output := buf.String()
			if !strings.Contains(output, method) {
				t.Errorf("Log should contain method %q", method)
			}
		})
	}
}
