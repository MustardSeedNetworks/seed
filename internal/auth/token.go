// token.go carries JWT minting, validation and revocation for package auth.
// The Manager that owns them is in auth.go.

package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Claims represents the JWT claims.
type Claims struct {
	jwt.RegisteredClaims

	Username     string `json:"username"`
	TokenVersion int    `json:"token_version"` // For token revocation (fixes #525)
	TokenType    string `json:"token_type"`    // "access" or "refresh"
	ClientID     string `json:"client_id"`     // Owning tenant; see client_context.go
}

// GenerateToken creates a new JWT token for the given username.
// This is primarily used for testing. For production use, use Authenticate().
func (m *Manager) GenerateToken(ctx context.Context, username string) (string, error) {
	return m.generateTokenWithType(ctx, username, "access", m.sessionTimeout)
}

// GenerateAccessToken creates a short-lived access token (fixes #478).
func (m *Manager) GenerateAccessToken(ctx context.Context, username string) (string, error) {
	return m.generateTokenWithType(ctx, username, "access", AccessTokenDuration)
}

// GenerateRefreshToken creates a long-lived refresh token (fixes #478).
func (m *Manager) GenerateRefreshToken(ctx context.Context, username string) (string, error) {
	return m.generateTokenWithType(ctx, username, "refresh", RefreshTokenDuration)
}

// generateTokenWithType creates a JWT token with specified type and duration.
func (m *Manager) generateTokenWithType(
	ctx context.Context,
	username, tokenType string,
	duration time.Duration,
) (string, error) {
	// Read token version with lock (fixes #520, #525)
	m.mu.RLock()
	currentVersion := m.tokenVersion
	userStore := m.userStore
	m.mu.RUnlock()

	// If we have a UserStore, get the token version from the database
	// This ensures tokens are generated with the correct version (fixes #927)
	//
	// The client claim resolves from the same store. Unlike the token version
	// there is no in-memory fallback to degrade to, so a store that cannot
	// answer fails the mint rather than issuing a token that claims the
	// default tenant on a deployment that has more than one.
	clientID := DefaultClientID
	if userStore != nil && username != "" {
		if dbVersion, versionErr := userStore.GetTokenVersion(ctx, username); versionErr == nil {
			currentVersion = dbVersion
		}
		// On error, fall back to in-memory version

		resolved, clientErr := userStore.GetClientID(ctx, username)
		if clientErr != nil {
			return "", fmt.Errorf("failed to resolve client for %q: %w", username, clientErr)
		}
		clientID = resolved
	}

	// A unique id per mint. Without it every claim is identical for two logins
	// by the same user in the same second -- NewNumericDate truncates to
	// seconds -- so the tokens are byte-identical. The blacklist keys on
	// sha256 of the whole token, so revoking one session would revoke every
	// other session minted in that second (#2214).
	tokenUniqueID, idErr := newTokenID()
	if idErr != nil {
		return "", fmt.Errorf("failed to generate token id: %w", idErr)
	}

	now := time.Now()
	claims := &Claims{
		Username:     username,
		TokenVersion: currentVersion, // Include version for revocation
		TokenType:    tokenType,
		ClientID:     clientID,
		ID:           tokenUniqueID,
		ExpiresAt:    jwt.NewNumericDate(now.Add(duration)),
		IssuedAt:     jwt.NewNumericDate(now),
		NotBefore:    jwt.NewNumericDate(now),
		Issuer:       "The Seed",
		Subject:      username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %w", err)
	}
	return signedToken, nil
}

// ValidateToken validates a JWT token and returns the claims.
// Checks token version to support revocation (fixes #525).
// Also checks the token blacklist for immediate revocation (ported from Stem).
// If UserStore is set, queries database for current token version.
func (m *Manager) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	// Check blacklist first (fast path for revoked tokens)
	tokenID := tokenFingerprint(tokenString)
	if m.blacklist != nil && m.blacklist.IsBlacklisted(tokenID) {
		logging.GetLogger().InfoContext(ctx, "Token is blacklisted", "token_id", tokenID[:8]+"...")
		return nil, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.jwtSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check token version for revocation (fixes #525)
	m.mu.RLock()
	currentVersion := m.tokenVersion
	userStore := m.userStore
	m.mu.RUnlock()

	// If we have a UserStore, get current token version from database
	if userStore != nil && claims.Username != "" {
		dbVersion, versionErr := userStore.GetTokenVersion(ctx, claims.Username)
		if versionErr == nil {
			currentVersion = dbVersion
		}
		// On error, fall back to in-memory version
	}

	if claims.TokenVersion < currentVersion {
		logging.GetLogger().
			InfoContext(ctx, "Token revoked", "version", claims.TokenVersion, "current", currentVersion)
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token and returns the claims (fixes #478).
// Ensures the token is actually a refresh token, not an access token.
func (m *Manager) ValidateRefreshToken(ctx context.Context, tokenString string) (*Claims, error) {
	claims, err := m.ValidateToken(ctx, tokenString)
	if err != nil {
		return nil, err
	}

	// Ensure it's a refresh token
	if claims.TokenType != "refresh" {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// RefreshAccessToken generates a new access token from a valid refresh token (fixes #478).
// This allows short-lived access tokens with long-lived refresh tokens.
// Enforces maximum session lifetime to prevent indefinite sessions (fixes #717).
func (m *Manager) RefreshAccessToken(ctx context.Context, refreshToken string) (string, error) {
	claims, err := m.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", err
	}

	// Check if session has exceeded maximum lifetime (fixes #717)
	// The IssuedAt claim represents when the refresh token (and thus the session) was created
	if claims.IssuedAt != nil {
		sessionAge := time.Since(claims.IssuedAt.Time)
		if sessionAge > MaxSessionLifetime {
			logging.GetLogger().InfoContext(ctx, "Session exceeded maximum lifetime",
				"age", sessionAge,
				"max", MaxSessionLifetime,
				"username", claims.Username)
			return "", ErrTokenExpired
		}
	}

	// Generate new access token with same username
	return m.GenerateAccessToken(ctx, claims.Username)
}

// tokenFingerprint generates a short, unique identifier for a JWT token.
// Uses SHA-256 hash of the token string for efficient blacklist storage.
// newTokenID returns a random JWT id (jti), making every minted token unique
// even when two are issued in the same second with otherwise identical claims.
func newTokenID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

func tokenFingerprint(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// RevokeToken adds a token to the blacklist for immediate revocation.
// The token remains blacklisted until its natural expiration time.
// This is used on logout to ensure the token cannot be reused.
func (m *Manager) RevokeToken(tokenString string) {
	if m.blacklist == nil || tokenString == "" {
		return
	}

	// Parse the token to get its expiration time
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(_ *jwt.Token) (any, error) {
		return m.jwtSecret, nil
	})
	if err != nil {
		// Even if parsing fails, blacklist with a default expiration
		m.blacklist.Add(tokenFingerprint(tokenString), time.Now().Add(AccessTokenDuration))
		return
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || claims.ExpiresAt == nil {
		m.blacklist.Add(tokenFingerprint(tokenString), time.Now().Add(AccessTokenDuration))
		return
	}

	// Add to blacklist with the token's actual expiration time
	m.blacklist.Add(tokenFingerprint(tokenString), claims.ExpiresAt.Time)
}
