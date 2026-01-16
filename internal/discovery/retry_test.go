package discovery_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestDefaultRetryConfig(t *testing.T) {
	cfg := discovery.DefaultRetryConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries should be 3, got %d", cfg.MaxRetries)
	}
	if cfg.InitialDelay != 100*time.Millisecond {
		t.Errorf("InitialDelay should be 100ms, got %v", cfg.InitialDelay)
	}
	if cfg.MaxDelay != 5*time.Second {
		t.Errorf("MaxDelay should be 5s, got %v", cfg.MaxDelay)
	}
	if cfg.BackoffFactor != 2.0 {
		t.Errorf("BackoffFactor should be 2.0, got %f", cfg.BackoffFactor)
	}
	if cfg.JitterPercent != 0.2 {
		t.Errorf("JitterPercent should be 0.2, got %f", cfg.JitterPercent)
	}
}

func TestFastRetryConfig(t *testing.T) {
	cfg := discovery.FastRetryConfig()

	if cfg.MaxRetries != 2 {
		t.Errorf("MaxRetries should be 2, got %d", cfg.MaxRetries)
	}
	if cfg.InitialDelay != 50*time.Millisecond {
		t.Errorf("InitialDelay should be 50ms, got %v", cfg.InitialDelay)
	}
	if cfg.MaxDelay != 500*time.Millisecond {
		t.Errorf("MaxDelay should be 500ms, got %v", cfg.MaxDelay)
	}
}

func TestSNMPRetryConfig(t *testing.T) {
	cfg := discovery.SNMPRetryConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries should be 3, got %d", cfg.MaxRetries)
	}
	if cfg.InitialDelay != 500*time.Millisecond {
		t.Errorf("InitialDelay should be 500ms, got %v", cfg.InitialDelay)
	}
	if cfg.MaxDelay != 10*time.Second {
		t.Errorf("MaxDelay should be 10s, got %v", cfg.MaxDelay)
	}
	if len(cfg.RetryableErrors) == 0 {
		t.Error("RetryableErrors should have entries")
	}
}

func TestNetworkRetryConfig(t *testing.T) {
	cfg := discovery.NetworkRetryConfig()

	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries should be 3, got %d", cfg.MaxRetries)
	}
	if len(cfg.RetryableErrors) == 0 {
		t.Error("RetryableErrors should have entries")
	}
}

func TestRetryWithBackoff_Success(t *testing.T) {
	ctx := context.Background()
	cfg := discovery.RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}

	callCount := 0
	result := discovery.RetryWithBackoff(ctx, cfg, func() error {
		callCount++
		return nil // Success on first attempt
	})

	if !result.Successful {
		t.Error("Operation should be successful")
	}
	if result.Attempts != 1 {
		t.Errorf("Should have 1 attempt, got %d", result.Attempts)
	}
	if result.LastError != nil {
		t.Errorf("LastError should be nil, got %v", result.LastError)
	}
	if callCount != 1 {
		t.Errorf("Operation should be called once, got %d", callCount)
	}
}

func TestRetryWithBackoff_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	cfg := discovery.RetryConfig{
		MaxRetries:    3,
		InitialDelay:  10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		BackoffFactor: 2.0,
	}

	callCount := 0
	result := discovery.RetryWithBackoff(ctx, cfg, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil // Success on 3rd attempt
	})

	if !result.Successful {
		t.Error("Operation should be successful")
	}
	if result.Attempts != 3 {
		t.Errorf("Should have 3 attempts, got %d", result.Attempts)
	}
	if callCount != 3 {
		t.Errorf("Operation should be called 3 times, got %d", callCount)
	}
	if result.TotalTime == 0 {
		t.Error("TotalTime should be non-zero")
	}
}

func TestRetryWithBackoff_AllRetriesFail(t *testing.T) {
	ctx := context.Background()
	cfg := discovery.RetryConfig{
		MaxRetries:   2,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     50 * time.Millisecond,
	}

	callCount := 0
	testErr := errors.New("persistent error")
	result := discovery.RetryWithBackoff(ctx, cfg, func() error {
		callCount++
		return testErr
	})

	if result.Successful {
		t.Error("Operation should have failed")
	}
	if result.Attempts != 3 { // Initial + 2 retries
		t.Errorf("Should have 3 attempts, got %d", result.Attempts)
	}
	if result.LastError == nil {
		t.Error("LastError should be set")
	}
	if result.LastError.Error() != testErr.Error() {
		t.Errorf("LastError should be %q, got %q", testErr.Error(), result.LastError.Error())
	}
	if callCount != 3 {
		t.Errorf("Operation should be called 3 times, got %d", callCount)
	}
}

