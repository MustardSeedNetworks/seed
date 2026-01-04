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
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/validation"
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

	// minSupportedVersion is the minimum iperf3 version required for reliable operation.
	// Version 3.17+ provides stable JSON output format for programmatic parsing.
	// Earlier versions have JSON parsing issues and missing fields.
	minSupportedVersion = "3.17"

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
	if len(server) > 253 {
		return errors.New("server hostname too long")
	}

	if !validHostnameRegex.MatchString(server) {
		return errors.New("invalid server address: must be a valid IP or hostname")
	}

	return nil
}

// iperfBinaryPath caches the resolved iperf3 binary path.
var iperfBinaryPath string

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

// findIperf3Binary locates the iperf3 binary using a robust detection strategy:
//  1. Try embedded binary (extracted to user cache directory).
//  2. Search system PATH using exec.LookPath (the proper way).
//  3. Return detailed error with OS-specific install instructions if not found.
func findIperf3Binary() (string, error) {
	// Return cached path if already found
	if iperfBinaryPath != "" {
		return iperfBinaryPath, nil
	}

	searchedPaths := make([]string, 0, 8) // Preallocate for typical search paths
	var embeddedErr, systemErr error

	// Strategy 1: Try embedded binary (extract to cache if needed)
	if HasEmbeddedBinary() {
		if path, err := extractEmbeddedBinary(); err == nil {
			// Validate the extracted binary works
			if validateBinary(path) {
				iperfBinaryPath = path
				logging.GetLogger().Info("Using embedded iperf3 binary", "path", path, "version", EmbeddedVersion)
				return path, nil
			}
			logging.GetLogger().Warn("Extracted iperf3 binary failed validation", "path", path)
		} else {
			embeddedErr = err
			logging.GetLogger().Debug("Failed to extract embedded iperf3", "error", err)
		}
	}

	// Strategy 2: Search system PATH using exec.LookPath
	// This is the proper way to find executables - it searches the entire PATH
	if path, err := findSystemIperf3(); err == nil {
		if validateBinary(path) {
			iperfBinaryPath = path
			logging.GetLogger().Info("Using system iperf3 binary", "path", path)
			return path, nil
		}
		logging.GetLogger().Warn("System iperf3 binary failed validation", "path", path)
	} else {
		systemErr = err
	}

	// Strategy 3: Check legacy paths for backwards compatibility
	// These are less reliable but may help in edge cases
	legacyPaths := getLegacyPaths()
	for _, path := range legacyPaths {
		searchedPaths = append(searchedPaths, path)
		if info, err := os.Stat(path); err == nil && info.Mode()&0o111 != 0 {
			if validateBinary(path) {
				iperfBinaryPath = path
				logging.GetLogger().Info("Using iperf3 from legacy path", "path", path)
				return path, nil
			}
		}
	}

	// Not found - return detailed error with install instructions.
	return "", &NotFoundError{
		SearchedPaths: searchedPaths,
		SystemError:   systemErr,
		EmbeddedError: embeddedErr,
	}
}

// getLegacyPaths returns platform-specific paths to check as a last resort.
func getLegacyPaths() []string {
	var paths []string

	// Paths relative to executable
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		paths = append(paths,
			filepath.Join(execDir, "bin", "iperf3"),
			filepath.Join(execDir, "iperf3"),
		)
	}

	// Paths relative to working directory (for development)
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths,
			filepath.Join(cwd, "bin", "iperf3"),
		)
	}

	return paths
}

// validateBinary checks if the binary at the given path is a valid iperf3 executable.
func validateBinary(path string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "--version")
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	// Should output something like "iperf 3.x"
	return strings.Contains(string(out), "iperf")
}

// CheckInstalled checks if iperf3 is available.
func CheckInstalled() error {
	_, err := findIperf3Binary()
	return err
}

// GetVersion returns the installed iperf3 version.
func GetVersion() (string, error) {
	binaryPath, err := findIperf3Binary()
	if err != nil {
		return "", err
	}

	// Use timeout context to prevent indefinite blocking
	ctx, cancel := context.WithTimeout(context.Background(), versionCheckTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "--version")
	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("iperf3 version check timed out after %v", versionCheckTimeout)
		}
		return "", fmt.Errorf("failed to get iperf3 version: %w", err)
	}

	// Output is like "iperf 3.16 (cJSON 1.7.17)\n..."
	// Extract just the version number (e.g., "3.16")
	lines := strings.Split(string(out), "\n")
	if len(lines) > 0 {
		line := strings.TrimSpace(lines[0])
		// Parse "iperf X.XX" format
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			return "v" + parts[1], nil
		}
		return line, nil
	}
	return "unknown", nil
}

