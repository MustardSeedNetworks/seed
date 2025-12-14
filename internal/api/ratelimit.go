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

// EndpointRateLimiter implements rate limiting for expensive API endpoints (fixes #530).
// Uses a sliding window approach to limit requests per IP per time window.
type EndpointRateLimiter struct {
	mu       sync.RWMutex
	requests map[string]*requestWindow
	maxReqs  int           // Maximum requests per window
	window   time.Duration // Time window for rate limiting
	stopCh   chan struct{}
}

type requestWindow struct {
	timestamps []time.Time
}

// EndpointRateLimitConfig holds configuration for endpoint rate limiting.
type EndpointRateLimitConfig struct {
	MaxRequests int           // Maximum requests per window (default: 5)
	Window      time.Duration // Time window (default: 1 minute)
}

// DefaultEndpointRateLimitConfig returns the default endpoint rate limiting configuration.
func DefaultEndpointRateLimitConfig() EndpointRateLimitConfig {
	return EndpointRateLimitConfig{
		MaxRequests: 5,
		Window:      1 * time.Minute,
	}
}

// NewEndpointRateLimiter creates a new endpoint rate limiter with the given configuration.
func NewEndpointRateLimiter(cfg EndpointRateLimitConfig) *EndpointRateLimiter {
	if cfg.MaxRequests <= 0 {
		cfg.MaxRequests = 5
	}
	if cfg.Window <= 0 {
		cfg.Window = 1 * time.Minute
	}

	erl := &EndpointRateLimiter{
		requests: make(map[string]*requestWindow),
		maxReqs:  cfg.MaxRequests,
		window:   cfg.Window,
		stopCh:   make(chan struct{}),
	}

	// Start cleanup goroutine
	go erl.cleanupLoop()

	return erl
}

// cleanupLoop periodically removes old entries.
func (erl *EndpointRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			erl.doCleanup()
		case <-erl.stopCh:
			return
		}
	}
}

// doCleanup removes expired request windows.
func (erl *EndpointRateLimiter) doCleanup() {
	erl.mu.Lock()
	defer erl.mu.Unlock()

	now := time.Now()
	for ip, window := range erl.requests {
		// Remove timestamps outside the window
		validTimestamps := []time.Time{}
		for _, ts := range window.timestamps {
			if now.Sub(ts) <= erl.window {
				validTimestamps = append(validTimestamps, ts)
			}
		}

		if len(validTimestamps) == 0 {
			delete(erl.requests, ip)
		} else {
			window.timestamps = validTimestamps
		}
	}
}

// Stop stops the rate limiter cleanup goroutine.
func (erl *EndpointRateLimiter) Stop() {
	close(erl.stopCh)
}

// Allow checks if a request from an IP should be allowed.
// Returns true if allowed, false if rate limit exceeded.
func (erl *EndpointRateLimiter) Allow(ip string) bool {
	erl.mu.Lock()
	defer erl.mu.Unlock()

	now := time.Now()

	window, exists := erl.requests[ip]
	if !exists {
		window = &requestWindow{
			timestamps: []time.Time{},
		}
		erl.requests[ip] = window
	}

	// Remove timestamps outside the window
	validTimestamps := []time.Time{}
	for _, ts := range window.timestamps {
		if now.Sub(ts) <= erl.window {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	window.timestamps = validTimestamps

	// Check if we're at the limit
	if len(window.timestamps) >= erl.maxReqs {
		return false
	}

	// Record this request
	window.timestamps = append(window.timestamps, now)
	return true
}

// RateLimitMiddleware returns HTTP middleware that rate limits expensive endpoints.
func (erl *EndpointRateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := GetClientIP(r)

		if !erl.Allow(ip) {
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
