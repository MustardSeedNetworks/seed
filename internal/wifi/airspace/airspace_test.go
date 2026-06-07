package airspace_test

import (
	"net"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/wifi/airspace"
	"github.com/MustardSeedNetworks/seed/internal/wifi/dot11"
)

func mac(t *testing.T, s string) net.HardwareAddr {
	t.Helper()
	hw, err := net.ParseMAC(s)
	if err != nil {
		t.Fatalf("ParseMAC(%q): %v", s, err)
	}
	return hw
}

func beacon(t *testing.T, bssid, ssid string, info dot11.BSS) *dot11.Frame {
	t.Helper()
	info.SSID = ssid
	return &dot11.Frame{
		Kind:       dot11.KindBeacon,
		BSSID:      mac(t, bssid),
		Band:       dot11.Band24GHz,
		ChannelNum: 6,
		SignalDBm:  -50,
		BSS:        &info,
	}
}

// dataFromClient builds a client→AP data frame (ToDS): Address1=BSSID,
// Address2=client.
func dataFromClient(t *testing.T, bssid, client string) *dot11.Frame {
	t.Helper()
	return &dot11.Frame{
		Kind:        dot11.KindData,
		ToDS:        true,
		Receiver:    mac(t, bssid),
		Transmitter: mac(t, client),
		SignalDBm:   -60,
	}
}

func baseTime() time.Time { return time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC) }

func TestSSIDGroupsTwoRadiosOneAP(t *testing.T) {
	t.Parallel()
	a := airspace.New()
	// Two BSSIDs of the same physical AP (last octet within the group mask)
	// advertising the same SSID across bands.
	a.Observe(
		beacon(
			t,
			"00:11:22:33:44:01",
			"CorpWiFi",
			dot11.BSS{Security: dot11.SecurityWPA2, Standard: dot11.Standard80211ax, PMFRequired: true},
		),
		baseTime(),
	)
	a.Observe(
		beacon(
			t,
			"00:11:22:33:44:09",
			"CorpWiFi",
			dot11.BSS{Security: dot11.SecurityWPA2, Standard: dot11.Standard80211ax},
		),
		baseTime(),
	)
	// A client transmitting through the first BSSID.
	a.Observe(dataFromClient(t, "00:11:22:33:44:01", "aa:bb:cc:dd:ee:ff"), baseTime())

	tree := a.Tree()
	if len(tree) != 1 {
		t.Fatalf("got %d SSID groups, want 1", len(tree))
	}
	g := tree[0]
	if g.SSID != "CorpWiFi" {
		t.Errorf("SSID = %q, want CorpWiFi", g.SSID)
	}
	if g.APCount != 1 {
		t.Errorf("APCount = %d, want 1 (two radios cluster to one AP)", g.APCount)
	}
	if g.BSSCount != 2 {
		t.Errorf("BSSCount = %d, want 2", g.BSSCount)
	}
	if g.StationCount != 1 {
		t.Errorf("StationCount = %d, want 1", g.StationCount)
	}
	// The client is under the BSSID it transmitted through.
	ap := g.APs[0]
	var found bool
	for _, b := range ap.BSSes {
		if b.BSSID == "00:11:22:33:44:01" {
			if len(b.Stations) != 1 || b.Stations[0].MAC != "aa:bb:cc:dd:ee:ff" {
				t.Errorf("client not attributed to BSSID 01: %+v", b.Stations)
			}
			found = true
		}
	}
	if !found {
		t.Error("BSSID 00:11:22:33:44:01 missing from tree")
	}
}

func TestDistinctAPsSameSSID(t *testing.T) {
	t.Parallel()
	a := airspace.New()
	a.Observe(beacon(t, "00:11:22:33:44:01", "Campus", dot11.BSS{Security: dot11.SecurityWPA3}), baseTime())
	a.Observe(beacon(t, "00:11:22:33:44:21", "Campus", dot11.BSS{Security: dot11.SecurityWPA3}), baseTime())

	tree := a.Tree()
	if len(tree) != 1 || tree[0].APCount != 2 {
		t.Fatalf("want 1 SSID with 2 APs, got %d SSIDs / APCount %d", len(tree), groupAPCount(tree))
	}
}

