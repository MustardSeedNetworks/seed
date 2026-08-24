//go:build darwin

package wifi

import (
	"github.com/MustardSeedNetworks/foundation/pkg/corewlan"

	"github.com/MustardSeedNetworks/seed/internal/wifi/wifihelper"
)

// NetworkFromCoreWLAN exports networkFromCoreWLAN for testing.
func NetworkFromCoreWLAN(n corewlan.Network) *ScannedNetwork {
	return networkFromCoreWLAN(n)
}

// IsDFSChannel exports isDFSChannel for testing.
func IsDFSChannel(channel int) bool {
	return isDFSChannel(channel)
}

// ChannelToFrequency exports channelToFrequency for testing.
func ChannelToFrequency(channel int) int {
	return channelToFrequency(channel)
}

// ChannelToFrequencyInBand exports channelToFrequencyInBand for testing.
func ChannelToFrequencyInBand(channel, bandGHz int) int {
	return channelToFrequencyInBand(channel, bandGHz)
}

// AssociationResult exports associationResult for testing.
func AssociationResult(ssid string, err error) *ConnectionResult {
	return associationResult(ssid, err)
}

// DisassociationResult exports disassociationResult for testing.
func DisassociationResult(err error) *ConnectionResult {
	return disassociationResult(err)
}

// ShouldDelegate exports shouldDelegate for testing.
func ShouldDelegate(err error) bool {
	return shouldDelegate(err)
}

// NetworkFromHelper exports networkFromHelper for testing.
func NetworkFromHelper(n wifihelper.Network) *ScannedNetwork {
	return networkFromHelper(n)
}

// CurrentHelper exports currentHelper for testing.
func (s *Scanner) CurrentHelper() Helper {
	return s.currentHelper()
}
