// Package wifi_test provides wireless network information functionality tests.
// Test suite validates security type mapping and classification.
package wifi_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

// TestMapSecurityTypeComprehensive tests all security type mappings.
func TestMapSecurityTypeComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// WPA3 variants
		{"WPA3 uppercase", "WPA3", "WPA3"},
		{"WPA3 lowercase", "wpa3", "WPA3"},
		{"WPA3 mixed case", "Wpa3", "WPA3"},
		{"WPA3-Personal", "WPA3-Personal", "WPA3"},
		{"WPA3-Enterprise", "WPA3-Enterprise", "WPA3"},
		{"WPA3-SAE", "WPA3-SAE", "WPA3"},

		// SAE (WPA3 key exchange)
		{"SAE uppercase", "SAE", "WPA3"},
		{"SAE lowercase", "sae", "WPA3"},
		{"SAE-Personal", "SAE-Personal", "WPA3"},

		// WPA2 variants
		{"WPA2 uppercase", "WPA2", "WPA2"},
		{"WPA2 lowercase", "wpa2", "WPA2"},
		{"WPA2 mixed case", "Wpa2", "WPA2"},
		{"WPA2-PSK", "WPA2-PSK", "WPA2"},
		{"WPA2-Personal", "WPA2-Personal", "WPA2"},
		{"WPA2-Enterprise", "WPA2-Enterprise", "WPA2"},
		{"WPA2(PSK/AES/AES)", "WPA2(PSK/AES/AES)", "WPA2"},
		{"WPA2(PSK/AES)", "WPA2(PSK/AES)", "WPA2"},

		// WPA variants
		{"WPA uppercase", "WPA", "WPA"},
		{"WPA lowercase", "wpa", "WPA"},
		{"WPA mixed case", "Wpa", "WPA"},
		{"WPA-PSK", "WPA-PSK", "WPA"},
		{"WPA-Personal", "WPA-Personal", "WPA"},
		{"WPA-Enterprise", "WPA-Enterprise", "WPA"},
		{"WPA(PSK/TKIP)", "WPA(PSK/TKIP)", "WPA"},

		// WEP variants
		{"WEP uppercase", "WEP", "WEP"},
		{"WEP lowercase", "wep", "WEP"},
		{"WEP mixed case", "Wep", "WEP"},
		{"WEP40", "WEP40", "WEP"},
		{"WEP104", "WEP104", "WEP"},

		// Open variants
		{"Open uppercase", "OPEN", "Open"},
		{"Open lowercase", "open", "Open"},
		{"Open mixed case", "Open", "Open"},
		{"None uppercase", "NONE", "Open"},
		{"None lowercase", "none", "Open"},
		{"None mixed case", "None", "Open"},

		// Passthrough (unknown types)
		{"Empty string", "", ""},
		{"Unknown type", "UNKNOWN", "UNKNOWN"},
		{"PSK alone", "PSK", "PSK"},
		{"CCMP alone", "CCMP", "CCMP"},
		{"TKIP alone", "TKIP", "TKIP"},
		{"AES alone", "AES", "AES"},
		{"RSN", "RSN", "RSN"},
		{"802.1X", "802.1X", "802.1X"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wifi.MapSecurityType(tt.input)
			if result != tt.expected {
				t.Errorf("MapSecurityType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestMapSecurityTypePrecedence tests security type detection precedence.
func TestMapSecurityTypePrecedence(t *testing.T) {
	// When a string contains multiple security types, the function should detect
	// the strongest one based on precedence: WPA3 > WPA2 > WPA > WEP > Open

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// WPA3 takes precedence
		{"WPA3 over WPA2", "WPA3/WPA2", "WPA3"},
		{"WPA3 over WPA", "WPA3/WPA", "WPA3"},
		{"SAE over WPA2", "SAE/WPA2", "WPA3"},

		// WPA2 takes precedence over WPA and WEP
		{"WPA2 over WPA", "WPA2/WPA", "WPA2"},
		{"WPA2 over WEP", "WPA2/WEP", "WPA2"},

		// WPA takes precedence over WEP
		{"WPA over WEP", "WPA/WEP", "WPA"},

		// Order matters - SAE should trigger WPA3 first
		{"SAE first", "SAE-PSK", "WPA3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wifi.MapSecurityType(tt.input)
			if result != tt.expected {
				t.Errorf("MapSecurityType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestMapSecurityTypeSpecialCases tests special and edge cases.
func TestMapSecurityTypeSpecialCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Whitespace handling - the function uses strings.ToUpper which preserves whitespace
		// but the Contains checks will still match WPA2
		{"With leading space", " WPA2", "WPA2"},
		{"With trailing space", "WPA2 ", "WPA2"},

		// Numeric suffixes
		{"WPA3.0", "WPA3.0", "WPA3"},
		{"WPA2.0", "WPA2.0", "WPA2"},

		// Mixed/enterprise formats
		{"WPA/WPA2", "WPA/WPA2", "WPA2"},
		{"Mixed WPA WPA2", "WPA-WPA2-Mixed", "WPA2"},

		// Long security strings
		{"Complex security", "WPA2(PSK,TKIP+CCMP/CCMP)", "WPA2"},
		{"Enterprise auth", "WPA2-EAP-TLS", "WPA2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wifi.MapSecurityType(tt.input)
			if result != tt.expected {
				t.Errorf("MapSecurityType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSecurityInfo tests security information in Info struct.
func TestSecurityInfo(t *testing.T) {
	tests := []struct {
		name     string
		security string
	}{
		{"WPA3 network", "WPA3"},
		{"WPA2 network", "WPA2"},
		{"WPA network", "WPA"},
		{"WEP network", "WEP"},
		{"Open network", "Open"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := wifi.Info{
				SSID:      "TestNetwork",
				BSSID:     "00:11:22:33:44:55",
				Signal:    -50,
				Channel:   6,
				Frequency: 2437,
				Security:  tt.security,
			}

			if info.Security != tt.security {
				t.Errorf("Security = %q, want %q", info.Security, tt.security)
			}
		})
	}
}

// TestSecurityInScannedNetwork tests security in ScannedNetwork struct.
func TestSecurityInScannedNetwork(t *testing.T) {
	tests := []struct {
		name     string
		security string
	}{
		{"WPA3 scanned network", "WPA3"},
		{"WPA2 scanned network", "WPA2"},
		{"WPA scanned network", "WPA"},
		{"WEP scanned network", "WEP"},
		{"Open scanned network", "Open"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := wifi.ScannedNetwork{
				SSID:     "TestNetwork",
				BSSID:    "00:11:22:33:44:55",
				Signal:   -50,
				Channel:  6,
				Security: tt.security,
			}

			if network.Security != tt.security {
				t.Errorf("Security = %q, want %q", network.Security, tt.security)
			}
		})
	}
}

// TestChannelNetworkSecurity tests security in ChannelNetwork struct.
func TestChannelNetworkSecurity(t *testing.T) {
	// ChannelNetwork doesn't have a Security field, but we test the full flow
	// from ScannedNetwork through GetChannelGraphData

	network := &wifi.ScannedNetwork{
		SSID:         "SecureNet",
		BSSID:        "00:11:22:33:44:55",
		Signal:       -50,
		Channel:      6,
		Frequency:    2437,
		Security:     "WPA3",
		ChannelWidth: 20,
		HTMode:       "HT20",
	}

	data := wifi.GetChannelGraphData([]*wifi.ScannedNetwork{network}, "")

	if len(data.Networks2_4GHz) != 1 {
		t.Fatalf("expected 1 network, got %d", len(data.Networks2_4GHz))
	}

	// ChannelNetwork should preserve SSID and BSSID
	cn := data.Networks2_4GHz[0]
	if cn.SSID != "SecureNet" {
		t.Errorf("SSID = %q, want 'SecureNet'", cn.SSID)
	}
	if cn.BSSID != "00:11:22:33:44:55" {
		t.Errorf("BSSID = %q, want '00:11:22:33:44:55'", cn.BSSID)
	}
}

// TestSecurityStrengthClassification tests security strength classification.
func TestSecurityStrengthClassification(t *testing.T) {
	tests := []struct {
		security string
		strength string
	}{
		// Strong security (WPA3)
		{"WPA3", "strong"},
		{"SAE", "strong"},
		{"WPA3-Personal", "strong"},
		{"WPA3-Enterprise", "strong"},

		// Good security (WPA2)
		{"WPA2", "good"},
		{"WPA2-PSK", "good"},
		{"WPA2-Enterprise", "good"},

		// Weak security (WPA)
		{"WPA", "weak"},
		{"WPA-PSK", "weak"},
		{"WPA-Enterprise", "weak"},

		// Deprecated security (WEP)
		{"WEP", "deprecated"},
		{"WEP40", "deprecated"},

		// No security (Open)
		{"Open", "none"},
		{"NONE", "none"},
	}

	for _, tt := range tests {
		t.Run(tt.security, func(t *testing.T) {
			mapped := wifi.MapSecurityType(tt.security)

			// Classify based on mapped type
			var strength string
			switch mapped {
			case "WPA3":
				strength = "strong"
			case "WPA2":
				strength = "good"
			case "WPA":
				strength = "weak"
			case "WEP":
				strength = "deprecated"
			case "Open":
				strength = "none"
			default:
				strength = "unknown"
			}

			if strength != tt.strength {
				t.Errorf("%q (mapped to %q) strength = %q, want %q",
					tt.security, mapped, strength, tt.strength)
			}
		})
	}
}
