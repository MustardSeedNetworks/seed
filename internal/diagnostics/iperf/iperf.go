// Package iperf provides network throughput testing using the iperf3 tool.
//
// This package wraps the iperf3 command-line tool to provide TCP and UDP bandwidth
// testing between two endpoints. It supports both client and server modes, allowing
// The Seed to act as either end of an iperf3 test.
//
// iperf3 modes:
//   - Client mode: Connects to a remote iperf3 server and performs throughput tests
//   - Server mode: Runs an iperf3 server that listens for incoming client connections
//
// Test types:
//   - TCP: Measures maximum achievable TCP throughput with congestion control
//   - UDP: Sends datagrams at a specified rate to measure packet loss and jitter
//   - Bidirectional: Tests both upload and download simultaneously (with --bidir flag)
//
// Features:
//   - Real-time progress updates via JSON output parsing
//   - Server lifecycle management (start, stop, health checks)
//   - Version detection and compatibility checking
//   - Port availability validation before server start
//   - Command injection protection via input validation
//   - Automatic cleanup of zombie processes
//   - Reverse mode support (server sends, client receives)
//
// Requirements:
//   - iperf3 binary must be installed and in PATH or at ./bin/iperf3
//   - Minimum version: 3.17 (for reliable JSON output)
//   - Server mode requires firewall rules allowing inbound connections on test port
//   - Client mode requires network connectivity to target server
//
// Security considerations:
//   - Input validation prevents command injection attacks
//   - Server mode binds to 0.0.0.0 by default (accepts connections from any IP)
//   - No authentication - servers accept connections from any client
//   - Recommended: Use firewall rules to restrict server access to trusted networks
//   - Server processes are tracked and automatically cleaned up on exit
//
// Performance:
//   - TCP tests typically run for 10 seconds by default
//   - UDP tests use 1 Mbps default target rate (configurable)
//   - Results include throughput, retransmits (TCP), packet loss and jitter (UDP)
//   - JSON output parsed in real-time for progress updates
//
// Platform support:
//   - Linux: Full support with optimal performance
//   - macOS: Full support
//   - Windows: Requires iperf3.exe in PATH or ./bin/
//
// Typical usage:
//
//	// Start server mode
//	mgr := iperf.NewManager()
//	if err := mgr.StartServer(5201); err != nil {
//	    log.Fatal(err)
//	}
//	defer mgr.StopServer()
//
//	// Run client test
//	result, err := mgr.RunClient(ctx, "192.168.1.100", 5201, 10)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Throughput: %.2f Mbps\n", result.Throughput)
package iperf

import (
	"context"
	"errors"
	"net"
	"os/exec"
	"regexp"
	"sync"
	"time"
)

const (
	// versionCheckTimeout is the maximum time allowed for iperf3 --version to complete.
	// Short timeout since version check should be instant for a healthy binary.
	versionCheckTimeout = 5 * time.Second

	// serverStartTimeout is the maximum time allowed for iperf3 server to start listening.
	// Includes time to parse command, bind to port, and begin accepting connections.
	serverStartTimeout = 10 * time.Second

	// portCheckTimeout is the maximum time allowed for TCP port availability check.
	// Short timeout since port bind should succeed or fail immediately.
	portCheckTimeout = 2 * time.Second

	// binaryValidationTimeoutSeconds is the timeout for validating iperf3 binary via --version check.
	binaryValidationTimeoutSeconds = 2

	// minSupportedVersion is the minimum iperf3 version required for reliable operation.
	// Version 3.17+ provides stable JSON output format for programmatic parsing.
	// Earlier versions have JSON parsing issues and missing fields.
	minSupportedVersion = "3.17"

	// maxHostnameLength is the maximum allowed length for a hostname per RFC 1035.
	maxHostnameLength = 253

	// minVersionParts is the minimum number of parts expected when parsing version output.
	// iperf3 outputs "iperf X.XX", so we expect at least 2 parts (command name and version).
	minVersionParts = 2

	// progressWarningThreshold is the progress percentage used when entering the "testing" phase.
	progressWarningThreshold = 30

	// progressCriticalThreshold is the progress percentage used when entering the "parsing" phase.
	progressCriticalThreshold = 80

	// progressMaxPercent is the maximum progress percentage (100%).
	progressMaxPercent = 100

	// bytesToMegabits is the conversion factor from bytes per second to megabits per second.
	// 1 megabit = 1,000,000 bits, and 8 bits = 1 byte.
	// So bytes/sec * 8 / 1,000,000 = Mbps, which simplifies to bytes/sec / 1,000,000 for bits/sec.
	bytesToMegabits = 1_000_000

	// searchPathsPrealloc is the preallocation size for the slice storing searched binary paths.
	// This covers typical search locations: embedded, system PATH, and legacy paths.
	searchPathsPrealloc = 8

	// portCheckIntervalMs is the interval in milliseconds between port availability checks.
	portCheckIntervalMs = 100

	// progressConnectingPercent is the progress percentage when client enters "connecting" phase.
	progressConnectingPercent = 10

	// Direction constants for iPerf tests.
	directionDownload      = "download"
	directionUpload        = "upload"
	directionBidirectional = "bidirectional"
)

// validHostnameRegex matches valid hostnames (letters, numbers, dots, hyphens).
var validHostnameRegex = regexp.MustCompile(
	`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`,
)

// validateServer validates the server address to prevent command injection.
func validateServer(server string) error {
	if server == "" {
		return errors.New("server address is required")
	}

	// Check if it's a valid IP address
	if ip := net.ParseIP(server); ip != nil {
		return nil
	}

	// Check if it's a valid hostname
	if len(server) > maxHostnameLength {
		return errors.New("server hostname too long")
	}

	if !validHostnameRegex.MatchString(server) {
		return errors.New("invalid server address: must be a valid IP or hostname")
	}

	return nil
}

