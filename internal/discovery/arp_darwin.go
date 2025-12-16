// Package discovery provides platform-specific ARP table management and neighbor discovery.
//
// This file contains OS-specific implementations for ARP cache reading and manipulation.
// The implementation varies by platform to use native system calls and command-line tools.
//
// Platform support:
//   - Darwin (macOS): Uses golang.org/x/net/route package for routing table access
//   - Linux: Reads /proc/net/arp and uses netlink for real-time updates
//
// Features:
//   - Read current ARP cache entries
//   - Monitor ARP changes for neighbor discovery
//   - Parse MAC addresses and IP mappings
//   - Detect incomplete/failed ARP entries

//go:build darwin

package discovery

import (
	"encoding/binary"
	"fmt"
	"net"
	"syscall"
	"time"

	"golang.org/x/net/route"
)

// RTF_LLDATA flag indicates the route has link-layer address info (ARP entry).
const rtfLLData = 0x400

// ribTypeFlags is NET_RT_FLAGS (2) which returns routes with specific flags.
// The golang.org/x/net/route package only exports RIBTypeRoute (1) and RIBTypeInterface (3),
// so we define our own constant for NET_RT_FLAGS to access ARP entries.
// This is the proper modern approach using the x/net/route package.
const ribTypeFlags route.RIBType = 2

// readARPTablePlatform reads the ARP table on macOS using the golang.org/x/net/route package.
// This uses the modern golang.org/x/net/route.FetchRIB API instead of the deprecated syscall.RouteRIB.
func (s *ARPScanner) readARPTablePlatform() ([]*ARPEntry, error) {
	// Use route.FetchRIB with NET_RT_FLAGS (2) to get ARP entries
	// AF_UNSPEC (0) returns all address families
	// The arg parameter is 0 for all entries (wildcard)
	rib, err := route.FetchRIB(syscall.AF_UNSPEC, ribTypeFlags, 0)
	if err != nil {
		return nil, fmt.Errorf("FetchRIB: %w", err)
	}

	entries, err := parseARPMessages(rib)
	if err != nil {
		return nil, err
	}

	// Filter by subnet if configured
	var filtered []*ARPEntry
	for _, entry := range entries {
		if s.isInSubnet(entry.IP) {
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}

// parseARPMessages parses the routing information base for ARP entries.
func parseARPMessages(rib []byte) ([]*ARPEntry, error) {
	var entries []*ARPEntry

	for len(rib) > 0 {
		if len(rib) < 4 {
			break
		}

		msgLen := int(binary.LittleEndian.Uint16(rib[0:2]))
		if msgLen == 0 || msgLen > len(rib) {
			break
		}

		entry := parseARPRouteMessage(rib[:msgLen])
		if entry != nil {
			entries = append(entries, entry)
		}

		rib = rib[msgLen:]
	}

	return entries, nil
}

// rt_msghdr structure size on macOS (64-bit).
const rtMsgHdrSize = 92

// parseARPRouteMessage parses a single route message for ARP info.
func parseARPRouteMessage(data []byte) *ARPEntry {
	if len(data) < rtMsgHdrSize {
		return nil
	}

	// Check rtm_flags has RTF_LLDATA (link-layer data / ARP entry)
	flags := int(binary.LittleEndian.Uint32(data[8:12]))
	if flags&rtfLLData == 0 {
		return nil
	}

	addrs := int(binary.LittleEndian.Uint32(data[12:16]))
	ifIndex := int(binary.LittleEndian.Uint16(data[4:6]))

	entry := &ARPEntry{
		State:    "REACHABLE",
		LastSeen: time.Now(),
	}

	// Get interface name
	if ifIndex > 0 {
		if iface, err := net.InterfaceByIndex(ifIndex); err == nil {
			entry.Interface = iface.Name
		}
	}

	// Parse socket addresses after header
	pos := rtMsgHdrSize

	// RTA_DST (1), RTA_GATEWAY (2), ...
	for i := 0; i < 8 && pos < len(data); i++ {
		bit := 1 << i
		if addrs&bit == 0 {
			continue
		}

		if pos >= len(data) {
			break
		}

		saLen := int(data[pos])
		if saLen == 0 {
			saLen = 4
		}
		saLen = (saLen + 3) &^ 3 // Round to 4-byte alignment

		if pos+saLen > len(data) {
			break
		}

		saFamily := int(data[pos+1])

		switch bit {
		case 1: // RTA_DST - IP address
			ip := parseSockaddrIP(data[pos:pos+saLen], saFamily)
			if ip != nil {
				entry.IP = ip.String()
			}
		case 2: // RTA_GATEWAY - MAC address (for ARP entries)
			mac := parseSockaddrDL(data[pos : pos+saLen])
			if mac != "" {
				entry.MAC = mac
			}
		}

		pos += saLen
	}

	// Need both IP and MAC for a valid ARP entry
	if entry.IP == "" || entry.MAC == "" {
		return nil
	}

	return entry
}

// parseSockaddrIP extracts an IP address from a sockaddr structure.
func parseSockaddrIP(data []byte, family int) net.IP {
	if len(data) < 4 {
		return nil
	}

	switch family {
	case syscall.AF_INET:
		if len(data) >= 8 {
			// sockaddr_in: len (1), family (1), port (2), addr (4)
			return net.IP(data[4:8])
		}
	case syscall.AF_INET6:
		if len(data) >= 24 {
			// sockaddr_in6: len (1), family (1), port (2), flowinfo (4), addr (16)
			return net.IP(data[8:24])
		}
	}
	return nil
}

// parseSockaddrDL extracts a MAC address from a sockaddr_dl structure.
func parseSockaddrDL(data []byte) string {
	if len(data) < 8 {
		return ""
	}

	// sockaddr_dl layout:
	// sdl_len (1), sdl_family (1), sdl_index (2), sdl_type (1), sdl_nlen (1), sdl_alen (1), sdl_slen (1)
	// followed by interface name (sdl_nlen bytes) then link address (sdl_alen bytes)

	family := data[1]
	if family != syscall.AF_LINK {
		return ""
	}

	nlen := int(data[5]) // interface name length
	alen := int(data[6]) // link address length

	if alen != 6 { // Ethernet MAC address
		return ""
	}

	// MAC address starts after the header (8 bytes) + interface name
	macStart := 8 + nlen
	if macStart+6 > len(data) {
		return ""
	}

	mac := data[macStart : macStart+6]
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
}
