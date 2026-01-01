package auth

import (
	"errors"
	"testing"
	"time"
)

func TestNewCSRFManager(t *testing.T) {
	manager := NewCSRFManager()
	if manager == nil {
		t.Fatal("NewCSRFManager returned nil")
	}

	if manager.tokens == nil {
		t.Fatal("tokens map not initialized")
	}

	if manager.ctx == nil {
		t.Fatal("context not initialized")
	}

	if manager.cancel == nil {
		t.Fatal("cancel function not initialized")
	}

	// Clean up
	manager.Stop()
}

func TestCSRFManagerCleanupStopsOnContextCancel(t *testing.T) {
	manager := NewCSRFManager()

	// Generate a token to ensure the manager is working
	token, err := manager.GenerateToken("test-session")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Stop the manager
	manager.Stop()

	// Give the goroutine a moment to stop
	time.Sleep(100 * time.Millisecond)

	// The test passes if we get here without hanging or panicking
	// In a real scenario, we'd verify the goroutine actually stopped,
	// but that's difficult without exposing internal state
}

func TestCSRFManagerGenerateAndValidate(t *testing.T) {
	manager := NewCSRFManager()
	defer manager.Stop()

	sessionID := "test-session"

	// Generate a token
	token, err := manager.GenerateToken(sessionID)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Validate the token
	if validateErr := manager.ValidateToken(sessionID, token); validateErr != nil {
		t.Errorf("failed to validate token: %v", validateErr)
	}

	// Validate with wrong session ID
	if wrongSessionErr := manager.ValidateToken("wrong-session", token); !errors.Is(wrongSessionErr, ErrCSRFTokenInvalid) {
		t.Errorf("expected ErrCSRFTokenInvalid, got %v", wrongSessionErr)
	}

	// Validate with wrong token
	if wrongTokenErr := manager.ValidateToken(sessionID, "wrong-token"); !errors.Is(wrongTokenErr, ErrCSRFTokenInvalid) {
		t.Errorf("expected ErrCSRFTokenInvalid, got %v", wrongTokenErr)
	}
}

func TestCSRFManagerStop(t *testing.T) {
	manager := NewCSRFManager()

	// Generate some tokens
	_, err := manager.GenerateToken("session1")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// Verify context is not canceled initially
	select {
	case <-manager.ctx.Done():
		t.Fatal("context should not be canceled initially")
	default:
		// Expected - context is still active
	}

	// Stop the manager
	manager.Stop()

	// Give goroutine time to exit
	time.Sleep(50 * time.Millisecond)

	// Verify context is canceled
	select {
	case <-manager.ctx.Done():
		// Expected - context is canceled
	default:
		t.Fatal("context should be canceled after Stop()")
	}
}
