// Package discovery provides network discovery functionality.
package discovery

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

// PortState represents the state of a TCP port.
type PortState string

// TCP port state constants indicating probe results.
const (
	PortOpen     PortState = "open"     // SYN/ACK received
	PortClosed   PortState = "closed"   // RST received
	PortFiltered PortState = "filtered" // No response (timeout)
)

// TCPProbeResult contains the result of a TCP probe.
type TCPProbeResult struct {
	IP    string        `json:"ip"`
	Port  int           `json:"port"`
	State PortState     `json:"state"`
	TTL   int           `json:"ttl"`
	RTT   time.Duration `json:"rtt"`
	Flags uint8         `json:"flags,omitempty"` // TCP flags in response
	Error error         `json:"error,omitempty"`
}

// TCP flags used for probe result analysis.
const (
	tcpSYN = 0x02
	tcpRST = 0x04
	tcpACK = 0x10
)

// TCPProber provides TCP connect probing functionality.
type TCPProber struct {
	timeout   time.Duration
	localIP   net.IP
	stopCh    chan struct{}
	stopped   bool
	stoppedMu sync.Mutex
}

// NewTCPProber creates a new TCP prober.
// Requires root privileges or CAP_NET_RAW capability.
func NewTCPProber(timeout time.Duration) (*TCPProber, error) {
	if timeout == 0 {
		timeout = defaultTimeout
	}

	// Get local IP for source address
	localIP, err := getLocalIP()
	if err != nil {
		return nil, fmt.Errorf("failed to get local IP: %w", err)
	}

	p := &TCPProber{
		timeout: timeout,
		localIP: localIP,
		stopCh:  make(chan struct{}),
	}

	return p, nil
}

// getLocalIP returns the local IP address used for outbound connections.
func getLocalIP() (net.IP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "udp", "8.8.8.8:53")
	if err != nil {
		return nil, fmt.Errorf("failed to dial UDP: %w", err)
	}
	defer func() { _ = conn.Close() }()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, errors.New("unexpected address type")
	}
	return localAddr.IP, nil
}

// Close stops the prober.
func (p *TCPProber) Close() error {
	p.stoppedMu.Lock()
	defer p.stoppedMu.Unlock()

	if !p.stopped {
		p.stopped = true
		close(p.stopCh)
	}
	return nil
}

// ProbeTCP sends a TCP SYN packet to the specified IP:port and analyzes the response.
// analyzeDialError determines port state from a connection error.
func analyzeDialError(err error, result *TCPProbeResult) {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		result.State = PortFiltered
		return
	}
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		return
	}
	var syscallErr *syscall.Errno
	if errors.As(opErr.Err, &syscallErr) {
		//nolint:exhaustive // syscall.Errno has too many cases, default handles the rest
		switch *syscallErr {
		case syscall.ECONNREFUSED:
			result.State = PortClosed
			result.Flags = tcpRST
		case syscall.EHOSTUNREACH, syscall.ENETUNREACH:
			result.State = PortFiltered
		default:
			result.State = PortFiltered
		}
	} else if opErr.Err != nil && strings.Contains(opErr.Err.Error(), "refused") {
		result.State = PortClosed
		result.Flags = tcpRST
	}
}

// extractTTL attempts to extract TTL from a TCP connection.
func extractTTL(conn net.Conn) int {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return -1
	}
	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return -1
	}
	ttl := -1
	err = rawConn.Control(func(fd uintptr) {
		if v, sockErr := syscall.GetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL); sockErr == nil {
			ttl = v
		}
	})
	if err != nil {
		return -1
	}
	return ttl
}

// ProbeTCP performs a TCP connect probe to determine if a port is open.
// This is a connect-based probe (not raw SYN) for portability.
func (p *TCPProber) ProbeTCP(ctx context.Context, ipStr string, port int) TCPProbeResult {
	result := TCPProbeResult{
		IP:    ipStr,
		Port:  port,
		State: PortFiltered,
		TTL:   -1,
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		result.Error = &net.ParseError{Type: "IP address", Text: ipStr}
		return result
	}

	// Use context timeout or prober timeout
	timeout := p.timeout
	if d, ok := ctx.Deadline(); ok {
		if remaining := time.Until(d); remaining < timeout {
			timeout = remaining
		}
	}

	start := time.Now()
	addr := fmt.Sprintf("%s:%d", ipStr, port)
	dialer := net.Dialer{Timeout: timeout}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	result.RTT = time.Since(start)

	if err != nil {
		analyzeDialError(err, &result)
		return result
	}

	// Connection succeeded - port is open
	result.State = PortOpen
	result.Flags = tcpSYN | tcpACK
	result.TTL = extractTTL(conn)
	_ = conn.Close()
	return result
}