// iperf binary path accessor functions use closure-encapsulated state for thread-safe singleton access.
// getIperfBinaryPath returns the cached iperf3 binary path.
// setIperfBinaryPath sets the cached iperf3 binary path.
// _ (clearIperfBinaryPath) resets the cached path to empty (unused but required for pattern).
//
//nolint:gochecknoglobals // Intentional thread-safe singleton using closure pattern
var (
	getIperfBinaryPath, setIperfBinaryPath, _ = func() (
		func() string,
		func(string),
		func(),
	) {
		var (
			mu   sync.RWMutex
			path string
		)

		return func() string {
				mu.RLock()
				defer mu.RUnlock()
				return path
			}, func(p string) {
				mu.Lock()
				defer mu.Unlock()
				path = p
			}, func() {
				mu.Lock()
				defer mu.Unlock()
				path = ""
			}
	}()
)

// ClientConfig holds iperf3 client test configuration.
type ClientConfig struct {
	Server    string `json:"server"`
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`            // "tcp" or "udp"
	Reverse   bool   `json:"reverse"`             // true = download (server sends), false = upload (client sends)
	Direction string `json:"direction,omitempty"` // upload, download, bidirectional
	Duration  int    `json:"duration"`            // seconds
	Parallel  int    `json:"parallel"`            // number of streams
}

// Result contains the iperf3 test results.
type Result struct {
	BitsPerSecond         float64   `json:"bitsPerSecond"`
	Bandwidth             float64   `json:"bandwidth"`   // Mbps
	Transfer              float64   `json:"transfer"`    // MB
	Retransmits           int       `json:"retransmits"` // TCP only
	Jitter                float64   `json:"jitter"`      // UDP only, ms
	LostPackets           int       `json:"lostPackets"` // UDP only
	LostPercent           float64   `json:"lostPercent"` // UDP only
	Protocol              string    `json:"protocol"`
	Direction             string    `json:"direction"` // "upload" or "download"
	Duration              float64   `json:"duration"`  // seconds
	Server                string    `json:"server"`
	Port                  int       `json:"port"`
	Timestamp             time.Time `json:"timestamp"`
	UploadBitsPerSecond   float64   `json:"uploadBitsPerSecond,omitempty"`
	DownloadBitsPerSecond float64   `json:"downloadBitsPerSecond,omitempty"`
	UploadBandwidth       float64   `json:"uploadBandwidth,omitempty"`   // Mbps
	DownloadBandwidth     float64   `json:"downloadBandwidth,omitempty"` // Mbps
	UploadTransfer        float64   `json:"uploadTransfer,omitempty"`    // MB
	DownloadTransfer      float64   `json:"downloadTransfer,omitempty"`  // MB
}

// ServerStatus represents the iperf3 server status.
type ServerStatus struct {
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	PID     int    `json:"pid"`
	Error   string `json:"error,omitempty"`
}

// ClientStatus represents the client test status.
type ClientStatus struct {
	Running  bool    `json:"running"`
	Phase    string  `json:"phase"` // "idle", "connecting", "testing", "complete"
	Progress float64 `json:"progress"`
}

// iperfJSON is the structure of iperf3 -J output.
type iperfJSON struct {
	Start struct {
		Connected []struct {
			Socket     int    `json:"socket"`
			LocalHost  string `json:"local_host"`
			LocalPort  int    `json:"local_port"`
			RemoteHost string `json:"remote_host"`
			RemotePort int    `json:"remote_port"`
		} `json:"connected"`
		TestStart struct {
			Protocol   string `json:"protocol"`
			NumStreams int    `json:"num_streams"`
			Duration   int    `json:"duration"`
			Reverse    int    `json:"reverse"`
		} `json:"test_start"`
	} `json:"start"`
	End struct {
		SumSent struct {
			Seconds       float64 `json:"seconds"`
			Bytes         float64 `json:"bytes"`
			BitsPerSecond float64 `json:"bits_per_second"`
			Retransmits   int     `json:"retransmits"`
		} `json:"sum_sent"`
		SumReceived struct {
			Seconds       float64 `json:"seconds"`
			Bytes         float64 `json:"bytes"`
			BitsPerSecond float64 `json:"bits_per_second"`
		} `json:"sum_received"`
		Sum struct {
			Seconds       float64 `json:"seconds"`
			Bytes         float64 `json:"bytes"`
			BitsPerSecond float64 `json:"bits_per_second"`
			JitterMs      float64 `json:"jitter_ms"`
			LostPackets   int     `json:"lost_packets"`
			Packets       int     `json:"packets"`
			LostPercent   float64 `json:"lost_percent"`
		} `json:"sum"`
	} `json:"end"`
	Error string `json:"error"`
}

// Manager handles iperf3 client and server operations.
type Manager struct {
	mu           sync.RWMutex
	serverStatus ServerStatus
	clientStatus ClientStatus
	lastResult   *Result
	serverCmd    *exec.Cmd
	serverCancel context.CancelFunc
}

// NewManager creates a new iperf3 manager.
func NewManager() *Manager {
	return &Manager{
		clientStatus: ClientStatus{Phase: "idle"},
	}
}

// GetServerStatus returns the current server status.
func (m *Manager) GetServerStatus() ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.serverStatus
}

// GetClientStatus returns the current client status.
func (m *Manager) GetClientStatus() ClientStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clientStatus
}

// GetLastResult returns the last test result.
func (m *Manager) GetLastResult() *Result {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastResult
}
