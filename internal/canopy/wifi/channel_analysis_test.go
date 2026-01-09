//go:build linux

package wifi_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

// TestChannel24GHzComplete tests all 2.4 GHz channels (1-14).
func TestChannel24GHzComplete(t *testing.T) {
	tests := []struct {
		channel int
		freq    int
	}{
		{1, 2412},
		{2, 2417},
		{3, 2422},
		{4, 2427},
		{5, 2432},
		{6, 2437},
		{7, 2442},
		{8, 2447},
		{9, 2452},
		{10, 2457},
		{11, 2462},
		{12, 2467},
		{13, 2472},
		{14, 2484}, // Japan only, 22 MHz spacing
	}

	for _, tt := range tests {
		t.Run("Channel"+string(rune('0'+tt.channel/10))+string(rune('0'+tt.channel%10)), func(t *testing.T) {
			gotFreq := wifi.ChannelToFrequency(tt.channel)
			if gotFreq != tt.freq {
				t.Errorf("ChannelToFrequency(%d) = %d, want %d", tt.channel, gotFreq, tt.freq)
			}

			gotChan := wifi.FrequencyToChannel(tt.freq)
			if gotChan != tt.channel {
				t.Errorf("FrequencyToChannel(%d) = %d, want %d", tt.freq, gotChan, tt.channel)
			}

			band := wifi.GetBand(tt.freq)
			if band != "2.4GHz" {
				t.Errorf("GetBand(%d) = %q, want '2.4GHz'", tt.freq, band)
			}
		})
	}
}

// TestChannel5GHzUNII1 tests 5 GHz UNII-1 channels (36-48).
func TestChannel5GHzUNII1(t *testing.T) {
	tests := []struct {
		channel int
		freq    int
		isDFS   bool
	}{
		{36, 5180, false},
		{40, 5200, false},
		{44, 5220, false},
		{48, 5240, false},
	}

	for _, tt := range tests {
		t.Run("Channel"+string(rune('0'+tt.channel/10))+string(rune('0'+tt.channel%10)), func(t *testing.T) {
			gotFreq := wifi.ChannelToFrequency(tt.channel)
			if gotFreq != tt.freq {
				t.Errorf("ChannelToFrequency(%d) = %d, want %d", tt.channel, gotFreq, tt.freq)
			}

			gotChan := wifi.FrequencyToChannel(tt.freq)
			if gotChan != tt.channel {
				t.Errorf("FrequencyToChannel(%d) = %d, want %d", tt.freq, gotChan, tt.channel)
			}

			band := wifi.GetBand(tt.freq)
			if band != "5GHz" {
				t.Errorf("GetBand(%d) = %q, want '5GHz'", tt.freq, band)
			}
		})
	}
}

// TestChannel5GHzUNII2 tests 5 GHz UNII-2 DFS channels (52-64).
func TestChannel5GHzUNII2(t *testing.T) {
	tests := []struct {
		channel int
		freq    int
	}{
		{52, 5260},
		{56, 5280},
		{60, 5300},
		{64, 5320},
	}

	for _, tt := range tests {
		t.Run("DFS_Channel"+string(rune('0'+tt.channel/10))+string(rune('0'+tt.channel%10)), func(t *testing.T) {
			gotFreq := wifi.ChannelToFrequency(tt.channel)
			if gotFreq != tt.freq {
				t.Errorf("ChannelToFrequency(%d) = %d, want %d", tt.channel, gotFreq, tt.freq)
			}

			gotChan := wifi.FrequencyToChannel(tt.freq)
			if gotChan != tt.channel {
				t.Errorf("FrequencyToChannel(%d) = %d, want %d", tt.freq, gotChan, tt.channel)
			}

			band := wifi.GetBand(tt.freq)
			if band != "5GHz" {
				t.Errorf("GetBand(%d) = %q, want '5GHz'", tt.freq, band)
			}
		})
	}
}

