package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/krisarmstrong/seed/internal/api"
)

// TestHandleStatus tests the status endpoint.
func TestHandleStatus(t *testing.T) {
	server := api.NewTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/status", http.NoBody)
	w := httptest.NewRecorder()

	server.HandleStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response contains expected fields
	body := w.Body.String()
	expectedFields := []string{`"status"`, `"version"`, `"interface"`, `"isWireless"`, `"icmpAvailable"`}
	for _, field := range expectedFields {
		if !strings.Contains(body, field) {
			t.Errorf("Expected response to contain %s, got: %s", field, body)
		}
	}
}

// TestHandleStatusMethodNotAllowed tests non-GET methods for status endpoint.
func TestHandleStatusMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/status", http.NoBody)
			w := httptest.NewRecorder()

			server.HandleStatus(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d",
					http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}

// TestHandleExportMethodNotAllowed tests non-GET methods for export endpoint.
func TestHandleExportMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()

	methods := []string{http.MethodPost, http.MethodPut, http.MethodDelete}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/api/export", http.NoBody)
			w := httptest.NewRecorder()

			server.HandleExport(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d",
					http.StatusMethodNotAllowed, method, w.Code)
			}
		})
	}
}
