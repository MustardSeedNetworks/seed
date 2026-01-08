//go:build darwin

// Package wifi_test provides wireless network information functionality tests.
// Test suite validates Darwin-specific WiFi scanning and parsing functionality.
package wifi_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

// TestParseAirportLineSecurityTypes tests parsing various security type formats.
func TestParseAirportLineSecurityTypes(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantNil    bool
		wantSec    string
		wantSignal int
	}{
		{
			name:       "WPA with PSK/AES",
			line:       "           SecureNet      aa:bb:cc:dd:ee:ff -45  6       Y  -- WPA(PSK/AES/AES)",
			wantNil:    false,
			wantSec:    "WPA",
			wantSignal: -45,
		},
		{
			name:       "WPA with TKIP",
			line:       "           LegacyNet      aa:bb:cc:dd:ee:ff -50  11      N  -- WPA(PSK/TKIP)",
			wantNil:    false,
			wantSec:    "WPA",
			wantSignal: -50,
		},
		{
			name:       "Open network",
			line:       "           GuestWiFi      00:11:22:33:44:55 -55  1       Y  -- Open",
			wantNil:    false,
			wantSec:    "Open",
			wantSignal: -55,
		},
		{
			name:       "WEP network",
			line:       "           OldRouter      11:22:33:44:55:66 -60  3       N  -- WEP",
			wantNil:    false,
			wantSec:    "WEP",
			wantSignal: -60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := wifi.ParseAirportLine(tt.line)

			if tt.wantNil {
				if network != nil {
					t.Errorf("ParseAirportLine() = %+v, want nil", network)
				}
				return
			}

			if network == nil {
				t.Fatal("ParseAirportLine() = nil, want non-nil")
			}

			if network.Security != tt.wantSec {
				t.Errorf("Security = %q, want %q", network.Security, tt.wantSec)
			}
			if network.Signal != tt.wantSignal {
				t.Errorf("Signal = %d, want %d", network.Signal, tt.wantSignal)
			}
		})
	}
}

// TestParseAirportLineHTModes tests parsing HT (High Throughput) flags.
func TestParseAirportLineHTModes(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantHTMode string
		wantWidth  int
	}{
		{
			name:       "HT enabled (Y)",
			line:       "           FastNet        aa:bb:cc:dd:ee:ff -40  36      Y  US WPA",
			wantHTMode: "HT40",
			wantWidth:  40,
		},
		{
			name:       "HT disabled (N)",
			line:       "           SlowNet        aa:bb:cc:dd:ee:ff -50  6       N  -- WPA",
			wantHTMode: "HT20",
			wantWidth:  20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := wifi.ParseAirportLine(tt.line)
			if network == nil {
				t.Fatal("ParseAirportLine() = nil, want non-nil")
			}

			if network.HTMode != tt.wantHTMode {
				t.Errorf("HTMode = %q, want %q", network.HTMode, tt.wantHTMode)
			}
			if network.ChannelWidth != tt.wantWidth {
				t.Errorf("ChannelWidth = %d, want %d", network.ChannelWidth, tt.wantWidth)
			}
		})
	}
}