// TestChannel5GHzUNII2Extended tests 5 GHz UNII-2 Extended DFS channels (100-144).
func TestChannel5GHzUNII2Extended(t *testing.T) {
	tests := []struct {
		channel int
		freq    int
	}{
		{100, 5500},
		{104, 5520},
		{108, 5540},
		{112, 5560},
		{116, 5580},
		{120, 5600},
		{124, 5620},
		{128, 5640},
		{132, 5660},
		{136, 5680},
		{140, 5700},
		{144, 5720},
	}

	for _, tt := range tests {
		t.Run("DFS_Channel"+string(rune('0'+tt.channel/100))+
			string(rune('0'+(tt.channel/10)%10))+
			string(rune('0'+tt.channel%10)), func(t *testing.T) {
			gotFreq := wifi.ChannelToFrequency(tt.channel)
			if gotFreq != tt.freq {
				t.Errorf("ChannelToFrequency(%d) = %d, want %d", tt.channel, gotFreq, tt.freq)
			}

			gotChan := wifi.FrequencyToChannel(tt.freq)
			if gotChan != tt.channel {
				t.Errorf("FrequencyToChannel(%d) = %d, want %d", tt.freq, gotChan, tt.channel)
			}
		})
	}
}

// TestChannel5GHzUNII3 tests 5 GHz UNII-3 channels (149-165).
func TestChannel5GHzUNII3(t *testing.T) {
	tests := []struct {
		channel int
		freq    int
	}{
		{149, 5745},
		{153, 5765},
		{157, 5785},
		{161, 5805},
		{165, 5825},
	}

	for _, tt := range tests {
		t.Run("Channel"+string(rune('0'+tt.channel/100))+
			string(rune('0'+(tt.channel/10)%10))+
			string(rune('0'+tt.channel%10)), func(t *testing.T) {
			gotFreq := wifi.ChannelToFrequency(tt.channel)
			if gotFreq != tt.freq {
				t.Errorf("ChannelToFrequency(%d) = %d, want %d", tt.channel, gotFreq, tt.freq)
			}

			gotChan := wifi.FrequencyToChannel(tt.freq)
			if gotChan != tt.channel {
				t.Errorf("FrequencyToChannel(%d) = %d, want %d", tt.freq, gotChan, tt.channel)
			}
		})
	}
}

// TestChannelInvalidValues tests handling of invalid channel values.
func TestChannelInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		channel int
		want    int
	}{
		{"Negative channel", -1, 0},
		{"Zero channel", 0, 0},
		{"Channel 234 (out of range)", 234, 0},
		{"Channel 250 (out of range)", 250, 0},
		{"Channel 500 (out of range)", 500, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wifi.ChannelToFrequency(tt.channel)
			if got != tt.want {
				t.Errorf("ChannelToFrequency(%d) = %d, want %d", tt.channel, got, tt.want)
			}
		})
	}
}

// TestFrequencyInvalidValues tests handling of invalid frequency values.
func TestFrequencyInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		freq int
		want int
	}{
		{"Negative frequency", -1, 0},
		{"Zero frequency", 0, 0},
		{"Very low frequency", 100, 0},
		{"Between 2.4 and 5 GHz", 3000, 0},
		{"Between 5 and 6 GHz", 5910, 0},
		{"Very high frequency", 10000, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wifi.FrequencyToChannel(tt.freq)
			if got != tt.want {
				t.Errorf("FrequencyToChannel(%d) = %d, want %d", tt.freq, got, tt.want)
			}
		})
	}
}

