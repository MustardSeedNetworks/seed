// Package logging provides secure logging utilities with automatic redaction of sensitive data.
//
// SECURITY MODEL FOR IP ADDRESS LOGGING (Plan G6):
//
// This package includes utilities for logging HTTP requests, including client IP addresses.
// It's critical to understand the trust model for IP address sources:
//
// UNTRUSTED SOURCES (can be spoofed by clients):
//   - X-Forwarded-For header
//   - X-Real-IP header
//   - X-Client-IP header
//   - Other proxy headers (CF-Connecting-IP, True-Client-IP, etc.)
//
// TRUSTED SOURCE (cannot be spoofed):
//   - r.RemoteAddr (the actual TCP connection source)
//
// GetClientIP() in this package uses UNTRUSTED sources for convenience in logging
// scenarios where you want to see the "apparent" client IP when behind a reverse proxy.
// However, WITHOUT proper proxy configuration, these values can be spoofed.
//
// For security-critical operations (rate limiting, access control, IP banning):
//   - Use api.GetClientIP() which ONLY uses r.RemoteAddr
//   - Never trust X-Forwarded-For or similar headers for security decisions
//
// Logs may show spoofed IPs in reverse proxy scenarios unless:
//  1. The reverse proxy is configured to strip/override client-supplied headers
//  2. Trusted proxy configuration is properly set up
//  3. The proxy is verified to be in the request path
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

// Sensitive header names (case-insensitive) (fixes #713, #714).
// Extended to include authentication, credential, and privacy-sensitive headers.
var sensitiveHeaders = map[string]bool{
	// Authentication and credentials
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
	// Privacy-sensitive headers (fixes #714 - client IP exposure)
	// SECURITY NOTE: These headers are UNTRUSTED and can be spoofed by clients.
	// They are redacted in logs for privacy, but also because they should NEVER
	// be used for security decisions (rate limiting, access control, etc.).
	// For security-critical IP tracking, use r.RemoteAddr only (see api.GetClientIP).
	"x-forwarded-for":     true, // UNTRUSTED - can be spoofed
	"x-real-ip":           true, // UNTRUSTED - can be spoofed
	"x-client-ip":         true, // UNTRUSTED - can be spoofed
	"cf-connecting-ip":    true, // Cloudflare - only trust if behind verified CF proxy
	"true-client-ip":      true, // Akamai - only trust if behind verified Akamai proxy
	"x-cluster-client-ip": true, // UNTRUSTED - can be spoofed
	"forwarded":           true, // RFC 7239 - UNTRUSTED without proxy verification
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
func RedactMap(data map[string]any) map[string]any {
	redacted := make(map[string]any)
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
func Logf(format string, args ...any) {
	// Convert args to strings and redact
	redactedArgs := make([]any, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case string:
			redactedArgs[i] = RedactString(v)
		case http.Header:
			redactedArgs[i] = RedactHeaders(v)
		case map[string]any:
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

// GetClientIP extracts client IP from request for logging and display purposes.
//
// SECURITY WARNING (fixes #714 - Plan G6):
// This function returns UNTRUSTED IP addresses that can be spoofed by malicious clients.
// It checks X-Forwarded-For and X-Real-IP headers which are trivially spoofed.
//
// TRUST MODEL:
// - X-Forwarded-For: UNTRUSTED - Any client can set this header to any value
// - X-Real-IP: UNTRUSTED - Any client can set this header to any value
// - r.RemoteAddr: TRUSTED - This is the actual TCP connection source
//
// USE CASES:
// - OK for logging/debugging (helps in reverse proxy scenarios)
// - OK for display in admin dashboards
// - NEVER use for rate limiting (see api.GetClientIP which uses only RemoteAddr)
// - NEVER use for access control or security decisions
// - NEVER use for ban lists or IP blocking
//
// REVERSE PROXY SCENARIOS:
// When behind a reverse proxy (nginx, Cloudflare, etc.), this function will show
// the client IP if the proxy sets XFF correctly. However, WITHOUT proper proxy
// configuration to strip client-supplied XFF headers, logs can show spoofed IPs.
//
// For production deployments behind reverse proxies:
// 1. Configure the proxy to strip/override client XFF headers
// 2. Set trusted proxy configuration
// 3. Use api.GetClientIP for any security-sensitive operations.
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (UNTRUSTED - can be spoofed by clients)
	// This is checked first for convenience in reverse proxy scenarios, but
	// the value should be treated as untrusted user input.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			clientIP := strings.TrimSpace(parts[0])
			// Log when XFF is present to indicate the IP may be untrusted
			slog.Debug("Request includes X-Forwarded-For header (UNTRUSTED)",
				"xff_value", xff,
				"parsed_ip", clientIP,
				"remote_addr", r.RemoteAddr,
				"security_note", "XFF can be spoofed - only use for logging, not security decisions")
			return clientIP
		}
	}

	// Check X-Real-IP header (UNTRUSTED - can be spoofed by clients)
	// Similar trust issues as X-Forwarded-For
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		slog.Debug("Request includes X-Real-IP header (UNTRUSTED)",
			"xri_value", xri,
			"remote_addr", r.RemoteAddr,
			"security_note", "X-Real-IP can be spoofed - only use for logging, not security decisions")
		return xri
	}

	// Fall back to RemoteAddr (TRUSTED - the only reliable source)
	// This is the actual TCP connection source and cannot be spoofed
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		addr = addr[:idx]
	}
	return addr
}
