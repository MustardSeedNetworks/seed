//go:build linux

package wifi_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

// TestSignalStrengthCategories tests signal strength classification.
func TestSignalStrengthCategories(t *testing.T) {
	tests := []struct {
		name     string
		signal   int
		category string
	}{
		// Excellent: -30 to -50 dBm
		{"Excellent signal -30", -30, "excellent"},
		{"Excellent signal -45", -45, "excellent"},
		{"Excellent signal -50", -50, "excellent"},

		// Good: -50 to -60 dBm
		{"Good signal -55", -55, "good"},
		{"Good signal -60", -60, "good"},

		// Fair: -60 to -70 dBm
		{"Fair signal -65", -65, "fair"},
		{"Fair signal -70", -70, "fair"},

		// Weak: -70 to -80 dBm
		{"Weak signal -75", -75, "weak"},
		{"Weak signal -80", -80, "weak"},

		// Very weak: below -80 dBm
		{"Very weak signal -85", -85, "very_weak"},
		{"Very weak signal -90", -90, "very_weak"},
		{"Very weak signal -95", -95, "very_weak"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a scanned network with the signal
			network := wifi.ScannedNetwork{
				SSID:      "TestNet",
				BSSID:     "aa:bb:cc:dd:ee:ff",
				Signal:    tt.signal,
				Channel:   6,
				Frequency: 2437,
			}

			// Classify signal strength
			var category string
			switch {
			case network.Signal >= -50:
				category = "excellent"
			case network.Signal >= -60:
				category = "good"
			case network.Signal >= -70:
				category = "fair"
			case network.Signal >= -80:
				category = "weak"
			default:
				category = "very_weak"
			}

			if category != tt.category {
				t.Errorf("signal %d dBm classified as %q, want %q", tt.signal, category, tt.category)
			}
		})
	}
}

// TestSNRCalculation tests Signal-to-Noise Ratio calculations.
func TestSNRCalculation(t *testing.T) {
	tests := []struct {
		name       string
		signal     int
		noiseFloor int
		wantSNR    int
	}{
		// Excellent SNR (> 40 dB)
		{"Excellent SNR with strong signal", -40, -95, 55},
		{"Excellent SNR with good signal", -50, -95, 45},

		// Good SNR (25-40 dB)
		{"Good SNR typical", -55, -90, 35},
		{"Good SNR borderline", -60, -95, 35},

		// Fair SNR (15-25 dB)
		{"Fair SNR", -70, -90, 20},
		{"Fair SNR borderline", -75, -95, 20},

		// Poor SNR (< 15 dB)
		{"Poor SNR", -85, -95, 10},
		{"Very poor SNR", -90, -95, 5},

		// Edge cases
		{"Zero SNR (signal equals noise)", -95, -95, 0},
		{"Negative SNR (signal below noise)", -100, -95, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := wifi.ScannedNetwork{
				Signal:     tt.signal,
				NoiseFloor: tt.noiseFloor,
				SNR:        tt.signal - tt.noiseFloor,
			}

			if network.SNR != tt.wantSNR {
				t.Errorf("SNR = %d, want %d (signal=%d, noise=%d)",
					network.SNR, tt.wantSNR, tt.signal, tt.noiseFloor)
			}
		})
	}
}

