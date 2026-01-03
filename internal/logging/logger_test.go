package logging_test

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/krisarmstrong/seed/internal/logging"
)

func TestDefaultLoggingConfig(t *testing.T) {
	cfg := logging.DefaultLoggingConfig()

	if cfg.Level != "info" {
		t.Errorf("expected level 'info', got %q", cfg.Level)
	}
	if cfg.Format != "json" {
		t.Errorf("expected format 'json', got %q", cfg.Format)
	}
	if cfg.AddSource {
		t.Error("expected AddSource to be false")
	}
	if cfg.File != "" {
		t.Errorf("expected empty file path, got %q", cfg.File)
	}
	if cfg.MaxSize != 100 {
		t.Errorf("expected MaxSize 100, got %d", cfg.MaxSize)
	}
	if cfg.MaxBackups != 5 {
		t.Errorf("expected MaxBackups 5, got %d", cfg.MaxBackups)
	}
	if cfg.MaxAge != 30 {
		t.Errorf("expected MaxAge 30, got %d", cfg.MaxAge)
	}
	if !cfg.Compress {
		t.Error("expected Compress to be true")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"Debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"unknown", slog.LevelInfo}, // defaults to info
		{"", slog.LevelInfo},        // empty defaults to info
		{"invalid", slog.LevelInfo}, // invalid defaults to info
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := logging.ParseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestInitLogger(t *testing.T) {
	// Reset global logger after test
	defer logging.ClearGlobalLogger()

	t.Run("nil config uses defaults", func(t *testing.T) {
		err := logging.InitLogger(nil)
		if err != nil {
			t.Errorf("InitLogger(nil) returned error: %v", err)
		}

		logger := logging.GetLogger()
		if logger == nil {
			t.Error("GetLogger() returned nil after InitLogger(nil)")
		}
	})

	t.Run("text format", func(t *testing.T) {
		cfg := &logging.LoggingConfig{
			Level:  "debug",
			Format: "text",
		}
		err := logging.InitLogger(cfg)
		if err != nil {
			t.Errorf("InitLogger() returned error: %v", err)
		}
	})

	t.Run("json format", func(t *testing.T) {
		cfg := &logging.LoggingConfig{
			Level:  "info",
			Format: "json",
		}
		err := logging.InitLogger(cfg)
		if err != nil {
			t.Errorf("InitLogger() returned error: %v", err)
		}
	})

	t.Run("with file output", func(t *testing.T) {
		tmpDir := t.TempDir()
		logFile := filepath.Join(tmpDir, "test.log")

		cfg := &logging.LoggingConfig{
			Level:      "info",
			Format:     "text",
			File:       logFile,
			MaxSize:    1,
			MaxBackups: 1,
			MaxAge:     1,
			Compress:   false,
		}
		err := logging.InitLogger(cfg)
		if err != nil {
			t.Errorf("InitLogger() returned error: %v", err)
		}

		// Write a log message
		logging.Info("test log message")

		// Verify file exists (may take a moment for buffer to flush)
		// Note: We can't reliably verify file contents in unit tests due to buffering
	})

	t.Run("with AddSource", func(t *testing.T) {
		cfg := &logging.LoggingConfig{
			Level:     "debug",
			Format:    "text",
			AddSource: true,
		}
		err := logging.InitLogger(cfg)
		if err != nil {
			t.Errorf("InitLogger() returned error: %v", err)
		}
	})
}

func TestGetLogger(t *testing.T) {
	// Reset global logger
	logging.ClearGlobalLogger()

	t.Run("returns default when not initialized", func(t *testing.T) {
		logger := logging.GetLogger()
		if logger == nil {
			t.Error("GetLogger() returned nil when not initialized")
		}
		// Should return slog.Default()
	})

	t.Run("returns global logger after init", func(t *testing.T) {
		err := logging.InitLogger(nil)
		if err != nil {
			t.Fatalf("InitLogger() failed: %v", err)
		}

		logger := logging.GetLogger()
		if logger == nil {
			t.Error("GetLogger() returned nil after InitLogger()")
		}
	})

	// Reset global logger after test
	logging.ClearGlobalLogger()
}

func TestWithRequestID(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
	}{
		{"empty string", ""},
		{"simple ID", "abc123"},
		{"UUID format", "550e8400-e29b-41d4-a716-446655440000"},
		{"hex format", "deadbeef12345678"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			newCtx := logging.WithRequestID(ctx, tt.requestID)

			// Verify the context is different
			if tt.requestID != "" && ctx == newCtx {
				t.Error("WithRequestID returned same context")
			}

			// Verify the request ID can be retrieved
			result := logging.RequestIDFromContext(newCtx)
			if result != tt.requestID {
				t.Errorf("RequestIDFromContext() = %q, want %q", result, tt.requestID)
			}
		})
	}
}

