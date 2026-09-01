//go:build linux

package dns

// getSystemDNSPlatform reads DNS servers on Linux from /etc/resolv.conf
// and systemd-resolved config files.
func getSystemDNSPlatform() []string {
	servers := []string{}

	// First try /etc/resolv.conf
	if s := parseResolvConf(resolvConfPath); len(s) > 0 {
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
