// Package discovery implements multi-protocol network device discovery.
// This file implements NetBIOS name resolution (NBNS) for Windows device discovery.
//
// NetBIOS Name Service (NBNS) allows querying Windows devices for their computer names.
// It uses UDP port 137 and is part of the NetBIOS over TCP/IP (NBT) protocol suite.
package discovery

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	netbiosPort        = 137
	netbiosTimeout     = 500 * time.Millisecond
	netbiosMaxWorkers  = 20
	netbiosNameLen     = 15
	netbiosQueryOpcode = 0x0000
)

// NetBIOSResolver performs NetBIOS name resolution (UDP 137).
type NetBIOSResolver struct {
	mu      sync.RWMutex
	names   map[string]string // IP -> NetBIOS name
	timeout time.Duration
}

// NetBIOSResult represents a resolved NetBIOS name.
type NetBIOSResult struct {
	IP   string
	Name string
	Err  error
}

// NewNetBIOSResolver creates a new NetBIOS name resolver.
func NewNetBIOSResolver() *NetBIOSResolver {
	return &NetBIOSResolver{
		names:   make(map[string]string),
		timeout: netbiosTimeout,
	}
}

// ResolveIP queries a single IP for its NetBIOS name.
func (r *NetBIOSResolver) ResolveIP(ctx context.Context, ip string) (string, error) {
	// Check cache first
	r.mu.RLock()
	if name, ok := r.names[ip]; ok {
		r.mu.RUnlock()
		return name, nil
	}
	r.mu.RUnlock()

	// Build NetBIOS name query packet
	query := buildNetBIOSQuery()

	// Set up UDP connection with timeout
	timeout := r.timeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	addr := net.JoinHostPort(ip, strconv.Itoa(netbiosPort))
	dialer := &net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "udp", addr)
	if err != nil {
		return "", fmt.Errorf("connect to %s: %w", addr, err)
	}
	defer func() { _ = conn.Close() }()

	// Set read/write deadlines
	_ = conn.SetDeadline(time.Now().Add(timeout))

	// Send query
	if _, writeErr := conn.Write(query); writeErr != nil {
		return "", fmt.Errorf("send query: %w", writeErr)
	}

	// Read response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	// Parse response
	name, err := parseNetBIOSResponse(buf[:n])
	if err != nil {
		return "", err
	}

	// Cache result
	r.mu.Lock()
	r.names[ip] = name
	r.mu.Unlock()

	return name, nil
}

// ResolveBatch resolves NetBIOS names for multiple IPs concurrently.
//
//nolint:dupl // Similar pattern to MDNSResolver.ResolveBatch but uses different result type
func (r *NetBIOSResolver) ResolveBatch(ctx context.Context, ips []string) []NetBIOSResult {
	results := make([]NetBIOSResult, len(ips))
	resultCh := make(chan struct {
		idx    int
		result NetBIOSResult
	}, len(ips))

	// Use a semaphore to limit concurrent workers
	sem := make(chan struct{}, netbiosMaxWorkers)

	var wg sync.WaitGroup
	for i, ip := range ips {
		wg.Add(1)
		go func(idx int, ipAddr string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				resultCh <- struct {
					idx    int
					result NetBIOSResult
				}{idx, NetBIOSResult{IP: ipAddr, Err: ctx.Err()}}
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			name, err := r.ResolveIP(ctx, ipAddr)
			resultCh <- struct {
				idx    int
				result NetBIOSResult
			}{idx, NetBIOSResult{IP: ipAddr, Name: name, Err: err}}
		}(i, ip)
	}

	// Close result channel when all workers done
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	for res := range resultCh {
		results[res.idx] = res.result
	}

	return results
}

// GetCached returns the cached NetBIOS name for an IP, if available.
func (r *NetBIOSResolver) GetCached(ip string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	name, ok := r.names[ip]
	return name, ok
}

// ClearCache clears the name cache.
func (r *NetBIOSResolver) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.names = make(map[string]string)
}

