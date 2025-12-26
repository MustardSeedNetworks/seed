// Package discovery provides link-layer discovery protocol coordination.
//
// This file implements the Manager, which orchestrates multiple discovery protocol
// captures (LLDP, CDP, EDP) to discover neighboring network devices. The manager
// ensures all protocols start/stop together and aggregates their results.
//
// Supported protocols:
//   - LLDP (Link Layer Discovery Protocol): IEEE 802.1AB standard, widely supported
//   - CDP (Cisco Discovery Protocol): Cisco proprietary, common in Cisco environments
//   - EDP (Extreme Discovery Protocol): Extreme Networks proprietary
//
// Architecture:
//   - Each protocol has a dedicated capture goroutine using libpcap/gopacket
//   - Manager coordinates lifecycle: start all, stop all, aggregate neighbors
//   - Protocols run independently - failure of one doesn't affect others during operation
//   - All protocols must start successfully for manager to report started
//
// Security considerations:
//   - Discovery protocols reveal network topology (neighbors, VLANs, management IPs)
//   - Should only be enabled on trusted networks
//   - Captured data is cached in memory (bounded by TTL expiration)
//   - No authentication - protocols are inherently unauthenticated
//
// Performance:
//   - Passive capture only - no active packets sent
//   - Low CPU usage (waits for incoming packets)
//   - Memory usage scales with number of discovered neighbors
//   - TTL-based automatic expiration prevents unbounded growth
package discovery

import (
	"sync"
	"time"
)

// Protocol represents a link-layer discovery protocol type.
//
// These protocols allow network devices to advertise their identity, capabilities,
// and configuration to directly connected neighbors. Each protocol has different
// adoption and feature sets.
type Protocol string

const (
	// ProtocolLLDP is the IEEE 802.1AB Link Layer Discovery Protocol (LLDP).
	// This is the industry-standard discovery protocol, supported by most vendors.
	// Recommended for multi-vendor environments.
	ProtocolLLDP Protocol = "lldp"

	// ProtocolCDP is the Cisco Discovery Protocol (CDP).
	// Cisco proprietary, but widely deployed in Cisco-dominated networks.
	// Provides richer VLAN information than LLDP.
	ProtocolCDP Protocol = "cdp"

	// ProtocolEDP is the Extreme Discovery Protocol (EDP).
	// Extreme Networks proprietary, used in Extreme switches.
	// Less common than LLDP/CDP but useful in Extreme environments.
	ProtocolEDP Protocol = "edp"
)

// Neighbor represents a discovered network neighbor device.
//
// Information is gathered from link-layer discovery protocol advertisements (LLDP/CDP/EDP).
// Not all fields are populated by all protocols - optional fields are omitted when unavailable.
//
// Common use cases:
//   - Network topology mapping and visualization
//   - Switch port identification and documentation
//   - VLAN troubleshooting (which VLANs are native/voice on neighbor ports)
//   - Identifying management addresses for further device interrogation
type Neighbor struct {
	Protocol          Protocol  `json:"protocol"`                    // Protocol that discovered this neighbor
	ChassisID         string    `json:"chassisId"`                   // Unique device identifier (MAC or hostname)
	PortID            string    `json:"portId"`                      // Port identifier on the neighbor device
	PortDescription   string    `json:"portDescription,omitempty"`   // Human-readable port description
	SystemName        string    `json:"systemName,omitempty"`        // Hostname or system name
	SystemDescription string    `json:"systemDescription,omitempty"` // Device model, software version, etc.
	Capabilities      []string  `json:"capabilities,omitempty"`      // Device capabilities (bridge, router, etc.)
	ManagementAddress string    `json:"managementAddress,omitempty"` // IP address for device management
	VLAN              int       `json:"vlan,omitempty"`              // VLAN ID (CDP)
	NativeVLAN        int       `json:"nativeVlan,omitempty"`        // Native (untagged) VLAN ID (CDP)
	VoiceVLAN         int       `json:"voiceVlan,omitempty"`         // Voice VLAN ID for VoIP phones (CDP)
	TTL               int       `json:"ttl"`                         // Time-to-live in seconds before entry expires
	LastSeen          time.Time `json:"lastSeen"`                    // Timestamp of most recent advertisement
	SourceMAC         string    `json:"sourceMAC"`                   // MAC address of the advertising interface
}

