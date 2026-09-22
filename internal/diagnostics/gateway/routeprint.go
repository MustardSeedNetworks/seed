package gateway

// routeprint.go parses the text of Windows' `route print`. It carries no build
// tag because it is pure text handling with no Windows API in it: the syscall
// side stays in gateway_windows.go, and keeping the parser here is what lets
// its table tests run on every platform rather than only where they cannot be
// executed.

import (
	"net"
	"net/netip"
	"strconv"
	"strings"
)

// The two tables route print prints have different columns, so they are
// indexed separately.
//
//	route print -4:  Network Destination  Netmask  Gateway  Interface  Metric
//	route print -6:  If  Metric  Network Destination  Gateway
//
// Reading the -6 table with the -4 layout is how the destination came back as
// an interface index (seed#2765).
const (
	v4FieldDestination = 0
	v4FieldNetmask     = 1
	v4FieldGateway     = 2
	v4FieldInterface   = 3
	v4FieldCount       = 4

	v6FieldIndex       = 0
	v6FieldDestination = 2
	v6FieldGateway     = 3
	v6FieldCount       = 4
)

// onLink is what route print writes in the gateway column of a connected
// route. It is not an address, and a caller looking for a gateway to talk to
// must not be handed it.
const onLink = "On-link"

// interfaceNamer resolves the interface a route print row names to its name.
// The IPv4 table names it by the interface's own local address and the IPv6
// table by its index, and neither is a name — which is the half of seed#2765
// that netif.rollback's defaultGatewayFor trips over. It is an interface so
// the parser can be driven from a table test on any platform.
type interfaceNamer interface {
	nameByAddress(addr string) string
	nameByIndex(index int) string
}

// parseRouteOutput parses Windows route print output.
func parseRouteOutput(output, family string, namer interfaceNamer) []RouteInfo {
	var routes []RouteInfo

	lines := strings.Split(output, "\n")
	inActiveRoutes := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for "Active Routes:" section
		if strings.Contains(line, "Active Routes") {
			inActiveRoutes = true
			continue
		}

		// End of routes section
		if inActiveRoutes && (strings.Contains(line, "Persistent Routes") || line == "") {
			continue
		}

		if !inActiveRoutes {
			continue
		}

		// Skip header line
		if strings.Contains(line, "Network Destination") || strings.Contains(line, "Metric") {
			continue
		}

		// Parse route line
		fields := strings.Fields(line)
		if ri, ok := parseRouteFields(fields, family, namer); ok {
			routes = append(routes, ri)
		}
	}

	return routes
}

// parseRouteFields builds a RouteInfo from one route print data line's
// whitespace-separated fields. The second return is false when the line does
// not describe a route in the given family.
func parseRouteFields(fields []string, family string, namer interfaceNamer) (RouteInfo, bool) {
	switch family {
	case "inet":
		return parseIPv4RouteFields(fields, namer)
	case "inet6":
		return parseIPv6RouteFields(fields, namer)
	}
	return RouteInfo{}, false
}

// parseIPv4RouteFields reads one row of `route print -4`, whose destination
// and netmask are separate columns and whose interface column holds the
// interface's own local address.
func parseIPv4RouteFields(fields []string, namer interfaceNamer) (RouteInfo, bool) {
	if len(fields) < v4FieldCount {
		return RouteInfo{}, false
	}
	destination := net.ParseIP(fields[v4FieldDestination])
	mask := net.ParseIP(fields[v4FieldNetmask])
	if destination.To4() == nil || mask.To4() == nil {
		return RouteInfo{}, false
	}
	prefix, ok := prefixFromMask(mask.To4())
	if !ok {
		return RouteInfo{}, false
	}

	ri := RouteInfo{
		Destination: destination.String(),
		Prefix:      prefix,
		Interface:   namer.nameByAddress(fields[v4FieldInterface]),
		Family:      "inet",
	}
	if gw := fields[v4FieldGateway]; gw != onLink && net.ParseIP(gw) != nil {
		ri.Gateway = gw
	}
	return ri, true
}

// parseIPv6RouteFields reads one row of `route print -6`, whose destination is
// already a CIDR and whose interface is named by index in the first column.
func parseIPv6RouteFields(fields []string, namer interfaceNamer) (RouteInfo, bool) {
	if len(fields) < v6FieldCount {
		return RouteInfo{}, false
	}
	destination, err := netip.ParsePrefix(fields[v6FieldDestination])
	if err != nil || !destination.Addr().Is6() {
		return RouteInfo{}, false
	}

	ri := RouteInfo{
		Destination: destination.Masked().Addr().String(),
		Prefix:      destination.Bits(),
		Family:      "inet6",
	}
	if index, convErr := strconv.Atoi(fields[v6FieldIndex]); convErr == nil {
		ri.Interface = namer.nameByIndex(index)
	}
	if gw := fields[v6FieldGateway]; gw != onLink && net.ParseIP(gw) != nil {
		ri.Gateway = gw
	}
	return ri, true
}
