//go:build darwin

package gateway

import (
	"net"
	"os/exec"
	"strings"
)

// RouteInfo contains information about a route.
type RouteInfo struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway,omitempty"`
	Interface   string `json:"interface,omitempty"`
	Family      string `json:"family"` // "inet" or "inet6"
}

// GetAllRoutes returns all routes (stub for macOS - uses exec).
func GetAllRoutes() ([]RouteInfo, error) {
	// On macOS, we'd need to parse netstat -rn output
	// For now, return empty slice - can be implemented later
	return nil, nil
}

// GetDefaultGatewayInterface returns the interface used for the default route.
func GetDefaultGatewayInterface() (string, error) {
	cmd := exec.Command("route", "-n", "get", "default")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "interface:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "interface:")), nil
		}
	}

	return "", nil
}

// isNetlinkAvailable returns false on macOS where netlink is not available.
func isNetlinkAvailable() bool {
	return false
}

// detectGatewayPlatform is the platform-specific gateway detection.
// On macOS, this uses exec commands (netstat/route).
func detectGatewayPlatform() (string, error) {
	// Use netstat -rn to get the default route
	cmd := exec.Command("netstat", "-rn")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output looking for default route
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "default" {
			gateway := fields[1]
			if net.ParseIP(gateway) != nil {
				return gateway, nil
			}
		}
	}

	// Try route -n get default
	cmd = exec.Command("route", "-n", "get", "default")
	output, err = cmd.Output()
	if err != nil {
		return "", err
	}

	lines = strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "gateway:") {
			gateway := strings.TrimSpace(strings.TrimPrefix(line, "gateway:"))
			if net.ParseIP(gateway) != nil {
				return gateway, nil
			}
		}
	}

	return "", nil
}

// detectGatewayIPv6Platform is the platform-specific IPv6 gateway detection.
// On macOS, this uses exec commands.
func detectGatewayIPv6Platform() (string, error) {
	cmd := exec.Command("netstat", "-rn", "-f", "inet6")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "default" {
			gateway := fields[1]
			// Remove interface scope suffix (e.g., %en0)
			if idx := strings.Index(gateway, "%"); idx > 0 {
				gateway = gateway[:idx]
			}
			if ip := net.ParseIP(gateway); ip != nil && ip.To4() == nil {
				return gateway, nil
			}
		}
	}

	return "", nil
}
