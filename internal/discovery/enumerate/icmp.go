package enumerate

// ICMP ping support enables active probing of devices to verify reachability,
// measure latency, and identify responsive hosts on the network. Supports both
// sequential pinging and broadcast ping sweeps for network enumeration.

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/netip"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

const (
	// ICMP protocol numbers.
	protocolICMP   = 1
	protocolICMPv6 = 58

	// Default values.
	defaultTimeout = 1 * time.Second
	maxPacketSize  = 1500

	// TTL-based OS detection results.
	ttlOSUnknown       = "Unknown"
	ttlOSLinuxMacOS    = "Linux/macOS"
	ttlOSWindows       = "Windows"
	ttlOSNetworkDevice = "Network Device"

	// TTL threshold values for OS detection.
	icmpTTLLinux   = 64  // Default TTL for Linux/macOS systems
	icmpTTLWindows = 128 // Default TTL for Windows systems
	icmpTTLNetwork = 255 // Default TTL for network devices

	// Sequence number mask for 16-bit ICMP sequence numbers.
	seqNumMask = 0xffff

	// Read deadline for ICMP receiver loop.
	icmpReadDeadlineMs = 100

	// Default sweep configuration values.
	defaultSweepWorkers    = 50 // Number of concurrent workers for default sweep
	politeSweepWorkers     = 10 // Number of concurrent workers for polite sweep
	politeSweepJitterMinMs = 10 // Minimum jitter delay in milliseconds
	politeSweepJitterMaxMs = 50 // Maximum jitter delay in milliseconds
)

// PingResult contains the result of a single ICMP ping.
type PingResult struct {
	IP        string
	Reachable bool
	TTL       int
	RTT       time.Duration
	Error     error
}

// pendingPing tracks an in-flight ping request.
type pendingPing struct {
	ip     string
	seq    int
	start  time.Time
	result chan PingResult
}

// ICMPPinger provides raw socket ICMP ping functionality.
// Uses a dedicated receiver goroutine to properly handle concurrent pings.
type ICMPPinger struct {
	conn    *icmp.PacketConn
	timeout time.Duration
	id      int
	seq     uint32

	// privileged records which socket the constructor got. A datagram socket
	// has its echo ID rewritten by the kernel, so the reply match and the
	// destination address type both depend on this.
	privileged bool

	// ipv6 marks an ICMPv6 pinger. One pinger speaks one family: the echo
	// types, the reply parser and the hop-limit control message all differ.
	ipv6 bool

	// egress is the index of the interface every echo must leave through, or
	// zero for host routing. See NewICMPPingerOn.
	egress int

	// Pending pings tracked by sequence number
	pending   map[int]*pendingPing
	pendingMu sync.Mutex

	// Channels for coordinating
	stopCh    chan struct{}
	stopped   bool
	stoppedMu sync.Mutex

	// Jitter settings for IDS-aware scanning
	_ time.Duration // Reserved: jitterMin - Minimum delay between pings per worker
	_ time.Duration // Reserved: jitterMax - Maximum delay (actual delay is random between min and max)
}

// SweepConfig configures ping sweep behavior.
type SweepConfig struct {
	Workers   int           // Number of concurrent workers (default: 50)
	JitterMin time.Duration // Minimum jitter delay between pings (default: 0)
	JitterMax time.Duration // Maximum jitter delay between pings (default: 0)
}

// DefaultSweepConfig returns conservative defaults for network scanning.
func DefaultSweepConfig() *SweepConfig {
	return &SweepConfig{
		Workers:   defaultSweepWorkers,
		JitterMin: 0,
		JitterMax: 0,
	}
}

// PoliteSweepConfig returns IDS-friendly settings with jitter.
func PoliteSweepConfig() *SweepConfig {
	return &SweepConfig{
		Workers:   politeSweepWorkers,
		JitterMin: politeSweepJitterMinMs * time.Millisecond,
		JitterMax: politeSweepJitterMaxMs * time.Millisecond,
	}
}