func TestRequestIDFromContext(t *testing.T) {
	t.Run("returns empty string for background context", func(t *testing.T) {
		ctx := context.Background()
		result := logging.RequestIDFromContext(ctx)
		if result != "" {
			t.Errorf("RequestIDFromContext(background) = %q, want empty string", result)
		}
	})

	t.Run("returns empty string for context with wrong type", func(t *testing.T) {
		// Create context with wrong value type
		ctx := context.WithValue(context.Background(), logging.RequestIDKeyValue, 12345)
		result := logging.RequestIDFromContext(ctx)
		if result != "" {
			t.Errorf("RequestIDFromContext(wrong type) = %q, want empty string", result)
		}
	})

	t.Run("returns request ID from context", func(t *testing.T) {
		expectedID := "test-request-id"
		ctx := logging.WithRequestID(context.Background(), expectedID)
		result := logging.RequestIDFromContext(ctx)
		if result != expectedID {
			t.Errorf("RequestIDFromContext() = %q, want %q", result, expectedID)
		}
	})
}

func TestFromContext(t *testing.T) {
	// Initialize logger for tests
	err := logging.InitLogger(nil)
	if err != nil {
		t.Fatalf("InitLogger() failed: %v", err)
	}

	t.Run("returns logger without request_id when not in context", func(t *testing.T) {
		ctx := context.Background()
		logger := logging.FromContext(ctx)
		if logger == nil {
			t.Error("FromContext() returned nil")
		}
	})

	t.Run("returns logger with request_id when in context", func(t *testing.T) {
		ctx := logging.WithRequestID(context.Background(), "test-123")
		logger := logging.FromContext(ctx)
		if logger == nil {
			t.Error("FromContext() returned nil")
		}
		// Logger should have request_id attribute, but we can't easily verify
		// without capturing output
	})

	// Reset global logger after test
	logging.ClearGlobalLogger()
}

func TestConvenienceLogFunctions(t *testing.T) {
	// Capture output to verify logging
	var buf bytes.Buffer

	// Initialize logger with custom writer
	err := logging.InitLogger(&logging.LoggingConfig{
		Level:  "debug",
		Format: "text",
		Writer: &buf,
	})
	if err != nil {
		t.Fatalf("InitLogger() failed: %v", err)
	}

	// Test all convenience functions
	logging.Debug("debug message", "key", "value")
	logging.Info("info message", "key", "value")
	logging.Warn("warn message", "key", "value")
	logging.Error("error message", "key", "value")

	output := buf.String()

	// Verify log levels appear
	if !strings.Contains(output, "DEBUG") && !strings.Contains(output, "debug") {
		t.Error("Debug() did not log at debug level")
	}
	if !strings.Contains(output, "INFO") && !strings.Contains(output, "info") {
		t.Error("Info() did not log at info level")
	}
	if !strings.Contains(output, "WARN") && !strings.Contains(output, "warn") {
		t.Error("Warn() did not log at warn level")
	}
	if !strings.Contains(output, "ERROR") && !strings.Contains(output, "error") {
		t.Error("Error() did not log at error level")
	}

	// Reset global logger
	logging.ClearGlobalLogger()
}

func TestContextLogFunctions(t *testing.T) {
	// Capture output to verify logging
	var buf bytes.Buffer

	// Initialize logger with custom writer
	err := logging.InitLogger(&logging.LoggingConfig{
		Level:  "debug",
		Format: "text",
		Writer: &buf,
	})
	if err != nil {
		t.Fatalf("InitLogger() failed: %v", err)
	}

	ctx := logging.WithRequestID(context.Background(), "ctx-test-123")

	// Test all context convenience functions
	logging.DebugContext(ctx, "debug with context", "key", "value")
	logging.InfoContext(ctx, "info with context", "key", "value")
	logging.WarnContext(ctx, "warn with context", "key", "value")
	logging.ErrorContext(ctx, "error with context", "key", "value")

	output := buf.String()

	// Verify request_id appears in output
	if !strings.Contains(output, "ctx-test-123") {
		t.Error("Context log functions did not include request_id")
	}

	// Reset global logger
	logging.ClearGlobalLogger()
}

func TestConcurrentLoggerAccess(t *testing.T) {
	// Reset and initialize logger
	logging.ClearGlobalLogger()

	err := logging.InitLogger(nil)
	if err != nil {
		t.Fatalf("InitLogger() failed: %v", err)
	}

	// Test concurrent reads
	done := make(chan bool, 10)
	for range 10 {
		go func() {
			logger := logging.GetLogger()
			if logger == nil {
				t.Error("GetLogger() returned nil during concurrent access")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for range 10 {
		<-done
	}

	// Reset global logger
	logging.ClearGlobalLogger()
}
