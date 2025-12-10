package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const (
	testUsername     = "admin"
	testPassword     = "netscope"
	testPasswordHash = "$2a$10$Dmw4tbpvJ3hoxg4ln8fl6uUCnhUIeXBm7Xy6txdvgwNAjhtYgzmsi"
)

func TestNewManager(t *testing.T) {
	m := NewManager("test-secret", time.Hour, testUsername, testPasswordHash)

	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.username != testUsername {
		t.Errorf("expected username %s, got %s", testUsername, m.username)
	}
	if m.sessionTimeout != time.Hour {
		t.Errorf("expected timeout 1h, got %v", m.sessionTimeout)
	}
}

func TestNewManagerWithEmptySecret(t *testing.T) {
	m := NewManager("", time.Hour, testUsername, testPasswordHash)

	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if len(m.jwtSecret) == 0 {
		t.Error("expected generated JWT secret, got empty")
	}
}

func TestAuthenticate(t *testing.T) {
	m := NewManager("test-secret", time.Hour, testUsername, testPasswordHash)

	tests := []struct {
		name     string
		username string
		password string
		wantErr  error
	}{
		{
			name:     "valid credentials",
			username: testUsername,
			password: testPassword,
			wantErr:  nil,
		},
		{
			name:     "wrong username",
			username: "wronguser",
			password: testPassword,
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "wrong password",
			username: testUsername,
			password: "wrongpassword",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "empty username",
			username: "",
			password: testPassword,
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "empty password",
			username: testUsername,
			password: "",
			wantErr:  ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := m.Authenticate(tt.username, tt.password)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				if token != "" {
					t.Error("expected empty token on error")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if token == "" {
					t.Error("expected token, got empty string")
				}
			}
		})
	}
}

func TestValidateToken(t *testing.T) {
	m := NewManager("test-secret", time.Hour, testUsername, testPasswordHash)

	// Generate valid token
	token, err := m.Authenticate(testUsername, testPassword)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	// Validate token
	claims, err := m.ValidateToken(token)
	if err != nil {
		t.Fatalf("failed to validate token: %v", err)
	}

	if claims.Username != testUsername {
		t.Errorf("expected username %s, got %s", testUsername, claims.Username)
	}
}

func TestValidateInvalidToken(t *testing.T) {
	m := NewManager("test-secret", time.Hour, testUsername, testPasswordHash)

	tests := []struct {
		name    string
		token   string
		wantErr error
	}{
		{
			name:    "empty token",
			token:   "",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "malformed token",
			token:   "not.a.valid.token",
			wantErr: ErrInvalidToken,
		},
		{
			name:    "wrong secret token",
			token:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluIiwiZXhwIjoxOTk5OTk5OTk5fQ.invalid",
			wantErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := m.ValidateToken(tt.token)

			if err != tt.wantErr {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}
			if claims != nil {
				t.Error("expected nil claims on error")
			}
		})
	}
}

func TestValidateExpiredToken(t *testing.T) {
	// Create manager with very short timeout
	m := NewManager("test-secret", time.Millisecond, testUsername, testPasswordHash)

	token, err := m.Authenticate(testUsername, testPassword)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	_, err = m.ValidateToken(token)
	if err != ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("testpassword")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("expected non-empty hash")
	}
	if hash == "testpassword" {
		t.Error("hash should not equal plaintext password")
	}
}

