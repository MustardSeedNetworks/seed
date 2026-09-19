//go:build linux

package gateway

import "github.com/vishvananda/netlink"

// AppendRouteInfo exposes appendRouteInfo for testing.
func AppendRouteInfo(result []RouteInfo, routes []netlink.Route, family string) []RouteInfo {
	return appendRouteInfo(result, routes, family)
}
