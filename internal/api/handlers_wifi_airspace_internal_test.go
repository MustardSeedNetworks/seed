package api

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	wifianomaly "github.com/krisarmstrong/seed/internal/wifi/anomaly"
	wifiapp "github.com/krisarmstrong/seed/internal/wifi/app"
	"github.com/krisarmstrong/seed/internal/wifi/dot11"
	"github.com/krisarmstrong/seed/internal/wifi/visibility"
)

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
	// No visibility source wired → graceful empty response, never 500/null.
	s := &Server{wifiQueries: wifiapp.NewQueries(nil)}
	rec := httptest.NewRecorder()
	s.handleWiFiAirspace(rec, httptest.NewRequest(http.MethodGet, "/api/v1/wifi/airspace", nil))

	resp := decodeJSON[WiFiAirspaceResponse](t, rec)
	if resp.SSIDs == nil {
		t.Error("ssids should serialize as [] not null")
	}
	if len(resp.SSIDs) != 0 || resp.Status.CaptureActive {
		t.Errorf("expected empty inactive airspace, got %+v", resp)
	}
}

func TestHandleWiFiAirspaceAndAnomaliesPopulated(t *testing.T) {
	svc, err := visibility.New()
	if err != nil {
		t.Fatalf("visibility.New: %v", err)
	}
	svc.SetSource("monitor0")
	svc.Ingest(openBeacon(t), time.Now())
	svc.Evaluate(time.Now())
	s := &Server{wifiQueries: wifiapp.NewQueries(svc)}

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

	// Anomaly stream contains the open-network detection.
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
