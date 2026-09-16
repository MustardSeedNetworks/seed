package api

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/wifi"
)

func TestWiFiObservationResponse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		observation wifi.Observation
		want        map[string]any
	}{
		{
			"associated",
			wifi.Observation{Status: wifi.StatusAssociated, Info: &wifi.Info{SSID: "lab", Channel: 44, Signal: -52}},
			map[string]any{
				"interface": "en0",
				"wireless":  true,
				"status":    "associated",
				"connected": true,
				"ssid":      "lab",
				"channel":   float64(44),
				"signal":    float64(-52),
			},
		},
		{
			"withheld omits any connection claim",
			wifi.Observation{
				Status:      wifi.StatusDetailsWithheld,
				Reason:      "location denied",
				Remediation: "check System Settings",
			},
			map[string]any{
				"interface":   "en0",
				"wireless":    true,
				"status":      "detailsWithheld",
				"reason":      "location denied",
				"remediation": "check System Settings",
			},
		},
		{
			"not associated",
			wifi.Observation{Status: wifi.StatusNotAssociated},
			map[string]any{"interface": "en0", "wireless": true, "status": "notAssociated", "connected": false},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raw, err := json.Marshal(wiFiStateResponse("en0", tt.observation))
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if decodeErr := json.Unmarshal(raw, &got); decodeErr != nil {
				t.Fatal(decodeErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("response = %s, want %+v", raw, tt.want)
			}
		})
	}
}
