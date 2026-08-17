package api

// Request body size limits for different request types.
// These constants prevent memory exhaustion attacks by limiting payload sizes.
const (
	// MaxBodySizeAuth is for login and authentication requests (1 KB).
	// Small limit since auth payloads only contain username/password.
	MaxBodySizeAuth int64 = 1 * 1024

	// MaxBodySizeConfig is for configuration updates (64 KB).
	// Moderate limit for settings and config JSON.
	MaxBodySizeConfig int64 = 64 * 1024

	// MaxBodySizeJSON is for general JSON API requests (256 KB).
	// Standard limit for most API endpoints.
	MaxBodySizeJSON int64 = 256 * 1024

	// MaxBodySizeDefault is the default limit for unspecified endpoints (1 MB).
	// Safe default for general use.
	MaxBodySizeDefault int64 = 1 * 1024 * 1024
)