// TestChannelWidthDetectionAllModes tests all supported HT modes.
func TestChannelWidthDetectionAllModes(t *testing.T) {
	tests := []struct {
		htMode string
		want   int
	}{
		// 802.11n modes
		{"HT20", 20},
		{"HT40", 40},
		{"HT40+", 40},
		{"HT40-", 40},

		// 802.11ac modes
		{"VHT20", 20},
		{"VHT40", 40},
		{"VHT80", 80},
		{"VHT160", 160},

		// 802.11ax (Wi-Fi 6) modes
		{"HE20", 20},
		{"HE40", 40},
		{"HE80", 80},
		{"HE160", 160},

		// 802.11be (Wi-Fi 7) modes
		{"EHT20", 20},
		{"EHT40", 40},
		{"EHT80", 80},
		{"EHT160", 160},
		{"EHT320", 320},
	}

	for _, tt := range tests {
		t.Run(tt.htMode, func(t *testing.T) {
			// Test with 5 GHz frequency (most modes are used there)
			got := wifi.DetectChannelWidth(5180, tt.htMode)
			if got != tt.want {
				t.Errorf("DetectChannelWidth(5180, %q) = %d, want %d", tt.htMode, got, tt.want)
			}
		})
	}
}

// TestChannelWidthFallbackByBand tests fallback width detection by band.
func TestChannelWidthFallbackByBand(t *testing.T) {
	tests := []struct {
		name string
		freq int
		want int
	}{
		// 2.4 GHz defaults to 20 MHz
		{"2.4 GHz ch1", 2412, 20},
		{"2.4 GHz ch6", 2437, 20},
		{"2.4 GHz ch11", 2462, 20},

		// 5 GHz defaults to 80 MHz
		{"5 GHz ch36", 5180, 80},
		{"5 GHz ch149", 5745, 80},

		// 6 GHz defaults to 160 MHz
		{"6 GHz start", 5955, 160},
		{"6 GHz mid", 6500, 160},

		// Unknown defaults to 20 MHz
		{"Unknown band", 1000, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Empty htMode triggers fallback
			got := wifi.DetectChannelWidth(tt.freq, "")
			if got != tt.want {
				t.Errorf("DetectChannelWidth(%d, '') = %d, want %d", tt.freq, got, tt.want)
			}
		})
	}
}

// TestChannelGraphDataMultipleBands tests channel graph with networks across all bands.
func TestChannelGraphDataMultipleBands(t *testing.T) {
	now := time.Now()

	networks := []*wifi.ScannedNetwork{
		// 2.4 GHz networks
		{SSID: "Net24_1", BSSID: "00:00:00:00:00:01", Channel: 1, Frequency: 2412, Signal: -50, LastSeen: now},
		{SSID: "Net24_2", BSSID: "00:00:00:00:00:02", Channel: 6, Frequency: 2437, Signal: -55, LastSeen: now},
		{SSID: "Net24_3", BSSID: "00:00:00:00:00:03", Channel: 11, Frequency: 2462, Signal: -60, LastSeen: now},

		// 5 GHz networks
		{SSID: "Net5_1", BSSID: "00:00:00:00:00:04", Channel: 36, Frequency: 5180, Signal: -45, LastSeen: now},
		{SSID: "Net5_2", BSSID: "00:00:00:00:00:05", Channel: 149, Frequency: 5745, Signal: -50, LastSeen: now},

		// 6 GHz networks
		{SSID: "Net6_1", BSSID: "00:00:00:00:00:06", Channel: 1, Frequency: 5955, Signal: -40, LastSeen: now},
	}

	data := wifi.GetChannelGraphData(networks, "00:00:00:00:00:04")

	// Verify band counts
	if len(data.Networks2_4GHz) != 3 {
		t.Errorf("2.4 GHz network count = %d, want 3", len(data.Networks2_4GHz))
	}
	if len(data.Networks5GHz) != 2 {
		t.Errorf("5 GHz network count = %d, want 2", len(data.Networks5GHz))
	}
	if len(data.Networks6GHz) != 1 {
		t.Errorf("6 GHz network count = %d, want 1", len(data.Networks6GHz))
	}

	// Verify connected BSSID
	if data.ConnectedBSSID != "00:00:00:00:00:04" {
		t.Errorf("ConnectedBSSID = %q, want '00:00:00:00:00:04'", data.ConnectedBSSID)
	}

	// Verify connected network is marked
	foundConnected := false
	for _, net := range data.Networks5GHz {
		if net.BSSID == "00:00:00:00:00:04" {
			if !net.IsConnected {
				t.Error("Connected network not marked as IsConnected")
			}
			foundConnected = true
		} else if net.IsConnected {
			t.Errorf("Non-connected network %q marked as IsConnected", net.BSSID)
		}
	}
	if !foundConnected {
		t.Error("Connected network not found in 5 GHz band")
	}
}

