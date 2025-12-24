// Package auth handles JWT authentication and CSRF protection.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CSRF token configuration.
const (
	// CSRFTokenLength is the length of the CSRF token in bytes before encoding.
	CSRFTokenLength = 32
	// CSRFTokenExpiry is how long a CSRF token remains valid.
	CSRFTokenExpiry = 24 * time.Hour
	// CSRFHeaderName is the HTTP header name for CSRF tokens.
	CSRFHeaderName = "X-CSRF-Token"
	// CSRFCookieName is the cookie name for CSRF tokens.
	CSRFCookieName = "csrf_token"
)

// CSRF errors.
var (
	// ErrCSRFTokenMissing is returned when no CSRF token is provided.
	ErrCSRFTokenMissing = errors.New("CSRF token missing")
	// ErrCSRFTokenInvalid is returned when the CSRF token is invalid.
	ErrCSRFTokenInvalid = errors.New("CSRF token invalid")
	// ErrCSRFTokenExpired is returned when the CSRF token has expired.
	ErrCSRFTokenExpired = errors.New("CSRF token expired")
)

// CSRFToken represents a CSRF token with its metadata.
type CSRFToken struct {
	Token     string    // The actual token string
	ExpiresAt time.Time // When the token expires
}

// CSRFManager manages CSRF token generation and validation.
type CSRFManager struct {
	mu     sync.RWMutex
	tokens map[string]*CSRFToken // Map of token to metadata, keyed by user session
	stopCh chan struct{}         // Channel to signal cleanup goroutine to stop (fixes #785)
}

// NewCSRFManager creates a new CSRF manager.
func NewCSRFManager() *CSRFManager {
	manager := &CSRFManager{
		tokens: make(map[string]*CSRFToken),
		stopCh: make(chan struct{}),
	}

	// Start background cleanup goroutine (fixes #785 - now has shutdown coordination)
	go manager.cleanupExpiredTokens()

	return manager
}

// Stop gracefully shuts down the CSRF manager's cleanup goroutine (fixes #785).
// This should be called during application shutdown to prevent goroutine leaks.
func (m *CSRFManager) Stop() {
	close(m.stopCh)
}

// GenerateToken creates a new CSRF token for the given session/user.
// The sessionID should be derived from the user's JWT or session cookie.
func (m *CSRFManager) GenerateToken(sessionID string) (string, error) {
	// Generate cryptographically secure random bytes
	tokenBytes := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	m.mu.Lock()
	defer m.mu.Unlock()

	// Store the token with expiry
	m.tokens[sessionID] = &CSRFToken{
		Token:     token,
		ExpiresAt: time.Now().Add(CSRFTokenExpiry),
	}

	return token, nil
}

// ValidateToken checks if the provided token is valid for the given session.
// Uses constant-time comparison to prevent timing attacks.
func (m *CSRFManager) ValidateToken(sessionID, token string) error {
	if token == "" {
		return ErrCSRFTokenMissing
	}

	m.mu.RLock()
	storedToken, exists := m.tokens[sessionID]
	m.mu.RUnlock()

	if !exists {
		return ErrCSRFTokenInvalid
	}

	// Check expiry
	if time.Now().After(storedToken.ExpiresAt) {
		// Clean up expired token
		m.mu.Lock()
		delete(m.tokens, sessionID)
		m.mu.Unlock()
		return ErrCSRFTokenExpired
	}

	// Constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare([]byte(token), []byte(storedToken.Token)) != 1 {
		return ErrCSRFTokenInvalid
	}

	return nil
}

// RevokeToken removes a CSRF token, typically on logout.
func (m *CSRFManager) RevokeToken(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tokens, sessionID)
}

// cleanupExpiredTokens periodically removes expired tokens (fixes #785 - respects shutdown signal).
func (m *CSRFManager) cleanupExpiredTokens() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			// Shutdown signal received, exit goroutine
			return
		case <-ticker.C:
			m.mu.Lock()
			now := time.Now()
			for sessionID, token := range m.tokens {
				if now.After(token.ExpiresAt) {
					delete(m.tokens, sessionID)
				}
			}
			m.mu.Unlock()
		}
	}
}

// CSRFMiddleware returns HTTP middleware that validates CSRF tokens on state-changing requests.
// It exempts GET, HEAD, OPTIONS, and TRACE methods as they should be safe/idempotent.
func (m *CSRFManager) CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF check for safe methods (RFC 7231)
		if r.Method == http.MethodGet ||
			r.Method == http.MethodHead ||
			r.Method == http.MethodOptions ||
			r.Method == http.MethodTrace {
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF for non-API routes (static files, etc.)
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Skip CSRF for auth endpoints that don't have a session yet, and SSO
		if r.URL.Path == "/api/auth/login" ||
			r.URL.Path == "/api/setup/status" ||
			r.URL.Path == "/api/setup/complete" ||
			strings.HasPrefix(r.URL.Path, "/api/sso/") {
			next.ServeHTTP(w, r)
			return
		}

		// Extract session ID from request (use username from JWT)
		sessionID := r.Header.Get("X-Username")
		if sessionID == "" {
			// Fall back to trying to get from cookie/header
			sessionID = getSessionIDFromRequest(r)
		}

		if sessionID == "" {
			slog.Warn("CSRF validation failed: no session ID",
				"path", r.URL.Path,
				"method", r.Method)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get CSRF token from request header
		token := r.Header.Get(CSRFHeaderName)

		// Validate the token
		if err := m.ValidateToken(sessionID, token); err != nil {
			slog.Warn("CSRF validation failed",
				"path", r.URL.Path,
				"method", r.Method,
				"error", err)

			switch {
			case errors.Is(err, ErrCSRFTokenMissing):
				http.Error(w, "CSRF token required", http.StatusForbidden)
			case errors.Is(err, ErrCSRFTokenExpired):
				http.Error(w, "CSRF token expired", http.StatusForbidden)
			default:
				http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getSessionIDFromRequest attempts to extract a session identifier from the request.
func getSessionIDFromRequest(r *http.Request) string {
	// Try to get username from header (set by auth middleware)
	if username := r.Header.Get("X-Username"); username != "" {
		return username
	}

	// Try to extract from JWT token in cookie
	token, _ := GetTokenFromRequest(r)
	if token != "" {
		// Use the first part of the token as a session identifier
		// This is a simplified approach - in production you'd decode the JWT
		parts := strings.Split(token, ".")
		if len(parts) >= 2 {
			return parts[1] // Use payload part as identifier
		}
	}

	return ""
}

