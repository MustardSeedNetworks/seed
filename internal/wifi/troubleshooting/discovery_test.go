package troubleshooting_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
)

type fakeDiscoverySource struct {
	scanRes  *discovery.WiFiScanResult
	scanErr  error
	networks []discovery.WiFiNetwork
	aps      []discovery.WiFiAccessPoint
	stats    *discovery.WiFiDiscoveryStats
}

func (f fakeDiscoverySource) Scan(context.Context) (*discovery.WiFiScanResult, error) {
	return f.scanRes, f.scanErr
}
func (f fakeDiscoverySource) Networks() []discovery.WiFiNetwork         { return f.networks }
func (f fakeDiscoverySource) AccessPoints() []discovery.WiFiAccessPoint { return f.aps }
func (f fakeDiscoverySource) Stats() *discovery.WiFiDiscoveryStats      { return f.stats }

func TestDiscoveryNilSourceUnavailable(t *testing.T) {
	d := troubleshooting.NewDiscovery(nil)

	if _, err := d.Scan(context.Background()); !errors.Is(err, troubleshooting.ErrDiscoveryUnavailable) {
		t.Errorf("Scan err = %v, want ErrDiscoveryUnavailable", err)
	}
	if _, err := d.Networks(); !errors.Is(err, troubleshooting.ErrDiscoveryUnavailable) {
		t.Errorf("Networks err = %v, want ErrDiscoveryUnavailable", err)
	}
	if _, err := d.AccessPoints(); !errors.Is(err, troubleshooting.ErrDiscoveryUnavailable) {
		t.Errorf("AccessPoints err = %v, want ErrDiscoveryUnavailable", err)
	}
	if _, err := d.Stats(); !errors.Is(err, troubleshooting.ErrDiscoveryUnavailable) {
		t.Errorf("Stats err = %v, want ErrDiscoveryUnavailable", err)
	}
}

func TestDiscoveryDelegates(t *testing.T) {
	src := fakeDiscoverySource{
		scanRes:  &discovery.WiFiScanResult{Interface: "wlan0"},
		networks: []discovery.WiFiNetwork{{SSID: "office"}},
		aps:      []discovery.WiFiAccessPoint{{BSSID: "aa:bb"}},
		stats:    &discovery.WiFiDiscoveryStats{TotalNetworks: 1},
	}
	d := troubleshooting.NewDiscovery(src)

	res, err := d.Scan(context.Background())
	if err != nil || res.Interface != "wlan0" {
		t.Fatalf("Scan = %v, %v", res, err)
	}
	nets, err := d.Networks()
	if err != nil || len(nets) != 1 {
		t.Fatalf("Networks = %v, %v", nets, err)
	}
	aps, err := d.AccessPoints()
	if err != nil || len(aps) != 1 {
		t.Fatalf("AccessPoints = %v, %v", aps, err)
	}
	stats, err := d.Stats()
	if err != nil || stats.TotalNetworks != 1 {
		t.Fatalf("Stats = %v, %v", stats, err)
	}
}
