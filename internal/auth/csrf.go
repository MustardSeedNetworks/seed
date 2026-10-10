package auth

import (
	"net/http"
	"strings"

	"github.com/MustardSeedNetworks/foundation/pkg/csrf"
)

// CSRFCookieName is the cookie name for CSRF tokens.
const CSRFCookieName = "csrf_token"

// GetSessionIDFromRequest derives the CSRF session key from the request's
// authenticated JWT. The JWT is extracted the same way the auth middleware
// reads it (cookie, then Authorization header), then hashed via foundation's
// SessionKey so the bearer plaintext is never stored in the manager. Returns ""
// when the request carries no JWT. Exported for the route registry's CSRF
// session key and the CSRF token endpoint, which must derive the same key.
func GetSessionIDFromRequest(r *http.Request) string {
	token, _ := GetTokenFromRequest(r)
	// Only a JWT-shaped bearer (a browser session token) is CSRF-relevant. A
	// resolved API token is exempted by the registry's session key on the
	// IsAPITokenAuth marker (#2450); any other non-JWT bearer has not
	// authenticated at all.
	const jwtMinParts = 2
	if len(strings.Split(token, ".")) < jwtMinParts {
		return ""
	}
	return csrf.SessionKey(token)
}
