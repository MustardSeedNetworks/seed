//go:build linux

package netif

import (
	"github.com/safchain/ethtool"
)

// getEthtoolSettings retrieves link settings using pure Go ethtool.
// Returns autoneg status and advertised link modes.
func getEthtoolSettings(name string) (bool, []string) {
	e, err := ethtool.NewEthtool()
	if err != nil {
		return false, nil
	}
	defer e.Close()

	// Get mapped settings
	cmd, err := e.CmdGetMapped(name)
	if err != nil {
		return false, nil
	}

	// Check auto-negotiation
	if an, ok := cmd["autoneg"]; ok {
		autoNeg := an == 1
		return autoNeg, advertisedModes(cmd)
	}
	return false, advertisedModes(cmd)
}

func advertisedModes(cmd map[string]uint64) []string {
	speedModes := map[uint64]string{
		0x001:   "10baseT/Half",
		0x002:   "10baseT/Full",
		0x004:   "100baseT/Half",
		0x008:   "100baseT/Full",
		0x010:   "1000baseT/Half",
		0x020:   "1000baseT/Full",
		0x20000: "2500baseT/Full",
		0x40000: "5000baseT/Full",
		0x1000:  "10000baseT/Full",
	}
	var advertised []string
	if adv, ok := cmd["advertising"]; ok {
		for mask, mode := range speedModes {
			if adv&mask != 0 {
				advertised = append(advertised, mode)
			}
		}
	}

	return advertised
}
