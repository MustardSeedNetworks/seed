package discovery

import (
	"sync"
	"time"
)

// Protocol represents a discovery protocol type.
type Protocol string

const (
	ProtocolLLDP Protocol = "lldp"
	ProtocolCDP  Protocol = "cdp"
	ProtocolEDP  Protocol = "edp"
)

// Neighbor represents a discovered network neighbor.
type Neighbor struct {
	Protocol          Protocol  `json:"protocol"`
	ChassisID         string    `json:"chassisId"`
	PortID            string    `json:"portId"`
	PortDescription   string    `json:"portDescription,omitempty"`
	SystemName        string    `json:"systemName,omitempty"`
	SystemDescription string    `json:"systemDescription,omitempty"`
	Capabilities      []string  `json:"capabilities,omitempty"`
	ManagementAddress string    `json:"managementAddress,omitempty"`
	VLAN              int       `json:"vlan,omitempty"`
	NativeVLAN        int       `json:"nativeVlan,omitempty"`
	VoiceVLAN         int       `json:"voiceVlan,omitempty"`
	TTL               int       `json:"ttl"`
	LastSeen          time.Time `json:"lastSeen"`
	SourceMAC         string    `json:"sourceMAC"`
}

// Manager coordinates discovery protocol captures.
type Manager struct {
	interfaceName string
	lldp          *LLDPCapture
	cdp           *CDPCapture
	edp           *EDPCapture
	mu            sync.RWMutex
	started       bool
}

// NewManager creates a new discovery manager.
func NewManager(interfaceName string) *Manager {
	return &Manager{
		interfaceName: interfaceName,
		lldp:          NewLLDPCapture(interfaceName),
		cdp:           NewCDPCapture(interfaceName),
		edp:           NewEDPCapture(interfaceName),
	}
}

// Start begins all discovery captures.
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

// Stop stops all discovery captures.
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
	wasRunning := m.started

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
