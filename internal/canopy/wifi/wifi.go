package wifi

import (
	"strings"
	"sync"
)

// WiFi frequency conversion constants.
const (
	// 2.4 GHz band constants.
	freq24GHzBaseOffset = 2407 // Base frequency offset for 2.4 GHz channels 1-13
	freq24GHzChannel14  = 2484 // Frequency for channel 14 (Japan only)
	channel14           = 14   // Special channel 14 number

	// 5 GHz band constants.
	freq5GHzBaseOffset = 5000 // Base frequency offset for 5 GHz channels

	// 6 GHz band constants.
	freq6GHzBaseOffset = 5950 // Base frequency offset for 6 GHz channels

	// Channel frequency spacing.
	channelSpacingMHz = 5 // Standard WiFi channel spacing in MHz
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

	return getInfoPlatform(iface)
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

// channelToFrequency converts a Wi-Fi channel to frequency in MHz.
func channelToFrequency(channel int) int {
	// 2.4 GHz band
	if channel >= 1 && channel <= 13 {
		return freq24GHzBaseOffset + (channel * channelSpacingMHz)
	}
	if channel == channel14 {
		return freq24GHzChannel14
	}

	// 5 GHz band
	if channel >= 36 && channel <= 64 {
		return freq5GHzBaseOffset + (channel * channelSpacingMHz)
	}
	if channel >= 100 && channel <= 144 {
		return freq5GHzBaseOffset + (channel * channelSpacingMHz)
	}
	if channel >= 149 && channel <= 165 {
		return freq5GHzBaseOffset + (channel * channelSpacingMHz)
	}

	// 6 GHz band
	if channel >= 1 && channel <= 233 {
		return freq6GHzBaseOffset + (channel * channelSpacingMHz)
	}

	return 0
}
