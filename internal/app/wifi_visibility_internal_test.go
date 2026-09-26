package app

import (
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/wifi"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
)

type fakeRadio struct {
	wireless bool
	networks []*wifi.ScannedNetwork
	err      error
}

func (f fakeRadio) ManagerAvailable() bool                { return true }
func (f fakeRadio) ScannerAvailable() bool                { return true }
func (f fakeRadio) IsWireless() bool                      { return f.wireless }
func (f fakeRadio) SetInterface(string)                   {}
func (f fakeRadio) Scan() ([]*wifi.ScannedNetwork, error) { return f.networks, f.err }
func (f fakeRadio) Connect(string, string) (*wifi.ConnectionResult, error) {
	return nil, errors.ErrUnsupported
}

func (f fakeRadio) Disconnect() (*wifi.ConnectionResult, error) { return nil, errors.ErrUnsupported }

type fixedInterface string

func (f fixedInterface) ResolvedWiFiInterface() string { return string(f) }
func (fixedInterface) SaveWiFiInterface(string) error  { return nil }

func TestWiFiScanSourceMapsScanResults(t *testing.T) {
	radio := fakeRadio{wireless: true, networks: []*wifi.ScannedNetwork{{
		SSID: "corp", BSSID: "00:00:5E:00:53:0A", Signal: -52, Channel: 36, Frequency: 5180,
		Security: "WPA2/WPA3", Standard: "802.11ax (Wi-Fi 6)", ChannelWidth: 80,
		CountryCode: "US", PMFRequired: true, FTSupported: true,
		HasBSSLoad: true, ChannelUtil: 200, AdvertisedStations: 7,
	}}}
	src := WiFiScanSource(troubleshooting.NewManagement(radio, nil, fixedInterface("wlan0")))

	views, err := src()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("got %d views, want 1", len(views))
	}
	v := views[0]
	if v.BSSID != "00:00:5e:00:53:0a" {
		t.Errorf("BSSID = %q, want the lowercase key capture uses", v.BSSID)
	}
	if v.Band != "5 GHz" || v.Channel != 36 || v.ChannelWidthMHz != 80 {
		t.Errorf("RF = %s ch %d %d MHz, want 5 GHz ch 36 80 MHz", v.Band, v.Channel, v.ChannelWidthMHz)
	}
	if v.Security != "WPA2/WPA3" || !v.PMFRequired || !v.FTSupported || v.CountryCode != "US" {
		t.Errorf("security fields not carried: %+v", v)
	}
	if !v.HasBSSLoad || v.ChannelUtil != 200 || v.AdvertisedStations != 7 || v.SignalDBm != -52 {
		t.Errorf("load/signal fields not carried: %+v", v)
	}
	if v.Stations == nil {
		t.Error("Stations is nil; the tree serializes it as [] for captured BSSes")
	}
}

func TestWiFiScanSourceReportsScansThatDidNotRun(t *testing.T) {
	for name, radio := range map[string]fakeRadio{
		"no wireless adapter": {wireless: false},
		"scan failed":         {wireless: true, err: errors.New("EBUSY")},
	} {
		t.Run(name, func(t *testing.T) {
			src := WiFiScanSource(troubleshooting.NewManagement(radio, nil, fixedInterface("wlan0")))
			if views, err := src(); err == nil {
				t.Errorf("got %d views and no error; an empty airspace would resolve live anomalies", len(views))
			}
		})
	}
}
