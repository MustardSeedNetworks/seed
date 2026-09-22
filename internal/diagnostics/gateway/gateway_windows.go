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
			if len(fields) > v4FieldGateway {
				gw := fields[v4FieldGateway]
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
		routes = append(routes, parseRouteOutput(string(output), "inet", systemInterfaceNamer{})...)
	}

	// Get IPv6 routes
	output, err = exec.CommandContext(ctx, "route", "print", "-6").Output()
	if err == nil {
		routes = append(routes, parseRouteOutput(string(output), "inet6", systemInterfaceNamer{})...)
	}

	return routes, nil
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
		if len(fields) > v4FieldInterface {
			return interfaceNameForIP(fields[v4FieldInterface]), nil
		}
	}

	return "", nil
}

// interfaceNameForIP returns the name of the local interface bound to ifaceIP,
// or ifaceIP itself as a fallback when no interface matches (or the
// interface list can't be read) -- the IP is still a usable, if less
// friendly, answer to what the outbound interface is.
func interfaceNameForIP(ifaceIP string) string {
	if name := (systemInterfaceNamer{}).nameByAddress(ifaceIP); name != "" {
		return name
	}
	return ifaceIP
}

// systemInterfaceNamer resolves route print's interface columns against the
// host's own interfaces. It answers "" rather than echoing the column back:
// RouteInfo.Interface is a name or nothing, because a caller matching it
// against net.Interface.Name is what seed#2765 is about.
type systemInterfaceNamer struct{}

func (systemInterfaceNamer) nameByAddress(addr string) string {
	want := net.ParseIP(addr)
	if want == nil {
		return ""
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for i := range ifaces {
		addrs, addrErr := ifaces[i].Addrs()
		if addrErr != nil {
			continue
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && ipnet.IP.Equal(want) {
				return ifaces[i].Name
			}
		}
	}
	return ""
}

func (systemInterfaceNamer) nameByIndex(index int) string {
	iface, err := net.InterfaceByIndex(index)
	if err != nil {
		return ""
	}
	return iface.Name
}
