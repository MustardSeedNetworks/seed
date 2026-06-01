//go:build darwin

package wifi

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Darwin platform constants.
const (
	infoTimeoutSeconds       = 5  // Timeout for WiFi info retrieval operations
	connectTimeoutSeconds    = 30 // Timeout for WiFi connection operations
	disconnectTimeoutSeconds = 10 // Timeout for WiFi disconnect operations
	keyValuePairCount        = 2  // Expected number of parts when splitting key:value pairs
)

// isWirelessPlatform checks if interface is wireless on macOS.
// macOS requires exec-based approach as there's no nl80211 equivalent.
func isWirelessPlatform(iface string) bool {
	// On macOS, Wi-Fi interface is typically en0 or starts with en
	// We can use networksetup to check
	ctx, cancel := context.WithTimeout(context.Background(), infoTimeoutSeconds*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "networksetup", "-listallhardwareports")
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

// getInfoPlatform gets Wi-Fi info on macOS.
// macOS requires exec-based approach using airport utility.
func getInfoPlatform(_ string) *Info {
	// Use airport command for Wi-Fi info
	airportPath := "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"
	ctx, cancel := context.WithTimeout(context.Background(), infoTimeoutSeconds*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, airportPath, "-I")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	info := &Info{}
	lines := strings.SplitSeq(string(output), "\n")

	for line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", keyValuePairCount)
		if len(parts) != keyValuePairCount {
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
			if sig, parseErr := strconv.Atoi(value); parseErr == nil {
				info.Signal = sig
			}
		case "channel":
			// Format can be "6" or "6,1" (for 80MHz channels)
			chParts := strings.Split(value, ",")
			if ch, chErr := strconv.Atoi(chParts[0]); chErr == nil {
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

// connectPlatform connects to a WiFi network on macOS.
// Uses networksetup command for connection management.
func connectPlatform(iface, ssid, password string) (*ConnectionResult, error) {
	// macOS uses networksetup to connect to networks
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeoutSeconds*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if password != "" {
		cmd = exec.CommandContext(ctx, "networksetup", "-setairportnetwork", iface, ssid, password)
	} else {
		cmd = exec.CommandContext(ctx, "networksetup", "-setairportnetwork", iface, ssid)
	}

	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	if err != nil {
		if strings.Contains(outputStr, "Could not find network") {
			return &ConnectionResult{
				Success: false,
				Message: "Network not found. Make sure the network is in range.",
				SSID:    ssid,
			}, nil
		}
		return &ConnectionResult{
			Success: false,
			Message: outputStr,
			SSID:    ssid,
		}, nil
	}

	// Empty output usually means success on macOS
	if outputStr == "" {
		return &ConnectionResult{
			Success: true,
			Message: "Successfully connected to " + ssid,
			SSID:    ssid,
		}, nil
	}

	return &ConnectionResult{
		Success: false,
		Message: outputStr,
		SSID:    ssid,
	}, nil
}

// disconnectPlatform disconnects from WiFi on macOS.
func disconnectPlatform(iface string) (*ConnectionResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), disconnectTimeoutSeconds*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "networksetup", "-setairportpower", iface, "off")
	if _, err := cmd.CombinedOutput(); err != nil {
		return &ConnectionResult{
			Success: false,
			Message: "Failed to disconnect",
		}, nil
	}

	// Turn WiFi back on (just disconnected from network)
	cmd = exec.CommandContext(ctx, "networksetup", "-setairportpower", iface, "on")
	_, _ = cmd.CombinedOutput()

	return &ConnectionResult{
		Success: true,
		Message: "Successfully disconnected",
	}, nil
}

// getSavedNetworksPlatform returns saved WiFi networks on macOS.
func getSavedNetworksPlatform() ([]SavedNetwork, error) {
	// macOS doesn't have an easy way to list saved networks via command line
	// Would need to use CoreWLAN framework or parse preference files
	return []SavedNetwork{}, nil
}

// forgetNetworkPlatform removes a saved WiFi network on macOS.
func forgetNetworkPlatform(ssid string) error {
	ctx, cancel := context.WithTimeout(context.Background(), disconnectTimeoutSeconds*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "networksetup", "-removepreferredwirelessnetwork", "en0", ssid)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to forget network: %s", strings.TrimSpace(string(output)))
	}
	return nil
}
