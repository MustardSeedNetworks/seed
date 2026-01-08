package auth_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/auth"
)

func TestNewRecoveryTokenManager(t *testing.T) {
	tmpDir := t.TempDir()
	m := auth.NewRecoveryTokenManager(tmpDir)

	if m == nil {
		t.Fatal("NewRecoveryTokenManager returned nil")
	}

	expectedTrigger := filepath.Join(tmpDir, ".recovery")
	if m.TriggerFilePath() != expectedTrigger {
		t.Errorf("expected trigger path %s, got %s", expectedTrigger, m.TriggerFilePath())
	}

	expectedToken := filepath.Join(tmpDir, ".recovery-token")
	if m.TokenFilePath() != expectedToken {
		t.Errorf("expected token path %s, got %s", expectedToken, m.TokenFilePath())
	}
}

func TestCheckRecoveryModeNoTrigger(t *testing.T) {
	tmpDir := t.TempDir()
	m := auth.NewRecoveryTokenManager(tmpDir)

	// Without trigger file, recovery mode should not be active
	if m.CheckRecoveryMode() {
		t.Error("expected recovery mode to be inactive without trigger file")
	}

	if m.IsActive() {
		t.Error("expected IsActive() to return false without trigger file")
	}
}

func TestCheckRecoveryModeWithTrigger(t *testing.T) {
	tmpDir := t.TempDir()
	m := auth.NewRecoveryTokenManager(tmpDir)

	// Create trigger file
	triggerPath := filepath.Join(tmpDir, ".recovery")
	if err := os.WriteFile(triggerPath, []byte{}, 0o600); err != nil {
		t.Fatalf("failed to create trigger file: %v", err)
	}

	// CheckRecoveryMode should return true and generate a token
	if !m.CheckRecoveryMode() {
		t.Error("expected recovery mode to be active with trigger file")
	}

	// Token file should have been created
	tokenPath := filepath.Join(tmpDir, ".recovery-token")
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		t.Error("expected token file to be created")
	}

	// IsActive should now return true
	if !m.IsActive() {
		t.Error("expected IsActive() to return true after token generation")
	}

	// RemainingTime should be greater than 0
	if m.RemainingTime() <= 0 {
		t.Error("expected RemainingTime() > 0 after token generation")
	}
}

func TestTokenExpiryDuration(t *testing.T) {
	tmpDir := t.TempDir()
	m := auth.NewRecoveryTokenManager(tmpDir)

	expected := 15 * time.Minute
	if m.TokenExpiryDuration() != expected {
		t.Errorf("expected token expiry duration %v, got %v", expected, m.TokenExpiryDuration())
	}
}

func TestValidateAndConsumeEmptyToken(t *testing.T) {
	tmpDir := t.TempDir()
	m := auth.NewRecoveryTokenManager(tmpDir)

	// Empty token should always fail
	if m.ValidateAndConsume("") {
		t.Error("expected empty token validation to fail")
	}
}

func TestValidateAndConsumeNoTokenGenerated(t *testing.T) {
	tmpDir := t.TempDir()
	m := auth.NewRecoveryTokenManager(tmpDir)

	// Without generating a token, validation should fail
	if m.ValidateAndConsume("some-token") {
		t.Error("expected validation to fail when no token has been generated")
	}
}

func TestValidateAndConsumeValidToken(t *testing.T) {
	tmpDir := t.TempDir()
	m := auth.NewRecoveryTokenManager(tmpDir)

	// Create trigger file to generate token
	triggerPath := filepath.Join(tmpDir, ".recovery")
	if err := os.WriteFile(triggerPath, []byte{}, 0o600); err != nil {
		t.Fatalf("failed to create trigger file: %v", err)
	}

	// Trigger token generation
	if !m.CheckRecoveryMode() {
		t.Fatal("failed to enter recovery mode")
	}

	// Read the generated token
	tokenPath := filepath.Join(tmpDir, ".recovery-token")
	tokenBytes, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("failed to read token file: %v", err)
	}

	// Token file has a newline at the end
	token := string(tokenBytes)
	if len(token) > 0 && token[len(token)-1] == '\n' {
		token = token[:len(token)-1]
	}

	// Validate the token
	if !m.ValidateAndConsume(token) {
		t.Error("expected valid token to be accepted")
	}

	// Token should be single-use - second validation should fail
	if m.ValidateAndConsume(token) {
		t.Error("expected token to be single-use, second validation should fail")
	}
}

