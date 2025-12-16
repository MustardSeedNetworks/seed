// Package i18n provides internationalization support.
package i18n

import (
	"context"
	"net/http"
)

// contextKey is used for storing values in request context.
type contextKey string

const (
	// LocalizerKey is the context key for the localizer.
	LocalizerKey contextKey = "i18n.localizer"
)

// Middleware creates HTTP middleware that extracts the Accept-Language header
// and attaches a Localizer to the request context.
//
// Usage:
//
//	router.Use(i18n.Middleware())
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//	    localizer := i18n.FromContext(r.Context())
//	    msg := localizer.T("errors.auth.invalidCredentials")
//	}
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get language from Accept-Language header
			lang := r.Header.Get("Accept-Language")
			if lang == "" {
				lang = DefaultLanguage
			}

			// Create localizer and add to context
			localizer := NewLocalizer(lang)
			ctx := context.WithValue(r.Context(), LocalizerKey, localizer)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FromContext retrieves the Localizer from the request context.
// Returns a default English localizer if not found.
func FromContext(ctx context.Context) *Localizer {
	if localizer, ok := ctx.Value(LocalizerKey).(*Localizer); ok {
		return localizer
	}
	return NewLocalizer(DefaultLanguage)
}

// FromRequest is a convenience function to get the Localizer from an HTTP request.
func FromRequest(r *http.Request) *Localizer {
	return FromContext(r.Context())
}
