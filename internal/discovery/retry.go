// Package discovery provides network device discovery functionality.
// This file implements retry logic with exponential backoff for transient failures.
package discovery

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"
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
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		JitterPercent: 0.2,
	}
}

// FastRetryConfig returns config for quick operations that should retry fast.
func FastRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    2,
		InitialDelay:  50 * time.Millisecond,
		MaxDelay:      500 * time.Millisecond,
		BackoffFactor: 2.0,
		JitterPercent: 0.1,
	}
}

// SNMPRetryConfig returns config optimized for SNMP operations.
func SNMPRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  500 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		BackoffFactor: 2.0,
		JitterPercent: 0.25,
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
			slog.Debug("Error not retryable, stopping",
				"error", err,
				"attempt", attempt+1)
			break
		}

		// Calculate delay with exponential backoff
		delay := calculateDelay(cfg, attempt)

		slog.Debug("Retrying operation",
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
			sLower[i] = c + 32
		} else {
			sLower[i] = c
		}
	}

	substrLower := make([]byte, len(substr))
	for i := range len(substr) {
		c := substr[i]
		if c >= 'A' && c <= 'Z' {
			substrLower[i] = c + 32
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
		MaxRetries:    3,
		InitialDelay:  200 * time.Millisecond,
		MaxDelay:      3 * time.Second,
		BackoffFactor: 2.0,
		JitterPercent: 0.15,
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
