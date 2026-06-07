package api_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/MustardSeedNetworks/seed/internal/api"
)

// Verifies bodyLimitMiddleware caps GET requests on NON-API paths (regression
// for #766) and, per ADR-0002, leaves /api/v1 bodies to the capability registry
// (route.maxBodyBytes) rather than capping them itself.
func TestBodyLimitMiddleware_GETEnforced(t *testing.T) {
	// Handler that reads the entire body and surfaces the read error.
	handler := api.ExportBodyLimitMiddleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := io.ReadAll(r.Body); err != nil {
				http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
				return
			}
			w.WriteHeader(http.StatusOK)
		}),
	)

	largeBody := bytes.Repeat([]byte("a"), 2*1024*1024) // 2MB > MaxBodySizeDefault (1MB)

	// Non-API path: the middleware backstop caps it -> 413.
	req := httptest.NewRequest(http.MethodGet, "/static/asset", bytes.NewReader(largeBody))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("non-API path: expected status %d, got %d", http.StatusRequestEntityTooLarge, rr.Code)
	}

	// API path: the middleware does NOT cap it (the registry does), so a read of
	// the full body succeeds here -> 200.
	apiReq := httptest.NewRequest(http.MethodGet, "/api/v1/test", bytes.NewReader(largeBody))
	apiRR := httptest.NewRecorder()
	handler.ServeHTTP(apiRR, apiReq)
	if apiRR.Code != http.StatusOK {
		t.Fatalf("API path (registry owns the limit): expected status %d, got %d",
			http.StatusOK, apiRR.Code)
	}
}
