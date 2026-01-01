//go:build darwin

// Package wifi provides wireless network information functionality.
// macOS implementation uses airport command-line utility to scan wireless networks
// and retrieve WiFi interface information from the System Configuration framework.
package wifi

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// isWirelessPlatform checks if interface is wireless on macOS.
// macOS requires exec-based approach as there's no nl80211 equivalent.
func isWirelessPlatform(iface string) bool {
	// On macOS, Wi-Fi interface is typically en0 or starts with en
	// We can use networksetup to check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
