package dhcp_test

import (
	"testing"
	"time"

	"github.com/gopacket/gopacket/layers"

	"github.com/krisarmstrong/seed/internal/dhcp"
)

// TestRogueDetectorNewComprehensive tests NewRogueDetector with various configurations.
func TestRogueDetectorNewComprehensive(t *testing.T) {
	tests := []struct {
		name                 string
		config               *dhcp.RogueDetectorConfig
		expectedInterface    string
		expectedKnownServers int
		expectedAlert        bool
	}{
		{
			name:                 "nil config",
			config:               nil,
			expectedInterface:    "",
			expectedKnownServers: 0,
			expectedAlert:        true,
		},
		{
			name: "empty config",
			config: &dhcp.RogueDetectorConfig{
				Interface:        "",
				KnownServers:     nil,
				AlertOnDetection: false,
			},
			expectedInterface:    "",
			expectedKnownServers: 0,
			expectedAlert:        false,
		},
		{
			name: "full config",
			config: &dhcp.RogueDetectorConfig{
				Interface:        "eth0",
				KnownServers:     []string{"192.168.1.1", "192.168.1.2", "10.0.0.1"},
				AlertOnDetection: true,
			},
			expectedInterface:    "eth0",
			expectedKnownServers: 3,
			expectedAlert:        true,
		},
		{
			name: "single known server",
			config: &dhcp.RogueDetectorConfig{
				Interface:        "wlan0",
				KnownServers:     []string{"192.168.1.1"},
				AlertOnDetection: false,
			},
			expectedInterface:    "wlan0",
			expectedKnownServers: 1,
			expectedAlert:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rd := dhcp.NewRogueDetector(tt.config)

			if rd == nil {
				t.Fatal("expected non-nil detector")
			}

			cfg := rd.GetConfig()
			if cfg.Interface != tt.expectedInterface {
				t.Errorf("Interface = %q, want %q", cfg.Interface, tt.expectedInterface)
			}
			if len(cfg.KnownServers) != tt.expectedKnownServers {
				t.Errorf("KnownServers count = %d, want %d", len(cfg.KnownServers), tt.expectedKnownServers)
			}
			if cfg.AlertOnDetection != tt.expectedAlert {
				t.Errorf("AlertOnDetection = %v, want %v", cfg.AlertOnDetection, tt.expectedAlert)
			}
		})
	}
}

// TestRogueDetectorRecordDetectedServerComprehensive tests recordDetectedServer scenarios.
func TestRogueDetectorRecordDetectedServerComprehensive(t *testing.T) {
	tests := []struct {
		name           string
		knownServers   []string
		serverIP       string
		serverMAC      string
		expectAuth     bool
		expectedOffers int
		recordCount    int
	}{
		{
			name:           "new unauthorized server",
			knownServers:   []string{"192.168.1.1"},
			serverIP:       "192.168.1.100",
			serverMAC:      "aa:bb:cc:dd:ee:ff",
			expectAuth:     false,
			expectedOffers: 1,
			recordCount:    1,
		},
		{
			name:           "new authorized server",
			knownServers:   []string{"192.168.1.1"},
			serverIP:       "192.168.1.1",
			serverMAC:      "11:22:33:44:55:66",
			expectAuth:     true,
			expectedOffers: 1,
			recordCount:    1,
		},
		{
			name:           "multiple records same server",
			knownServers:   []string{},
			serverIP:       "10.0.0.1",
			serverMAC:      "aa:bb:cc:dd:ee:ff",
			expectAuth:     false,
			expectedOffers: 5,
			recordCount:    5,
		},
		{
			name:           "server with empty MAC",
			knownServers:   []string{},
			serverIP:       "172.16.0.1",
			serverMAC:      "",
			expectAuth:     false,
			expectedOffers: 1,
			recordCount:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &dhcp.RogueDetectorConfig{
				Interface:        "eth0",
				KnownServers:     tt.knownServers,
				AlertOnDetection: false, // Suppress alerts in tests
			}
			rd := dhcp.NewRogueDetector(config)

			for range tt.recordCount {
				rd.RecordDetectedServer(tt.serverIP, tt.serverMAC)
			}

			servers := rd.GetDetectedServers()
			if len(servers) != 1 {
				t.Fatalf("expected 1 server, got %d", len(servers))
			}

			server := servers[0]
			if server.IP != tt.serverIP {
				t.Errorf("IP = %q, want %q", server.IP, tt.serverIP)
			}
			if server.IsAuthorized != tt.expectAuth {
				t.Errorf("IsAuthorized = %v, want %v", server.IsAuthorized, tt.expectAuth)
			}
			if server.OfferCount != tt.expectedOffers {
				t.Errorf("OfferCount = %d, want %d", server.OfferCount, tt.expectedOffers)
			}
		})
	}
}

