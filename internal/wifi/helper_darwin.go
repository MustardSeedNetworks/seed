//go:build darwin

package wifi

import (
	"errors"

	"github.com/MustardSeedNetworks/foundation/pkg/corewlan"

	"github.com/MustardSeedNetworks/seed/internal/wifi/wifihelper"
)

// shouldDelegate reports whether an in-process CoreWLAN failure is the kind a
// helper can resolve. Only a missing Location grant qualifies: any other error
// would fail identically in the helper, and retrying it there would replace a
// precise message with a vaguer one.
func shouldDelegate(err error) bool {
	return errors.Is(err, corewlan.ErrLocationDenied)
}

// networkFromHelper maps a helper observation onto the scanner's model, sharing
// the derivation rules with the in-process path.
func networkFromHelper(n wifihelper.Network) *ScannedNetwork {
	return networkFromCoreWLAN(corewlan.Network{
		SSID:         n.SSID,
		BSSID:        n.BSSID,
		RSSI:         n.RSSI,
		Noise:        n.Noise,
		Channel:      n.Channel,
		ChannelWidth: n.ChannelWidth,
		Band:         corewlan.Band(n.Band),
		PHYMode:      n.PHYMode,
		Security:     n.Security,
	})
}
