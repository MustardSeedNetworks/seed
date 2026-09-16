//go:build darwin

package wifi_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/corewlan"

	"github.com/MustardSeedNetworks/seed/internal/wifi"
)

// TestObserveDarwinThreeStates pins the distinction #2670 reported missing: a
// Mac that will not name the network is not a Mac that is not associated.
func TestObserveDarwinThreeStates(t *testing.T) {
	associated := &corewlan.Network{
		SSID: "lab-5g", BSSID: "aa:bb:cc:dd:ee:ff",
		RSSI: -52, Channel: 44, Band: 5, Security: "WPA2_PERSONAL",
	}

	tests := []struct {
		name        string
		current     func() (*corewlan.Network, error)
		wantStatus  wifi.Status
		wantSSID    string
		wantReason  bool
		wantInfoNil bool
	}{
		{
			name:       "associated reports the network",
			current:    func() (*corewlan.Network, error) { return associated, nil },
			wantStatus: wifi.StatusAssociated,
			wantSSID:   "lab-5g",
		},
		{
			name:        "location denied withholds details, never disconnects",
			current:     func() (*corewlan.Network, error) { return nil, corewlan.ErrLocationDenied },
			wantStatus:  wifi.StatusDetailsWithheld,
			wantReason:  true,
			wantInfoNil: true,
		},
		{
			name: "wrapped location denied is still withheld",
			current: func() (*corewlan.Network, error) {
				return nil, errors.Join(errors.New("current"), corewlan.ErrLocationDenied)
			},
			wantStatus:  wifi.StatusDetailsWithheld,
			wantReason:  true,
			wantInfoNil: true,
		},
		{
			name:        "not associated stays not associated",
			current:     func() (*corewlan.Network, error) { return nil, corewlan.ErrNotAssociated },
			wantStatus:  wifi.StatusNotAssociated,
			wantInfoNil: true,
		},
		{
			name:        "no adapter is not an association claim",
			current:     func() (*corewlan.Network, error) { return nil, corewlan.ErrNoInterface },
			wantStatus:  wifi.StatusNotAssociated,
			wantInfoNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wifi.ObserveCurrentForTest(tt.current, nil)

			if got.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", got.Status, tt.wantStatus)
			}
			assertObservedNetwork(t, got.Info, tt.wantInfoNil, tt.wantSSID)
			if tt.wantReason && (got.Reason == "" || got.Remediation == "") {
				t.Errorf("Reason/Remediation = %q/%q, want both set: the withheld state has to say why and what to do",
					got.Reason, got.Remediation)
			}
			if !tt.wantReason && got.Reason != "" {
				t.Errorf("Reason = %q, want empty", got.Reason)
			}
		})
	}
}

func assertObservedNetwork(t *testing.T, info *wifi.Info, wantNil bool, wantSSID string) {
	t.Helper()
	if wantNil {
		if info != nil {
			t.Fatalf("Info = %+v, want nil", info)
		}
		return
	}
	if info == nil {
		t.Fatal("Info = nil, want the associated network")
	}
	if info.SSID != wantSSID {
		t.Errorf("SSID = %q, want %q", info.SSID, wantSSID)
	}
}
