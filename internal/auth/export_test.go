// Package auth exports internal functions for testing.
package auth

// RandomChar exports randomChar for testing.
var RandomChar = randomChar

// RandomInt exports randomInt for testing.
var RandomInt = randomInt

// CryptoRandRead exports cryptoRandRead for testing.
var CryptoRandRead = cryptoRandRead

// ExtractTokenFromSubprotocol exports extractTokenFromSubprotocol for testing.
var ExtractTokenFromSubprotocol = extractTokenFromSubprotocol

// ManagerUsername returns the username from a Manager for testing.
func (m *Manager) ManagerUsername() string {
	return m.username
}

// ManagerSessionTimeout returns the sessionTimeout from a Manager for testing.
func (m *Manager) ManagerSessionTimeout() interface{} {
	return m.sessionTimeout
}

// ManagerJWTSecret returns the jwtSecret from a Manager for testing.
func (m *Manager) ManagerJWTSecret() []byte {
	return m.jwtSecret
}

// CSRFManagerTokens returns the tokens map from a CSRFManager for testing.
// Returns a map keyed by session ID for testing purposes only.
func (c *CSRFManager) CSRFManagerTokens() map[string]*CSRFToken {
	return c.tokens
}

// CSRFManagerCtxDone returns the context Done channel for testing.
func (c *CSRFManager) CSRFManagerCtxDone() <-chan struct{} {
	return c.ctx.Done()
}