// TestRogueDetectorUpdateKnownServersComprehensive tests UpdateKnownServers scenarios.
func TestRogueDetectorUpdateKnownServersComprehensive(t *testing.T) {
	t.Run("add server to known list", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(&dhcp.RogueDetectorConfig{
			Interface:        "eth0",
			KnownServers:     []string{},
			AlertOnDetection: false,
		})

		// Add rogue server
		rd.RecordDetectedServer("192.168.1.100", "aa:bb:cc:dd:ee:ff")

		// Verify it's unauthorized
		rogues := rd.GetRogueServers()
		if len(rogues) != 1 {
			t.Fatalf("expected 1 rogue, got %d", len(rogues))
		}

		// Update known servers to include this IP
		rd.UpdateKnownServers([]string{"192.168.1.100"})

		// Verify it's now authorized
		rogues = rd.GetRogueServers()
		if len(rogues) != 0 {
			t.Errorf("expected 0 rogues after update, got %d", len(rogues))
		}
	})

	t.Run("remove server from known list", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(&dhcp.RogueDetectorConfig{
			Interface:        "eth0",
			KnownServers:     []string{"192.168.1.1"},
			AlertOnDetection: false,
		})

		// Add authorized server
		rd.RecordDetectedServer("192.168.1.1", "aa:bb:cc:dd:ee:ff")

		// Verify it's authorized
		rogues := rd.GetRogueServers()
		if len(rogues) != 0 {
			t.Fatalf("expected 0 rogues initially, got %d", len(rogues))
		}

		// Update known servers to empty list
		rd.UpdateKnownServers([]string{})

		// Verify it's now unauthorized
		rogues = rd.GetRogueServers()
		if len(rogues) != 1 {
			t.Errorf("expected 1 rogue after update, got %d", len(rogues))
		}
	})

	t.Run("replace known servers", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(&dhcp.RogueDetectorConfig{
			Interface:        "eth0",
			KnownServers:     []string{"192.168.1.1"},
			AlertOnDetection: false,
		})

		// Add servers
		rd.RecordDetectedServer("192.168.1.1", "aa:bb:cc:dd:ee:ff")
		rd.RecordDetectedServer("192.168.1.2", "11:22:33:44:55:66")

		// Replace known servers
		rd.UpdateKnownServers([]string{"192.168.1.2"})

		// Check authorization status
		servers := rd.GetDetectedServers()
		for _, s := range servers {
			if s.IP == "192.168.1.1" && s.IsAuthorized {
				t.Error("192.168.1.1 should not be authorized")
			}
			if s.IP == "192.168.1.2" && !s.IsAuthorized {
				t.Error("192.168.1.2 should be authorized")
			}
		}
	})
}