// NewICMPPinger creates an ICMPv4 pinger. It prefers a raw socket and falls
// back to the unprivileged datagram socket where raw is refused (seed#2629).
// Requires root privileges or CAP_NET_RAW capability on Linux.
func NewICMPPinger(timeout time.Duration) (*ICMPPinger, error) {
	return newICMPPinger(timeout, false, nil)
}

// NewICMPv6Pinger creates an ICMPv6 pinger, with the same raw-then-datagram
// fallback: Linux's net.ipv4.ping_group_range governs the datagram socket of
// both families. Targets may carry a zone ("fe80::1%eth0"), which a
// link-local address needs to leave the host (seed#3251).
func NewICMPv6Pinger(timeout time.Duration) (*ICMPPinger, error) {
	return newICMPPinger(timeout, true, nil)
}

// NewICMPPingerOn creates a pinger of the given family whose echoes leave
// through iface whatever the host's routing says, so a target answers only if
// it is reachable over that link.
//
// Two mechanisms, because the platforms differ. The socket is bound to the
// interface's own address, which is enough on Windows (strong host model).
// Linux and macOS route a packet by destination whatever its source, so each
// echo also names the interface in its packet-info control message.
func NewICMPPingerOn(timeout time.Duration, ipv6Family bool, iface *net.Interface) (*ICMPPinger, error) {
	return newICMPPinger(timeout, ipv6Family, iface)
}

// interfaceAddress returns iface's source address for the requested family.
func interfaceAddress(iface *net.Interface, ipv6Family bool) (string, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return "", fmt.Errorf("read %s addresses: %w", iface.Name, err)
	}
	return sourceAddress(iface.Name, addrs, ipv6Family)
}

// sourceAddress picks the first address of the requested family, preferring
// one usable beyond the link: a link-local source cannot carry an echo to the
// internet.
func sourceAddress(name string, addrs []net.Addr, ipv6Family bool) (string, error) {
	var linkLocal string
	for _, a := range addrs {
		prefix, err := netip.ParsePrefix(a.String())
		if err != nil {
			continue
		}
		addr := prefix.Addr()
		if addr.Is4() == ipv6Family {
			continue
		}
		if !addr.IsLinkLocalUnicast() {
			return addr.String(), nil
		}
		if linkLocal == "" {
			linkLocal = addr.WithZone(name).String()
		}
	}
	if linkLocal != "" {
		return linkLocal, nil
	}
	family := "IPv4"
	if ipv6Family {
		family = "IPv6"
	}
	return "", fmt.Errorf("%s has no %s address", name, family)
}

func newICMPPinger(timeout time.Duration, ipv6Family bool, iface *net.Interface) (*ICMPPinger, error) {
	if timeout == 0 {
		timeout = defaultTimeout
	}

	p := &ICMPPinger{
		timeout: timeout,
		id:      os.Getpid() & seqNumMask,
		ipv6:    ipv6Family,
		pending: make(map[int]*pendingPing),
		stopCh:  make(chan struct{}),
	}

	rawNetwork, datagramNetwork, wildcard := "ip4:icmp", "udp4", "0.0.0.0"
	if ipv6Family {
		rawNetwork, datagramNetwork, wildcard = "ip6:ipv6-icmp", "udp6", "::"
	}
	if iface != nil {
		local, err := interfaceAddress(iface, ipv6Family)
		if err != nil {
			return nil, err
		}
		wildcard = local
		p.egress = iface.Index
	}

	// A raw socket needs root or CAP_NET_RAW. Where it is refused, fall back to
	// the datagram ICMP socket, which is unprivileged on macOS and on any Linux
	// host whose net.ipv4.ping_group_range admits this process (seed#2629).
	// Without the fallback an unprivileged daemon sweeps nothing at all — and
	// the target networks an operator configured are reached only by the sweep.
	conn, err := icmp.ListenPacket(rawNetwork, wildcard)
	if err != nil {
		rawErr := err
		if conn, err = icmp.ListenPacket(datagramNetwork, wildcard); err != nil {
			return nil, fmt.Errorf("failed to open ICMP socket (raw: %w; datagram: %w)", rawErr, err)
		}
	} else {
		p.privileged = true
	}
	p.conn = conn

	// Enable TTL (hop limit) in control messages for OS fingerprinting
	var ctrlErr error
	if ipv6Family {
		ctrlErr = conn.IPv6PacketConn().SetControlMessage(ipv6.FlagHopLimit, true)
	} else {
		ctrlErr = conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)
	}
	if ctrlErr != nil {
		// Non-fatal - TTL extraction may not work but ping will still function
		logging.GetLogger().Warn("failed to enable TTL control message", "error", ctrlErr)
	}

	// Start the receiver goroutine
	go p.receiver()

	// Set finalizer to ensure cleanup if Close() is not called.
	// This prevents goroutine leaks if the pinger is abandoned.
	runtime.SetFinalizer(p, func(pinger *ICMPPinger) {
		if closeErr := pinger.Close(); closeErr != nil {
			logging.GetLogger().Debug("ICMPPinger finalizer close error", "error", closeErr)
		}
	})

	return p, nil
}

