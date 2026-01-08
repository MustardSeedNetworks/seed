//go:build darwin

//nolint:testpackage // This file exports internal functions for testing purposes
package wifi

// ParseAirportLine exports parseAirportLine for testing.
func ParseAirportLine(line string) *ScannedNetwork {
	return parseAirportLine(line)
}

// IsDFSChannel exports isDFSChannel for testing.
func IsDFSChannel(channel int) bool {
	return isDFSChannel(channel)
}