// ValidateVersion checks if the installed iperf3 version meets minimum requirements.
func ValidateVersion() error {
	version, err := GetVersion()
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	// Remove 'v' prefix for comparison
	version = strings.TrimPrefix(version, "v")

	// Compare version strings (simple lexicographic comparison works for x.y format)
	if compareVersions(version, minSupportedVersion) < 0 {
		return fmt.Errorf("iperf3 version %s is below minimum supported version %s", version, minSupportedVersion)
	}

	return nil
}

// compareVersions compares two version strings in format "x.y" or "x.y.z"
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2.
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part
	for i := 0; i < len(parts1) || i < len(parts2); i++ {
		var n1, n2 int

		if i < len(parts1) {
			_, _ = fmt.Sscanf(parts1[i], "%d", &n1)
		}
		if i < len(parts2) {
			_, _ = fmt.Sscanf(parts2[i], "%d", &n2)
		}

		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}

	return 0
}

// waitForPortReady checks if a TCP port is ready to accept connections.
func waitForPortReady(port int, timeout time.Duration) error {
	// Validate port number
	if err := validation.ValidatePort(port); err != nil {
		return err
	}

	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("port %d not ready after %v", port, timeout)
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

// StartServer starts the iperf3 server.
func (m *Manager) StartServer(port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.serverStatus.Running {
		return fmt.Errorf("server already running on port %d", m.serverStatus.Port)
	}

	// Validate port number
	if err := validation.ValidatePort(port); err != nil {
		return err
	}

	binaryPath, err := findIperf3Binary()
	if err != nil {
		return err
	}

	// Use timeout context for server startup monitoring
	ctx, cancel := context.WithTimeout(context.Background(), serverStartTimeout)
	m.serverCancel = cancel

	// Start iperf3 server: iperf3 -s -p <port>

	cmd := exec.CommandContext(ctx, binaryPath, "-s", "-p", strconv.Itoa(port))
	if startErr := cmd.Start(); startErr != nil {
		cancel()
		return fmt.Errorf("failed to start iperf3 server: %w", startErr)
	}

	// Wait for port to be ready
	if portErr := waitForPortReady(port, portCheckTimeout); portErr != nil {
		// Kill the process if port check fails
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		cancel()
		return fmt.Errorf("iperf3 server failed to start listening: %w", portErr)
	}

	m.serverCmd = cmd
	m.serverStatus = ServerStatus{
		Running: true,
		Port:    port,
		PID:     cmd.Process.Pid,
	}

	// Monitor the server process
	go func() {
		waitErr := cmd.Wait()
		m.mu.Lock()
		m.serverStatus.Running = false
		if waitErr != nil && ctx.Err() == nil {
			m.serverStatus.Error = waitErr.Error()
		}
		m.mu.Unlock()
	}()

	return nil
}

// StopServer stops the iperf3 server.
func (m *Manager) StopServer() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.serverStatus.Running {
		return errors.New("server not running")
	}

	if m.serverCancel != nil {
		m.serverCancel()
	}

	if m.serverCmd != nil && m.serverCmd.Process != nil {
		if err := m.serverCmd.Process.Kill(); err != nil {
			// Log the error, but don't fail, as we are trying to stop the server
			// and it might already be dead or unreachable.
			logging.GetLogger().Warn("Error killing iperf3 server process", "pid", m.serverCmd.Process.Pid, "error", err)
		}
	}

	m.serverStatus = ServerStatus{Running: false}
	m.serverCmd = nil
	m.serverCancel = nil

	return nil
}

// RunClient runs an iperf3 client test.
func (m *Manager) RunClient(ctx context.Context, config *ClientConfig) (*Result, error) {
	if err := validateServer(config.Server); err != nil {
		return nil, err
	}
	config.Protocol = strings.ToLower(config.Protocol)
	config.Direction = strings.ToLower(config.Direction)

	if err := m.startClientTest(); err != nil {
		return nil, err
	}
	defer m.resetClientStatus()

	binaryPath, err := findIperf3Binary()
	if err != nil {
		return nil, err
	}

	setClientDefaults(config)
	direction := normalizeDirection(config)
	args := buildClientArgs(config, direction)

	m.updateClientProgress("testing", 30)

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		var iperfOut iperfJSON
		if jsonErr := json.Unmarshal(output, &iperfOut); jsonErr == nil && iperfOut.Error != "" {
			return nil, fmt.Errorf("iperf3 error: %s", iperfOut.Error)
		}
		return nil, fmt.Errorf("iperf3 failed: %w", err)
	}

	m.updateClientProgress("parsing", 80)

	var iperfOut iperfJSON
	if unmarshalErr := json.Unmarshal(output, &iperfOut); unmarshalErr != nil {
		return nil, fmt.Errorf("failed to parse iperf3 output: %w", unmarshalErr)
	}
	if iperfOut.Error != "" {
		return nil, fmt.Errorf("iperf3 error: %s", iperfOut.Error)
	}

	result := parseClientResult(&iperfOut, config, direction)

	m.mu.Lock()
	m.lastResult = result
	m.clientStatus.Phase = "complete"
	m.clientStatus.Progress = 100
	m.mu.Unlock()

	return result, nil
}

