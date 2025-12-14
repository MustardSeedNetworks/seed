//go:build linux

package wifi

import (
	"fmt"

	"github.com/mdlayher/wifi"
)

// scanPlatform performs a WiFi scan on Linux using nl80211.
func scanPlatform(iface string) ([]*ScannedNetwork, error) {
	client, err := wifi.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create wifi client: %w", err)
	}
	defer client.Close()

	// Get interface
	interfaces, err := client.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	var wifiIface *wifi.Interface
	for _, i := range interfaces {
		if i.Name == iface {
			wifiIface = i
			break
		}
	}

	if wifiIface == nil {
		return nil, fmt.Errorf("interface %s not found or not wireless", iface)
	}

	// Trigger scan
	if err := client.Scan(wifiIface); err != nil {
		return nil, fmt.Errorf("failed to trigger scan: %w", err)
	}

	// Get scan results
	bssList, err := client.ScanResults(wifiIface)
	if err != nil {
		return nil, fmt.Errorf("failed to get scan results: %w", err)
	}

	// Convert to ScannedNetwork
	networks := make([]*ScannedNetwork, 0, len(bssList))
	for _, bss := range bssList {
		network := &ScannedNetwork{
			SSID:      bss.SSID,
			BSSID:     bss.BSSID.String(),
			Frequency: int(bss.Frequency / 1000000), // Hz to MHz
		}

		// Calculate channel from frequency
		if network.Frequency > 0 {
			network.Channel = frequencyToChannel(network.Frequency)
		}

		// Signal strength (already in dBm from nl80211)
		network.Signal = int(bss.Signal)

		// Determine security type from capabilities
		network.Security = inferSecurity(bss)

		// Skip networks with empty SSID (hidden networks)
		if network.SSID != "" {
			networks = append(networks, network)
		}
	}

	return networks, nil
}

// inferSecurity infers security type from BSS information.
func inferSecurity(bss *wifi.BSS) string {
	// Check RSN (WPA2/WPA3) information element
	if len(bss.RSN) > 0 {
		// Check for WPA3 (SAE)
		for _, akm := range bss.RSN.AKMs {
			if akm == wifi.AKMSuite(8) { // SAE
				return "WPA3"
			}
		}
		// Otherwise it's WPA2
		return "WPA2"
	}

	// Check for WPA information element
	if len(bss.WPA) > 0 {
		return "WPA"
	}

	// Check for WEP capability
	if bss.Capabilities&0x0010 != 0 { // Privacy bit
		return "WEP"
	}

	// No security
	return "Open"
}
