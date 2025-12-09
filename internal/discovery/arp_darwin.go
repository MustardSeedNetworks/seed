//go:build darwin

package discovery

import (
	"bufio"
	"os/exec"
	"strings"
	"time"
)

// readARPTablePlatform reads the ARP table on macOS using exec commands.
func (s *ARPScanner) readARPTablePlatform() ([]*ARPEntry, error) {
	cmd := exec.Command("arp", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var entries []*ARPEntry
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		// Format: hostname (IP) at MAC on interface [ifscope ...]
		// Example: ? (192.168.1.1) at 00:11:22:33:44:55 on en0 ifscope [ethernet]

		entry := s.parseARPLineDarwin(line)
		if entry != nil && s.isInSubnet(entry.IP) {
			entries = append(entries, entry)
		}
	}

	return entries, scanner.Err()
}

// parseARPLineDarwin parses a single ARP line on macOS.
func (s *ARPScanner) parseARPLineDarwin(line string) *ARPEntry {
	// Find IP in parentheses
	start := strings.Index(line, "(")
	end := strings.Index(line, ")")
	if start < 0 || end < 0 || end <= start {
		return nil
	}
	ip := line[start+1 : end]

	// Find MAC after "at "
	atIdx := strings.Index(line, " at ")
	if atIdx < 0 {
		return nil
	}
	rest := line[atIdx+4:]

	// MAC is next token
	parts := strings.Fields(rest)
	if len(parts) < 1 {
		return nil
	}
	mac := parts[0]

	// Skip incomplete entries
	if mac == "(incomplete)" {
		return nil
	}

	// Find interface after "on "
	var iface string
	onIdx := strings.Index(rest, " on ")
	if onIdx >= 0 {
		ifaceParts := strings.Fields(rest[onIdx+4:])
		if len(ifaceParts) > 0 {
			iface = ifaceParts[0]
		}
	}

	// Normalize MAC format
	mac = normalizeMac(mac)

	return &ARPEntry{
		IP:        ip,
		MAC:       mac,
		Interface: iface,
		State:     "REACHABLE",
		LastSeen:  time.Now(),
	}
}
