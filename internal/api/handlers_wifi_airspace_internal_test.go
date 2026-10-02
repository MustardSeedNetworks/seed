package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/license"
	wifianomaly "github.com/MustardSeedNetworks/seed/internal/wifi/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi/dot11"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
	"github.com/MustardSeedNetworks/seed/internal/wifi/visibility"
)

// stubAnomalyStore is the troubleshooting.AnomalyStore half of the read use-case:
// the source=wifi anomaly list the handler now reads from the unified store
// (ADR-0029 §4), instead of from the in-memory visibility engine.
type stubAnomalyStore struct {
	available bool
	anomalies []anomaly.Anomaly
}

func (s stubAnomalyStore) Available() bool { return s.available }
func (s stubAnomalyStore) ActiveWiFi(context.Context) ([]anomaly.Anomaly, error) {
	return s.anomalies, nil
}

// The feature-gate middleware is applied at route registration and exercised by
// the authchain golden harness; these tests drive the handlers directly to
// characterize their own logic — graceful empty response vs. populated read.

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var out T
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func openBeacon(t *testing.T) *dot11.Frame {
	t.Helper()
	mac, err := net.ParseMAC("00:11:22:33:44:55")
	if err != nil {
		t.Fatal(err)
	}
	return &dot11.Frame{
		Kind:       dot11.KindBeacon,
		BSSID:      mac,
		Band:       dot11.Band24GHz,
		ChannelNum: 6,
		BSS:        &dot11.BSS{SSID: "guest", Security: dot11.SecurityOpen, Standard: dot11.Standard80211ac},
	}
}

func TestHandleWiFiAirspaceEmptyWhenNoComponent(t *testing.T) {
	// No visibility source / store wired → graceful empty response, never 500/null.
	s := &Server{wifiQueries: troubleshooting.NewQueries(nil, nil)}
	rec := httptest.NewRecorder()
	s.handleWiFiAirspace(rec, httptest.NewRequest(http.MethodGet, "/api/v1/wifi/airspace", nil))

	resp := decodeJSON[WiFiAirspaceResponse](t, rec)
	if resp.SSIDs == nil {
		t.Error("ssids should serialize as [] not null")
	}
	if len(resp.SSIDs) != 0 || resp.Status.CaptureActive {
		t.Errorf("expected empty inactive airspace, got %+v", resp)
	}

	// Anomalies likewise degrade to an empty stream (never 500/null).
	recN := httptest.NewRecorder()
	s.handleWiFiAnomalies(recN, httptest.NewRequest(http.MethodGet, "/api/v1/wifi/anomalies", nil))
	an := decodeJSON[WiFiAnomaliesResponse](t, recN)
	if an.Anomalies == nil || len(an.Anomalies) != 0 {
		t.Errorf("expected empty anomaly stream, got %+v", an.Anomalies)
	}
}

func TestHandleWiFiAirspaceAndAnomaliesPopulated(t *testing.T) {
	// Airspace (tree + status) comes from the live visibility component...
	svc := visibility.New()
	svc.SetSource("monitor0")
	svc.Ingest(openBeacon(t), time.Now())
	svc.Evaluate(context.Background(), time.Now())
	// ...while the anomaly list comes from the unified store (ADR-0029 §4).
	store := stubAnomalyStore{
		available: true,
		anomalies: []anomaly.Anomaly{{
			DefKey:  wifianomaly.DefOpenNetwork,
			Subject: anomaly.SubjectRef{Kind: anomaly.SubjectBSSID, ID: "00:11:22:33:44:55"},
		}},
	}
	s := &Server{wifiQueries: troubleshooting.NewQueries(svc, store)}

	// Airspace tree is populated and reports the active source.
	recA := httptest.NewRecorder()
	s.handleWiFiAirspace(recA, httptest.NewRequest(http.MethodGet, "/api/v1/wifi/airspace", nil))
	air := decodeJSON[WiFiAirspaceResponse](t, recA)
	if len(air.SSIDs) == 0 {
		t.Error("expected a populated airspace tree")
	}
	if !air.Status.CaptureActive || air.Status.Source != "monitor0" {
		t.Errorf("status should reflect the active source: %+v", air.Status)
	}

	// Anomaly stream (read from the store) contains the open-network detection.
	recN := httptest.NewRecorder()
	s.handleWiFiAnomalies(recN, httptest.NewRequest(http.MethodGet, "/api/v1/wifi/anomalies", nil))
	an := decodeJSON[WiFiAnomaliesResponse](t, recN)
	found := false
	for _, a := range an.Anomalies {
		if a.DefKey == wifianomaly.DefOpenNetwork {
			found = true
		}
	}
	if !found {
		t.Errorf("expected %s in the anomaly stream, got %+v", wifianomaly.DefOpenNetwork, an.Anomalies)
	}
}