// buildNetBIOSQuery creates a NetBIOS name query packet.
// This queries for the "*" (wildcard) name to get all registered names.
func buildNetBIOSQuery() []byte {
	// NetBIOS Name Service header:
	// - Transaction ID (2 bytes)
	// - Flags (2 bytes): 0x0000 for query
	// - Questions (2 bytes): 1
	// - Answer RRs (2 bytes): 0
	// - Authority RRs (2 bytes): 0
	// - Additional RRs (2 bytes): 0
	// - Question section

	packet := make([]byte, 50)

	// Transaction ID (random but we'll use a fixed value)
	binary.BigEndian.PutUint16(packet[0:2], 0x1234)

	// Flags: standard query
	binary.BigEndian.PutUint16(packet[2:4], netbiosQueryOpcode)

	// Questions: 1
	binary.BigEndian.PutUint16(packet[4:6], 1)

	// Answer RRs: 0
	binary.BigEndian.PutUint16(packet[6:8], 0)

	// Authority RRs: 0
	binary.BigEndian.PutUint16(packet[8:10], 0)

	// Additional RRs: 0
	binary.BigEndian.PutUint16(packet[10:12], 0)

	// Question: Query for NBSTAT (node status) - "*" wildcard name
	// NetBIOS names are encoded as: length byte + encoded name (32 bytes)
	// The "*" wildcard is encoded specially

	// Length of encoded name section: 32 bytes
	packet[12] = 32

	// Encode "*" name (first-level encoding)
	// "*" padded to 15 characters with spaces, then a null suffix (0x00)
	// Each character is split into two nibbles, added to 'A' (0x41)
	name := "*" + strings.Repeat(" ", 14) // 15 chars
	suffix := byte(0x00)                  // Workstation service

	for i := range 15 {
		ch := name[i]
		packet[13+i*2] = 'A' + (ch >> 4)
		packet[13+i*2+1] = 'A' + (ch & 0x0F)
	}
	// Encode suffix
	packet[13+30] = 'A' + (suffix >> 4)
	packet[13+31] = 'A' + (suffix & 0x0F)

	// Null terminator for name
	packet[45] = 0x00

	// Query type: NBSTAT (0x0021)
	binary.BigEndian.PutUint16(packet[46:48], 0x0021)

	// Query class: IN (0x0001)
	binary.BigEndian.PutUint16(packet[48:50], 0x0001)

	return packet
}

// parseNetBIOSResponse parses a NetBIOS node status response.
func parseNetBIOSResponse(data []byte) (string, error) {
	if len(data) < 57 {
		return "", fmt.Errorf("response too short: %d bytes", len(data))
	}

	// Skip header (12 bytes) and question section
	// The response includes the query back, so we need to find the answer section
	// After header, skip the name and query fields to get to the answer

	// Find the answer section (starts after header + question)
	// Question section: encoded name (34 bytes) + type (2) + class (2) = 38 bytes
	// Header: 12 bytes
	// Answer starts around offset 50-56, but structure varies

	// Look for the name list in the response
	// After the header and answer metadata, there's a count of names
	// followed by the name entries (18 bytes each: 15 name + 1 suffix + 2 flags)

	offset := 56 // Approximate start of answer data section
	if offset >= len(data) {
		return "", errors.New("no answer section found")
	}

	// Number of names follows the TTL in the answer
	if offset+1 > len(data) {
		return "", errors.New("missing name count")
	}

	numNames := int(data[offset])
	offset++

	if numNames == 0 {
		return "", errors.New("no names in response")
	}

	// Read names - look for the computer name (suffix 0x00 = Workstation Service)
	var computerName string
	for i := 0; i < numNames && offset+18 <= len(data); i++ {
		// Each entry: 15-byte name + 1-byte suffix + 2-byte flags
		name := strings.TrimRight(string(data[offset:offset+15]), " \x00")
		suffix := data[offset+15]
		// Flags at offset+16:offset+18 are unused but present in protocol

		// Suffix 0x00 = Workstation service (computer name)
		// Suffix 0x20 = File server service
		// Look for 0x00 suffix which is the primary computer name
		if suffix == 0x00 && computerName == "" {
			computerName = name
		}

		offset += 18
	}

	if computerName == "" {
		return "", errors.New("computer name not found in response")
	}

	return computerName, nil
}
