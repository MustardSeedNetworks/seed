// Package discovery implements multi-protocol network device discovery.
// This file implements mDNS/Bonjour name resolution for Apple/Linux device discovery.
//
// mDNS (Multicast DNS) allows devices to advertise their hostname on the local network
// using the .local domain. It uses UDP multicast to 224.0.0.251:5353 (IPv4) or
// ff02::fb:5353 (IPv6).
package discovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/dns/dnsmessage"
)

const (
	mdnsPort       = 5353
	mdnsIPv4Addr   = "224.0.0.251"
	mdnsIPv6Addr   = "ff02::fb"
	mdnsTimeout    = 2 * time.Second
	mdnsMaxWorkers = 10
)

// MDNSResolver performs mDNS name resolution.
type MDNSResolver struct {
	mu            sync.RWMutex
	names         map[string]string // IP -> mDNS name
	ipByName      map[string]string // name -> IP (for reverse lookup)
	timeout       time.Duration
	interfaceName string
}

// MDNSResult represents a resolved mDNS name.
type MDNSResult struct {
	IP   string
	Name string
	Err  error
}

// NewMDNSResolver creates a new mDNS name resolver.
func NewMDNSResolver(interfaceName string) *MDNSResolver {
	return &MDNSResolver{
		names:         make(map[string]string),
		ipByName:      make(map[string]string),
		timeout:       mdnsTimeout,
		interfaceName: interfaceName,
	}
}

// ResolveIP queries mDNS for a device's hostname via reverse lookup.
// It constructs a PTR query for the IP address.
func (r *MDNSResolver) ResolveIP(ctx context.Context, ip string) (string, error) {
	// Check cache first
	r.mu.RLock()
	if name, ok := r.names[ip]; ok {
		r.mu.RUnlock()
		return name, nil
	}
	r.mu.RUnlock()

	// Parse IP address
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("invalid IP address: %s", ip)
	}

	// Build reverse lookup name
	var ptrName string
	if ipv4 := parsedIP.To4(); ipv4 != nil {
		// IPv4: reverse octets + .in-addr.arpa
		ptrName = fmt.Sprintf("%d.%d.%d.%d.in-addr.arpa.",
			ipv4[3], ipv4[2], ipv4[1], ipv4[0])
	} else {
		// IPv6: expand and reverse nibbles + .ip6.arpa
		// This is more complex, skip for now
		return "", errors.New("IPv6 mDNS lookup not implemented")
	}

	// Build mDNS query
	query, err := buildMDNSQuery(ptrName, dnsmessage.TypePTR)
	if err != nil {
		return "", fmt.Errorf("build query: %w", err)
	}

	// Send query and get response
	name, err := r.sendMDNSQuery(ctx, query, parsedIP)
	if err != nil {
		return "", err
	}

	// Cache result
	r.mu.Lock()
	r.names[ip] = name
	r.ipByName[name] = ip
	r.mu.Unlock()

	return name, nil
}

// ResolveName queries mDNS for a hostname and returns its IP address.
// Useful for looking up "hostname.local" addresses.
func (r *MDNSResolver) ResolveName(ctx context.Context, name string) (string, error) {
	// Ensure .local suffix
	if !strings.HasSuffix(name, ".local") && !strings.HasSuffix(name, ".local.") {
		name += ".local"
	}
	if !strings.HasSuffix(name, ".") {
		name += "."
	}

	// Check cache first
	r.mu.RLock()
	if ip, ok := r.ipByName[name]; ok {
		r.mu.RUnlock()
		return ip, nil
	}
	r.mu.RUnlock()

	// Build mDNS query for A record
	query, err := buildMDNSQuery(name, dnsmessage.TypeA)
	if err != nil {
		return "", fmt.Errorf("build query: %w", err)
	}

	// Send query and get response
	ip, err := r.sendMDNSQueryForIP(ctx, query)
	if err != nil {
		return "", err
	}

	// Cache result
	r.mu.Lock()
	r.ipByName[name] = ip
	r.names[ip] = strings.TrimSuffix(name, ".")
	r.mu.Unlock()

	return ip, nil
}

// sendMDNSQuery sends an mDNS multicast query and waits for a response.
func (r *MDNSResolver) sendMDNSQuery(ctx context.Context, query []byte, targetIP net.IP) (string, error) {
	timeout := r.timeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	// Use unicast query directly to the target IP for more reliable results
	addr := &net.UDPAddr{IP: targetIP, Port: mdnsPort}
	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return "", fmt.Errorf("dial mDNS: %w", err)
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	// Send query
	if _, writeErr := conn.Write(query); writeErr != nil {
		return "", fmt.Errorf("send query: %w", writeErr)
	}

	// Read response
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	// Parse response
	return parseMDNSResponse(buf[:n])
}

