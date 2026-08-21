package api

// handlers_polling_targets_tenancy_internal_test.go covers the API half of
// #1797: the tenant comes from the session, a caller cannot name one, and a
// request that carries no claim is refused rather than defaulted.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/database"
)

const otherClientID = "other-tenant"

// seedOtherClient creates a second tenant so a cross-client request has
// something real to reach for.
func seedOtherClient(t *testing.T, s *Server) {
	t.Helper()
	if err := s.db().Clients().Create(context.Background(), &database.Client{
		ID: otherClientID, Name: "Other", Slug: "other",
	}); err != nil {
		t.Fatalf("create client: %v", err)
	}
}

func TestListPollingTargetsShowsOnlyTheSessionsClient(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	seedOtherClient(t, s)
	seedTarget(t, s.db(), "mine")
	seedTargetFor(t, s.db(), otherClientID, "theirs")

	req := withClaim(
		httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/polling-targets", http.NoBody),
	)
	w := httptest.NewRecorder()
	s.handlePollingTargets(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var body struct {
		Count   int              `json:"count"`
		Targets []map[string]any `json:"targets"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Count != 1 {
		t.Fatalf("count = %d, want 1 — the other client's target is visible", body.Count)
	}
	if body.Targets[0]["name"] != "mine" {
		t.Errorf("listed %v, want the session client's own target", body.Targets[0]["name"])
	}
}

func TestPollingTargetsRejectARequestWithNoClaim(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	seedTarget(t, s.db(), "mine")

	// No withClaim: this is what a route reached outside the auth middleware
	// looks like. It must not fall back to the default tenant.
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/polling-targets", http.NoBody)
	w := httptest.NewRecorder()
	s.handlePollingTargets(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for a request that cannot be attributed to a tenant", w.Code)
	}
}

func TestCreatePollingTargetRejectsACallerNamedTenant(t *testing.T) {
	s := newPollingTargetsTestServer(t)

	req := postJSON(t, map[string]any{
		"name":      "router-1",
		"ipAddress": "10.0.0.1",
		"clientId":  otherClientID,
	})
	w := httptest.NewRecorder()
	s.handlePollingTargets(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 — clientId is not a field a caller may send", w.Code)
	}
	list, err := s.db().PollingTargets().ListAll(context.Background(), testClientID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 0 {
		t.Error("the rejected request still created a target")
	}
}

func TestCreatePollingTargetOwnsItToTheSessionsClient(t *testing.T) {
	s := newPollingTargetsTestServer(t)

	req := postJSON(t, map[string]any{"name": "router-1", "ipAddress": "10.0.0.1"})
	w := httptest.NewRecorder()
	s.handlePollingTargets(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	list, err := s.db().PollingTargets().ListAll(context.Background(), testClientID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("created %d targets for the session's client, want 1", len(list))
	}
}

func TestCrossClientRoutesAre404NotForbidden(t *testing.T) {
	s := newPollingTargetsTestServer(t)
	seedOtherClient(t, s)
	foreign := seedTargetFor(t, s.db(), otherClientID, "theirs")

	body, err := json.Marshal(map[string]any{"name": "stolen", "ipAddress": "10.9.9.9"})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	cases := []struct {
		name   string
		method string
		body   []byte
	}{
		{"get", http.MethodGet, nil},
		{"update", http.MethodPut, body},
		{"delete", http.MethodDelete, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var reader *bytes.Reader
			if tc.body != nil {
				reader = bytes.NewReader(tc.body)
			}
			var req *http.Request
			if reader != nil {
				req = httptest.NewRequest(tc.method, APIVersionPrefix+"/polling-targets/"+foreign.ID, reader)
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, APIVersionPrefix+"/polling-targets/"+foreign.ID, http.NoBody)
			}
			w := httptest.NewRecorder()
			s.handlePollingTargetByID(w, withClaim(req))

			if w.Code != http.StatusNotFound {
				t.Errorf("status = %d, want 404 — a 403 would confirm the id exists", w.Code)
			}
		})
	}

	after, err := s.db().PollingTargets().Get(context.Background(), otherClientID, foreign.ID)
	if err != nil {
		t.Fatalf("the foreign target did not survive: %v", err)
	}
	if after.Name != "theirs" {
		t.Errorf("the foreign target was mutated to %q", after.Name)
	}
}