// Close closes the ICMP socket and stops the receiver.
// Fixes #894: Drain pending pings to prevent goroutines waiting forever.
func (p *ICMPPinger) Close() error {
	p.stoppedMu.Lock()
	if !p.stopped {
		p.stopped = true
		close(p.stopCh)
	}
	p.stoppedMu.Unlock()

	// Fixes #894: Drain pending pings so waiting goroutines can complete
	p.pendingMu.Lock()
	for seq, pp := range p.pending {
		// Send timeout result to unblock waiting goroutines
		select {
		case pp.result <- PingResult{IP: pp.ip, TTL: -1, Error: context.Canceled}:
		default:
		}
		delete(p.pending, seq)
	}
	p.pendingMu.Unlock()

	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			return fmt.Errorf("failed to close ICMP connection: %w", err)
		}
	}
	return nil
}

// write sends one echo, naming the egress interface when the pinger has one.
func (p *ICMPPinger) write(msg []byte, dst net.Addr) error {
	var err error
	switch {
	case p.egress == 0:
		_, err = p.conn.WriteTo(msg, dst)
	case p.ipv6:
		_, err = p.conn.IPv6PacketConn().WriteTo(msg, &ipv6.ControlMessage{IfIndex: p.egress}, dst)
	default:
		_, err = p.conn.IPv4PacketConn().WriteTo(msg, &ipv4.ControlMessage{IfIndex: p.egress}, dst)
	}
	return err
}

// Privileged reports whether the pinger holds a raw ICMP socket. False means it
// fell back to the unprivileged datagram socket (seed#2629).
func (p *ICMPPinger) Privileged() bool { return p.privileged }

// nextSeq returns the next sequence number.
func (p *ICMPPinger) nextSeq() int {
	return int(atomic.AddUint32(&p.seq, 1) & seqNumMask)
}

// receiverReadResult represents the outcome of a single packet read attempt.
// ttl is the reply's TTL or hop limit, -1 when the kernel did not report it.
type receiverReadResult struct {
	n          int
	ttl        int
	shouldExit bool
}

// readPacket attempts to read a single ICMP packet from the connection.
// Returns the read result and whether to continue the receiver loop.
func (p *ICMPPinger) readPacket(reply []byte) (receiverReadResult, bool) {
	// Set a short read deadline so we can check stopCh periodically
	if err := p.conn.SetReadDeadline(time.Now().Add(icmpReadDeadlineMs * time.Millisecond)); err != nil {
		logging.GetLogger().Error("failed to set ICMP read deadline", "error", err)
		return receiverReadResult{}, true // continue loop
	}

	if p.ipv6 {
		n, cm, _, err := p.conn.IPv6PacketConn().ReadFrom(reply)
		if err != nil {
			return p.handleReadError(err)
		}
		if cm == nil {
			return receiverReadResult{n: n, ttl: -1}, true
		}
		return receiverReadResult{n: n, ttl: cm.HopLimit}, true
	}

	n, cm, _, err := p.conn.IPv4PacketConn().ReadFrom(reply)
	if err != nil {
		return p.handleReadError(err)
	}
	if cm == nil {
		return receiverReadResult{n: n, ttl: -1}, true
	}
	return receiverReadResult{n: n, ttl: cm.TTL}, true
}

