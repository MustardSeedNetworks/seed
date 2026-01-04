package dhcp_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/dhcp"
)

func TestNewRogueDetector(t *testing.T) {
	// Test with nil config - interface is empty (must be set by caller #572)
	rd := dhcp.NewRogueDetector(nil)
	if rd == nil {
		t.Fatal("NewRogueDetector returned nil")
	}
	if rd.RogueConfig() == nil {
		t.Fatal("config should not be nil")
	}
	if rd.RogueConfig().Interface != "" {
		t.Errorf("expected default interface to be empty (must be set by caller), got %s", rd.RogueConfig().Interface)
	}

	// Test with custom config
	config := &dhcp.RogueDetectorConfig{
		Interface:        "wlan0",
		KnownServers:     []string{"192.168.1.1", "192.168.1.2"},
		AlertOnDetection: false,
	}
	rd = dhcp.NewRogueDetector(config)
	if rd.RogueConfig().Interface != "wlan0" {
		t.Errorf("expected interface wlan0, got %s", rd.RogueConfig().Interface)
	}
	if len(rd.RogueKnownServerSet()) != 2 {
		t.Errorf("expected 2 known servers, got %d", len(rd.RogueKnownServerSet()))
	}
	if !rd.RogueKnownServerSet()["192.168.1.1"] {
		t.Error("192.168.1.1 should be in known server set")
	}
}

func TestRogueDetector_IsRunning(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	if rd.IsRunning() {
		t.Error("detector should not be running initially")
	}

	// Manually set running to test getter
	rd.SetRogueRunning(true)

	if !rd.IsRunning() {
		t.Error("detector should be running")
	}
}

func TestRogueDetector_UpdateKnownServers(t *testing.T) {
	config := &dhcp.RogueDetectorConfig{
		Interface:        "eth0",
		KnownServers:     []string{"192.168.1.1"},
		AlertOnDetection: true,
	}
	rd := dhcp.NewRogueDetector(config)

	// Simulate detected servers
	rd.AddDetectedServer(&dhcp.RogueServer{
		IP:           "192.168.1.1",
		IsAuthorized: true,
	})
	rd.AddDetectedServer(&dhcp.RogueServer{
		IP:           "192.168.1.100",
		IsAuthorized: false,
	})

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
	if !rd.RogueKnownServerSet()["192.168.1.100"] {
		t.Error("192.168.1.100 should be in known server set")
	}
}

func TestRogueDetector_GetDetectedServers(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	// Should return empty list initially
	servers := rd.GetDetectedServers()
	if len(servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(servers))
	}

	// Add some servers
	now := time.Now()
	rd.AddDetectedServer(&dhcp.RogueServer{
		IP:           "192.168.1.1",
		MAC:          "aa:bb:cc:dd:ee:ff",
		FirstSeen:    now,
		LastSeen:     now,
		OfferCount:   5,
		IsAuthorized: true,
	})
	rd.AddDetectedServer(&dhcp.RogueServer{
		IP:           "192.168.1.100",
		MAC:          "11:22:33:44:55:66",
		FirstSeen:    now.Add(-1 * time.Hour),
		LastSeen:     now,
		OfferCount:   3,
		IsAuthorized: false,
	})

	servers = rd.GetDetectedServers()
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}

	// Verify server data (order may vary)
	serverMap := make(map[string]*dhcp.RogueServer)
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
	config := &dhcp.RogueDetectorConfig{
		Interface:        "eth0",
		KnownServers:     []string{"192.168.1.1"},
		AlertOnDetection: true,
	}
	rd := dhcp.NewRogueDetector(config)

	// Add authorized and unauthorized servers
	now := time.Now()
	rd.AddDetectedServer(&dhcp.RogueServer{
		IP:           "192.168.1.1",
		FirstSeen:    now,
		IsAuthorized: true,
	})
	rd.AddDetectedServer(&dhcp.RogueServer{
		IP:           "192.168.1.100",
		FirstSeen:    now,
		IsAuthorized: false,
	})
	rd.AddDetectedServer(&dhcp.RogueServer{
		IP:           "192.168.1.200",
		FirstSeen:    now,
		IsAuthorized: false,
	})

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
	rd := dhcp.NewRogueDetector(nil)

	// Add some servers
	rd.AddDetectedServer(&dhcp.RogueServer{IP: "192.168.1.1"})
	rd.AddDetectedServer(&dhcp.RogueServer{IP: "192.168.1.100"})

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
	originalConfig := &dhcp.RogueDetectorConfig{
		Interface:        "wlan0",
		KnownServers:     []string{"192.168.1.1", "192.168.1.2"},
		AlertOnDetection: false,
	}
	rd := dhcp.NewRogueDetector(originalConfig)

	config := rd.GetConfig()

	// Verify config is a copy
	if config == rd.RogueConfig() {
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
	config := &dhcp.RogueDetectorConfig{
		Interface:        "nonexistent999",
		KnownServers:     []string{},
		AlertOnDetection: false,
	}
	rd := dhcp.NewRogueDetector(config)

	err := rd.Start()
	if err == nil {
		t.Error("Start should fail with invalid interface")
		_ = rd.Stop()
	}
}

func TestRogueDetector_Stop_NotRunning(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	// Stopping when not running should not error
	err := rd.Stop()
	if err != nil {
		t.Errorf("Stop should not error when not running: %v", err)
	}
}
