// Package wifi_test provides wireless network information functionality tests.
// Test suite validates network detection, interface checking, and manager operations.
package wifi_test

import (
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/canopy/wifi"
)

// TestManagerCreation tests various Manager creation scenarios.
func TestManagerCreation(t *testing.T) {
	tests := []struct {
		name          string
		interfaceName string
		wantInterface string
	}{
		{"macOS en0", "en0", "en0"},
		{"macOS en1", "en1", "en1"},
		{"Linux wlan0", "wlan0", "wlan0"},
		{"Linux wlan1", "wlan1", "wlan1"},
		{"Linux wlp2s0", "wlp2s0", "wlp2s0"},
		{"Empty interface", "", ""},
		{"Custom interface", "wifi0", "wifi0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := wifi.NewManager(tt.interfaceName)
			if manager == nil {
				t.Fatal("NewManager returned nil")
			}

			if manager.InterfaceName() != tt.wantInterface {
				t.Errorf("InterfaceName() = %q, want %q", manager.InterfaceName(), tt.wantInterface)
			}
		})
	}
}

// TestManagerSetInterfaceMultiple tests changing interface multiple times.
func TestManagerSetInterfaceMultiple(t *testing.T) {
	manager := wifi.NewManager("en0")

	interfaces := []string{"wlan0", "en1", "wlan1", "en0", "eth0", "wifi0"}

	for _, iface := range interfaces {
		manager.SetInterface(iface)
		if got := manager.InterfaceName(); got != iface {
			t.Errorf("after SetInterface(%q), InterfaceName() = %q, want %q", iface, got, iface)
		}
	}
}

// TestManagerConcurrentSetAndGet tests concurrent access to Manager.
func TestManagerConcurrentSetAndGet(t *testing.T) {
	manager := wifi.NewManager("en0")

	var wg sync.WaitGroup
	const numGoroutines = 20
	const numIterations = 100

	for i := range numGoroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range numIterations {
				// Mix of read and write operations
				if j%2 == 0 {
					manager.SetInterface("iface" + string(rune('0'+id%10)))
				}
				_ = manager.InterfaceName()
				_ = manager.IsWireless()
			}
		}(i)
	}

	wg.Wait()
}

// TestScannerCreation tests various Scanner creation scenarios.
func TestScannerCreation(t *testing.T) {
	tests := []struct {
		name          string
		interfaceName string
		wantInterface string
	}{
		{"macOS en0", "en0", "en0"},
		{"Linux wlan0", "wlan0", "wlan0"},
		{"Empty interface", "", ""},
		{"Custom interface", "wifi0", "wifi0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanner := wifi.NewScanner(tt.interfaceName)
			if scanner == nil {
				t.Fatal("NewScanner returned nil")
			}

			if got := scanner.ScannerInterfaceName(); got != tt.wantInterface {
				t.Errorf("ScannerInterfaceName() = %q, want %q", got, tt.wantInterface)
			}
		})
	}
}

// TestScannerSetInterfaceMultiple tests changing scanner interface multiple times.
func TestScannerSetInterfaceMultiple(t *testing.T) {
	scanner := wifi.NewScanner("en0")

	interfaces := []string{"wlan0", "en1", "wlan1", "en0", "eth0", "wifi0"}

	for _, iface := range interfaces {
		scanner.SetInterface(iface)
		if got := scanner.ScannerInterfaceName(); got != iface {
			t.Errorf("after SetInterface(%q), ScannerInterfaceName() = %q, want %q", iface, got, iface)
		}
	}
}

// TestScannerCachePersistence tests that cached networks persist correctly.
func TestScannerCachePersistence(t *testing.T) {
	scanner := wifi.NewScanner("en0")

	// Set initial networks
	initialTime := time.Now().Add(-time.Minute)
	initialNetworks := map[string]*wifi.ScannedNetwork{
		"00:11:22:33:44:55": {
			SSID:      "InitialNet",
			BSSID:     "00:11:22:33:44:55",
			Signal:    -50,
			Channel:   6,
			Frequency: 2437,
		},
	}
	scanner.SetCachedNetworks(initialNetworks, initialTime)

	// Verify initial state
	networks := scanner.GetCachedNetworks()
	if len(networks) != 1 {
		t.Fatalf("expected 1 network, got %d", len(networks))
	}
	if networks[0].SSID != "InitialNet" {
		t.Errorf("expected SSID 'InitialNet', got %q", networks[0].SSID)
	}

	// Update with new networks
	newTime := time.Now()
	newNetworks := map[string]*wifi.ScannedNetwork{
		"AA:BB:CC:DD:EE:FF": {
			SSID:      "UpdatedNet1",
			BSSID:     "AA:BB:CC:DD:EE:FF",
			Signal:    -45,
			Channel:   1,
			Frequency: 2412,
		},
		"11:22:33:44:55:66": {
			SSID:      "UpdatedNet2",
			BSSID:     "11:22:33:44:55:66",
			Signal:    -55,
			Channel:   11,
			Frequency: 2462,
		},
	}
	scanner.SetCachedNetworks(newNetworks, newTime)

	// Verify update
	networks = scanner.GetCachedNetworks()
	if len(networks) != 2 {
		t.Fatalf("expected 2 networks, got %d", len(networks))
	}

	// Verify time was updated
	lastScan := scanner.GetLastScanTime()
	if !lastScan.Equal(newTime) {
		t.Errorf("GetLastScanTime() = %v, want %v", lastScan, newTime)
	}
}