// TestChannelGraphDataFieldPreservation tests that all fields are preserved.
func TestChannelGraphDataFieldPreservation(t *testing.T) {
	now := time.Now()

	network := &wifi.ScannedNetwork{
		SSID:         "TestNetwork",
		BSSID:        "AA:BB:CC:DD:EE:FF",
		Signal:       -55,
		Channel:      36,
		Frequency:    5180,
		Security:     "WPA3",
		ChannelWidth: 80,
		NoiseFloor:   -95,
		SNR:          40,
		HTMode:       "VHT80",
		IsDFS:        false,
		LastSeen:     now,
	}

	data := wifi.GetChannelGraphData([]*wifi.ScannedNetwork{network}, "")

	if len(data.Networks5GHz) != 1 {
		t.Fatalf("expected 1 network in 5 GHz band")
	}

	cn := data.Networks5GHz[0]

	// Verify field mapping
	if cn.SSID != network.SSID {
		t.Errorf("SSID = %q, want %q", cn.SSID, network.SSID)
	}
	if cn.BSSID != network.BSSID {
		t.Errorf("BSSID = %q, want %q", cn.BSSID, network.BSSID)
	}
	if cn.Signal != network.Signal {
		t.Errorf("Signal = %d, want %d", cn.Signal, network.Signal)
	}
	if cn.Channel != network.Channel {
		t.Errorf("Channel = %d, want %d", cn.Channel, network.Channel)
	}
	if cn.CenterFreq != network.Frequency {
		t.Errorf("CenterFreq = %d, want %d", cn.CenterFreq, network.Frequency)
	}
	if cn.ChannelWidth != network.ChannelWidth {
		t.Errorf("ChannelWidth = %d, want %d", cn.ChannelWidth, network.ChannelWidth)
	}
	if cn.Band != "5GHz" {
		t.Errorf("Band = %q, want '5GHz'", cn.Band)
	}
}

// TestChannelOverlapScenarios tests common channel overlap scenarios.
func TestChannelOverlapScenarios(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		networks  []*wifi.ScannedNetwork
		wantCount int
	}{
		{
			name: "Overlapping 2.4 GHz channels 1,6,11",
			networks: []*wifi.ScannedNetwork{
				{SSID: "Net1", BSSID: "00:00:00:00:00:01", Channel: 1, Frequency: 2412, LastSeen: now},
				{SSID: "Net6", BSSID: "00:00:00:00:00:02", Channel: 6, Frequency: 2437, LastSeen: now},
				{SSID: "Net11", BSSID: "00:00:00:00:00:03", Channel: 11, Frequency: 2462, LastSeen: now},
			},
			wantCount: 3,
		},
		{
			name: "Same channel multiple networks",
			networks: []*wifi.ScannedNetwork{
				{SSID: "Net1_Ch6", BSSID: "00:00:00:00:00:01", Channel: 6, Frequency: 2437, LastSeen: now},
				{SSID: "Net2_Ch6", BSSID: "00:00:00:00:00:02", Channel: 6, Frequency: 2437, LastSeen: now},
				{SSID: "Net3_Ch6", BSSID: "00:00:00:00:00:03", Channel: 6, Frequency: 2437, LastSeen: now},
			},
			wantCount: 3,
		},
		{
			name: "Adjacent channels",
			networks: []*wifi.ScannedNetwork{
				{SSID: "Net1", BSSID: "00:00:00:00:00:01", Channel: 5, Frequency: 2432, LastSeen: now},
				{SSID: "Net2", BSSID: "00:00:00:00:00:02", Channel: 6, Frequency: 2437, LastSeen: now},
				{SSID: "Net3", BSSID: "00:00:00:00:00:03", Channel: 7, Frequency: 2442, LastSeen: now},
			},
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := wifi.GetChannelGraphData(tt.networks, "")
			if len(data.Networks2_4GHz) != tt.wantCount {
				t.Errorf("network count = %d, want %d", len(data.Networks2_4GHz), tt.wantCount)
			}
		})
	}
}

