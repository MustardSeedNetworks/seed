//go:build windows

// Windows-specific DNS server detection implementation.
// Uses netsh and ipconfig commands to read configured DNS servers.
package dns

import (
	"context"
	"net"
	"os/exec"
	"strings"
	"time"
)

// Command timeout for DNS operations.
const dnsTimeoutSeconds = 10

// getSystemDNSPlatform reads DNS servers on Windows using netsh and ipconfig.
func getSystemDNSPlatform() []string {
	ctx, cancel := context.WithTimeout(context.Background(), dnsTimeoutSeconds*time.Second)
	defer cancel()

	// Try ipconfig /all first (most reliable)
	if servers := getDNSFromIPConfig(ctx); len(servers) > 0 {
		return servers
	}

	// Fallback to netsh
	return getDNSFromNetsh(ctx)
}

// getDNSFromIPConfig extracts DNS servers from ipconfig /all output.
func getDNSFromIPConfig(ctx context.Context) []string {
	output, err := exec.CommandContext(ctx, "ipconfig", "/all").Output()
	if err != nil {
		return nil
	}

	servers := []string{}
	seen := make(map[string]bool)
	lines := strings.Split(string(output), "\n")

	inDNSSection := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for DNS Servers line
		if strings.Contains(line, "DNS Servers") || strings.Contains(line, "DNS サーバー") {
			inDNSSection = true
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				server := strings.TrimSpace(parts[1])
				if isValidDNSServer(server) && !seen[server] {
					servers = append(servers, server)
					seen[server] = true
				}
			}
			continue
		}

		// Additional DNS servers are on subsequent lines (indented)
		if inDNSSection {
			// Check if line is just an IP address (continuation of DNS list)
			if isValidDNSServer(line) && !seen[line] {
				servers = append(servers, line)
				seen[line] = true
			} else if line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				// End of DNS section if non-indented, non-empty line
				inDNSSection = false
			}
		}
	}

	return servers
}

// getDNSFromNetsh extracts DNS servers from netsh output.
func getDNSFromNetsh(ctx context.Context) []string {
	output, err := exec.CommandContext(ctx, "netsh", "interface", "ip", "show", "dns").Output()
	if err != nil {
		return nil
	}

	servers := []string{}
	seen := make(map[string]bool)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for "Statically Configured DNS Servers" or "DHCP" sections
		if strings.Contains(line, "DNS") || strings.Contains(line, "dns") {
			continue // Skip header lines
		}

		// Check if line contains an IP address
		fields := strings.Fields(line)
		for _, field := range fields {
			if isValidDNSServer(field) && !seen[field] {
				servers = append(servers, field)
				seen[field] = true
			}
		}
	}

	return servers
}

// isValidDNSServer checks if the string is a valid DNS server IP.
func isValidDNSServer(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	// Must be a valid IP address
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}

	// Skip loopback (unless it's a DNS stub like systemd-resolved, which doesn't apply to Windows)
	if ip.IsLoopback() {
		return false
	}

	// Skip link-local
	if ip.IsLinkLocalUnicast() {
		return false
	}

	return true
}