func (m *Manager) startClientTest() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.clientStatus.Running {
		return errors.New("test already in progress")
	}
	m.clientStatus = ClientStatus{Running: true, Phase: "connecting", Progress: 10}
	return nil
}

func (m *Manager) resetClientStatus() {
	m.mu.Lock()
	m.clientStatus = ClientStatus{Running: false, Phase: "idle", Progress: 0}
	m.mu.Unlock()
}

func (m *Manager) updateClientProgress(phase string, progress float64) {
	m.mu.Lock()
	m.clientStatus.Phase = phase
	m.clientStatus.Progress = progress
	m.mu.Unlock()
}

func setClientDefaults(config *ClientConfig) {
	if config.Port == 0 {
		config.Port = 5201
	}
	if config.Duration == 0 {
		config.Duration = 10
	}
	if config.Parallel == 0 {
		config.Parallel = 1
	}
	if config.Protocol == "" {
		config.Protocol = "tcp"
	}
}

func normalizeDirection(config *ClientConfig) string {
	direction := strings.ToLower(config.Direction)
	if direction == "" {
		if config.Reverse {
			direction = directionDownload
		} else {
			direction = directionUpload
		}
	}
	if direction != directionDownload && direction != directionBidirectional {
		direction = directionUpload
	}
	config.Direction = direction
	return direction
}

func buildClientArgs(config *ClientConfig, direction string) []string {
	args := []string{
		"-c", config.Server,
		"-p", strconv.Itoa(config.Port),
		"-t", strconv.Itoa(config.Duration),
		"-P", strconv.Itoa(config.Parallel),
		"-J",
	}
	if config.Protocol == "udp" {
		args = append(args, "-u", "-b", "0")
	}
	switch direction {
	case directionDownload:
		config.Reverse = true
		args = append(args, "-R")
	case directionBidirectional:
		args = append(args, "--bidir")
		config.Reverse = false
	default:
		config.Reverse = false
	}
	return args
}

func parseClientResult(iperfOut *iperfJSON, config *ClientConfig, direction string) *Result {
	result := &Result{
		Protocol: config.Protocol, Server: config.Server, Port: config.Port,
		Timestamp: time.Now(), Direction: direction,
	}
	switch direction {
	case directionDownload:
		result.BitsPerSecond = iperfOut.End.SumReceived.BitsPerSecond
		result.Bandwidth = iperfOut.End.SumReceived.BitsPerSecond / 1_000_000
		result.Transfer = iperfOut.End.SumReceived.Bytes / 1_000_000
		result.Duration = iperfOut.End.SumReceived.Seconds
	case directionBidirectional:
		result.DownloadBitsPerSecond = iperfOut.End.SumReceived.BitsPerSecond
		result.DownloadBandwidth = iperfOut.End.SumReceived.BitsPerSecond / 1_000_000
		result.DownloadTransfer = iperfOut.End.SumReceived.Bytes / 1_000_000
		result.UploadBitsPerSecond = iperfOut.End.SumSent.BitsPerSecond
		result.UploadBandwidth = iperfOut.End.SumSent.BitsPerSecond / 1_000_000
		result.UploadTransfer = iperfOut.End.SumSent.Bytes / 1_000_000
		result.BitsPerSecond = result.DownloadBitsPerSecond
		result.Bandwidth = result.DownloadBandwidth
		result.Transfer = result.DownloadTransfer
		result.Duration = iperfOut.End.Sum.Seconds
		result.Retransmits = iperfOut.End.SumSent.Retransmits
	default:
		result.BitsPerSecond = iperfOut.End.SumSent.BitsPerSecond
		result.Bandwidth = iperfOut.End.SumSent.BitsPerSecond / 1_000_000
		result.Transfer = iperfOut.End.SumSent.Bytes / 1_000_000
		result.Duration = iperfOut.End.SumSent.Seconds
		result.Retransmits = iperfOut.End.SumSent.Retransmits
	}
	if config.Protocol == "udp" {
		result.Jitter = iperfOut.End.Sum.JitterMs
		result.LostPackets = iperfOut.End.Sum.LostPackets
		result.LostPercent = iperfOut.End.Sum.LostPercent
	}
	return result
}
