//go:build darwin

package detection

import "github.com/MustardSeedNetworks/foundation/pkg/corewlan"

// wirelessInterfaces names the host's Wi-Fi adapters.
//
// CoreWLAN answers this without Location Services authorization — the interface
// list is not a network identifier — so it is the one reliable signal on a Mac,
// where the Wi-Fi adapter is `en0` and every name pattern reads that as
// Ethernet (#2670).
func wirelessInterfaces() []string {
	names, err := corewlan.Interfaces()
	if err != nil {
		return nil
	}
	return names
}
