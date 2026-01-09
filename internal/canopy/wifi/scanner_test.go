//go:build linux

package wifi_test

import (
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

func TestNewScanner(t *testing.T) {
	tests := []struct {
		name      string
		iface     string
		wantIface string
	}{
		{
			name:      "standard interface",
			iface:     "en0",
			wantIface: "en0",
		},
		{
			name:      "linux wlan interface",
			iface:     "wlan0",
			wantIface: "wlan0",
		},
		{
			name:      "empty interface",
			iface:     "",
			wantIface: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := wifi.NewScanner(tt.iface)
			if scanner == nil {
				t.Fatal("expected non-nil scanner")
			}

			if got := scanner.ScannerInterfaceName(); got != tt.wantIface {
				t.Errorf("ScannerInterfaceName() = %q, want %q", got, tt.wantIface)
			}
		})
	}
}

func TestScannerSetInterface(t *testing.T) {
	scanner := wifi.NewScanner("en0")

	tests := []struct {
		name      string
		iface     string
		wantIface string
	}{
		{"change to wlan0", "wlan0", "wlan0"},
		{"change to eth0", "eth0", "eth0"},
		{"change to empty", "", ""},
		{"change back to en0", "en0", "en0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner.SetInterface(tt.iface)
			if got := scanner.ScannerInterfaceName(); got != tt.wantIface {
				t.Errorf("after SetInterface(%q), ScannerInterfaceName() = %q, want %q",
					tt.iface, got, tt.wantIface)
			}
		})
	}
}

func TestScannerGetCachedNetworks(t *testing.T) {
	scanner := wifi.NewScanner("en0")
	scanTime := time.Now()

	// Set up test networks
	testNetworks := map[string]*wifi.ScannedNetwork{
		"00:11:22:33:44:55": {
			SSID:      "Network1",
			BSSID:     "00:11:22:33:44:55",
			Signal:    -50,
			Channel:   6,
			Frequency: 2437,
		},
		"AA:BB:CC:DD:EE:FF": {
			SSID:      "Network2",
			BSSID:     "AA:BB:CC:DD:EE:FF",
			Signal:    -65,
			Channel:   36,
			Frequency: 5180,
		},
		"11:22:33:44:55:66": {
			SSID:      "Network3",
			BSSID:     "11:22:33:44:55:66",
			Signal:    -45,
			Channel:   1,
			Frequency: 2412,
		},
	}

	scanner.SetCachedNetworks(testNetworks, scanTime)

	// Get cached networks
	networks := scanner.GetCachedNetworks()

	// Verify count
	if len(networks) != 3 {
		t.Errorf("GetCachedNetworks() returned %d networks, want 3", len(networks))
	}

	// Verify sorting by signal strength (strongest first)
	for i := range len(networks) - 1 {
		if networks[i].Signal < networks[i+1].Signal {
			t.Errorf("networks not sorted by signal: %d dBm before %d dBm",
				networks[i].Signal, networks[i+1].Signal)
		}
	}

	// Verify first network is strongest signal
	if len(networks) > 0 && networks[0].Signal != -45 {
		t.Errorf("first network signal = %d, want -45 (strongest)", networks[0].Signal)
	}
}

func TestScannerGetCachedNetworksEmpty(t *testing.T) {
	scanner := wifi.NewScanner("en0")

	networks := scanner.GetCachedNetworks()
	if len(networks) != 0 {
		t.Errorf("GetCachedNetworks() on new scanner returned %d networks, want 0", len(networks))
	}
}

func TestScannerGetLastScanTime(t *testing.T) {
	scanner := wifi.NewScanner("en0")

	// Initially should be zero time
	if !scanner.GetLastScanTime().IsZero() {
		t.Error("GetLastScanTime() on new scanner should return zero time")
	}

	// Set a scan time
	expectedTime := time.Now()
	scanner.SetCachedNetworks(map[string]*wifi.ScannedNetwork{}, expectedTime)

	lastScan := scanner.GetLastScanTime()
	if !lastScan.Equal(expectedTime) {
		t.Errorf("GetLastScanTime() = %v, want %v", lastScan, expectedTime)
	}
}

