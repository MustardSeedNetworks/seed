//go:build windows

package discovery

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
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

// TCP probe constants.
const tcpProbeDialTimeoutS = 5 // Default timeout for local IP detection dial

// TCPProber provides TCP connect probing functionality.
type TCPProber struct {
	timeout   time.Duration
	localIP   net.IP
	stopCh    chan struct{}
	stopped   bool
	stoppedMu sync.Mutex
}

// NewTCPProber creates a new TCP prober.
// On Windows, this uses standard Go net.Dial which works without elevated privileges.
func NewTCPProber(timeout time.Duration) (*TCPProber, error) {
	if timeout == 0 {
		timeout = defaultTimeout
	}

	// Get local IP for source address
	localIP, err := getLocalIPWindows()
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

// getLocalIPWindows returns the local IP address used for outbound connections.
func getLocalIPWindows() (net.IP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), tcpProbeDialTimeoutS*time.Second)
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

// analyzeDialErrorWindows determines port state from a connection error.
func analyzeDialErrorWindows(err error, result *TCPProbeResult) {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		result.State = PortFiltered
		return
	}
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		return
	}
	// Check error message for connection refused
	if opErr.Err != nil && strings.Contains(opErr.Err.Error(), "refused") {
		result.State = PortClosed
		result.Flags = tcpRST
		return
	}
	// Default to filtered for other errors
	result.State = PortFiltered
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
		analyzeDialErrorWindows(err, &result)
		return result
	}

	// Connection succeeded - port is open
	result.State = PortOpen
	result.Flags = tcpSYN | tcpACK
	result.TTL = -1 // TTL extraction not available on Windows without raw sockets
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
	var wg sync.WaitGroup
	portCh := make(chan int, len(ports))

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case <-p.stopCh:
					return
				case port, ok := <-portCh:
					if !ok {
						return
					}
					// Find the index for this port
					for idx, prt := range ports {
						if prt == port {
							results[idx] = p.ProbeTCP(ctx, ipStr, port)
							break
						}
					}
				}
			}
		}()
	}

	// Send ports to workers
	for _, port := range ports {
		select {
		case <-ctx.Done():
			break
		case portCh <- port:
		}
	}
	close(portCh)

	wg.Wait()
	return results
}

// GetCommonPorts returns a list of commonly scanned ports.
func GetCommonPorts() []int {
	return []int{
		21, 22, 23, 25, 53, 80, 110, 111, 135, 139, 143, 443, 445, 993, 995,
		1723, 3306, 3389, 5432, 5900, 6379, 8080, 8443,
	}
}

// GetWebPorts returns common web server ports.
func GetWebPorts() []int {
	return []int{80, 443, 8080, 8443}
}

// CheckTCPPrivileges checks if we have sufficient privileges for TCP probing.
// On Windows with standard net.Dial, no special privileges are required.
func CheckTCPPrivileges() error {
	return nil
}
