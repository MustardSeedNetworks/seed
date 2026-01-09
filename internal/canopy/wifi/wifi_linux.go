//go:build linux

package wifi

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

// frequencyToChannel converts a frequency in MHz to Wi-Fi channel.
func frequencyToChannel(freq int) int {
	// 2.4 GHz band
	if freq >= 2412 && freq <= 2472 {
		return (freq - freq24GHzBaseOffset) / channelSpacingMHz
	}
	if freq == freq24GHzChannel14 {
		return channel14
	}

	// 5 GHz band
	if freq >= 5180 && freq <= 5825 {
		return (freq - freq5GHzBaseOffset) / channelSpacingMHz
	}

	// 6 GHz band
	if freq >= 5955 && freq <= 7115 {
		return (freq - freq6GHzBaseOffset) / channelSpacingMHz
	}

	return 0
}

// connectPlatform connects to a WiFi network on Linux using nmcli.
func connectPlatform(iface, ssid, password string) (*ConnectionResult, error) {
	var cmd *exec.Cmd

	if password != "" {
		// Connect with password - creates new connection profile
		//nolint:gosec // ssid and password are user-provided, iface is validated
		cmd = exec.Command("nmcli", "device", "wifi", "connect", ssid,
			"password", password, "ifname", iface)
	} else {
		// Try to connect using saved connection
		//nolint:gosec // ssid is user-provided, iface is validated
		cmd = exec.Command("nmcli", "device", "wifi", "connect", ssid, "ifname", iface)
	}

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		// Parse common error messages
		if strings.Contains(outputStr, "Secrets were required") {
			return &ConnectionResult{
				Success: false,
				Message: "Password required for this network",
				SSID:    ssid,
			}, nil
		}
		if strings.Contains(outputStr, "No network with SSID") {
			return &ConnectionResult{
				Success: false,
				Message: "Network not found. Make sure the network is in range.",
				SSID:    ssid,
			}, nil
		}
		if strings.Contains(outputStr, "Error") {
			return &ConnectionResult{
				Success: false,
				Message: outputStr,
				SSID:    ssid,
			}, nil
		}
		return &ConnectionResult{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %s", outputStr),
			SSID:    ssid,
		}, nil
	}

	return &ConnectionResult{
		Success: true,
		Message: fmt.Sprintf("Successfully connected to %s", ssid),
		SSID:    ssid,
	}, nil
}

// disconnectPlatform disconnects from WiFi on Linux using nmcli.
func disconnectPlatform(iface string) (*ConnectionResult, error) {
	//nolint:gosec // iface is validated by caller
	cmd := exec.Command("nmcli", "device", "disconnect", iface)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		return &ConnectionResult{
			Success: false,
			Message: fmt.Sprintf("Disconnect failed: %s", outputStr),
		}, nil
	}

	return &ConnectionResult{
		Success: true,
		Message: "Successfully disconnected",
	}, nil
}

// getSavedNetworksPlatform returns saved WiFi networks on Linux using nmcli.
func getSavedNetworksPlatform() ([]SavedNetwork, error) {
	// List saved WiFi connections
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,UUID,TYPE,DEVICE", "connection", "show")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list saved networks: %w", err)
	}

	var networks []SavedNetwork
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 3 && parts[2] == "802-11-wireless" {
			network := SavedNetwork{
				SSID: parts[0],
				UUID: parts[1],
				Type: "wifi",
			}
			if len(parts) >= 4 {
				network.Device = parts[3]
			}
			networks = append(networks, network)
		}
	}

	return networks, nil
}

// forgetNetworkPlatform removes a saved WiFi network on Linux using nmcli.
func forgetNetworkPlatform(ssid string) error {
	//nolint:gosec // ssid is user-provided
	cmd := exec.Command("nmcli", "connection", "delete", ssid)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to forget network: %s", strings.TrimSpace(string(output)))
	}
	return nil
}
