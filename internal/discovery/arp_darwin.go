//go:build darwin

package discovery

import (
	"fmt"
	"net"
	"syscall"
	"time"
	"unsafe"
)

// RTF_LLDATA flag indicates the route has link-layer address info (ARP entry)
const rtfLLData = 0x400

// readARPTablePlatform reads the ARP table on macOS using the routing socket.
func (s *ARPScanner) readARPTablePlatform() ([]*ARPEntry, error) {
	// Use sysctl to get ARP entries via the routing table
	// NET_RT_FLAGS with 0 returns all routes, we filter for LLDATA below
	rib, err := syscall.RouteRIB(syscall.NET_RT_FLAGS, 0)
	if err != nil {
		return nil, fmt.Errorf("RouteRIB: %w", err)
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

		msgLen := int(nativeEndian.Uint16(rib[0:2]))
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

// rt_msghdr structure size on macOS (64-bit)
const rtMsgHdrSize = 92

// parseARPRouteMessage parses a single route message for ARP info.
func parseARPRouteMessage(data []byte) *ARPEntry {
	if len(data) < rtMsgHdrSize {
		return nil
	}

	// Check rtm_type - should be RTM_GET (for sysctl query results)
	// Check rtm_flags has RTF_LLDATA (link-layer data / ARP entry)
	flags := int(nativeEndian.Uint32(data[8:12]))
	if flags&rtfLLData == 0 {
		return nil
	}

	addrs := int(nativeEndian.Uint32(data[12:16]))
	ifIndex := int(nativeEndian.Uint16(data[4:6]))

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

// nativeEndian detects the byte order of the system.
var nativeEndian = getNativeEndian()

func getNativeEndian() interface {
	Uint16([]byte) uint16
	Uint32([]byte) uint32
} {
	buf := [2]byte{}
	*(*uint16)(unsafe.Pointer(&buf[0])) = 0x0102
	if buf[0] == 0x01 {
		return bigEndian{}
	}
	return littleEndian{}
}

type littleEndian struct{}

func (littleEndian) Uint16(b []byte) uint16 {
	return uint16(b[0]) | uint16(b[1])<<8
}

func (littleEndian) Uint32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

type bigEndian struct{}

func (bigEndian) Uint16(b []byte) uint16 {
	return uint16(b[1]) | uint16(b[0])<<8
}

func (bigEndian) Uint32(b []byte) uint32 {
	return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24
}