// TestScannerConcurrentReadWrite tests concurrent access to Scanner cache.
func TestScannerConcurrentReadWrite(t *testing.T) {
	scanner := wifi.NewScanner("en0")
	scanTime := time.Now()

	// Initial networks
	networks := map[string]*wifi.ScannedNetwork{
		"00:00:00:00:00:01": {
			SSID:   "Net1",
			BSSID:  "00:00:00:00:00:01",
			Signal: -50,
		},
	}
	scanner.SetCachedNetworks(networks, scanTime)

	var wg sync.WaitGroup
	const numReaders = 10
	const numWriters = 3
	const numIterations = 50

	// Start readers
	for range numReaders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range numIterations {
				_ = scanner.GetCachedNetworks()
				_ = scanner.GetLastScanTime()
				_ = scanner.ScannerInterfaceName()
			}
		}()
	}

	// Start writers
	for i := range numWriters {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range numIterations {
				newNetworks := map[string]*wifi.ScannedNetwork{
					"00:00:00:00:00:0" + string(rune('0'+id)): {
						SSID:   "Net" + string(rune('0'+j%10)),
						BSSID:  "00:00:00:00:00:0" + string(rune('0'+id)),
						Signal: -50 - j,
					},
				}
				scanner.SetCachedNetworks(newNetworks, time.Now())
				scanner.SetInterface("en" + string(rune('0'+id)))
			}
		}(i)
	}

	wg.Wait()
}

// TestInterfaceNamePatterns tests various interface name patterns.
func TestInterfaceNamePatterns(t *testing.T) {
	tests := []struct {
		name    string
		iface   string
		isValid bool
	}{
		// macOS patterns
		{"macOS WiFi primary", "en0", true},
		{"macOS WiFi secondary", "en1", true},
		{"macOS Thunderbolt", "en4", true},

		// Linux patterns
		{"Linux wlan standard", "wlan0", true},
		{"Linux wlan secondary", "wlan1", true},
		{"Linux predictable names", "wlp2s0", true},
		{"Linux predictable names alt", "wlp3s0b1", true},

		// Edge cases
		{"Empty interface", "", true},
		{"Numbered interface", "wifi0", true},
		{"Long interface name", "verylonginterfacename0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := wifi.NewManager(tt.iface)
			if manager == nil {
				t.Fatal("NewManager returned nil")
			}

			got := manager.InterfaceName()
			if got != tt.iface {
				t.Errorf("InterfaceName() = %q, want %q", got, tt.iface)
			}
		})
	}
}

// TestScannerEmptyCache tests behavior with empty cache.
func TestScannerEmptyCache(t *testing.T) {
	scanner := wifi.NewScanner("en0")

	// Empty cache should return empty slice, not nil
	networks := scanner.GetCachedNetworks()
	if networks == nil {
		t.Error("GetCachedNetworks() returned nil, want empty slice")
	}
	if len(networks) != 0 {
		t.Errorf("GetCachedNetworks() returned %d networks, want 0", len(networks))
	}

	// Last scan time should be zero
	lastScan := scanner.GetLastScanTime()
	if !lastScan.IsZero() {
		t.Errorf("GetLastScanTime() = %v, want zero time", lastScan)
	}
}

// TestScannerCacheOverwrite tests that cache is properly overwritten.
func TestScannerCacheOverwrite(t *testing.T) {
	scanner := wifi.NewScanner("en0")

	// First set of networks
	time1 := time.Now().Add(-2 * time.Minute)
	networks1 := map[string]*wifi.ScannedNetwork{
		"00:00:00:00:00:01": {SSID: "Net1", BSSID: "00:00:00:00:00:01", Signal: -50},
		"00:00:00:00:00:02": {SSID: "Net2", BSSID: "00:00:00:00:00:02", Signal: -60},
		"00:00:00:00:00:03": {SSID: "Net3", BSSID: "00:00:00:00:00:03", Signal: -70},
	}
	scanner.SetCachedNetworks(networks1, time1)

	if len(scanner.GetCachedNetworks()) != 3 {
		t.Fatalf("expected 3 networks after first set")
	}

	// Second set with different networks (overwrites completely)
	time2 := time.Now()
	networks2 := map[string]*wifi.ScannedNetwork{
		"AA:BB:CC:DD:EE:FF": {SSID: "NewNet", BSSID: "AA:BB:CC:DD:EE:FF", Signal: -45},
	}
	scanner.SetCachedNetworks(networks2, time2)

	cached := scanner.GetCachedNetworks()
	if len(cached) != 1 {
		t.Fatalf("expected 1 network after overwrite, got %d", len(cached))
	}
	if cached[0].SSID != "NewNet" {
		t.Errorf("expected SSID 'NewNet', got %q", cached[0].SSID)
	}

	// Verify old networks are gone
	for _, network := range cached {
		if network.SSID == "Net1" || network.SSID == "Net2" || network.SSID == "Net3" {
			t.Errorf("old network %q still present after overwrite", network.SSID)
		}
	}
}

