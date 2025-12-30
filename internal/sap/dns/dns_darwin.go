//go:build darwin

// Package dns provides DNS testing and lookup functionality with timing.
// macOS implementation uses scutil to query System Configuration framework
// for DNS resolver configuration and nameserver discovery.
package dns

import (
	"bufio"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// getSystemDNSPlatform reads DNS servers on macOS from resolver config files.
// This reads the resolver configuration directly instead of calling scutil.
func getSystemDNSPlatform() []string {
	servers := []string{}
	seen := make(map[string]bool)

	// Read from /etc/resolv.conf first
	if s := parseResolvConfDarwin("/etc/resolv.conf"); len(s) > 0 {
		for _, server := range s {
			if !seen[server] {
				seen[server] = true
				servers = append(servers, server)
			}
		}
	}

	// Read from /etc/resolver/* for additional DNS configs
	resolverDir := "/etc/resolver"
	entries, err := os.ReadDir(resolverDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(resolverDir, entry.Name())
			for _, server := range parseResolvConfDarwin(path) {
				if !seen[server] {
					seen[server] = true
					servers = append(servers, server)
				}
			}
		}
	}

	// If we still have no servers, try to get from network interface config
	if len(servers) == 0 {
		servers = getDNSFromInterfaces()
	}

	return servers
}

// parseResolvConfDarwin reads nameserver entries from a resolv.conf-style file.
func parseResolvConfDarwin(path string) []string {
	servers := []string{}

	//nolint:gosec // G304: path is system resolv.conf path from known locations
	file, err := os.Open(path)
	if err != nil {
		return servers
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}
		// Parse nameserver lines
		if strings.HasPrefix(line, "nameserver") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				servers = append(servers, parts[1])
			}
		}
	}

	return servers
}

// getDNSFromInterfaces tries to get DNS servers from network interfaces.
// On macOS, this is a fallback when resolver files don't have info.
func getDNSFromInterfaces() []string {
	servers := []string{}

	// Use Go's net package to try common DNS patterns
	// Get default gateway network and use common router DNS addresses
	ifaces, err := net.Interfaces()
	if err != nil {
		return servers
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil {
				continue
			}

			// Common router addresses that might be DNS servers
			// This is a heuristic - typically .1 on the subnet is the router
			network := ipNet.IP.Mask(ipNet.Mask)
			routerIP := make(net.IP, len(network))
			copy(routerIP, network)
			routerIP[len(routerIP)-1] = 1

			// We don't actually add these as we can't verify they're DNS servers
			// This function is just a placeholder for future enhancement
			_ = routerIP
		}
	}

	return servers
}