// TestWiFiAnomaliesNameTheRulesThatNeedCapture is S5-1's second acceptance
// clause on the wire: on a scan-only host the anomaly stream says which rules
// could not run, so their absence does not read as a clean airspace.
func TestWiFiAnomaliesNameTheRulesThatNeedCapture(t *testing.T) {
	svc := visibility.New()
	s := &Server{wifiQueries: troubleshooting.NewQueries(svc, stubAnomalyStore{available: true})}

	get := func() map[string]any {
		rec := httptest.NewRecorder()
		s.handleWiFiAnomalies(rec, httptest.NewRequest(http.MethodGet, "/api/v1/wifi/anomalies", nil))
		body := decodeJSON[map[string]any](t, rec)
		status, ok := body["status"].(map[string]any)
		if !ok {
			t.Fatalf("no status object in %v", body)
		}
		return status
	}

	rules, ok := get()["needsCapture"].([]any)
	if !ok || len(rules) != 1 {
		t.Fatalf("scan-only needsCapture = %v, want the deauth-flood rule", rules)
	}
	if rule, _ := rules[0].(map[string]any); rule["id"] != wifianomaly.DefDeauthFlood || rule["title"] == "" {
		t.Errorf("needsCapture[0] = %v, want id %s with its title", rules[0], wifianomaly.DefDeauthFlood)
	}

	svc.SetSource("monitor0")
	if got, present := get()["needsCapture"]; present {
		t.Errorf("capturing: needsCapture = %v, want the key absent", got)
	}
}

// TestWiFiVisibilityTiers pins the #2351 retier through the registered routes:
// Free scans (and is refused the analysis), Starter sees the airspace and the
// anomalies with the clients withheld, Pro sees the clients.
func TestWiFiVisibilityTiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		activate    func(t *testing.T, mgr *license.Manager)
		wantStatus  int
		wantClients bool
	}{
		{
			name:       "free is refused the analysis",
			activate:   func(*testing.T, *license.Manager) {},
			wantStatus: http.StatusPaymentRequired,
		},
		{
			name: "starter sees the airspace without its clients",
			activate: func(t *testing.T, mgr *license.Manager) {
				t.Helper()
				if res := mgr.Activate(prodSeedStarterVector); !res.Success {
					t.Fatalf("Starter vector did not activate: %s", res.Message)
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "pro sees the clients",
			activate: func(t *testing.T, mgr *license.Manager) {
				t.Helper()
				if res := mgr.StartTrial(); !res.Success {
					t.Fatalf("StartTrial: %s", res.Message)
				}
			},
			wantStatus:  http.StatusOK,
			wantClients: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			s, mgr := apiTokenTestSetup(t)
			s.setupRoutes()
			tc.activate(t, mgr)
			s.wifiQueries = troubleshooting.NewQueries(airspaceWithOneClient(t), stubAnomalyStore{available: true})

			get := func(path string) *httptest.ResponseRecorder {
				rec := httptest.NewRecorder()
				s.mux.ServeHTTP(rec, newAuthedRequest(http.MethodGet, APIVersionPrefix+path, nil, "alice"))
				if rec.Code != tc.wantStatus {
					t.Fatalf("GET %s: status = %d, want %d; body=%s", path, rec.Code, tc.wantStatus, rec.Body.String())
				}
				return rec
			}

			get("/wifi/anomalies")
			rec := get("/wifi/airspace")
			if tc.wantStatus == http.StatusOK {
				assertClients(t, decodeJSON[WiFiAirspaceResponse](t, rec), tc.wantClients)
			}
		})
	}
}

// airspaceWithOneClient is openBeacon's BSS with one associated client.
func airspaceWithOneClient(t *testing.T) *visibility.Service {
	t.Helper()
	bssid, err := net.ParseMAC("00:11:22:33:44:55")
	if err != nil {
		t.Fatal(err)
	}
	sta, err := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatal(err)
	}
	svc := visibility.New()
	svc.SetSource("monitor0")
	now := time.Now()
	svc.Ingest(openBeacon(t), now)
	svc.Ingest(&dot11.Frame{
		Kind:        dot11.KindAssocRequest,
		BSSID:       bssid,
		Transmitter: sta,
		Band:        dot11.Band24GHz,
		ChannelNum:  6,
	}, now)
	return svc
}

// assertClients checks that the one client is either present everywhere the
// response counts it, or absent everywhere and flagged as withheld.
func assertClients(t *testing.T, air WiFiAirspaceResponse, want bool) {
	t.Helper()
	if len(air.SSIDs) != 1 || len(air.SSIDs[0].APs) != 1 || len(air.SSIDs[0].APs[0].BSSes) != 1 {
		t.Fatalf("want one SSID/AP/BSS, got %+v", air.SSIDs)
	}
	wantN := 0
	if want {
		wantN = 1
	}
	counts := []int{len(air.SSIDs[0].APs[0].BSSes[0].Stations), air.SSIDs[0].StationCount, air.Status.Stations}
	if !slices.Equal(counts, []int{wantN, wantN, wantN}) {
		t.Errorf("stations/stationCount/status.stations = %v, want all %d", counts, wantN)
	}
	if air.ClientsWithheld == want {
		t.Errorf("clientsWithheld = %v, want %v", air.ClientsWithheld, !want)
	}
}
