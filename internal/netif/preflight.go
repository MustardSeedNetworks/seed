package netif

import (
	"errors"
	"fmt"
	"net/netip"
)

// ErrInvalidConfig marks a configuration this package refused before touching
// the interface. It is the difference between "you asked for something that
// cannot work" and "the apply failed", which the caller has to be able to tell
// apart: the first is a 400 naming the field, the second a 500.
var ErrInvalidConfig = errors.New("invalid network configuration")

// Pre-flight validation for a static IP configuration (#50).
//
// The syntactic checks in validateIPConfig confirm each field parses. These
// confirm the configuration can actually work, which is the risk the issue
// names: "a bad configuration could render the device unreachable, requiring
// physical access to recover". Every case below parses cleanly and strands the
// box anyway.
//
// Pure functions over the requested values — no interface is touched, so this
// runs before anything is applied and is testable without a NIC.

// pointToPointBits is the /31 prefix length. RFC 3021 gives a /31 two usable
// host addresses, so the network and broadcast checks do not apply to it, nor
// to a /32.
const (
	pointToPointBits = 31

	// bitsPerOctet is how many host bits one octet of an IPv4 address carries.
	bitsPerOctet = 8
)

// preflightStaticIP reports why a syntactically valid configuration cannot work.
//
// A value this cannot parse is not this function's to report: validateIPConfig
// has already rejected it by the time we are called, and duplicating the
// message here would mean two places to keep in step.
func preflightStaticIP(cfg *StaticIPConfig) error {
	bits, ok := parseNetmask(cfg.Netmask)
	if !ok {
		return nil
	}

	addr, ok := parseIPv4(cfg.Address)
	if !ok {
		return nil
	}

	if err := checkHostAddress(addr, "address"); err != nil {
		return err
	}

	prefix := netip.PrefixFrom(addr, bits).Masked()
	if err := checkAssignable(addr, prefix, bits, "address"); err != nil {
		return err
	}

	if cfg.Gateway == "" {
		return nil
	}

	return preflightGateway(cfg.Gateway, addr, prefix, bits)
}

// parseIPv4 parses an IPv4 address, reporting whether it is one.
func parseIPv4(value string) (netip.Addr, bool) {
	addr, err := netip.ParseAddr(value)
	if err != nil || !addr.Is4() {
		return netip.Addr{}, false
	}

	return addr, true
}

// preflightGateway checks the next hop is one the host can actually use.
func preflightGateway(gateway string, addr netip.Addr, prefix netip.Prefix, bits int) error {
	gw, ok := parseIPv4(gateway)
	if !ok {
		return nil
	}

	if gw == addr {
		return fmt.Errorf(
			"%w: gateway %s is the same address as the interface, so the default "+
				"route would point at this host", ErrInvalidConfig, gw)
	}
	if hostErr := checkHostAddress(gw, "gateway"); hostErr != nil {
		return hostErr
	}
	if assignErr := checkAssignable(gw, prefix, bits, "gateway"); assignErr != nil {
		return assignErr
	}

	// A /32 has no on-link neighbours, so an off-subnet gateway is the only
	// kind it can have; every other prefix needs the next hop on the wire.
	if bits < ipv4BitLength && !prefix.Contains(gw) {
		return fmt.Errorf(
			"%w: gateway %s is not reachable from %s/%d, so the next hop must be "+
				"inside the interface's own subnet", ErrInvalidConfig, gw, addr, bits)
	}

	return nil
}

// checkHostAddress rejects addresses that cannot be assigned to, or routed
// through, an interface at all.
func checkHostAddress(addr netip.Addr, role string) error {
	switch {
	case addr.IsLoopback():
		return fmt.Errorf("%w: %s %s is a loopback address", ErrInvalidConfig, role, addr)
	case addr.IsMulticast():
		return fmt.Errorf("%w: %s %s is a multicast address", ErrInvalidConfig, role, addr)
	case addr.IsUnspecified():
		return fmt.Errorf("%w: %s %s is the unspecified address", ErrInvalidConfig, role, addr)
	case addr.IsLinkLocalMulticast():
		return fmt.Errorf("%w: %s %s is a link-local multicast address", ErrInvalidConfig, role, addr)
	default:
		return nil
	}
}

// checkAssignable rejects the two addresses in a subnet that name the subnet
// rather than a host on it.
func checkAssignable(addr netip.Addr, prefix netip.Prefix, bits int, role string) error {
	if bits >= pointToPointBits {
		return nil // RFC 3021: /31 and /32 have no network or broadcast address.
	}
	if addr == prefix.Addr() {
		return fmt.Errorf("%w: %s %s is the network address of %s", ErrInvalidConfig, role, addr, prefix)
	}
	if addr == broadcastOf(prefix) {
		return fmt.Errorf("%w: %s %s is the broadcast address of %s", ErrInvalidConfig, role, addr, prefix)
	}

	return nil
}

// broadcastOf returns the all-ones address of an IPv4 prefix.
//
// Sets every host bit on the network address, octet by octet. Packing the four
// bytes into a uint32 would be shorter and would need two conversions gosec
// flags as potential overflows; this needs none.
func broadcastOf(prefix netip.Prefix) netip.Addr {
	octets := prefix.Addr().As4()
	hostBits := ipv4BitLength - prefix.Bits()

	for i := len(octets) - 1; i >= 0 && hostBits > 0; i-- {
		width := min(hostBits, bitsPerOctet)
		octets[i] |= byte(1<<width - 1)
		hostBits -= width
	}

	return netip.AddrFrom4(octets)
}
