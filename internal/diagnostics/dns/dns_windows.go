//go:build windows

package dns

// Windows-specific DNS server detection implementation.
// Uses netsh and ipconfig commands to read configured DNS servers.

import (
	"context"
	"net"
	"os/exec"
	"strings"
	"time"
)

// Command timeout for DNS operations.
const dnsTimeoutSeconds = 10

// colonSplitParts caps SplitN at 2 pieces when parsing "key: value"
// netsh output lines, so a value containing ':' is not split further.
const colonSplitParts = 2

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
			addDNSServerFromLabelledLine(line, &servers, seen)
			continue
		}

		// Additional DNS servers are on subsequent lines (indented)
		if inDNSSection {
			inDNSSection = addDNSServerFromContinuationLine(line, &servers, seen)
		}
	}

	return servers
}

// addDNSServerFromLabelledLine parses the "DNS Servers: <ip>" line itself,
// appending the address to servers when it is a new, valid one.
func addDNSServerFromLabelledLine(line string, servers *[]string, seen map[string]bool) {
	parts := strings.SplitN(line, ":", colonSplitParts)
	if len(parts) != colonSplitParts {
		return
	}
	server := strings.TrimSpace(parts[1])
	if isValidDNSServer(server) && !seen[server] {
		*servers = append(*servers, server)
		seen[server] = true
	}
}

// addDNSServerFromContinuationLine handles one of the indented lines that can
// follow "DNS Servers:", each carrying one more address. It returns whether
// the DNS section is still open: a further address line keeps it open, a
// non-indented, non-empty line closes it.
func addDNSServerFromContinuationLine(line string, servers *[]string, seen map[string]bool) bool {
	if isValidDNSServer(line) && !seen[line] {
		*servers = append(*servers, line)
		seen[line] = true
		return true
	}
	if line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
		return false
	}
	return true
}

// getDNSFromNetsh extracts DNS servers from netsh output.
func getDNSFromNetsh(ctx context.Context) []string {
	output, err := exec.CommandContext(ctx, "netsh", "interface", "ip", "show", "dns").Output()
	if err != nil {
		return nil
	}

	servers := []string{}
	seen := make(map[string]bool)
	lines := strings.SplitSeq(string(output), "\n")

	for line := range lines {
		line = strings.TrimSpace(line)

		// Look for "Statically Configured DNS Servers" or "DHCP" sections
		if strings.Contains(line, "DNS") || strings.Contains(line, "dns") {
			continue // Skip header lines
		}

		// Check if line contains an IP address
		fields := strings.FieldsSeq(line)
		for field := range fields {
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
