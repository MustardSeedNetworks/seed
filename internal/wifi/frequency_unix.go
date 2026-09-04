//go:build linux || darwin

package wifi

// WiFi frequency conversion constants.
//
// Windows never uses these: scanner_windows.go computes frequency from
// netsh's own output via channelToFrequencyWindows instead of deriving it
// from the channel number, so this file is linux/darwin only rather than
// shared -- keeping it out of the Windows build is what makes
// channelToFrequency (and its darwin/linux callers) real dead code on that
// platform, not a false unused finding.
const (
	// 2.4 GHz band constants.
	freq24GHzBaseOffset = 2407 // Base frequency offset for 2.4 GHz channels 1-13
	freq24GHzChannel14  = 2484 // Frequency for channel 14 (Japan only)
	channel14           = 14   // Special channel 14 number

	// 5 GHz band constants.
	freq5GHzBaseOffset = 5000 // Base frequency offset for 5 GHz channels

	// 6 GHz band constants.
	freq6GHzBaseOffset = 5950 // Base frequency offset for 6 GHz channels

	// Channel frequency spacing.
	channelSpacingMHz = 5 // Standard WiFi channel spacing in MHz
)

// channelToFrequency converts a Wi-Fi channel to frequency in MHz.
func channelToFrequency(channel int) int {
	// 2.4 GHz band
	if channel >= 1 && channel <= 13 {
		return freq24GHzBaseOffset + (channel * channelSpacingMHz)
	}
	if channel == channel14 {
		return freq24GHzChannel14
	}

	// 5 GHz band
	if channel >= 36 && channel <= 64 {
		return freq5GHzBaseOffset + (channel * channelSpacingMHz)
	}
	if channel >= 100 && channel <= 144 {
		return freq5GHzBaseOffset + (channel * channelSpacingMHz)
	}
	if channel >= 149 && channel <= 165 {
		return freq5GHzBaseOffset + (channel * channelSpacingMHz)
	}

	// 6 GHz band
	if channel >= 1 && channel <= 233 {
		return freq6GHzBaseOffset + (channel * channelSpacingMHz)
	}

	return 0
}