// TestChannel6GHzRange tests 6 GHz band channel coverage.
func TestChannel6GHzRange(t *testing.T) {
	// Sample 6 GHz channels
	tests := []struct {
		channel int
		freq    int
	}{
		{1, 5955},
		{5, 5975},
		{9, 5995},
		{13, 6015},
		{17, 6035},
		{21, 6055},
		{25, 6075},
		{29, 6095},
		{33, 6115},
		{37, 6135},
		{41, 6155},
		{45, 6175},
		{49, 6195},
		{53, 6215},
		{57, 6235},
		{61, 6255},
		{65, 6275},
		{93, 6415},
		{117, 6535},
		{149, 6695},
		{181, 6855},
		{213, 7015},
		{229, 7095},
		{233, 7115},
	}

	for _, tt := range tests {
		t.Run("Channel6GHz_"+string(rune('0'+tt.channel/100))+
			string(rune('0'+(tt.channel/10)%10))+
			string(rune('0'+tt.channel%10)), func(t *testing.T) {
			gotChan := wifi.FrequencyToChannel(tt.freq)
			if gotChan != tt.channel {
				t.Errorf("FrequencyToChannel(%d) = %d, want %d", tt.freq, gotChan, tt.channel)
			}

			band := wifi.GetBand(tt.freq)
			if band != "6GHz" {
				t.Errorf("GetBand(%d) = %q, want '6GHz'", tt.freq, band)
			}
		})
	}
}

// TestChannelGraphDataNoConnected tests with no connected network.
func TestChannelGraphDataNoConnected(t *testing.T) {
	now := time.Now()

	networks := []*wifi.ScannedNetwork{
		{SSID: "Net1", BSSID: "00:00:00:00:00:01", Channel: 6, Frequency: 2437, LastSeen: now},
	}

	data := wifi.GetChannelGraphData(networks, "")

	if data.ConnectedBSSID != "" {
		t.Errorf("ConnectedBSSID = %q, want empty", data.ConnectedBSSID)
	}

	for _, net := range data.Networks2_4GHz {
		if net.IsConnected {
			t.Errorf("Network %q incorrectly marked as connected", net.SSID)
		}
	}
}

// TestChannelGraphDataNonExistentConnected tests with connected BSSID not in network list.
func TestChannelGraphDataNonExistentConnected(t *testing.T) {
	now := time.Now()

	networks := []*wifi.ScannedNetwork{
		{SSID: "Net1", BSSID: "00:00:00:00:00:01", Channel: 6, Frequency: 2437, LastSeen: now},
	}

	// Provide a BSSID that doesn't exist in the network list
	data := wifi.GetChannelGraphData(networks, "FF:FF:FF:FF:FF:FF")

	if data.ConnectedBSSID != "FF:FF:FF:FF:FF:FF" {
		t.Errorf("ConnectedBSSID = %q, want 'FF:FF:FF:FF:FF:FF'", data.ConnectedBSSID)
	}

	// No network should be marked as connected
	for _, net := range data.Networks2_4GHz {
		if net.IsConnected {
			t.Errorf("Network %q incorrectly marked as connected", net.SSID)
		}
	}
}
