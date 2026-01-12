//go:build linux

package wifi_test

import (
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

// TestEndToEndNetworkFlow tests the full flow from scan to graph data.
func TestEndToEndNetworkFlow(t *testing.T) {
	scanner := wifi.NewScanner("en0")
	scanTime := time.Now()

	// Simulate scanned networks from different bands
	networks := map[string]*wifi.ScannedNetwork{
		// 2.4 GHz network
		"00:11:22:33:44:01": {
			SSID:         "HomeNetwork_2G",
			BSSID:        "00:11:22:33:44:01",
			Signal:       -55,
			Channel:      6,
			Frequency:    2437,
			Security:     "WPA2",
			ChannelWidth: 20,
			NoiseFloor:   -95,
			SNR:          40,
			HTMode:       "HT20",
			IsDFS:        false,
		},
		// 5 GHz non-DFS network
		"00:11:22:33:44:02": {
			SSID:         "HomeNetwork_5G",
			BSSID:        "00:11:22:33:44:02",
			Signal:       -50,
			Channel:      36,
			Frequency:    5180,
			Security:     "WPA3",
			ChannelWidth: 80,
			NoiseFloor:   -92,
			SNR:          42,
			HTMode:       "VHT80",
			IsDFS:        false,
		},
		// 5 GHz DFS network
		"00:11:22:33:44:03": {
			SSID:         "OfficeNetwork",
			BSSID:        "00:11:22:33:44:03",
			Signal:       -60,
			Channel:      52,
			Frequency:    5260,
			Security:     "WPA2",
			ChannelWidth: 40,
			NoiseFloor:   -93,
			SNR:          33,
			HTMode:       "HT40",
			IsDFS:        true,
		},
	}

	scanner.SetCachedNetworks(networks, scanTime)

	// Test GetCachedNetworks
	cached := scanner.GetCachedNetworks()
	if len(cached) != 3 {
		t.Fatalf("expected 3 cached networks, got %d", len(cached))
	}

	// Verify sorting by signal strength
	for i := range len(cached) - 1 {
		if cached[i].Signal < cached[i+1].Signal {
			t.Errorf("networks not sorted: %d dBm < %d dBm", cached[i].Signal, cached[i+1].Signal)
		}
	}

	// Convert to channel graph data
	graphData := wifi.GetChannelGraphData(cached, "00:11:22:33:44:02")

	// Verify band separation
	if len(graphData.Networks2_4GHz) != 1 {
		t.Errorf("expected 1 2.4 GHz network, got %d", len(graphData.Networks2_4GHz))
	}
	if len(graphData.Networks5GHz) != 2 {
		t.Errorf("expected 2 5 GHz networks, got %d", len(graphData.Networks5GHz))
	}

	// Verify connected network
	if graphData.ConnectedBSSID != "00:11:22:33:44:02" {
		t.Errorf("expected connected BSSID '00:11:22:33:44:02', got %q", graphData.ConnectedBSSID)
	}

	// Verify IsConnected flag
	foundConnected := false
	for _, net := range graphData.Networks5GHz {
		if net.BSSID == "00:11:22:33:44:02" {
			if !net.IsConnected {
				t.Error("connected network not marked as IsConnected")
			}
			foundConnected = true
		}
	}
	if !foundConnected {
		t.Error("connected network not found in 5 GHz band")
	}
}

// TestFrequencyChannelConsistency tests consistency between frequency and channel conversions.
func TestFrequencyChannelConsistency(t *testing.T) {
	// Test all common channels
	tests := []struct {
		channel int
		band    string
	}{
		// 2.4 GHz
		{1, "2.4GHz"},
		{2, "2.4GHz"},
		{3, "2.4GHz"},
		{4, "2.4GHz"},
		{5, "2.4GHz"},
		{6, "2.4GHz"},
		{7, "2.4GHz"},
		{8, "2.4GHz"},
		{9, "2.4GHz"},
		{10, "2.4GHz"},
		{11, "2.4GHz"},
		{12, "2.4GHz"},
		{13, "2.4GHz"},
		{14, "2.4GHz"},

		// 5 GHz UNII-1
		{36, "5GHz"},
		{40, "5GHz"},
		{44, "5GHz"},
		{48, "5GHz"},

		// 5 GHz UNII-2
		{52, "5GHz"},
		{56, "5GHz"},
		{60, "5GHz"},
		{64, "5GHz"},

		// 5 GHz UNII-2E
		{100, "5GHz"},
		{104, "5GHz"},
		{108, "5GHz"},
		{112, "5GHz"},
		{116, "5GHz"},
		{120, "5GHz"},
		{124, "5GHz"},
		{128, "5GHz"},
		{132, "5GHz"},
		{136, "5GHz"},
		{140, "5GHz"},
		{144, "5GHz"},

		// 5 GHz UNII-3
		{149, "5GHz"},
		{153, "5GHz"},
		{157, "5GHz"},
		{161, "5GHz"},
		{165, "5GHz"},
	}

	for _, tt := range tests {
		t.Run("Channel"+string(rune('0'+tt.channel/100))+
			string(rune('0'+(tt.channel/10)%10))+
			string(rune('0'+tt.channel%10)), func(t *testing.T) {
			// Channel -> Frequency
			freq := wifi.ChannelToFrequency(tt.channel)
			if freq == 0 {
				t.Fatalf("ChannelToFrequency(%d) = 0", tt.channel)
			}

			// Frequency -> Channel (round trip)
			gotChannel := wifi.FrequencyToChannel(freq)
			if gotChannel != tt.channel {
				t.Errorf("Round trip failed: channel %d -> freq %d -> channel %d",
					tt.channel, freq, gotChannel)
			}

			// Frequency -> Band
			band := wifi.GetBand(freq)
			if band != tt.band {
				t.Errorf("GetBand(%d) = %q, want %q", freq, band, tt.band)
			}
		})
	}
}

// TestConcurrentManagerAndScanner tests concurrent access to Manager and Scanner.
func TestConcurrentManagerAndScanner(t *testing.T) {
	manager := wifi.NewManager("en0")
	scanner := wifi.NewScanner("en0")

	var wg sync.WaitGroup
	const numGoroutines = 20
	const numIterations = 100

	// Set up initial scanner data
	scanTime := time.Now()
	networks := map[string]*wifi.ScannedNetwork{
		"00:11:22:33:44:55": {
			SSID:   "TestNet",
			BSSID:  "00:11:22:33:44:55",
			Signal: -50,
		},
	}
	scanner.SetCachedNetworks(networks, scanTime)

	// Run concurrent operations on both
	for i := range numGoroutines {
		wg.Add(2)

		// Manager operations
		go func(id int) {
			defer wg.Done()
			for range numIterations {
				manager.SetInterface("en" + string(rune('0'+id%10)))
				_ = manager.InterfaceName()
				_ = manager.IsWireless()
			}
		}(i)

		// Scanner operations
		go func(id int) {
			defer wg.Done()
			for range numIterations {
				scanner.SetInterface("wlan" + string(rune('0'+id%10)))
				_ = scanner.ScannerInterfaceName()
				_ = scanner.GetCachedNetworks()
				_ = scanner.GetLastScanTime()
			}
		}(i)
	}

	wg.Wait()
}

// TestChannelGraphDataWithMixedSignals tests graph data with varying signal strengths.
func TestChannelGraphDataWithMixedSignals(t *testing.T) {
	now := time.Now()

	// Create networks with varying signal strengths across bands
	networks := []*wifi.ScannedNetwork{
		// 2.4 GHz with different signals
		{
			SSID:      "Strong2G",
			BSSID:     "00:00:00:00:00:01",
			Signal:    -30,
			Channel:   1,
			Frequency: 2412,
			LastSeen:  now,
		},
		{
			SSID:      "Medium2G",
			BSSID:     "00:00:00:00:00:02",
			Signal:    -55,
			Channel:   6,
			Frequency: 2437,
			LastSeen:  now,
		},
		{
			SSID:      "Weak2G",
			BSSID:     "00:00:00:00:00:03",
			Signal:    -80,
			Channel:   11,
			Frequency: 2462,
			LastSeen:  now,
		},
		// 5 GHz with different signals
		{
			SSID:      "Strong5G",
			BSSID:     "00:00:00:00:00:04",
			Signal:    -35,
			Channel:   36,
			Frequency: 5180,
			LastSeen:  now,
		},
		{
			SSID:      "Medium5G",
			BSSID:     "00:00:00:00:00:05",
			Signal:    -50,
			Channel:   149,
			Frequency: 5745,
			LastSeen:  now,
		},
		{
			SSID:      "Weak5G",
			BSSID:     "00:00:00:00:00:06",
			Signal:    -75,
			Channel:   52,
			Frequency: 5260,
			LastSeen:  now,
		},
	}

	data := wifi.GetChannelGraphData(networks, "00:00:00:00:00:04")

	// Verify counts
	if len(data.Networks2_4GHz) != 3 {
		t.Errorf("expected 3 2.4 GHz networks, got %d", len(data.Networks2_4GHz))
	}
	if len(data.Networks5GHz) != 3 {
		t.Errorf("expected 3 5 GHz networks, got %d", len(data.Networks5GHz))
	}

	// Verify connected network
	for _, net := range data.Networks5GHz {
		if net.BSSID == "00:00:00:00:00:04" {
			if !net.IsConnected {
				t.Error("expected Strong5G to be marked as connected")
			}
		} else if net.IsConnected {
			t.Errorf("unexpected network %q marked as connected", net.SSID)
		}
	}
}

// TestAllBandsChannelGraphData tests channel graph with networks in all three bands.
func TestAllBandsChannelGraphData(t *testing.T) {
	now := time.Now()

	networks := []*wifi.ScannedNetwork{
		// 2.4 GHz
		{SSID: "Net24", BSSID: "00:00:00:00:00:01", Channel: 6, Frequency: 2437, Signal: -50, LastSeen: now},
		// 5 GHz
		{SSID: "Net5", BSSID: "00:00:00:00:00:02", Channel: 36, Frequency: 5180, Signal: -45, LastSeen: now},
		// 6 GHz
		{SSID: "Net6", BSSID: "00:00:00:00:00:03", Channel: 1, Frequency: 5955, Signal: -40, LastSeen: now},
	}

	data := wifi.GetChannelGraphData(networks, "")

	if len(data.Networks2_4GHz) != 1 {
		t.Errorf("expected 1 2.4 GHz network, got %d", len(data.Networks2_4GHz))
	}
	if len(data.Networks5GHz) != 1 {
		t.Errorf("expected 1 5 GHz network, got %d", len(data.Networks5GHz))
	}
	if len(data.Networks6GHz) != 1 {
		t.Errorf("expected 1 6 GHz network, got %d", len(data.Networks6GHz))
	}

	// Verify band assignments
	if len(data.Networks2_4GHz) > 0 && data.Networks2_4GHz[0].Band != "2.4GHz" {
		t.Errorf("2.4 GHz network has wrong band: %q", data.Networks2_4GHz[0].Band)
	}
	if len(data.Networks5GHz) > 0 && data.Networks5GHz[0].Band != "5GHz" {
		t.Errorf("5 GHz network has wrong band: %q", data.Networks5GHz[0].Band)
	}
	if len(data.Networks6GHz) > 0 && data.Networks6GHz[0].Band != "6GHz" {
		t.Errorf("6 GHz network has wrong band: %q", data.Networks6GHz[0].Band)
	}
}

// TestManagerIsWirelessWithDifferentInterfaces tests wireless detection for various interfaces.
func TestManagerIsWirelessWithDifferentInterfaces(t *testing.T) {
	tests := []struct {
		name  string
		iface string
	}{
		{"macOS primary WiFi", "en0"},
		{"macOS secondary WiFi", "en1"},
		{"Linux wlan0", "wlan0"},
		{"Linux wlan1", "wlan1"},
		{"Linux predictable", "wlp2s0"},
		{"Loopback", "lo0"},
		{"Ethernet", "eth0"},
		{"Empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := wifi.NewManager(tt.iface)

			// Just verify it doesn't panic
			_ = manager.IsWireless()
		})
	}
}

