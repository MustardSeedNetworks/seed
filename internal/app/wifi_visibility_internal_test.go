package app

import (
	"errors"
	"fmt"
	"maps"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/wifi"
	"github.com/MustardSeedNetworks/seed/internal/wifi/airspace"
	wifianomaly "github.com/MustardSeedNetworks/seed/internal/wifi/anomaly"
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

// scanNet is a clean 5 GHz WPA3 network that trips no rule on its own; each
// case below changes only what its rule reads.
func scanNet(bssid, ssid string, edit func(n *wifi.ScannedNetwork)) *wifi.ScannedNetwork {
	n := &wifi.ScannedNetwork{
		SSID: ssid, BSSID: bssid, Signal: -60, Channel: 36, Frequency: 5180,
		Security: "WPA3", Standard: "802.11ax (Wi-Fi 6)", ChannelWidth: 80, PMFRequired: true,
	}
	if edit != nil {
		edit(n)
	}
	return n
}

func on24GHz(channel int) func(n *wifi.ScannedNetwork) {
	return func(n *wifi.ScannedNetwork) {
		n.Channel, n.Frequency, n.ChannelWidth = channel, 2407+5*channel, 20
	}
}

// scanCases holds, for every rule a managed-mode scan can feed, networks that
// fire it through the real scan mapping. BSSIDs are the RFC 7042 documentation
// range; 02:00:5e is a locally administered twin for the second vendor.
func scanCases(t *testing.T) map[string]struct {
	nets []*wifi.ScannedNetwork
	opts []wifianomaly.Option
} {
	t.Helper()
	type scanCase = struct {
		nets []*wifi.ScannedNetwork
		opts []wifianomaly.Option
	}
	one := func(edit func(n *wifi.ScannedNetwork)) scanCase {
		return scanCase{nets: []*wifi.ScannedNetwork{scanNet("00:00:5e:00:53:01", "corp", edit)}}
	}
	pair := func(a, b func(n *wifi.ScannedNetwork)) scanCase {
		return scanCase{nets: []*wifi.ScannedNetwork{
			scanNet("00:00:5e:00:53:01", "corp", a), scanNet("00:00:5e:00:53:21", "corp", b),
		}}
	}
	many := func(n int, ssid func(i int) string) scanCase {
		var c scanCase
		for i := range n {
			c.nets = append(c.nets, scanNet(fmt.Sprintf("00:00:5e:00:53:%02x", 0x30+i), ssid(i), nil))
		}
		return c
	}
	return map[string]scanCase{
		wifianomaly.DefOpenNetwork:             one(func(n *wifi.ScannedNetwork) { n.Security = "Open" }),
		wifianomaly.DefWEPInUse:                one(func(n *wifi.ScannedNetwork) { n.Security = "WEP" }),
		wifianomaly.DefWPA3TransitionDowngrade: one(func(n *wifi.ScannedNetwork) { n.Security = "WPA2/WPA3" }),
		wifianomaly.DefWPSEnabled:              one(func(n *wifi.ScannedNetwork) { n.WPSEnabled = true }),
		wifianomaly.DefPMFNotRequired:          one(func(n *wifi.ScannedNetwork) { n.PMFRequired = false }),
		wifianomaly.DefHiddenSSID:              one(func(n *wifi.ScannedNetwork) { n.SSID, n.Hidden = "", true }),
		wifianomaly.DefAdjacentChannelOverlap:  one(on24GHz(3)),
		wifianomaly.DefWideChannel24GHz: one(func(n *wifi.ScannedNetwork) {
			on24GHz(6)(n)
			n.ChannelWidth = 40
		}),
		wifianomaly.DefBSSLoadSaturation: one(func(n *wifi.ScannedNetwork) { n.HasBSSLoad, n.ChannelUtil = true, 200 }),
		wifianomaly.DefRegulatoryViolation: one(func(n *wifi.ScannedNetwork) {
			on24GHz(13)(n)
			n.CountryCode = "US"
		}),
		wifianomaly.DefDefaultSSIDName:  one(func(n *wifi.ScannedNetwork) { n.SSID = "linksys" }),
		wifianomaly.DefSecurityMismatch: pair(nil, func(n *wifi.ScannedNetwork) { n.Security = "Open" }),
		wifianomaly.DefEvilTwin: {nets: []*wifi.ScannedNetwork{
			scanNet("00:00:5e:00:53:01", "corp", nil), scanNet("02:00:5e:00:53:01", "corp", nil),
		}},
		wifianomaly.DefStandardMismatch: pair(
			nil,
			func(n *wifi.ScannedNetwork) { n.Standard = "802.11ac (Wi-Fi 5)" },
		),
		wifianomaly.DefChannelWidthMismatch: pair(nil, func(n *wifi.ScannedNetwork) { n.ChannelWidth = 20 }),
		wifianomaly.DefInconsistentRoaming:  pair(nil, func(n *wifi.ScannedNetwork) { n.FTSupported = true }),
		wifianomaly.DefCountryConflict: pair(
			func(n *wifi.ScannedNetwork) { n.CountryCode = "US" },
			func(n *wifi.ScannedNetwork) { n.CountryCode = "DE" },
		),
		wifianomaly.DefCoChannelContention: many(4, func(i int) string { return fmt.Sprintf("net%d", i) }),
		// The AP key masks the low nibble, so these five BSSIDs are one AP.
		wifianomaly.DefSSIDSprawl: {nets: []*wifi.ScannedNetwork{
			scanNet("00:00:5e:00:53:40", "a", nil), scanNet("00:00:5e:00:53:41", "b", nil),
			scanNet("00:00:5e:00:53:42", "c", nil), scanNet("00:00:5e:00:53:43", "d", nil),
			scanNet("00:00:5e:00:53:44", "e", nil),
		}},
		wifianomaly.DefRogueAPOnLAN: {
			nets: []*wifi.ScannedNetwork{scanNet("00:00:5e:00:53:01", "corp", nil)},
			opts: []wifianomaly.Option{wifianomaly.WithWiredMACs("00:00:5e:00:53:01")},
		},
	}
}

func scanFires(t *testing.T, nets []*wifi.ScannedNetwork, opts []wifianomaly.Option) map[string]bool {
	t.Helper()
	src := WiFiScanSource(
		troubleshooting.NewManagement(fakeRadio{wireless: true, networks: nets}, nil, fixedInterface("wlan0")),
	)
	views, err := src()
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	fired := map[string]bool{}
	for _, d := range wifianomaly.NewDetector(opts...).Detect(airspace.TreeFromBSSViews(views)) {
		fired[d.DefKey] = true
	}
	return fired
}

// TestCaptureOnlyRulesAreExactlyTheOnesAScanCannotFeed partitions the catalog:
// every rule either fires from a managed-mode scan or is listed as capture-only,
// never both. A new frame- or station-level rule fails here until it is listed,
// so the status cannot report it as clear on a host with no monitor mode.
func TestCaptureOnlyRulesAreExactlyTheOnesAScanCannotFeed(t *testing.T) {
	captureOnly := map[string]bool{}
	for _, r := range wifianomaly.CaptureOnlyRules() {
		captureOnly[r.ID] = true
	}
	cases := scanCases(t)
	everyScan := map[string]bool{}
	for _, c := range cases {
		maps.Copy(everyScan, scanFires(t, c.nets, c.opts))
	}
	for _, d := range wifianomaly.Defs() {
		t.Run(d.ID, func(t *testing.T) {
			c, hasCase := cases[d.ID]
			switch {
			case captureOnly[d.ID] && hasCase:
				t.Error("listed as capture-only but has a scan case")
			case captureOnly[d.ID] && everyScan[d.ID]:
				t.Error("listed as capture-only but a scan fired it")
			case !captureOnly[d.ID] && !hasCase:
				t.Error("no scan case: add one, or list the rule in CaptureOnlyRules")
			case !captureOnly[d.ID] && !scanFires(t, c.nets, c.opts)[d.ID]:
				t.Error("its scan case did not fire it")
			}
		})
	}
}
