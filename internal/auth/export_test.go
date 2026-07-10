package auth

import (
	"os"
)

// ExportRandomChar exports randomChar for testing.
func ExportRandomChar(chars string) (byte, error) {
	return randomChar(chars)
}

// SetHIBPEndpointForTest swaps the HIBP endpoint URL (used to point at
// a [httptest.Server]) via the SEED_HIBP_ENDPOINT_TEST environment
// variable. Returns a restore function the test should defer.
//
// We use the env-var seam rather than a package-level mutable variable
// so that the production code path keeps no global state (satisfying
// gochecknoglobals) while still letting tests redirect requests.
func SetHIBPEndpointForTest(url string) func() {
	orig, hadOrig := os.LookupEnv(hibpEnvEndpoint)
	_ = os.Setenv(hibpEnvEndpoint, url)
	return func() {
		if hadOrig {
			_ = os.Setenv(hibpEnvEndpoint, orig)
			return
		}
		_ = os.Unsetenv(hibpEnvEndpoint)
	}
}

// ExportRandomInt exports randomInt for testing.
func ExportRandomInt(n int) (int, error) {
	return randomInt(n)
}

// ExportCryptoRandRead exports cryptoRandRead for testing.
func ExportCryptoRandRead(b []byte, operation string) error {
	return cryptoRandRead(b, operation)
}

// ExportExtractTokenFromSubprotocol exports extractTokenFromSubprotocol for testing.
func ExportExtractTokenFromSubprotocol(protocols string) string {
	return extractTokenFromSubprotocol(protocols)
}

// ManagerUsername returns the username from a Manager for testing.
func (m *Manager) ManagerUsername() string {
	return m.username
}

// ExportIsCSRFExemptPath exports isCSRFExemptPath so the CSRF exempt-list
// coverage gate (#1223) can pin the policy from the external test package.
func ExportIsCSRFExemptPath(path string) bool {
	return isCSRFExemptPath(path)
}

// ManagerSessionTimeout returns the sessionTimeout from a Manager for testing.
func (m *Manager) ManagerSessionTimeout() any {
	return m.sessionTimeout
}

// ManagerJWTSecret returns the jwtSecret from a Manager for testing.
func (m *Manager) ManagerJWTSecret() []byte {
	return m.jwtSecret
}
