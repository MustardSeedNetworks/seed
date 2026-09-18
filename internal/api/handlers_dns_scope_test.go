// SPDX-License-Identifier: BUSL-1.1

package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	api "github.com/MustardSeedNetworks/seed/internal/api"
)

// Both DNS surfaces have to carry the scope. The card reads whichever one
// answers first — the REST fetch on mount, the broadcast afterwards — and a
// field present on one and absent on the other is read as undefined half the
// time, which is how the licence feature catalog stayed invisible (#2688).
func TestBothDNSSurfacesCarryTheServerScope(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	server := api.NewTestServer()
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry/dns", http.NoBody)
	w := httptest.NewRecorder()
	server.Mux().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /telemetry/dns: status = %d: %s", w.Code, w.Body.String())
	}

	var rest map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &rest); err != nil {
		t.Fatalf("decode REST payload: %v", err)
	}
	restScope, ok := rest["serverScope"].(string)
	if !ok {
		t.Fatalf("REST payload has no serverScope: %v", rest)
	}
	if restScope == "" {
		t.Error("REST serverScope is empty; the card cannot tell whose resolvers it shows")
	}

	broadcast := server.ExportCollectDNSData()
	if broadcast == nil {
		t.Fatal("broadcast payload is nil")
	}
	wsScope, ok := broadcast["serverScope"].(string)
	if !ok {
		t.Fatalf("broadcast payload has no serverScope: %v", broadcast)
	}
	if wsScope != restScope {
		t.Errorf("broadcast serverScope = %q, REST = %q; the two surfaces disagree", wsScope, restScope)
	}
}
