package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	api "github.com/MustardSeedNetworks/seed/internal/api"
)

// TestHandleStatus tests the status endpoint.
func TestHandleStatus(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", http.NoBody)
	w := httptest.NewRecorder()

	server.HandleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response contains expected fields
	body := w.Body.String()
	expectedFields := []string{
		`"status"`,
		`"version"`,
		`"interface"`,
		`"isWireless"`,
		`"icmpAvailable"`,
	}
	for _, field := range expectedFields {
		if !strings.Contains(body, field) {
			t.Errorf("Expected response to contain %s, got: %s", field, body)
		}
	}
}