func TestRetryWithBackoff_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := discovery.RetryConfig{
		MaxRetries:   5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
	}

	callCount := 0
	result := discovery.RetryWithBackoff(ctx, cfg, func() error {
		callCount++
		if callCount == 1 {
			cancel() // Cancel after first attempt
			return errors.New("error")
		}
		return nil
	})

	if result.Successful {
		t.Error("Operation should have failed due to cancellation")
	}
	if !errors.Is(result.LastError, context.Canceled) {
		t.Errorf("LastError should be context.Canceled, got %v", result.LastError)
	}
}

func TestRetryWithBackoff_ContextCancelledBeforeAttempt(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before starting

	cfg := discovery.RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
	}

	callCount := 0
	result := discovery.RetryWithBackoff(ctx, cfg, func() error {
		callCount++
		return nil
	})

	if result.Successful {
		t.Error("Operation should have failed")
	}
	if !errors.Is(result.LastError, context.Canceled) {
		t.Errorf("LastError should be context.Canceled, got %v", result.LastError)
	}
	if callCount != 0 {
		t.Errorf("Operation should not be called if context is already cancelled, got %d calls", callCount)
	}
}

func TestRetryWithBackoff_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := discovery.RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		RetryableErrors: []string{"timeout", "temporary"},
	}

	callCount := 0
	result := discovery.RetryWithBackoff(ctx, cfg, func() error {
		callCount++
		return errors.New("permanent error") // Not in RetryableErrors
	})

	if result.Successful {
		t.Error("Operation should have failed")
	}
	if result.Attempts != 1 {
		t.Errorf("Should stop after 1 attempt for non-retryable error, got %d", result.Attempts)
	}
}

func TestRetryWithBackoff_RetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := discovery.RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        50 * time.Millisecond,
		RetryableErrors: []string{"timeout", "temporary"},
	}

	callCount := 0
	result := discovery.RetryWithBackoff(ctx, cfg, func() error {
		callCount++
		if callCount < 3 {
			return errors.New("connection timeout") // Contains "timeout"
		}
		return nil
	})

	if !result.Successful {
		t.Error("Operation should be successful after retries")
	}
	if result.Attempts != 3 {
		t.Errorf("Should have 3 attempts, got %d", result.Attempts)
	}
}

func TestRetryWithBackoff_CaseInsensitiveMatch(t *testing.T) {
	ctx := context.Background()
	cfg := discovery.RetryConfig{
		MaxRetries:      2,
		InitialDelay:    10 * time.Millisecond,
		RetryableErrors: []string{"TIMEOUT"},
	}

	callCount := 0
	result := discovery.RetryWithBackoff(ctx, cfg, func() error {
		callCount++
		if callCount == 1 {
			return errors.New("connection Timeout occurred") // Mixed case
		}
		return nil
	})

	if !result.Successful {
		t.Error("Should retry on case-insensitive match")
	}
	if result.Attempts != 2 {
		t.Errorf("Should have 2 attempts, got %d", result.Attempts)
	}
}

func TestRetryWithBackoff_NoRetries(t *testing.T) {
	ctx := context.Background()
	cfg := discovery.RetryConfig{
		MaxRetries: 0, // No retries
	}

	callCount := 0
	result := discovery.RetryWithBackoff(ctx, cfg, func() error {
		callCount++
		return errors.New("error")
	})

	if result.Successful {
		t.Error("Operation should have failed")
	}
	if result.Attempts != 1 {
		t.Errorf("Should have exactly 1 attempt, got %d", result.Attempts)
	}
}

func TestRetryWithBackoffResult_Success(t *testing.T) {
	ctx := context.Background()
	cfg := discovery.RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
	}

	value, result := discovery.RetryWithBackoffResult(ctx, cfg, func() (string, error) {
		return "success value", nil
	})

	if !result.Successful {
		t.Error("Operation should be successful")
	}
	if value != "success value" {
		t.Errorf("Value should be 'success value', got %q", value)
	}
}

func TestRetryWithBackoffResult_SuccessAfterRetries(t *testing.T) {
	ctx := context.Background()
	cfg := discovery.RetryConfig{
		MaxRetries:   3,
		InitialDelay: 10 * time.Millisecond,
	}

	callCount := 0
	value, result := discovery.RetryWithBackoffResult(ctx, cfg, func() (int, error) {
		callCount++
		if callCount < 2 {
			return 0, errors.New("temporary")
		}
		return 42, nil
	})

	if !result.Successful {
		t.Error("Operation should be successful")
	}
	if value != 42 {
		t.Errorf("Value should be 42, got %d", value)
	}
	if result.Attempts != 2 {
		t.Errorf("Should have 2 attempts, got %d", result.Attempts)
	}
}

