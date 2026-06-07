package wifiapp_test

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi/airspace"
	wifianomaly "github.com/MustardSeedNetworks/seed/internal/wifi/anomaly"
	wifiapp "github.com/MustardSeedNetworks/seed/internal/wifi/app"
)

func hasDef(anoms []anomaly.Anomaly, defKey string) bool {
	for _, a := range anoms {
		if a.DefKey == defKey {
			return true
		}
	}
	return false
}

func TestAnalyzeBSSesEmpty(t *testing.T) {
	got, err := wifiapp.AnalyzeBSSes(nil, nil, time.Unix(0, 0))
	if err != nil {
		t.Fatalf("AnalyzeBSSes: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("anomalies = %v, want none", got)
	}
}

func TestAnalyzeBSSesOpenNetwork(t *testing.T) {
	bsses := []airspace.BSSView{
		{BSSID: "aa:bb:cc:00:00:01", SSID: "guest", Band: "2.4 GHz", Channel: 6, Security: "Open"},
	}
	got, err := wifiapp.AnalyzeBSSes(bsses, nil, time.Unix(100, 0))
	if err != nil {
		t.Fatalf("AnalyzeBSSes: %v", err)
	}
	if !hasDef(got, wifianomaly.DefOpenNetwork) {
		t.Errorf("expected %s anomaly, got %+v", wifianomaly.DefOpenNetwork, got)
	}
}

func TestAnalyzeBSSesSecurityMismatch(t *testing.T) {
	// One SSID advertised by a weak (Open) and a strong (WPA2) BSS.
	bsses := []airspace.BSSView{
		{BSSID: "aa:bb:cc:00:00:01", SSID: "corp", Band: "5 GHz", Channel: 36, Security: "Open"},
		{BSSID: "dd:ee:ff:00:00:02", SSID: "corp", Band: "5 GHz", Channel: 40, Security: "WPA2"},
	}
	got, err := wifiapp.AnalyzeBSSes(bsses, nil, time.Unix(100, 0))
	if err != nil {
		t.Fatalf("AnalyzeBSSes: %v", err)
	}
	if !hasDef(got, wifianomaly.DefSecurityMismatch) {
		t.Errorf("expected %s anomaly, got %+v", wifianomaly.DefSecurityMismatch, got)
	}
}

func TestAnalyzeBSSesDeterministic(t *testing.T) {
	bsses := []airspace.BSSView{
		{BSSID: "aa:bb:cc:00:00:01", SSID: "guest", Band: "2.4 GHz", Channel: 6, Security: "Open"},
		{BSSID: "aa:bb:cc:00:00:02", SSID: "guest2", Band: "2.4 GHz", Channel: 6, Security: "WEP"},
	}
	at := time.Unix(100, 0)
	first, err := wifiapp.AnalyzeBSSes(bsses, nil, at)
	if err != nil {
		t.Fatalf("AnalyzeBSSes: %v", err)
	}
	second, _ := wifiapp.AnalyzeBSSes(bsses, nil, at)
	if len(first) != len(second) {
		t.Fatalf("non-deterministic length: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].DefKey != second[i].DefKey || first[i].Subject != second[i].Subject {
			t.Errorf("order differs at %d: %v vs %v", i, first[i], second[i])
		}
	}
}