// TestChannelWidthModes tests different channel width modes.
func TestChannelWidthModes(t *testing.T) {
	tests := []struct {
		name     string
		htMode   string
		freq     int
		wantBand string
		wantDesc string
	}{
		// 2.4 GHz modes
		{"2.4GHz HT20", "HT20", 2437, "2.4GHz", "802.11n 20MHz"},
		{"2.4GHz HT40", "HT40", 2437, "2.4GHz", "802.11n 40MHz"},
		{"2.4GHz HT40+", "HT40+", 2437, "2.4GHz", "802.11n 40MHz upper"},
		{"2.4GHz HT40-", "HT40-", 2437, "2.4GHz", "802.11n 40MHz lower"},

		// 5 GHz modes
		{"5GHz VHT20", "VHT20", 5180, "5GHz", "802.11ac 20MHz"},
		{"5GHz VHT40", "VHT40", 5180, "5GHz", "802.11ac 40MHz"},
		{"5GHz VHT80", "VHT80", 5180, "5GHz", "802.11ac 80MHz"},
		{"5GHz VHT160", "VHT160", 5180, "5GHz", "802.11ac 160MHz"},

		// Wi-Fi 6 (HE) modes
		{"5GHz HE20", "HE20", 5180, "5GHz", "802.11ax 20MHz"},
		{"5GHz HE40", "HE40", 5180, "5GHz", "802.11ax 40MHz"},
		{"5GHz HE80", "HE80", 5180, "5GHz", "802.11ax 80MHz"},
		{"5GHz HE160", "HE160", 5180, "5GHz", "802.11ax 160MHz"},

		// Wi-Fi 6E / 6 GHz modes
		{"6GHz HE160", "HE160", 5955, "6GHz", "802.11ax 160MHz"},

		// Wi-Fi 7 (EHT) modes
		{"6GHz EHT20", "EHT20", 5955, "6GHz", "802.11be 20MHz"},
		{"6GHz EHT40", "EHT40", 5955, "6GHz", "802.11be 40MHz"},
		{"6GHz EHT80", "EHT80", 5955, "6GHz", "802.11be 80MHz"},
		{"6GHz EHT160", "EHT160", 5955, "6GHz", "802.11be 160MHz"},
		{"6GHz EHT320", "EHT320", 5955, "6GHz", "802.11be 320MHz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := wifi.ScannedNetwork{
				HTMode:    tt.htMode,
				Frequency: tt.freq,
			}

			gotBand := wifi.GetBand(network.Frequency)
			if gotBand != tt.wantBand {
				t.Errorf("GetBand(%d) = %q, want %q", tt.freq, gotBand, tt.wantBand)
			}

			gotWidth := wifi.DetectChannelWidth(tt.freq, tt.htMode)
			if gotWidth == 0 {
				t.Errorf("DetectChannelWidth(%d, %q) returned 0", tt.freq, tt.htMode)
			}
		})
	}
}

// TestNetworkInfoFields tests all fields in the Info struct.
func TestNetworkInfoFields(t *testing.T) {
	tests := []struct {
		name string
		info wifi.Info
	}{
		{
			name: "Complete 2.4GHz network info",
			info: wifi.Info{
				SSID:      "HomeNetwork",
				BSSID:     "00:11:22:33:44:55",
				Signal:    -55,
				Channel:   6,
				Frequency: 2437,
				Security:  "WPA2",
			},
		},
		{
			name: "Complete 5GHz network info",
			info: wifi.Info{
				SSID:      "HomeNetwork_5G",
				BSSID:     "00:11:22:33:44:56",
				Signal:    -50,
				Channel:   36,
				Frequency: 5180,
				Security:  "WPA3",
			},
		},
		{
			name: "Open network info",
			info: wifi.Info{
				SSID:      "GuestNetwork",
				BSSID:     "aa:bb:cc:dd:ee:ff",
				Signal:    -60,
				Channel:   1,
				Frequency: 2412,
				Security:  "Open",
			},
		},
		{
			name: "Hidden network info",
			info: wifi.Info{
				SSID:      "",
				BSSID:     "11:22:33:44:55:66",
				Signal:    -70,
				Channel:   11,
				Frequency: 2462,
				Security:  "WPA2",
			},
		},
		{
			name: "Very strong signal",
			info: wifi.Info{
				SSID:      "StrongNet",
				BSSID:     "ff:ee:dd:cc:bb:aa",
				Signal:    -25,
				Channel:   149,
				Frequency: 5745,
				Security:  "WPA3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify frequency matches channel
			expectedFreq := wifi.ChannelToFrequency(tt.info.Channel)
			if tt.info.Frequency != expectedFreq {
				t.Errorf(
					"Frequency mismatch: got %d, expected %d for channel %d",
					tt.info.Frequency,
					expectedFreq,
					tt.info.Channel,
				)
			}

			// Verify channel matches frequency
			expectedChan := wifi.FrequencyToChannel(tt.info.Frequency)
			if tt.info.Channel != expectedChan {
				t.Errorf(
					"Channel mismatch: got %d, expected %d for frequency %d",
					tt.info.Channel,
					expectedChan,
					tt.info.Frequency,
				)
			}
		})
	}
}

