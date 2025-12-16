package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/krisarmstrong/luminetiq/internal/auth"
	"github.com/krisarmstrong/luminetiq/internal/config"
)

// TestHandleRefreshToken tests the token refresh endpoint.
func TestHandleRefreshToken(t *testing.T) {
	tests := []struct {
		name           string
		setupCookie    bool
		cookieValue    string
		expectedStatus int
		expectToken    bool
	}{
		{
			name:           "successful refresh with valid token",
			setupCookie:    true,
			cookieValue:    "", // Will be set to valid refresh token
			expectedStatus: http.StatusOK,
			expectToken:    true,
		},
		{
			name:           "fails without refresh token cookie",
			setupCookie:    false,
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
		{
			name:           "fails with invalid refresh token",
			setupCookie:    true,
			cookieValue:    "invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTestServer()

			// Generate a valid refresh token for tests that need it
			var refreshToken string
			if tt.setupCookie && tt.cookieValue == "" {
				var err error
				refreshToken, err = server.authManager.GenerateRefreshToken("admin")
				if err != nil {
					t.Fatalf("Failed to generate refresh token: %v", err)
				}
			} else {
				refreshToken = tt.cookieValue
			}

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", http.NoBody)

			// Add refresh token cookie if needed
			if tt.setupCookie {
				req.AddCookie(&http.Cookie{
					Name:  auth.CookieNameRefresh,
					Value: refreshToken,
				})
			}

			// Execute request
			w := httptest.NewRecorder()
			server.handleRefreshToken(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check for new access token in response
			if tt.expectToken {
				var resp LoginResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if resp.Token == "" {
					t.Error("Expected token in response, got empty string")
				}

				if resp.Expires == 0 {
					t.Error("Expected expires in response, got 0")
				}

				// Verify new access token cookie was set
				cookies := w.Result().Cookies()
				var foundAccessCookie bool
				for _, cookie := range cookies {
					if cookie.Name == auth.CookieNameAccess {
						foundAccessCookie = true
						if cookie.Value == "" {
							t.Error("Access token cookie has empty value")
						}
						if !cookie.HttpOnly {
							t.Error("Access token cookie should be HttpOnly")
						}
						break
					}
				}
				if !foundAccessCookie {
					t.Error("Expected access token cookie in response")
				}
			}
		})
	}
}

// TestHandleRefreshTokenMethodNotAllowed tests non-POST methods.
func TestHandleRefreshTokenMethodNotAllowed(t *testing.T) {
	server := NewTestServer()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/auth/refresh", http.NoBody)
			w := httptest.NewRecorder()

			server.handleRefreshToken(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d",
					http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleLogout tests the logout endpoint.
func TestHandleLogout(t *testing.T) {
	server := NewTestServer()

	// Create authenticated request
	token, err := server.authManager.GenerateToken("admin")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	server.handleLogout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify cookies were cleared
	cookies := w.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == auth.CookieNameAccess || cookie.Name == auth.CookieNameRefresh {
			if cookie.MaxAge != -1 {
				t.Errorf("Expected cookie %s to have MaxAge -1, got %d", cookie.Name, cookie.MaxAge)
			}
			if !cookie.Expires.Before(time.Now()) {
				t.Errorf("Expected cookie %s to be expired", cookie.Name)
			}
		}
	}

	// Verify JSON response
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "logged out" {
		t.Errorf("Expected status 'logged out', got '%s'", resp["status"])
	}
}

// TestHandleLogoutMethodNotAllowed tests non-POST methods for logout.
func TestHandleLogoutMethodNotAllowed(t *testing.T) {
	server := NewTestServer()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/auth/logout", http.NoBody)
			w := httptest.NewRecorder()

			server.handleLogout(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d",
					http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleSetupComplete tests the setup completion endpoint.
func TestHandleSetupComplete(t *testing.T) {
	tests := []struct {
		name           string
		password       string
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "valid strong password",
			password:       "MySecure123!Pass",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "weak password - too short",
			password:       "Short1!",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "weak password - no uppercase",
			password:       "mysecure123!pass",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "weak password - no special char",
			password:       "MySecure123Pass",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "empty password",
			password:       "",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server with config that needs setup
			cfg := config.DefaultConfig()
			cfg.Auth.DefaultPasswordHash = "$2y$10$1w5ktZnNS0UxbOvHKH2.hu01jsPh2RjkszVsP.7jR5cOZYa4oAI52" // Default "netscope" hash

			server := NewTestServerWithConfig(cfg)

			// Create request
			reqBody := SetupCompleteRequest{
				Password: tt.password,
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/setup/complete", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.handleSetupComplete(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response: %s",
					tt.expectedStatus, w.Code, w.Body.String())
			}

			if !tt.expectError && w.Code == http.StatusOK {
				var resp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}
				if resp["status"] != "success" {
					t.Errorf("Expected status 'success', got '%s'", resp["status"])
				}
			}
		})
	}
}

// TestHandleSetupStatus tests the setup status endpoint.
func TestHandleSetupStatus(t *testing.T) {
	tests := []struct {
		name           string
		useDefaultHash bool
		expectSetup    bool
	}{
		{
			name:           "needs setup with default password",
			useDefaultHash: true,
			expectSetup:    true,
		},
		{
			name:           "setup complete with custom password",
			useDefaultHash: false,
			expectSetup:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			if tt.useDefaultHash {
				cfg.Auth.DefaultPasswordHash = "$2y$10$1w5ktZnNS0UxbOvHKH2.hu01jsPh2RjkszVsP.7jR5cOZYa4oAI52"
			} else {
				// Use a different hash (for "different-password")
				cfg.Auth.DefaultPasswordHash = "$2a$10$abcdefghijklmnopqrstuuabcdefghijklmnopqrstuv"
			}

			server := NewTestServerWithConfig(cfg)

			req := httptest.NewRequest(http.MethodGet, "/api/setup/status", http.NoBody)
			w := httptest.NewRecorder()

			server.handleSetupStatus(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			var resp SetupStatusResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if resp.NeedsSetup != tt.expectSetup {
				t.Errorf("Expected needsSetup=%v, got %v", tt.expectSetup, resp.NeedsSetup)
			}

			if tt.expectSetup && resp.SuggestedPassword == "" {
				t.Error("Expected suggested password when setup is needed")
			}

			if !tt.expectSetup && resp.SuggestedPassword != "" {
				t.Error("Expected no suggested password when setup is complete")
			}
		})
	}
}
