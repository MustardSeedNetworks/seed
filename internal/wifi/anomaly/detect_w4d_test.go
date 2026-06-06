package wifianomaly_test

import (
	"testing"

	wifianomaly "github.com/krisarmstrong/seed/internal/wifi/anomaly"
)

func TestBSSLoadSaturation(t *testing.T) {
	det := wifianomaly.NewDetector()

	hi := bss("00:00:00:00:00:01", "corp", "WPA3", "5 GHz", 36)
	hi.PMFRequired = true
	hi.HasBSSLoad = true
	hi.ChannelUtil = 200 // ~78% busy
	if !hasDef(det.Detect(tree("corp", hi)), wifianomaly.DefBSSLoadSaturation) {
		t.Error("high channel utilization should raise bss-load-saturation")
	}

	lo := bss("00:00:00:00:00:02", "corp", "WPA3", "5 GHz", 40)
	lo.PMFRequired = true
	lo.HasBSSLoad = true
	lo.ChannelUtil = 40 // ~16% busy
	if hasDef(det.Detect(tree("corp", lo)), wifianomaly.DefBSSLoadSaturation) {
		t.Error("low utilization must not raise bss-load-saturation")
	}

	// No BSS Load element → no detection even if ChannelUtil happens to be set.
	none := bss("00:00:00:00:00:03", "corp", "WPA3", "5 GHz", 44)
	none.PMFRequired = true
	none.ChannelUtil = 250
	if hasDef(det.Detect(tree("corp", none)), wifianomaly.DefBSSLoadSaturation) {
		t.Error("without HasBSSLoad the utilization figure is meaningless")
	}
}

func TestWideChannel24GHz(t *testing.T) {
	det := wifianomaly.NewDetector()

	wide := bss("00:00:00:00:00:01", "corp", "WPA3", "2.4 GHz", 6)
	wide.PMFRequired = true
	wide.ChannelWidthMHz = 40
	if !hasDef(det.Detect(tree("corp", wide)), wifianomaly.DefWideChannel24GHz) {
		t.Error("40 MHz in 2.4 GHz should be flagged")
	}

	narrow := bss("00:00:00:00:00:02", "corp", "WPA3", "2.4 GHz", 6)
	narrow.PMFRequired = true
	narrow.ChannelWidthMHz = 20
	if hasDef(det.Detect(tree("corp", narrow)), wifianomaly.DefWideChannel24GHz) {
		t.Error("20 MHz in 2.4 GHz is fine")
	}

	// 80 MHz in 5 GHz is normal — must not flag.
	fiveWide := bss("00:00:00:00:00:03", "corp", "WPA3", "5 GHz", 36)
	fiveWide.PMFRequired = true
	fiveWide.ChannelWidthMHz = 80
	if hasDef(det.Detect(tree("corp", fiveWide)), wifianomaly.DefWideChannel24GHz) {
		t.Error("wide channels in 5 GHz must not be flagged")
	}
}

func TestChannelWidthMismatch(t *testing.T) {
	det := wifianomaly.NewDetector()

	b20 := bss("00:00:00:00:00:01", "corp", "WPA3", "5 GHz", 36)
	b20.PMFRequired = true
	b20.ChannelWidthMHz = 20
	b80 := bss("00:00:00:00:00:02", "corp", "WPA3", "5 GHz", 40)
	b80.PMFRequired = true
	b80.ChannelWidthMHz = 80
	if !hasDef(det.Detect(tree("corp", b20, b80)), wifianomaly.DefChannelWidthMismatch) {
		t.Error("mixed channel widths under one SSID should be flagged")
	}

	// Uniform width → no mismatch.
	c1 := bss("00:00:00:00:00:03", "corp", "WPA3", "5 GHz", 36)
	c1.PMFRequired = true
	c1.ChannelWidthMHz = 80
	c2 := bss("00:00:00:00:00:04", "corp", "WPA3", "5 GHz", 149)
	c2.PMFRequired = true
	c2.ChannelWidthMHz = 80
	if hasDef(det.Detect(tree("corp", c1, c2)), wifianomaly.DefChannelWidthMismatch) {
		t.Error("uniform channel width must not be flagged")
	}
}

// sanity: multiple W4d signals on one BSS both fire.
func TestW4dMultipleSignalsOnOneBSS(t *testing.T) {
	det := wifianomaly.NewDetector()
	b := bss("00:00:00:00:00:01", "corp", "WPA3", "2.4 GHz", 6)
	b.PMFRequired = true
	b.ChannelWidthMHz = 40
	b.HasBSSLoad = true
	b.ChannelUtil = 220
	got := det.Detect(tree("corp", b))
	if !hasDef(got, wifianomaly.DefWideChannel24GHz) || !hasDef(got, wifianomaly.DefBSSLoadSaturation) {
		t.Errorf("expected both wide-channel and bss-load detections, got %v", defKeys(got))
	}
}
