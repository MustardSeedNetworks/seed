//go:build darwin

package wifi

import "github.com/MustardSeedNetworks/foundation/pkg/corewlan"

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
