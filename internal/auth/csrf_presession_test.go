package auth_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/auth"
)

// TestPreSessionPathsAreCSRFExempt derives one list from the other so they
// cannot drift apart again.
//
// A path that bypasses the JWT middleware has no session, so if it is not also
// CSRF-exempt the CSRF middleware finds no session ID and answers 401. That is
// not protection — it makes the route unreachable. It is how enrolling TOTP
// became a permanent lockout: /api/v1/auth/login/totp bypassed auth, was not
// exempt from CSRF, and returned 401 to a correct code, with
// /api/v1/recovery/complete failing the same way (#2391).
func TestPreSessionPathsAreCSRFExempt(t *testing.T) {
	t.Parallel()

	for _, path := range auth.ExportPreSessionPaths() {
		if !auth.ExportIsCSRFExemptPath(path) {
			t.Errorf(
				"%s bypasses authentication but is not CSRF-exempt, so it answers 401 "+
					"and cannot be called at all — add it to isCSRFExemptPath",
				path,
			)
		}
	}
}