// TestParseAirportLineChannelVariety tests various channel configurations.
func TestParseAirportLineChannelVariety(t *testing.T) {
	tests := []struct {
		name        string
		line        string
		wantChannel int
		wantFreq    int
		wantDFS     bool
	}{
		// 2.4 GHz channels
		{
			name:        "Channel 1",
			line:        "           Net1           aa:bb:cc:dd:ee:ff -45  1       Y  -- WPA",
			wantChannel: 1,
			wantFreq:    2412,
			wantDFS:     false,
		},
		{
			name:        "Channel 6",
			line:        "           Net6           aa:bb:cc:dd:ee:ff -50  6       Y  -- WPA",
			wantChannel: 6,
			wantFreq:    2437,
			wantDFS:     false,
		},
		{
			name:        "Channel 11",
			line:        "           Net11          aa:bb:cc:dd:ee:ff -55  11      Y  -- WPA",
			wantChannel: 11,
			wantFreq:    2462,
			wantDFS:     false,
		},
		// 5 GHz non-DFS channels
		{
			name:        "Channel 36 (non-DFS)",
			line:        "           Net36          aa:bb:cc:dd:ee:ff -40  36      Y  US WPA",
			wantChannel: 36,
			wantFreq:    5180,
			wantDFS:     false,
		},
		{
			name:        "Channel 149 (non-DFS)",
			line:        "           Net149         aa:bb:cc:dd:ee:ff -45  149     Y  US WPA",
			wantChannel: 149,
			wantFreq:    5745,
			wantDFS:     false,
		},
		// DFS channels
		{
			name:        "Channel 52 (DFS)",
			line:        "           NetDFS52       aa:bb:cc:dd:ee:ff -50  52      Y  US WPA",
			wantChannel: 52,
			wantFreq:    5260,
			wantDFS:     true,
		},
		{
			name:        "Channel 64 (DFS boundary)",
			line:        "           NetDFS64       aa:bb:cc:dd:ee:ff -50  64      Y  US WPA",
			wantChannel: 64,
			wantFreq:    5320,
			wantDFS:     true,
		},
		{
			name:        "Channel 100 (DFS)",
			line:        "           NetDFS100      aa:bb:cc:dd:ee:ff -50  100     Y  US WPA",
			wantChannel: 100,
			wantFreq:    5500,
			wantDFS:     true,
		},
		{
			name:        "Channel 144 (DFS boundary)",
			line:        "           NetDFS144      aa:bb:cc:dd:ee:ff -50  144     Y  JP WPA",
			wantChannel: 144,
			wantFreq:    5720,
			wantDFS:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := wifi.ParseAirportLine(tt.line)
			if network == nil {
				t.Fatal("ParseAirportLine() = nil, want non-nil")
			}

			if network.Channel != tt.wantChannel {
				t.Errorf("Channel = %d, want %d", network.Channel, tt.wantChannel)
			}
			if network.Frequency != tt.wantFreq {
				t.Errorf("Frequency = %d, want %d", network.Frequency, tt.wantFreq)
			}
			if network.IsDFS != tt.wantDFS {
				t.Errorf("IsDFS = %v, want %v", network.IsDFS, tt.wantDFS)
			}
		})
	}
}

// TestParseAirportLineSignalStrength tests various signal strengths.
func TestParseAirportLineSignalStrength(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantSignal int
		wantSNR    int
	}{
		{
			name:       "Excellent signal -25",
			line:       "           StrongNet      aa:bb:cc:dd:ee:ff -25  6       Y  -- WPA",
			wantSignal: -25,
			wantSNR:    70, // -25 - (-95) = 70
		},
		{
			name:       "Good signal -45",
			line:       "           GoodNet        aa:bb:cc:dd:ee:ff -45  6       Y  -- WPA",
			wantSignal: -45,
			wantSNR:    50, // -45 - (-95) = 50
		},
		{
			name:       "Fair signal -65",
			line:       "           FairNet        aa:bb:cc:dd:ee:ff -65  6       Y  -- WPA",
			wantSignal: -65,
			wantSNR:    30, // -65 - (-95) = 30
		},
		{
			name:       "Weak signal -80",
			line:       "           WeakNet        aa:bb:cc:dd:ee:ff -80  6       N  -- WPA",
			wantSignal: -80,
			wantSNR:    15, // -80 - (-95) = 15
		},
		{
			name:       "Very weak signal -90",
			line:       "           VeryWeakNet    aa:bb:cc:dd:ee:ff -90  6       N  -- Open",
			wantSignal: -90,
			wantSNR:    5, // -90 - (-95) = 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := wifi.ParseAirportLine(tt.line)
			if network == nil {
				t.Fatal("ParseAirportLine() = nil, want non-nil")
			}

			if network.Signal != tt.wantSignal {
				t.Errorf("Signal = %d, want %d", network.Signal, tt.wantSignal)
			}
			if network.SNR != tt.wantSNR {
				t.Errorf("SNR = %d, want %d", network.SNR, tt.wantSNR)
			}
		})
	}
}

