//go:build darwin

package wifi

import (
	"errors"
	"fmt"
	"slices"

	"github.com/MustardSeedNetworks/foundation/pkg/corewlan"
)

// isWirelessPlatform reports whether the named interface is the Wi-Fi adapter.
func isWirelessPlatform(iface string) bool {
	names, err := corewlan.Interfaces()
	if err != nil {
		return false
	}
	return slices.Contains(names, iface)
}

// getInfoPlatform reports the current association on macOS.
//
// Returns nil when the interface is not associated, when the host has no Wi-Fi
// adapter, or when Location Services authorization is missing — CoreWLAN
// redacts the SSID and BSSID without it, and a record identifying no network is
// worse than none.
func getInfoPlatform(_ string) *Info {
	current, err := corewlan.Current()
	if err != nil {
		return nil
	}

	return &Info{
		SSID:      current.SSID,
		BSSID:     current.BSSID,
		Signal:    current.RSSI,
		Channel:   current.Channel,
		Frequency: channelToFrequencyInBand(current.Channel, int(current.Band)),
		Security:  mapSecurityType(current.Security),
	}
}

// connectPlatform joins a Wi-Fi network on macOS. Pass an empty password for an
// open network.
func connectPlatform(_, ssid, password string) (*ConnectionResult, error) {
	return associationResult(ssid, corewlan.Associate(ssid, password)), nil
}

// associationResult reports a join outcome. A refused association is an outcome
// callers render, not a Go error, which is the contract every platform shares.
func associationResult(ssid string, err error) *ConnectionResult {
	if err != nil {
		return &ConnectionResult{Success: false, Message: err.Error(), SSID: ssid}
	}
	return &ConnectionResult{
		Success: true,
		Message: "Successfully connected to " + ssid,
		SSID:    ssid,
	}
}

// disconnectPlatform leaves the current network on macOS.
//
// This disassociates rather than cycling the radio: the previous implementation
// powered Wi-Fi off and back on, which drops every interface consumer to achieve
// a disconnect.
func disconnectPlatform(_ string) (*ConnectionResult, error) {
	return disassociationResult(corewlan.Disassociate()), nil
}

// disassociationResult reports a leave outcome, on the same contract as
// [associationResult].
func disassociationResult(err error) *ConnectionResult {
	if err != nil {
		return &ConnectionResult{Success: false, Message: err.Error()}
	}
	return &ConnectionResult{Success: true, Message: "Successfully disconnected"}
}

// getSavedNetworksPlatform returns the networks macOS remembers.
func getSavedNetworksPlatform() ([]SavedNetwork, error) {
	names, err := corewlan.SavedNetworks()
	if err != nil {
		if errors.Is(err, corewlan.ErrNoInterface) {
			return []SavedNetwork{}, nil
		}
		return nil, fmt.Errorf("read saved networks: %w", err)
	}

	saved := make([]SavedNetwork, 0, len(names))
	for _, name := range names {
		saved = append(saved, SavedNetwork{SSID: name})
	}
	return saved, nil
}

// forgetNetworkPlatform removes a remembered network on macOS.
//
// Rewriting the stored configuration is an administrative operation, so this
// fails without the system-configuration right rather than appearing to succeed.
func forgetNetworkPlatform(ssid string) error {
	if err := corewlan.Forget(ssid); err != nil {
		return fmt.Errorf("failed to forget network: %w", err)
	}
	return nil
}
