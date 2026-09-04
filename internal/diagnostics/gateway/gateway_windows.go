//go:build windows

package gateway

// Windows-specific gateway detection implementation.
// Uses netsh and route commands to detect default IPv4 and IPv6 gateways.

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
)

// Command timeout for netsh operations.
const netshTimeoutSeconds = 15

// colonSplitParts caps SplitN at 2 pieces when parsing "key: value"
// netsh output lines, so a value containing ':' is not split further.
const colonSplitParts = 2

// route print's "Network Destination Netmask Gateway Interface Metric"
// columns: routeFieldsMinimal is enough to reach the Gateway column
// (fields[2]), routeFieldsWithInterface enough to also reach Interface
// (fields[3]).
const (
	routeFieldsMinimal       = 3
	routeFieldsWithInterface = 4
)

// detectGatewayPlatform detects the default IPv4 gateway on Windows.
func detectGatewayPlatform() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), netshTimeoutSeconds*time.Second)
	defer cancel()

	// Use route print to get routing table
	output, err := exec.CommandContext(ctx, "route", "print", "-4").Output()
	if err != nil {
		// Fallback to netsh
		return detectGatewayNetsh(ctx)
	}

	// Parse output for default route (0.0.0.0)
	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		// Look for lines starting with "0.0.0.0" (default route)
		if strings.HasPrefix(line, "0.0.0.0") {
			fields := strings.Fields(line)
			// Format: Network Destination    Netmask         Gateway         Interface   Metric
			if len(fields) >= routeFieldsMinimal {
				gw := fields[2]
				// Validate it's an IP
				if net.ParseIP(gw) != nil && gw != "0.0.0.0" {
					return gw, nil
				}
			}
		}
	}

	return "", nil
}

// detectGatewayNetsh uses netsh as fallback for gateway detection.
func detectGatewayNetsh(ctx context.Context) (string, error) {
	output, err := exec.CommandContext(ctx, "netsh", "interface", "ip", "show", "config").Output()
	if err != nil {
		return "", fmt.Errorf("netsh failed: %w", err)
	}

	// Parse for "Default Gateway" line
	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Default Gateway") || strings.Contains(line, "デフォルト ゲートウェイ") {
			parts := strings.SplitN(line, ":", colonSplitParts)
			if len(parts) == colonSplitParts {
				gw := strings.TrimSpace(parts[1])
				if gw != "" && net.ParseIP(gw) != nil {
					return gw, nil
				}
			}
		}
	}

	return "", nil
}

// detectGatewayIPv6Platform detects the default IPv6 gateway on Windows.
func detectGatewayIPv6Platform() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), netshTimeoutSeconds*time.Second)
	defer cancel()

	// Use route print for IPv6
	output, err := exec.CommandContext(ctx, "route", "print", "-6").Output()
	if err != nil {
		// Fallback to netsh
		return detectGatewayIPv6Netsh(ctx)
	}

	// Parse output for default route (::/0)
	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		// Look for lines with ::/0 (default route) or starting with ::
		if strings.Contains(line, "::/0") || strings.HasPrefix(line, "::") {
			if gw, ok := findIPv6GatewayInLine(line); ok {
				return gw, nil
			}
		}
	}

	return "", nil
}

// findIPv6GatewayInLine scans a route-print line's whitespace-separated
// fields for a non-link-local IPv6 address, the gateway column's shape.
func findIPv6GatewayInLine(line string) (string, bool) {
	for field := range strings.FieldsSeq(line) {
		// Find field that looks like an IPv6 address (contains ::)
		if !strings.Contains(field, ":") || strings.HasPrefix(field, "::") {
			continue
		}
		ip := net.ParseIP(field)
		if ip == nil || ip.To4() != nil {
			continue
		}
		if !ip.IsLinkLocalUnicast() {
			continue // route lines list the destination prefix too; only the
			// link-local address is the gateway's own next-hop
		}
		return ip.String(), true
	}
	return "", false
}

