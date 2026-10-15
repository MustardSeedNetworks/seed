// Package ipam reports how much of each network Seed knows about is in use
// (P-D5): its assignable addresses, the ones discovery and the routers' ARP
// tables have seen, and what is left.
//
// The rules are arithmetic over prefixes and addresses, so they live here,
// apart from the discovery model and the stores that supply them.
package ipam

import (
	"encoding/binary"
	"net/netip"
	"slices"
)

// Source says why Seed knows about a network.
type Source string

const (
	// SourceLocal is the active interface's own subnet.
	SourceLocal Source = "local"
	// SourceConfigured is a target network an operator entered.
	SourceConfigured Source = "configured"
	// SourceLearned is a target network learned from routing data (seed#2695).
	SourceLearned Source = "learned"
)

// Network is one network to report on.
type Network struct {
	Prefix netip.Prefix
	Name   string
	Source Source
}

// Usage is one network's address utilization.
type Usage struct {
	Network

	// Size is the number of assignable host addresses.
	Size uint64
	// InUse is the assignable addresses seen in use, in address order.
	InUse []netip.Addr
}

// Free is the number of assignable addresses nobody has been seen on.
func (u Usage) Free() uint64 {
	return u.Size - uint64(len(u.InUse))
}

// Percent is the share of assignable addresses in use, 0 to 100.
func (u Usage) Percent() float64 {
	return float64(len(u.InUse)) * 100 / float64(u.Size)
}

// Utilization returns the usage of each IPv4 network, in the order given.
//
// A network listed twice is reported once, under its first entry, so the
// caller's order decides which source names it. IPv6 networks are skipped:
// a /64 holds 2^64 addresses, and a count of the few in use says nothing
// about exhaustion. Overlapping networks are each counted in full; a /16
// and a /24 inside it are two networks to the operator who entered both.
func Utilization(networks []Network, inUse []netip.Addr) []Usage {
	addrs := slices.Clone(inUse)
	slices.SortFunc(addrs, netip.Addr.Compare)
	addrs = slices.Compact(addrs)

	var out []Usage
	seen := make(map[netip.Prefix]bool)
	for _, network := range networks {
		prefix := network.Prefix.Masked()
		if !prefix.Addr().Is4() || seen[prefix] {
			continue
		}
		seen[prefix] = true
		network.Prefix = prefix

		usage := Usage{Network: network, Size: assignable(prefix)}
		for _, addr := range addrs {
			if prefix.Contains(addr) && isHost(prefix, addr) {
				usage.InUse = append(usage.InUse, addr)
			}
		}
		out = append(out, usage)
	}
	return out
}

// IPv4 prefix arithmetic. A /31 is a point-to-point link whose two addresses
// are both hosts (RFC 3021); any wider prefix reserves its network and
// broadcast addresses.
const (
	ipv4Bits          = 32
	pointToPointBits  = 31
	reservedAddresses = 2
)

// assignable is the number of host addresses in an IPv4 prefix.
func assignable(prefix netip.Prefix) uint64 {
	total := uint64(1) << (ipv4Bits - prefix.Bits())
	if prefix.Bits() >= pointToPointBits {
		return total
	}
	return total - reservedAddresses
}

// isHost reports whether addr is assignable in prefix, which contains it.
func isHost(prefix netip.Prefix, addr netip.Addr) bool {
	if prefix.Bits() >= pointToPointBits {
		return true
	}
	return addr != prefix.Addr() && addr != broadcast(prefix)
}

func broadcast(prefix netip.Prefix) netip.Addr {
	b := prefix.Addr().As4()
	hostMask := uint32(1)<<(ipv4Bits-prefix.Bits()) - 1
	binary.BigEndian.PutUint32(b[:], binary.BigEndian.Uint32(b[:])|hostMask)
	return netip.AddrFrom4(b)
}
