package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery/bonjour"
)

func TestBonjourWindowParsing(t *testing.T) {
	tests := []struct {
		query   string
		want    time.Duration
		wantErr bool
	}{
		{query: "", want: 0},
		{query: "window=6", want: 6 * time.Second},
		{query: "window=0", wantErr: true},
		{query: "window=-3", wantErr: true},
		{query: "window=soon", wantErr: true},
	}
	for _, tc := range tests {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/discovery/bonjour?"+tc.query, nil)
		got, err := bonjourWindow(r)
		if tc.wantErr {
			if err == nil {
				t.Errorf("%q: err = nil, want an error", tc.query)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: err = %v, want nil", tc.query, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%q: window = %v, want %v", tc.query, got, tc.want)
		}
	}
}

func TestBonjourBrowseRejectsNonGET(t *testing.T) {
	s := &Server{}
	w := httptest.NewRecorder()
	s.handleBonjourBrowse(w, httptest.NewRequest(http.MethodPost, "/api/v1/discovery/bonjour", nil))

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestBonjourBrowseRejectsAnUnparsableWindowBeforeTouchingTheNetwork(t *testing.T) {
	// The check has to precede the browse: a bad parameter must not cost the
	// caller a multicast listen window first.
	s := &Server{}
	w := httptest.NewRecorder()
	started := time.Now()
	s.handleBonjourBrowse(w, httptest.NewRequest(http.MethodGet, "/api/v1/discovery/bonjour?window=forever", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Errorf("took %v: the window was opened before the parameter was rejected", elapsed)
	}
}

// TestBonjourResultSerialisesTheFieldsTheUIReads pins the wire contract the
// card is generated from: a renamed or dropped field here is a broken card,
// and the card's own types come from this shape.
func TestBonjourResultSerialisesTheFieldsTheUIReads(t *testing.T) {
	result := bonjour.BrowseResult{
		Interface:     "en0",
		LocalPrefixes: []string{"192.168.20.0/24"},
		ServiceTypes:  []string{"_airplay._tcp"},
		Services: []bonjour.ServiceInstance{{
			Instance:        "Living Room",
			Type:            "_airplay._tcp",
			Host:            "apple-tv.local",
			Port:            7000,
			Addresses:       []string{"192.168.20.40"},
			TXT:             map[string]string{"model": "AppleTV6,2"},
			Origin:          bonjour.OriginLocal,
			SourceAddresses: []string{"192.168.20.40"},
		}},
		ReflectorStatus: bonjour.ReflectorStatus{
			State:         bonjour.StateReflected,
			RemoteSubnets: []string{"10.44.40.0/24"},
			ForwardedBy:   []string{"192.168.20.1"},
		},
		ResponsesObserved: 4,
		DurationMs:        4001,
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if unmarshalErr := json.Unmarshal(encoded, &decoded); unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}

	for _, key := range []string{
		"interface", "localPrefixes", "serviceTypes", "services",
		"reflector", "responsesObserved", "truncated", "durationMs",
	} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("response has no %q field", key)
		}
	}
	// Duration is a Go type with no useful JSON form; durationMs is what the
	// card reads, so the raw field must stay out of the payload.
	if _, ok := decoded["Duration"]; ok {
		t.Error("response leaks the time.Duration field; the card reads durationMs")
	}

	svc := decoded["services"].([]any)[0].(map[string]any)
	for _, key := range []string{"instance", "type", "host", "port", "addresses", "txt", "origin", "sourceAddresses"} {
		if _, ok := svc[key]; !ok {
			t.Errorf("service has no %q field", key)
		}
	}
	reflector := decoded["reflector"].(map[string]any)
	if reflector["state"] != string(bonjour.StateReflected) {
		t.Errorf("reflector.state = %v, want %q", reflector["state"], bonjour.StateReflected)
	}
}
