// Package detection provides intelligent network interface auto-detection.
// Test suite validates interface scoring, chipset identification, and capability detection.
package detection

import (
	"net"
	"testing"
)

func TestNewDetector(t *testing.T) {
	d := NewDetector()
	if d == nil {
		t.Fatal("NewDetector() returned nil")
	}

	if d.chipsetDB == nil {
		t.Error("chipsetDB should not be nil")
	}
}

func TestDetectAll(t *testing.T) {
	d := NewDetector()

	scores, err := d.DetectAll()
	if err != nil {
		t.Fatalf("DetectAll() error: %v", err)
	}

	// Should return some interfaces (at least loopback excluded)
	// Note: actual count depends on system
	t.Logf("Detected %d interfaces", len(scores))

	// Verify sorting (highest score first)
	for i := 1; i < len(scores); i++ {
		if scores[i].Score > scores[i-1].Score {
			t.Errorf("Interfaces not sorted: %s(%d) > %s(%d)",
				scores[i].Name, scores[i].Score,
				scores[i-1].Name, scores[i-1].Score)
		}
	}

	// Verify no loopback interfaces
	for _, score := range scores {
		if score.Name == "lo" || score.Name == "lo0" {
			t.Errorf("Loopback interface should be excluded: %s", score.Name)
		}
	}
}

func TestDetectBest(t *testing.T) {
	d := NewDetector()

	best, err := d.DetectBest()
	if err != nil {
		t.Fatalf("DetectBest() error: %v", err)
	}

	// May be nil if no interfaces exist (unlikely)
	if best != nil {
		t.Logf("Best interface: %s (score=%d)", best.Name, best.Score)
	}
}

func TestScoreInterface(t *testing.T) {
	d := NewDetector()

	// Get an actual interface for testing
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("net.Interfaces() error: %v", err)
	}

	for _, iface := range ifaces {
		// Skip loopback
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		score := d.ScoreInterface(iface)

		// Basic validation
		if score.Name != iface.Name {
			t.Errorf("Name mismatch: got %s, want %s", score.Name, iface.Name)
		}

		// Virtual interfaces should have negative score
		if score.Type == "virtual" && score.Score >= 0 {
			t.Errorf("Virtual interface %s should have negative score, got %d", score.Name, score.Score)
		}

		t.Logf("Interface %s: type=%s, score=%d, link=%v, ip=%v",
			score.Name, score.Type, score.Score, score.LinkStatus, score.HasIP)
	}
}

func TestDetectType(t *testing.T) {
	tests := []struct {
		name     string
		wantType string
	}{
		{"eth0", "ethernet"},
		{"eth1", "ethernet"},
		{"enp3s0", "ethernet"},
		{"ens192", "ethernet"},
		{"eno1", "ethernet"},
		{"em1", "ethernet"},
		{"en0", "ethernet"},

		{"wlan0", "wifi"},
		{"wlp2s0", "wifi"},
		{"wifi0", "wifi"},
		{"ath0", "wifi"},

		{"docker0", "virtual"},
		{"br-12345", "virtual"},
		{"veth123abc", "virtual"},
		{"virbr0", "virtual"},
		{"tun0", "virtual"},
		{"tap0", "virtual"},
		{"vmnet1", "virtual"},
		{"vboxnet0", "virtual"},
		{"utun0", "virtual"},

		{"sfp0", "fiber"},
		{"xfp0", "fiber"},

		{"unknown", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectType(tt.name)
			if got != tt.wantType {
				t.Errorf("detectType(%q) = %q, want %q", tt.name, got, tt.wantType)
			}
		})
	}
}

func TestHasRoutableAddress(t *testing.T) {
	tests := []struct {
		name      string
		addresses []string
		want      bool
	}{
		{
			name:      "routable IPv4",
			addresses: []string{"192.168.1.100/24"},
			want:      true,
		},
		{
			name:      "routable IPv6",
			addresses: []string{"2001:db8::1/64"},
			want:      true,
		},
		{
			name:      "link-local IPv4 only",
			addresses: []string{"169.254.1.1/16"},
			want:      false,
		},
		{
			name:      "link-local IPv6 only",
			addresses: []string{"fe80::1/64"},
			want:      false,
		},
		{
			name:      "loopback only",
			addresses: []string{"127.0.0.1/8"},
			want:      false,
		},
		{
			name:      "mixed with routable",
			addresses: []string{"fe80::1/64", "192.168.1.100/24"},
			want:      true,
		},
		{
			name:      "empty",
			addresses: []string{},
			want:      false,
		},
		{
			name:      "invalid address",
			addresses: []string{"not-an-ip"},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasRoutableAddress(tt.addresses)
			if got != tt.want {
				t.Errorf("hasRoutableAddress(%v) = %v, want %v", tt.addresses, got, tt.want)
			}
		})
	}
}

