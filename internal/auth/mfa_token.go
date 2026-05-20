// mfa_token.go — short-lived "MFA pending" token used between the
// password step and the TOTP/passkey step of an interactive login.
//
// The token is a JWT signed with the same secret as the main session
// tokens but carries an "mfa_pending=true" claim so it cannot be
// confused with a real access token. It expires after MFAPendingTTL
// (5 minutes) and is rejected by Authenticate's normal validation
// because the TokenType field is "mfa_pending".
//
// Wave 3 (task #85).

package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MFAPendingTTL is how long an MFA pending token is valid. After this
// the user must restart the login flow from the password step.
const MFAPendingTTL = 5 * time.Minute

// mfaPendingTokenType is the value used in Claims.TokenType for MFA
// pending tokens. It must not collide with the regular access/refresh
// token types so ValidateToken (used by API middleware) keeps rejecting
// these tokens for ordinary API access.
const mfaPendingTokenType = "mfa_pending"

// ErrInvalidMFAToken is returned when the supplied MFA pending token
// is missing, expired, of the wrong type, or its signature is bad.
var ErrInvalidMFAToken = errors.New("invalid mfa pending token")

// MFAPendingClaims extends the standard claims with a token-type guard
// so MFA pending tokens cannot be replayed against the JWT middleware.
type MFAPendingClaims struct {
	jwt.RegisteredClaims

	Username  string `json:"username"`
	TokenType string `json:"token_type"`
}

// GenerateMFAPendingToken issues a 5-minute single-purpose token tied
// to username. It does NOT bump the user's token_version — the regular
// access token won't be issued until the user passes the second factor.
func (m *Manager) GenerateMFAPendingToken(_ context.Context, username string) (string, error) {
	if username == "" {
		return "", fmt.Errorf("%w: empty username", ErrInvalidMFAToken)
	}

	now := time.Now()
	claims := &MFAPendingClaims{
		Username:  username,
		TokenType: mfaPendingTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(MFAPendingTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "The Seed",
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign mfa pending token: %w", err)
	}
	return signed, nil
}

// ValidateMFAPendingToken parses tokenString and returns the username
// it represents on success. It rejects tokens of the wrong type so a
// stolen access token cannot be replayed here, and vice versa.
func (m *Manager) ValidateMFAPendingToken(tokenString string) (string, error) {
	if tokenString == "" {
		return "", ErrInvalidMFAToken
	}

	parsed, err := jwt.ParseWithClaims(
		tokenString, &MFAPendingClaims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, ErrInvalidMFAToken
			}
			return m.jwtSecret, nil
		},
	)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrInvalidMFAToken, err)
	}

	claims, ok := parsed.Claims.(*MFAPendingClaims)
	if !ok || !parsed.Valid {
		return "", ErrInvalidMFAToken
	}
	if claims.TokenType != mfaPendingTokenType {
		return "", ErrInvalidMFAToken
	}
	if claims.Username == "" {
		return "", ErrInvalidMFAToken
	}
	return claims.Username, nil
}
