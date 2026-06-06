package wifianomaly_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/wifi/airspace"
	wifianomaly "github.com/krisarmstrong/seed/internal/wifi/anomaly"
)

func TestWPA3TransitionDowngrade(t *testing.T) {
	det := wifianomaly.NewDetector()
	// Transition mode (WPA2/WPA3) → flagged; pure WPA3 → clean.
	tm := bss("00:00:00:00:00:01", "corp", "WPA2/WPA3", "5 GHz", 36)
	tm.PMFRequired = true
	if !hasDef(det.Detect(tree("corp", tm)), wifianomaly.DefWPA3TransitionDowngrade) {
		t.Error("WPA2/WPA3 transition mode should be flagged as downgrade-capable")
	}
	pure := bss("00:00:00:00:00:02", "corp", "WPA3", "5 GHz", 36)
	pure.PMFRequired = true
	if hasDef(det.Detect(tree("corp", pure)), wifianomaly.DefWPA3TransitionDowngrade) {
		t.Error("pure WPA3 must not be flagged as transition downgrade")
	}
}

func TestDefaultSSIDName(t *testing.T) {
	det := wifianomaly.NewDetector()
	for _, name := range []string{"NETGEAR47", "linksys", "D-Link_2.4G", "default"} {
		b := bss("00:00:00:00:00:01", name, "WPA2", "5 GHz", 36)
		b.PMFRequired = true
		if !hasDef(det.Detect(tree(name, b)), wifianomaly.DefDefaultSSIDName) {
			t.Errorf("SSID %q should be flagged as a default name", name)
		}
	}
	// A configured name is not a default.
	b := bss("00:00:00:00:00:01", "AcmeCorp-Staff", "WPA2", "5 GHz", 36)
	b.PMFRequired = true
	if hasDef(det.Detect(tree("AcmeCorp-Staff", b)), wifianomaly.DefDefaultSSIDName) {
		t.Error("a configured SSID must not be flagged as default")
	}
}

func TestSSIDSprawl(t *testing.T) {
	// One AP (shared key) advertising 3 SSIDs, threshold 3 → sprawl.
	mkTree := func(ssids ...string) []airspace.SSIDGroup {
		groups := make([]airspace.SSIDGroup, 0, len(ssids))
		for i, s := range ssids {
			b := bss("00:00:00:00:00:0"+string(rune('a'+i)), s, "WPA3", "5 GHz", 36)
			b.PMFRequired = true
			groups = append(groups, airspace.SSIDGroup{
				SSID:    s,
				APCount: 1,
				APs:     []airspace.APGroup{{Key: "shared-ap", Vendor: "Cisco", BSSes: []airspace.BSSView{b}}},
			})
		}
		return groups
	}
	det := wifianomaly.NewDetector(wifianomaly.WithSSIDSprawlThreshold(3))
	if hasDef(det.Detect(mkTree("a", "b")), wifianomaly.DefSSIDSprawl) {
		t.Error("2 SSIDs is below sprawl threshold 3")
	}
	if !hasDef(det.Detect(mkTree("a", "b", "c")), wifianomaly.DefSSIDSprawl) {
		t.Error("3 SSIDs on one AP meets sprawl threshold 3")
	}
}

func TestInconsistentRoaming(t *testing.T) {
	det := wifianomaly.NewDetector()
	ftYes := bss("00:00:00:00:00:01", "corp", "WPA3", "5 GHz", 36)
	ftYes.PMFRequired = true
	ftYes.FTSupported = true
	ftNo := bss("00:00:00:00:00:02", "corp", "WPA3", "5 GHz", 40)
	ftNo.PMFRequired = true
	ftNo.FTSupported = false
	if !hasDef(det.Detect(tree("corp", ftYes, ftNo)), wifianomaly.DefInconsistentRoaming) {
		t.Error("mixed 802.11r support across an SSID should be flagged")
	}

	// Uniform support → no inconsistency.
	a := bss("00:00:00:00:00:03", "corp", "WPA3", "5 GHz", 36)
	a.PMFRequired, a.FTSupported, a.RRMNeighbor, a.BTMSupported = true, true, true, true
	b := bss("00:00:00:00:00:04", "corp", "WPA3", "5 GHz", 40)
	b.PMFRequired, b.FTSupported, b.RRMNeighbor, b.BTMSupported = true, true, true, true
	if hasDef(det.Detect(tree("corp", a, b)), wifianomaly.DefInconsistentRoaming) {
		t.Error("uniform roaming support must not be flagged")
	}
}
