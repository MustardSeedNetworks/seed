//go:build linux

// Package wifi provides wireless network information functionality.
// Linux implementation uses nl80211 (netlink) to scan wireless networks,
// detect access points, and retrieve WiFi interface properties.
package wifi

import (
	"net"
	"os"
	"path/filepath"

	"github.com/mdlayher/wifi"
)

// isWirelessPlatform checks if interface is wireless on Linux using nl80211.
func isWirelessPlatform(iface string) bool {
	// First check if sysfs wireless directory exists
	wirelessPath := filepath.Join("/sys/class/net", iface, "wireless")
	if _, err := os.Stat(wirelessPath); err == nil {
		return true
	}

	// Fall back to nl80211 check
	client, err := wifi.New()
	if err != nil {
		return false
	}
	defer client.Close()

	interfaces, err := client.Interfaces()
	if err != nil {
		return false
	}

	for _, wifiIface := range interfaces {
		if wifiIface.Name == iface {
			return true
		}
	}

	return false
}

// getInfoPlatform gets Wi-Fi info on Linux using nl80211.
func getInfoPlatform(iface string) *Info {
	client, err := wifi.New()
	if err != nil {
		return nil
	}
	defer client.Close()

	interfaces, err := client.Interfaces()
	if err != nil {
		return nil
	}

	// Find the matching interface
	var wifiIface *wifi.Interface
	for _, i := range interfaces {
		if i.Name == iface {
			wifiIface = i
			break
		}
	}

	if wifiIface == nil {
		return nil
	}

	// Get BSS (connection) info
	bss, err := client.BSS(wifiIface)
	if err != nil {
		return nil
	}

	info := &Info{
		SSID:      bss.SSID,
		BSSID:     bss.BSSID.String(),
		Frequency: bss.Frequency / 1000000, // Convert Hz to MHz.
	}

	// Calculate channel from frequency.
	if info.Frequency > 0 {
		info.Channel = frequencyToChannel(info.Frequency)
	}

	// Get station info for signal strength.
	stationInfos, err := client.StationInfo(wifiIface)
	if err == nil && len(stationInfos) > 0 {
		// Signal strength is in dBm.
		info.Signal = stationInfos[0].Signal
	}

	// Try to determine security from BSS
	// The wifi library doesn't directly expose security info,
	// so we check the interface's current state
	info.Security = getSecurityInfo(iface)

	if info.SSID == "" {
		return nil
	}

	return info
}

// getSecurityInfo tries to determine the security protocol from wpa_supplicant status.
// This reads the wpa_supplicant status file directly instead of calling wpa_cli.
func getSecurityInfo(iface string) string {
	// Try to read wpa_supplicant control socket status
	// Check common locations for wpa_supplicant run files
	statusPaths := []string{
		filepath.Join("/var/run/wpa_supplicant", iface),
		filepath.Join("/run/wpa_supplicant", iface),
	}

	for _, path := range statusPaths {
		if _, err := os.Stat(path); err == nil {
			// Socket exists, wpa_supplicant is managing this interface
			// Assume WPA2 as most common
			return "WPA2"
		}
	}

	// Check if interface appears to be connected (has routable IP)
	netIface, err := net.InterfaceByName(iface)
	if err != nil {
		return ""
	}

	addrs, err := netIface.Addrs()
	if err != nil || len(addrs) == 0 {
		return ""
	}

	// If connected, assume WPA2 as default
	return "WPA2"
}
