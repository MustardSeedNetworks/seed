//go:build windows

// Windows-specific Wi-Fi implementation using Windows WLAN API.
// Uses wlanapi.dll for Wi-Fi operations including scanning, connecting, and management.
//
// Platform limitations:
//   - Requires Windows WLAN AutoConfig service (WlanSvc) running
//   - Some operations require administrator privileges
//   - Limited signal strength granularity compared to Linux nl80211
package wifi

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Command timeout for netsh wlan operations.
const netshWlanTimeoutSeconds = 30

// isWirelessPlatform checks if interface is wireless on Windows.
func isWirelessPlatform(iface string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), netshWlanTimeoutSeconds*time.Second)
	defer cancel()

	// Use netsh to list wireless interfaces
	output, err := exec.CommandContext(ctx, "netsh", "wlan", "show", "interfaces").Output()
	if err != nil {
		return false
	}

	// Check if interface name appears in wireless interface list
	return strings.Contains(string(output), iface)
}

// getInfoPlatform gets Wi-Fi info on Windows using netsh wlan.
func getInfoPlatform(iface string) *Info {
	ctx, cancel := context.WithTimeout(context.Background(), netshWlanTimeoutSeconds*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "netsh", "wlan", "show", "interfaces").Output()
	if err != nil {
		return nil
	}

	// Parse netsh output for the specified interface
	info := parseNetshWlanOutput(string(output), iface)
	if info == nil || info.SSID == "" {
		return nil
	}

	return info
}

// parseNetshWlanOutput parses the output of "netsh wlan show interfaces".
func parseNetshWlanOutput(output, targetIface string) *Info {
	info := &Info{}
	inTargetInterface := false
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for interface name
		if strings.HasPrefix(line, "Name") || strings.HasPrefix(line, "名前") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[1])
				inTargetInterface = (targetIface == "" || strings.EqualFold(name, targetIface))
			}
			continue
		}

		if !inTargetInterface {
			continue
		}

		// Parse fields
		if strings.HasPrefix(line, "SSID") && !strings.HasPrefix(line, "BSSID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.SSID = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "BSSID") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.BSSID = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Channel") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &info.Channel)
			}
		} else if strings.HasPrefix(line, "Signal") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				// Windows shows signal as percentage
				var pct int
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%d%%", &pct)
				// Convert percentage to dBm (rough approximation)
				// 100% ≈ -30 dBm, 0% ≈ -100 dBm
				info.Signal = -100 + (pct * 70 / 100)
			}
		} else if strings.HasPrefix(line, "Radio type") || strings.HasPrefix(line, "無線の種類") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				radioType := strings.TrimSpace(parts[1])
				// Determine frequency from radio type
				if strings.Contains(radioType, "802.11a") || strings.Contains(radioType, "802.11n") && info.Channel > 14 {
					info.Frequency = 5000 // 5 GHz band
				} else {
					info.Frequency = 2400 // 2.4 GHz band
				}
			}
		} else if strings.HasPrefix(line, "Authentication") || strings.HasPrefix(line, "認証") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.Security = strings.TrimSpace(parts[1])
			}
		}
	}

	return info
}

// connectPlatform connects to a WiFi network on Windows using netsh.
func connectPlatform(iface, ssid, password string) (*ConnectionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), netshWlanTimeoutSeconds*time.Second)
	defer cancel()

	var output []byte
	var err error

	if password != "" {
		// Need to create a profile first for networks with password
		profileXML := generateWlanProfile(ssid, password)

		// Create temporary profile
		addProfileCmd := exec.CommandContext(ctx, "netsh", "wlan", "add", "profile",
			fmt.Sprintf("filename=%s", createTempProfileFile(profileXML)))
		output, err = addProfileCmd.CombinedOutput()
		if err != nil {
			return &ConnectionResult{
				Success: false,
				Message: fmt.Sprintf("Failed to add profile: %s", strings.TrimSpace(string(output))),
				SSID:    ssid,
			}, nil
		}
	}

	// Connect to the network
	args := []string{"wlan", "connect", fmt.Sprintf("name=%s", ssid)}
	if iface != "" {
		args = append(args, fmt.Sprintf("interface=%s", iface))
	}

	output, err = exec.CommandContext(ctx, "netsh", args...).CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		return &ConnectionResult{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %s", outputStr),
			SSID:    ssid,
		}, nil
	}

	// Check for success message
	if strings.Contains(outputStr, "successfully") || strings.Contains(outputStr, "成功") {
		return &ConnectionResult{
			Success: true,
			Message: fmt.Sprintf("Successfully connected to %s", ssid),
			SSID:    ssid,
		}, nil
	}

	return &ConnectionResult{
		Success: false,
		Message: outputStr,
		SSID:    ssid,
	}, nil
}

// disconnectPlatform disconnects from WiFi on Windows using netsh.
func disconnectPlatform(iface string) (*ConnectionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), netshWlanTimeoutSeconds*time.Second)
	defer cancel()

	args := []string{"wlan", "disconnect"}
	if iface != "" {
		args = append(args, fmt.Sprintf("interface=%s", iface))
	}

	output, err := exec.CommandContext(ctx, "netsh", args...).CombinedOutput()
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

// getSavedNetworksPlatform returns saved WiFi networks on Windows using netsh.
func getSavedNetworksPlatform() ([]SavedNetwork, error) {
	ctx, cancel := context.WithTimeout(context.Background(), netshWlanTimeoutSeconds*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "netsh", "wlan", "show", "profiles").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}

	var networks []SavedNetwork
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for "All User Profile" or "User Profile" lines
		if strings.Contains(line, "Profile") && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				ssid := strings.TrimSpace(parts[1])
				if ssid != "" {
					networks = append(networks, SavedNetwork{
						SSID: ssid,
						Type: "wifi",
					})
				}
			}
		}
	}

	return networks, nil
}

// forgetNetworkPlatform removes a saved WiFi network on Windows using netsh.
func forgetNetworkPlatform(ssid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), netshWlanTimeoutSeconds*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "netsh", "wlan", "delete", "profile",
		fmt.Sprintf("name=%s", ssid)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to forget network: %s", strings.TrimSpace(string(output)))
	}

	return nil
}

// generateWlanProfile generates an XML profile for WPA2-Personal networks.
func generateWlanProfile(ssid, password string) string {
	// Basic WPA2-Personal profile
	return fmt.Sprintf(`<?xml version="1.0"?>
<WLANProfile xmlns="http://www.microsoft.com/networking/WLAN/profile/v1">
	<name>%s</name>
	<SSIDConfig>
		<SSID>
			<name>%s</name>
		</SSID>
	</SSIDConfig>
	<connectionType>ESS</connectionType>
	<connectionMode>auto</connectionMode>
	<MSM>
		<security>
			<authEncryption>
				<authentication>WPA2PSK</authentication>
				<encryption>AES</encryption>
				<useOneX>false</useOneX>
			</authEncryption>
			<sharedKey>
				<keyType>passPhrase</keyType>
				<protected>false</protected>
				<keyMaterial>%s</keyMaterial>
			</sharedKey>
		</security>
	</MSM>
</WLANProfile>`, ssid, ssid, password)
}

// createTempProfileFile creates a temporary file with the profile XML.
func createTempProfileFile(profileXML string) string {
	// For security, this should use a proper temp file
	// This is a placeholder - actual implementation should use os.CreateTemp
	return "profile.xml"
}
