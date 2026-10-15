package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
	snmppoller "github.com/MustardSeedNetworks/seed/internal/polling/snmp"
)

const pollingTargetPollPath = pollingTargetPath + "/poll"

func postPoll(t *testing.T, s *Server, id string) *httptest.ResponseRecorder {
	t.Helper()
	req := withClaim(httptest.NewRequest(http.MethodPost, pollingTargetsPath+"/"+id+"/poll", nil))
	rec := httptest.NewRecorder()
	routed(pollingTargetPollPath, s.handlePollingTargetPoll)(rec, req)
	return rec
}

// startedPoller wires a running poller over the server's own table, with no
// scheduler ticks and no credential resolver.
func startedPoller(t *testing.T, s *Server, db *database.DB) {
	t.Helper()
	poller := snmppoller.NewPoller(db.PollingTargets(), noopScheduler{}, nil)
	if err := poller.Start(context.Background()); err != nil {
		t.Fatalf("start poller: %v", err)
	}
	t.Cleanup(func() { _ = poller.Stop(context.Background()) })
	s.snmpPoller = poller
}

// "Test target" on a paused target runs its chain and answers the outcome
// the poll recorded. With no credential resolver every poll fails closed, so
// the reason in lastError is the poller's, read back from the table.
func TestHandlePollingTargetPoll_AnswersTheRecordedOutcome(t *testing.T) {
	s, db := newPollingTargetsTestServer(t)
	startedPoller(t, s, db)
	target := seedTarget(t, db, "paused-switch")
	target.Enabled = false
	if err := db.PollingTargets().Update(context.Background(), testClientID, target); err != nil {
		t.Fatalf("disable target: %v", err)
	}

	rec := postPoll(t, s, target.ID)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var got PollingTargetResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.LastStatus != "error" || !strings.Contains(got.LastError, "no credential resolver") {
		t.Errorf("lastStatus = %q, lastError = %q, want the poll's credential failure", got.LastStatus, got.LastError)
	}
	if got.LastPolledAt == "" {
		t.Error("lastPolledAt is empty after a poll")
	}
	if got.LastSuccessAt != "" {
		t.Errorf("lastSuccessAt = %q after only a failure, want absent", got.LastSuccessAt)
	}
	if got.Enabled {
		t.Error("a test poll enabled the target")
	}
}

func TestHandlePollingTargetPoll_Refusals(t *testing.T) {
	tests := []struct {
		name    string
		poller  func(t *testing.T, s *Server, db *database.DB)
		id      func(t *testing.T, s *Server, db *database.DB) string
		want    int
		message string
	}{
		{
			name:   "no poller (no keyring)",
			poller: func(*testing.T, *Server, *database.DB) {},
			id:     func(t *testing.T, _ *Server, db *database.DB) string { return seedTarget(t, db, "a").ID },
			want:   http.StatusServiceUnavailable, message: "not running",
		},
		{
			name: "poller below its licence tier is never started",
			poller: func(_ *testing.T, s *Server, db *database.DB) {
				s.snmpPoller = snmppoller.NewPoller(db.PollingTargets(), noopScheduler{}, nil)
			},
			id:   func(t *testing.T, _ *Server, db *database.DB) string { return seedTarget(t, db, "a").ID },
			want: http.StatusServiceUnavailable, message: "not running",
		},
		{
			name:   "unknown target",
			poller: startedPoller,
			id:     func(*testing.T, *Server, *database.DB) string { return "missing" },
			want:   http.StatusNotFound, message: "Target not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, db := newPollingTargetsTestServer(t)
			tt.poller(t, s, db)

			rec := postPoll(t, s, tt.id(t, s, db))

			if rec.Code != tt.want || !strings.Contains(rec.Body.String(), tt.message) {
				t.Errorf("got %d %s, want %d containing %q", rec.Code, rec.Body.String(), tt.want, tt.message)
			}
		})
	}
}
