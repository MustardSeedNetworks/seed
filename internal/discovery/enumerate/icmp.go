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
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

const (
	// ICMP protocol numbers.
	protocolICMP = 1

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

// NewICMPPinger creates a new ICMP pinger with raw socket.
// Requires root privileges or CAP_NET_RAW capability on Linux.
func NewICMPPinger(timeout time.Duration) (*ICMPPinger, error) {
	if timeout == 0 {
		timeout = defaultTimeout
	}

	p := &ICMPPinger{
		timeout: timeout,
		id:      os.Getpid() & seqNumMask,
		pending: make(map[int]*pendingPing),
		stopCh:  make(chan struct{}),
	}

	// Open privileged raw ICMP socket (requires root/CAP_NET_RAW)
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return nil, fmt.Errorf("failed to open ICMP socket: %w", err)
	}
	p.conn = conn

	// Enable TTL in control messages for OS fingerprinting
	if ctrlErr := conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true); ctrlErr != nil {
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

// nextSeq returns the next sequence number.
func (p *ICMPPinger) nextSeq() int {
	return int(atomic.AddUint32(&p.seq, 1) & seqNumMask)
}

// receiverReadResult represents the outcome of a single packet read attempt.
type receiverReadResult struct {
	n          int
	cm         *ipv4.ControlMessage
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

	n, cm, _, err := p.conn.IPv4PacketConn().ReadFrom(reply)
	if err != nil {
		return p.handleReadError(err)
	}

	return receiverReadResult{n: n, cm: cm}, true
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
	rm, err := icmp.ParseMessage(protocolICMP, data)
	if err != nil {
		return nil, false
	}

	if rm.Type != ipv4.ICMPTypeEchoReply {
		return nil, false
	}

	echo, ok := rm.Body.(*icmp.Echo)
	if !ok || echo.ID != p.id {
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
func (p *ICMPPinger) sendPingResult(pp *pendingPing, cm *ipv4.ControlMessage) {
	result := PingResult{
		IP:        pp.ip,
		Reachable: true,
		RTT:       time.Since(pp.start),
		TTL:       -1,
	}
	if cm != nil {
		result.TTL = cm.TTL
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

	p.sendPingResult(pp, readResult.cm)
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

// Ping sends an ICMP echo request to the specified IP and waits for a reply.
func (p *ICMPPinger) Ping(ctx context.Context, ipStr string) PingResult {
	result := PingResult{
		IP:  ipStr,
		TTL: -1,
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		result.Error = &net.ParseError{Type: "IP address", Text: ipStr}
		return result
	}

	dst := &net.IPAddr{IP: ip}
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

	// Build ICMP echo request
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
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

	// Send ICMP echo request
	if _, writeErr := p.conn.WriteTo(msgBytes, dst); writeErr != nil {
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
