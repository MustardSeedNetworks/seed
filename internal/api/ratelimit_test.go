// Package api provides the HTTP/WebSocket server.
// Test suite validates rate limiter configuration, token bucket behavior, and HTTP middleware.
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	if rl == nil {
		t.Fatal("NewRateLimiter returned nil")
	}
	if rl.limit != cfg.MaxAttempts {
		t.Errorf("expected limit %d, got %d", cfg.MaxAttempts, rl.limit)
	}
}

func TestRateLimiterNotBlockedInitially(t *testing.T) {
	rl := NewRateLimiter(DefaultRateLimitConfig())
	defer rl.Stop()

	if rl.IsBlocked("192.168.1.1") {
		t.Error("new IP should not be blocked")
	}
}

func TestRateLimiterRemainingAttempts(t *testing.T) {
	cfg := RateLimitConfig{
		MaxAttempts: 5,
		Window:      15 * time.Minute,
		BlockTime:   15 * time.Minute,
	}
	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	ip := "192.168.1.100"

	// Should start with max attempts
	if remaining := rl.RemainingAttempts(ip); remaining != 5 {
		t.Errorf("expected 5 remaining attempts, got %d", remaining)
	}

	// Record a failed attempt
	rl.RecordAttempt(ip, false)
	if remaining := rl.RemainingAttempts(ip); remaining != 4 {
		t.Errorf("expected 4 remaining attempts after 1 failure, got %d", remaining)
	}
}

func TestRateLimiterBlocksAfterMaxAttempts(t *testing.T) {
	cfg := RateLimitConfig{
		MaxAttempts: 3,
		Window:      15 * time.Minute,
		BlockTime:   15 * time.Minute,
	}
	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	ip := "10.0.0.1"

	// Record failures up to the limit
	for i := range 3 {
		blocked := rl.RecordAttempt(ip, false)
		if i < 2 && blocked {
			t.Errorf("should not be blocked after %d attempts", i+1)
		}
		if i == 2 && !blocked {
			t.Error("should be blocked after reaching limit")
		}
	}

	// Verify blocked
	if !rl.IsBlocked(ip) {
		t.Error("IP should be blocked after max attempts")
	}
	if remaining := rl.RemainingAttempts(ip); remaining != 0 {
		t.Errorf("expected 0 remaining attempts when blocked, got %d", remaining)
	}
}

func TestRateLimiterSuccessfulLoginClearsAttempts(t *testing.T) {
	cfg := RateLimitConfig{
		MaxAttempts: 5,
		Window:      15 * time.Minute,
		BlockTime:   15 * time.Minute,
	}
	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	ip := "172.16.0.50"

	// Record some failed attempts
	rl.RecordAttempt(ip, false)
	rl.RecordAttempt(ip, false)
	if remaining := rl.RemainingAttempts(ip); remaining != 3 {
		t.Errorf("expected 3 remaining after 2 failures, got %d", remaining)
	}

	// Successful login should reset
	rl.RecordAttempt(ip, true)
	if remaining := rl.RemainingAttempts(ip); remaining != 5 {
		t.Errorf("expected 5 remaining after successful login, got %d", remaining)
	}
}

func TestRateLimiterDifferentIPsAreIndependent(t *testing.T) {
	cfg := RateLimitConfig{
		MaxAttempts: 3,
		Window:      15 * time.Minute,
		BlockTime:   15 * time.Minute,
	}
	rl := NewRateLimiter(cfg)
	defer rl.Stop()

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// Block IP1
	for range 3 {
		rl.RecordAttempt(ip1, false)
	}

	if !rl.IsBlocked(ip1) {
		t.Error("IP1 should be blocked")
	}
	if rl.IsBlocked(ip2) {
		t.Error("IP2 should not be blocked")
	}
	if remaining := rl.RemainingAttempts(ip2); remaining != 3 {
		t.Errorf("IP2 should have all attempts remaining, got %d", remaining)
	}
}

func TestGetClientIP(t *testing.T) {
	// Tests verify that GetClientIP always uses RemoteAddr (never trusts headers)
	// to prevent X-Forwarded-For spoofing attacks that could bypass rate limiting.
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			name:       "remote address only",
			remoteAddr: "192.168.1.100:12345",
			headers:    nil,
			expected:   "192.168.1.100",
		},
		{
			name:       "X-Forwarded-For ignored for security",
			remoteAddr: "127.0.0.1:8080",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1"},
			expected:   "127.0.0.1", // Must use RemoteAddr, NOT the spoofable header
		},
		{
			name:       "X-Forwarded-For multiple IPs ignored",
			remoteAddr: "127.0.0.1:8080",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1, 10.0.0.2, 10.0.0.3"},
			expected:   "127.0.0.1", // Must use RemoteAddr
		},
		{
			name:       "X-Real-IP header ignored for security",
			remoteAddr: "127.0.0.1:8080",
			headers:    map[string]string{"X-Real-IP": "172.16.0.50"},
			expected:   "127.0.0.1", // Must use RemoteAddr
		},
		{
			name:       "all headers ignored for security",
			remoteAddr: "192.168.1.1:8080",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
				"X-Real-IP":       "172.16.0.50",
			},
			expected: "192.168.1.1", // Must use RemoteAddr
		},
		{
			name:       "remote address without port",
			remoteAddr: "192.168.1.100",
			headers:    nil,
			expected:   "192.168.1.100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/auth/login", http.NoBody)
			req.RemoteAddr = tt.remoteAddr
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := GetClientIP(req)
			if got != tt.expected {
				t.Errorf("GetClientIP() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRateLimiterStop(_ *testing.T) {
	rl := NewRateLimiter(DefaultRateLimitConfig())

	// Record some attempts
	rl.RecordAttempt("1.2.3.4", false)

	// Stop should not panic
	rl.Stop()
}

func TestDefaultRateLimitConfig(t *testing.T) {
	cfg := DefaultRateLimitConfig()

	if cfg.MaxAttempts != 5 {
		t.Errorf("expected MaxAttempts 5, got %d", cfg.MaxAttempts)
	}
	if cfg.Window != 15*time.Minute {
		t.Errorf("expected Window 15m, got %v", cfg.Window)
	}
	if cfg.BlockTime != 15*time.Minute {
		t.Errorf("expected BlockTime 15m, got %v", cfg.BlockTime)
	}
}
