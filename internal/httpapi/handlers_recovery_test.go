package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/krisarmstrong/seed/internal/auth"
	api "github.com/krisarmstrong/seed/internal/httpapi"
)

// TestHandleRecoveryStatusNoManager tests recovery status when manager is not configured.
func TestHandleRecoveryStatusNoManager(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/recovery/status", http.NoBody)
	w := httptest.NewRecorder()

	server.HandleRecoveryStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp["active"] != false {
		t.Error("expected active to be false when recovery manager is not configured")
	}
}

// TestHandleRecoveryStatusMethodNotAllowed tests non-GET methods.
func TestHandleRecoveryStatusMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/recovery/status", http.NoBody)
			w := httptest.NewRecorder()

			server.HandleRecoveryStatus(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d for %s, got %d",
					http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleRecoveryStatusWithManager tests recovery status with a configured manager.
func TestHandleRecoveryStatusWithManager(t *testing.T) {
	server := api.NewTestServer()

	// Create a temp directory for recovery files
	tmpDir := t.TempDir()
	recoveryManager := auth.NewRecoveryTokenManager(tmpDir)
	server.SetRecoveryManager(recoveryManager)

	// Test without trigger file
	t.Run("no trigger file", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/recovery/status", http.NoBody)
		w := httptest.NewRecorder()

		server.HandleRecoveryStatus(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp["active"] != false {
			t.Error("expected active to be false without trigger file")
		}
	})

	// Test with trigger file
	t.Run("with trigger file", func(t *testing.T) {
		// Create trigger file
		triggerPath := filepath.Join(tmpDir, ".recovery")
		if err := os.WriteFile(triggerPath, []byte{}, 0o600); err != nil {
			t.Fatalf("failed to create trigger file: %v", err)
		}
		defer os.Remove(triggerPath)

		req := httptest.NewRequest(http.MethodGet, "/api/recovery/status", http.NoBody)
		w := httptest.NewRecorder()

		server.HandleRecoveryStatus(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
		}

		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp["active"] != true {
			t.Error("expected active to be true with trigger file")
		}

		if _, ok := resp["remainingTime"]; !ok {
			t.Error("expected remainingTime field when recovery is active")
		}
	})
}

// TestHandleRecoveryCompleteNoManager tests recovery complete when manager is not configured.
func TestHandleRecoveryCompleteNoManager(t *testing.T) {
	server := api.NewTestServer()

	reqBody := map[string]string{
		"token":    "some-token",
		"password": "StrongPassword123!",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/recovery/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.HandleRecoveryComplete(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// TestHandleRecoveryCompleteMethodNotAllowed tests non-POST methods.
func TestHandleRecoveryCompleteMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/recovery/complete", http.NoBody)
			w := httptest.NewRecorder()

			server.HandleRecoveryComplete(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d for %s, got %d",
					http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleRecoveryCompleteInvalidToken tests recovery with invalid token.
func TestHandleRecoveryCompleteInvalidToken(t *testing.T) {
	server := api.NewTestServer()

	// Create a temp directory for recovery files
	tmpDir := t.TempDir()
	recoveryManager := auth.NewRecoveryTokenManager(tmpDir)
	server.SetRecoveryManager(recoveryManager)

	// Create trigger file to generate token
	triggerPath := filepath.Join(tmpDir, ".recovery")
	if err := os.WriteFile(triggerPath, []byte{}, 0o600); err != nil {
		t.Fatalf("failed to create trigger file: %v", err)
	}

	// Trigger token generation
	recoveryManager.CheckRecoveryMode()

	reqBody := map[string]string{
		"token":    "invalid-token",
		"password": "StrongPassword123!",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/recovery/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.HandleRecoveryComplete(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

// TestHandleRecoveryInstructionsNoManager tests recovery instructions when manager is not configured.
func TestHandleRecoveryInstructionsNoManager(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/recovery/instructions", http.NoBody)
	w := httptest.NewRecorder()

	server.HandleRecoveryInstructions(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// TestHandleRecoveryInstructionsMethodNotAllowed tests non-GET methods.
func TestHandleRecoveryInstructionsMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/recovery/instructions", http.NoBody)
			w := httptest.NewRecorder()

			server.HandleRecoveryInstructions(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d for %s, got %d",
					http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleRecoveryInstructionsWithManager tests recovery instructions with a configured manager.
func TestHandleRecoveryInstructionsWithManager(t *testing.T) {
	server := api.NewTestServer()

	// Create a temp directory for recovery files
	tmpDir := t.TempDir()
	recoveryManager := auth.NewRecoveryTokenManager(tmpDir)
	server.SetRecoveryManager(recoveryManager)

	req := httptest.NewRequest(http.MethodGet, "/api/recovery/instructions", http.NoBody)
	w := httptest.NewRecorder()

	server.HandleRecoveryInstructions(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Verify required fields
	expectedFields := []string{"triggerFile", "tokenFile", "expiryTime", "steps"}
	for _, field := range expectedFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("expected field %s in response", field)
		}
	}

	// Verify steps is an array
	steps, ok := resp["steps"].([]any)
	if !ok {
		t.Error("expected steps to be an array")
	} else if len(steps) == 0 {
		t.Error("expected steps array to have at least one item")
	}
}

// TestHandleRecoveryCompleteInvalidJSON tests recovery with invalid JSON body.
func TestHandleRecoveryCompleteInvalidJSON(t *testing.T) {
	server := api.NewTestServer()

	// Create a temp directory for recovery files
	tmpDir := t.TempDir()
	recoveryManager := auth.NewRecoveryTokenManager(tmpDir)
	server.SetRecoveryManager(recoveryManager)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/recovery/complete",
		bytes.NewReader([]byte("invalid json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.HandleRecoveryComplete(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
