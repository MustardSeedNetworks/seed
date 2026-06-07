package api

// server_middleware.go contains the http.Handler middlewares applied to the
// main server mux: security headers, CORS (with null-origin rejection), body
// size limits, and panic recovery.

import (
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// securityHeadersMiddleware adds security headers to all responses.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HSTS (HTTP Strict Transport Security) - only set over HTTPS
		if r.TLS != nil {
			// max-age=31536000 (1 year), includeSubDomains
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// XSS protection (legacy header, but doesn't hurt)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy - strict policy without unsafe-inline (fixes #532)
		w.Header().
			Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self' ws: wss:; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'")

		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers with origin validation.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Reject null Origin header to prevent CORS bypass attacks (fixes #709)
		// Null origins can occur in sandboxed iframes or redirected requests
		if origin == "null" {
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusForbidden)
			} else {
				logger := logging.FromContext(r.Context())
				localizer := i18n.FromRequest(r)
				message := localizer.T("errors.security.nullOriginForbidden")
				sendErrorResponseWithDetails(
					w,
					logger,
					http.StatusForbidden,
					ErrCodeForbidden,
					message,
					"",
				) // fixes #694
			}
			return
		}

		// Allow requests from same origin (no Origin header) or validated origins
		if origin == "" || isAllowedOrigin(origin) {
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().
				Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			// Cache preflight requests for 24 hours to reduce overhead (fixes #531)
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// bodyLimitMiddleware caps request bodies on NON-API paths (a DoS backstop for
// the SPA/static + infra routes). Per-route body limits for /api/v1 are
// authoritative in the capability registry (route.maxBodyBytes, applied by
// register()), so a path-switch here would be a second, drift-prone source of
// truth — the registry is the single source (ADR-0002).
func bodyLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, APIVersionPrefix) {
			r.Body = http.MaxBytesReader(w, r.Body, MaxBodySizeDefault)
		}
		next.ServeHTTP(w, r)
	})
}

// recoverMiddleware recovers from panics in HTTP handlers (fixes #519).
// Prevents a single panic from crashing the entire server.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logging.GetLogger().ErrorContext(r.Context(), "PANIC in handler",
					"method", r.Method,
					"path", r.URL.Path,
					"error", err,
					"stack", string(debug.Stack()))
				logger := logging.FromContext(r.Context())
				localizer := i18n.FromRequest(r)
				sendErrorResponseWithDetails(
					w,
					logger,
					http.StatusInternalServerError,
					ErrCodeInternal,
					localizer.T("errors.security.panicRecovered"),
					"",
				) // fixes #694
			}
		}()
		next.ServeHTTP(w, r)
	})
}