func TestMiddleware(t *testing.T) {
	m := NewManager("test-secret", time.Hour, testUsername, testPasswordHash)

	// Get a valid token
	token, err := m.Authenticate(testUsername, testPassword)
	if err != nil {
		t.Fatalf("failed to authenticate: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := m.Middleware(handler)

	tests := []struct {
		name           string
		path           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "valid token",
			path:           "/api/test",
			authHeader:     "Bearer " + token,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "missing auth header",
			path:           "/api/test",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid auth format",
			path:           "/api/test",
			authHeader:     "InvalidFormat",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid token",
			path:           "/api/test",
			authHeader:     "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "skip auth for login",
			path:           "/api/auth/login",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "skip auth for static files",
			path:           "/static/index.html",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, http.NoBody)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rec := httptest.NewRecorder()
			middleware.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid strong password",
			password: "SecurePass1",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "Short1",
			wantErr:  true,
		},
		{
			name:     "no uppercase",
			password: "lowercase123",
			wantErr:  true,
		},
		{
			name:     "no lowercase",
			password: "UPPERCASE123",
			wantErr:  true,
		},
		{
			name:     "no digits",
			password: "NoDigitsHere",
			wantErr:  true,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
		{
			name:     "exactly 8 chars with requirements",
			password: "Abcdef1!",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrength(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePasswordStrength(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestGenerateSecurePassword(t *testing.T) {
	// Test password generation
	password, err := GenerateSecurePassword(16)
	if err != nil {
		t.Fatalf("failed to generate password: %v", err)
	}

	if len(password) != 16 {
		t.Errorf("expected password length 16, got %d", len(password))
	}

	// Generated password should pass validation
	if err := ValidatePasswordStrength(password); err != nil {
		t.Errorf("generated password failed validation: %v", err)
	}

	// Generate another password and ensure it's different
	password2, err := GenerateSecurePassword(16)
	if err != nil {
		t.Fatalf("failed to generate second password: %v", err)
	}

	if password == password2 {
		t.Error("two generated passwords should not be identical")
	}
}

func TestGenerateSecurePasswordMinLength(t *testing.T) {
	// Request a password shorter than minimum
	password, err := GenerateSecurePassword(4)
	if err != nil {
		t.Fatalf("failed to generate password: %v", err)
	}

	// Should be at least MinPasswordLength
	if len(password) < MinPasswordLength {
		t.Errorf("expected minimum password length %d, got %d", MinPasswordLength, len(password))
	}
}

func TestGenerateInitialCredentials(t *testing.T) {
	creds, err := GenerateInitialCredentials("admin")
	if err != nil {
		t.Fatalf("failed to generate initial credentials: %v", err)
	}

	if creds.Username != "admin" {
		t.Errorf("expected username 'admin', got %q", creds.Username)
	}

	if creds.Password == "" {
		t.Error("password should not be empty")
	}

	if creds.PasswordHash == "" {
		t.Error("password hash should not be empty")
	}

	if creds.JWTSecret == "" {
		t.Error("JWT secret should not be empty")
	}

	// Verify the hash matches the password
	if err := ValidatePasswordStrength(creds.Password); err != nil {
		t.Errorf("generated password failed validation: %v", err)
	}
}

func TestIsDefaultPasswordHash(t *testing.T) {
	// The known default hash for "netscope"
	defaultHash := "$2y$10$1w5ktZnNS0UxbOvHKH2.hu01jsPh2RjkszVsP.7jR5cOZYa4oAI52"

	if !IsDefaultPasswordHash(defaultHash) {
		t.Error("expected known default hash to be detected")
	}

	// Generate a new hash for "netscope" - should also be detected
	newHash, err := HashPassword("netscope")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if !IsDefaultPasswordHash(newHash) {
		t.Error("expected new hash of 'netscope' to be detected as default")
	}

	// Generate a hash for a different password - should NOT be detected
	secureHash, err := HashPassword("SecurePassword123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if IsDefaultPasswordHash(secureHash) {
		t.Error("expected secure password hash to NOT be detected as default")
	}

	// Empty hash should not be detected as default
	if IsDefaultPasswordHash("") {
		t.Error("empty hash should not be detected as default")
	}
}

func TestGenerateJWTSecret(t *testing.T) {
	secret1 := GenerateJWTSecret()
	secret2 := GenerateJWTSecret()

	if secret1 == "" {
		t.Error("JWT secret should not be empty")
	}

	if secret1 == secret2 {
		t.Error("two generated JWT secrets should not be identical")
	}

	// Should be base64 encoded and reasonably long
	if len(secret1) < 32 {
		t.Errorf("JWT secret seems too short: %d characters", len(secret1))
	}
}
