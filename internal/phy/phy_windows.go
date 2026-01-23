//go:build windows

// Windows-specific PHY layer implementation.
// Uses WMI and PowerShell for PHY information retrieval.
//
// Platform limitations:
//   - Windows doesn't expose detailed PHY registers through standard APIs
//   - Limited compared to Linux ethtool capabilities
//   - DOM (Digital Optical Monitoring) for SFP/SFP+ modules requires vendor tools
package phy

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Command timeout for PHY operations.
const phyTimeoutSeconds = 15

// PHYInfo contains PHY layer information.
type PHYInfo struct {
	Interface      string   `json:"interface"`
	Speed          string   `json:"speed"`
	Duplex         string   `json:"duplex"`
	AutoNeg        bool     `json:"auto_neg"`
	LinkStatus     bool     `json:"link_status"`
	SupportedModes []string `json:"supported_modes,omitempty"`
	AdvertisedModes []string `json:"advertised_modes,omitempty"`
	PhyAddress     int      `json:"phy_address,omitempty"`
	Driver         string   `json:"driver,omitempty"`
	Firmware       string   `json:"firmware,omitempty"`
}

// GetPHYInfo retrieves PHY layer information for an interface on Windows.
func GetPHYInfo(iface string) (*PHYInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), phyTimeoutSeconds*time.Second)
	defer cancel()

	info := &PHYInfo{
		Interface: iface,
	}

	// Get adapter info via PowerShell
	psCmd := fmt.Sprintf(`Get-NetAdapter -Name '%s' -ErrorAction SilentlyContinue | `+
		`Select-Object LinkSpeed, FullDuplex, MediaConnectionState, DriverName, DriverVersion, `+
		`@{n='SupportedSpeeds';e={($_ | Get-NetAdapterAdvancedProperty -RegistryKeyword '*SpeedDuplex' -ErrorAction SilentlyContinue).ValidDisplayValues -join ','}} | `+
		`ConvertTo-Json`, iface)

	output, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get PHY info: %w", err)
	}

	// Parse JSON output
	parsePHYJson(string(output), info)

	return info, nil
}

// parsePHYJson parses PowerShell JSON output into PHYInfo.
func parsePHYJson(output string, info *PHYInfo) {
	output = strings.TrimSpace(output)
	if output == "" || output == "null" {
		return
	}

	// Simple parsing - look for key fields in JSON
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "LinkSpeed") {
			// Extract value between quotes
			if idx := strings.Index(line, ":"); idx > 0 {
				value := strings.Trim(strings.TrimSpace(line[idx+1:]), `",`)
				info.Speed = value
			}
		} else if strings.Contains(line, "FullDuplex") {
			if strings.Contains(line, "true") || strings.Contains(line, "True") {
				info.Duplex = "full"
			} else {
				info.Duplex = "half"
			}
		} else if strings.Contains(line, "MediaConnectionState") {
			if strings.Contains(line, "Connected") {
				info.LinkStatus = true
			}
		} else if strings.Contains(line, "DriverName") {
			if idx := strings.Index(line, ":"); idx > 0 {
				info.Driver = strings.Trim(strings.TrimSpace(line[idx+1:]), `",`)
			}
		} else if strings.Contains(line, "DriverVersion") {
			if idx := strings.Index(line, ":"); idx > 0 {
				info.Firmware = strings.Trim(strings.TrimSpace(line[idx+1:]), `",`)
			}
		} else if strings.Contains(line, "SupportedSpeeds") {
			if idx := strings.Index(line, ":"); idx > 0 {
				value := strings.Trim(strings.TrimSpace(line[idx+1:]), `",`)
				if value != "" && value != "null" {
					info.SupportedModes = strings.Split(value, ",")
				}
			}
		}
	}

	// Auto-neg is typically enabled by default on Windows
	info.AutoNeg = true
}

// SetPHYSpeed sets the PHY link speed on Windows.
// This is limited to what Windows exposes through adapter properties.
func SetPHYSpeed(iface string, speed string) error {
	ctx, cancel := context.WithTimeout(context.Background(), phyTimeoutSeconds*time.Second)
	defer cancel()

	// Map common speed strings to Windows property values
	speedValue := mapSpeedToWindows(speed)
	if speedValue == "" {
		return fmt.Errorf("unsupported speed: %s", speed)
	}

	// Use PowerShell to set adapter speed
	psCmd := fmt.Sprintf(`Set-NetAdapterAdvancedProperty -Name '%s' -RegistryKeyword '*SpeedDuplex' -RegistryValue %s`, iface, speedValue)

	output, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", psCmd).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set speed: %s", strings.TrimSpace(string(output)))
	}

	return nil
}

// mapSpeedToWindows maps speed strings to Windows registry values.
func mapSpeedToWindows(speed string) string {
	// Common *SpeedDuplex values:
	// 0 = Auto, 1 = 10 Half, 2 = 10 Full, 3 = 100 Half, 4 = 100 Full
	// 5 = 1000 Half, 6 = 1000 Full, etc.
	speedMap := map[string]string{
		"auto":     "0",
		"10half":   "1",
		"10full":   "2",
		"100half":  "3",
		"100full":  "4",
		"1000half": "5",
		"1000full": "6",
		"1g":       "6",
		"2.5g":     "7",
		"5g":       "8",
		"10g":      "9",
	}

	return speedMap[strings.ToLower(strings.ReplaceAll(speed, " ", ""))]
}

// GetDOMInfo retrieves Digital Optical Monitoring info for SFP/SFP+ modules.
// This is not available through standard Windows APIs.
func GetDOMInfo(iface string) error {
	return fmt.Errorf("DOM (Digital Optical Monitoring) on Windows is not available through standard APIs. " +
		"SFP/SFP+ monitoring requires vendor-specific tools:\n" +
		"  - Intel NICs: Intel PROSet\n" +
		"  - Broadcom NICs: Broadcom Advanced Control Suite\n" +
		"  - Mellanox NICs: Mellanox WinOF\n" +
		"See HARDWARE.md for compatible hardware recommendations")
}

// getPoEStatus detects PoE power status on Windows.
// Windows doesn't expose PoE information through standard APIs.
func getPoEStatus(_ string) *PoEStatus {
	// PoE detection on Windows requires vendor-specific tools
	// Most NICs don't expose this information to the OS
	return &PoEStatus{
		Detected: false,
	}
}

// getSFPInfo reads SFP module info on Windows.
// Windows doesn't expose SFP/DDM information through standard APIs.
func getSFPInfo(_ string) *SFPInfo {
	// SFP/DDM information requires vendor-specific tools on Windows:
	// - Intel: Intel PROSet
	// - Mellanox: WinOF
	// - Broadcom: BACS
	return &SFPInfo{
		Present:    false,
		DDMSupport: false,
	}
}