// TestRogueDetectorClearDetectedServersComprehensive tests ClearDetectedServers.
func TestRogueDetectorClearDetectedServersComprehensive(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	// Add multiple servers
	for i := range 10 {
		ip := "192.168.1." + string(rune('0'+i))
		rd.RecordDetectedServer(ip, "aa:bb:cc:dd:ee:ff")
	}

	// Verify servers exist
	if rd.DetectedServersCount() == 0 {
		t.Fatal("expected servers to be present")
	}

	// Clear
	rd.ClearDetectedServers()

	// Verify cleared
	if rd.DetectedServersCount() != 0 {
		t.Errorf("expected 0 servers after clear, got %d", rd.DetectedServersCount())
	}

	// Verify GetDetectedServers returns empty
	servers := rd.GetDetectedServers()
	if len(servers) != 0 {
		t.Errorf("expected empty slice, got %d servers", len(servers))
	}

	// Verify GetRogueServers returns empty
	rogues := rd.GetRogueServers()
	if len(rogues) != 0 {
		t.Errorf("expected empty rogues, got %d", len(rogues))
	}
}

// TestRogueDetectorGetRogueServersComprehensive tests GetRogueServers filtering.
func TestRogueDetectorGetRogueServersComprehensive(t *testing.T) {
	config := &dhcp.RogueDetectorConfig{
		Interface:        "eth0",
		KnownServers:     []string{"192.168.1.1", "192.168.1.2"},
		AlertOnDetection: false,
	}
	rd := dhcp.NewRogueDetector(config)

	// Add mix of authorized and unauthorized servers
	servers := []struct {
		ip   string
		auth bool
	}{
		{"192.168.1.1", true},    // Known
		{"192.168.1.2", true},    // Known
		{"192.168.1.100", false}, // Rogue
		{"192.168.1.200", false}, // Rogue
		{"10.0.0.1", false},      // Rogue
	}

	for _, s := range servers {
		rd.RecordDetectedServer(s.ip, "aa:bb:cc:dd:ee:ff")
	}

	// Get only rogues
	rogues := rd.GetRogueServers()

	// Count expected rogues
	expectedRogues := 0
	for _, s := range servers {
		if !s.auth {
			expectedRogues++
		}
	}

	if len(rogues) != expectedRogues {
		t.Errorf("expected %d rogues, got %d", expectedRogues, len(rogues))
	}

	// Verify none are authorized
	for _, r := range rogues {
		if r.IsAuthorized {
			t.Errorf("rogue server %s should not be authorized", r.IP)
		}
	}
}

// TestRogueDetectorGetDetectedServersComprehensive tests GetDetectedServers copy behavior.
func TestRogueDetectorGetDetectedServersComprehensive(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	now := time.Now()
	rd.AddDetectedServer(&dhcp.RogueServer{
		IP:           "192.168.1.1",
		MAC:          "aa:bb:cc:dd:ee:ff",
		FirstSeen:    now,
		LastSeen:     now,
		OfferCount:   5,
		IsAuthorized: false,
	})

	// Get servers
	servers := rd.GetDetectedServers()

	// Verify it's a copy
	if len(servers) != 1 {
		t.Fatal("expected 1 server")
	}

	// Modify the returned slice
	servers[0].OfferCount = 999
	servers[0].IsAuthorized = true

	// Get again and verify original is unchanged
	servers2 := rd.GetDetectedServers()
	if servers2[0].OfferCount == 999 {
		t.Error("original should not be modified")
	}
	if servers2[0].IsAuthorized {
		t.Error("original IsAuthorized should not be modified")
	}
}

// TestRogueDetectorPruneExpiredServersComprehensive tests server expiry.
func TestRogueDetectorPruneExpiredServersComprehensive(t *testing.T) {
	t.Run("prune expired servers", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(nil)

		now := time.Now()
		expired := now.Add(-25 * time.Hour) // More than 24 hours
		fresh := now.Add(-1 * time.Hour)    // Less than 24 hours

		// Add 600 servers (over half of 1000 max to trigger pruning)
		for i := range 600 {
			lastSeen := expired
			if i%10 == 0 {
				lastSeen = fresh // Every 10th is fresh
			}
			ip := "192." + string(
				rune('0'+i/256/256%10),
			) + "." + string(
				rune('0'+i/256%10),
			) + "." + string(
				rune('0'+i%256),
			)
			rd.AddDetectedServer(&dhcp.RogueServer{
				IP:       ip,
				LastSeen: lastSeen,
			})
		}

		initialCount := rd.DetectedServersCount()
		rd.PruneExpiredServers(now)

		// Some servers should be pruned
		if rd.DetectedServersCount() >= initialCount {
			t.Errorf("expected fewer servers after pruning")
		}
	})

	t.Run("no pruning under threshold", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(nil)

		now := time.Now()
		expired := now.Add(-25 * time.Hour)

		// Add just a few servers (under threshold)
		for i := range 10 {
			ip := "10.0.0." + string(rune('0'+i))
			rd.AddDetectedServer(&dhcp.RogueServer{
				IP:       ip,
				LastSeen: expired,
			})
		}

		initialCount := rd.DetectedServersCount()
		rd.PruneExpiredServers(now)

		// No pruning should occur
		if rd.DetectedServersCount() != initialCount {
			t.Errorf("expected %d servers, got %d", initialCount, rd.DetectedServersCount())
		}
	})
}

