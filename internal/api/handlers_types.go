// Package api provides the HTTP/WebSocket server.
package api

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
)

// ============================================================================
// Standardized API Error Responses
// ============================================================================

// ErrorResponse represents a standardized error response for the API.
// This ensures consistent error formats across all endpoints.
type ErrorResponse struct {
	Error   string `json:"error"`             // Human-readable error message
	Code    string `json:"code"`              // Machine-readable error code
	Details string `json:"details,omitempty"` // Additional details (optional)
}

// Common error codes for consistent API responses.
const (
	ErrCodeBadRequest       = "BAD_REQUEST"
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeForbidden        = "FORBIDDEN"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeConflict         = "CONFLICT"
	ErrCodeMethodNotAllowed = "METHOD_NOT_ALLOWED"
	ErrCodeInternal         = "INTERNAL_ERROR"
	ErrCodeServiceUnavail   = "SERVICE_UNAVAILABLE"
	ErrCodeValidation       = "VALIDATION_ERROR"
	ErrCodeRateLimit        = "RATE_LIMIT_EXCEEDED"
)

// sendErrorResponseWithDetails sends a standardized JSON error response with additional details.
// Use this for all error responses to ensure consistency across the API (fixes #694).
func sendErrorResponseWithDetails(w http.ResponseWriter, logger *slog.Logger, status int, code, message, details string) {
	resp := ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		if logger != nil {
			logger.Error("Error encoding error response", "error", err)
		} else {
			slog.Error("Error encoding error response (no logger context)", "error", err)
		}
	}
}

// sendJSONResponse is a helper to send JSON responses and handle encoding errors.
// Used across all handler files (fixes #544 - shared utilities).
func sendJSONResponse(w http.ResponseWriter, logger *slog.Logger, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		if logger != nil {
			logger.Error("Error encoding JSON response", "error", err)
		} else {
			slog.Error("Error encoding JSON response (no logger context)", "error", err)
		}
	}
}

// readLastLines reads the last N lines from a file, up to maxBytes from the end.
// Used by handleLogs and other log-reading handlers (fixes #544 - shared utilities).
func readLastLines(path string, maxBytes int64, maxLines int) ([]string, error) {
	//nolint:gosec // G304: path is from config for log file location
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	var start int64
	if info.Size() > maxBytes {
		start = info.Size() - maxBytes
	}
	if start > 0 {
		if _, err := f.Seek(start, io.SeekStart); err != nil {
			return nil, err
		}
	}

	scanner := bufio.NewScanner(f)
	// Allow long lines (up to 1MB)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lines := make([]string, 0, maxLines)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > maxLines {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
