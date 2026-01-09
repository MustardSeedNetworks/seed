//go:build darwin

package wifi

// ParseAirportLine exports parseAirportLine for testing.
func ParseAirportLine(line string) *ScannedNetwork {
	return parseAirportLine(line)
}

// IsDFSChannel exports isDFSChannel for testing.
func IsDFSChannel(channel int) bool {
	return isDFSChannel(channel)
}

// ChannelToFrequency exports channelToFrequency for testing.
func ChannelToFrequency(channel int) int {
	return channelToFrequency(channel)
}