func TestValidateAndConsumeInvalidToken(t *testing.T) {
	tmpDir := t.TempDir()
	m := auth.NewRecoveryTokenManager(tmpDir)

	// Create trigger file to generate token
	triggerPath := filepath.Join(tmpDir, ".recovery")
	if err := os.WriteFile(triggerPath, []byte{}, 0o600); err != nil {
		t.Fatalf("failed to create trigger file: %v", err)
	}

	// Trigger token generation
	if !m.CheckRecoveryMode() {
		t.Fatal("failed to enter recovery mode")
	}

	// Try with an invalid token
	if m.ValidateAndConsume("invalid-token-value") {
		t.Error("expected invalid token to be rejected")
	}
}

func TestCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	m := auth.NewRecoveryTokenManager(tmpDir)

	// Create trigger file
	triggerPath := filepath.Join(tmpDir, ".recovery")
	if err := os.WriteFile(triggerPath, []byte{}, 0o600); err != nil {
		t.Fatalf("failed to create trigger file: %v", err)
	}

	// Trigger token generation
	if !m.CheckRecoveryMode() {
		t.Fatal("failed to enter recovery mode")
	}

	tokenPath := filepath.Join(tmpDir, ".recovery-token")

	// Verify files exist
	if _, err := os.Stat(triggerPath); os.IsNotExist(err) {
		t.Error("expected trigger file to exist before cleanup")
	}
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		t.Error("expected token file to exist before cleanup")
	}

	// Cleanup
	m.Cleanup()

	// Verify files are removed
	if _, err := os.Stat(triggerPath); !os.IsNotExist(err) {
		t.Error("expected trigger file to be removed after cleanup")
	}
	if _, err := os.Stat(tokenPath); !os.IsNotExist(err) {
		t.Error("expected token file to be removed after cleanup")
	}

	// IsActive should return false after cleanup
	if m.IsActive() {
		t.Error("expected IsActive() to return false after cleanup")
	}
}

func TestInvalidate(t *testing.T) {
	tmpDir := t.TempDir()
	m := auth.NewRecoveryTokenManager(tmpDir)

	// Create trigger file
	triggerPath := filepath.Join(tmpDir, ".recovery")
	if err := os.WriteFile(triggerPath, []byte{}, 0o600); err != nil {
		t.Fatalf("failed to create trigger file: %v", err)
	}

	// Trigger token generation
	if !m.CheckRecoveryMode() {
		t.Fatal("failed to enter recovery mode")
	}

	// Read the token before invalidation
	tokenPath := filepath.Join(tmpDir, ".recovery-token")
	tokenBytes, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("failed to read token file: %v", err)
	}
	token := string(tokenBytes)
	if len(token) > 0 && token[len(token)-1] == '\n' {
		token = token[:len(token)-1]
	}

	// Invalidate should clear the internal token but leave files
	m.Invalidate()

	// Trigger file should still exist
	if _, statErr := os.Stat(triggerPath); os.IsNotExist(statErr) {
		t.Error("expected trigger file to still exist after Invalidate")
	}

	// Token should no longer validate
	if m.ValidateAndConsume(token) {
		t.Error("expected token validation to fail after Invalidate")
	}
}

func TestRemainingTimeNoToken(t *testing.T) {
	tmpDir := t.TempDir()
	m := auth.NewRecoveryTokenManager(tmpDir)

	// Without a token, RemainingTime should return 0
	if m.RemainingTime() != 0 {
		t.Errorf("expected RemainingTime() to return 0 without token, got %v", m.RemainingTime())
	}
}

func TestMultipleCheckRecoveryModeCalls(t *testing.T) {
	tmpDir := t.TempDir()
	m := auth.NewRecoveryTokenManager(tmpDir)

	// Create trigger file
	triggerPath := filepath.Join(tmpDir, ".recovery")
	if err := os.WriteFile(triggerPath, []byte{}, 0o600); err != nil {
		t.Fatalf("failed to create trigger file: %v", err)
	}

	// First call generates token
	if !m.CheckRecoveryMode() {
		t.Fatal("first CheckRecoveryMode should return true")
	}

	// Read the first token
	tokenPath := filepath.Join(tmpDir, ".recovery-token")
	firstTokenBytes, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("failed to read token file: %v", err)
	}

	// Second call should return true but use same token (not regenerate)
	if !m.CheckRecoveryMode() {
		t.Fatal("second CheckRecoveryMode should return true")
	}

	// Token should be the same
	secondTokenBytes, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("failed to read token file on second call: %v", err)
	}

	if string(firstTokenBytes) != string(secondTokenBytes) {
		t.Error("expected token to remain the same on repeated CheckRecoveryMode calls")
	}
}
