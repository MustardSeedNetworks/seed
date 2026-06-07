package survey

import (
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/wifi"
	wifianomaly "github.com/MustardSeedNetworks/seed/internal/wifi/anomaly"
)

func TestBandLabel(t *testing.T) {
	tests := []struct {
		freq, ch int
		want     string
	}{
		{2412, 1, "2.4 GHz"},
		{2484, 14, "2.4 GHz"},
		{5180, 36, "5 GHz"},
		{5955, 1, "6 GHz"},
		{0, 6, "2.4 GHz"},  // frequency missing, channel fallback
		{0, 36, "Unknown"}, // missing frequency, non-2.4 channel
		{3000, 0, "Unknown"},
	}
	for _, tc := range tests {
		if got := bandLabel(tc.freq, tc.ch); got != tc.want {
			t.Errorf("bandLabel(%d,%d) = %q, want %q", tc.freq, tc.ch, got, tc.want)
		}
	}
}

func surveyWith(samples ...*SamplePoint) *Survey {
	floor := &Floor{ID: "f1", Name: "Floor 1", Level: 1, Samples: samples}
	return &Survey{
		ID:            "s1",
		Floors:        []*Floor{floor},
		ActiveFloorID: floor.ID,
		UpdatedAt:     time.Unix(500, 0),
	}
}

func TestSurveyBSSViewsDedupKeepsStrongest(t *testing.T) {
	now := time.Unix(1000, 0)
	later := time.Unix(2000, 0)
	s := surveyWith(
		&SamplePoint{Timestamp: now, SampleData: &PassiveSample{Networks: []*wifi.ScannedNetwork{
			{SSID: "corp", BSSID: "aa:bb:cc:00:00:01", Signal: -80, Channel: 6, Frequency: 2437, Security: "WPA2"},
		}}},
		&SamplePoint{Timestamp: later, SampleData: &PassiveSample{Networks: []*wifi.ScannedNetwork{
			{SSID: "corp", BSSID: "aa:bb:cc:00:00:01", Signal: -55, Channel: 6, Frequency: 2437, Security: "WPA2"},
		}}},
	)

	views, at := surveyBSSViews(s)
	if len(views) != 1 {
		t.Fatalf("views = %d, want 1 (deduped by BSSID)", len(views))
	}
	if views[0].SignalDBm != -55 {
		t.Errorf("kept signal = %d, want strongest -55", views[0].SignalDBm)
	}
	if views[0].Band != "2.4 GHz" {
		t.Errorf("band = %q, want 2.4 GHz", views[0].Band)
	}
	if !at.Equal(later) {
		t.Errorf("latest timestamp = %v, want %v", at, later)
	}
}

func TestAnalyzeAnomaliesDetectsOpenNetwork(t *testing.T) {
	s := surveyWith(
		&SamplePoint{Timestamp: time.Unix(1000, 0), SampleData: &PassiveSample{Networks: []*wifi.ScannedNetwork{
			{SSID: "guest", BSSID: "aa:bb:cc:00:00:01", Signal: -60, Channel: 1, Frequency: 2412, Security: "Open"},
		}}},
	)

	anoms, err := AnalyzeAnomalies(s)
	if err != nil {
		t.Fatalf("AnalyzeAnomalies: %v", err)
	}
	found := false
	for _, a := range anoms {
		if a.DefKey == wifianomaly.DefOpenNetwork {
			found = true
		}
	}
	if !found {
		t.Errorf("expected %s anomaly, got %+v", wifianomaly.DefOpenNetwork, anoms)
	}
}

func TestAnalyzeAnomaliesNoPassiveSamples(t *testing.T) {
	s := surveyWith(&SamplePoint{
		Timestamp:  time.Unix(1000, 0),
		SampleData: &ActiveSample{SSID: "corp", BSSID: "aa:bb:cc:00:00:01", RSSI: -50},
	})
	anoms, err := AnalyzeAnomalies(s)
	if err != nil {
		t.Fatalf("AnalyzeAnomalies: %v", err)
	}
	if len(anoms) != 0 {
		t.Errorf("anomalies = %v, want none (no passive APs)", anoms)
	}
}
