// Package logging provides secure logging utilities with automatic redaction of sensitive data.
package logging

import (
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
)

// Sensitive field patterns that should always be redacted (fixes #713).
// Comprehensive patterns for: passwords, tokens, API keys, secrets, SSNs, credit cards, etc.
var sensitivePatterns = []*regexp.Regexp{
	// Passwords and credentials
	regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*[^\s&]+`),
	regexp.MustCompile(`(?i)(token|auth|api[_-]?key|secret)\s*[=:]\s*[^\s&]+`),
	regexp.MustCompile(`(?i)(bearer\s+)\S+`),
	regexp.MustCompile(`(?i)(basic\s+)\S+`),

	// API keys and tokens - common formats
	regexp.MustCompile(`(?i)(api[_-]?key|apikey|access[_-]?key)\s*[=:]\s*[a-zA-Z0-9_\-\.]+`),
	regexp.MustCompile(`(?i)(client[_-]?secret|client_id)\s*[=:]\s*[a-zA-Z0-9_\-.]+`),
	regexp.MustCompile(`(?i)(oauth[_-]?token|refresh[_-]?token)\s*[=:]\s*[a-zA-Z0-9_\-\.]+`),

	// AWS-style keys
	regexp.MustCompile(`(?i)(aws[_-]?access[_-]?key[_-]?id|aws[_-]?secret[_-]?access[_-]?key)\s*[=:]\s*[A-Z0-9]+`),
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`), // AWS Access Key ID pattern

	// GitHub/GitLab tokens
	regexp.MustCompile(`(?i)(github[_-]?token|gh[ps]_[a-zA-Z0-9_]{36,})`),
	regexp.MustCompile(`(?i)(gitlab[_-]?token|glpat-[a-zA-Z0-9_\-]{20,})`),

	// Private keys
	regexp.MustCompile(`-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----[^-]*-----END\s+(RSA\s+)?PRIVATE\s+KEY-----`),
	regexp.MustCompile(`(?i)(private[_-]?key|privatekey)\s*[=:]\s*[^\s&]+`),

	// Social Security Numbers (US) - XXX-XX-XXXX format
	// Note: Go regexp doesn't support negative lookahead, so this is a simple pattern
	regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),

	// Credit card numbers (13-19 digits, with or without spaces/dashes)
	// Matches Visa, MasterCard, Amex, Discover, etc.
	regexp.MustCompile(`\b(?:\d{4}[\s\-]?){3}\d{1,7}\b`),
	regexp.MustCompile(`\b\d{13,19}\b`), // Simple 13-19 digit sequence

	// Email addresses (for privacy)
	regexp.MustCompile(`\b[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Z|a-z]{2,}\b`),

	// JWT tokens (base64.base64.base64 format)
	regexp.MustCompile(`eyJ[a-zA-Z0-9_\-]*\.eyJ[a-zA-Z0-9_\-]*\.[a-zA-Z0-9_\-]*`),

	// Generic secrets and credentials
	regexp.MustCompile(`(?i)(credential|credentials|auth[_-]?token)\s*[=:]\s*[^\s&]+`),
	regexp.MustCompile(`(?i)(passphrase|pin|shared[_-]?secret)\s*[=:]\s*[^\s&]+`),
}

// Sensitive header names (case-insensitive) (fixes #713).
// Extended to include more common authentication and credential headers.
var sensitiveHeaders = map[string]bool{
	"authorization":       true,
	"x-api-key":           true,
	"x-auth-token":        true,
	"cookie":              true,
	"set-cookie":          true,
	"x-csrf-token":        true,
	"x-xsrf-token":        true,
	"proxy-authorization": true,
	"x-access-token":      true,
	"x-refresh-token":     true,
	"x-session-token":     true,
	"x-client-secret":     true,
	"x-client-id":         true,
	"x-oauth-token":       true,
	"apikey":              true,
	"api-key":             true,
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
