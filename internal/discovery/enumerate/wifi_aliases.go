package enumerate

import "github.com/MustardSeedNetworks/seed/internal/discovery"

// Kernel Wi-Fi type aliases. The result/data types live in the discovery kernel
// (they sit in the WiFiCollectorPort signatures and the kernel enumerate-stage
// converter, so they must live there). Aliasing them here lets the WiFiBridge
// reference them unqualified without duplicating the definitions or inverting the
// dependency direction.
type (
	// WiFiBand is the kernel Wi-Fi frequency-band enum.
	WiFiBand = discovery.WiFiBand
	// WiFiSecurityType is the kernel Wi-Fi security-protocol enum.
	WiFiSecurityType = discovery.WiFiSecurityType
	// WiFiAuthorizationStatus is the kernel authorization-status enum.
	WiFiAuthorizationStatus = discovery.WiFiAuthorizationStatus
	// WiFiNetwork is a discovered Wi-Fi network/SSID (kernel result type).
	WiFiNetwork = discovery.WiFiNetwork
	// WiFiAccessPoint is a discovered Wi-Fi access point/BSSID (kernel result type).
	WiFiAccessPoint = discovery.WiFiAccessPoint
	// WiFiClient is a discovered Wi-Fi client (kernel result type).
	WiFiClient = discovery.WiFiClient
	// ChannelUtilization is per-channel usage metrics (kernel result type).
	ChannelUtilization = discovery.ChannelUtilization
	// WiFiScanResult is a complete Wi-Fi scan result (kernel result type).
	WiFiScanResult = discovery.WiFiScanResult
	// WiFiDiscoveryStats is aggregated Wi-Fi statistics (kernel result type).
	WiFiDiscoveryStats = discovery.WiFiDiscoveryStats
)

// Kernel constant aliases, re-exported so the bridge logic references them
// unqualified.
const (
	WiFiBand24GHz = discovery.WiFiBand24GHz
	WiFiBand5GHz  = discovery.WiFiBand5GHz
	WiFiBand6GHz  = discovery.WiFiBand6GHz

	WiFiSecurityOpen = discovery.WiFiSecurityOpen
	WiFiSecurityWEP  = discovery.WiFiSecurityWEP
	WiFiSecurityWPA  = discovery.WiFiSecurityWPA
	WiFiSecurityWPA2 = discovery.WiFiSecurityWPA2
	WiFiSecurityWPA3 = discovery.WiFiSecurityWPA3

	WiFiAuthAuthorized   = discovery.WiFiAuthAuthorized
	WiFiAuthUnauthorized = discovery.WiFiAuthUnauthorized
	WiFiAuthUnknown      = discovery.WiFiAuthUnknown
)

// ChannelToBand returns the Wi-Fi band for a given channel number. It is domain
// knowledge used only by the Wi-Fi collector, so it lives with the bridge in the
// enumerate stage (it moved out of the kernel wifi.go alongside the collector).
func ChannelToBand(channel int) WiFiBand {
	switch {
	case channel >= 1 && channel <= 14:
		return WiFiBand24GHz
	case channel >= 32 && channel <= 177:
		return WiFiBand5GHz
	case channel >= 1 && channel <= 233: // 6GHz uses different numbering
		// 6GHz channels start at 1 but go higher
		// This is simplified - actual 6GHz detection needs frequency
		return WiFiBand6GHz
	default:
		return WiFiBand24GHz
	}
}
