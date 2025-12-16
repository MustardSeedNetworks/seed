// Package dhcp provides DHCP monitoring including rogue DHCP server detection.
// Test suite validates rogue server detection, packet parsing, and security alerts.
package dhcp

import (
	"testing"
	"time"
)

func TestNewRogueDetector(t *testing.T) {
	// Test with nil config - interface is empty (must be set by caller #572)
	rd := NewRogueDetector(nil)
	if rd == nil {
		t.Fatal("NewRogueDetector returned nil")
	}
	if rd.config == nil {
		t.Fatal("config should not be nil")
	}
	if rd.config.Interface != "" {
		t.Errorf("expected default interface to be empty (must be set by caller), got %s", rd.config.Interface)
	}

	// Test with custom config
	config := &RogueDetectorConfig{
		Interface:        "wlan0",
		KnownServers:     []string{"192.168.1.1", "192.168.1.2"},
		AlertOnDetection: false,
	}
	rd = NewRogueDetector(config)
	if rd.config.Interface != "wlan0" {
		t.Errorf("expected interface wlan0, got %s", rd.config.Interface)
	}
	if len(rd.knownServerSet) != 2 {
		t.Errorf("expected 2 known servers, got %d", len(rd.knownServerSet))
	}
	if !rd.knownServerSet["192.168.1.1"] {
		t.Error("192.168.1.1 should be in known server set")
	}
}

func TestRogueDetector_IsRunning(t *testing.T) {
	rd := NewRogueDetector(nil)

	if rd.IsRunning() {
		t.Error("detector should not be running initially")
	}

	// Manually set running to test getter
	rd.mu.Lock()
	rd.running = true
	rd.mu.Unlock()

	if !rd.IsRunning() {
		t.Error("detector should be running")
	}
}

func TestRogueDetector_UpdateKnownServers(t *testing.T) {
	config := &RogueDetectorConfig{
		Interface:        "eth0",
		KnownServers:     []string{"192.168.1.1"},
		AlertOnDetection: true,
	}
	rd := NewRogueDetector(config)

	// Simulate detected servers
	rd.mu.Lock()
	rd.detectedServers["192.168.1.1"] = &RogueServer{
		IP:           "192.168.1.1",
		IsAuthorized: true,
	}
	rd.detectedServers["192.168.1.100"] = &RogueServer{
		IP:           "192.168.1.100",
		IsAuthorized: false,
	}
	rd.mu.Unlock()

	// Update known servers to include the rogue
	rd.UpdateKnownServers([]string{"192.168.1.1", "192.168.1.100"})

	// Verify both servers are now authorized
	servers := rd.GetDetectedServers()
	for _, server := range servers {
		if !server.IsAuthorized {
			t.Errorf("server %s should now be authorized", server.IP)
		}
	}

	// Verify known server set
	rd.mu.RLock()
	if !rd.knownServerSet["192.168.1.100"] {
		t.Error("192.168.1.100 should be in known server set")
	}
	rd.mu.RUnlock()
}

func TestRogueDetector_GetDetectedServers(t *testing.T) {
	rd := NewRogueDetector(nil)

	// Should return empty list initially
	servers := rd.GetDetectedServers()
	if len(servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(servers))
	}

	// Add some servers
	now := time.Now()
	rd.mu.Lock()
	rd.detectedServers["192.168.1.1"] = &RogueServer{
		IP:           "192.168.1.1",
		MAC:          "aa:bb:cc:dd:ee:ff",
		FirstSeen:    now,
		LastSeen:     now,
		OfferCount:   5,
		IsAuthorized: true,
	}
	rd.detectedServers["192.168.1.100"] = &RogueServer{
		IP:           "192.168.1.100",
		MAC:          "11:22:33:44:55:66",
		FirstSeen:    now.Add(-1 * time.Hour),
		LastSeen:     now,
		OfferCount:   3,
		IsAuthorized: false,
	}
	rd.mu.Unlock()

	servers = rd.GetDetectedServers()
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}

	// Verify server data (order may vary)
	serverMap := make(map[string]*RogueServer)
	for _, s := range servers {
		serverMap[s.IP] = s
	}

	if server, ok := serverMap["192.168.1.1"]; !ok {
		t.Error("192.168.1.1 not found")
	} else {
		if server.OfferCount != 5 {
			t.Errorf("expected offer count 5, got %d", server.OfferCount)
		}
		if !server.IsAuthorized {
			t.Error("192.168.1.1 should be authorized")
		}
	}

	if server, ok := serverMap["192.168.1.100"]; !ok {
		t.Error("192.168.1.100 not found")
	} else {
		if server.OfferCount != 3 {
			t.Errorf("expected offer count 3, got %d", server.OfferCount)
		}
		if server.IsAuthorized {
			t.Error("192.168.1.100 should not be authorized")
		}
	}
}