// detectGatewayIPv6Netsh uses netsh for IPv6 gateway detection.
func detectGatewayIPv6Netsh(ctx context.Context) (string, error) {
	output, err := exec.CommandContext(ctx, "netsh", "interface", "ipv6", "show", "route").Output()
	if err != nil {
		return "", fmt.Errorf("netsh ipv6 failed: %w", err)
	}

	// Parse for default route (::)
	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "::/0") {
			fields := strings.FieldsSeq(line)
			// Look for gateway address in fields
			for field := range fields {
				ip := net.ParseIP(field)
				if ip != nil && ip.To4() == nil {
					return ip.String(), nil
				}
			}
		}
	}

	return "", nil
}

// GetAllRoutes returns all routes using route print (for debugging/display).
func GetAllRoutes() ([]RouteInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), netshTimeoutSeconds*time.Second)
	defer cancel()

	var routes []RouteInfo

	// Get IPv4 routes
	output, err := exec.CommandContext(ctx, "route", "print", "-4").Output()
	if err == nil {
		routes = append(routes, parseRouteOutput(string(output), "inet")...)
	}

	// Get IPv6 routes
	output, err = exec.CommandContext(ctx, "route", "print", "-6").Output()
	if err == nil {
		routes = append(routes, parseRouteOutput(string(output), "inet6")...)
	}

	return routes, nil
}

// parseRouteOutput parses Windows route print output.
func parseRouteOutput(output, family string) []RouteInfo {
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
		if len(fields) < routeFieldsMinimal {
			continue
		}
		if ri, ok := parseRouteFields(fields, family); ok {
			routes = append(routes, ri)
		}
	}

	return routes
}

// parseRouteFields builds a RouteInfo from one route print data line's
// whitespace-separated fields, already known to have at least
// routeFieldsMinimal entries. The second return is false when the line
// didn't carry a destination for the given family (nothing to add).
func parseRouteFields(fields []string, family string) (RouteInfo, bool) {
	ri := RouteInfo{
		Family: family,
	}

	switch {
	case family == "inet" && len(fields) >= routeFieldsWithInterface:
		ri.Destination = fields[0]
		if fields[0] == "0.0.0.0" {
			ri.Destination = "default"
		}
		ri.Gateway = fields[2]
		ri.Interface = fields[3]
	case family == "inet6" && len(fields) >= routeFieldsMinimal:
		ri.Destination = fields[0]
		if fields[0] == "::/0" {
			ri.Destination = "default"
		}
		// IPv6 format varies
		for _, f := range fields[1:] {
			ip := net.ParseIP(f)
			if ip != nil && ip.To4() == nil {
				ri.Gateway = f
				break
			}
		}
	}

	return ri, ri.Destination != ""
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
	ctx, cancel := context.WithTimeout(context.Background(), netshTimeoutSeconds*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, "route", "print", "-4", "0.0.0.0").Output()
	if err != nil {
		return "", fmt.Errorf("route print failed: %w", err)
	}

	// Parse for interface
	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "0.0.0.0") {
			continue
		}
		fields := strings.Fields(line)
		// Format: Network Destination    Netmask         Gateway         Interface   Metric
		if len(fields) >= routeFieldsWithInterface {
			return interfaceNameForIP(fields[3]), nil
		}
	}

	return "", nil
}

// interfaceNameForIP returns the name of the local interface bound to ifaceIP,
// or ifaceIP itself as a fallback when no interface matches (or the
// interface list can't be read) -- the IP is still a usable, if less
// friendly, answer to what the outbound interface is.
func interfaceNameForIP(ifaceIP string) string {
	ifaces, ifacesErr := net.Interfaces()
	if ifacesErr != nil {
		return ifaceIP
	}
	for _, iface := range ifaces {
		addrs, addrsErr := iface.Addrs()
		if addrsErr != nil {
			continue
		}
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && ipNet.IP.String() == ifaceIP {
				return iface.Name
			}
		}
	}
	return ifaceIP
}
