package wifianomaly_test

import (
	"testing"

	wifianomaly "github.com/MustardSeedNetworks/seed/internal/wifi/anomaly"
)

func TestRogueAPOnLANDormantByDefault(t *testing.T) {
	det := wifianomaly.NewDetector()
	b := bss("aa:bb:cc:00:00:01", "corp", "WPA3", "5 GHz", 36)
	b.PMFRequired = true
	if hasDef(det.Detect(tree("corp", b)), wifianomaly.DefRogueAPOnLAN) {
		t.Error("with no wired-MAC set the rogue-AP rule must stay dormant")
	}
}

func TestRogueAPOnLANFlagsWiredBSSID(t *testing.T) {
	// The captured BSSID is also present on the wired side → bridged rogue AP.
	det := wifianomaly.NewDetector(wifianomaly.WithWiredMACs("AA:BB:CC:00:00:01", "11:22:33:44:55:66"))

	rogue := bss("aa:bb:cc:00:00:01", "corp", "WPA3", "5 GHz", 36)
	rogue.PMFRequired = true
	if !hasDef(det.Detect(tree("corp", rogue)), wifianomaly.DefRogueAPOnLAN) {
		t.Error("a BSSID present in the wired set should raise rogue-ap-on-lan (case-insensitive)")
	}

	// A BSSID not on the wired side must not flag.
	clean := bss("de:ad:be:ef:00:99", "corp", "WPA3", "5 GHz", 40)
	clean.PMFRequired = true
	if hasDef(det.Detect(tree("corp", clean)), wifianomaly.DefRogueAPOnLAN) {
		t.Error("a BSSID absent from the wired set must not be flagged")
	}
}
