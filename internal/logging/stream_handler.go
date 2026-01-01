// Package logging provides structured logging with automatic redaction of sensitive data.
package logging

import (
	"context"
	"log/slog"
	"runtime"
	"time"
)

// StreamingHandler wraps an slog.Handler and broadcasts log entries to connected clients.
// It extracts context values (request_id, layer, component) and creates LogEntry objects
// for real-time streaming to the frontend log viewer.
type StreamingHandler struct {
	wrapped     slog.Handler
	broadcaster *LogBroadcaster
	attrs       []slog.Attr
	groups      []string
}

// NewStreamingHandler creates a new handler that wraps the given handler
// and broadcasts logs to the provided broadcaster.
func NewStreamingHandler(wrapped slog.Handler, broadcaster *LogBroadcaster) *StreamingHandler {
	return &StreamingHandler{
		wrapped:     wrapped,
		broadcaster: broadcaster,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *StreamingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.wrapped.Enabled(ctx, level)
}

// Handle processes a log record, broadcasting it to clients and passing to the wrapped handler.
//

func (h *StreamingHandler) Handle(ctx context.Context, r slog.Record) error {
	// First, let the wrapped handler process the record
	if err := h.wrapped.Handle(ctx, r); err != nil {
		return err
	}

	// Create a LogEntry from the record
	entry := h.recordToEntry(ctx, r)

	// Broadcast to connected clients
	if h.broadcaster != nil {
		h.broadcaster.Write(entry)
	}

	return nil
}

// WithAttrs returns a new handler with the given attributes added.
func (h *StreamingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &StreamingHandler{
		wrapped:     h.wrapped.WithAttrs(attrs),
		broadcaster: h.broadcaster,
		attrs:       append(h.attrs, attrs...),
		groups:      h.groups,
	}
}

// WithGroup returns a new handler with the given group name prepended.
func (h *StreamingHandler) WithGroup(name string) slog.Handler {
	return &StreamingHandler{
		wrapped:     h.wrapped.WithGroup(name),
		broadcaster: h.broadcaster,
		attrs:       h.attrs,
		groups:      append(h.groups, name),
	}
}

// recordToEntry converts an slog.Record to a LogEntry.
//

func (h *StreamingHandler) recordToEntry(ctx context.Context, r slog.Record) *LogEntry {
	entry := &LogEntry{
		Timestamp: r.Time,
		Level:     levelToString(r.Level),
		Message:   r.Message,
		Layer:     LayerFromContext(ctx),
		RequestID: RequestIDFromContext(ctx),
		Component: ComponentFromContext(ctx),
		Metadata:  make(map[string]any),
	}

	// If no layer in context, default to API layer for HTTP handlers
	if entry.Layer == "" {
		entry.Layer = LayerAPI
	}

	// Add pre-existing attributes from handler
	for _, attr := range h.attrs {
		h.addAttrToEntry(entry, attr)
	}

	// Add attributes from the record
	r.Attrs(func(attr slog.Attr) bool {
		h.addAttrToEntry(entry, attr)
		return true
	})

	// Add source information for errors
	if r.Level >= slog.LevelError {
		if r.PC != 0 {
			fs := runtime.CallersFrames([]uintptr{r.PC})
			f, _ := fs.Next()
			if f.File != "" {
				entry.AddMetadata("source_file", f.File)
				entry.AddMetadata("source_line", f.Line)
				entry.AddMetadata("source_func", f.Function)
			}
		}
	}

	return entry
}

// addAttrToEntry adds an slog.Attr to the LogEntry metadata.
//

func (h *StreamingHandler) addAttrToEntry(entry *LogEntry, attr slog.Attr) {
	key := attr.Key
	value := attr.Value.Any()

	// Handle special keys that map to LogEntry fields
	switch key {
	case "request_id":
		if entry.RequestID == "" {
			if s, ok := value.(string); ok {
				entry.RequestID = s
			}
		}
	case "component":
		if entry.Component == "" {
			if s, ok := value.(string); ok {
				entry.Component = s
			}
		}
	case "layer":
		if s, ok := value.(string); ok {
			entry.Layer = s
		}
	case "duration_ms":
		switch d := value.(type) {
		case int64:
			entry.DurationMs = d
		case int:
			entry.DurationMs = int64(d)
		}
	case "stack":
		if s, ok := value.(string); ok {
			entry.Stack = s
		}
	case "error":
		// Extract error message
		if err, ok := value.(error); ok {
			entry.AddMetadata(key, err.Error())
		} else {
			entry.AddMetadata(key, value)
		}
	default:
		// Add to metadata
		entry.AddMetadata(key, value)
	}
}

// levelToString converts slog.Level to a string.
func levelToString(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "ERROR"
	case level >= slog.LevelWarn:
		return "WARN"
	case level >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}

// InitLoggerWithBroadcaster initializes the global logger with streaming capabilities.
// This sets up the full logging pipeline: redaction → streaming → file/stdout output.
func InitLoggerWithBroadcaster(cfg *LoggingConfig, broadcaster *LogBroadcaster) error {
	if cfg == nil {
		cfg = DefaultLoggingConfig()
	}

	// First initialize the base logger without streaming
	if err := InitLogger(cfg); err != nil {
		return err
	}

	// If no broadcaster provided, create one
	if broadcaster == nil {
		broadcaster = InitBroadcaster(1000) // Default 1000 entry buffer
	}

	// Get the current handler and wrap it with streaming
	loggerMu.Lock()
	defer loggerMu.Unlock()

	currentHandler := globalLogger.Handler()
	streamingHandler := NewStreamingHandler(currentHandler, broadcaster)
	globalLogger = slog.New(streamingHandler)
	slog.SetDefault(globalLogger)

	return nil
}

// LogWithContext logs a message with context-aware attributes.
// This is a convenience function that extracts layer, component, and request_id from context.
func LogWithContext(ctx context.Context, level slog.Level, msg string, args ...any) {
	logger := FromContext(ctx)

	// Add layer and component if present in context
	if layer := LayerFromContext(ctx); layer != "" {
		args = append(args, "layer", layer)
	}
	if component := ComponentFromContext(ctx); component != "" {
		args = append(args, "component", component)
	}

	logger.Log(ctx, level, msg, args...)
}

// TimedOperation logs the start and completion of an operation with duration.
// Returns a function to be deferred that logs completion.
func TimedOperation(ctx context.Context, operation, component string) func() {
	start := time.Now()
	logger := FromContext(ctx).With("component", component)
	logger.Info(operation + " started")

	return func() {
		duration := time.Since(start)
		logger.Info(operation+" completed", "duration_ms", duration.Milliseconds())
	}
}
