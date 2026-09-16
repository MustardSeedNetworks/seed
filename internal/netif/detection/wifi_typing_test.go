package detection_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/netif/detection"
)

// TestDetectTypeConsultsThePlatform pins #2670's second defect: on macOS the
// Wi-Fi adapter is en0, which every name pattern reads as Ethernet. The name is
// not evidence on that platform; the Wi-Fi interface list is.
func TestDetectTypeConsultsThePlatform(t *testing.T) {
	tests := []struct {
		name     string
		iface    string
		wireless []string
		want     string
	}{
		{"macOS Wi-Fi adapter named like an Ethernet one", "en0", []string{"en0"}, "wifi"},
		{"a real Ethernet port on the same host", "en5", []string{"en0"}, "ethernet"},
		{"name patterns still decide where the platform knows nothing", "wlan0", nil, "wifi"},
		{"virtual interfaces are unaffected", "utun3", []string{"en0"}, "virtual"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detection.DetectTypeWithWirelessForTest(tt.iface, tt.wireless); got != tt.want {
				t.Errorf("detectType(%q) with wireless %v = %q, want %q",
					tt.iface, tt.wireless, got, tt.want)
			}
		})
	}
}

// TestWirelessSpeedIsNotBucketedToEthernetSteps pins the "228 Mbps shown as
// 100 Mbps" half of #2670. An Ethernet link rate is one of a few fixed steps;
// a Wi-Fi transmit rate is continuous, so rounding it down to the nearest
// Ethernet step understates it by more than half.
func TestWirelessSpeedIsNotBucketedToEthernetSteps(t *testing.T) {
	tests := []struct {
		name   string
		bps    int64
		ifType string
		want   string
	}{
		{"the reported association", 228_540_000, "wifi", "229 Mbps"},
		{"a slow association", 65_000_000, "wifi", "65 Mbps"},
		{"a fast association", 1_200_000_000, "wifi", "1.2 Gbps"},
		{"no rate reported", 0, "wifi", ""},
		{"ethernet keeps its fixed steps", 228_540_000, "ethernet", "100 Mbps"},
		{"gigabit ethernet is exact", 1_000_000_000, "ethernet", "1 Gbps"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := detection.FormatLinkSpeedForTest(tt.bps, tt.ifType); got != tt.want {
				t.Errorf("formatLinkSpeed(%d, %q) = %q, want %q", tt.bps, tt.ifType, got, tt.want)
			}
		})
	}
}

// TestWirelessFriendlyNameNamesTheInterface: "WiFi Adapter" on every host is
// not a name, and the header on the reported Mac read "100 Mbps Ethernet".
func TestWirelessFriendlyNameNamesTheInterface(t *testing.T) {
	got := detection.FriendlyNameForTest("en0", "wifi", "229 Mbps")
	if want := "Wi-Fi (en0)"; got != want {
		t.Errorf("friendly name = %q, want %q", got, want)
	}
}