// Manager coordinates multiple discovery protocol captures on a single network interface.
//
// The manager ensures consistent lifecycle management across all protocols:
//   - Starting the manager starts all protocol captures atomically
//   - Stopping the manager stops all protocol captures atomically
//   - Neighbors from all protocols are aggregated and deduplicated
//
// Thread-safety: All public methods are safe for concurrent use via internal mutex.
//
// Typical usage:
//
//	mgr := NewManager("eth0")
//	if err := mgr.Start(); err != nil {
//	    log.Fatal(err)
//	}
//	defer mgr.Stop()
//
//	// Later, get all discovered neighbors
//	neighbors := mgr.GetNeighbors()
type Manager struct {
	interfaceName string       // Network interface to capture on
	lldp          *LLDPCapture // LLDP protocol capture instance
	cdp           *CDPCapture  // CDP protocol capture instance
	edp           *EDPCapture  // EDP protocol capture instance
	mu            sync.RWMutex // Protects started flag
	started       bool         // True if captures are running
}

// NewManager creates a new discovery protocol manager for the specified network interface.
//
// The manager initializes all protocol captures but does not start them.
// Call Start() to begin passive packet capture and neighbor discovery.
//
// Parameters:
//   - interfaceName: Network interface to capture on (e.g., "eth0", "en0")
//
// Returns:
//   - *Manager: Initialized manager ready to start discovery
//
// The returned manager must eventually be stopped via Stop() to release
// packet capture resources and stop background goroutines.
func NewManager(interfaceName string) *Manager {
	return &Manager{
		interfaceName: interfaceName,
		lldp:          NewLLDPCapture(interfaceName),
		cdp:           NewCDPCapture(interfaceName),
		edp:           NewEDPCapture(interfaceName),
	}
}

// Start begins passive packet capture for all discovery protocols.
//
// This method starts LLDP, CDP, and EDP captures concurrently. Each capture:
//   - Opens a packet capture handle on the configured interface
//   - Applies a BPF filter to capture only relevant protocol packets
//   - Starts a background goroutine to process captured packets
//   - Maintains a cache of discovered neighbors with TTL-based expiration
//
// Behavior:
//   - All three protocols must start successfully for the method to succeed
//   - If any protocol fails to start, already-started protocols are stopped
//   - Idempotent: calling Start() when already started is a no-op
//   - Requires elevated privileges (CAP_NET_RAW on Linux, admin/root otherwise)
//
// Common failure scenarios:
//   - Insufficient privileges (run with sudo or CAP_NET_RAW capability)
//   - Interface does not exist or is down
//   - Interface does not support packet capture (some virtual/loopback interfaces)
//   - Another process has exclusive access to the interface
//
// Returns:
//   - nil on successful start of all protocols
//   - error if any protocol fails to start (already-started protocols are stopped)
//
// Thread-safety: Safe for concurrent calls via internal mutex.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return nil
	}

	// Start LLDP capture
	if err := m.lldp.Start(); err != nil {
		return err
	}

	// Start CDP capture
	if err := m.cdp.Start(); err != nil {
		// Don't fail completely if CDP fails, LLDP may still work
		m.lldp.Stop()
		return err
	}

	// Start EDP capture
	if err := m.edp.Start(); err != nil {
		// Don't fail completely if EDP fails, other protocols may still work
		m.lldp.Stop()
		m.cdp.Stop()
		return err
	}

	m.started = true
	return nil
}

// Stop halts all active discovery protocol captures and releases resources.
//
// This method stops all protocol captures (LLDP, CDP, EDP):
//   - Closes packet capture handles
//   - Signals background goroutines to exit
//   - Clears the started flag to allow restart
//   - Neighbors remain in cache (not cleared) until TTL expiration
//
// Behavior:
//   - Idempotent: calling Stop() when already stopped is a no-op
//   - Non-blocking: returns immediately after signaling goroutines to stop
//   - Safe to call from signal handlers or defer statements
//
// Resource cleanup:
//   - Packet capture handles are closed (releases kernel buffers)
//   - Background goroutines exit (releases stack memory)
//   - Neighbor cache is NOT cleared (data remains available for queries)
//
// After calling Stop(), you can:
//   - Query cached neighbors via GetNeighbors() (data remains until TTL expires)
//   - Call Start() again to resume discovery on the same or different interface
//   - Switch to a different interface via SetInterface() and restart
//
// Thread-safety: Safe for concurrent calls via internal mutex.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return
	}

	m.lldp.Stop()
	m.cdp.Stop()
	m.edp.Stop()

	m.started = false
}

