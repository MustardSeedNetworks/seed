//go:build linux

// Package dns provides DNS testing and lookup functionality with timing.
// Linux implementation reads /etc/resolv.conf to discover configured name servers
// and performs DNS queries for nameserver enumeration.
package dns

import (
	"bufio"
	"os"
	"strings"
)

// getSystemDNSPlatform reads DNS servers on Linux from /etc/resolv.conf
// and systemd-resolved config files.
func getSystemDNSPlatform() []string {
	servers := []string{}

	// First try /etc/resolv.conf
	if s := parseResolvConf("/etc/resolv.conf"); len(s) > 0 {
		// If only systemd-resolved stub is found, try to get real servers
		if len(s) == 1 && s[0] == "127.0.0.53" {
			if realServers := getSystemdResolvedDNS(); len(realServers) > 0 {
				return realServers
			}
		}
		return s
	}

	return servers
}

// parseResolvConf reads nameserver entries from a resolv.conf file.
func parseResolvConf(path string) []string {
	servers := []string{}

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

// getSystemdResolvedDNS reads DNS servers from systemd-resolved config files.
// This avoids the need to exec resolvectl or systemd-resolve.
func getSystemdResolvedDNS() []string {
	servers := []string{}
	seen := make(map[string]bool)

	// Try systemd-resolved's own resolv.conf which has the upstream servers
	resolvedPaths := []string{
		"/run/systemd/resolve/resolv.conf", // Upstream DNS servers
		"/run/systemd/resolve/stub-resolv.conf",
	}

	for _, path := range resolvedPaths {
		for _, server := range parseResolvConf(path) {
			// Skip the stub resolver
			if server == "127.0.0.53" || server == "127.0.0.54" {
				continue
			}
			if !seen[server] {
				seen[server] = true
				servers = append(servers, server)
			}
		}
	}

	// Also check for NetworkManager managed resolv.conf
	if len(servers) == 0 {
		for _, server := range parseResolvConf("/var/run/NetworkManager/resolv.conf") {
			if !seen[server] {
				seen[server] = true
				servers = append(servers, server)
			}
		}
	}

	return servers
}
