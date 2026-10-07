package enumerate

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Scan performs an active ARP scan of the network.
func (s *ARPScanner) Scan(ctx context.Context) error {
	s.mu.Lock()
	if s.scanning {
		s.mu.Unlock()
		return errors.New("scan already in progress")
	}
	s.scanning = true
	s.pingResponders = nil             // Clear previous ping responders
	targetNetworks := s.targetNetworks // Copy while holding lock
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.scanning = false
		s.lastScan = time.Now()
		s.mu.Unlock()
	}()

	// Get subnet info
	subnet, localIP, err := s.getSubnet()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.subnet = subnet
	s.localIP = localIP
	s.mu.Unlock()

	// Perform ping sweep on primary subnet with retry logic
	// Note: ping sweep may fail without CAP_NET_RAW - continue with ARP table
	result := discovery.RetryWithBackoff(ctx, discovery.NetworkRetryConfig(), func() error {
		_, sweepErr := s.pingSweep(ctx, subnet)
		return sweepErr
	})
	if !result.Successful {
		logging.GetLogger().WarnContext(ctx, "Ping sweep failed after retries (continuing with ARP table only)",
			"subnet", subnet,
			"attempts", result.Attempts,
			"duration", result.TotalTime,
			"error", result.LastError)
	}

	// Perform ping sweep on target networks with retry logic
	var targetSwept []PingResult
	for _, additionalSubnet := range targetNetworks {
		select {
		case <-ctx.Done():
			return fmt.Errorf("ping sweep cancelled: %w", ctx.Err())
		default:
			// Retry logic for target networks - continue even if some fail
			subnetCopy := additionalSubnet // Capture for closure
			var swept []PingResult
			subnetResult := discovery.RetryWithBackoff(ctx, discovery.NetworkRetryConfig(), func() error {
				var sweepErr error
				swept, sweepErr = s.pingSweep(ctx, subnetCopy)
				return sweepErr
			})
			targetSwept = append(targetSwept, swept...)
			if !subnetResult.Successful {
				logging.GetLogger().WarnContext(ctx, "Ping sweep failed for target network after retries",
					"subnet", additionalSubnet,
					"attempts", subnetResult.Attempts,
					"duration", subnetResult.TotalTime,
					"error", subnetResult.LastError)
			}
		}
	}

	// Read ARP table (will include entries from all scanned subnets)
	entries, err := s.readARPTable()
	if err != nil {
		return fmt.Errorf("failed to read ARP table: %w", err)
	}

	// Mark ARP entries based on whether they're in the primary subnet
	// Note: ARP can capture entries from target networks if they're routed through us
	for _, entry := range entries {
		entry.IsLocal = s.isInLocalSubnet(entry.IP)
	}

	// Merge ping responders that aren't in ARP table (remote subnets)
	s.mu.RLock()
	responders := make([]string, len(s.pingResponders))
	copy(responders, s.pingResponders)
	s.mu.RUnlock()

	// Create a map of existing IPs from ARP
	existingIPs := make(map[string]bool)
	for _, entry := range entries {
		existingIPs[entry.IP] = true
	}

	// Add ping responders not in ARP table
	// These could be local (in primary subnet) or from additional/extended subnets
	for _, ip := range responders {
		if !existingIPs[ip] {
			entries = append(entries, &ARPEntry{
				IP:       ip,
				MAC:      "", // No MAC - either not in ARP cache or remote host
				State:    "PING_ONLY",
				LastSeen: time.Now(),
				IsLocal:  s.isInLocalSubnet(ip), // Only primary subnet is "local"
			})
		}
	}

	entries = append(entries, s.probeSilentTargets(ctx, targetSwept, targetNetworks, existingIPs)...)

	// Enrich entries with OUI lookup and hostname resolution
	s.enrichEntries(ctx, entries)

	return nil
}

