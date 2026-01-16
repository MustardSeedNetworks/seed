package auth

import (
	"context"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

const blacklistCleanupInterval = 5 * time.Minute

// TokenBlacklist stores revoked tokens until they expire.
// Uses [sync.Map] for concurrent access without locking overhead.
type TokenBlacklist struct {
	tokens sync.Map
	ctx    context.Context
	cancel context.CancelFunc
}

// NewTokenBlacklist creates a new token blacklist with context-based cleanup.
func NewTokenBlacklist() *TokenBlacklist {
	ctx, cancel := context.WithCancel(context.Background())
	bl := &TokenBlacklist{
		tokens: sync.Map{},
		ctx:    ctx,
		cancel: cancel,
	}
	// Start cleanup goroutine to remove expired tokens.
	go bl.cleanupLoop()
	return bl
}

// Add adds a token to the blacklist.
// The token remains blacklisted until its expiration time.
func (bl *TokenBlacklist) Add(tokenID string, expiresAt time.Time) {
	if tokenID == "" {
		return
	}
	bl.tokens.Store(tokenID, expiresAt)
}

// IsBlacklisted checks if a token ID is in the blacklist.
func (bl *TokenBlacklist) IsBlacklisted(tokenID string) bool {
	if tokenID == "" {
		return false
	}
	_, exists := bl.tokens.Load(tokenID)
	return exists
}

// Remove removes a token from the blacklist (for testing).
func (bl *TokenBlacklist) Remove(tokenID string) {
	bl.tokens.Delete(tokenID)
}

// Cleanup removes all expired tokens from the blacklist.
// This is exposed for testing purposes.
func (bl *TokenBlacklist) Cleanup() {
	bl.cleanup()
}

// Stop gracefully shuts down the blacklist by stopping the cleanup goroutine.
func (bl *TokenBlacklist) Stop() {
	bl.cancel()
}

// cleanupLoop periodically removes expired tokens from the blacklist.
func (bl *TokenBlacklist) cleanupLoop() {
	ticker := time.NewTicker(blacklistCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-bl.ctx.Done():
			logging.GetLogger().Debug("Token blacklist cleanup goroutine stopping")
			return
		case <-ticker.C:
			bl.cleanup()
		}
	}
}

// cleanup removes all expired tokens.
func (bl *TokenBlacklist) cleanup() {
	now := time.Now()
	bl.tokens.Range(func(key, value any) bool {
		if expiresAt, ok := value.(time.Time); ok {
			if now.After(expiresAt) {
				bl.tokens.Delete(key)
			}
		}
		return true
	})
}
