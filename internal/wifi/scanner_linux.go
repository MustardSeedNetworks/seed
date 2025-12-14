//go:build linux

package wifi

import (
	"context"
	"fmt"
	"time"

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

	// Trigger scan with context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Scan(ctx, wifiIface); err != nil {
		return nil, fmt.Errorf("failed to trigger scan: %w", err)
	}

	// Get BSS (scan results) - this returns the current BSS we're connected to
	// For a full scan, we would need to use StationInfo, but that's more complex
	// For now, just return an error indicating scanning is not fully implemented
	return nil, fmt.Errorf("WiFi scanning not fully implemented on Linux - requires additional nl80211 work")
}
