package wifianomaly_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/wifi/airspace"
	wifianomaly "github.com/krisarmstrong/seed/internal/wifi/anomaly"
)

// reg builds a single-BSS tree on a 2.4 GHz channel with a declared country.
func reg(country string, ch int) []airspace.SSIDGroup {
	b := bss("00:00:00:00:00:01", "corp", "WPA3", "2.4 GHz", ch)
	b.PMFRequired = true
	b.CountryCode = country
	return tree("corp", b)
}

func TestRegulatoryViolation(t *testing.T) {
	det := wifianomaly.NewDetector()
	id := wifianomaly.DefRegulatoryViolation

	violations := []struct {
		country string
		ch      int
		why     string
	}{
		{"US", 12, "FCC forbids 2.4 GHz channel 12"},
		{"US", 13, "FCC forbids 2.4 GHz channel 13"},
		{"CA", 12, "Canada follows FCC (1-11)"},
		{"US", 14, "channel 14 is JP-only"},
		{"DE", 14, "channel 14 is JP-only even where 12-13 are fine"},
	}
	for _, v := range violations {
		if !hasDef(det.Detect(reg(v.country, v.ch)), id) {
			t.Errorf("expected regulatory-violation: %s (country=%s ch=%d)", v.why, v.country, v.ch)
		}
	}

	clean := []struct {
		country string
		ch      int
	}{
		{"US", 11}, // FCC legal
		{"JP", 13}, // JP permits 1-14
		{"JP", 14}, // JP permits 14
		{"DE", 13}, // EU permits 13
		{"GB", 6},  // common channel, any domain
	}
	for _, c := range clean {
		if hasDef(det.Detect(reg(c.country, c.ch)), id) {
			t.Errorf("false positive regulatory-violation for country=%s ch=%d", c.country, c.ch)
		}
	}

	// No country code → cannot assess → no detection.
	noCountry := bss("00:00:00:00:00:09", "corp", "WPA3", "2.4 GHz", 13)
	noCountry.PMFRequired = true
	if hasDef(det.Detect(tree("corp", noCountry)), id) {
		t.Error("must not flag a regulatory violation without a declared country")
	}

	// 5 GHz is out of scope (plans too country-specific to assert) → no detection.
	fiveGHz := bss("00:00:00:00:00:0a", "corp", "WPA3", "5 GHz", 36)
	fiveGHz.PMFRequired = true
	fiveGHz.CountryCode = "US"
	if hasDef(det.Detect(tree("corp", fiveGHz)), id) {
		t.Error("5 GHz must not raise regulatory-violation (out of scope)")
	}
}