func TestRetryWithBackoffResult_Failure(t *testing.T) {
	ctx := context.Background()
	cfg := discovery.RetryConfig{
		MaxRetries:   2,
		InitialDelay: 10 * time.Millisecond,
	}

	value, result := discovery.RetryWithBackoffResult(ctx, cfg, func() (string, error) {
		return "partial", errors.New("always fails")
	})

	if result.Successful {
		t.Error("Operation should have failed")
	}
	// Should keep last value even on failure
	if value != "partial" {
		t.Errorf("Value should be 'partial', got %q", value)
	}
}

func TestRetryWithBackoffResult_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := discovery.RetryConfig{
		MaxRetries:   5,
		InitialDelay: 100 * time.Millisecond,
	}

	callCount := 0
	_, result := discovery.RetryWithBackoffResult(ctx, cfg, func() (int, error) {
		callCount++
		if callCount == 1 {
			cancel()
			return 0, errors.New("error")
		}
		return 0, nil
	})

	if result.Successful {
		t.Error("Operation should have failed due to cancellation")
	}
	if !errors.Is(result.LastError, context.Canceled) {
		t.Errorf("LastError should be context.Canceled, got %v", result.LastError)
	}
}

func TestRetryWithBackoffResult_NonRetryableError(t *testing.T) {
	ctx := context.Background()
	cfg := discovery.RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		RetryableErrors: []string{"timeout"},
	}

	callCount := 0
	_, result := discovery.RetryWithBackoffResult(ctx, cfg, func() (int, error) {
		callCount++
		return 0, errors.New("permanent failure")
	})

	if result.Successful {
		t.Error("Operation should have failed")
	}
	if result.Attempts != 1 {
		t.Errorf("Should stop after 1 attempt for non-retryable error, got %d", result.Attempts)
	}
}

func TestRetryResult_Fields(t *testing.T) {
	result := &discovery.RetryResult{
		Attempts:   3,
		LastError:  errors.New("test error"),
		TotalTime:  150 * time.Millisecond,
		Successful: false,
	}

	if result.Attempts != 3 {
		t.Errorf("Attempts should be 3, got %d", result.Attempts)
	}
	if result.LastError == nil {
		t.Error("LastError should be set")
	}
	if result.TotalTime != 150*time.Millisecond {
		t.Errorf("TotalTime should be 150ms, got %v", result.TotalTime)
	}
	if result.Successful {
		t.Error("Successful should be false")
	}
}

func TestRetryConfig_Fields(t *testing.T) {
	cfg := discovery.RetryConfig{
		MaxRetries:      5,
		InitialDelay:    200 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		BackoffFactor:   1.5,
		JitterPercent:   0.25,
		RetryableErrors: []string{"timeout", "connection refused"},
	}

	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries should be 5, got %d", cfg.MaxRetries)
	}
	if cfg.InitialDelay != 200*time.Millisecond {
		t.Errorf("InitialDelay should be 200ms, got %v", cfg.InitialDelay)
	}
	if cfg.MaxDelay != 10*time.Second {
		t.Errorf("MaxDelay should be 10s, got %v", cfg.MaxDelay)
	}
	if cfg.BackoffFactor != 1.5 {
		t.Errorf("BackoffFactor should be 1.5, got %f", cfg.BackoffFactor)
	}
	if cfg.JitterPercent != 0.25 {
		t.Errorf("JitterPercent should be 0.25, got %f", cfg.JitterPercent)
	}
	if len(cfg.RetryableErrors) != 2 {
		t.Errorf("RetryableErrors should have 2 entries, got %d", len(cfg.RetryableErrors))
	}
}

func TestRetryWithBackoff_BackoffDelay(t *testing.T) {
	ctx := context.Background()
	cfg := discovery.RetryConfig{
		MaxRetries:    3,
		InitialDelay:  50 * time.Millisecond,
		MaxDelay:      500 * time.Millisecond,
		BackoffFactor: 2.0,
		JitterPercent: 0, // No jitter for predictable timing
	}

	start := time.Now()
	callCount := 0
	result := discovery.RetryWithBackoff(ctx, cfg, func() error {
		callCount++
		if callCount <= 3 {
			return errors.New("temporary")
		}
		return nil
	})

	elapsed := time.Since(start)

	if !result.Successful {
		t.Error("Operation should be successful")
	}

	// Expected: 50ms + 100ms + 200ms = 350ms minimum delay
	// With some tolerance for execution time
	if elapsed < 300*time.Millisecond {
		t.Errorf("Elapsed time should be at least ~350ms due to backoff, got %v", elapsed)
	}
}
