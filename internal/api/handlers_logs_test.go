package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// The frontend logger's batch body is decoded with DisallowUnknownFields, so
// the two sides of this endpoint have to agree key for key. They did not: the
// UI sent a `layer` field the DTO never declared, so every batch was rejected
// and no frontend log has ever reached the store (#2343 diagnosis).
func TestHandleClientLogsWireContract(t *testing.T) {
	logging.InitBroadcaster(16)

	tests := []struct {
		name     string
		body     string
		wantCode int
	}{
		{
			name: "payload the UI sends",
			body: `{"entries":[{"timestamp":"2026-09-07T02:28:53.599Z","level":"INFO",` +
				`"component":"auth","message":"User logged in successfully",` +
				`"sessionId":"mfz4-8f0","requestId":"d85e5ac8b3793594",` +
				`"metadata":{"username":"admin"}}]}`,
			wantCode: http.StatusOK,
		},
		{
			name: "entry claiming its own layer",
			body: `{"entries":[{"timestamp":"2026-09-07T02:28:53.599Z","level":"INFO",` +
				`"layer":"frontend","component":"auth","message":"User logged in successfully"}]}`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := api.NewTestServer()
			defer server.Close()

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/reporting/logs/client",
				strings.NewReader(tt.body),
			)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.HandleClientLogs(w, req)

			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tt.wantCode, w.Body.String())
			}
			if tt.wantCode != http.StatusOK {
				return
			}

			var body struct {
				Status   string `json:"status"`
				Received int    `json:"received"`
			}
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Received != 1 {
				t.Errorf("received = %d, want 1", body.Received)
			}
		})
	}
}

// The log viewer loads its history from /logs/recent and reads requestId,
// sessionId and durationMs. The handler served the logging package's records
// directly, whose keys are snake_case, so those columns were always blank.
func TestHandleLogsRecentWireKeys(t *testing.T) {
	b := logging.InitBroadcaster(16)
	entry := logging.NewLogEntry("INFO", "request served").
		WithLayer("api").
		WithRequestID("req-1").
		WithDuration(42 * time.Millisecond)
	entry.SessionID = "sess-1"
	b.Write(entry)

	server := api.NewTestServer()
	defer server.Close()

	w := httptest.NewRecorder()
	server.HandleLogsRecent(w, httptest.NewRequest(http.MethodGet, "/api/v1/reporting/logs/recent", http.NoBody))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %s)", w.Code, w.Body.String())
	}

	var body struct {
		Count int              `json:"count"`
		Logs  []map[string]any `json:"logs"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Count != 1 || len(body.Logs) != 1 {
		t.Fatalf("count = %d, logs = %d, want 1 and 1", body.Count, len(body.Logs))
	}
	want := map[string]any{"requestId": "req-1", "sessionId": "sess-1", "durationMs": float64(42)}
	for key, value := range want {
		if got := body.Logs[0][key]; got != value {
			t.Errorf("%s = %v, want %v (entry %v)", key, got, value, body.Logs[0])
		}
	}
}
