// Package discovery provides network discovery functionality.
package discovery

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// PortState represents the state of a TCP port.
type PortState string

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

// TCP flags
const (
	tcpFIN = 0x01
	tcpSYN = 0x02
	tcpRST = 0x04
	tcpPSH = 0x08
	tcpACK = 0x10
	tcpURG = 0x20
)

// TCPProber provides raw socket TCP SYN probing functionality.
type TCPProber struct {
	timeout   time.Duration
	srcPort   uint32 // Atomic counter for source ports
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
		srcPort: 40000, // Start from ephemeral port range
		localIP: localIP,
		stopCh:  make(chan struct{}),
	}

	return p, nil
}

// getLocalIP returns the local IP address used for outbound connections.
func getLocalIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf("unexpected address type")
	}
	return localAddr.IP, nil
}

// nextSrcPort returns the next source port to use.
// nolint:unused // Reserved for future raw socket implementation
func (p *TCPProber) nextSrcPort() uint16 {
	// Use ports 40000-60000
	port := atomic.AddUint32(&p.srcPort, 1)
	if port > 60000 {
		atomic.StoreUint32(&p.srcPort, 40000)
		port = 40000
	}
	// #nosec G115 -- port is bounded to 40000-60000, safe for uint16
	return uint16(port)
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
		remaining := time.Until(d)
		if remaining < timeout {
			timeout = remaining
		}
	}

	start := time.Now()

	// Use TCP connect probe (portable, works without raw sockets)
	addr := fmt.Sprintf("%s:%d", ipStr, port)
	dialer := net.Dialer{
		Timeout: timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", addr)
	result.RTT = time.Since(start)

	if err != nil {
		// Analyze the error to determine port state
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			result.State = PortFiltered
		} else if opErr, ok := err.(*net.OpError); ok {
			if syscallErr, ok := opErr.Err.(*syscall.Errno); ok {
				switch *syscallErr {
				case syscall.ECONNREFUSED:
					// RST received - port is closed
					result.State = PortClosed
					result.Flags = tcpRST
				case syscall.EHOSTUNREACH, syscall.ENETUNREACH:
					result.State = PortFiltered
				default:
					result.State = PortFiltered
				}
			} else if opErr.Err != nil {
				// Check for "connection refused" in error string
				errStr := opErr.Err.Error()
				if strings.Contains(errStr, "refused") {
					result.State = PortClosed
					result.Flags = tcpRST
				}
			}
		}
		return result
	}

	// Connection succeeded - port is open
	result.State = PortOpen
	result.Flags = tcpSYN | tcpACK

	// Try to get TTL from connection
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if rawConn, err := tcpConn.SyscallConn(); err == nil {
			//nolint:errcheck // Best-effort TTL extraction
			rawConn.Control(func(fd uintptr) {
				ttl, err := syscall.GetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL)
				if err == nil {
					result.TTL = ttl
				}
			})
		}
	}

	conn.Close()
	return result
}

// ScanPorts probes multiple ports on a single IP concurrently.
func (p *TCPProber) ScanPorts(ctx context.Context, ipStr string, ports []int, workers int) []TCPProbeResult {
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

	for w := 0; w < workers; w++ {
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
	log.Printf("Port scan of %s complete: %d open, %d closed, %d filtered", ipStr, open, closed, filtered)

	return results
}

// ScanPortsOnHosts probes the same ports across multiple hosts.
func (p *TCPProber) ScanPortsOnHosts(ctx context.Context, ips []string, ports []int, workers int) map[string][]TCPProbeResult {
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

	for w := 0; w < workers; w++ {
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

// buildTCPHeader creates a TCP header (for future raw socket implementation).
// nolint:unused // Reserved for future raw socket implementation
func buildTCPHeader(srcPort, dstPort uint16, seq uint32, flags uint8) []byte {
	header := make([]byte, 20)

	binary.BigEndian.PutUint16(header[0:2], srcPort) // Source port
	binary.BigEndian.PutUint16(header[2:4], dstPort) // Destination port
	binary.BigEndian.PutUint32(header[4:8], seq)     // Sequence number
	binary.BigEndian.PutUint32(header[8:12], 0)      // Acknowledgment number
	header[12] = 5 << 4                              // Data offset (5 * 4 = 20 bytes)
	header[13] = flags                               // Flags
	binary.BigEndian.PutUint16(header[14:16], 65535) // Window size
	// Checksum and urgent pointer left as 0

	return header
}

// CheckTCPPrivileges checks if raw TCP sockets can be created.
func CheckTCPPrivileges() error {
	// Try to create a raw socket
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)
	if err != nil {
		return fmt.Errorf("raw TCP socket privileges unavailable: %w", err)
	}
	syscall.Close(fd)
	return nil
}