// pingSweep sends ICMP echo requests to the hosts of subnet using raw sockets.
// A subnet wider than the per-sweep host cap is swept in the /24s planSweep
// picks for it, so successive sweeps rotate through it (seed#2832).
func (s *ARPScanner) pingSweep(ctx context.Context, subnet *net.IPNet) ([]PingResult, error) {
	target, ok := ipv4Prefix(subnet)
	if !ok || target.Bits() >= cidrMask24 {
		return s.pingSweepChunk(ctx, subnet)
	}

	maxHosts := s.GetMaxHostsPerSubnet()
	s.mu.Lock()
	evidence := slices.Clone(s.evidence)
	if s.localIP != nil {
		if local, valid := netip.AddrFromSlice(s.localIP.To4()); valid {
			evidence = append(evidence, local)
		}
	}
	if s.rotations == nil {
		s.rotations = make(map[netip.Prefix]*rotation)
	}
	turn := s.rotations[target]
	if turn == nil {
		turn = &rotation{}
		s.rotations[target] = turn
	}
	plan := turn.planSweep(target, evidence, maxHosts)
	s.mu.Unlock()

	logging.GetLogger().InfoContext(ctx, "Sweeping part of a target network wider than one sweep",
		"subnet", target.String(),
		"blocks", len(plan.blocks),
		"firstHostsOnly", plan.probe,
		"maxHosts", maxHosts)

	hosts := plan.hosts()
	ips := make([]net.IP, 0, len(hosts))
	for _, host := range hosts {
		ips = append(ips, host.AsSlice())
	}
	results, err := s.pingHosts(ctx, ips)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	turn.commit(target, plan)
	s.mu.Unlock()
	return results, nil
}

// pingSweepChunk scans a single /24 or smaller subnet chunk.
func (s *ARPScanner) pingSweepChunk(ctx context.Context, subnet *net.IPNet) ([]PingResult, error) {
	ones, bits := subnet.Mask.Size()
	numHosts := 1<<(bits-ones) - subnetExcludeCount // Exclude network and broadcast

	// Generate host IPs
	baseIP := subnet.IP.Mask(subnet.Mask).To4()
	if baseIP == nil {
		return nil, errors.New("invalid subnet")
	}

	var ips []net.IP
	for i := 1; i <= numHosts; i++ {
		if ip := incrementIP(baseIP, i); ip != nil {
			ips = append(ips, ip)
		}
	}
	return s.pingHosts(ctx, ips)
}

// pingHosts pings ips, skipping this host's own address, records who answered
// and returns every result.
func (s *ARPScanner) pingHosts(ctx context.Context, ips []net.IP) ([]PingResult, error) {
	ips = slices.DeleteFunc(ips, func(ip net.IP) bool { return ip.Equal(s.localIP) })

	// Initialize pinger if needed (fixes #822 - check under lock)
	s.mu.Lock()
	if s.pinger == nil {
		pinger, err := NewICMPPinger(time.Second)
		if err != nil {
			s.pingerErr = err
			s.mu.Unlock()
			logging.GetLogger().WarnContext(ctx, "Failed to create ICMP pinger", "error", err)
			return nil, err
		}
		s.pinger = pinger
	}
	s.pingerErr = nil
	pinger := s.pinger // Copy reference under lock
	s.mu.Unlock()

	// Perform ping sweep using raw ICMP sockets
	results := pinger.PingSweep(ctx, ips, pingSweepWorkers)

	// Store results and track responders
	s.mu.Lock()
	if s.pingResults == nil {
		s.pingResults = make(map[string]PingResult)
	}
	for _, result := range results {
		if result.Reachable {
			s.pingResponders = append(s.pingResponders, result.IP)
		}
		s.pingResults[result.IP] = result
	}
	s.mu.Unlock()

	return results, nil
}

// probeSilentTargets asks the swept target-network addresses that answered no
// echo for SNMP, recording what the probe did (seed#2449).
func (s *ARPScanner) probeSilentTargets(
	ctx context.Context, swept []PingResult, targets []*net.IPNet, known map[string]bool,
) []*ARPEntry {
	s.mu.RLock()
	prober := s.snmpProber
	s.mu.RUnlock()

	silent := silentTargets(swept, targets, known)
	var found []*ARPEntry
	report := SNMPProbeReport{Silent: len(silent), Skipped: "no SNMP credential source"}
	if prober != nil {
		found, report = prober.probe(ctx, silent)
	}

	s.mu.Lock()
	s.snmpProbe = report
	s.mu.Unlock()
	return found
}