// handleReadError processes errors from packet reads.
// Returns the result and whether to continue the receiver loop.
func (p *ICMPPinger) handleReadError(err error) (receiverReadResult, bool) {
	// Timeout is expected, just continue
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return receiverReadResult{}, true // continue loop
	}
	// Socket closed - signal exit
	return receiverReadResult{shouldExit: true}, false
}

// extractEchoReply parses an ICMP message and extracts the echo reply if valid.
// Returns the echo body and whether it's a valid echo reply for this pinger.
func (p *ICMPPinger) extractEchoReply(data []byte) (*icmp.Echo, bool) {
	proto, echoReply := protocolICMP, icmp.Type(ipv4.ICMPTypeEchoReply)
	if p.ipv6 {
		proto, echoReply = protocolICMPv6, ipv6.ICMPTypeEchoReply
	}
	rm, err := icmp.ParseMessage(proto, data)
	if err != nil {
		return nil, false
	}

	if rm.Type != echoReply {
		return nil, false
	}

	echo, ok := rm.Body.(*icmp.Echo)
	if !ok {
		return nil, false
	}
	// A raw socket sees every process's replies, so the ID is what tells ours
	// apart. A datagram socket is demultiplexed by the kernel, which also
	// assigns the ID it sent — comparing it there rejects every real reply.
	if p.privileged && echo.ID != p.id {
		return nil, false
	}

	return echo, true
}

// completePendingPing finds and completes a pending ping by sequence number.
// Returns the pending ping if found and removed from the map.
func (p *ICMPPinger) completePendingPing(seq int) (*pendingPing, bool) {
	p.pendingMu.Lock()
	defer p.pendingMu.Unlock()

	pp, found := p.pending[seq]
	if found {
		delete(p.pending, seq)
	}
	return pp, found
}

// sendPingResult constructs and sends the ping result to the waiting goroutine.
func (p *ICMPPinger) sendPingResult(pp *pendingPing, ttl int) {
	result := PingResult{
		IP:        pp.ip,
		Reachable: true,
		RTT:       time.Since(pp.start),
		TTL:       ttl,
	}

	select {
	case pp.result <- result:
	default:
	}
}

// processReceivedPacket handles a successfully received ICMP packet.
func (p *ICMPPinger) processReceivedPacket(reply []byte, readResult receiverReadResult) {
	echo, valid := p.extractEchoReply(reply[:readResult.n])
	if !valid {
		return
	}

	pp, found := p.completePendingPing(echo.Seq)
	if !found {
		return
	}

	p.sendPingResult(pp, readResult.ttl)
}

// receiver runs in a goroutine and dispatches received ICMP replies.
func (p *ICMPPinger) receiver() {
	reply := make([]byte, maxPacketSize)

	for {
		select {
		case <-p.stopCh:
			return
		default:
		}

		readResult, shouldContinue := p.readPacket(reply)
		if readResult.shouldExit {
			return
		}
		if !shouldContinue || readResult.n == 0 {
			continue
		}

		p.processReceivedPacket(reply, readResult)
	}
}

// destination returns the address type the pinger's socket accepts: a raw
// socket writes to an IPAddr, a datagram socket to a UDPAddr (port unused).
func (p *ICMPPinger) destination(addr netip.Addr) net.Addr {
	ip, zone := net.IP(addr.AsSlice()), addr.Zone()
	if p.privileged {
		return &net.IPAddr{IP: ip, Zone: zone}
	}
	return &net.UDPAddr{IP: ip, Zone: zone}
}

