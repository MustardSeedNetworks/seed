//go:build linux

package enumerate

import (
	"time"

	"github.com/vishvananda/netlink"
)

// readARPTablePlatform reads the ARP/neighbor table on Linux using netlink.
func (s *ARPScanner) readARPTablePlatform() ([]*ARPEntry, error) {
	// Get all neighbors (ARP entries) using netlink
	neighbors, err := netlink.NeighList(0, netlink.FAMILY_V4)
	if err != nil {
		return nil, err
	}

	entries := make([]*ARPEntry, 0, len(neighbors))
	for i := range neighbors {
		neigh := &neighbors[i]
		// Skip entries without valid MAC or IP.
		if len(neigh.HardwareAddr) == 0 {
			continue
		}
		if neigh.IP == nil || neigh.IP.To4() == nil {
			continue
		}

		// Map netlink state to string.
		state := neighStateToString(neigh.State)

		// Skip failed/incomplete entries
		if state == "INCOMPLETE" || state == "FAILED" {
			continue
		}

		// Get interface name
		var ifaceName string
		if neigh.LinkIndex > 0 {
			link, linkErr := netlink.LinkByIndex(neigh.LinkIndex)
			if linkErr == nil {
				ifaceName = link.Attrs().Name
			}
		}

		// Check if this IP is in our target subnets
		if !s.isInSubnet(neigh.IP.String()) {
			continue
		}

		mac := normalizeMac(neigh.HardwareAddr.String())

		entries = append(entries, &ARPEntry{
			IP:        neigh.IP.String(),
			MAC:       mac,
			Interface: ifaceName,
			State:     state,
			LastSeen:  time.Now(),
		})
	}

	return entries, nil
}

// neighStateToString converts netlink neighbor state to human-readable string.
func neighStateToString(state int) string {
	// Netlink neighbor states from linux/neighbour.h
	const (
		nudIncomplete = 0x01
		nudReachable  = 0x02
		nudStale      = 0x04
		nudDelay      = 0x08
		nudProbe      = 0x10
		nudFailed     = 0x20
		nudNoARP      = 0x40
		nudPermanent  = 0x80
	)

	switch {
	case state&nudReachable != 0:
		return "REACHABLE"
	case state&nudStale != 0:
		return "STALE"
	case state&nudDelay != 0:
		return "DELAY"
	case state&nudProbe != 0:
		return "PROBE"
	case state&nudPermanent != 0:
		return "PERMANENT"
	case state&nudNoARP != 0:
		return "NOARP"
	case state&nudIncomplete != 0:
		return "INCOMPLETE"
	case state&nudFailed != 0:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}
