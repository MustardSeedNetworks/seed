package wifi

import (
	"strings"
	"sync"
)

// Info contains wireless network information.
type Info struct {
	SSID      string `json:"ssid"`
	BSSID     string `json:"bssid"`
	Signal    int    `json:"signal"` // dBm
	Channel   int    `json:"channel"`
	Frequency int    `json:"frequency"` // MHz
	Security  string `json:"security"`
}

// Manager handles wireless network information retrieval.
type Manager struct {
	interfaceName string
	mu            sync.RWMutex
	helper        Helper // nil except on macOS
}

// NewManager creates a new Wi-Fi manager.
func NewManager(interfaceName string) *Manager {
	return &Manager{
		interfaceName: interfaceName,
	}
}

// SetInterface updates the interface to monitor.
func (m *Manager) SetInterface(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.interfaceName = name
}

// IsWireless checks if the current interface is wireless.
func (m *Manager) IsWireless() bool {
	m.mu.RLock()
	iface := m.interfaceName
	m.mu.RUnlock()

	return isWirelessPlatform(iface)
}

// GetInfo returns current wireless network information.
func (m *Manager) GetInfo() *Info {
	m.mu.RLock()
	iface := m.interfaceName
	m.mu.RUnlock()

	return getInfoPlatform(iface, m.currentHelper())
}

// mapSecurityType maps security protocol to display string.
func mapSecurityType(secType string) string {
	secType = strings.ToUpper(secType)
	switch {
	case strings.Contains(secType, "SAE"):
		return "WPA3"
	case strings.Contains(secType, "WPA3"):
		return "WPA3"
	case strings.Contains(secType, "WPA2"):
		return "WPA2"
	case strings.Contains(secType, "WPA"):
		return "WPA"
	case strings.Contains(secType, "WEP"):
		return "WEP"
	case strings.Contains(secType, "OPEN"):
		return "Open"
	case strings.Contains(secType, "NONE"):
		return "Open"
	default:
		return secType
	}
}

// ConnectionResult represents the result of a WiFi connection attempt.
type ConnectionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	SSID    string `json:"ssid,omitempty"`
}

// SavedNetwork represents a saved/known WiFi network.
type SavedNetwork struct {
	SSID     string `json:"ssid"`
	UUID     string `json:"uuid,omitempty"`
	Type     string `json:"type,omitempty"`     // e.g., "wifi"
	Device   string `json:"device,omitempty"`   // e.g., "wls34u2"
	Security string `json:"security,omitempty"` // e.g., "WPA2"
}

// Connect attempts to connect to a WiFi network.
// If password is empty, it tries to use a saved connection.
func (m *Manager) Connect(ssid, password string) (*ConnectionResult, error) {
	m.mu.RLock()
	iface := m.interfaceName
	m.mu.RUnlock()

	return connectPlatform(iface, ssid, password)
}

// Disconnect disconnects from the current WiFi network.
func (m *Manager) Disconnect() (*ConnectionResult, error) {
	m.mu.RLock()
	iface := m.interfaceName
	m.mu.RUnlock()

	return disconnectPlatform(iface)
}

// GetSavedNetworks returns a list of saved/known WiFi networks.
func (m *Manager) GetSavedNetworks() ([]SavedNetwork, error) {
	return getSavedNetworksPlatform(m.currentHelper())
}

// ForgetNetwork removes a saved WiFi network.
func (m *Manager) ForgetNetwork(ssid string) error {
	return forgetNetworkPlatform(ssid)
}
