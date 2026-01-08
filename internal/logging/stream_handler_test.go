package logging_test

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

func TestNewStreamingHandler(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	broadcaster := logging.NewLogBroadcaster(100)

	sh := logging.NewStreamingHandler(baseHandler, broadcaster)
	if sh == nil {
		t.Fatal("NewStreamingHandler() returned nil")
	}
}

func TestStreamingHandler_Enabled(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	broadcaster := logging.NewLogBroadcaster(100)

	sh := logging.NewStreamingHandler(baseHandler, broadcaster)

	if !sh.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("Enabled(LevelInfo) = false, want true")
	}
	if !sh.Enabled(context.Background(), slog.LevelError) {
		t.Error("Enabled(LevelError) = false, want true")
	}
	if sh.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Enabled(LevelDebug) = true, want false")
	}
}

func TestStreamingHandler_Handle_Broadcast(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)
	logger := slog.New(sh)

	logger.Info("test message", "key", "value")

	entries := broadcaster.GetAllLogs()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 broadcast entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Message != "test message" {
		t.Errorf("Message = %q, want test message", entry.Message)
	}
	if entry.Level != "INFO" {
		t.Errorf("Level = %q, want INFO", entry.Level)
	}
}

func TestStreamingHandler_Handle_RequestID(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)
	logger := slog.New(sh)

	ctx := logging.WithRequestID(context.Background(), "req-test-123")
	logger.InfoContext(ctx, "test with request ID")

	entries := broadcaster.GetAllLogs()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 broadcast entry, got %d", len(entries))
	}

	if entries[0].RequestID != "req-test-123" {
		t.Errorf("RequestID = %q, want req-test-123", entries[0].RequestID)
	}
}

func TestStreamingHandler_Handle_Layer(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)
	logger := slog.New(sh)

	ctx := logging.WithLayer(context.Background(), logging.LayerAPI)
	logger.InfoContext(ctx, "test with layer")

	entries := broadcaster.GetAllLogs()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 broadcast entry, got %d", len(entries))
	}

	if entries[0].Layer != logging.LayerAPI {
		t.Errorf("Layer = %q, want %q", entries[0].Layer, logging.LayerAPI)
	}
}

func TestStreamingHandler_Handle_Component(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)
	logger := slog.New(sh)

	ctx := logging.WithComponent(context.Background(), logging.ComponentAuth)
	logger.InfoContext(ctx, "test with component")

	entries := broadcaster.GetAllLogs()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 broadcast entry, got %d", len(entries))
	}

	if entries[0].Component != logging.ComponentAuth {
		t.Errorf("Component = %q, want %q", entries[0].Component, logging.ComponentAuth)
	}
}

func TestStreamingHandler_Handle_NilBroadcaster(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	sh := logging.NewStreamingHandler(baseHandler, nil)
	logger := slog.New(sh)

	// Should not panic
	logger.Info("test without broadcaster")

	// Output should still be written to base handler
	if buf.Len() == 0 {
		t.Error("Base handler should have received log")
	}
}

func TestStreamingHandler_Handle_DifferentLevels(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)
	logger := slog.New(sh)

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	entries := broadcaster.GetAllLogs()
	if len(entries) != 4 {
		t.Fatalf("Expected 4 broadcast entries, got %d", len(entries))
	}

	expectedLevels := []string{"DEBUG", "INFO", "WARN", "ERROR"}
	for i, entry := range entries {
		if entry.Level != expectedLevels[i] {
			t.Errorf("entries[%d].Level = %q, want %q", i, entry.Level, expectedLevels[i])
		}
	}
}

func TestStreamingHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)

	// Add attributes to handler
	newHandler := sh.WithAttrs([]slog.Attr{
		slog.String("service", "test-service"),
		slog.Int("version", 1),
	})

	if newHandler == nil {
		t.Fatal("WithAttrs() returned nil")
	}

	// Log with the new handler
	logger := slog.New(newHandler)
	logger.Info("test message")

	entries := broadcaster.GetAllLogs()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 broadcast entry, got %d", len(entries))
	}

	// Attributes should be in metadata
	if entries[0].Metadata["service"] != "test-service" {
		t.Errorf("Metadata[service] = %v, want test-service", entries[0].Metadata["service"])
	}
}

func TestStreamingHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)

	newHandler := sh.WithGroup("request")
	if newHandler == nil {
		t.Fatal("WithGroup() returned nil")
	}

	// Log with the new handler
	logger := slog.New(newHandler)
	logger.Info("test message", "path", "/api/test")

	// Should not panic and should log
	if buf.Len() == 0 {
		t.Error("Base handler should have received log")
	}
}

func TestStreamingHandler_DurationMsInt64(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)
	logger := slog.New(sh)

	logger.Info("request completed", "duration_ms", int64(1500))

	entries := broadcaster.GetAllLogs()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].DurationMs != 1500 {
		t.Errorf("DurationMs = %d, want 1500", entries[0].DurationMs)
	}
}

func TestStreamingHandler_DurationMsInt(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)
	logger := slog.New(sh)

	logger.Info("request completed", "duration_ms", 250)

	entries := broadcaster.GetAllLogs()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].DurationMs != 250 {
		t.Errorf("DurationMs = %d, want 250", entries[0].DurationMs)
	}
}

func TestStreamingHandler_StackAttribute(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)
	logger := slog.New(sh)

	logger.Error("panic occurred", "stack", "goroutine 1 [running]:\nmain.main()")

	entries := broadcaster.GetAllLogs()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].Stack != "goroutine 1 [running]:\nmain.main()" {
		t.Errorf("Stack = %q, want goroutine stack", entries[0].Stack)
	}
}

func TestStreamingHandler_ErrorAttribute(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)
	logger := slog.New(sh)

	testErr := &testStreamError{msg: "connection refused"}
	logger.Error("operation failed", "error", testErr)

	entries := broadcaster.GetAllLogs()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].Metadata["error"] != "connection refused" {
		t.Errorf("Metadata[error] = %v, want connection refused", entries[0].Metadata["error"])
	}
}

func TestStreamingHandler_ComponentAttrOverride(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)
	logger := slog.New(sh)

	logger.Info("test", "component", logging.ComponentNetwork)

	entries := broadcaster.GetAllLogs()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].Component != logging.ComponentNetwork {
		t.Errorf("Component = %q, want %q", entries[0].Component, logging.ComponentNetwork)
	}
}

func TestStreamingHandler_RequestIDAttrOverride(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)
	logger := slog.New(sh)

	logger.Info("test", "request_id", "attr-req-id")

	entries := broadcaster.GetAllLogs()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].RequestID != "attr-req-id" {
		t.Errorf("RequestID = %q, want attr-req-id", entries[0].RequestID)
	}
}

func TestStreamingHandler_LayerAttr(t *testing.T) {
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, nil)
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)
	logger := slog.New(sh)

	logger.Info("test", "layer", logging.LayerFrontend)

	entries := broadcaster.GetAllLogs()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}

	if entries[0].Layer != logging.LayerFrontend {
		t.Errorf("Layer = %q, want %q", entries[0].Layer, logging.LayerFrontend)
	}
}