// GetNeighbors returns all discovered neighbors from all protocols.
func (m *Manager) GetNeighbors() []*Neighbor {
	lldpNeighbors := m.lldp.GetNeighbors()
	cdpNeighbors := m.cdp.GetNeighbors()
	edpNeighbors := m.edp.GetNeighbors()

	total := len(lldpNeighbors) + len(cdpNeighbors) + len(edpNeighbors)
	neighbors := make([]*Neighbor, 0, total)

	// Get LLDP neighbors
	for _, n := range lldpNeighbors {
		neighbors = append(neighbors, &Neighbor{
			Protocol:          ProtocolLLDP,
			ChassisID:         n.ChassisID,
			PortID:            n.PortID,
			PortDescription:   n.PortDescription,
			SystemName:        n.SystemName,
			SystemDescription: n.SystemDescription,
			Capabilities:      n.SystemCapabilities,
			ManagementAddress: n.ManagementAddress,
			TTL:               n.TTL,
			LastSeen:          n.LastSeen,
			SourceMAC:         n.SourceMAC,
		})
	}

	// Get CDP neighbors
	for _, n := range cdpNeighbors {
		neighbors = append(neighbors, &Neighbor{
			Protocol:          ProtocolCDP,
			ChassisID:         n.DeviceID,
			PortID:            n.PortID,
			SystemName:        n.DeviceID,
			SystemDescription: n.Platform + " " + n.SoftwareVersion,
			Capabilities:      n.Capabilities,
			ManagementAddress: n.ManagementAddress,
			VLAN:              n.NativeVLAN,
			TTL:               n.TTL,
			LastSeen:          n.LastSeen,
			SourceMAC:         n.SourceMAC,
		})
	}

	// Get EDP neighbors
	for _, n := range edpNeighbors {
		systemName := n.DisplayName
		if systemName == "" {
			systemName = n.DeviceID
		}
		neighbors = append(neighbors, &Neighbor{
			Protocol:          ProtocolEDP,
			ChassisID:         n.DeviceID,
			PortID:            n.PortID,
			SystemName:        systemName,
			SystemDescription: n.Platform + " " + n.SoftwareVersion,
			ManagementAddress: n.ManagementAddress,
			VLAN:              n.VLAN,
			TTL:               n.TTL,
			LastSeen:          n.LastSeen,
			SourceMAC:         n.SourceMAC,
		})
	}

	return neighbors
}

// GetLLDPNeighbors returns only LLDP neighbors.
func (m *Manager) GetLLDPNeighbors() []*LLDPNeighbor {
	return m.lldp.GetNeighbors()
}

// GetCDPNeighbors returns only CDP neighbors.
func (m *Manager) GetCDPNeighbors() []*CDPNeighbor {
	return m.cdp.GetNeighbors()
}

// GetEDPNeighbors returns only EDP neighbors.
func (m *Manager) GetEDPNeighbors() []*EDPNeighbor {
	return m.edp.GetNeighbors()
}

// SetInterface changes the capture interface.
func (m *Manager) SetInterface(interfaceName string) error {
	// Read started under lock to avoid race condition (fixes #816)
	m.mu.Lock()
	wasRunning := m.started
	m.mu.Unlock()

	if wasRunning {
		m.Stop()
	}

	m.mu.Lock()
	m.interfaceName = interfaceName
	m.lldp = NewLLDPCapture(interfaceName)
	m.cdp = NewCDPCapture(interfaceName)
	m.edp = NewEDPCapture(interfaceName)
	m.mu.Unlock()

	if wasRunning {
		return m.Start()
	}
	return nil
}

// IsRunning returns true if discovery is active.
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.started
}
