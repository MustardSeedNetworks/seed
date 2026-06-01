//go:build windows

// Windows-specific gateway detection implementation.
// Uses netsh and route commands to detect default IPv4 and IPv6 gateways.
package gateway

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
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for lines starting with "0.0.0.0" (default route)
		if strings.HasPrefix(line, "0.0.0.0") {
			fields := strings.Fields(line)
			// Format: Network Destination    Netmask         Gateway         Interface   Metric
			if len(fields) >= 3 {
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
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Default Gateway") || strings.Contains(line, "デフォルト ゲートウェイ") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
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
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for lines with ::/0 (default route) or starting with ::
		if strings.Contains(line, "::/0") || strings.HasPrefix(line, "::") {
			fields := strings.Fields(line)
			for _, field := range fields {
				// Find field that looks like an IPv6 address (contains ::)
				if strings.Contains(field, ":") && !strings.HasPrefix(field, "::") {
					ip := net.ParseIP(field)
					if ip != nil && ip.To4() == nil && !ip.IsLinkLocalUnicast() {
						continue // Skip link-local unless it's the gateway
					}
					if ip != nil && ip.To4() == nil {
						return ip.String(), nil
					}
				}
			}
		}
	}

	return "", nil
}

// detectGatewayIPv6Netsh uses netsh for IPv6 gateway detection.
func detectGatewayIPv6Netsh(ctx context.Context) (string, error) {
	output, err := exec.CommandContext(ctx, "netsh", "interface", "ipv6", "show", "route").Output()
	if err != nil {
		return "", fmt.Errorf("netsh ipv6 failed: %w", err)
	}

	// Parse for default route (::)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "::/0") {
			fields := strings.Fields(line)
			// Look for gateway address in fields
			for _, field := range fields {
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
		if len(fields) >= 3 {
			ri := RouteInfo{
				Family: family,
			}

			if family == "inet" && len(fields) >= 4 {
				ri.Destination = fields[0]
				if fields[0] == "0.0.0.0" {
					ri.Destination = "default"
				}
				ri.Gateway = fields[2]
				if len(fields) >= 4 {
					ri.Interface = fields[3]
				}
			} else if family == "inet6" && len(fields) >= 3 {
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

			if ri.Destination != "" {
				routes = append(routes, ri)
			}
		}
	}

	return routes
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
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "0.0.0.0") {
			fields := strings.Fields(line)
			// Format: Network Destination    Netmask         Gateway         Interface   Metric
			if len(fields) >= 4 {
				ifaceIP := fields[3]
				// Find interface by IP
				ifaces, err := net.Interfaces()
				if err != nil {
					return ifaceIP, nil // Return IP as fallback
				}
				for _, iface := range ifaces {
					addrs, err := iface.Addrs()
					if err != nil {
						continue
					}
					for _, addr := range addrs {
						if ipNet, ok := addr.(*net.IPNet); ok {
							if ipNet.IP.String() == ifaceIP {
								return iface.Name, nil
							}
						}
					}
				}
				return ifaceIP, nil
			}
		}
	}

	return "", nil
}
