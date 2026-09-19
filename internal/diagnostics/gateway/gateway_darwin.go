//go:build darwin

package gateway

// macOS implementation uses routing socket interface to detect default IPv4 and IPv6 gateways,
// enumerate all routes, and monitor routing changes. Also provides detailed route information.

import (
	"fmt"
	"net"
	"syscall"

	"golang.org/x/net/route"
)

// extractRouteIP extracts IP address and family from a route address.
func extractRouteIP(addr route.Addr) (string, string) {
	switch a := addr.(type) {
	case *route.Inet4Addr:
		return net.IP(a.IP[:]).String(), "inet"
	case *route.Inet6Addr:
		return net.IP(a.IP[:]).String(), "inet6"
	}
	return "", ""
}

// parseRouteMessage extracts RouteInfo from a route message.
func parseRouteMessage(rm *route.RouteMessage) *RouteInfo {
	ri := &RouteInfo{}

	// Get destination
	if len(rm.Addrs) > syscall.RTAX_DST && rm.Addrs[syscall.RTAX_DST] != nil {
		ri.Destination, ri.Family = extractRouteIP(rm.Addrs[syscall.RTAX_DST])
	}

	// Get gateway
	if len(rm.Addrs) > syscall.RTAX_GATEWAY && rm.Addrs[syscall.RTAX_GATEWAY] != nil {
		ri.Gateway, _ = extractRouteIP(rm.Addrs[syscall.RTAX_GATEWAY])
	}

	// Get interface
	if rm.Index > 0 {
		if iface, err := net.InterfaceByIndex(rm.Index); err == nil {
			ri.Interface = iface.Name
		}
	}

	ri.Prefix = routePrefix(rm, ri.Family)

	if ri.Destination == "" && ri.Gateway == "" {
		return nil
	}
	return ri
}

// Host prefix lengths, i.e. the mask a route to a single address carries.
const (
	hostPrefixIPv4 = 32
	hostPrefixIPv6 = 128
)

// hostPrefix is the prefix length of a host route in the given family.
func hostPrefix(family string) int {
	if family == "inet6" {
		return hostPrefixIPv6
	}
	return hostPrefixIPv4
}

// routePrefix is the route's prefix length. RTF_HOST is read before the
// netmask because the kernel omits RTAX_NETMASK entirely for a host route:
// taking a missing mask for a /0 is exactly the confusion seed#2765 is about,
// and it would turn every ARP-cloned neighbour entry into a default route.
// The real default route does carry a mask, of 0.0.0.0.
func routePrefix(rm *route.RouteMessage, family string) int {
	if rm.Flags&syscall.RTF_HOST != 0 {
		return hostPrefix(family)
	}
	if len(rm.Addrs) > syscall.RTAX_NETMASK {
		if bits, ok := maskBits(rm.Addrs[syscall.RTAX_NETMASK]); ok {
			return bits
		}
	}
	return 0
}

// maskBits counts the leading ones of a netmask carried as a route address.
// The second return is false when the slot holds no mask at all.
func maskBits(addr route.Addr) (int, bool) {
	switch a := addr.(type) {
	case *route.Inet4Addr:
		return prefixFromMask(a.IP[:])
	case *route.Inet6Addr:
		return prefixFromMask(a.IP[:])
	}
	return 0, false
}

// GetAllRoutes returns all routes using the routing socket.
func GetAllRoutes() ([]RouteInfo, error) {
	rib, err := route.FetchRIB(syscall.AF_UNSPEC, syscall.NET_RT_DUMP, 0)
	if err != nil {
		return nil, fmt.Errorf("fetch RIB: %w", err)
	}

	msgs, err := route.ParseRIB(syscall.NET_RT_DUMP, rib)
	if err != nil {
		return nil, fmt.Errorf("parse RIB: %w", err)
	}

	var routes []RouteInfo
	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok {
			continue
		}
		if ri := parseRouteMessage(rm); ri != nil {
			routes = append(routes, *ri)
		}
	}
	return routes, nil
}

