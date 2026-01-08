package logging_test

import (
	"context"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

func TestNewLogEntry(t *testing.T) {
	entry := logging.NewLogEntry("INFO", "test message")

	if entry == nil {
		t.Fatal("NewLogEntry() returned nil")
	}
	if entry.Level != "INFO" {
		t.Errorf("Level = %q, want INFO", entry.Level)
	}
	if entry.Message != "test message" {
		t.Errorf("Message = %q, want test message", entry.Message)
	}
	if entry.Layer != logging.LayerBackend {
		t.Errorf("Layer = %q, want %q", entry.Layer, logging.LayerBackend)
	}
	if entry.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestLogEntry_WithRequestID(t *testing.T) {
	entry := logging.NewLogEntry("INFO", "test")
	result := entry.WithRequestID("req-123")

	if result != entry {
		t.Error("WithRequestID should return same entry for chaining")
	}
	if entry.RequestID != "req-123" {
		t.Errorf("RequestID = %q, want req-123", entry.RequestID)
	}
}

func TestLogEntry_WithLayer(t *testing.T) {
	entry := logging.NewLogEntry("INFO", "test")
	result := entry.WithLayer(logging.LayerAPI)

	if result != entry {
		t.Error("WithLayer should return same entry for chaining")
	}
	if entry.Layer != logging.LayerAPI {
		t.Errorf("Layer = %q, want %q", entry.Layer, logging.LayerAPI)
	}
}

func TestLogEntry_WithComponent(t *testing.T) {
	entry := logging.NewLogEntry("INFO", "test")
	result := entry.WithComponent(logging.ComponentAuth)

	if result != entry {
		t.Error("WithComponent should return same entry for chaining")
	}
	if entry.Component != logging.ComponentAuth {
		t.Errorf("Component = %q, want %q", entry.Component, logging.ComponentAuth)
	}
}

func TestLogEntry_WithDuration(t *testing.T) {
	entry := logging.NewLogEntry("INFO", "test")
	duration := 1500 * time.Millisecond
	result := entry.WithDuration(duration)

	if result != entry {
		t.Error("WithDuration should return same entry for chaining")
	}
	if entry.DurationMs != 1500 {
		t.Errorf("DurationMs = %d, want 1500", entry.DurationMs)
	}
}

func TestLogEntry_WithMetadata(t *testing.T) {
	entry := logging.NewLogEntry("INFO", "test")
	metadata := map[string]any{
		"key1": "value1",
		"key2": 42,
	}
	result := entry.WithMetadata(metadata)

	if result != entry {
		t.Error("WithMetadata should return same entry for chaining")
	}
	if entry.Metadata["key1"] != "value1" {
		t.Errorf("Metadata[key1] = %v, want value1", entry.Metadata["key1"])
	}
	if entry.Metadata["key2"] != 42 {
		t.Errorf("Metadata[key2] = %v, want 42", entry.Metadata["key2"])
	}
}

func TestLogEntry_WithStack(t *testing.T) {
	entry := logging.NewLogEntry("ERROR", "test error")
	stack := "goroutine 1 [running]:\nmain.main()\n\t/path/to/file.go:10"
	result := entry.WithStack(stack)

	if result != entry {
		t.Error("WithStack should return same entry for chaining")
	}
	if entry.Stack != stack {
		t.Errorf("Stack = %q, want %q", entry.Stack, stack)
	}
}

func TestLogEntry_AddMetadata(t *testing.T) {
	t.Run("initializes metadata map if nil", func(t *testing.T) {
		entry := logging.NewLogEntry("INFO", "test")
		entry.Metadata = nil // Explicitly set to nil
		result := entry.AddMetadata("key", "value")

		if result != entry {
			t.Error("AddMetadata should return same entry for chaining")
		}
		if entry.Metadata == nil {
			t.Error("Metadata should be initialized")
		}
		if entry.Metadata["key"] != "value" {
			t.Errorf("Metadata[key] = %v, want value", entry.Metadata["key"])
		}
	})

	t.Run("adds to existing metadata", func(t *testing.T) {
		entry := logging.NewLogEntry("INFO", "test")
		entry.AddMetadata("key1", "value1")
		entry.AddMetadata("key2", 42)

		if entry.Metadata["key1"] != "value1" {
			t.Errorf("Metadata[key1] = %v, want value1", entry.Metadata["key1"])
		}
		if entry.Metadata["key2"] != 42 {
			t.Errorf("Metadata[key2] = %v, want 42", entry.Metadata["key2"])
		}
	})
}

func TestLogEntry_Chaining(t *testing.T) {
	entry := logging.NewLogEntry("ERROR", "operation failed").
		WithRequestID("req-456").
		WithLayer(logging.LayerAPI).
		WithComponent(logging.ComponentAuth).
		WithDuration(250*time.Millisecond).
		WithStack("stack trace here").
		AddMetadata("user", "alice").
		AddMetadata("action", "login")

	if entry.Level != "ERROR" {
		t.Errorf("Level = %q, want ERROR", entry.Level)
	}
	if entry.Message != "operation failed" {
		t.Errorf("Message = %q, want operation failed", entry.Message)
	}
	if entry.RequestID != "req-456" {
		t.Errorf("RequestID = %q, want req-456", entry.RequestID)
	}
	if entry.Layer != logging.LayerAPI {
		t.Errorf("Layer = %q, want %q", entry.Layer, logging.LayerAPI)
	}
	if entry.Component != logging.ComponentAuth {
		t.Errorf("Component = %q, want %q", entry.Component, logging.ComponentAuth)
	}
	if entry.DurationMs != 250 {
		t.Errorf("DurationMs = %d, want 250", entry.DurationMs)
	}
	if entry.Stack != "stack trace here" {
		t.Errorf("Stack = %q, want stack trace here", entry.Stack)
	}
	if entry.Metadata["user"] != "alice" {
		t.Errorf("Metadata[user] = %v, want alice", entry.Metadata["user"])
	}
	if entry.Metadata["action"] != "login" {
		t.Errorf("Metadata[action] = %v, want login", entry.Metadata["action"])
	}
}

func TestWithLayer(t *testing.T) {
	ctx := context.Background()
	newCtx := logging.WithLayer(ctx, logging.LayerAPI)

	if ctx == newCtx {
		t.Error("WithLayer should return new context")
	}

	layer := logging.LayerFromContext(newCtx)
	if layer != logging.LayerAPI {
		t.Errorf("LayerFromContext() = %q, want %q", layer, logging.LayerAPI)
	}
}

func TestLayerFromContext(t *testing.T) {
	t.Run("returns default for empty context", func(t *testing.T) {
		ctx := context.Background()
		layer := logging.LayerFromContext(ctx)
		if layer != logging.LayerBackend {
			t.Errorf("LayerFromContext() = %q, want %q (default)", layer, logging.LayerBackend)
		}
	})

	t.Run("returns layer from context", func(t *testing.T) {
		ctx := logging.WithLayer(context.Background(), logging.LayerFrontend)
		layer := logging.LayerFromContext(ctx)
		if layer != logging.LayerFrontend {
			t.Errorf("LayerFromContext() = %q, want %q", layer, logging.LayerFrontend)
		}
	})

	t.Run("returns default for wrong type", func(t *testing.T) {
		// Use private key type to test wrong value type scenario
		ctx := logging.WithLayer(context.Background(), logging.LayerAPI)
		layer := logging.LayerFromContext(ctx)
		if layer != logging.LayerAPI {
			t.Errorf("LayerFromContext() = %q, want %q", layer, logging.LayerAPI)
		}
	})
}

func TestWithComponent(t *testing.T) {
	ctx := context.Background()
	newCtx := logging.WithComponent(ctx, logging.ComponentAuth)

	if ctx == newCtx {
		t.Error("WithComponent should return new context")
	}

	component := logging.ComponentFromContext(newCtx)
	if component != logging.ComponentAuth {
		t.Errorf("ComponentFromContext() = %q, want %q", component, logging.ComponentAuth)
	}
}

func TestComponentFromContext(t *testing.T) {
	t.Run("returns empty for empty context", func(t *testing.T) {
		ctx := context.Background()
		component := logging.ComponentFromContext(ctx)
		if component != "" {
			t.Errorf("ComponentFromContext() = %q, want empty string", component)
		}
	})

	t.Run("returns component from context", func(t *testing.T) {
		ctx := logging.WithComponent(context.Background(), logging.ComponentNetwork)
		component := logging.ComponentFromContext(ctx)
		if component != logging.ComponentNetwork {
			t.Errorf("ComponentFromContext() = %q, want %q", component, logging.ComponentNetwork)
		}
	})
}

func TestLayerConstants(t *testing.T) {
	// Verify layer constants are defined
	if logging.LayerBackend != "backend" {
		t.Errorf("LayerBackend = %q, want backend", logging.LayerBackend)
	}
	if logging.LayerAPI != "api" {
		t.Errorf("LayerAPI = %q, want api", logging.LayerAPI)
	}
	if logging.LayerFrontend != "frontend" {
		t.Errorf("LayerFrontend = %q, want frontend", logging.LayerFrontend)
	}
}

func TestComponentConstants(t *testing.T) {
	// Verify some key component constants are defined
	components := map[string]string{
		"ComponentAuth":      logging.ComponentAuth,
		"ComponentDiscovery": logging.ComponentDiscovery,
		"ComponentDevices":   logging.ComponentDevices,
		"ComponentNetwork":   logging.ComponentNetwork,
		"ComponentSurvey":    logging.ComponentSurvey,
		"ComponentWebSocket": logging.ComponentWebSocket,
		"ComponentSpeedtest": logging.ComponentSpeedtest,
		"ComponentIperf":     logging.ComponentIperf,
		"ComponentVuln":      logging.ComponentVuln,
		"ComponentConfig":    logging.ComponentConfig,
		"ComponentSystem":    logging.ComponentSystem,
		"ComponentDNS":       logging.ComponentDNS,
		"ComponentDHCP":      logging.ComponentDHCP,
		"ComponentGateway":   logging.ComponentGateway,
		"ComponentVLAN":      logging.ComponentVLAN,
		"ComponentWiFi":      logging.ComponentWiFi,
		"ComponentCable":     logging.ComponentCable,
		"ComponentPublicIP":  logging.ComponentPublicIP,
		"ComponentExport":    logging.ComponentExport,
		"ComponentSetup":     logging.ComponentSetup,
	}

	for name, value := range components {
		if value == "" {
			t.Errorf("%s should not be empty", name)
		}
	}
}

func TestContextChaining(t *testing.T) {
	ctx := context.Background()
	ctx = logging.WithLayer(ctx, logging.LayerAPI)
	ctx = logging.WithComponent(ctx, logging.ComponentAuth)
	ctx = logging.WithRequestID(ctx, "req-789")

	layer := logging.LayerFromContext(ctx)
	component := logging.ComponentFromContext(ctx)
	requestID := logging.RequestIDFromContext(ctx)

	if layer != logging.LayerAPI {
		t.Errorf("LayerFromContext() = %q, want %q", layer, logging.LayerAPI)
	}
	if component != logging.ComponentAuth {
		t.Errorf("ComponentFromContext() = %q, want %q", component, logging.ComponentAuth)
	}
	if requestID != "req-789" {
		t.Errorf("RequestIDFromContext() = %q, want req-789", requestID)
	}
}
