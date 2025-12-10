// Package auth handles JWT authentication.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidCredentials is returned when username/password is incorrect.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrInvalidToken is returned when the JWT token is invalid.
	ErrInvalidToken = errors.New("invalid token")
	// ErrTokenExpired is returned when the JWT token has expired.
	ErrTokenExpired = errors.New("token expired")
	// ErrWeakPassword is returned when a password doesn't meet strength requirements.
	ErrWeakPassword = errors.New("password does not meet strength requirements")
)

// PasswordRequirements describes minimum password requirements.
const (
	MinPasswordLength = 8
)

// Claims represents the JWT claims.
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Manager handles authentication operations.
type Manager struct {
	jwtSecret      []byte
	sessionTimeout time.Duration
	passwordHash   string
	username       string
}

// NewManager creates a new authentication manager.
func NewManager(jwtSecret string, sessionTimeout time.Duration, username, passwordHash string) *Manager {
	secret := jwtSecret
	if secret == "" {
		// Generate a random secret if not provided
		secret = generateRandomSecret()
	}

	return &Manager{
		jwtSecret:      []byte(secret),
		sessionTimeout: sessionTimeout,
		passwordHash:   passwordHash,
		username:       username,
	}
}

// generateRandomSecret creates a cryptographically secure random JWT secret.
// Note: This generates a new secret on each server restart, which will invalidate
// existing tokens. For persistent sessions across restarts, configure jwt_secret in the config file.
func generateRandomSecret() string {
	bytes := make([]byte, 32) // 256-bit key
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a deterministic but unique-per-run secret if crypto/rand fails
		// This should never happen on modern systems
		return "netscope-fallback-" + time.Now().Format(time.RFC3339Nano)
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// Authenticate validates credentials and returns a JWT token.
func (m *Manager) Authenticate(username, password string) (string, error) {
	if username != m.username {
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(m.passwordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return m.generateToken(username)
}

// generateToken creates a new JWT token.
func (m *Manager) generateToken(username string) (string, error) {
	now := time.Now()
	claims := &Claims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.sessionTimeout)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "netscope",
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.jwtSecret)
}

// ValidateToken validates a JWT token and returns the claims.
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
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

	return claims, nil
}

// HashPassword creates a bcrypt hash of a password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Middleware returns an HTTP middleware that validates JWT tokens.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Development bypass: set NETSCOPE_NO_AUTH=1 to disable auth checks.
		if os.Getenv("NETSCOPE_NO_AUTH") == "1" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for login endpoint
		if r.URL.Path == "/api/auth/login" {
			next.ServeHTTP(w, r)
			return
		}

		// Skip auth for static files
		if !strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasPrefix(r.URL.Path, "/ws") {
			next.ServeHTTP(w, r)
			return
		}

		var tokenString string

		// For WebSocket connections, check query parameter first (browsers can't send headers)
		if strings.HasPrefix(r.URL.Path, "/ws") {
			tokenString = r.URL.Query().Get("token")
		}

		// If no query token, check Authorization header
		if tokenString == "" {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Parse Bearer token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
				return
			}
			tokenString = parts[1]
		}

		claims, err := m.ValidateToken(tokenString)
		if err != nil {
			if errors.Is(err, ErrTokenExpired) {
				http.Error(w, "Token expired", http.StatusUnauthorized)
				return
			}
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add claims to request context
		r.Header.Set("X-Username", claims.Username)
		next.ServeHTTP(w, r)
	})
}

// ValidatePasswordStrength checks if a password meets minimum requirements.
// Requirements: at least 8 characters, contains uppercase, lowercase, and digit.
func ValidatePasswordStrength(password string) error {
	if len(password) < MinPasswordLength {
		return ErrWeakPassword
	}

	var hasUpper, hasLower, hasDigit bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsDigit(c):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return ErrWeakPassword
	}

	return nil
}

// GenerateSecurePassword creates a cryptographically secure random password.
// The password will contain uppercase, lowercase, and digits.
func GenerateSecurePassword(length int) (string, error) {
	if length < MinPasswordLength {
		length = MinPasswordLength
	}

	// Character sets for password generation
	const (
		lowerChars = "abcdefghijklmnopqrstuvwxyz"
		upperChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digitChars = "0123456789"
		allChars   = lowerChars + upperChars + digitChars
	)

	// Ensure at least one of each required type
	password := make([]byte, length)

	// Get random bytes for character selection
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	// Ensure we have at least one of each required type
	password[0] = lowerChars[randomBytes[0]%byte(len(lowerChars))]
	password[1] = upperChars[randomBytes[1]%byte(len(upperChars))]
	password[2] = digitChars[randomBytes[2]%byte(len(digitChars))]

	// Fill the rest randomly from all characters
	for i := 3; i < length; i++ {
		password[i] = allChars[randomBytes[i]%byte(len(allChars))]
	}

	// Shuffle the password to randomize positions of required characters
	for i := len(password) - 1; i > 0; i-- {
		j := int(randomBytes[i]) % (i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password), nil
}

// GenerateJWTSecret creates a cryptographically secure JWT signing secret.
func GenerateJWTSecret() string {
	return generateRandomSecret()
}

// InitialCredentials holds the generated initial credentials for display.
type InitialCredentials struct {
	Username     string
	Password     string
	PasswordHash string
	JWTSecret    string
}

// GenerateInitialCredentials creates new secure credentials for first-boot setup.
func GenerateInitialCredentials(username string) (*InitialCredentials, error) {
	password, err := GenerateSecurePassword(16)
	if err != nil {
		return nil, err
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	return &InitialCredentials{
		Username:     username,
		Password:     password,
		PasswordHash: hash,
		JWTSecret:    GenerateJWTSecret(),
	}, nil
}

// IsDefaultPasswordHash checks if the given hash matches the default "netscope" password.
// This is used to detect if credentials have been changed from the insecure default.
func IsDefaultPasswordHash(hash string) bool {
	// The default hash for "netscope" - if this matches, credentials are insecure
	defaultHashes := []string{
		"$2y$10$1w5ktZnNS0UxbOvHKH2.hu01jsPh2RjkszVsP.7jR5cOZYa4oAI52",
	}

	for _, defaultHash := range defaultHashes {
		if hash == defaultHash {
			return true
		}
	}

	// Also check by actually comparing against "netscope"
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("netscope")); err == nil {
		return true
	}

	return false
}
