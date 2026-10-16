package api

import (
	"net/http"
)

// ExportIsAllowedOrigin exposes isAllowedOrigin for testing.
func ExportIsAllowedOrigin(origin string) bool {
	return isAllowedOrigin(origin)
}

// ExportIsRFC1918Origin exposes isRFC1918Origin for testing.
func ExportIsRFC1918Origin(origin string) bool {
	return isRFC1918Origin(origin)
}

// ExportExtractHostFromOrigin exposes extractHostFromOrigin for testing.
func ExportExtractHostFromOrigin(origin string) (string, bool) {
	return extractHostFromOrigin(origin)
}

// ExportIsLocalhostAddress exposes isLocalhostAddress for testing.
func ExportIsLocalhostAddress(host string) bool {
	return isLocalhostAddress(host)
}

// ExportIsPrivateNetworkAddress exposes isPrivateNetworkAddress for testing.
func ExportIsPrivateNetworkAddress(host string) bool {
	return isPrivateNetworkAddress(host)
}

// ExportIsValidClassCAddress exposes isValidClassCAddress for testing.
func ExportIsValidClassCAddress(host string) bool {
	return isValidClassCAddress(host)
}

// ExportIsValidClassAAddress exposes isValidClassAAddress for testing.
func ExportIsValidClassAAddress(host string) bool {
	return isValidClassAAddress(host)
}

// ExportIsValidClassBAddress exposes isValidClassBAddress for testing.
func ExportIsValidClassBAddress(host string) bool {
	return isValidClassBAddress(host)
}

// SetAllowedOrigins sets the allowed origins for CORS in testing.
func SetAllowedOrigins(origins []string) {
	getOriginState().setAllowedOrigins(origins)
}

// ClearAllowedOrigins clears the allowed origins for testing.
func ClearAllowedOrigins() {
	getOriginState().setAllowedOrigins(nil)
}

// ExportSecurityHeadersMiddleware exposes securityHeadersMiddleware for testing.
func ExportSecurityHeadersMiddleware(next http.Handler) http.Handler {
	return securityHeadersMiddleware(next)
}

// ExportCORSMiddleware exposes corsMiddleware for testing.
func ExportCORSMiddleware(next http.Handler) http.Handler {
	return corsMiddleware(next)
}