func TestBeaconFieldsDecodedIntoView(t *testing.T) {
	t.Parallel()
	a := airspace.New()
	a.Observe(beacon(t, "00:11:22:33:44:01", "Secure", dot11.BSS{
		Security: dot11.SecurityWPA3, Standard: dot11.Standard80211be,
		PMFRequired: true, RRMNeighbor: true, BTMSupported: true, FTSupported: true, CountryCode: "US",
	}), baseTime())

	b := a.Tree()[0].APs[0].BSSes[0]
	if b.Security != "WPA3" || b.Standard != "802.11be (Wi-Fi 7)" {
		t.Errorf("security/standard = %q/%q", b.Security, b.Standard)
	}
	if !b.PMFRequired || !b.RRMNeighbor || !b.BTMSupported || !b.FTSupported {
		t.Errorf("capability flags not carried: %+v", b)
	}
	if b.CountryCode != "US" || b.Band != "2.4 GHz" || b.Channel != 6 {
		t.Errorf("country/band/channel = %q/%q/%d", b.CountryCode, b.Band, b.Channel)
	}
}

func TestClientOnlyBSSThenBeacon(t *testing.T) {
	t.Parallel()
	a := airspace.New()
	// Client seen before any beacon → a thin BSS is created so the client isn't lost.
	a.Observe(dataFromClient(t, "00:11:22:33:44:01", "aa:bb:cc:dd:ee:ff"), baseTime())
	if a.Tree()[0].StationCount != 1 {
		t.Fatal("client-only BSS should still carry its station")
	}
	// Beacon later fills in the SSID.
	a.Observe(beacon(t, "00:11:22:33:44:01", "Late", dot11.BSS{Security: dot11.SecurityWPA2}), baseTime())
	tree := a.Tree()
	if len(tree) != 1 || tree[0].SSID != "Late" || tree[0].StationCount != 1 {
		t.Errorf("beacon should fill SSID + keep the client: %+v", tree)
	}
}

func TestPruneAgesOutStaleStationsAndClientOnlyBSS(t *testing.T) {
	t.Parallel()
	a := airspace.New()
	a.Observe(dataFromClient(t, "00:11:22:33:44:01", "aa:bb:cc:dd:ee:ff"), baseTime())
	// A beaconed BSS with a stale client.
	a.Observe(beacon(t, "00:11:22:33:44:02", "Keep", dot11.BSS{Security: dot11.SecurityWPA2}), baseTime())
	a.Observe(dataFromClient(t, "00:11:22:33:44:02", "11:22:33:44:55:66"), baseTime())

	a.Prune(baseTime().Add(time.Minute)) // cutoff after baseTime → everything stale

	tree := a.Tree()
	// Client-only BSS (01) is gone; beaconed BSS (02) stays but loses its client.
	for _, g := range tree {
		for _, ap := range g.APs {
			for _, b := range ap.BSSes {
				if b.BSSID == "00:11:22:33:44:01" {
					t.Error("client-only stale BSS should be pruned")
				}
				if b.BSSID == "00:11:22:33:44:02" && len(b.Stations) != 0 {
					t.Error("stale station should be pruned from beaconed BSS")
				}
			}
		}
	}
}

func TestTreeIsDeterministic(t *testing.T) {
	t.Parallel()
	a := airspace.New()
	a.Observe(beacon(t, "00:11:22:33:44:21", "Zeta", dot11.BSS{Security: dot11.SecurityWPA2}), baseTime())
	a.Observe(beacon(t, "00:11:22:33:44:01", "Alpha", dot11.BSS{Security: dot11.SecurityWPA2}), baseTime())
	first := a.Tree()
	second := a.Tree()
	if len(first) != 2 || first[0].SSID != "Alpha" || first[1].SSID != "Zeta" {
		t.Fatalf("SSID groups not sorted: %+v", first)
	}
	if first[0].SSID != second[0].SSID || first[1].SSID != second[1].SSID {
		t.Error("Tree() ordering is not stable across calls")
	}
}

func TestNilFrameIsNoOp(t *testing.T) {
	t.Parallel()
	a := airspace.New()
	a.Observe(nil, baseTime())
	if len(a.Tree()) != 0 {
		t.Error("nil frame should not create any state")
	}
}

func groupAPCount(tree []airspace.SSIDGroup) int {
	if len(tree) == 0 {
		return 0
	}
	return tree[0].APCount
}
