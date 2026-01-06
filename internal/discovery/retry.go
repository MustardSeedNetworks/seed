package discovery

// This file implements retry logic with exponential backoff for transient failures.

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

// Character case conversion constant.
const asciiCaseOffset = 32 // Difference between ASCII uppercase and lowercase letters

// Retry configuration constants.
const (
	retryDefaultMaxAttempts = 3   // Default maximum retry attempts
	retryDefaultInitDelayMs = 100 // Default initial delay in milliseconds
	retryDefaultMaxDelayS   = 5   // Default maximum delay in seconds
	retryFastMaxAttempts    = 2   // Fast retry max attempts
	retryFastInitDelayMs    = 50  // Fast retry initial delay in milliseconds
	retryFastMaxDelayMs     = 500 // Fast retry max delay in milliseconds
	retrySNMPInitDelayMs    = 500 // SNMP retry initial delay in milliseconds
	retrySNMPMaxDelayS      = 10  // SNMP retry max delay in seconds

	// Backoff factor constants.
	retryBackoffFactor       = 2.0  // Default exponential backoff multiplier
	retryDefaultJitter       = 0.2  // Default jitter percentage (20%)
	retryFastJitter          = 0.1  // Fast retry jitter percentage (10%)
	retrySNMPJitter          = 0.25 // SNMP retry jitter percentage (25%)
	retryDiscoveryInitDelayM = 200  // Discovery retry initial delay in milliseconds
	retryDiscoveryMaxDelayS  = 3    // Discovery retry max delay in seconds
	retryDiscoveryJitter     = 0.15 // Discovery retry jitter percentage (15%)
)

// RetryConfig configures retry behavior for network operations.
type RetryConfig struct {
	MaxRetries      int           // Maximum number of retry attempts (0 = no retries)
	InitialDelay    time.Duration // Initial delay before first retry
	MaxDelay        time.Duration // Maximum delay between retries
	BackoffFactor   float64       // Multiplier for each subsequent retry (exponential backoff)
	JitterPercent   float64       // Random jitter as percentage of delay (0.0-1.0)
	RetryableErrors []string      // Error substrings that trigger retry (empty = retry all)
}

// DefaultRetryConfig returns sensible defaults for network operations.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    retryDefaultMaxAttempts,
		InitialDelay:  retryDefaultInitDelayMs * time.Millisecond,
		MaxDelay:      retryDefaultMaxDelayS * time.Second,
		BackoffFactor: retryBackoffFactor,
		JitterPercent: retryDefaultJitter,
	}
}

// FastRetryConfig returns config for quick operations that should retry fast.
func FastRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    retryFastMaxAttempts,
		InitialDelay:  retryFastInitDelayMs * time.Millisecond,
		MaxDelay:      retryFastMaxDelayMs * time.Millisecond,
		BackoffFactor: retryBackoffFactor,
		JitterPercent: retryFastJitter,
	}
}

// SNMPRetryConfig returns config optimized for SNMP operations.
func SNMPRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    retryDefaultMaxAttempts,
		InitialDelay:  retrySNMPInitDelayMs * time.Millisecond,
		MaxDelay:      retrySNMPMaxDelayS * time.Second,
		BackoffFactor: retryBackoffFactor,
		JitterPercent: retrySNMPJitter,
		RetryableErrors: []string{
			"timeout",
			"connection refused",
			"no route to host",
			"network unreachable",
		},
	}
}

// RetryResult captures the outcome of a retry operation.
type RetryResult struct {
	Attempts   int           // Total attempts made
	LastError  error         // Last error encountered (nil if successful)
	TotalTime  time.Duration // Total time spent including retries
	Successful bool          // Whether the operation eventually succeeded
}