// ScanPorts probes multiple ports on a single IP concurrently.
func (p *TCPProber) ScanPorts(
	ctx context.Context,
	ipStr string,
	ports []int,
	workers int,
) []TCPProbeResult {
	if workers <= 0 {
		workers = 20
	}

	results := make([]TCPProbeResult, len(ports))
	resultsMu := sync.Mutex{}

	// Create work channel
	work := make(chan int, len(ports))
	for i := range ports {
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
				case <-p.stopCh:
					return
				default:
				}

				port := ports[idx]
				result := p.ProbeTCP(ctx, ipStr, port)

				resultsMu.Lock()
				results[idx] = result
				resultsMu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Log summary
	open, closed, filtered := 0, 0, 0
	for _, r := range results {
		switch r.State {
		case PortOpen:
			open++
		case PortClosed:
			closed++
		case PortFiltered:
			filtered++
		}
	}
	logging.GetLogger().InfoContext(ctx,
		"Port scan complete",
		"ip",
		ipStr,
		"open",
		open,
		"closed",
		closed,
		"filtered",
		filtered,
	)

	return results
}

// ScanPortsOnHosts probes the same ports across multiple hosts.
func (p *TCPProber) ScanPortsOnHosts(
	ctx context.Context,
	ips []string,
	ports []int,
	workers int,
) map[string][]TCPProbeResult {
	if workers <= 0 {
		workers = 50
	}

	results := make(map[string][]TCPProbeResult)
	resultsMu := sync.Mutex{}

	type task struct {
		ip   string
		port int
	}

	// Create work channel
	tasks := make([]task, 0, len(ips)*len(ports))
	for _, ip := range ips {
		for _, port := range ports {
			tasks = append(tasks, task{ip: ip, port: port})
		}
	}

	work := make(chan task, len(tasks))
	for _, t := range tasks {
		work <- t
	}
	close(work)

	// Initialize result maps
	for _, ip := range ips {
		results[ip] = make([]TCPProbeResult, len(ports))
	}

	// Create port index map
	portIdx := make(map[int]int)
	for i, port := range ports {
		portIdx[port] = i
	}

	// Create worker pool
	var wg sync.WaitGroup
	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()
			for t := range work {
				select {
				case <-ctx.Done():
					return
				case <-p.stopCh:
					return
				default:
				}

				result := p.ProbeTCP(ctx, t.ip, t.port)

				resultsMu.Lock()
				idx := portIdx[t.port]
				results[t.ip][idx] = result
				resultsMu.Unlock()
			}
		}()
	}

	wg.Wait()
	return results
}

// CommonPorts contains commonly scanned ports.
var CommonPorts = []int{
	21,   // FTP
	22,   // SSH
	23,   // Telnet
	25,   // SMTP
	53,   // DNS
	80,   // HTTP
	110,  // POP3
	111,  // RPCBind
	135,  // MSRPC
	139,  // NetBIOS
	143,  // IMAP
	443,  // HTTPS
	445,  // SMB
	993,  // IMAPS
	995,  // POP3S
	1723, // PPTP
	3306, // MySQL
	3389, // RDP
	5432, // PostgreSQL
	5900, // VNC
	8080, // HTTP Alt
	8443, // HTTPS Alt
}

// WebPorts contains common web server ports.
var WebPorts = []int{80, 443, 8080, 8443, 8000, 8888}

// CheckTCPPrivileges checks if raw TCP sockets can be created.
func CheckTCPPrivileges() error {
	// Try to create a raw socket
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)
	if err != nil {
		return fmt.Errorf("raw TCP socket privileges unavailable: %w", err)
	}
	_ = syscall.Close(fd)
	return nil
}
