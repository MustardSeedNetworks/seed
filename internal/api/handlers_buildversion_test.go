package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/krisarmstrong/seed/internal/api"
)

func TestHandleBuildVersionGET(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/__version", http.NoBody)
	w := httptest.NewRecorder()
	server.HandleBuildVersion(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	for _, key := range []string{"version", "commit", "buildTime", "uiBuildHash"} {
		if body[key] == "" {
			t.Errorf("response missing or empty field %q (got %#v)", key, body)
		}
	}
}

func TestHandleBuildVersionMethodNotAllowed(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/__version", http.NoBody)
			w := httptest.NewRecorder()
			server.HandleBuildVersion(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
			}
		})
	}
}
