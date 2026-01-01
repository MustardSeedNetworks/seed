// Package logging provides structured logging with automatic redaction of sensitive data.
//
// This package wraps Go's log/slog with automatic sensitive data redaction,
// request ID correlation, and configurable output formats (text/JSON).
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LoggingConfig contains logging configuration options.
//
//nolint:revive // stuttering: keeping name for backward compatibility.
type LoggingConfig struct {
	Level      string    `yaml:"level"`       // DEBUG, INFO, WARN, ERROR (default: INFO)
	Format     string    `yaml:"format"`      // text or json (default: json)
	AddSource  bool      `yaml:"add_source"`  // Include file:line in logs
	File       string    `yaml:"file"`        // Log file path (empty = stdout only)
	MaxSize    int       `yaml:"max_size"`    // Max MB per log file before rotation
	MaxBackups int       `yaml:"max_backups"` // Number of old files to keep
	MaxAge     int       `yaml:"max_age"`     // Days to keep old files
	Compress   bool      `yaml:"compress"`    // Compress rotated files
	Writer     io.Writer `yaml:"-"`           // Override output writer (for testing, default: os.Stdout)
}

// DefaultLoggingConfig returns sensible defaults for logging.
func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Level:      "info",
		Format:     "json",
		AddSource:  false,
		File:       "",
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}
}

// contextKey is a type for context keys to avoid collisions.
type contextKey string

const (
	// requestIDKey is the context key for request IDs.
	requestIDKey contextKey = "request_id"
	// userIDKey is the context key for user IDs.
	userIDKey contextKey = "user_id"
)

var (
	// globalLogger is the package-level logger instance.
	globalLogger *slog.Logger
	loggerMu     sync.RWMutex
)

// parseLevel converts a string level to slog.Level.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// InitLogger initializes the global structured logger with the given configuration.
// It sets up output writers (file and/or stdout), log rotation, and the redacting handler.
func InitLogger(cfg *LoggingConfig) error {
	if cfg == nil {
		cfg = DefaultLoggingConfig()
	}

	// Determine output writers
	var writers []io.Writer
	if cfg.Writer != nil {
		writers = append(writers, cfg.Writer)
	} else {
		writers = append(writers, os.Stdout)
	}

	// Add file writer with rotation if configured (fixes #714)
	// Log rotation policy:
	// - Rotates when file reaches MaxSize MB
	// - Keeps up to MaxBackups old log files
	// - Deletes files older than MaxAge days
	// - Compresses rotated files if Compress is true
	if cfg.File != "" {
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize,    // MB per log file before rotation
			MaxBackups: cfg.MaxBackups, // Number of old files to keep
			MaxAge:     cfg.MaxAge,     // Days to keep old files
			Compress:   cfg.Compress,   // Compress rotated files
		}
		writers = append(writers, fileWriter)
	}

	// Create multi-writer for both stdout and file
	var output io.Writer
	if len(writers) == 1 {
		output = writers[0]
	} else {
		output = io.MultiWriter(writers...)
	}

	// Configure handler options
	opts := &slog.HandlerOptions{
		Level:     parseLevel(cfg.Level),
		AddSource: cfg.AddSource,
	}

	// Create base handler based on format
	var baseHandler slog.Handler
	if strings.EqualFold(cfg.Format, "json") {
		baseHandler = slog.NewJSONHandler(output, opts)
	} else {
		baseHandler = slog.NewTextHandler(output, opts)
	}

	// Wrap with redacting handler for automatic sensitive data redaction
	redactingHandler := NewRedactingHandler(baseHandler)

	// Set global logger
	loggerMu.Lock()
	globalLogger = slog.New(redactingHandler)
	slog.SetDefault(globalLogger)
	loggerMu.Unlock()

	return nil
}

// GetLogger returns the global logger instance.
// If InitLogger hasn't been called, returns slog.Default().
func GetLogger() *slog.Logger {
	loggerMu.RLock()
	defer loggerMu.RUnlock()

	if globalLogger == nil {
		return slog.Default()
	}
	return globalLogger
}

// WithRequestID returns a new context with the given request ID.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext extracts the request ID from the context.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithUserID returns a new context with the given user ID.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// UserIDFromContext extracts the user ID from the context.
func UserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}

// FromContext returns a logger with contextual information (request ID, user ID).
// This is the preferred way to get a logger in HTTP handlers.
func FromContext(ctx context.Context) *slog.Logger {
	logger := GetLogger()
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		logger = logger.With("request_id", requestID)
	}
	if userID := UserIDFromContext(ctx); userID != "" {
		logger = logger.With("user_id", userID)
	}
	return logger
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	GetLogger().Debug(msg, args...)
}

// Info logs an info message.
func Info(msg string, args ...any) {
	GetLogger().Info(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	GetLogger().Warn(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	GetLogger().Error(msg, args...)
}

// DebugContext logs a debug message with context (includes request_id if present).
func DebugContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Debug(msg, args...)
}

// InfoContext logs an info message with context (includes request_id if present).
func InfoContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Info(msg, args...)
}

// WarnContext logs a warning message with context (includes request_id if present).
func WarnContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Warn(msg, args...)
}

// ErrorContext logs an error message with context (includes request_id if present).
func ErrorContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Error(msg, args...)
}
