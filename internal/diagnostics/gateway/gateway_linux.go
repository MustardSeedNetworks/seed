//go:build linux

package gateway

import (
	"net"

	"github.com/vishvananda/netlink"
)

// detectGatewayNetlink uses netlink to detect the default IPv4 gateway.
func detectGatewayNetlink() (string, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return "", err
	}

	for i := range routes {
		route := &routes[i]
		// Default route has nil Dst OR 0.0.0.0/0.
		isDefault := route.Dst == nil ||
			(route.Dst != nil && route.Dst.IP.Equal(net.IPv4zero) && route.Dst.Mask.String() == "00000000")
		if isDefault && route.Gw != nil {
			return route.Gw.String(), nil
		}
	}

	return "", nil
}

// detectGatewayIPv6Netlink uses netlink to detect the default IPv6 gateway.
func detectGatewayIPv6Netlink() (string, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V6)
	if err != nil {
		return "", err
	}

	for i := range routes {
		route := &routes[i]
		// Default route has nil Dst (or ::/0).
		if route.Dst == nil && route.Gw != nil {
			// Ensure it's a valid IPv6 address (not IPv4-mapped).
			if ip := route.Gw; ip != nil && ip.To4() == nil {
				return ip.String(), nil
			}
		}
		// Also check for explicit ::/0 destination.
		if route.Dst != nil && route.Dst.String() == "::/0" && route.Gw != nil {
			if ip := route.Gw; ip != nil && ip.To4() == nil {
				return ip.String(), nil
			}
		}
	}

	return "", nil
}

// GetAllRoutes returns all routes using netlink (for debugging/display).
func GetAllRoutes() ([]RouteInfo, error) {
	var routes []RouteInfo
	v4Routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err == nil {
		routes = appendRouteInfo(routes, v4Routes, "inet")
	}
	v6Routes, err := netlink.RouteList(nil, netlink.FAMILY_V6)
	if err == nil {
		routes = appendRouteInfo(routes, v6Routes, "inet6")
	}
	return routes, nil
}

func appendRouteInfo(result []RouteInfo, routes []netlink.Route, family string) []RouteInfo {
	for index := range routes {
		route := &routes[index]
		// A nil Dst is the default route, which is 0.0.0.0/0 or ::/0 — the
		// same shape as every other row rather than the word "default"
		// (seed#2765), so a caller can read one field and get a network.
		info := RouteInfo{Destination: net.IPv4zero.String(), Family: family}
		if family == "inet6" {
			info.Destination = net.IPv6zero.String()
		}
		if route.Dst != nil {
			info.Destination = route.Dst.IP.String()
			info.Prefix, _ = route.Dst.Mask.Size()
		}
		if route.Gw != nil {
			info.Gateway = route.Gw.String()
		}
		if route.LinkIndex > 0 {
			if link, err := netlink.LinkByIndex(route.LinkIndex); err == nil {
				info.Interface = link.Attrs().Name
			}
		}
		result = append(result, info)
	}
	return result
}

// GetDefaultGatewayInterface returns the interface used for the default route.
func GetDefaultGatewayInterface() (string, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return "", err
	}

	for i := range routes {
		route := &routes[i]
		// Default route has nil Dst OR 0.0.0.0/0.
		isDefault := route.Dst == nil ||
			(route.Dst != nil && route.Dst.IP.Equal(net.IPv4zero) && route.Dst.Mask.String() == "00000000")
		if isDefault && route.LinkIndex > 0 {
			link, linkErr := netlink.LinkByIndex(route.LinkIndex)
			if linkErr == nil {
				return link.Attrs().Name, nil
			}
		}
	}

	return "", nil
}

// detectGatewayPlatform is the platform-specific gateway detection.
// On Linux, this uses netlink.
func detectGatewayPlatform() (string, error) {
	return detectGatewayNetlink()
}

// detectGatewayIPv6Platform is the platform-specific IPv6 gateway detection.
// On Linux, this uses netlink.
func detectGatewayIPv6Platform() (string, error) {
	return detectGatewayIPv6Netlink()
}
