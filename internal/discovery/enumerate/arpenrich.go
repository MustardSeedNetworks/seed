package enumerate

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery/resolve"
)

// readARPTable reads the system ARP table, narrowed to the scanner's
// configured scope.
//
// The platform readers return the whole kernel table; the subnet filter lives
// here rather than repeated in three build-tagged files, because #328 needs the
// same read *without* it. Discovery's behaviour is unchanged: it called this,
// and this still filters.
func (s *ARPScanner) readARPTable() ([]*ARPEntry, error) {
	entries, err := s.readARPTablePlatform()
	if err != nil {
		return nil, err
	}

	filtered := make([]*ARPEntry, 0, len(entries))
	for _, entry := range entries {
		if s.isInSubnet(entry.IP) {
			filtered = append(filtered, entry)
		}
	}

	return filtered, nil
}

// ReadNeighbourCache returns this device's own neighbour cache as the kernel
// holds it right now — every entry, unfiltered, vendor-enriched.
//
// Deliberately not the same thing as GET /api/v1/topology/arp, which serves
// SNMP-harvested bindings from remote nodes and answers "what does that switch
// think its ARP table is". This answers "what does this box see on the wire in
// front of it", which is what an operator needs when an IP is not resolving to
// a MAC on the segment they are plugged into (#328).
func (s *ARPScanner) ReadNeighbourCache() ([]*ARPEntry, error) {
	entries, err := s.readARPTablePlatform()
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		entry.Vendor = vendorFor(s.oui, entry.MAC)
	}

	return entries, nil
}

// vendorFor names the manufacturer behind a MAC, or says why it cannot.
//
// A locally administered address has no manufacturer to look up — it was
// assigned by software, which is what a randomised client MAC looks like — so
// it is labelled rather than reported as an unknown vendor.
func vendorFor(db *resolve.OUIDatabase, mac string) string {
	if mac == "" {
		return ""
	}
	// Checked before the database, not inside it: whether a MAC is locally
	// administered is a property of the address, and stays true whether or not
	// an OUI database happens to be loaded. The previous code only reached this
	// branch when one was, so a randomised client MAC on a host with no OUI
	// data was reported as having no vendor rather than as software-assigned.
	if isLocallyAdministeredMAC(mac) {
		return "LAA"
	}
	if db == nil {
		return ""
	}

	return db.LookupWithDefault(mac, "Unknown")
}

// isInSubnet checks if an IP is in the current subnet or any target networks.
func (s *ARPScanner) isInSubnet(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Check primary subnet
	if s.subnet != nil && s.subnet.Contains(ip) {
		return true
	}

	// Check target networks
	for _, subnet := range s.targetNetworks {
		if subnet.Contains(ip) {
			return true
		}
	}

	// If no subnets configured, accept all (fallback)
	return s.subnet == nil && len(s.targetNetworks) == 0
}

// isInLocalSubnet checks if an IP is in the PRIMARY subnet only (not target networks).
// This is used to determine if a device should be shown in "Local Network" vs "Extended Networks".
func (s *ARPScanner) isInLocalSubnet(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// Only check primary subnet - target networks are "extended" networks
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
		entry.Vendor = vendorFor(s.oui, entry.MAC)

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
