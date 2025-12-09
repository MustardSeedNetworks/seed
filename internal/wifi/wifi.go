// Package wifi provides wireless network information functionality.
package wifi

import (
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
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

	switch runtime.GOOS {
	case "darwin":
		return isWirelessDarwin(iface)
	case "linux":
		return isWirelessLinux(iface)
	default:
		return false
	}
}

// GetInfo returns current wireless network information.
func (m *Manager) GetInfo() *Info {
	m.mu.RLock()
	iface := m.interfaceName
	m.mu.RUnlock()

	switch runtime.GOOS {
	case "darwin":
		return getInfoDarwin(iface)
	case "linux":
		return getInfoLinux(iface)
	default:
		return nil
	}
}

// isWirelessDarwin checks if interface is wireless on macOS.
func isWirelessDarwin(iface string) bool {
	// On macOS, Wi-Fi interface is typically en0 or starts with en
	// We can use networksetup to check
	cmd := exec.Command("networksetup", "-listallhardwareports")
	output, err := cmd.Output()
	if err != nil {
		return strings.HasPrefix(iface, "en")
	}

	lines := strings.Split(string(output), "\n")
	foundWiFi := false
	for _, line := range lines {
		if strings.Contains(line, "Wi-Fi") {
			foundWiFi = true
		}
		if foundWiFi && strings.Contains(line, "Device:") {
			device := strings.TrimPrefix(line, "Device: ")
			device = strings.TrimSpace(device)
			if device == iface {
				return true
			}
			foundWiFi = false
		}
	}
	return false
}

// isWirelessLinux checks if interface is wireless on Linux.
func isWirelessLinux(iface string) bool {
	// Check if interface has wireless extensions
	cmd := exec.Command("iw", "dev", iface, "info")
	err := cmd.Run()
	return err == nil
}

// getInfoDarwin gets Wi-Fi info on macOS.
func getInfoDarwin(iface string) *Info {
	// Use airport command for Wi-Fi info
	airportPath := "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"
	cmd := exec.Command(airportPath, "-I")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	info := &Info{}
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "SSID":
			info.SSID = value
		case "BSSID":
			info.BSSID = value
		case "agrCtlRSSI":
			if sig, err := strconv.Atoi(value); err == nil {
				info.Signal = sig
			}
		case "channel":
			// Format can be "6" or "6,1" (for 80MHz channels)
			parts := strings.Split(value, ",")
			if ch, err := strconv.Atoi(parts[0]); err == nil {
				info.Channel = ch
			}
		case "link auth":
			info.Security = mapSecurityType(value)
		}
	}

	// Calculate frequency from channel if we got a channel
	if info.Channel > 0 {
		info.Frequency = channelToFrequency(info.Channel)
	}

	// If no security info, try to get it another way
	if info.Security == "" && info.SSID != "" {
		info.Security = "WPA2" // Default assumption
	}

	if info.SSID == "" {
		return nil
	}

	return info
}

// getInfoLinux gets Wi-Fi info on Linux.
func getInfoLinux(iface string) *Info {
	info := &Info{}

	// Get connection info using iw
	cmd := exec.Command("iw", "dev", iface, "link")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "SSID:"):
			info.SSID = strings.TrimPrefix(line, "SSID: ")
		case strings.HasPrefix(line, "signal:"):
			// Format: "signal: -65 dBm"
			re := regexp.MustCompile(`-?\d+`)
			if match := re.FindString(line); match != "" {
				if sig, err := strconv.Atoi(match); err == nil {
					info.Signal = sig
				}
			}
		case strings.HasPrefix(line, "freq:"):
			// Format: "freq: 5180"
			re := regexp.MustCompile(`\d+`)
			if match := re.FindString(line); match != "" {
				if freq, err := strconv.Atoi(match); err == nil {
					info.Frequency = freq
				}
			}
		case strings.Contains(line, "Connected to"):
			// Format: "Connected to 00:11:22:33:44:55"
			re := regexp.MustCompile(`([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}`)
			if match := re.FindString(line); match != "" {
				info.BSSID = match
			}
		}
	}

	// Get channel from frequency
	if info.Frequency > 0 {
		info.Channel = frequencyToChannel(info.Frequency)
	}

	// Get security info using iwconfig
	cmd = exec.Command("iwconfig", iface)
	output, err = cmd.Output()
	if err == nil {
		outStr := string(output)
		if strings.Contains(outStr, "Encryption key:on") {
			info.Security = "WPA2" // Simplified
		} else if strings.Contains(outStr, "Encryption key:off") {
			info.Security = "Open"
		}
	}

	// Try wpa_cli for better security info
	cmd = exec.Command("wpa_cli", "-i", iface, "status")
	output, err = cmd.Output()
	if err == nil {
		lines = strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "key_mgmt=") {
				keyMgmt := strings.TrimPrefix(line, "key_mgmt=")
				info.Security = mapSecurityType(keyMgmt)
			}
		}
	}

	if info.SSID == "" {
		return nil
	}

	return info
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
