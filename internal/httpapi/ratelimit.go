package httpapi

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// Rate limiting timing constants.
const (
	// rateLimitWindowMin is the default rate limiting window duration in minutes.
	rateLimitWindowMin = 15

	// rateLimitCleanupMin is the cleanup interval for rate limiter entries in minutes.
	rateLimitCleanupMin = 5

	// defaultMaxAttempts is the default maximum login attempts allowed before blocking.
	defaultMaxAttempts = 5

	// defaultEndpointMaxRequests is the default maximum requests per window for endpoint rate limiting.
	defaultEndpointMaxRequests = 5

	// Memory protection constants (ported from Stem).
	// maxVisitors is the maximum number of unique IPs to track.
	// Prevents memory exhaustion from IP spoofing attacks.
	maxVisitors = 10000

	// capacityThresholdHigh is 80% capacity - triggers moderate cleanup.
	capacityThresholdHigh = 80
	// capacityThresholdCritical is 90% capacity - triggers aggressive cleanup.
	capacityThresholdCritical = 90
	// percentDivisor is used for percentage calculations.
	percentDivisor = 100

	// defaultVisitorTTL is how long to keep a visitor's rate limiter after last access.
	defaultVisitorTTL = 15 * time.Minute
	// ttlDivisorModerate reduces TTL by half at high capacity.
	ttlDivisorModerate = 2
	// ttlDivisorAggressive reduces TTL to quarter at critical capacity.
	ttlDivisorAggressive = 4
)

// RateLimiter implements a simple in-memory rate limiter for login attempts.
// It tracks failed attempts per IP and blocks requests that exceed the limit.
// This provides protection against both single-source and distributed brute force
// attacks by rate limiting each IP address independently (fixes #716).
// Includes memory protection from Stem to prevent exhaustion from IP spoofing.
type RateLimiter struct {
	mu          sync.RWMutex
	attempts    map[string]*attemptInfo
	limit       int           // Maximum attempts allowed
	window      time.Duration // Time window for rate limiting
	blockTime   time.Duration // How long to block after exceeding limit
	cleanup     time.Duration // How often to clean up old entries
	stopCh      chan struct{}
	maxVisitors int // Maximum number of unique IPs to track (memory protection)
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
		MaxAttempts: defaultMaxAttempts,
		Window:      rateLimitWindowMin * time.Minute,
		BlockTime:   rateLimitWindowMin * time.Minute,
	}
}

// NewRateLimiter creates a new rate limiter with the given configuration.
// Includes memory protection to limit tracked IPs (ported from Stem).
func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = defaultMaxAttempts
	}
	if cfg.Window <= 0 {
		cfg.Window = rateLimitWindowMin * time.Minute
	}
	if cfg.BlockTime <= 0 {
		cfg.BlockTime = rateLimitWindowMin * time.Minute
	}

	rl := &RateLimiter{
		attempts:    make(map[string]*attemptInfo),
		limit:       cfg.MaxAttempts,
		window:      cfg.Window,
		blockTime:   cfg.BlockTime,
		cleanup:     rateLimitCleanupMin * time.Minute,
		stopCh:      make(chan struct{}),
		maxVisitors: maxVisitors, // Memory protection from Stem
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
// Uses adaptive TTL based on capacity (ported from Stem) to prevent memory exhaustion.
func (rl *RateLimiter) doCleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	visitorCount := len(rl.attempts)
	ttl := rl.window

	// Calculate capacity thresholds (adaptive cleanup from Stem).
	criticalThreshold := rl.maxVisitors * capacityThresholdCritical / percentDivisor
	highThreshold := rl.maxVisitors * capacityThresholdHigh / percentDivisor

	// Use more aggressive cleanup when approaching max capacity.
	if visitorCount > criticalThreshold {
		ttl = rl.window / ttlDivisorAggressive
	} else if visitorCount > highThreshold {
		ttl = rl.window / ttlDivisorModerate
	}

	for ip, info := range rl.attempts {
		// Remove if window has expired and not blocked (use adaptive TTL)
		if !info.blocked && now.Sub(info.firstSeen) > ttl {
			delete(rl.attempts, ip)
			continue
		}
		// Remove if block has expired (use blockTime, adjusted by capacity)
		blockTTL := rl.blockTime
		if visitorCount > criticalThreshold {
			blockTTL = rl.blockTime / ttlDivisorAggressive
		} else if visitorCount > highThreshold {
			blockTTL = rl.blockTime / ttlDivisorModerate
		}
		if info.blocked && now.Sub(info.blockedAt) > blockTTL {
			delete(rl.attempts, ip)
		}
	}

	// Log if we're still at high capacity after cleanup.
	newCount := len(rl.attempts)
	if newCount > highThreshold {
		logging.GetLogger().Warn("Login rate limiter at high capacity after cleanup",
			"visitorCount", newCount,
			"maxVisitors", rl.maxVisitors,
		)
	}
}

