//go:build windows

package wifi

// Windows-specific Wi-Fi implementation using Windows WLAN API.
// Uses wlanapi.dll for Wi-Fi operations including scanning, connecting, and management.
//
// Platform limitations:
//   - Requires Windows WLAN AutoConfig service (WlanSvc) running
//   - Some operations require administrator privileges
//   - Limited signal strength granularity compared to Linux nl80211

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Command timeout for netsh wlan operations.
const netshWlanTimeoutSeconds = 30

// Radio-type-to-band estimate: netsh does not report frequency directly, so
// 802.11a/802.11n-on-a-high-channel is guessed as 5 GHz and everything else
// as 2.4 GHz.
const (
	maxChannel24GHz   = 14
	band5GHzEstimate  = 5000
	band24GHzEstimate = 2400
)

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
func getInfoPlatform(iface string, _ Helper) *Info {
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
	lines := strings.SplitSeq(output, "\n")

	for line := range lines {
		line = strings.TrimSpace(line)

		// Check for interface name
		if strings.HasPrefix(line, "Name") || strings.HasPrefix(line, "名前") {
			parts := strings.SplitN(line, ":", colonSplitParts)
			if len(parts) == colonSplitParts {
				name := strings.TrimSpace(parts[1])
				inTargetInterface = (targetIface == "" || strings.EqualFold(name, targetIface))
			}
			continue
		}

		if !inTargetInterface {
			continue
		}

		switch {
		case strings.HasPrefix(line, "SSID") && !strings.HasPrefix(line, "BSSID"):
			parseInterfaceSSID(line, info)
		case strings.HasPrefix(line, "BSSID"):
			parseInterfaceBSSID(line, info)
		case strings.HasPrefix(line, "Channel"):
			parseInterfaceChannel(line, info)
		case strings.HasPrefix(line, "Signal"):
			parseInterfaceSignal(line, info)
		case strings.HasPrefix(line, "Radio type") || strings.HasPrefix(line, "無線の種類"):
			parseInterfaceRadioType(line, info)
		case strings.HasPrefix(line, "Authentication") || strings.HasPrefix(line, "認証"):
			parseInterfaceAuth(line, info)
		}
	}

	return info
}

// parseInterfaceSSID reads the SSID field of "netsh wlan show interfaces"
// into info.
func parseInterfaceSSID(line string, info *Info) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		info.SSID = strings.TrimSpace(parts[1])
	}
}

// parseInterfaceBSSID reads the BSSID field into info.
func parseInterfaceBSSID(line string, info *Info) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		info.BSSID = strings.TrimSpace(parts[1])
	}
}

// parseInterfaceChannel reads the Channel field into info.
func parseInterfaceChannel(line string, info *Info) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		_, _ = fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &info.Channel) // best-effort; zero value is fine
	}
}

// parseInterfaceSignal reads the Signal field, converting netsh's percentage
// to an approximate dBm reading (100% =~ -30 dBm, 0% =~ -100 dBm).
func parseInterfaceSignal(line string, info *Info) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		var pct int
		_, _ = fmt.Sscanf(strings.TrimSpace(parts[1]), "%d%%", &pct) // best-effort; zero value is fine
		info.Signal = signalDbmFloor + (pct * signalDbmRange / signalPercentScale)
	}
}

// parseInterfaceRadioType reads the Radio type field and estimates the band
// (2.4 vs 5 GHz) from it, since netsh does not report frequency directly.
func parseInterfaceRadioType(line string, info *Info) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) != colonSplitParts {
		return
	}
	radioType := strings.TrimSpace(parts[1])
	if strings.Contains(radioType, "802.11a") ||
		strings.Contains(radioType, "802.11n") && info.Channel > maxChannel24GHz {
		info.Frequency = band5GHzEstimate
	} else {
		info.Frequency = band24GHzEstimate
	}
}

// parseInterfaceAuth reads the Authentication field into info's Security.
func parseInterfaceAuth(line string, info *Info) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) == colonSplitParts {
		info.Security = strings.TrimSpace(parts[1])
	}
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
		profilePath, tmpErr := createTempProfileFile(profileXML)
		if tmpErr != nil {
			return &ConnectionResult{
				Success: false,
				Message: fmt.Sprintf("Failed to prepare profile: %s", tmpErr),
				SSID:    ssid,
			}, nil
		}
		defer os.Remove(profilePath)

		addProfileCmd := exec.CommandContext(ctx, "netsh", "wlan", "add", "profile",
			fmt.Sprintf("filename=%s", profilePath))
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
func getSavedNetworksPlatform(_ Helper) ([]SavedNetwork, error) {
	ctx, cancel := context.WithTimeout(context.Background(), netshWlanTimeoutSeconds*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "netsh", "wlan", "show", "profiles").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}

	var networks []SavedNetwork
	lines := strings.SplitSeq(string(output), "\n")

	for line := range lines {
		line = strings.TrimSpace(line)
		// Look for "All User Profile" or "User Profile" lines
		if strings.Contains(line, "Profile") && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", colonSplitParts)
			if len(parts) == colonSplitParts {
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

// createTempProfileFile writes the profile XML to a real temporary file and
// returns its path so netsh has an actual profile to read, rather than a
// literal "profile.xml" that only existed by coincidence if the working
// directory happened to contain one.
func createTempProfileFile(profileXML string) (string, error) {
	f, err := os.CreateTemp("", "seed-wlan-profile-*.xml")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary profile file: %w", err)
	}
	defer f.Close()

	if _, writeErr := f.WriteString(profileXML); writeErr != nil {
		return "", fmt.Errorf("failed to write temporary profile file: %w", writeErr)
	}

	return f.Name(), nil
}