// GetDefaultGatewayInterface returns the interface used for the default route.
func GetDefaultGatewayInterface() (string, error) {
	rib, err := route.FetchRIB(syscall.AF_INET, syscall.NET_RT_DUMP, 0)
	if err != nil {
		return "", fmt.Errorf("fetch RIB: %w", err)
	}

	msgs, err := route.ParseRIB(syscall.NET_RT_DUMP, rib)
	if err != nil {
		return "", fmt.Errorf("parse RIB: %w", err)
	}

	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok {
			continue
		}

		if isDefaultRouteMsg(rm, syscall.AF_INET) && rm.Index > 0 {
			if iface, ifaceErr := net.InterfaceByIndex(rm.Index); ifaceErr == nil {
				return iface.Name, nil
			}
		}
	}

	return "", nil
}

// detectGatewayPlatform is the platform-specific gateway detection.
// On macOS, this uses golang.org/x/net/route for reliable parsing.
func detectGatewayPlatform() (string, error) {
	rib, err := route.FetchRIB(syscall.AF_INET, syscall.NET_RT_DUMP, 0)
	if err != nil {
		return "", fmt.Errorf("fetch RIB: %w", err)
	}

	msgs, err := route.ParseRIB(syscall.NET_RT_DUMP, rib)
	if err != nil {
		return "", fmt.Errorf("parse RIB: %w", err)
	}

	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok {
			continue
		}

		if !isDefaultRouteMsg(rm, syscall.AF_INET) {
			continue
		}

		// Get gateway address
		if len(rm.Addrs) > syscall.RTAX_GATEWAY {
			if gw := rm.Addrs[syscall.RTAX_GATEWAY]; gw != nil {
				if a, addrOK := gw.(*route.Inet4Addr); addrOK {
					return net.IP(a.IP[:]).String(), nil
				}
			}
		}
	}

	return "", nil
}

// extractIPv6Gateway extracts the IPv6 gateway from a route message.
// Returns empty string if not found. If skipLinkLocal is true, link-local addresses are skipped.
func extractIPv6Gateway(rm *route.RouteMessage, skipLinkLocal bool) string {
	if len(rm.Addrs) <= syscall.RTAX_GATEWAY {
		return ""
	}
	gw := rm.Addrs[syscall.RTAX_GATEWAY]
	if gw == nil {
		return ""
	}
	a, ok := gw.(*route.Inet6Addr)
	if !ok {
		return ""
	}
	ip := net.IP(a.IP[:])
	if skipLinkLocal && ip.IsLinkLocalUnicast() {
		return ""
	}
	return ip.String()
}

// detectGatewayIPv6Platform is the platform-specific IPv6 gateway detection.
func detectGatewayIPv6Platform() (string, error) {
	rib, err := route.FetchRIB(syscall.AF_INET6, syscall.NET_RT_DUMP, 0)
	if err != nil {
		return "", fmt.Errorf("fetch IPv6 RIB: %w", err)
	}

	msgs, err := route.ParseRIB(syscall.NET_RT_DUMP, rib)
	if err != nil {
		return "", fmt.Errorf("parse IPv6 RIB: %w", err)
	}

	// First pass: prefer non-link-local addresses
	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok || !isDefaultRouteMsg(rm, syscall.AF_INET6) {
			continue
		}
		if gw := extractIPv6Gateway(rm, true); gw != "" {
			return gw, nil
		}
	}

	// Second pass: accept link-local addresses
	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok || !isDefaultRouteMsg(rm, syscall.AF_INET6) {
			continue
		}
		if gw := extractIPv6Gateway(rm, false); gw != "" {
			return gw, nil
		}
	}

	return "", nil
}

// isDefaultRouteMsg checks if a route message represents the default route.
func isDefaultRouteMsg(rm *route.RouteMessage, family int) bool {
	if len(rm.Addrs) <= syscall.RTAX_DST {
		return false
	}

	dst := rm.Addrs[syscall.RTAX_DST]
	if dst == nil {
		// No destination means default route
		return true
	}

	switch family {
	case syscall.AF_INET:
		if a, ok := dst.(*route.Inet4Addr); ok {
			// Check for 0.0.0.0
			return a.IP == [4]byte{0, 0, 0, 0}
		}
	case syscall.AF_INET6:
		if a, ok := dst.(*route.Inet6Addr); ok {
			// Check for ::
			return a.IP == [16]byte{}
		}
	}

	return false
}