// target parses ipStr, zone included, and checks it belongs to the pinger's
// family; an IPv4-mapped IPv6 address counts as IPv4, as [net.ParseIP] had it.
func (p *ICMPPinger) target(ipStr string) (netip.Addr, error) {
	addr, err := netip.ParseAddr(ipStr)
	if err != nil {
		return netip.Addr{}, &net.ParseError{Type: "IP address", Text: ipStr}
	}
	if addr.Is4In6() {
		addr = addr.Unmap()
	}
	if addr.Is6() != p.ipv6 {
		return netip.Addr{}, &net.AddrError{Err: "address family does not match the pinger", Addr: ipStr}
	}
	return addr, nil
}

// Ping sends an ICMP echo request to the specified IP and waits for a reply.
func (p *ICMPPinger) Ping(ctx context.Context, ipStr string) PingResult {
	result := PingResult{
		IP:  ipStr,
		TTL: -1,
	}

	addr, err := p.target(ipStr)
	if err != nil {
		result.Error = err
		return result
	}

	dst := p.destination(addr)
	seq := p.nextSeq()

	// Create result channel
	resultCh := make(chan PingResult, 1)

	// Register pending ping
	pp := &pendingPing{
		ip:     ipStr,
		seq:    seq,
		start:  time.Now(),
		result: resultCh,
	}

	p.pendingMu.Lock()
	p.pending[seq] = pp
	p.pendingMu.Unlock()

	// Cleanup on exit
	defer func() {
		p.pendingMu.Lock()
		delete(p.pending, seq)
		p.pendingMu.Unlock()
	}()

	// Build ICMP echo request. The kernel fills in the ICMPv6 checksum, which
	// covers a pseudo-header only it knows the source address of.
	echoRequest := icmp.Type(ipv4.ICMPTypeEcho)
	if p.ipv6 {
		echoRequest = ipv6.ICMPTypeEchoRequest
	}
	msg := icmp.Message{
		Type: echoRequest,
		Code: 0,
		Body: &icmp.Echo{
			ID:   p.id,
			Seq:  seq,
			Data: []byte("seed"),
		},
	}

	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		result.Error = err
		return result
	}

	if writeErr := p.write(msgBytes, dst); writeErr != nil {
		result.Error = writeErr
		return result
	}

	// Wait for reply or timeout
	timeout := p.timeout
	if d, ok := ctx.Deadline(); ok {
		remaining := time.Until(d)
		if remaining < timeout {
			timeout = remaining
		}
	}

	select {
	case r := <-resultCh:
		return r
	case <-time.After(timeout):
		// Timeout - host not reachable
		return result
	case <-ctx.Done():
		result.Error = ctx.Err()
		return result
	}
}

// PingSweep pings multiple hosts concurrently and returns results.
// For IDS-aware scanning with jitter, use PingSweepWithConfig.
func (p *ICMPPinger) PingSweep(ctx context.Context, ips []net.IP, workers int) []PingResult {
	cfg := DefaultSweepConfig()
	if workers > 0 {
		cfg.Workers = workers
	}
	return p.PingSweepWithConfig(ctx, ips, cfg)
}

// normalizeSweepConfig returns a valid sweep configuration with defaults applied.
func normalizeSweepConfig(cfg *SweepConfig) *SweepConfig {
	if cfg == nil {
		return DefaultSweepConfig()
	}
	normalized := *cfg
	if normalized.Workers <= 0 {
		normalized.Workers = 50
	}
	return &normalized
}

// createWorkChannel creates and populates a buffered channel with work indices.
func createWorkChannel(count int) chan int {
	work := make(chan int, count)
	for i := range count {
		work <- i
	}
	close(work)
	return work
}

// calculateJitter computes a random jitter duration within the configured range.
func calculateJitter(cfg *SweepConfig) time.Duration {
	jitter := cfg.JitterMin
	if cfg.JitterMax > cfg.JitterMin {
		jitter += time.Duration(
			rand.Int64N(
				int64(cfg.JitterMax - cfg.JitterMin),
			), // #nosec G404 -- weak RNG acceptable for timing jitter
		)
	}
	return jitter
}

