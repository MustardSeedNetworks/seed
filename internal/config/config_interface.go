package config

// config_interface.go contains credential-update helpers plus the
// active-network-interface detection used at startup to pick the NIC the
// rest of the app binds to.

import (
	"net"
	"strings"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// UpdateCredentials updates the authentication credentials in the config.
func (c *Config) UpdateCredentials(username, passwordHash, jwtSecret string) {
	c.Auth.DefaultUsername = username
	c.Auth.DefaultPasswordHash = passwordHash
	if jwtSecret != "" {
		c.Auth.JWTSecret = jwtSecret
	}
}

// UpdateJWTSecret updates only the JWT secret in the config.
func (c *Config) UpdateJWTSecret(secret string) {
	c.Auth.JWTSecret = secret
}

// GetActiveInterface returns an active network interface with an IPv4 address.
// It first tries the configured default, then fallbacks, then auto-detects.
// Returns the interface name and whether fallback was used.
func (c *Config) GetActiveInterface() (string, bool) {
	// Try the configured default interface first
	if c.Interface.Default != "" {
		if hasIPv4Address(c.Interface.Default) {
			return c.Interface.Default, false
		}
		logging.GetLogger().Warn(
			"Configured interface has no IPv4 address or doesn't exist",
			"interface",
			c.Interface.Default,
		)
	}

	// Try fallback interfaces
	for _, iface := range c.Interface.Fallbacks {
		if hasIPv4Address(iface) {
			logging.GetLogger().Info("Using fallback interface", "interface", iface)
			return iface, true
		}
	}

	// Auto-detect: scan all interfaces for one with an IPv4 address
	detected := detectActiveInterface()
	if detected != "" {
		logging.GetLogger().Info("Auto-detected active interface", "interface", detected)
		return detected, true
	}

	// Last resort: return the configured default even if it might not work
	if c.Interface.Default != "" {
		logging.GetLogger().Warn(
			"No active interface found, using configured default",
			"interface",
			c.Interface.Default,
		)
		return c.Interface.Default, true
	}

	// No hardcoded fallback - return empty to signal no interface found (#572)
	logging.GetLogger().Error("No active network interface found")
	return "", false
}

// hasIPv4Address checks if an interface exists and has at least one IPv4 address.
func hasIPv4Address(ifaceName string) bool {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return false
	}

	// Check if interface is up
	if iface.Flags&net.FlagUp == 0 {
		return false
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return false
	}

	for _, addr := range addrs {
		// Check for IPv4 address (not loopback)
		if ipNet, ok := addr.(*net.IPNet); ok {
			if ipv4 := ipNet.IP.To4(); ipv4 != nil && !ipv4.IsLoopback() {
				return true
			}
		}
	}

	return false
}

// detectActiveInterface scans all network interfaces and returns the first
// non-loopback interface with an IPv4 address that is up.
func detectActiveInterface() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	// Priority order: prefer ethernet over wifi, physical over virtual
	// Pre-allocate slice with expected capacity
	candidates := make([]string, 0, len(interfaces))

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Skip virtual/bridge interfaces (common prefixes)
		name := iface.Name
		if strings.HasPrefix(name, "docker") ||
			strings.HasPrefix(name, "br-") ||
			strings.HasPrefix(name, "veth") ||
			strings.HasPrefix(name, "virbr") ||
			strings.HasPrefix(name, "vbox") {
			continue
		}

		// Check if it has an IPv4 address
		if !hasIPv4Address(name) {
			continue
		}

		candidates = append(candidates, name)
	}

	if len(candidates) == 0 {
		return ""
	}

	// Sort candidates by preference (ethernet before wifi)
	// Common ethernet: eth*, enp*, eno*, ens*
	// Common wifi: wlan*, wlp*
	for _, c := range candidates {
		if strings.HasPrefix(c, "eth") || strings.HasPrefix(c, "en") {
			return c
		}
	}

	// Return first candidate if no ethernet found
	return candidates[0]
}
