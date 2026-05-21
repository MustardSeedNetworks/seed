package netif

import "sync"

// Mock interface constants for testing.
const (
	mockMTUEthernet  = 1500
	mockMTULoopback  = 65536
	mockSpeedGigabit = 1000000000 // 1 Gbps
	mockSpeedWiFi    = 866000000  // 866 Mbps (WiFi 5 AC)

	// mockEth0 is the canonical mock ethernet interface name used as
	// the default in DefaultMockConfig and as a map key for the same
	// InterfaceInfo entry.
	mockEth0 = "eth0"
)

// MockManagerConfig contains configuration for creating a mock network manager.
type MockManagerConfig struct {
	// CurrentInterface is the currently selected interface name.
	CurrentInterface string
	// Interfaces is the map of interface names to their info.
	Interfaces map[string]*InterfaceInfo
}

// DefaultMockConfig returns a sensible default mock configuration
// with a single ethernet interface for testing.
func DefaultMockConfig() MockManagerConfig {
	return MockManagerConfig{
		CurrentInterface: mockEth0,
		Interfaces: map[string]*InterfaceInfo{
			mockEth0: {
				Name:         mockEth0,
				FriendlyName: "Ethernet 0",
				Description:  "Mock Ethernet Interface",
				Type:         InterfaceTypeEthernet,
				Up:           true,
				Running:      true,
				HardwareAddr: "00:11:22:33:44:55",
				MTU:          mockMTUEthernet,
				Addresses:    []string{"192.168.1.100/24"},
				Speed:        mockSpeedGigabit,
				SpeedDisplay: "1 Gbps",
			},
			"wlan0": {
				Name:         "wlan0",
				FriendlyName: "WiFi",
				Description:  "Mock WiFi Interface",
				Type:         InterfaceTypeWiFi,
				Up:           true,
				Running:      true,
				HardwareAddr: "AA:BB:CC:DD:EE:FF",
				MTU:          mockMTUEthernet,
				Addresses:    []string{"192.168.1.101/24"},
				Speed:        mockSpeedWiFi,
				SpeedDisplay: "866 Mbps",
			},
			"lo": {
				Name:         "lo",
				FriendlyName: "Loopback",
				Description:  "Loopback Interface",
				Type:         InterfaceTypeLoopback,
				Up:           true,
				Running:      true,
				HardwareAddr: "",
				MTU:          mockMTULoopback,
				Addresses:    []string{"127.0.0.1/8", "::1/128"},
			},
		},
	}
}

// NewMockManager creates a network Manager with mock data for testing.
// This bypasses hardware detection and uses pre-configured interface data.
// The returned Manager is fully functional for testing handlers that use
// network interface information.
func NewMockManager(cfg MockManagerConfig) *Manager {
	// Make a copy of the interfaces map to avoid mutation issues
	interfaces := make(map[string]*InterfaceInfo, len(cfg.Interfaces))
	for name, info := range cfg.Interfaces {
		copied := *info
		interfaces[name] = &copied
	}

	return &Manager{
		mu:               sync.RWMutex{},
		currentInterface: cfg.CurrentInterface,
		interfaces:       interfaces,
		detector:         nil, // Not needed for mock
		callbackMu:       sync.RWMutex{},
		callbacks:        nil,
	}
}
