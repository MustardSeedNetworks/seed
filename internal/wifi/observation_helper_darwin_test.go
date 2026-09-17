//go:build darwin

package wifi_test

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/foundation/pkg/corewlan"

	"github.com/MustardSeedNetworks/seed/internal/wifi"
	"github.com/MustardSeedNetworks/seed/internal/wifi/wifihelper"
)

type observationHelper struct {
	network wifihelper.Network
	err     error
}

func (h observationHelper) Current() (wifihelper.Network, error) { return h.network, h.err }
func (h observationHelper) Scan() ([]wifihelper.Network, error) {
	return []wifihelper.Network{h.network}, h.err
}
func (h observationHelper) Saved() ([]string, error) { return nil, h.err }

func TestObserveDarwinHelper(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		helper wifi.Helper
		want   wifi.Status
	}{
		{"helper absent", nil, wifi.StatusDetailsWithheld},
		{
			"helper has no connected agent",
			observationHelper{err: errors.New("no helper agent connected")},
			wifi.StatusDetailsWithheld,
		},
		{
			"authorized helper",
			observationHelper{network: wifihelper.Network{SSID: "lab", Channel: 44, RSSI: -52, Band: 5}},
			wifi.StatusAssociated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := wifi.ObserveCurrentForTest(
				func() (*corewlan.Network, error) { return nil, corewlan.ErrLocationDenied },
				tt.helper,
			)
			if got.Status != tt.want {
				t.Fatalf("status = %q, want %q", got.Status, tt.want)
			}
			if got.Status == wifi.StatusAssociated &&
				(got.Info.SSID != "lab" || got.Info.Channel != 44 || got.Info.Signal != -52 || got.Info.Frequency != 5220) {
				t.Fatalf("helper association = %+v", got.Info)
			}
			if got.Status == wifi.StatusDetailsWithheld && (got.Info != nil || got.Remediation == "") {
				t.Fatalf("withheld association = %+v", got)
			}
		})
	}
}

func TestScanDarwinWithheldAndHelper(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		directErr error
		helper    wifi.Helper
		withheld  bool
		wantSSID  string
	}{
		{"helper absent", corewlan.ErrLocationDenied, nil, true, ""},
		{
			"helper has no connected agent",
			corewlan.ErrLocationDenied,
			observationHelper{err: errors.New("no helper agent connected")},
			true,
			"",
		},
		{
			"authorized helper",
			corewlan.ErrLocationDenied,
			observationHelper{network: wifihelper.Network{SSID: "lab", Channel: 44, RSSI: -52, Band: 5}},
			false,
			"lab",
		},
		{"unrelated scan error", corewlan.ErrNoInterface, nil, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := wifi.ScanCurrentForTest(
				func() ([]corewlan.Network, error) { return nil, tt.directErr },
				tt.helper,
			)
			if errors.Is(err, wifi.ErrDetailsWithheld) != tt.withheld {
				t.Fatalf("scan error = %v, withheld want %t", err, tt.withheld)
			}
			if tt.wantSSID != "" {
				if err != nil || len(got) != 1 || got[0].SSID != tt.wantSSID {
					t.Fatalf("scan = %+v, err = %v", got, err)
				}
			} else if err == nil || got != nil {
				t.Fatalf("failed scan = %+v, err = %v", got, err)
			}
		})
	}
}
