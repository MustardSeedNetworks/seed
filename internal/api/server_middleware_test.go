package api_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krisarmstrong/seed/internal/api"
)

// Verifies BodyLimitMiddleware applies to GET requests (regression for #766).
func TestBodyLimitMiddleware_GETEnforced(t *testing.T) {
	// Handler that reads the entire body and surfaces the read error
	handler := api.ExportBodyLimitMiddleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	)

	// Build a GET request with a body larger than the default 1MB limit
	largeBody := bytes.Repeat([]byte("a"), 2*1024*1024) // 2MB
	req := httptest.NewRequest(http.MethodGet, "/api/test", bytes.NewReader(largeBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, rr.Code)
	}
}