// TestManagerAndScannerIndependence tests that Manager and Scanner are independent.
func TestManagerAndScannerIndependence(t *testing.T) {
	manager := wifi.NewManager("en0")
	scanner := wifi.NewScanner("wlan0")

	// Verify they have different interfaces
	if manager.InterfaceName() == scanner.ScannerInterfaceName() {
		t.Error("Manager and Scanner should have different interfaces")
	}

	// Change Manager interface
	manager.SetInterface("en1")

	// Scanner should be unaffected
	if scanner.ScannerInterfaceName() != "wlan0" {
		t.Errorf("Scanner interface changed unexpectedly to %q", scanner.ScannerInterfaceName())
	}

	// Change Scanner interface
	scanner.SetInterface("wlan1")

	// Manager should be unaffected
	if manager.InterfaceName() != "en1" {
		t.Errorf("Manager interface changed unexpectedly to %q", manager.InterfaceName())
	}
}

// TestMultipleManagerInstances tests multiple Manager instances.
func TestMultipleManagerInstances(t *testing.T) {
	managers := make([]*wifi.Manager, 5)
	for i := range managers {
		managers[i] = wifi.NewManager("en" + string(rune('0'+i)))
	}

	// Verify each has correct interface
	for i, m := range managers {
		expected := "en" + string(rune('0'+i))
		if m.InterfaceName() != expected {
			t.Errorf("manager[%d].InterfaceName() = %q, want %q", i, m.InterfaceName(), expected)
		}
	}

	// Change one and verify others are unaffected
	managers[2].SetInterface("wlan0")

	for i, m := range managers {
		if i == 2 {
			if m.InterfaceName() != "wlan0" {
				t.Errorf("manager[2] should be wlan0, got %q", m.InterfaceName())
			}
		} else {
			expected := "en" + string(rune('0'+i))
			if m.InterfaceName() != expected {
				t.Errorf("manager[%d] changed unexpectedly to %q", i, m.InterfaceName())
			}
		}
	}
}

// TestMultipleScannerInstances tests multiple Scanner instances.
func TestMultipleScannerInstances(t *testing.T) {
	scanners := make([]*wifi.Scanner, 3)
	for i := range scanners {
		scanners[i] = wifi.NewScanner("wlan" + string(rune('0'+i)))
	}

	// Set different networks for each
	for i, s := range scanners {
		networks := map[string]*wifi.ScannedNetwork{
			"00:00:00:00:00:0" + string(rune('0'+i)): {
				SSID:   "Net" + string(rune('0'+i)),
				BSSID:  "00:00:00:00:00:0" + string(rune('0'+i)),
				Signal: -50 - i*10,
			},
		}
		s.SetCachedNetworks(networks, time.Now())
	}

	// Verify each has correct networks
	for i, s := range scanners {
		cached := s.GetCachedNetworks()
		if len(cached) != 1 {
			t.Fatalf("scanner[%d] has %d networks, want 1", i, len(cached))
		}
		expectedSSID := "Net" + string(rune('0'+i))
		if cached[0].SSID != expectedSSID {
			t.Errorf("scanner[%d] SSID = %q, want %q", i, cached[0].SSID, expectedSSID)
		}
	}
}

// TestScannerLargeNetworkCount tests handling of many networks.
func TestScannerLargeNetworkCount(t *testing.T) {
	scanner := wifi.NewScanner("en0")

	// Create many networks
	const numNetworks = 100
	networks := make(map[string]*wifi.ScannedNetwork, numNetworks)

	for i := range numNetworks {
		bssid := "00:00:00:00:" + string(rune('0'+i/10)) + string(rune('0'+i%10)) + ":ff"
		networks[bssid] = &wifi.ScannedNetwork{
			SSID:      "Network" + string(rune('0'+i%10)),
			BSSID:     bssid,
			Signal:    -30 - (i % 70), // Signals from -30 to -99
			Channel:   1 + (i % 11),
			Frequency: 2412 + (i%11)*5,
		}
	}

	scanner.SetCachedNetworks(networks, time.Now())

	cached := scanner.GetCachedNetworks()
	if len(cached) != numNetworks {
		t.Fatalf("expected %d networks, got %d", numNetworks, len(cached))
	}

	// Verify sorting
	for i := range len(cached) - 1 {
		if cached[i].Signal < cached[i+1].Signal {
			t.Errorf("networks not sorted at index %d: %d < %d",
				i, cached[i].Signal, cached[i+1].Signal)
		}
	}
}
