package enumerate

import (
	"context"
	"errors"
	"fmt"
	"net"
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
	s.pingResponders = nil                   // Clear previous ping responders
	additionalSubnets := s.additionalSubnets // Copy while holding lock
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
		return s.pingSweep(ctx, subnet)
	})
	if !result.Successful {
		logging.GetLogger().WarnContext(ctx, "Ping sweep failed after retries (continuing with ARP table only)",
			"subnet", subnet,
			"attempts", result.Attempts,
			"duration", result.TotalTime,
			"error", result.LastError)
	}

	// Perform ping sweep on additional subnets with retry logic
	for _, additionalSubnet := range additionalSubnets {
		select {
		case <-ctx.Done():
			return fmt.Errorf("ping sweep cancelled: %w", ctx.Err())
		default:
			// Retry logic for additional subnets - continue even if some fail
			subnetCopy := additionalSubnet // Capture for closure
			subnetResult := discovery.RetryWithBackoff(ctx, discovery.NetworkRetryConfig(), func() error {
				return s.pingSweep(ctx, subnetCopy)
			})
			if !subnetResult.Successful {
				logging.GetLogger().WarnContext(ctx, "Ping sweep failed for additional subnet after retries",
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
	// Note: ARP can capture entries from additional subnets if they're routed through us
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

	// Enrich entries with OUI lookup and hostname resolution
	s.enrichEntries(ctx, entries)

	return nil
}

// MaxChunksDefault is the default maximum number of /24 chunks to scan.
// This provides a safety guardrail for very large CIDRs like /8.
// 256 chunks = /16 subnet = 65,534 hosts (reasonable upper bound).
const MaxChunksDefault = 256

// splitSubnetIntoChunks splits a large subnet into /24 chunks for manageable scanning.
// Returns the original subnet in a slice if it's already /24 or smaller.
// maxChunks limits the number of chunks to prevent memory/time issues with huge CIDRs.
// Pass 0 for maxChunks to use MaxChunksDefault.
func splitSubnetIntoChunks(subnet *net.IPNet, maxChunks int) []*net.IPNet {
	ones, bits := subnet.Mask.Size()
	if bits != cidrBits32 {
		// IPv6 or invalid - return as-is
		return []*net.IPNet{subnet}
	}

	// /24 or smaller - no need to chunk
	if ones >= cidrMask24 {
		return []*net.IPNet{subnet}
	}

	// Calculate number of /24 chunks needed
	numChunks := 1 << (cidrMask24 - ones) // e.g., /22 = 4 chunks, /16 = 256 chunks

	// Apply safety cap
	if maxChunks <= 0 {
		maxChunks = MaxChunksDefault
	}
	if numChunks > maxChunks {
		logging.GetLogger().Warn("Subnet too large - capping chunk count",
			"subnet", subnet.String(),
			"totalChunks", numChunks,
			"maxChunks", maxChunks,
			"coverage", fmt.Sprintf("%.1f%%", float64(maxChunks)/float64(numChunks)*percentMultiplier))
		numChunks = maxChunks
	}

	chunks := make([]*net.IPNet, 0, numChunks)
	baseIP := subnet.IP.Mask(subnet.Mask).To4()
	if baseIP == nil {
		return []*net.IPNet{subnet}
	}

	// Convert base IP to uint32 for proper arithmetic
	baseUint := uint32(
		baseIP[0],
	)<<byteShift24 | uint32(
		baseIP[1],
	)<<byteShift16 | uint32(
		baseIP[2],
	)<<byteShift8 | uint32(
		baseIP[3],
	)

	for i := range numChunks {
		// Calculate the starting IP for this /24 chunk
		// Explicit bounds check for safe uint32 conversion (i is bounded by maxChunks ≤ 256)
		if i < 0 || i > 256 {
			continue
		}
		offset := uint32(i) * hostsPerSubnet24Block
		chunkUint := baseUint + offset

		// Mask each shifted value to byte explicitly. The shifts produce
		// values that fit in byte (8 bits) but gosec G115 needs the mask
		// to prove the bound.
		chunkIP := net.IP{
			byte((chunkUint >> byteShift24) & byteMask),
			byte((chunkUint >> byteShift16) & byteMask),
			byte((chunkUint >> byteShift8) & byteMask),
			0, // Start of /24 block
		}

		chunk := &net.IPNet{
			IP:   chunkIP,
			Mask: net.CIDRMask(cidrMask24, cidrBits32),
		}
		chunks = append(chunks, chunk)
	}

	return chunks
}

// pingSweep sends ICMP echo requests to all hosts in the subnet using raw sockets.
// For subnets larger than /24, automatically splits into /24 chunks and scans sequentially.
// Respects maxHostsPerSubnet configuration to cap total hosts scanned.
func (s *ARPScanner) pingSweep(ctx context.Context, subnet *net.IPNet) error {
	ones, bits := subnet.Mask.Size()
	totalHosts := 1<<(bits-ones) - subnetExcludeCount // Exclude network and broadcast

	// Calculate max chunks based on configured host limit
	// maxHostsPerSubnet / 254 = max /24 chunks to scan
	maxHosts := s.GetMaxHostsPerSubnet()
	maxChunks := (maxHosts + roundUpAdjust) / hostsPerSubnet24 // Round up

	// For large subnets, split into /24 chunks and scan sequentially
	chunks := splitSubnetIntoChunks(subnet, maxChunks)
	if len(chunks) > 1 {
		logging.GetLogger().InfoContext(ctx, "Large subnet detected - scanning in chunks",
			"subnet", subnet.String(),
			"totalHosts", totalHosts,
			"chunks", len(chunks),
			"maxHosts", maxHosts)

		for i, chunk := range chunks {
			select {
			case <-ctx.Done():
				return fmt.Errorf("chunk scan cancelled: %w", ctx.Err())
			default:
				logging.GetLogger().DebugContext(ctx, "Scanning chunk",
					"chunk", fmt.Sprintf("%d/%d", i+1, len(chunks)),
					"subnet", chunk.String())

				if err := s.pingSweepChunk(ctx, chunk); err != nil {
					logging.GetLogger().WarnContext(ctx, "Chunk scan failed - continuing with remaining chunks",
						"chunk", chunk.String(),
						"error", err)
				}
			}
		}
		return nil
	}

	// Small subnet - scan directly
	return s.pingSweepChunk(ctx, subnet)
}

// pingSweepChunk scans a single /24 or smaller subnet chunk.
func (s *ARPScanner) pingSweepChunk(ctx context.Context, subnet *net.IPNet) error {
	ones, bits := subnet.Mask.Size()
	numHosts := 1<<(bits-ones) - subnetExcludeCount // Exclude network and broadcast

	// Generate host IPs
	baseIP := subnet.IP.Mask(subnet.Mask).To4()
	if baseIP == nil {
		return errors.New("invalid subnet")
	}

	// Build list of IPs to ping
	var ips []net.IP
	for i := 1; i <= numHosts; i++ {
		ip := incrementIP(baseIP, i)
		if ip != nil && !ip.Equal(s.localIP) {
			ips = append(ips, ip)
		}
	}

	// Initialize pinger if needed (fixes #822 - check under lock)
	s.mu.Lock()
	if s.pinger == nil {
		pinger, err := NewICMPPinger(time.Second)
		if err != nil {
			s.mu.Unlock()
			logging.GetLogger().WarnContext(ctx, "Failed to create ICMP pinger", "error", err)
			return err
		}
		s.pinger = pinger
	}
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

	return nil
}