// TestManagerGetInfoWithDifferentInterfaces tests info retrieval for various interfaces.
func TestManagerGetInfoWithDifferentInterfaces(t *testing.T) {
	tests := []struct {
		name  string
		iface string
	}{
		{"macOS primary WiFi", "en0"},
		{"macOS secondary WiFi", "en1"},
		{"Linux wlan0", "wlan0"},
		{"Empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := wifi.NewManager(tt.iface)

			// Just verify it doesn't panic
			// Result depends on actual system state
			_ = manager.GetInfo()
		})
	}
}

// TestInfoStructFieldAccess tests access to all Info struct fields.
func TestInfoStructFieldAccess(t *testing.T) {
	info := wifi.Info{
		SSID:      "TestNetwork",
		BSSID:     "00:11:22:33:44:55",
		Signal:    -55,
		Channel:   6,
		Frequency: 2437,
		Security:  "WPA3",
	}

	// Test all fields are accessible and have expected values
	if info.SSID != "TestNetwork" {
		t.Errorf("SSID = %q, want 'TestNetwork'", info.SSID)
	}
	if info.BSSID != "00:11:22:33:44:55" {
		t.Errorf("BSSID = %q, want '00:11:22:33:44:55'", info.BSSID)
	}
	if info.Signal != -55 {
		t.Errorf("Signal = %d, want -55", info.Signal)
	}
	if info.Channel != 6 {
		t.Errorf("Channel = %d, want 6", info.Channel)
	}
	if info.Frequency != 2437 {
		t.Errorf("Frequency = %d, want 2437", info.Frequency)
	}
	if info.Security != "WPA3" {
		t.Errorf("Security = %q, want 'WPA3'", info.Security)
	}
}

