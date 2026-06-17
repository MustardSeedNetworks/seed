package enumerate

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

// readARPTable reads the system ARP table.
// Uses netlink on Linux, exec commands on macOS.
func (s *ARPScanner) readARPTable() ([]*ARPEntry, error) {
	return s.readARPTablePlatform()
}

// isInSubnet checks if an IP is in the current subnet or any additional subnets.
func (s *ARPScanner) isInSubnet(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Check primary subnet
	if s.subnet != nil && s.subnet.Contains(ip) {
		return true
	}

	// Check additional subnets
	for _, subnet := range s.additionalSubnets {
		if subnet.Contains(ip) {
			return true
		}
	}

	// If no subnets configured, accept all (fallback)
	return s.subnet == nil && len(s.additionalSubnets) == 0
}

// isInLocalSubnet checks if an IP is in the PRIMARY subnet only (not additional subnets).
// This is used to determine if a device should be shown in "Local Network" vs "Extended Networks".
func (s *ARPScanner) isInLocalSubnet(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Only check primary subnet - additional subnets are "extended" networks
	return s.subnet != nil && s.subnet.Contains(ip)
}

// enrichEntries adds OUI lookups, hostname resolution, and TTL-based OS guessing.
func (s *ARPScanner) enrichEntries(ctx context.Context, entries []*ARPEntry) {
	s.mu.Lock()

	// Copy pingResults under lock to avoid data race (fixes #819)
	pingResults := s.pingResults

	// Clear old entries not in current scan
	newEntries := make(map[string]*ARPEntry)

	for _, entry := range entries {
		// OUI lookup - but first check if it's a locally administered address
		if s.oui != nil {
			if isLocallyAdministeredMAC(entry.MAC) {
				entry.Vendor = "LAA"
			} else {
				entry.Vendor = s.oui.LookupWithDefault(entry.MAC, "Unknown")
			}
		}

		// Use TTL from cached ping results (already collected during ping sweep)
		if pr, ok := pingResults[entry.IP]; ok && pr.Reachable && pr.TTL > 0 {
			entry.TTL = pr.TTL
			entry.OSGuess = guessOSFromTTL(pr.TTL)
			entry.ResponseTime = pr.RTT.Milliseconds()
		}

		newEntries[entry.IP] = entry
	}

	s.entries = newEntries
	s.mu.Unlock()

	// Hostname resolution with WaitGroup to prevent goroutine leak (fixes #823)
	var wg sync.WaitGroup
	for _, entry := range entries {
		wg.Add(1)
		go func(e *ARPEntry) {
			defer wg.Done()

			resolveCtx, cancel := context.WithTimeout(ctx, hostnameResolveMs*time.Millisecond)
			defer cancel()

			names, err := net.DefaultResolver.LookupAddr(resolveCtx, e.IP)
			if err == nil && len(names) > 0 {
				s.mu.Lock()
				if existing, ok := s.entries[e.IP]; ok {
					existing.Hostname = strings.TrimSuffix(names[0], ".")
				}
				s.mu.Unlock()
			}
		}(entry)
	}
	wg.Wait()
}

// guessOSFromTTL makes a rough OS guess based on default TTL values.
func guessOSFromTTL(ttl int) string {
	// Common default TTL values:
	// 64: Linux, macOS, iOS, Android, FreeBSD
	// 128: Windows
	// 255: Cisco IOS, Solaris, some network devices
	// 60: HP-UX
	// 30: Some older network devices

	switch {
	case ttl <= ttlThresholdLow:
		return "Network Device (Low TTL)"
	case ttl <= ttlThresholdLinux:
		return "Linux/macOS/Unix"
	case ttl <= ttlThresholdWindows:
		return "Windows"
	case ttl <= ttlThresholdNetwork:
		return "Network Device/Cisco"
	default:
		return "Unknown"
	}
}

// normalizeMac converts MAC to uppercase colon-separated format.
func normalizeMac(mac string) string {
	mac = strings.ToUpper(mac)
	mac = strings.ReplaceAll(mac, "-", ":")
	return mac
}
