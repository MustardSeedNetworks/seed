//go:build linux

// Package gateway provides gateway reachability testing and latency measurement.
// Linux implementation uses netlink to detect default IPv4 and IPv6 gateways from kernel routing table,
// enabling accurate gateway address resolution for network diagnostics.
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

	// Get IPv4 routes.
	v4Routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err == nil {
		for i := range v4Routes {
			r := &v4Routes[i]
			ri := RouteInfo{
				Family: "inet",
			}
			if r.Dst != nil {
				ri.Destination = r.Dst.String()
			} else {
				ri.Destination = "default"
			}
			if r.Gw != nil {
				ri.Gateway = r.Gw.String()
			}
			if r.LinkIndex > 0 {
				if link, linkErr := netlink.LinkByIndex(r.LinkIndex); linkErr == nil {
					ri.Interface = link.Attrs().Name
				}
			}
			routes = append(routes, ri)
		}
	}

	// Get IPv6 routes.
	v6Routes, err := netlink.RouteList(nil, netlink.FAMILY_V6)
	if err == nil {
		for i := range v6Routes {
			r := &v6Routes[i]
			ri := RouteInfo{
				Family: "inet6",
			}
			if r.Dst != nil {
				ri.Destination = r.Dst.String()
			} else {
				ri.Destination = "default"
			}
			if r.Gw != nil {
				ri.Gateway = r.Gw.String()
			}
			if r.LinkIndex > 0 {
				if link, linkErr := netlink.LinkByIndex(r.LinkIndex); linkErr == nil {
					ri.Interface = link.Attrs().Name
				}
			}
			routes = append(routes, ri)
		}
	}

	return routes, nil
}

// RouteInfo contains information about a route.
type RouteInfo struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway,omitempty"`
	Interface   string `json:"interface,omitempty"`
	Family      string `json:"family"` // "inet" or "inet6"
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