// Stop stops the rate limiter cleanup goroutine.
// Safe to call multiple times (fixes #844).
func (rl *RateLimiter) Stop() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.stopCh == nil {
		return // Already stopped
	}
	close(rl.stopCh)
	rl.stopCh = nil
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
// Includes memory protection from Stem to limit tracked IPs.
func (rl *RateLimiter) RecordAttempt(ip string, success bool) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	info, exists := rl.attempts[ip]
	if !exists {
		// Memory protection: if at max capacity and this is a new IP, block it
		// This prevents memory exhaustion from IP spoofing attacks (ported from Stem)
		if len(rl.attempts) >= rl.maxVisitors {
			logging.GetLogger().Warn("Login rate limiter at max capacity, blocking new IP",
				"ip", ip,
				"maxVisitors", rl.maxVisitors,
			)
			return true // Block new IPs when at capacity
		}
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
// If The Seed is behind a trusted reverse proxy, use GetClientIPWithProxy
// which checks the --trusted-proxies configuration.
func GetClientIP(r *http.Request) string {
	// Always use RemoteAddr for rate limiting to prevent header spoofing bypass
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// GetClientIPWithTrustedProxies extracts the client IP considering trusted proxies.
// If trustedProxies is nil or empty, falls back to GetClientIP (RemoteAddr only).
// If the request comes from a trusted proxy, uses X-Forwarded-For header.
func GetClientIPWithTrustedProxies(r *http.Request, tp *TrustedProxies) string {
	if tp == nil || tp.IsEmpty() {
		return GetClientIP(r)
	}
	return tp.GetClientIPWithProxy(r)
}

// EndpointRateLimiter implements rate limiting for expensive API endpoints (fixes #530).
// Uses a sliding window approach to limit requests per IP per time window.
// Includes memory protection from Stem to prevent exhaustion from IP spoofing.
type EndpointRateLimiter struct {
	mu          sync.RWMutex
	requests    map[string]*requestWindow
	maxReqs     int           // Maximum requests per window
	window      time.Duration // Time window for rate limiting
	stopCh      chan struct{}
	maxVisitors int // Maximum number of unique IPs to track (memory protection)
}

type requestWindow struct {
	timestamps []time.Time
	lastSeen   time.Time // For adaptive TTL cleanup
}

// EndpointRateLimitConfig holds configuration for endpoint rate limiting.
type EndpointRateLimitConfig struct {
	MaxRequests int           // Maximum requests per window (default: 5)
	Window      time.Duration // Time window (default: 1 minute)
}

// DefaultEndpointRateLimitConfig returns the default endpoint rate limiting configuration.
func DefaultEndpointRateLimitConfig() EndpointRateLimitConfig {
	return EndpointRateLimitConfig{
		MaxRequests: defaultEndpointMaxRequests,
		Window:      1 * time.Minute,
	}
}

// NewEndpointRateLimiter creates a new endpoint rate limiter with the given configuration.
// Includes memory protection to limit tracked IPs (ported from Stem).
func NewEndpointRateLimiter(cfg EndpointRateLimitConfig) *EndpointRateLimiter {
	if cfg.MaxRequests <= 0 {
		cfg.MaxRequests = defaultEndpointMaxRequests
	}
	if cfg.Window <= 0 {
		cfg.Window = 1 * time.Minute
	}

	erl := &EndpointRateLimiter{
		requests:    make(map[string]*requestWindow),
		maxReqs:     cfg.MaxRequests,
		window:      cfg.Window,
		stopCh:      make(chan struct{}),
		maxVisitors: maxVisitors, // Memory protection from Stem
	}

	// Start cleanup goroutine
	go erl.cleanupLoop()

	return erl
}

// cleanupLoop periodically removes old entries.
func (erl *EndpointRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rateLimitCleanupMin * time.Minute)
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
// Uses adaptive TTL based on capacity (ported from Stem) to prevent memory exhaustion.
func (erl *EndpointRateLimiter) doCleanup() {
	erl.mu.Lock()
	defer erl.mu.Unlock()

	now := time.Now()
	visitorCount := len(erl.requests)
	ttl := defaultVisitorTTL

	// Calculate capacity thresholds (adaptive cleanup from Stem).
	criticalThreshold := erl.maxVisitors * capacityThresholdCritical / percentDivisor
	highThreshold := erl.maxVisitors * capacityThresholdHigh / percentDivisor

	// Use more aggressive cleanup when approaching max capacity.
	// At 90% capacity, use quarter TTL; at 80%, use half TTL.
	if visitorCount > criticalThreshold {
		ttl = defaultVisitorTTL / ttlDivisorAggressive
	} else if visitorCount > highThreshold {
		ttl = defaultVisitorTTL / ttlDivisorModerate
	}

	cutoff := now.Add(-ttl)
	for ip, window := range erl.requests {
		// Remove timestamps outside the window
		validTimestamps := []time.Time{}
		for _, ts := range window.timestamps {
			if now.Sub(ts) <= erl.window {
				validTimestamps = append(validTimestamps, ts)
			}
		}

		// Delete if no valid timestamps OR if lastSeen is too old (adaptive TTL)
		if len(validTimestamps) == 0 || window.lastSeen.Before(cutoff) {
			delete(erl.requests, ip)
		} else {
			window.timestamps = validTimestamps
		}
	}

	// Log if we're still at high capacity after cleanup.
	newCount := len(erl.requests)
	if newCount > highThreshold {
		logging.GetLogger().Warn("Endpoint rate limiter at high capacity after cleanup",
			"visitorCount", newCount,
			"maxVisitors", erl.maxVisitors,
			"ttlUsed", ttl.String(),
		)
	}
}

// Stop stops the rate limiter cleanup goroutine.
// Safe to call multiple times (fixes #844).
func (erl *EndpointRateLimiter) Stop() {
	erl.mu.Lock()
	defer erl.mu.Unlock()

	if erl.stopCh == nil {
		return // Already stopped
	}
	close(erl.stopCh)
	erl.stopCh = nil
}

// Allow checks if a request from an IP should be allowed.
// Returns true if allowed, false if rate limit exceeded.
// Fixes #890: Pre-allocate with capacity limit to prevent unbounded memory growth.
// Includes memory protection from Stem to limit tracked IPs.
func (erl *EndpointRateLimiter) Allow(ip string) bool {
	erl.mu.Lock()
	defer erl.mu.Unlock()

	now := time.Now()

	window, exists := erl.requests[ip]
	if !exists {
		// Memory protection: reject new IPs when at max capacity (ported from Stem)
		if len(erl.requests) >= erl.maxVisitors {
			logging.GetLogger().Warn("Endpoint rate limiter at max capacity, rejecting new IP",
				"ip", ip,
				"maxVisitors", erl.maxVisitors,
			)
			return false
		}

		window = &requestWindow{
			// Pre-allocate with maxReqs capacity to avoid repeated allocations
			timestamps: make([]time.Time, 0, erl.maxReqs),
			lastSeen:   now,
		}
		erl.requests[ip] = window
	}

	// Update lastSeen for adaptive TTL cleanup
	window.lastSeen = now

	// Remove timestamps outside the window
	// Fixes #890: Reuse slice with bounded capacity instead of creating new slices
	validIdx := 0
	for _, ts := range window.timestamps {
		if now.Sub(ts) <= erl.window {
			window.timestamps[validIdx] = ts
			validIdx++
		}
	}
	window.timestamps = window.timestamps[:validIdx]

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
			logger := logging.FromContext(r.Context())
			localizer := i18n.FromRequest(r)
			sendErrorResponseWithDetails(
				w,
				logger,
				http.StatusTooManyRequests,
				ErrCodeRateLimit,
				localizer.T("errors.api.rateLimitExceeded"),
				"",
			) // fixes #694
			return
		}

		next.ServeHTTP(w, r)
	})
}
