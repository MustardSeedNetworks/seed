// Package learn derives candidate target networks from what discovery has
// already seen on the wire, so a routed site behind the active interface does
// not have to be typed in one /24 at a time (seed#2695).
//
// The inputs are deliberately narrow value types rather than the discovery
// device model: the rules here are arithmetic over prefixes and belong in a
// package that can be exercised without an SNMP agent.
package learn

import "net/netip"

// Source names the table a candidate came out of, so the operator deciding
// whether to sweep a learned network can see why Seed thinks it exists.
type Source string

const (
	// SourceRouteTable is a router's forwarding table (IP-FORWARD-MIB).
	SourceRouteTable Source = "snmp-route-table"
	// SourceAddressTable is a device's own interface addresses (IP-MIB).
	SourceAddressTable Source = "snmp-address-table"
)

// Route type values that mean "traffic for this network is dropped here"
// (IP-FORWARD-MIB ipCidrRouteType, as internal/protocols/snmp renders them).
// A network the router refuses to forward to is not a network to sweep.
const (
	typeReject    = "reject"
	typeBlackhole = "blackhole"
)

// Prefix bounds for a learnable network.
//
// The floor exists because a summary route — 10.0.0.0/8 is the common one — is
// not a network anyone runs a host on; offering it as a sweep target would put
// 16 million addresses behind a toggle. The ceiling drops host (/32) and
// point-to-point (/31) routes, which name a peer rather than a network.
const (
	widestLearnablePrefix    = 16
	narrowestLearnablePrefix = 30
)

// Candidate is a network Seed learned rather than being told about.
type Candidate struct {
	// CIDR is the canonical masked network, e.g. "10.44.10.0/24".
	CIDR string
	// Source is the table it was read from.
	Source Source
	// Router is the address of the device whose table named it.
	Router string
}

// Route is one forwarding-table row, as much of it as the rules need.
type Route struct {
	Destination string
	Prefix      int
	Type        string
	Protocol    string
}

// Address is one interface address from a device's IP address table.
type Address struct {
	Address string
	Prefix  int
}

// Device is one discovered device's view of the networks around it.
type Device struct {
	IP        string
	Routes    []Route
	Addresses []Address
}

// Candidates returns the private IPv4 networks named by the devices' routing
// and address tables that local does not already cover, in the order they were
// first seen.
//
// Routes are kept whichever protocol learned them and whether the router calls
// them local or remote: the networks #2695 is about sit *behind* the edge
// router, so a connected-only filter would learn nothing there. What keeps the
// result sane is the range rule, not the route's provenance — a candidate has
// to be an RFC 1918 network between /16 and /30. That excludes public space,
// carrier-grade NAT, loopback, link-local and multicast in one rule, so no
// learned toggle can ever point a sweep at someone else's address space.
func Candidates(devices []Device, local []netip.Prefix) []Candidate {
	var out []Candidate
	seen := make(map[netip.Prefix]bool)

	add := func(network netip.Prefix, source Source, router string) {
		if seen[network] || !learnable(network) || covered(network, local) {
			return
		}
		seen[network] = true
		out = append(out, Candidate{CIDR: network.String(), Source: source, Router: router})
	}

	for _, device := range devices {
		for _, route := range device.Routes {
			if route.Type == typeReject || route.Type == typeBlackhole {
				continue
			}
			if network, ok := network(route.Destination, route.Prefix); ok {
				add(network, SourceRouteTable, device.IP)
			}
		}
		for _, address := range device.Addresses {
			if network, ok := network(address.Address, address.Prefix); ok {
				add(network, SourceAddressTable, device.IP)
			}
		}
	}

	return out
}

// network builds the masked network an address and prefix length describe. The
// second return is false when the pair does not describe one — an unparseable
// or non-IPv4 address, or a zero prefix, which is both the default route and
// what an unreadable ipAdEntNetMask leaves behind.
func network(address string, prefix int) (netip.Prefix, bool) {
	if prefix <= 0 {
		return netip.Prefix{}, false
	}
	addr, err := netip.ParseAddr(address)
	if err != nil || !addr.Is4() {
		return netip.Prefix{}, false
	}
	parsed, err := addr.Prefix(prefix)
	if err != nil {
		return netip.Prefix{}, false
	}
	return parsed, true
}

// learnable reports whether a network may become a sweep target.
func learnable(network netip.Prefix) bool {
	if network.Bits() < widestLearnablePrefix || network.Bits() > narrowestLearnablePrefix {
		return false
	}
	return network.Addr().IsPrivate()
}

// covered reports whether a network is already inside one Seed sweeps.
//
// Containment, not overlap: a candidate that *contains* a swept network is
// wider than it and carries addresses nothing reaches today, so it is a real
// candidate. Only a candidate wholly inside a swept network adds nothing.
func covered(network netip.Prefix, local []netip.Prefix) bool {
	for _, prefix := range local {
		if prefix.Bits() <= network.Bits() && prefix.Contains(network.Addr()) {
			return true
		}
	}
	return false
}
