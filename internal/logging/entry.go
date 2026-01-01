// Package logging provides structured logging with automatic redaction of sensitive data.
package logging

import (
	"context"
	"time"
)

// LogEntry represents a structured log entry that can be broadcast to clients
// and stored for querying. It provides a unified format for logs across all layers
// (backend, API handlers, frontend).
type LogEntry struct {
	Timestamp  time.Time      `json:"timestamp"`             // When the log was created
	Level      string         `json:"level"`                 // ERROR, WARN, INFO, DEBUG
	Layer      string         `json:"layer"`                 // backend, api, frontend
	RequestID  string         `json:"request_id,omitempty"`  // Correlation ID for request tracing
	SessionID  string         `json:"session_id,omitempty"`  // User session ID
	Message    string         `json:"message"`               // Human-readable log message
	Component  string         `json:"component,omitempty"`   // Component that generated the log
	DurationMs int64          `json:"duration_ms,omitempty"` // Duration for timed operations
	Metadata   map[string]any `json:"metadata,omitempty"`    // Additional structured data
	Stack      string         `json:"stack,omitempty"`       // Stack trace for errors
}

// Standard component names for consistent categorization across the codebase.
const (
	ComponentAuth      = "auth"            // Authentication and authorization
	ComponentDiscovery = "discovery"       // Network protocol discovery (LLDP/CDP/EDP)
	ComponentDevices   = "devices"         // Device scanning and management
	ComponentNetwork   = "network"         // Network interface management
	ComponentSurvey    = "survey"          // WiFi survey operations
	ComponentWebSocket = "websocket"       // WebSocket connections and messaging
	ComponentSpeedtest = "speedtest"       // Speed testing
	ComponentIperf     = "iperf"           // iPerf3 operations
	ComponentVuln      = "vulnerabilities" // Vulnerability scanning
	ComponentConfig    = "config"          // Configuration management
	ComponentSystem    = "system"          // System health and status
	ComponentDNS       = "dns"             // DNS testing
	ComponentDHCP      = "dhcp"            // DHCP monitoring
	ComponentGateway   = "gateway"         // Gateway testing
	ComponentVLAN      = "vlan"            // VLAN management
	ComponentWiFi      = "wifi"            // WiFi scanning
	ComponentCable     = "cable"           // Cable diagnostics
	ComponentPublicIP  = "publicip"        // Public IP detection
	ComponentExport    = "export"          // Data export
	ComponentSetup     = "setup"           // Initial setup wizard
)

// Standard layer names for log categorization.
const (
	LayerBackend  = "backend"  // Core backend services
	LayerAPI      = "api"      // API request handlers
	LayerFrontend = "frontend" // Frontend JavaScript logs
)

// Context keys for layer and component.
const (
	layerKey     contextKey = "log_layer"
	componentKey contextKey = "log_component"
)

// WithLayer returns a new context with the specified layer.
func WithLayer(ctx context.Context, layer string) context.Context {
	return context.WithValue(ctx, layerKey, layer)
}

// LayerFromContext extracts the layer from the context.
func LayerFromContext(ctx context.Context) string {
	if layer, ok := ctx.Value(layerKey).(string); ok {
		return layer
	}
	return LayerBackend // Default to backend
}

// WithComponent returns a new context with the specified component.
func WithComponent(ctx context.Context, component string) context.Context {
	return context.WithValue(ctx, componentKey, component)
}

// ComponentFromContext extracts the component from the context.
func ComponentFromContext(ctx context.Context) string {
	if component, ok := ctx.Value(componentKey).(string); ok {
		return component
	}
	return ""
}

// NewLogEntry creates a new LogEntry with the current timestamp.
func NewLogEntry(level, message string) *LogEntry {
	return &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Layer:     LayerBackend,
	}
}

// WithRequestID sets the request ID on the log entry.
func (e *LogEntry) WithRequestID(requestID string) *LogEntry {
	e.RequestID = requestID
	return e
}

// WithLayer sets the layer on the log entry.
func (e *LogEntry) WithLayer(layer string) *LogEntry {
	e.Layer = layer
	return e
}

// WithComponent sets the component on the log entry.
func (e *LogEntry) WithComponent(component string) *LogEntry {
	e.Component = component
	return e
}

// WithDuration sets the duration in milliseconds on the log entry.
func (e *LogEntry) WithDuration(d time.Duration) *LogEntry {
	e.DurationMs = d.Milliseconds()
	return e
}

// WithMetadata sets additional metadata on the log entry.
func (e *LogEntry) WithMetadata(metadata map[string]any) *LogEntry {
	e.Metadata = metadata
	return e
}

// WithStack sets the stack trace on the log entry.
func (e *LogEntry) WithStack(stack string) *LogEntry {
	e.Stack = stack
	return e
}

// AddMetadata adds a key-value pair to the metadata.
func (e *LogEntry) AddMetadata(key string, value any) *LogEntry {
	if e.Metadata == nil {
		e.Metadata = make(map[string]any)
	}
	e.Metadata[key] = value
	return e
}