// TestRogueDetectorAddNewServerLimitComprehensive tests max server limit.
func TestRogueDetectorAddNewServerLimitComprehensive(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	now := time.Now()

	// Fill to max (1000)
	for i := range 1000 {
		ip := "10." + string(rune('0'+i/256/256%10)) + "." + string(rune('0'+i/256%10)) + "." + string(rune('0'+i%256))
		rd.AddNewServer(ip, "aa:bb:cc:dd:ee:ff", now)
	}

	if rd.DetectedServersCount() != 1000 {
		t.Errorf("expected 1000 servers, got %d", rd.DetectedServersCount())
	}

	// Try to add one more
	rd.AddNewServer("192.168.99.99", "11:22:33:44:55:66", now)

	// Should still be 1000
	if rd.DetectedServersCount() != 1000 {
		t.Errorf("expected 1000 servers (at limit), got %d", rd.DetectedServersCount())
	}

	// Verify the new one wasn't added
	_, exists := rd.GetDetectedServer("192.168.99.99")
	if exists {
		t.Error("server should not have been added when at limit")
	}
}

// TestRogueDetectorUpdateExistingServerComprehensive tests updating existing servers.
func TestRogueDetectorUpdateExistingServerComprehensive(t *testing.T) {
	t.Run("update MAC when empty", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(nil)
		now := time.Now()

		// Add with empty MAC
		rd.AddNewServer("192.168.1.1", "", now)

		// Get server
		server, _ := rd.GetDetectedServer("192.168.1.1")
		if server.MAC != "" {
			t.Errorf("initial MAC should be empty, got %q", server.MAC)
		}

		// Update with MAC
		rd.UpdateExistingServer(server, "aa:bb:cc:dd:ee:ff", now.Add(time.Second))

		// Verify
		updated, _ := rd.GetDetectedServer("192.168.1.1")
		if updated.MAC != "aa:bb:cc:dd:ee:ff" {
			t.Errorf("MAC should be updated, got %q", updated.MAC)
		}
		if updated.OfferCount != 2 {
			t.Errorf("OfferCount should be 2, got %d", updated.OfferCount)
		}
	})

	t.Run("preserve existing MAC", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(nil)
		now := time.Now()

		// Add with MAC
		rd.AddNewServer("192.168.1.1", "original:mac:00:00:00:00", now)

		// Get server
		server, _ := rd.GetDetectedServer("192.168.1.1")

		// Try to update with different MAC
		rd.UpdateExistingServer(server, "different:mac:11:11:11:11", now.Add(time.Second))

		// Verify original MAC preserved
		updated, _ := rd.GetDetectedServer("192.168.1.1")
		if updated.MAC != "original:mac:00:00:00:00" {
			t.Errorf("original MAC should be preserved, got %q", updated.MAC)
		}
	})

	t.Run("update LastSeen", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(nil)
		now := time.Now()

		rd.AddNewServer("192.168.1.1", "aa:bb:cc:dd:ee:ff", now)

		server, _ := rd.GetDetectedServer("192.168.1.1")
		originalLastSeen := server.LastSeen

		later := now.Add(time.Hour)
		rd.UpdateExistingServer(server, "", later)

		updated, _ := rd.GetDetectedServer("192.168.1.1")
		if !updated.LastSeen.After(originalLastSeen) {
			t.Error("LastSeen should be updated")
		}
	})
}

