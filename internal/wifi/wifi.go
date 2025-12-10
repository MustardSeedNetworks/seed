// Package wifi provides wireless network information functionality.
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
		return 2407 + (channel * 5)
	}
	if channel == 14 {
		return 2484
	}

	// 5 GHz band
	if channel >= 36 && channel <= 64 {
		return 5000 + (channel * 5)
	}
	if channel >= 100 && channel <= 144 {
		return 5000 + (channel * 5)
	}
	if channel >= 149 && channel <= 165 {
		return 5000 + (channel * 5)
	}

	// 6 GHz band
	if channel >= 1 && channel <= 233 {
		return 5950 + (channel * 5)
	}

	return 0
}

// frequencyToChannel converts a frequency in MHz to Wi-Fi channel.
func frequencyToChannel(freq int) int {
	// 2.4 GHz band
	if freq >= 2412 && freq <= 2472 {
		return (freq - 2407) / 5
	}
	if freq == 2484 {
		return 14
	}

	// 5 GHz band
	if freq >= 5180 && freq <= 5825 {
		return (freq - 5000) / 5
	}

	// 6 GHz band
	if freq >= 5955 && freq <= 7115 {
		return (freq - 5950) / 5
	}

	return 0
}