func TestFormatSpeed(t *testing.T) {
	tests := []struct {
		bps  int64
		want string
	}{
		{100_000_000_000, "100 Gbps"},
		{40_000_000_000, "40 Gbps"},
		{25_000_000_000, "25 Gbps"},
		{10_000_000_000, "10 Gbps"},
		{5_000_000_000, "5 Gbps"},
		{2_500_000_000, "2.5 Gbps"},
		{1_000_000_000, "1 Gbps"},
		{100_000_000, "100 Mbps"},
		{10_000_000, "10 Mbps"},
		{1_000_000, "< 10 Mbps"},
		{0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatSpeed(tt.bps)
			if got != tt.want {
				t.Errorf("formatSpeed(%d) = %q, want %q", tt.bps, got, tt.want)
			}
		})
	}
}

func TestCalculateScore(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name     string
		score    InterfaceScore
		wantMin  int
		wantMax  int
		wantSign int // 1 = positive, -1 = negative, 0 = any
	}{
		{
			name: "virtual interface",
			score: InterfaceScore{
				Type: "virtual",
			},
			wantSign: -1,
		},
		{
			name: "linked ethernet with IP",
			score: InterfaceScore{
				Type:       "ethernet",
				LinkStatus: true,
				HasIP:      true,
				Speed:      1_000_000_000,
			},
			wantMin: 1500, // 1000 (link) + 500 (IP) + ...
		},
		{
			name: "linked ethernet with TDR",
			score: InterfaceScore{
				Type:       "ethernet",
				LinkStatus: true,
				HasTDR:     true,
				Speed:      1_000_000_000,
			},
			wantMin: 2000, // 1000 (link) + 1000 (TDR) + ...
		},
		{
			name: "unlinked ethernet",
			score: InterfaceScore{
				Type:       "ethernet",
				LinkStatus: false,
				Speed:      1_000_000_000,
			},
			wantMax: 500, // No link bonus
		},
		{
			name: "10G with DOM",
			score: InterfaceScore{
				Type:       "fiber",
				LinkStatus: true,
				HasDOM:     true,
				Speed:      10_000_000_000,
			},
			wantMin: 1900, // 1000 + 500 (DOM) + 400 (10G) + ...
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.calculateScore(&tt.score)

			if tt.wantSign < 0 && got >= 0 {
				t.Errorf("calculateScore() = %d, want negative", got)
			}
			if tt.wantSign > 0 && got <= 0 {
				t.Errorf("calculateScore() = %d, want positive", got)
			}
			if tt.wantMin > 0 && got < tt.wantMin {
				t.Errorf("calculateScore() = %d, want >= %d", got, tt.wantMin)
			}
			if tt.wantMax > 0 && got > tt.wantMax {
				t.Errorf("calculateScore() = %d, want <= %d", got, tt.wantMax)
			}
		})
	}
}

func TestGenerateFriendlyName(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name  string
		score InterfaceScore
		want  string
	}{
		{
			name: "with chipset info",
			score: InterfaceScore{
				ChipsetVendor: "Intel",
				ChipsetModel:  "I225-V",
			},
			want: "Intel I225-V",
		},
		{
			name: "ethernet with speed",
			score: InterfaceScore{
				Type:         "ethernet",
				SpeedDisplay: "2.5 Gbps",
			},
			want: "2.5 Gbps Ethernet",
		},
		{
			name: "ethernet no speed",
			score: InterfaceScore{
				Type: "ethernet",
			},
			want: "Ethernet Adapter",
		},
		{
			name: "wifi",
			score: InterfaceScore{
				Type: "wifi",
			},
			want: "WiFi Adapter",
		},
		{
			name: "fallback to name",
			score: InterfaceScore{
				Name: "eth0",
				Type: "other",
			},
			want: "eth0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.generateFriendlyName(&tt.score)
			if got != tt.want {
				t.Errorf("generateFriendlyName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateDescription(t *testing.T) {
	d := NewDetector()

	tests := []struct {
		name  string
		score InterfaceScore
		want  string
	}{
		{
			name: "full description",
			score: InterfaceScore{
				Type:         "ethernet",
				SpeedDisplay: "1 Gbps",
				HasTDR:       true,
			},
			want: "1 Gbps Ethernet with TDR",
		},
		{
			name: "wifi no TDR",
			score: InterfaceScore{
				Type:         "wifi",
				SpeedDisplay: "WiFi6",
			},
			want: "WiFi6 WiFi",
		},
		{
			name:  "no info",
			score: InterfaceScore{},
			want:  "Network Interface",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.generateDescription(&tt.score)
			if got != tt.want {
				t.Errorf("generateDescription() = %q, want %q", got, tt.want)
			}
		})
	}
}