// applyJitterDelay waits for the jitter duration, respecting context cancellation.
// Returns true if the context was cancelled during the wait.
func applyJitterDelay(ctx context.Context, cfg *SweepConfig) bool {
	if cfg.JitterMax <= 0 {
		return false
	}

	jitter := calculateJitter(cfg)
	select {
	case <-ctx.Done():
		return true
	case <-time.After(jitter):
		return false
	}
}

// sweepWorkerState holds the shared state for sweep workers.
type sweepWorkerState struct {
	results   []PingResult
	resultsMu *sync.Mutex
	ips       []net.IP
	cfg       *SweepConfig
}

// runSweepWorker processes work items from the channel, pinging each IP.
func (p *ICMPPinger) runSweepWorker(ctx context.Context, work <-chan int, state *sweepWorkerState) {
	for idx := range work {
		if ctx.Err() != nil {
			return
		}

		result := p.Ping(ctx, state.ips[idx].String())

		state.resultsMu.Lock()
		state.results[idx] = result
		state.resultsMu.Unlock()

		if applyJitterDelay(ctx, state.cfg) {
			return
		}
	}
}

// countReachable counts the number of reachable hosts in the results.
func countReachable(results []PingResult) int {
	count := 0
	for _, r := range results {
		if r.Reachable {
			count++
		}
	}
	return count
}

// PingSweepWithConfig pings multiple hosts with configurable jitter for IDS-aware scanning.
func (p *ICMPPinger) PingSweepWithConfig(
	ctx context.Context,
	ips []net.IP,
	cfg *SweepConfig,
) []PingResult {
	cfg = normalizeSweepConfig(cfg)

	state := &sweepWorkerState{
		results:   make([]PingResult, len(ips)),
		resultsMu: &sync.Mutex{},
		ips:       ips,
		cfg:       cfg,
	}

	work := createWorkChannel(len(ips))

	var wg sync.WaitGroup
	wg.Add(cfg.Workers)

	for range cfg.Workers {
		go func() {
			defer wg.Done()
			p.runSweepWorker(ctx, work, state)
		}()
	}

	wg.Wait()

	logging.GetLogger().InfoContext(ctx, "Ping sweep complete",
		"reachable", countReachable(state.results),
		"total", len(ips))

	return state.results
}

// PingSweepReachable is a convenience method that returns only reachable hosts.
func (p *ICMPPinger) PingSweepReachable(
	ctx context.Context,
	ips []net.IP,
	workers int,
) []PingResult {
	all := p.PingSweep(ctx, ips, workers)
	reachable := make([]PingResult, 0, len(all))
	for _, r := range all {
		if r.Reachable {
			reachable = append(reachable, r)
		}
	}
	return reachable
}

// CheckICMPPrivileges checks if the current process has privileges to use raw ICMP sockets.
// Returns nil if privileged, error otherwise.
func CheckICMPPrivileges() error {
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return fmt.Errorf("failed to create ICMP socket: %w", err)
	}
	_ = conn.Close()
	return nil
}

// TTLToOS attempts to guess the operating system based on TTL value.
// Returns the OS name or "Unknown" if TTL doesn't match known patterns.
func TTLToOS(ttl int) string {
	switch {
	case ttl <= 0:
		return ttlOSUnknown
	case ttl <= icmpTTLLinux:
		return ttlOSLinuxMacOS
	case ttl <= icmpTTLWindows:
		return ttlOSWindows
	case ttl <= icmpTTLNetwork:
		return ttlOSNetworkDevice
	default:
		return ttlOSUnknown
	}
}

// ErrICMPPrivileges is returned when raw ICMP socket privileges are unavailable.
var ErrICMPPrivileges = errors.New("raw ICMP socket privileges unavailable")

// CheckICMPPrivilegesWithMessage checks if the current process has privileges to use raw ICMP sockets.
// Returns nil if privileged, a descriptive error otherwise.
func CheckICMPPrivilegesWithMessage() error {
	if err := CheckICMPPrivileges(); err != nil {
		return fmt.Errorf(
			"%w: %w (run with sudo or grant CAP_NET_RAW capability)",
			ErrICMPPrivileges,
			err,
		)
	}
	return nil
}