// TestRogueDetectorSetInterfaceComprehensive tests SetInterface.
func TestRogueDetectorSetInterfaceComprehensive(t *testing.T) {
	t.Run("set interface when not running", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(&dhcp.RogueDetectorConfig{
			Interface: "eth0",
		})

		err := rd.SetInterface("wlan0")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		cfg := rd.GetConfig()
		if cfg.Interface != "wlan0" {
			t.Errorf("Interface = %q, want wlan0", cfg.Interface)
		}
	})

	t.Run("multiple interface changes", func(t *testing.T) {
		rd := dhcp.NewRogueDetector(nil)

		interfaces := []string{"eth0", "eth1", "wlan0", "en0", "lo0"}
		for _, iface := range interfaces {
			err := rd.SetInterface(iface)
			if err != nil {
				t.Errorf("SetInterface(%q) error: %v", iface, err)
			}
			cfg := rd.GetConfig()
			if cfg.Interface != iface {
				t.Errorf("Interface = %q, want %q", cfg.Interface, iface)
			}
		}
	})
}

// TestRogueDetectorGetConfigCopyComprehensive tests config copy behavior.
func TestRogueDetectorGetConfigCopyComprehensive(t *testing.T) {
	original := &dhcp.RogueDetectorConfig{
		Interface:        "eth0",
		KnownServers:     []string{"192.168.1.1", "192.168.1.2"},
		AlertOnDetection: true,
	}
	rd := dhcp.NewRogueDetector(original)

	// Get first copy
	copy1 := rd.GetConfig()

	// Modify the copy
	copy1.Interface = "modified"
	copy1.KnownServers = append(copy1.KnownServers, "10.0.0.1")
	copy1.AlertOnDetection = false

	// Get second copy
	copy2 := rd.GetConfig()

	// Verify second copy is unaffected
	if copy2.Interface != "eth0" {
		t.Errorf("Interface was modified: %q", copy2.Interface)
	}
	if len(copy2.KnownServers) != 2 {
		t.Errorf("KnownServers was modified: %v", copy2.KnownServers)
	}
	if !copy2.AlertOnDetection {
		t.Error("AlertOnDetection was modified")
	}
}

// TestRogueDetectorStartAlreadyRunningComprehensive tests Start when already running.
func TestRogueDetectorStartAlreadyRunningComprehensive(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	// Manually set running
	rd.SetRogueRunning(true)

	err := rd.Start()
	if err == nil {
		t.Error("expected error when starting already running detector")
	}
	if err.Error() != "rogue detector already running" {
		t.Errorf("unexpected error message: %v", err)
	}

	// Reset
	rd.SetRogueRunning(false)
}

// TestRogueDetectorStopNotRunningComprehensive tests Stop when not running.
func TestRogueDetectorStopNotRunningComprehensive(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	// Stop when not running should not error
	err := rd.Stop()
	if err != nil {
		t.Errorf("Stop should not error when not running: %v", err)
	}

	// Multiple stops should be safe
	for range 5 {
		err = rd.Stop()
		if err != nil {
			t.Errorf("multiple Stop calls should be safe: %v", err)
		}
	}
}

// TestRogueDetectorIsRunningComprehensive tests IsRunning state.
func TestRogueDetectorIsRunningComprehensive(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	// Initially not running
	if rd.IsRunning() {
		t.Error("should not be running initially")
	}

	// Manually set running
	rd.SetRogueRunning(true)
	if !rd.IsRunning() {
		t.Error("should be running after setting")
	}

	// Manually unset
	rd.SetRogueRunning(false)
	if rd.IsRunning() {
		t.Error("should not be running after unsetting")
	}
}

