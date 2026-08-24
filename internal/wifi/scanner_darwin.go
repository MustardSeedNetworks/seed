//go:build darwin

package wifi

import (
	"fmt"

	"github.com/MustardSeedNetworks/foundation/pkg/corewlan"
)

// defaultNoiseFloorDBm is used when CoreWLAN reports no noise measurement for a
// scanned network, which it omits on some adapters. Reporting 0 dBm instead
// would make the derived SNR nonsense.
const defaultNoiseFloorDBm = -95

// freq6GHzChannel1 is the frequency of 6 GHz channel 1, in MHz.
const freq6GHzChannel1 = 5950

// Frequency bands, in GHz, as reported by the driver.
const (
	band24GHz = 2
	band5GHz  = 5
	band6GHz  = 6
)

// scanPlatform performs a Wi-Fi scan on macOS via CoreWLAN.
//
// The `airport` utility this previously shelled out to was removed in macOS 26.
// A scan without Location Services authorization does not fail — CoreWLAN
// returns every SSID and BSSID emptied — so the binding reports that as
// [corewlan.ErrLocationDenied] and it is surfaced here rather than looking like
// an empty airspace.
func scanPlatform(_ string) ([]*ScannedNetwork, error) {
	found, err := corewlan.Scan()
	if err != nil {
		return nil, fmt.Errorf("wifi scan: %w", err)
	}

	networks := make([]*ScannedNetwork, 0, len(found))
	for _, n := range found {
		networks = append(networks, networkFromCoreWLAN(n))
	}
	return networks, nil
}

// networkFromCoreWLAN maps a CoreWLAN observation onto the scanner's model.
func networkFromCoreWLAN(n corewlan.Network) *ScannedNetwork {
	noise := n.Noise
	if noise == 0 {
		noise = defaultNoiseFloorDBm
	}

	width := n.ChannelWidth
	if width == 0 {
		width = ChannelWidth20MHz
	}

	return &ScannedNetwork{
		SSID:         n.SSID,
		BSSID:        n.BSSID,
		Signal:       n.RSSI,
		Channel:      n.Channel,
		Frequency:    channelToFrequencyInBand(n.Channel, int(n.Band)),
		Security:     mapSecurityType(n.Security),
		ChannelWidth: width,
		NoiseFloor:   noise,
		SNR:          n.RSSI - noise,
		HTMode:       htModeForWidth(width),
		IsDFS:        n.Band == corewlan.Band5GHz && isDFSChannel(n.Channel),
	}
}

// htModeForWidth names the widest PHY that carries a given channel width, using
// the vocabulary detectChannelWidth already parses.
func htModeForWidth(width int) string {
	switch width {
	case ChannelWidth40MHz:
		return "HT40"
	case ChannelWidth80MHz:
		return "VHT80"
	case ChannelWidth160MHz:
		return "HE160"
	case ChannelWidth320MHz:
		return "EHT320"
	default:
		return "HT20"
	}
}

// channelToFrequencyInBand converts a channel to MHz using the band reported by
// the driver. Channel numbers collide across bands — 6 GHz channel 1 is 5955 MHz
// while 2.4 GHz channel 1 is 2412 MHz — so a channel number alone cannot be
// resolved. Falls back to [channelToFrequency] when the band is unknown.
func channelToFrequencyInBand(channel, bandGHz int) int {
	switch bandGHz {
	case band24GHz:
		if channel == channel14 {
			return freq24GHzChannel14
		}
		return freq24GHzBaseOffset + (channel * channelSpacingMHz)
	case band5GHz:
		return freq5GHzBaseOffset + (channel * channelSpacingMHz)
	case band6GHz:
		return freq6GHzChannel1 + (channel * channelSpacingMHz)
	default:
		return channelToFrequency(channel)
	}
}

// isDFSChannel reports whether a 5 GHz channel requires radar detection:
// 52-64 (UNII-2) and 100-144 (UNII-2 Extended).
func isDFSChannel(channel int) bool {
	return (channel >= 52 && channel <= 64) || (channel >= 100 && channel <= 144)
}
