// Package discovery implements multi-protocol network device discovery.
// ICMP ping support enables active probing of devices to verify reachability,
// measure latency, and identify responsive hosts on the network. Supports both
// sequential pinging and broadcast ping sweeps for network enumeration.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
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
		Workers:   50,
		JitterMin: 0,
		JitterMax: 0,
	}
}

// PoliteSweepConfig returns IDS-friendly settings with jitter.
func PoliteSweepConfig() *SweepConfig {
	return &SweepConfig{
		Workers:   10,
		JitterMin: 10 * time.Millisecond,
		JitterMax: 50 * time.Millisecond,
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
		id:      os.Getpid() & 0xffff,
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
		slog.Warn("failed to enable TTL control message", "error", ctrlErr)
	}

	// Start the receiver goroutine
	go p.receiver()

	// Set finalizer to ensure cleanup if Close() is not called.
	// This prevents goroutine leaks if the pinger is abandoned.
	runtime.SetFinalizer(p, func(pinger *ICMPPinger) {
		if closeErr := pinger.Close(); closeErr != nil {
			slog.Debug("ICMPPinger finalizer close error", "error", closeErr)
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
	return int(atomic.AddUint32(&p.seq, 1) & 0xffff)
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

		// Set a short read deadline so we can check stopCh periodically
		if err := p.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
			slog.Error("failed to set ICMP read deadline", "error", err)
			continue
		}

		n, cm, _, err := p.conn.IPv4PacketConn().ReadFrom(reply)
		if err != nil {
			// Timeout is expected, just continue
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				continue
			}
			// Socket closed
			return
		}

		// Parse ICMP message
		rm, err := icmp.ParseMessage(protocolICMP, reply[:n])
		if err != nil {
			continue
		}

		// Check if this is an echo reply for us
		if rm.Type == ipv4.ICMPTypeEchoReply {
			if echo, ok := rm.Body.(*icmp.Echo); ok {
				if echo.ID == p.id {
					// Find the pending ping for this sequence
					p.pendingMu.Lock()
					pp, found := p.pending[echo.Seq]
					if found {
						delete(p.pending, echo.Seq)
					}
					p.pendingMu.Unlock()

					if found {
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
				}
			}
		}
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

// PingSweepWithConfig pings multiple hosts with configurable jitter for IDS-aware scanning.
func (p *ICMPPinger) PingSweepWithConfig(ctx context.Context, ips []net.IP, cfg *SweepConfig) []PingResult {
	if cfg == nil {
		cfg = DefaultSweepConfig()
	}
	workers := cfg.Workers
	if workers <= 0 {
		workers = 50
	}

	results := make([]PingResult, len(ips))
	resultsMu := sync.Mutex{}

	// Create work channel
	work := make(chan int, len(ips))
	for i := range ips {
		work <- i
	}
	close(work)

	// Create worker pool
	var wg sync.WaitGroup
	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()
			for idx := range work {
				select {
				case <-ctx.Done():
					return
				default:
				}

				ip := ips[idx]
				result := p.Ping(ctx, ip.String())

				resultsMu.Lock()
				results[idx] = result
				resultsMu.Unlock()

				// Apply jitter delay between pings (IDS-aware pacing)
				if cfg.JitterMax > 0 {
					jitter := cfg.JitterMin
					if cfg.JitterMax > cfg.JitterMin {
						jitter += time.Duration(
							rand.Int64N(
								int64(cfg.JitterMax - cfg.JitterMin),
							), // #nosec G404 -- weak RNG acceptable for timing jitter
						)
					}
					select {
					case <-ctx.Done():
						return
					case <-time.After(jitter):
					}
				}
			}
		}()
	}

	wg.Wait()

	// Log summary
	reachable := 0
	for _, r := range results {
		if r.Reachable {
			reachable++
		}
	}
	slog.Info("Ping sweep complete", "reachable", reachable, "total", len(ips))

	return results
}

// PingSweepReachable is a convenience method that returns only reachable hosts.
func (p *ICMPPinger) PingSweepReachable(ctx context.Context, ips []net.IP, workers int) []PingResult {
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
	case ttl <= 64:
		return ttlOSLinuxMacOS
	case ttl <= 128:
		return ttlOSWindows
	case ttl <= 255:
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
		return fmt.Errorf("%w: %w (run with sudo or grant CAP_NET_RAW capability)", ErrICMPPrivileges, err)
	}
	return nil
}
