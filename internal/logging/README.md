# Logging Package - Structured Logging with Automatic Redaction

This package provides structured logging utilities built on Go's `log/slog` with automatic redaction
of sensitive data like passwords, tokens, and API keys.

## Migration Status

The logging infrastructure is complete. Migration of individual files is ongoing.

### Completed

- [x] Core logging package (logger.go, handler.go, middleware.go)
- [x] Config support (LoggingConfig in config.go)
- [x] RequestIDMiddleware integrated into server
- [x] Security fix #301 (removed insecure LOG_ACCESS_TOKEN)
- [x] cmd/seed/main.go migrated to slog

### Pending Migration

Files with `log.Printf` calls that need migration to `slog`:

- internal/api/server.go (~40 calls)
- internal/api/handlers\_\*.go
- internal/discovery/\*.go
- internal/wifi/\*.go
- Other internal packages

### Migration Pattern

```go
// Before
log.Printf("Starting server on port %d", port)
log.Printf("Error: %v", err)

// After - Import "log/slog" and use:
slog.Info("Starting server", "port", port)
slog.Error("Operation failed", "error", err)

// With request context (in handlers):
logging.InfoContext(ctx, "Request processed", "method", r.Method, "path", r.URL.Path)
```

## Why Use This Package?

**Problem:** Logging request bodies, headers, or error messages can accidentally expose sensitive
information like:

- Passwords in login requests
- JWT tokens in Authorization headers
- API keys in URL parameters
- Session cookies

**Solution:** This package provides drop-in replacements for `log.Printf` and utilities for safely
logging HTTP requests, headers, and data structures.

## Quick Start

### Basic String Redaction

```go
import "github.com/krisarmstrong/seed/internal/logging"

// Instead of:
log.Printf("error: %v", err) // May contain passwords!

// Use:
logging.Logf("error: %v", err) // Automatically redacted
```

### HTTP Request Logging

```go
// Instead of:
log.Printf("request headers: %v", r.Header) // Exposes Authorization!

// Use:
logging.LogRequest(r, "incoming request") // Safe
```

### Map/JSON Redaction

```go
data := map[string]interface{}{
    "username": "admin",
    "password": "secret123", // Sensitive!
    "status":   "active",
}

// Instead of:
log.Printf("user data: %v", data) // Leaks password!

// Use:
safeData := logging.RedactMap(data)
log.Printf("user data: %v", safeData) // password: [REDACTED]
```

### Header Redaction

```go
// Redact Authorization, Cookie, X-API-Key headers
safeHeaders := logging.RedactHeaders(r.Header)
log.Printf("headers: %v", safeHeaders) // Safe to log
```

## API Reference

### `Logf(format string, args ...interface{})`

Drop-in replacement for `log.Printf` with automatic redaction.

```go
logging.Logf("user %s login failed: %v", username, err)
// Automatically redacts passwords/tokens in err.Error()
```

### `RedactString(s string) string`

Redacts sensitive patterns from a string.

**Redacted patterns:**

- `password=...`
- `token=...`
- `api_key=...`
- `Bearer ...`
- `Basic ...`
- And more (see code for full list)

```go
msg := "login failed with password=secret123"
safe := logging.RedactString(msg)
// Result: "login failed with [REDACTED]"
```

### `RedactHeaders(headers http.Header) map[string]string`

Returns headers with sensitive values redacted.

**Redacted headers:**

- `Authorization`
- `Cookie`
- `Set-Cookie`
- `X-API-Key`
- `X-Auth-Token`
- `X-CSRF-Token`
- `Proxy-Authorization`

```go
safeHeaders := logging.RedactHeaders(r.Header)
log.Printf("request headers: %v", safeHeaders)
```

### `RedactMap(data map[string]interface{}) map[string]interface{}`

Redacts fields with sensitive names or values.

**Redacted field names:**

- Contains "password"
- Contains "secret"
- Contains "token"
- Contains "key"
- Contains "auth"

```go
data := map[string]interface{}{
    "username":   "admin",
    "password":   "secret",
    "auth_token": "xyz",
}
safe := logging.RedactMap(data)
// password and auth_token are [REDACTED]
```

### `LogRequest(r *http.Request, message string)`

Logs an HTTP request with safe header redaction.

```go
logging.LogRequest(r, "incoming API request")
// Logs: method, path, client IP, and REDACTED headers
```

### `SafeError(err error, context string) error`

Creates a safe error with redacted content.

```go
err := someFunction() // May contain password in error message
safeErr := logging.SafeError(err, "database connection")
return safeErr // Safe to return to client or log
```

## Examples

### Before (UNSAFE ❌)

```go
func handleLogin(w http.ResponseWriter, r *http.Request) {
    bodyBytes, _ := io.ReadAll(r.Body)
    var req LoginRequest
    if err := json.Unmarshal(bodyBytes, &req); err != nil {
        // DANGER: Logs entire body including password!
        log.Printf("decode error: %v body=%q", err, string(bodyBytes))
        return
    }
}
```

### After (SAFE ✅)

```go
import "github.com/krisarmstrong/seed/internal/logging"

func handleLogin(w http.ResponseWriter, r *http.Request) {
    r.Body = http.MaxBytesReader(w, r.Body, 1024)
    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        // Safe: Only logs client IP and error (no credentials)
        logging.Logf("login decode error from %s: %v",
            logging.GetClientIP(r), err)
        return
    }
}
```

## Testing

The package includes comprehensive tests:

```bash
go test ./internal/logging/... -v
```

Test coverage:

- String pattern redaction
- Header redaction
- Map/JSON redaction
- Error redaction
- Edge cases (nil values, empty strings, etc.)

## CI Integration

The CI pipeline includes automated checks for unsafe logging patterns:

```yaml
# .github/workflows/ci.yml
- name: Check for unsafe logging patterns
  run: |
    # Fails if code uses log.Print with r.Body, passwords, etc.
    # Enforces use of logging.Redact* functions
```

**Detected patterns:**

- `log.Print.*r.Body`
- `log.Print.*password`
- `log.Print.*token`
- `io.ReadAll.*r.Body.*log`
- And more (see CI workflow)

## Best Practices

1. **Always use `logging.Logf` instead of `log.Printf`** for user-generated or external data
2. **Never log raw request bodies** - they may contain passwords
3. **Redact headers before logging** - use `logging.RedactHeaders()`
4. **Use `http.MaxBytesReader`** to prevent memory exhaustion
5. **Return safe errors** - use `logging.SafeError()` for errors exposed to users

## Performance

- **Minimal overhead**: Regex patterns are pre-compiled
- **Zero allocations** for safe strings (no sensitive data found)
- **Lazy evaluation**: Only processes strings that are actually logged

## Related Issues

- Fixes #454 - Prevents credential leakage in login errors
- Implements #479 - Systematic redaction and CI enforcement

## Contributing

When adding new sensitive patterns:

1. Update `sensitivePatterns` or `sensitiveHeaders` in `redact.go`
2. Add test cases in `redact_test.go`
3. Run tests: `go test ./internal/logging/... -v`
4. Update CI patterns in `.github/workflows/ci.yml` if needed