func TestInitLoggerWithBroadcaster(t *testing.T) {
	defer logging.ExportClearGlobalLogger()

	t.Run("with nil config uses defaults", func(t *testing.T) {
		broadcaster := logging.NewLogBroadcaster(100)
		err := logging.InitLoggerWithBroadcaster(nil, broadcaster)
		if err != nil {
			t.Errorf("InitLoggerWithBroadcaster() error = %v", err)
		}
	})

	t.Run("with nil broadcaster creates one", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := &logging.LoggingConfig{
			Level:  "info",
			Format: "text",
			Writer: &buf,
		}
		err := logging.InitLoggerWithBroadcaster(cfg, nil)
		if err != nil {
			t.Errorf("InitLoggerWithBroadcaster() error = %v", err)
		}

		// Verify logging works
		logging.Info("test message")
		if buf.Len() == 0 {
			t.Error("Logger should have written to buffer")
		}
	})

	t.Run("broadcasts to provided broadcaster", func(t *testing.T) {
		var buf bytes.Buffer
		cfg := &logging.LoggingConfig{
			Level:  "debug",
			Format: "text",
			Writer: &buf,
		}
		broadcaster := logging.NewLogBroadcaster(100)
		err := logging.InitLoggerWithBroadcaster(cfg, broadcaster)
		if err != nil {
			t.Errorf("InitLoggerWithBroadcaster() error = %v", err)
		}

		logging.Info("test broadcast message")

		entries := broadcaster.GetAllLogs()
		if len(entries) != 1 {
			t.Errorf("Expected 1 broadcast entry, got %d", len(entries))
		}
	})
}

func TestLogWithContext(t *testing.T) {
	defer logging.ExportClearGlobalLogger()

	var buf bytes.Buffer
	cfg := &logging.LoggingConfig{
		Level:  "debug",
		Format: "text",
		Writer: &buf,
	}
	broadcaster := logging.NewLogBroadcaster(100)
	_ = logging.InitLoggerWithBroadcaster(cfg, broadcaster)

	ctx := context.Background()
	ctx = logging.WithLayer(ctx, logging.LayerAPI)
	ctx = logging.WithComponent(ctx, logging.ComponentAuth)
	ctx = logging.WithRequestID(ctx, "req-test-456")

	logging.LogWithContext(ctx, slog.LevelInfo, "context log test", "extra", "data")

	output := buf.String()
	if output == "" {
		t.Error("LogWithContext should have written to buffer")
	}
}

func TestTimedOperation(t *testing.T) {
	defer logging.ExportClearGlobalLogger()

	var buf bytes.Buffer
	cfg := &logging.LoggingConfig{
		Level:  "debug",
		Format: "text",
		Writer: &buf,
	}
	_ = logging.InitLogger(cfg)

	ctx := logging.WithRequestID(context.Background(), "req-timed-123")
	done := logging.TimedOperation(ctx, "database query", logging.ComponentSystem)

	// Simulate some work
	time.Sleep(10 * time.Millisecond)

	done()

	output := buf.String()
	if !testContainsSubstring(output, "started") {
		t.Error("TimedOperation should log 'started' message")
	}
	if !testContainsSubstring(output, "completed") {
		t.Error("TimedOperation should log 'completed' message")
	}
	if !testContainsSubstring(output, "duration_ms") {
		t.Error("TimedOperation should include duration_ms")
	}
}

func TestLevelToString(t *testing.T) {
	// Test via StreamingHandler
	var buf bytes.Buffer
	baseHandler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	broadcaster := logging.NewLogBroadcaster(100)
	sh := logging.NewStreamingHandler(baseHandler, broadcaster)
	logger := slog.New(sh)

	tests := []struct {
		logFunc     func(string, ...any)
		expectedLvl string
	}{
		{logger.Debug, "DEBUG"},
		{logger.Info, "INFO"},
		{logger.Warn, "WARN"},
		{logger.Error, "ERROR"},
	}

	for _, tt := range tests {
		buf.Reset()
		tt.logFunc("test message")
		entries := broadcaster.GetRecentLogs(1)
		if len(entries) == 0 {
			t.Fatalf("No entries for level %s", tt.expectedLvl)
		}
		if entries[0].Level != tt.expectedLvl {
			t.Errorf("Level = %q, want %q", entries[0].Level, tt.expectedLvl)
		}
	}
}

// Helper types

type testStreamError struct {
	msg string
}

func (e *testStreamError) Error() string {
	return e.msg
}

func testContainsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
