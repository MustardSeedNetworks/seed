//go:build darwin

package enumerate

import (
	"fmt"
	"net"
	"syscall"
	"time"

	"golang.org/x/net/route"
)

// ribTypeFlags is NET_RT_FLAGS (2), the sysctl that returns routes carrying a
// given flag. golang.org/x/net/route exports RIBTypeRoute (1) and
// RIBTypeInterface (3) but not this one, and neither holds the neighbour
// cache.
const ribTypeFlags route.RIBType = 2

// readARPTablePlatform reads the IPv4 neighbour cache on macOS from the
// routing socket, the same source `arp -an` reads.
//
// Both halves of the call matter (#2272). The family must be AF_INET, and the
// flag argument must be RTF_LLINFO: NET_RT_FLAGS filters on the flag it is
// given, so asking for 0 asks for routes with no flags and the kernel returns
// zero bytes — which is why this reported an empty table while `arp -an`
// listed nine neighbours. Measured on Darwin 27.0.0, 2026-09-03:
//
//	AF_INET NET_RT_FLAGS arg=0            bytes=0
//	AF_INET NET_RT_FLAGS arg=RTF_LLINFO   bytes=1292  9 entries, byte-identical to `arp -an`
//
// Parsing is route.ParseRIB's rather than hand-walked byte offsets. The offsets
// this replaces mis-read every sockaddr_dl as 02:00:00:00:00:00 and left the
// IPv6 scope id embedded in the address.
func (s *ARPScanner) readARPTablePlatform() ([]*ARPEntry, error) {
	rib, err := route.FetchRIB(syscall.AF_INET, ribTypeFlags, syscall.RTF_LLINFO)
	if err != nil {
		return nil, fmt.Errorf("fetch routing table: %w", err)
	}

	msgs, err := route.ParseRIB(ribTypeFlags, rib)
	if err != nil {
		return nil, fmt.Errorf("parse routing table: %w", err)
	}

	// Everything the kernel holds; the caller filters. See arp_linux.go.
	entries := make([]*ARPEntry, 0, len(msgs))
	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok {
			continue
		}
		if entry := arpEntryFromRoute(rm); entry != nil {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// arpEntryFromRoute turns one link-layer route into an ARP entry, or nil when
// the route is not one.
//
// Skips the same rows the Linux reader skips, for the same reasons: an entry
// with no resolved link address is an unanswered probe rather than a
// neighbour, and a multicast group is not a host. macOS carries both in this
// table; netlink does not hand them to arp_linux.go.
func arpEntryFromRoute(rm *route.RouteMessage) *ARPEntry {
	if len(rm.Addrs) <= syscall.RTAX_GATEWAY {
		return nil
	}

	dst, ok := rm.Addrs[syscall.RTAX_DST].(*route.Inet4Addr)
	if !ok {
		return nil
	}
	ip := net.IP(dst.IP[:])
	if ip.IsMulticast() || ip.IsUnspecified() {
		return nil
	}

	// macOctets is defined alongside the NDP reader; both platforms' link-layer
	// addresses are Ethernet.
	link, ok := rm.Addrs[syscall.RTAX_GATEWAY].(*route.LinkAddr)
	if !ok || len(link.Addr) != macOctets {
		return nil
	}

	iface := link.Name
	if iface == "" && rm.Index > 0 {
		if byIndex, err := net.InterfaceByIndex(rm.Index); err == nil {
			iface = byIndex.Name
		}
	}

	return &ARPEntry{
		IP:        ip.String(),
		MAC:       normalizeMac(net.HardwareAddr(link.Addr).String()),
		Interface: iface,
		// The kernel only hands out a link-layer route once the address is
		// resolved, so every row that reaches here is reachable. macOS does not
		// expose the NUD state netlink does, so there is nothing finer to map.
		State:    "REACHABLE",
		LastSeen: time.Now(),
	}
}