// TestRogueDetectorExportGetServerIdentifierComprehensive tests getServerIdentifier.
func TestRogueDetectorExportGetServerIdentifierComprehensive(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	tests := []struct {
		name     string
		dhcp     *dhcp.MockDHCPv4
		expected string
	}{
		{
			name: "with server ID option (192.168.1.1)",
			dhcp: &dhcp.MockDHCPv4{
				Options: []dhcp.MockDHCPOption{
					{Type: 54, Data: []byte{192, 168, 1, 1}}, // Server Identifier
				},
			},
			expected: "192.168.1.1",
		},
		{
			name: "with server ID option (10.0.0.1)",
			dhcp: &dhcp.MockDHCPv4{
				Options: []dhcp.MockDHCPOption{
					{Type: 54, Data: []byte{10, 0, 0, 1}},
				},
			},
			expected: "10.0.0.1",
		},
		{
			name: "with server ID option (8.8.8.8)",
			dhcp: &dhcp.MockDHCPv4{
				Options: []dhcp.MockDHCPOption{
					{Type: 54, Data: []byte{8, 8, 8, 8}},
				},
			},
			expected: "8.8.8.8",
		},
		{
			name:     "empty options",
			dhcp:     &dhcp.MockDHCPv4{Options: []dhcp.MockDHCPOption{}},
			expected: "",
		},
		{
			name: "wrong option type",
			dhcp: &dhcp.MockDHCPv4{
				Options: []dhcp.MockDHCPOption{
					{Type: 53, Data: []byte{2}}, // Message Type instead
				},
			},
			expected: "",
		},
		{
			name: "wrong data length",
			dhcp: &dhcp.MockDHCPv4{
				Options: []dhcp.MockDHCPOption{
					{Type: 54, Data: []byte{192, 168}}, // Only 2 bytes
				},
			},
			expected: "",
		},
		{
			name: "multiple options with server ID",
			dhcp: &dhcp.MockDHCPv4{
				Options: []dhcp.MockDHCPOption{
					{Type: 53, Data: []byte{2}},               // Message Type
					{Type: 1, Data: []byte{255, 255, 255, 0}}, // Subnet Mask
					{Type: 54, Data: []byte{172, 16, 0, 1}},   // Server Identifier
				},
			},
			expected: "172.16.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rd.ExportGetServerIdentifier(tt.dhcp.ToLayers())
			if result != tt.expected {
				t.Errorf("getServerIdentifier() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestRogueDetectorExportGetDHCPMessageTypeComprehensive tests getDHCPMessageType.
func TestRogueDetectorExportGetDHCPMessageTypeComprehensive(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	tests := []struct {
		name     string
		dhcp     *dhcp.MockDHCPv4
		expected layers.DHCPMsgType
	}{
		{
			name: "DISCOVER",
			dhcp: &dhcp.MockDHCPv4{
				Options: []dhcp.MockDHCPOption{
					{Type: 53, Data: []byte{1}},
				},
			},
			expected: layers.DHCPMsgTypeDiscover,
		},
		{
			name: "OFFER",
			dhcp: &dhcp.MockDHCPv4{
				Options: []dhcp.MockDHCPOption{
					{Type: 53, Data: []byte{2}},
				},
			},
			expected: layers.DHCPMsgTypeOffer,
		},
		{
			name: "REQUEST",
			dhcp: &dhcp.MockDHCPv4{
				Options: []dhcp.MockDHCPOption{
					{Type: 53, Data: []byte{3}},
				},
			},
			expected: layers.DHCPMsgTypeRequest,
		},
		{
			name: "DECLINE",
			dhcp: &dhcp.MockDHCPv4{
				Options: []dhcp.MockDHCPOption{
					{Type: 53, Data: []byte{4}},
				},
			},
			expected: layers.DHCPMsgTypeDecline,
		},
		{
			name: "ACK",
			dhcp: &dhcp.MockDHCPv4{
				Options: []dhcp.MockDHCPOption{
					{Type: 53, Data: []byte{5}},
				},
			},
			expected: layers.DHCPMsgTypeAck,
		},
		{
			name: "NAK",
			dhcp: &dhcp.MockDHCPv4{
				Options: []dhcp.MockDHCPOption{
					{Type: 53, Data: []byte{6}},
				},
			},
			expected: layers.DHCPMsgTypeNak,
		},
		{
			name:     "empty options",
			dhcp:     &dhcp.MockDHCPv4{Options: []dhcp.MockDHCPOption{}},
			expected: 0,
		},
		{
			name: "empty data",
			dhcp: &dhcp.MockDHCPv4{
				Options: []dhcp.MockDHCPOption{
					{Type: 53, Data: []byte{}},
				},
			},
			expected: 0,
		},
		{
			name: "wrong option type",
			dhcp: &dhcp.MockDHCPv4{
				Options: []dhcp.MockDHCPOption{
					{Type: 54, Data: []byte{192, 168, 1, 1}},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rd.ExportGetDHCPMessageType(tt.dhcp.ToLayers())
			if result != tt.expected {
				t.Errorf("getDHCPMessageType() = %d, want %d", result, tt.expected)
			}
		})
	}
}

// TestRogueServerFieldsComprehensive tests RogueServer struct fields.
func TestRogueServerFieldsComprehensive(t *testing.T) {
	now := time.Now()

	servers := []dhcp.RogueServer{
		{
			IP:           "192.168.1.1",
			MAC:          "aa:bb:cc:dd:ee:ff",
			FirstSeen:    now,
			LastSeen:     now.Add(time.Hour),
			OfferCount:   100,
			IsAuthorized: true,
		},
		{
			IP:           "10.0.0.1",
			MAC:          "",
			FirstSeen:    now,
			LastSeen:     now,
			OfferCount:   1,
			IsAuthorized: false,
		},
		{
			IP:           "172.16.0.1",
			MAC:          "11:22:33:44:55:66",
			FirstSeen:    now.Add(-24 * time.Hour),
			LastSeen:     now,
			OfferCount:   50,
			IsAuthorized: false,
		},
	}

	for _, server := range servers {
		if server.IP == "" {
			t.Error("IP should not be empty")
		}
		if server.OfferCount < 0 {
			t.Error("OfferCount should not be negative")
		}
		if server.LastSeen.Before(server.FirstSeen) {
			t.Error("LastSeen should not be before FirstSeen")
		}
	}
}

// TestRogueDetectorConcurrentAccessComprehensive tests concurrent access patterns.
func TestRogueDetectorConcurrentAccessComprehensive(t *testing.T) {
	rd := dhcp.NewRogueDetector(nil)

	done := make(chan bool, 100)

	// Writers
	for i := range 10 {
		go func(id int) {
			for j := range 50 {
				ip := "192.168." + string(rune('0'+id)) + "." + string(rune('0'+j%10))
				rd.RecordDetectedServer(ip, "aa:bb:cc:dd:ee:ff")
			}
			done <- true
		}(i)
	}

	// Readers
	for i := range 10 {
		go func(_ int) {
			for range 50 {
				_ = rd.GetDetectedServers()
				_ = rd.GetRogueServers()
				_ = rd.IsRunning()
				_ = rd.GetConfig()
			}
			done <- true
		}(i)
	}

	// Wait for all
	for range 20 {
		<-done
	}

	// Verify no panic occurred
	if rd.DetectedServersCount() == 0 {
		t.Error("expected some servers to be recorded")
	}
}

// TestRogueDetectorStartInvalidInterfaceComprehensive tests Start with invalid interface.
func TestRogueDetectorStartInvalidInterfaceComprehensive(t *testing.T) {
	invalidInterfaces := []string{
		"nonexistent_interface_12345",
		"invalid-if-name",
		"eth99999",
		"",
	}

	for _, iface := range invalidInterfaces {
		t.Run("interface_"+iface, func(t *testing.T) {
			rd := dhcp.NewRogueDetector(&dhcp.RogueDetectorConfig{
				Interface:        iface,
				AlertOnDetection: false,
			})

			err := rd.Start()
			if err == nil {
				t.Error("expected error with invalid interface")
				_ = rd.Stop()
			}
		})
	}
}