// TestParseAirportLineInvalidFormats tests various invalid line formats.
func TestParseAirportLineInvalidFormats(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{"Empty string", ""},
		{"Only whitespace", "                    "},
		{"Missing BSSID", "           NetworkName                -45  6       Y  -- WPA2"},
		{"Invalid BSSID format", "           NetworkName     invalid    -45  6       Y  -- WPA2"},
		{"Missing signal", "           NetworkName     aa:bb:cc:dd:ee:ff      6       Y  -- WPA2"},
		{"Missing channel", "           NetworkName     aa:bb:cc:dd:ee:ff -45         Y  -- WPA2"},
		{"No security type", "           NetworkName     aa:bb:cc:dd:ee:ff -45  6       Y  -- "},
		{"Header line", "                         SSID BSSID             RSSI CHANNEL HT CC SECURITY"},
		{"Partial BSSID", "           NetworkName     aa:bb:cc       -45  6       Y  -- WPA2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := wifi.ParseAirportLine(tt.line)
			if network != nil {
				t.Errorf("ParseAirportLine(%q) = %+v, want nil", tt.line, network)
			}
		})
	}
}

// TestParseAirportLineSSIDVariations tests SSID parsing variations.
func TestParseAirportLineSSIDVariations(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantSSID string
	}{
		{
			name:     "Simple SSID",
			line:     "           MyNetwork      aa:bb:cc:dd:ee:ff -45  6       Y  -- WPA",
			wantSSID: "MyNetwork",
		},
		{
			name:     "SSID with numbers",
			line:     "           Network123     aa:bb:cc:dd:ee:ff -45  6       Y  -- WPA",
			wantSSID: "Network123",
		},
		{
			name:     "SSID with dash",
			line:     "           My-Network     aa:bb:cc:dd:ee:ff -45  6       Y  -- WPA",
			wantSSID: "My-Network",
		},
		{
			name:     "SSID with underscore",
			line:     "           My_Network     aa:bb:cc:dd:ee:ff -45  6       Y  -- WPA",
			wantSSID: "My_Network",
		},
		{
			name:     "Short SSID",
			line:     "           N              aa:bb:cc:dd:ee:ff -45  6       Y  -- WPA",
			wantSSID: "N",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := wifi.ParseAirportLine(tt.line)
			if network == nil {
				t.Fatal("ParseAirportLine() = nil, want non-nil")
			}

			if network.SSID != tt.wantSSID {
				t.Errorf("SSID = %q, want %q", network.SSID, tt.wantSSID)
			}
		})
	}
}

// TestParseAirportLineBSSIDParsing tests BSSID parsing.
func TestParseAirportLineBSSIDParsing(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantBSSID string
	}{
		{
			name:      "Lowercase BSSID",
			line:      "           Network        aa:bb:cc:dd:ee:ff -45  6       Y  -- WPA",
			wantBSSID: "aa:bb:cc:dd:ee:ff",
		},
		{
			name:      "All zeros BSSID",
			line:      "           Network        00:00:00:00:00:00 -45  6       Y  -- WPA",
			wantBSSID: "00:00:00:00:00:00",
		},
		{
			name:      "All F's BSSID",
			line:      "           Network        ff:ff:ff:ff:ff:ff -45  6       Y  -- WPA",
			wantBSSID: "ff:ff:ff:ff:ff:ff",
		},
		{
			name:      "Mixed BSSID",
			line:      "           Network        12:34:56:78:9a:bc -45  6       Y  -- WPA",
			wantBSSID: "12:34:56:78:9a:bc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := wifi.ParseAirportLine(tt.line)
			if network == nil {
				t.Fatal("ParseAirportLine() = nil, want non-nil")
			}

			if network.BSSID != tt.wantBSSID {
				t.Errorf("BSSID = %q, want %q", network.BSSID, tt.wantBSSID)
			}
		})
	}
}

