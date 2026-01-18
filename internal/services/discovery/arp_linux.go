// Package discovery provides platform-specific ARP table management and neighbor discovery.
//
// This file contains OS-specific implementations for ARP cache reading and manipulation.
// The implementation varies by platform to use native system calls and command-line tools.
//
// Platform support:
//   - Darwin (macOS): Uses 'arp -an' command and route(4) system calls
//   - Linux: Reads /proc/net/arp and uses netlink for real-time updates
//
// Features:
//   - Read current ARP cache entries
//   - Monitor ARP changes for neighbor discovery
//   - Parse MAC addresses and IP mappings
//   - Detect incomplete/failed ARP entries

//go:build linux

package discovery

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
			link, err := netlink.LinkByIndex(neigh.LinkIndex)
			if err == nil {
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
		NUD_INCOMPLETE = 0x01
		NUD_REACHABLE  = 0x02
		NUD_STALE      = 0x04
		NUD_DELAY      = 0x08
		NUD_PROBE      = 0x10
		NUD_FAILED     = 0x20
		NUD_NOARP      = 0x40
		NUD_PERMANENT  = 0x80
	)

	switch {
	case state&NUD_REACHABLE != 0:
		return "REACHABLE"
	case state&NUD_STALE != 0:
		return "STALE"
	case state&NUD_DELAY != 0:
		return "DELAY"
	case state&NUD_PROBE != 0:
		return "PROBE"
	case state&NUD_PERMANENT != 0:
		return "PERMANENT"
	case state&NUD_NOARP != 0:
		return "NOARP"
	case state&NUD_INCOMPLETE != 0:
		return "INCOMPLETE"
	case state&NUD_FAILED != 0:
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}