func TestScannerConcurrentAccess(_ *testing.T) {
	scanner := wifi.NewScanner("en0")
	scanTime := time.Now()

	// Prepare test data
	testNetworks := map[string]*wifi.ScannedNetwork{
		"00:11:22:33:44:55": {
			SSID:      "ConcurrentNet",
			BSSID:     "00:11:22:33:44:55",
			Signal:    -55,
			Channel:   11,
			Frequency: 2462,
		},
	}
	scanner.SetCachedNetworks(testNetworks, scanTime)

	var wg sync.WaitGroup
	const numGoroutines = 10
	const numIterations = 50

	// Run concurrent operations
	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for range numIterations {
				// Mix of read and write operations
				_ = scanner.ScannerInterfaceName()
				_ = scanner.GetCachedNetworks()
				_ = scanner.GetLastScanTime()
				if id%2 == 0 {
					scanner.SetInterface("en" + string(rune('0'+id%10)))
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestScannedNetworkFields(t *testing.T) {
	now := time.Now()
	network := wifi.ScannedNetwork{
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

	// Verify all fields are correctly set
	if network.SSID != "TestNetwork" {
		t.Errorf("SSID = %q, want %q", network.SSID, "TestNetwork")
	}
	if network.BSSID != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("BSSID = %q, want %q", network.BSSID, "AA:BB:CC:DD:EE:FF")
	}
	if network.Signal != -55 {
		t.Errorf("Signal = %d, want %d", network.Signal, -55)
	}
	if network.Channel != 36 {
		t.Errorf("Channel = %d, want %d", network.Channel, 36)
	}
	if network.Frequency != 5180 {
		t.Errorf("Frequency = %d, want %d", network.Frequency, 5180)
	}
	if network.Security != "WPA3" {
		t.Errorf("Security = %q, want %q", network.Security, "WPA3")
	}
	if network.ChannelWidth != 80 {
		t.Errorf("ChannelWidth = %d, want %d", network.ChannelWidth, 80)
	}
	if network.NoiseFloor != -95 {
		t.Errorf("NoiseFloor = %d, want %d", network.NoiseFloor, -95)
	}
	if network.SNR != 40 {
		t.Errorf("SNR = %d, want %d", network.SNR, 40)
	}
	if network.HTMode != "VHT80" {
		t.Errorf("HTMode = %q, want %q", network.HTMode, "VHT80")
	}
	if network.IsDFS {
		t.Error("IsDFS = true, want false")
	}
	if !network.LastSeen.Equal(now) {
		t.Errorf("LastSeen = %v, want %v", network.LastSeen, now)
	}
}

func TestScannedNetworkDFSFlag(t *testing.T) {
	// Test network on a DFS channel - verify the IsDFS flag is properly stored
	network := wifi.ScannedNetwork{
		IsDFS: true,
	}

	if !network.IsDFS {
		t.Error("expected IsDFS to be true when set to true")
	}

	// Also test when it's false
	nonDFSNetwork := wifi.ScannedNetwork{
		IsDFS: false,
	}
	if nonDFSNetwork.IsDFS {
		t.Error("expected IsDFS to be false when set to false")
	}
}

func TestScannerSortOrder(t *testing.T) {
	scanner := wifi.NewScanner("en0")
	scanTime := time.Now()

	// Networks with varying signal strengths
	testNetworks := map[string]*wifi.ScannedNetwork{
		"00:00:00:00:00:01": {SSID: "Weak", BSSID: "00:00:00:00:00:01", Signal: -80},
		"00:00:00:00:00:02": {SSID: "Strong", BSSID: "00:00:00:00:00:02", Signal: -30},
		"00:00:00:00:00:03": {SSID: "Medium", BSSID: "00:00:00:00:00:03", Signal: -55},
		"00:00:00:00:00:04": {SSID: "VeryWeak", BSSID: "00:00:00:00:00:04", Signal: -90},
		"00:00:00:00:00:05": {SSID: "Best", BSSID: "00:00:00:00:00:05", Signal: -25},
	}

	scanner.SetCachedNetworks(testNetworks, scanTime)
	networks := scanner.GetCachedNetworks()

	// Verify descending order by signal strength
	expectedOrder := []string{"Best", "Strong", "Medium", "Weak", "VeryWeak"}
	for i, expected := range expectedOrder {
		if networks[i].SSID != expected {
			t.Errorf("network[%d].SSID = %q, want %q", i, networks[i].SSID, expected)
		}
	}
}