// RetryWithBackoff executes an operation with exponential backoff on failure.
// The operation function should return nil on success or an error to trigger retry.
// Returns the final error if all retries exhausted, or nil on success.
func RetryWithBackoff(ctx context.Context, cfg RetryConfig, operation func() error) *RetryResult {
	result := &RetryResult{}
	start := time.Now()

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		result.Attempts = attempt + 1

		// Check context before attempting
		select {
		case <-ctx.Done():
			result.LastError = ctx.Err()
			result.TotalTime = time.Since(start)
			return result
		default:
		}

		// Execute the operation
		err := operation()
		if err == nil {
			result.Successful = true
			result.TotalTime = time.Since(start)
			return result
		}

		result.LastError = err

		// Don't retry if this was the last attempt
		if attempt >= cfg.MaxRetries {
			break
		}

		// Check if error is retryable
		if !isRetryableError(err, cfg.RetryableErrors) {
			logging.GetLogger().DebugContext(ctx, "Error not retryable, stopping",
				"error", err,
				"attempt", attempt+1)
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(cfg, attempt)

		logging.GetLogger().DebugContext(ctx, "Retrying operation",
			"attempt", attempt+1,
			"maxRetries", cfg.MaxRetries,
			"delay", delay,
			"error", err)

		// Wait with context cancellation support
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			result.LastError = ctx.Err()
			result.TotalTime = time.Since(start)
			return result
		}
	}

	result.TotalTime = time.Since(start)
	return result
}

// RetryWithBackoffResult is like RetryWithBackoff but for operations that return a value.
func RetryWithBackoffResult[T any](
	ctx context.Context,
	cfg RetryConfig,
	operation func() (T, error),
) (T, *RetryResult) {
	var finalResult T
	result := &RetryResult{}
	start := time.Now()

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		result.Attempts = attempt + 1

		// Check context before attempting
		select {
		case <-ctx.Done():
			result.LastError = ctx.Err()
			result.TotalTime = time.Since(start)
			return finalResult, result
		default:
		}

		// Execute the operation
		val, err := operation()
		if err == nil {
			result.Successful = true
			result.TotalTime = time.Since(start)
			return val, result
		}

		result.LastError = err
		finalResult = val // Keep last value even if error

		// Don't retry if this was the last attempt
		if attempt >= cfg.MaxRetries {
			break
		}

		// Check if error is retryable
		if !isRetryableError(err, cfg.RetryableErrors) {
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(cfg, attempt)

		// Wait with context cancellation support
		select {
		case <-time.After(delay):
			// Continue to next attempt
		case <-ctx.Done():
			result.LastError = ctx.Err()
			result.TotalTime = time.Since(start)
			return finalResult, result
		}
	}

	result.TotalTime = time.Since(start)
	return finalResult, result
}

// calculateDelay computes the delay for a given attempt with jitter.
func calculateDelay(cfg RetryConfig, attempt int) time.Duration {
	// Exponential backoff: delay = initialDelay * (backoffFactor ^ attempt)
	delay := float64(cfg.InitialDelay)
	for range attempt {
		delay *= cfg.BackoffFactor
	}

	// Cap at max delay
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}

	// Add jitter
	if cfg.JitterPercent > 0 {
		jitter := delay * cfg.JitterPercent * (rand.Float64()*2 - 1) // #nosec G404 -- weak RNG acceptable for timing jitter
		delay += jitter
	}

	// Ensure non-negative
	if delay < 0 {
		delay = float64(cfg.InitialDelay)
	}

	return time.Duration(delay)
}

// isRetryableError checks if an error should trigger a retry.
func isRetryableError(err error, retryableErrors []string) bool {
	if err == nil {
		return false
	}

	// If no specific errors defined, retry all errors
	if len(retryableErrors) == 0 {
		return true
	}

	errStr := err.Error()
	for _, pattern := range retryableErrors {
		if containsIgnoreCase(errStr, pattern) {
			return true
		}
	}

	return false
}

// containsIgnoreCase checks if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	// Simple case-insensitive contains
	sLower := make([]byte, len(s))
	for i := range len(s) {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			sLower[i] = c + asciiCaseOffset
		} else {
			sLower[i] = c
		}
	}

	substrLower := make([]byte, len(substr))
	for i := range len(substr) {
		c := substr[i]
		if c >= 'A' && c <= 'Z' {
			substrLower[i] = c + asciiCaseOffset
		} else {
			substrLower[i] = c
		}
	}

	// Use simple byte search
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		match := true
		for j := range substrLower {
			if sLower[i+j] != substrLower[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// NetworkRetryConfig returns config for general network operations.
func NetworkRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    retryDefaultMaxAttempts,
		InitialDelay:  retryDiscoveryInitDelayM * time.Millisecond,
		MaxDelay:      retryDiscoveryMaxDelayS * time.Second,
		BackoffFactor: retryBackoffFactor,
		JitterPercent: retryDiscoveryJitter,
		RetryableErrors: []string{
			"timeout",
			"connection refused",
			"no route to host",
			"network unreachable",
			"temporary failure",
			"i/o timeout",
			"connection reset",
		},
	}
}