// TestIsDFSChannelBoundaries tests DFS channel boundary conditions.
func TestIsDFSChannelBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		channel int
		want    bool
	}{
		// UNII-2 DFS boundaries (52-64)
		{"Channel 51 (just before UNII-2)", 51, false},
		{"Channel 52 (UNII-2 start)", 52, true},
		{"Channel 64 (UNII-2 end)", 64, true},
		{"Channel 65 (just after UNII-2)", 65, false},

		// UNII-2E DFS boundaries (100-144)
		{"Channel 99 (just before UNII-2E)", 99, false},
		{"Channel 100 (UNII-2E start)", 100, true},
		{"Channel 144 (UNII-2E end)", 144, true},
		{"Channel 145 (just after UNII-2E)", 145, false},

		// Gap between UNII-2 and UNII-2E
		{"Channel 96 (gap)", 96, false},
		{"Channel 98 (gap)", 98, false},

		// Within ranges
		{"Channel 56 (mid UNII-2)", 56, true},
		{"Channel 60 (mid UNII-2)", 60, true},
		{"Channel 120 (mid UNII-2E)", 120, true},
		{"Channel 132 (mid UNII-2E)", 132, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wifi.IsDFSChannel(tt.channel)
			if got != tt.want {
				t.Errorf("IsDFSChannel(%d) = %v, want %v", tt.channel, got, tt.want)
			}
		})
	}
}

// TestParseAirportLineCountryCode tests parsing with various country codes.
func TestParseAirportLineCountryCode(t *testing.T) {
	tests := []struct {
		name string
		line string
	}{
		{
			name: "US country code",
			line: "           USNetwork      aa:bb:cc:dd:ee:ff -45  36      Y  US WPA",
		},
		{
			name: "JP country code",
			line: "           JPNetwork      aa:bb:cc:dd:ee:ff -45  14      N  JP WPA",
		},
		{
			name: "EU country code",
			line: "           EUNetwork      aa:bb:cc:dd:ee:ff -45  13      Y  EU WPA",
		},
		{
			name: "Dash as country code",
			line: "           GenericNet     aa:bb:cc:dd:ee:ff -45  6       Y  -- WPA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := wifi.ParseAirportLine(tt.line)
			if network == nil {
				t.Error("ParseAirportLine() = nil, want non-nil")
			}
		})
	}
}

// TestParseAirportLineNoiseFloorDefault tests default noise floor value.
func TestParseAirportLineNoiseFloorDefault(t *testing.T) {
	lines := []string{
		"           Net1           aa:bb:cc:dd:ee:ff -30  6       Y  -- WPA",
		"           Net2           aa:bb:cc:dd:ee:ff -50  11      Y  -- WPA",
		"           Net3           aa:bb:cc:dd:ee:ff -70  1       N  -- Open",
		"           Net4           aa:bb:cc:dd:ee:ff -90  3       N  -- WEP",
	}

	for _, line := range lines {
		t.Run(line[11:23], func(t *testing.T) {
			network := wifi.ParseAirportLine(line)
			if network == nil {
				t.Fatal("ParseAirportLine() = nil, want non-nil")
			}

			// Default noise floor should be -95 dBm
			const expectedNoiseFloor = -95
			if network.NoiseFloor != expectedNoiseFloor {
				t.Errorf("NoiseFloor = %d, want %d", network.NoiseFloor, expectedNoiseFloor)
			}

			// Verify SNR calculation
			expectedSNR := network.Signal - expectedNoiseFloor
			if network.SNR != expectedSNR {
				t.Errorf(
					"SNR = %d, want %d (Signal=%d, NoiseFloor=%d)",
					network.SNR,
					expectedSNR,
					network.Signal,
					expectedNoiseFloor,
				)
			}
		})
	}
}
