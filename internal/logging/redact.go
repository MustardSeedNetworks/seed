// Package logging provides secure logging utilities with automatic redaction of sensitive data.
package logging

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
)

// Sensitive field patterns that should always be redacted.
var sensitivePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*[^\s&]+`),
	regexp.MustCompile(`(?i)(token|auth|api[_-]?key|secret)\s*[=:]\s*[^\s&]+`),
	regexp.MustCompile(`(?i)(bearer\s+)\S+`),
	regexp.MustCompile(`(?i)(basic\s+)\S+`),
}

// Sensitive header names (case-insensitive).
var sensitiveHeaders = map[string]bool{
	"authorization":       true,
	"x-api-key":           true,
	"x-auth-token":        true,
	"cookie":              true,
	"set-cookie":          true,
	"x-csrf-token":        true,
	"x-xsrf-token":        true,
	"proxy-authorization": true,
}

// RedactString removes sensitive data from a string.
func RedactString(s string) string {
	for _, pattern := range sensitivePatterns {
		s = pattern.ReplaceAllString(s, "[REDACTED]")
	}
	return s
}

// RedactHeaders returns a map of headers with sensitive values redacted.
func RedactHeaders(headers http.Header) map[string]string {
	redacted := make(map[string]string)
	for key, values := range headers {
		lowerKey := strings.ToLower(key)
		if sensitiveHeaders[lowerKey] {
			redacted[key] = "[REDACTED]"
		} else {
			redacted[key] = strings.Join(values, ", ")
		}
	}
	return redacted
}

// RedactMap redacts sensitive fields in a map (useful for JSON logging).
func RedactMap(data map[string]interface{}) map[string]interface{} {
	redacted := make(map[string]interface{})
	for key, value := range data {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "password") ||
			strings.Contains(lowerKey, "secret") ||
			strings.Contains(lowerKey, "token") ||
			strings.Contains(lowerKey, "key") ||
			strings.Contains(lowerKey, "auth") {
			redacted[key] = "[REDACTED]"
		} else {
			// For string values, apply pattern-based redaction
			if strVal, ok := value.(string); ok {
				redacted[key] = RedactString(strVal)
			} else {
				redacted[key] = value
			}
		}
	}
	return redacted
}

// Logf is a safe logging function that redacts sensitive data.
// Note: Prefer using slog directly with the RedactingHandler for new code.
func Logf(format string, args ...interface{}) {
	// Convert args to strings and redact
	redactedArgs := make([]interface{}, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case string:
			redactedArgs[i] = RedactString(v)
		case http.Header:
			redactedArgs[i] = RedactHeaders(v)
		case map[string]interface{}:
			redactedArgs[i] = RedactMap(v)
		default:
			redactedArgs[i] = arg
		}
	}
	slog.Info(fmt.Sprintf(format, redactedArgs...))
}

// SafeError creates a safe error message with redacted content.
func SafeError(err error, context string) error {
	if err == nil {
		return nil
	}
	redactedMsg := RedactString(err.Error())
	return fmt.Errorf("%s: %s", context, redactedMsg)
}

// LogRequest logs an HTTP request with sensitive data redacted.
// Note: Prefer using LoggingMiddleware for request logging in new code.
func LogRequest(r *http.Request, message string) {
	slog.Info(message,
		"method", r.Method,
		"path", r.URL.Path,
		"client_ip", GetClientIP(r),
		"headers", RedactHeaders(r.Header),
	)
}

// GetClientIP extracts client IP from request (handles proxies).
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For (but don't log it - could be sensitive)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		addr = addr[:idx]
	}
	return addr
}