// TestScannedNetworkStructFieldAccess tests access to all ScannedNetwork struct fields.
func TestScannedNetworkStructFieldAccess(t *testing.T) {
	now := time.Now()

	network := wifi.ScannedNetwork{
		SSID:         "TestNetwork",
		BSSID:        "00:11:22:33:44:55",
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

	// Test all fields
	if network.SSID != "TestNetwork" {
		t.Errorf("SSID = %q, want 'TestNetwork'", network.SSID)
	}
	if network.BSSID != "00:11:22:33:44:55" {
		t.Errorf("BSSID = %q, want '00:11:22:33:44:55'", network.BSSID)
	}
	if network.Signal != -55 {
		t.Errorf("Signal = %d, want -55", network.Signal)
	}
	if network.Channel != 36 {
		t.Errorf("Channel = %d, want 36", network.Channel)
	}
	if network.Frequency != 5180 {
		t.Errorf("Frequency = %d, want 5180", network.Frequency)
	}
	if network.Security != "WPA3" {
		t.Errorf("Security = %q, want 'WPA3'", network.Security)
	}
	if network.ChannelWidth != 80 {
		t.Errorf("ChannelWidth = %d, want 80", network.ChannelWidth)
	}
	if network.NoiseFloor != -95 {
		t.Errorf("NoiseFloor = %d, want -95", network.NoiseFloor)
	}
	if network.SNR != 40 {
		t.Errorf("SNR = %d, want 40", network.SNR)
	}
	if network.HTMode != "VHT80" {
		t.Errorf("HTMode = %q, want 'VHT80'", network.HTMode)
	}
	if network.IsDFS {
		t.Error("IsDFS = true, want false")
	}
	if !network.LastSeen.Equal(now) {
		t.Errorf("LastSeen = %v, want %v", network.LastSeen, now)
	}
}

// TestChannelNetworkStructFieldAccess tests access to all ChannelNetwork struct fields.
func TestChannelNetworkStructFieldAccess(t *testing.T) {
	cn := wifi.ChannelNetwork{
		SSID:         "TestNetwork",
		BSSID:        "00:11:22:33:44:55",
		Channel:      36,
		CenterFreq:   5180,
		ChannelWidth: 80,
		Signal:       -55,
		Band:         "5GHz",
		IsConnected:  true,
	}

	if cn.SSID != "TestNetwork" {
		t.Errorf("SSID = %q, want 'TestNetwork'", cn.SSID)
	}
	if cn.BSSID != "00:11:22:33:44:55" {
		t.Errorf("BSSID = %q, want '00:11:22:33:44:55'", cn.BSSID)
	}
	if cn.Channel != 36 {
		t.Errorf("Channel = %d, want 36", cn.Channel)
	}
	if cn.CenterFreq != 5180 {
		t.Errorf("CenterFreq = %d, want 5180", cn.CenterFreq)
	}
	if cn.ChannelWidth != 80 {
		t.Errorf("ChannelWidth = %d, want 80", cn.ChannelWidth)
	}
	if cn.Signal != -55 {
		t.Errorf("Signal = %d, want -55", cn.Signal)
	}
	if cn.Band != "5GHz" {
		t.Errorf("Band = %q, want '5GHz'", cn.Band)
	}
	if !cn.IsConnected {
		t.Error("IsConnected = false, want true")
	}
}

// TestChannelGraphDataStructFieldAccess tests access to all ChannelGraphData struct fields.
func TestChannelGraphDataStructFieldAccess(t *testing.T) {
	now := time.Now()

	data := wifi.ChannelGraphData{
		Networks2_4GHz: []wifi.ChannelNetwork{{SSID: "Net24"}},
		Networks5GHz:   []wifi.ChannelNetwork{{SSID: "Net5"}},
		Networks6GHz:   []wifi.ChannelNetwork{{SSID: "Net6"}},
		ConnectedBSSID: "00:11:22:33:44:55",
		ScanTime:       now,
	}

	if len(data.Networks2_4GHz) != 1 {
		t.Errorf("Networks2_4GHz length = %d, want 1", len(data.Networks2_4GHz))
	}
	if len(data.Networks5GHz) != 1 {
		t.Errorf("Networks5GHz length = %d, want 1", len(data.Networks5GHz))
	}
	if len(data.Networks6GHz) != 1 {
		t.Errorf("Networks6GHz length = %d, want 1", len(data.Networks6GHz))
	}
	if data.ConnectedBSSID != "00:11:22:33:44:55" {
		t.Errorf("ConnectedBSSID = %q, want '00:11:22:33:44:55'", data.ConnectedBSSID)
	}
	if !data.ScanTime.Equal(now) {
		t.Errorf("ScanTime = %v, want %v", data.ScanTime, now)
	}
}
