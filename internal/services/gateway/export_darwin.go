//go:build darwin

package gateway

import (
	"golang.org/x/net/route"
)

// ExtractRouteIP exposes extractRouteIP for testing.
func ExtractRouteIP(addr route.Addr) (string, string) {
	return extractRouteIP(addr)
}

// ParseRouteMessage exposes parseRouteMessage for testing.
func ParseRouteMessage(rm *route.RouteMessage) *RouteInfo {
	return parseRouteMessage(rm)
}

// IsDefaultRouteMsg exposes isDefaultRouteMsg for testing.
func IsDefaultRouteMsg(rm *route.RouteMessage, family int) bool {
	return isDefaultRouteMsg(rm, family)
}

// ExtractIPv6Gateway exposes extractIPv6Gateway for testing.
func ExtractIPv6Gateway(rm *route.RouteMessage, skipLinkLocal bool) string {
	return extractIPv6Gateway(rm, skipLinkLocal)
}