// sendMDNSQueryForIP sends an mDNS multicast query to resolve a name to IP.
func (r *MDNSResolver) sendMDNSQueryForIP(ctx context.Context, query []byte) (string, error) {
	timeout := r.timeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	// Use multicast for name lookups
	multicastAddr := &net.UDPAddr{IP: net.ParseIP(mdnsIPv4Addr), Port: mdnsPort}

	// Create UDP socket
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{Port: 0})
	if err != nil {
		return "", fmt.Errorf("listen mDNS: %w", err)
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	// Send query to multicast address
	if _, writeErr := conn.WriteToUDP(query, multicastAddr); writeErr != nil {
		return "", fmt.Errorf("send query: %w", writeErr)
	}

	// Read responses (devices respond with unicast)
	buf := make([]byte, 4096)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	// Parse response for IP
	return parseMDNSResponseForIP(buf[:n])
}

// ResolveBatch resolves mDNS names for multiple IPs concurrently.
//
//nolint:dupl // Similar pattern to NetBIOSResolver.ResolveBatch but uses different result type
func (r *MDNSResolver) ResolveBatch(ctx context.Context, ips []string) []MDNSResult {
	results := make([]MDNSResult, len(ips))
	resultCh := make(chan struct {
		idx    int
		result MDNSResult
	}, len(ips))

	// Use a semaphore to limit concurrent workers
	sem := make(chan struct{}, mdnsMaxWorkers)

	var wg sync.WaitGroup
	for i, ip := range ips {
		wg.Add(1)
		go func(idx int, ipAddr string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				resultCh <- struct {
					idx    int
					result MDNSResult
				}{idx, MDNSResult{IP: ipAddr, Err: ctx.Err()}}
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			name, err := r.ResolveIP(ctx, ipAddr)
			resultCh <- struct {
				idx    int
				result MDNSResult
			}{idx, MDNSResult{IP: ipAddr, Name: name, Err: err}}
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

// GetCached returns the cached mDNS name for an IP, if available.
func (r *MDNSResolver) GetCached(ip string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	name, ok := r.names[ip]
	return name, ok
}

// SetName manually sets a name for an IP (from passive capture).
func (r *MDNSResolver) SetName(ip, name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.names[ip] = name
	r.ipByName[name] = ip
}

// ClearCache clears the name cache.
func (r *MDNSResolver) ClearCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.names = make(map[string]string)
	r.ipByName = make(map[string]string)
}

// buildMDNSQuery creates an mDNS query packet.
func buildMDNSQuery(name string, qtype dnsmessage.Type) ([]byte, error) {
	var msg dnsmessage.Message
	msg.ID = 0                   // mDNS uses 0 for queries
	msg.RecursionDesired = false // Query, not response

	// Parse the name
	n, err := dnsmessage.NewName(name)
	if err != nil {
		return nil, fmt.Errorf("invalid name: %w", err)
	}

	msg.Questions = []dnsmessage.Question{
		{
			Name:  n,
			Type:  qtype,
			Class: dnsmessage.ClassINET,
		},
	}

	data, err := msg.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack mDNS query: %w", err)
	}
	return data, nil
}

// parseMDNSResponse parses an mDNS response for PTR records.
func parseMDNSResponse(data []byte) (string, error) {
	var msg dnsmessage.Message
	if err := msg.Unpack(data); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	// Look for PTR answers
	for i := range msg.Answers {
		if msg.Answers[i].Header.Type == dnsmessage.TypePTR {
			ptr, ok := msg.Answers[i].Body.(*dnsmessage.PTRResource)
			if ok {
				name := ptr.PTR.String()
				// Remove trailing dot and clean up
				name = strings.TrimSuffix(name, ".")
				return name, nil
			}
		}
	}

	// Also check additional records
	for i := range msg.Additionals {
		if msg.Additionals[i].Header.Type == dnsmessage.TypePTR {
			ptr, ok := msg.Additionals[i].Body.(*dnsmessage.PTRResource)
			if ok {
				name := ptr.PTR.String()
				name = strings.TrimSuffix(name, ".")
				return name, nil
			}
		}
	}

	return "", errors.New("no PTR record in response")
}

// parseMDNSResponseForIP parses an mDNS response for A records.
func parseMDNSResponseForIP(data []byte) (string, error) {
	var msg dnsmessage.Message
	if err := msg.Unpack(data); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	// Look for A answers
	for i := range msg.Answers {
		if msg.Answers[i].Header.Type == dnsmessage.TypeA {
			a, ok := msg.Answers[i].Body.(*dnsmessage.AResource)
			if ok {
				ip := net.IP(a.A[:])
				return ip.String(), nil
			}
		}
	}

	// Also check additional records
	for i := range msg.Additionals {
		if msg.Additionals[i].Header.Type == dnsmessage.TypeA {
			a, ok := msg.Additionals[i].Body.(*dnsmessage.AResource)
			if ok {
				ip := net.IP(a.A[:])
				return ip.String(), nil
			}
		}
	}

	return "", errors.New("no A record in response")
}

// MDNSListener passively captures mDNS announcements on the network.
type MDNSListener struct {
	mu            sync.RWMutex
	names         map[string]string // IP -> mDNS name
	running       bool
	stopCh        chan struct{}
	interfaceName string
}

// NewMDNSListener creates a new passive mDNS listener.
func NewMDNSListener(interfaceName string) *MDNSListener {
	return &MDNSListener{
		names:         make(map[string]string),
		interfaceName: interfaceName,
	}
}

// Start begins listening for mDNS announcements.
func (l *MDNSListener) Start() error {
	l.mu.Lock()
	if l.running {
		l.mu.Unlock()
		return nil
	}
	l.running = true
	l.stopCh = make(chan struct{})
	l.mu.Unlock()

	go l.listen()
	return nil
}

// Stop stops the mDNS listener.
func (l *MDNSListener) Stop() {
	l.mu.Lock()
	if !l.running {
		l.mu.Unlock()
		return
	}
	l.running = false
	close(l.stopCh)
	l.mu.Unlock()
}

// listen captures mDNS multicast traffic.
func (l *MDNSListener) listen() {
	// Join mDNS multicast group
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", mdnsIPv4Addr, mdnsPort))
	if err != nil {
		slog.Error("mDNS: failed to resolve multicast address", "error", err)
		return
	}

	// Get interface
	var iface *net.Interface
	if l.interfaceName != "" {
		iface, err = net.InterfaceByName(l.interfaceName)
		if err != nil {
			slog.Warn("mDNS: interface not found, using default", "interface", l.interfaceName, "error", err)
		}
	}

	conn, err := net.ListenMulticastUDP("udp4", iface, addr)
	if err != nil {
		slog.Error("mDNS: failed to join multicast group", "error", err)
		return
	}
	defer func() { _ = conn.Close() }()

	// Set read buffer
	_ = conn.SetReadBuffer(65536)

	slog.Info("mDNS listener started", "interface", l.interfaceName)

	buf := make([]byte, 4096)
	for {
		select {
		case <-l.stopCh:
			return
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		n, remoteAddr, readErr := conn.ReadFromUDP(buf)
		if readErr != nil {
			var netErr net.Error
			if errors.As(readErr, &netErr) && netErr.Timeout() {
				continue
			}
			continue
		}

		l.processPacket(buf[:n], remoteAddr)
	}
}

// processPacket handles an incoming mDNS packet.
func (l *MDNSListener) processPacket(data []byte, remoteAddr *net.UDPAddr) {
	var msg dnsmessage.Message
	if err := msg.Unpack(data); err != nil {
		return
	}

	// We're interested in responses (announcements)
	if !msg.Response {
		return
	}

	ip := remoteAddr.IP.String()

	// Look for A records to get hostname->IP mappings
	for i := range msg.Answers {
		if msg.Answers[i].Header.Type == dnsmessage.TypeA {
			name := strings.TrimSuffix(msg.Answers[i].Header.Name.String(), ".")
			if strings.HasSuffix(name, ".local") {
				if a, ok := msg.Answers[i].Body.(*dnsmessage.AResource); ok {
					answerIP := net.IP(a.A[:]).String()
					l.mu.Lock()
					l.names[answerIP] = name
					l.mu.Unlock()
					slog.Debug("mDNS: captured name", "ip", answerIP, "name", name)
				}
			}
		}
	}

	// Also check for PTR records (service announcements often include the device name)
	for i := range msg.Answers {
		if msg.Answers[i].Header.Type == dnsmessage.TypePTR {
			if ptr, ok := msg.Answers[i].Body.(*dnsmessage.PTRResource); ok {
				name := strings.TrimSuffix(ptr.PTR.String(), ".")
				if strings.HasSuffix(name, ".local") {
					l.mu.Lock()
					if _, exists := l.names[ip]; !exists {
						l.names[ip] = name
					}
					l.mu.Unlock()
				}
			}
		}
	}
}

// GetNames returns all captured mDNS names.
func (l *MDNSListener) GetNames() map[string]string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	result := make(map[string]string, len(l.names))
	maps.Copy(result, l.names)
	return result
}

// GetName returns the mDNS name for an IP if captured.
func (l *MDNSListener) GetName(ip string) (string, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	name, ok := l.names[ip]
	return name, ok
}