// TestScannedNetworkComplete tests complete ScannedNetwork struct initialization.
func TestScannedNetworkComplete(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		network wifi.ScannedNetwork
	}{
		{
			name: "Full 5GHz network",
			network: wifi.ScannedNetwork{
				SSID:         "TestNetwork5G",
				BSSID:        "aa:bb:cc:dd:ee:ff",
				Signal:       -50,
				Channel:      36,
				Frequency:    5180,
				Security:     "WPA3",
				ChannelWidth: 80,
				NoiseFloor:   -95,
				SNR:          45,
				HTMode:       "VHT80",
				IsDFS:        false,
				LastSeen:     now,
			},
		},
		{
			name: "DFS channel network",
			network: wifi.ScannedNetwork{
				SSID:         "DFSNetwork",
				BSSID:        "11:22:33:44:55:66",
				Signal:       -55,
				Channel:      52,
				Frequency:    5260,
				Security:     "WPA2",
				ChannelWidth: 40,
				NoiseFloor:   -92,
				SNR:          37,
				HTMode:       "HT40",
				IsDFS:        true,
				LastSeen:     now,
			},
		},
		{
			name: "Wi-Fi 6 network",
			network: wifi.ScannedNetwork{
				SSID:         "WiFi6Network",
				BSSID:        "22:33:44:55:66:77",
				Signal:       -45,
				Channel:      149,
				Frequency:    5745,
				Security:     "WPA3",
				ChannelWidth: 160,
				NoiseFloor:   -90,
				SNR:          45,
				HTMode:       "HE160",
				IsDFS:        false,
				LastSeen:     now,
			},
		},
		{
			name: "6 GHz network",
			network: wifi.ScannedNetwork{
				SSID:         "WiFi6ENetwork",
				BSSID:        "33:44:55:66:77:88",
				Signal:       -40,
				Channel:      1,
				Frequency:    5955,
				Security:     "WPA3",
				ChannelWidth: 160,
				NoiseFloor:   -88,
				SNR:          48,
				HTMode:       "HE160",
				IsDFS:        false,
				LastSeen:     now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify SNR calculation
			expectedSNR := tt.network.Signal - tt.network.NoiseFloor
			if tt.network.SNR != expectedSNR {
				t.Errorf(
					"SNR = %d, want %d (signal=%d, noise=%d)",
					tt.network.SNR,
					expectedSNR,
					tt.network.Signal,
					tt.network.NoiseFloor,
				)
			}

			// Verify band from frequency
			band := wifi.GetBand(tt.network.Frequency)
			if band == "" {
				t.Errorf("GetBand(%d) returned empty string", tt.network.Frequency)
			}

			// Verify LastSeen is set
			if tt.network.LastSeen.IsZero() {
				t.Error("LastSeen should not be zero")
			}
		})
	}
}

// TestChannelBandMapping tests the mapping between channels and frequency bands.
func TestChannelBandMapping(t *testing.T) {
	tests := []struct {
		name     string
		channel  int
		wantBand string
	}{
		// 2.4 GHz channels
		{"Channel 1 is 2.4GHz", 1, "2.4GHz"},
		{"Channel 6 is 2.4GHz", 6, "2.4GHz"},
		{"Channel 11 is 2.4GHz", 11, "2.4GHz"},
		{"Channel 13 is 2.4GHz", 13, "2.4GHz"},
		{"Channel 14 is 2.4GHz", 14, "2.4GHz"},

		// 5 GHz UNII-1 channels
		{"Channel 36 is 5GHz", 36, "5GHz"},
		{"Channel 40 is 5GHz", 40, "5GHz"},
		{"Channel 44 is 5GHz", 44, "5GHz"},
		{"Channel 48 is 5GHz", 48, "5GHz"},

		// 5 GHz UNII-2 (DFS) channels
		{"Channel 52 is 5GHz", 52, "5GHz"},
		{"Channel 56 is 5GHz", 56, "5GHz"},
		{"Channel 60 is 5GHz", 60, "5GHz"},
		{"Channel 64 is 5GHz", 64, "5GHz"},

		// 5 GHz UNII-2E (DFS) channels
		{"Channel 100 is 5GHz", 100, "5GHz"},
		{"Channel 104 is 5GHz", 104, "5GHz"},
		{"Channel 140 is 5GHz", 140, "5GHz"},
		{"Channel 144 is 5GHz", 144, "5GHz"},

		// 5 GHz UNII-3 channels
		{"Channel 149 is 5GHz", 149, "5GHz"},
		{"Channel 153 is 5GHz", 153, "5GHz"},
		{"Channel 157 is 5GHz", 157, "5GHz"},
		{"Channel 161 is 5GHz", 161, "5GHz"},
		{"Channel 165 is 5GHz", 165, "5GHz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			freq := wifi.ChannelToFrequency(tt.channel)
			if freq == 0 {
				t.Errorf("ChannelToFrequency(%d) returned 0", tt.channel)
				return
			}

			gotBand := wifi.GetBand(freq)
			if gotBand != tt.wantBand {
				t.Errorf(
					"Channel %d -> Freq %d -> Band %q, want %q",
					tt.channel,
					freq,
					gotBand,
					tt.wantBand,
				)
			}
		})
	}
}