func TestRogueDetector_GetRogueServers(t *testing.T) {
	config := &RogueDetectorConfig{
		Interface:        "eth0",
		KnownServers:     []string{"192.168.1.1"},
		AlertOnDetection: true,
	}
	rd := NewRogueDetector(config)

	// Add authorized and unauthorized servers
	now := time.Now()
	rd.mu.Lock()
	rd.detectedServers["192.168.1.1"] = &RogueServer{
		IP:           "192.168.1.1",
		FirstSeen:    now,
		IsAuthorized: true,
	}
	rd.detectedServers["192.168.1.100"] = &RogueServer{
		IP:           "192.168.1.100",
		FirstSeen:    now,
		IsAuthorized: false,
	}
	rd.detectedServers["192.168.1.200"] = &RogueServer{
		IP:           "192.168.1.200",
		FirstSeen:    now,
		IsAuthorized: false,
	}
	rd.mu.Unlock()

	rogues := rd.GetRogueServers()
	if len(rogues) != 2 {
		t.Fatalf("expected 2 rogue servers, got %d", len(rogues))
	}

	// Verify only unauthorized servers returned
	for _, server := range rogues {
		if server.IsAuthorized {
			t.Errorf("server %s should not be authorized", server.IP)
		}
		if server.IP != "192.168.1.100" && server.IP != "192.168.1.200" {
			t.Errorf("unexpected server IP: %s", server.IP)
		}
	}
}

func TestRogueDetector_ClearDetectedServers(t *testing.T) {
	rd := NewRogueDetector(nil)

	// Add some servers
	rd.mu.Lock()
	rd.detectedServers["192.168.1.1"] = &RogueServer{IP: "192.168.1.1"}
	rd.detectedServers["192.168.1.100"] = &RogueServer{IP: "192.168.1.100"}
	rd.mu.Unlock()

	// Verify they exist
	if len(rd.GetDetectedServers()) != 2 {
		t.Fatal("servers should be present before clear")
	}

	// Clear
	rd.ClearDetectedServers()

	// Verify cleared
	servers := rd.GetDetectedServers()
	if len(servers) != 0 {
		t.Errorf("expected 0 servers after clear, got %d", len(servers))
	}
}

func TestRogueDetector_GetConfig(t *testing.T) {
	originalConfig := &RogueDetectorConfig{
		Interface:        "wlan0",
		KnownServers:     []string{"192.168.1.1", "192.168.1.2"},
		AlertOnDetection: false,
	}
	rd := NewRogueDetector(originalConfig)

	config := rd.GetConfig()

	// Verify config is a copy
	if config == rd.config {
		t.Error("GetConfig should return a copy, not the original")
	}

	// Verify values
	if config.Interface != "wlan0" {
		t.Errorf("expected interface wlan0, got %s", config.Interface)
	}
	if len(config.KnownServers) != 2 {
		t.Errorf("expected 2 known servers, got %d", len(config.KnownServers))
	}
	if config.AlertOnDetection {
		t.Error("AlertOnDetection should be false")
	}

	// Modify returned config should not affect original
	config.KnownServers = []string{"10.0.0.1"}
	retrievedConfig := rd.GetConfig()
	if len(retrievedConfig.KnownServers) != 2 {
		t.Error("modifying returned config should not affect original")
	}
}

func TestRogueDetector_Start_InvalidInterface(t *testing.T) {
	config := &RogueDetectorConfig{
		Interface:        "nonexistent999",
		KnownServers:     []string{},
		AlertOnDetection: false,
	}
	rd := NewRogueDetector(config)

	err := rd.Start()
	if err == nil {
		t.Error("Start should fail with invalid interface")
		rd.Stop()
	}
}

func TestRogueDetector_Stop_NotRunning(t *testing.T) {
	rd := NewRogueDetector(nil)

	// Stopping when not running should not error
	err := rd.Stop()
	if err != nil {
		t.Errorf("Stop should not error when not running: %v", err)
	}
}
