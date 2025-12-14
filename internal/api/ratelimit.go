// Package api provides the HTTP/WebSocket server.
package api

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a simple in-memory rate limiter for login attempts.
// It tracks failed attempts per IP and blocks requests that exceed the limit.
type RateLimiter struct {
	mu        sync.RWMutex
	attempts  map[string]*attemptInfo
	limit     int           // Maximum attempts allowed
	window    time.Duration // Time window for rate limiting
	blockTime time.Duration // How long to block after exceeding limit
	cleanup   time.Duration // How often to clean up old entries
	stopCh    chan struct{}
}

type attemptInfo struct {
	count     int
	firstSeen time.Time
	blocked   bool
	blockedAt time.Time
}

// RateLimitConfig holds configuration for the rate limiter.
type RateLimitConfig struct {
	MaxAttempts int           // Maximum login attempts per window (default: 5)
	Window      time.Duration // Time window for rate limiting (default: 15 minutes)
	BlockTime   time.Duration // How long to block after exceeding limit (default: 15 minutes)
}

// DefaultRateLimitConfig returns the default rate limiting configuration.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		MaxAttempts: 5,
		Window:      15 * time.Minute,
		BlockTime:   15 * time.Minute,
	}
}

// NewRateLimiter creates a new rate limiter with the given configuration.
func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 5
	}
	if cfg.Window <= 0 {
		cfg.Window = 15 * time.Minute
	}
	if cfg.BlockTime <= 0 {
		cfg.BlockTime = 15 * time.Minute
	}

	rl := &RateLimiter{
		attempts:  make(map[string]*attemptInfo),
		limit:     cfg.MaxAttempts,
		window:    cfg.Window,
		blockTime: cfg.BlockTime,
		cleanup:   5 * time.Minute,
		stopCh:    make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// cleanupLoop periodically removes old entries from the rate limiter.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.doCleanup()
		case <-rl.stopCh:
			return
		}
	}
}

// doCleanup removes entries that have expired.
func (rl *RateLimiter) doCleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, info := range rl.attempts {
		// Remove if window has expired and not blocked
		if !info.blocked && now.Sub(info.firstSeen) > rl.window {
			delete(rl.attempts, ip)
			continue
		}
		// Remove if block has expired (use blockTime, not window)
		if info.blocked && now.Sub(info.blockedAt) > rl.blockTime {
			delete(rl.attempts, ip)
		}
	}
}

// Stop stops the rate limiter cleanup goroutine.
func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
}

// IsBlocked checks if an IP is currently blocked.
func (rl *RateLimiter) IsBlocked(ip string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	info, exists := rl.attempts[ip]
	if !exists {
		return false
	}

	// Check if block has expired (use blockTime, not window)
	if info.blocked {
		return time.Since(info.blockedAt) <= rl.blockTime
	}

	return false
}

// RecordAttempt records a login attempt from an IP.
// Returns true if the IP is now blocked, false otherwise.
func (rl *RateLimiter) RecordAttempt(ip string, success bool) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	info, exists := rl.attempts[ip]
	if !exists {
		info = &attemptInfo{
			firstSeen: now,
		}
		rl.attempts[ip] = info
	}

	// Reset if window has expired
	if now.Sub(info.firstSeen) > rl.window && !info.blocked {
		info.count = 0
		info.firstSeen = now
		info.blocked = false
	}

	// If already blocked, check if block has expired (use blockTime, not window)
	if info.blocked {
		if now.Sub(info.blockedAt) > rl.blockTime {
			// Unblock but start fresh
			info.blocked = false
			info.count = 0
			info.firstSeen = now
		} else {
			return true // Still blocked
		}
	}

	// Successful login resets the counter
	if success {
		delete(rl.attempts, ip)
		return false
	}

	// Record failed attempt
	info.count++

	// Block if exceeded limit
	if info.count >= rl.limit {
		info.blocked = true
		info.blockedAt = now
		return true
	}

	return false
}

// RemainingAttempts returns how many attempts are left for an IP.
func (rl *RateLimiter) RemainingAttempts(ip string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	info, exists := rl.attempts[ip]
	if !exists {
		return rl.limit
	}

	// Check if window has expired
	if time.Since(info.firstSeen) > rl.window && !info.blocked {
		return rl.limit
	}

	if info.blocked {
		return 0
	}

	remaining := rl.limit - info.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetClientIP extracts the client IP from a request for rate limiting.
// SECURITY: Always uses RemoteAddr to prevent X-Forwarded-For spoofing.
// Attackers can trivially bypass rate limiting by spoofing X-Forwarded-For
// headers if we trust them. For rate limiting purposes, we must use the
// actual TCP connection source (RemoteAddr).
//
// If LuminetIQ is behind a trusted reverse proxy, the proxy should be
// configured to overwrite X-Forwarded-For and LuminetIQ's RemoteAddr will
// reflect the proxy's IP. For more sophisticated setups, consider using
// a dedicated reverse proxy that handles rate limiting.
func GetClientIP(r *http.Request) string {
	// Always use RemoteAddr for rate limiting to prevent header spoofing bypass
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