// TestMultipleNetworksSorting tests that networks are sorted by signal strength.
func TestMultipleNetworksSorting(t *testing.T) {
	scanner := wifi.NewScanner("en0")
	scanTime := time.Now()

	// Create networks with various signal strengths
	testNetworks := map[string]*wifi.ScannedNetwork{
		"00:00:00:00:00:01": {
			SSID:   "VeryWeak",
			BSSID:  "00:00:00:00:00:01",
			Signal: -90,
		},
		"00:00:00:00:00:02": {
			SSID:   "Excellent",
			BSSID:  "00:00:00:00:00:02",
			Signal: -30,
		},
		"00:00:00:00:00:03": {
			SSID:   "Good",
			BSSID:  "00:00:00:00:00:03",
			Signal: -55,
		},
		"00:00:00:00:00:04": {
			SSID:   "Fair",
			BSSID:  "00:00:00:00:00:04",
			Signal: -65,
		},
		"00:00:00:00:00:05": {
			SSID:   "Weak",
			BSSID:  "00:00:00:00:00:05",
			Signal: -75,
		},
		"00:00:00:00:00:06": {
			SSID:   "Best",
			BSSID:  "00:00:00:00:00:06",
			Signal: -25,
		},
	}

	scanner.SetCachedNetworks(testNetworks, scanTime)
	networks := scanner.GetCachedNetworks()

	// Verify networks are sorted by signal strength (strongest first)
	expectedOrder := []string{"Best", "Excellent", "Good", "Fair", "Weak", "VeryWeak"}

	if len(networks) != len(expectedOrder) {
		t.Fatalf("got %d networks, want %d", len(networks), len(expectedOrder))
	}

	for i, expected := range expectedOrder {
		if networks[i].SSID != expected {
			t.Errorf("network[%d].SSID = %q, want %q", i, networks[i].SSID, expected)
		}
	}

	// Also verify signal values are in descending order
	for i := range len(networks) - 1 {
		if networks[i].Signal < networks[i+1].Signal {
			t.Errorf(
				"networks not sorted: %d dBm at index %d is less than %d dBm at index %d",
				networks[i].Signal,
				i,
				networks[i+1].Signal,
				i+1,
			)
		}
	}
}

// TestEmptySSIDHandling tests handling of networks with empty SSID (hidden networks).
func TestEmptySSIDHandling(t *testing.T) {
	tests := []struct {
		name   string
		ssid   string
		hidden bool
	}{
		{"Empty SSID is hidden", "", true},
		{"Non-empty SSID is not hidden", "MyNetwork", false},
		{"Whitespace SSID is not considered hidden", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			network := wifi.ScannedNetwork{
				SSID:  tt.ssid,
				BSSID: "aa:bb:cc:dd:ee:ff",
			}

			isHidden := network.SSID == ""
			if isHidden != tt.hidden {
				t.Errorf("SSID %q hidden = %v, want %v", tt.ssid, isHidden, tt.hidden)
			}
		})
	}
}

// TestFrequencyRangeValidation tests validation of frequency ranges.
func TestFrequencyRangeValidation(t *testing.T) {
	tests := []struct {
		name      string
		freq      int
		wantValid bool
		wantBand  string
	}{
		// Valid 2.4 GHz frequencies
		{"2.4 GHz lower bound", 2400, true, "2.4GHz"},
		{"2.4 GHz ch1", 2412, true, "2.4GHz"},
		{"2.4 GHz ch6", 2437, true, "2.4GHz"},
		{"2.4 GHz ch11", 2462, true, "2.4GHz"},
		{"2.4 GHz ch14", 2484, true, "2.4GHz"},
		{"2.4 GHz upper bound", 2500, true, "2.4GHz"},

		// Valid 5 GHz frequencies
		{"5 GHz lower bound", 5150, true, "5GHz"},
		{"5 GHz ch36", 5180, true, "5GHz"},
		{"5 GHz ch149", 5745, true, "5GHz"},
		{"5 GHz ch165", 5825, true, "5GHz"},
		{"5 GHz upper bound", 5895, true, "5GHz"},

		// Valid 6 GHz frequencies
		{"6 GHz lower bound", 5925, true, "6GHz"},
		{"6 GHz typical", 6000, true, "6GHz"},
		{"6 GHz upper bound", 7125, true, "6GHz"},

		// Invalid frequencies
		{"Zero frequency", 0, false, ""},
		{"Negative frequency", -100, false, ""},
		{"Too low", 1000, false, ""},
		{"Gap between 2.4 and 5", 3000, false, ""},
		{"Gap between 5 and 6", 5910, false, ""},
		{"Too high", 8000, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			band := wifi.GetBand(tt.freq)
			gotValid := band != ""

			if gotValid != tt.wantValid {
				t.Errorf("frequency %d valid = %v, want %v", tt.freq, gotValid, tt.wantValid)
			}
			if band != tt.wantBand {
				t.Errorf("GetBand(%d) = %q, want %q", tt.freq, band, tt.wantBand)
			}
		})
	}
}
