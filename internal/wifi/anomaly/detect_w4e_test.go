package wifianomaly_test

import (
	"testing"

	wifianomaly "github.com/MustardSeedNetworks/seed/internal/wifi/anomaly"
)

func TestDeauthFlood(t *testing.T) {
	det := wifianomaly.NewDetector()

	// At/above the default threshold (5) → flagged.
	flooded := bss("00:00:00:00:00:01", "corp", "WPA3", "5 GHz", 36)
	flooded.PMFRequired = true
	flooded.RecentDeauths = 5
	if !hasDef(det.Detect(tree("corp", flooded)), wifianomaly.DefDeauthFlood) {
		t.Error("a deauth burst at the threshold should raise deauth-flood")
	}

	// Below threshold (normal roam churn) → not flagged.
	calm := bss("00:00:00:00:00:02", "corp", "WPA3", "5 GHz", 40)
	calm.PMFRequired = true
	calm.RecentDeauths = 2
	if hasDef(det.Detect(tree("corp", calm)), wifianomaly.DefDeauthFlood) {
		t.Error("a handful of deauths must not raise deauth-flood")
	}
}

func TestDeauthFloodThresholdConfigurable(t *testing.T) {
	det := wifianomaly.NewDetector(wifianomaly.WithDeauthFloodThreshold(3))
	b := bss("00:00:00:00:00:01", "corp", "WPA3", "5 GHz", 36)
	b.PMFRequired = true
	b.RecentDeauths = 3
	if !hasDef(det.Detect(tree("corp", b)), wifianomaly.DefDeauthFlood) {
		t.Error("lowered threshold should flag at 3 deauths")
	}

	// Threshold floor: values below 2 clamp to 2.
	clamped := wifianomaly.NewDetector(wifianomaly.WithDeauthFloodThreshold(0))
	one := bss("00:00:00:00:00:02", "corp", "WPA3", "5 GHz", 40)
	one.PMFRequired = true
	one.RecentDeauths = 1
	if hasDef(clamped.Detect(tree("corp", one)), wifianomaly.DefDeauthFlood) {
		t.Error("threshold must clamp to a floor of 2; a single deauth must not flag")
	}
}
