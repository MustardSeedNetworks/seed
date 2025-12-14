package iperf

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/krisarmstrong/luminetiq/internal/validation"
)

const (
	// Timeout durations for iperf3 operations
	versionCheckTimeout = 5 * time.Second
	serverStartTimeout  = 10 * time.Second
	portCheckTimeout    = 2 * time.Second
	expectedVersion     = "3.20"
)

// validHostnameRegex matches valid hostnames (letters, numbers, dots, hyphens)
var validHostnameRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

// validateServer validates the server address to prevent command injection.
func validateServer(server string) error {
	if server == "" {
		return fmt.Errorf("server address is required")
	}

	// Check if it's a valid IP address
	if ip := net.ParseIP(server); ip != nil {
		return nil
	}

	// Check if it's a valid hostname
	if len(server) > 253 {
		return fmt.Errorf("server hostname too long")
	}

	if !validHostnameRegex.MatchString(server) {
		return fmt.Errorf("invalid server address: must be a valid IP or hostname")
	}

	return nil
}

// iperfBinaryPath caches the resolved iperf3 binary path
var iperfBinaryPath string

// ClientConfig holds iperf3 client test configuration
type ClientConfig struct {
	Server    string `json:"server"`
	Port      int    `json:"port"`
	Protocol  string `json:"protocol"`            // "tcp" or "udp"
	Reverse   bool   `json:"reverse"`             // true = download (server sends), false = upload (client sends)
	Direction string `json:"direction,omitempty"` // upload, download, bidirectional
	Duration  int    `json:"duration"`            // seconds
	Parallel  int    `json:"parallel"`            // number of streams
}

// Result contains the iperf3 test results
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

// ServerStatus represents the iperf3 server status
type ServerStatus struct {
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	PID     int    `json:"pid"`
	Error   string `json:"error,omitempty"`
}

// ClientStatus represents the client test status
type ClientStatus struct {
	Running  bool    `json:"running"`
	Phase    string  `json:"phase"` // "idle", "connecting", "testing", "complete"
	Progress float64 `json:"progress"`
}

// iperfJSON is the structure of iperf3 -J output
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

// Manager handles iperf3 client and server operations
type Manager struct {
	mu           sync.RWMutex
	serverStatus ServerStatus
	clientStatus ClientStatus
	lastResult   *Result
	serverCmd    *exec.Cmd
	serverCancel context.CancelFunc
}

// NewManager creates a new iperf3 manager
func NewManager() *Manager {
	return &Manager{
		clientStatus: ClientStatus{Phase: "idle"},
	}
}

// findIperf3Binary locates the iperf3 binary, checking bundled paths first
func findIperf3Binary() (string, error) {
	// Return cached path if already found
	if iperfBinaryPath != "" {
		return iperfBinaryPath, nil
	}

	// Track all searched paths for better error message
	var searchedPaths []string

	// Get executable path to find bundled binary
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)

		// Check bundled locations relative to executable
		bundledPaths := []string{
			filepath.Join(execDir, "bin", "iperf3"),
			filepath.Join(execDir, "iperf3"),
			filepath.Join(execDir, "..", "bin", "iperf3"),
		}

		for _, path := range bundledPaths {
			searchedPaths = append(searchedPaths, path)
			if _, err := os.Stat(path); err == nil {
				// Verify binary is executable
				if info, err := os.Stat(path); err == nil && info.Mode()&0111 != 0 {
					iperfBinaryPath = path
					return path, nil
				}
			}
		}
	}

	// Check relative to working directory (for development)
	cwd, err := os.Getwd()
	if err == nil {
		devPaths := []string{
			filepath.Join(cwd, "bin", "iperf3"),
			filepath.Join(cwd, "iperf3"),
		}

		for _, path := range devPaths {
			searchedPaths = append(searchedPaths, path)
			if _, err := os.Stat(path); err == nil {
				if info, err := os.Stat(path); err == nil && info.Mode()&0111 != 0 {
					iperfBinaryPath = path
					return path, nil
				}
			}
		}
	}

	// Check common system paths explicitly (systemd services may have minimal PATH)
	systemPaths := []string{
		"/usr/local/bin/iperf3",
		"/usr/bin/iperf3",
		"/bin/iperf3",
		"/usr/local/sbin/iperf3",
		"/usr/sbin/iperf3",
	}
	for _, path := range systemPaths {
		searchedPaths = append(searchedPaths, path)
		if _, err := os.Stat(path); err == nil {
			if info, err := os.Stat(path); err == nil && info.Mode()&0111 != 0 {
				iperfBinaryPath = path
				return path, nil
			}
		}
	}

	// Fall back to system PATH
	path, err := exec.LookPath("iperf3")
	if err != nil {
		// Build detailed error message
		errMsg := "iperf3 not found. Searched paths:\n"
		for _, p := range searchedPaths {
			errMsg += fmt.Sprintf("  - %s\n", p)
		}
		errMsg += "\nTo build a bundled iperf3 binary, run:\n  scripts/build-iperf3.sh"
		return "", fmt.Errorf("%s", errMsg)
	}

	iperfBinaryPath = path
	return path, nil
}

// CheckInstalled checks if iperf3 is available
func CheckInstalled() error {
	_, err := findIperf3Binary()
	return err
}

// GetVersion returns the installed iperf3 version
func GetVersion() (string, error) {
	binaryPath, err := findIperf3Binary()
	if err != nil {
		return "", err
	}

	// Use timeout context to prevent indefinite blocking
	ctx, cancel := context.WithTimeout(context.Background(), versionCheckTimeout)
	defer cancel()

	//nolint:gosec // G204: binaryPath is from findIperf3Binary() which validates the path
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

// ValidateVersion checks if the installed iperf3 version matches the expected version
func ValidateVersion() error {
	version, err := GetVersion()
	if err != nil {
		return fmt.Errorf("failed to get version: %w", err)
	}

	// Remove 'v' prefix for comparison
	version = strings.TrimPrefix(version, "v")
	if version != expectedVersion {
		return fmt.Errorf("iperf3 version mismatch: expected v%s, got v%s", expectedVersion, version)
	}

	return nil
}

// waitForPortReady checks if a TCP port is ready to accept connections
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
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("port %d not ready after %v", port, timeout)
}

// GetServerStatus returns the current server status
func (m *Manager) GetServerStatus() ServerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.serverStatus
}

// GetClientStatus returns the current client status
func (m *Manager) GetClientStatus() ClientStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clientStatus
}

// GetLastResult returns the last test result
func (m *Manager) GetLastResult() *Result {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastResult
}

// StartServer starts the iperf3 server
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
	//nolint:gosec // G204: binaryPath is from findIperf3Binary() which validates the path
	cmd := exec.CommandContext(ctx, binaryPath, "-s", "-p", fmt.Sprintf("%d", port))
	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("failed to start iperf3 server: %w", err)
	}

	// Wait for port to be ready
	if err := waitForPortReady(port, portCheckTimeout); err != nil {
		// Kill the process if port check fails
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		cancel()
		return fmt.Errorf("iperf3 server failed to start listening: %w", err)
	}

	m.serverCmd = cmd
	m.serverStatus = ServerStatus{
		Running: true,
		Port:    port,
		PID:     cmd.Process.Pid,
	}

	// Monitor the server process
	go func() {
		err := cmd.Wait()
		m.mu.Lock()
		m.serverStatus.Running = false
		if err != nil && ctx.Err() == nil {
			m.serverStatus.Error = err.Error()
		}
		m.mu.Unlock()
	}()

	return nil
}

// StopServer stops the iperf3 server
func (m *Manager) StopServer() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.serverStatus.Running {
		return fmt.Errorf("server not running")
	}

	if m.serverCancel != nil {
		m.serverCancel()
	}

	if m.serverCmd != nil && m.serverCmd.Process != nil {
		if err := m.serverCmd.Process.Kill(); err != nil {
			// Log the error, but don't fail, as we are trying to stop the server
			// and it might already be dead or unreachable.
			fmt.Printf("Error killing iperf3 server process (PID %d): %v\n", m.serverCmd.Process.Pid, err)
		}
	}

	m.serverStatus = ServerStatus{Running: false}
	m.serverCmd = nil
	m.serverCancel = nil

	return nil
}

// RunClient runs an iperf3 client test
func (m *Manager) RunClient(ctx context.Context, config *ClientConfig) (*Result, error) {
	// Validate server address to prevent command injection
	if err := validateServer(config.Server); err != nil {
		return nil, err
	}
	config.Protocol = strings.ToLower(config.Protocol)
	config.Direction = strings.ToLower(config.Direction)

	m.mu.Lock()
	if m.clientStatus.Running {
		m.mu.Unlock()
		return nil, fmt.Errorf("test already in progress")
	}
	m.clientStatus = ClientStatus{Running: true, Phase: "connecting", Progress: 10}
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.clientStatus = ClientStatus{Running: false, Phase: "idle", Progress: 0}
		m.mu.Unlock()
	}()

	binaryPath, err := findIperf3Binary()
	if err != nil {
		return nil, err
	}

	// Set defaults
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

	direction := strings.ToLower(config.Direction)
	if direction == "" {
		if config.Reverse {
			direction = "download"
		} else {
			direction = "upload"
		}
	}
	if direction != "download" && direction != "bidirectional" {
		direction = "upload"
	}
	config.Direction = direction

	// Build command
	args := []string{
		"-c", config.Server,
		"-p", fmt.Sprintf("%d", config.Port),
		"-t", fmt.Sprintf("%d", config.Duration),
		"-P", fmt.Sprintf("%d", config.Parallel),
		"-J", // JSON output
	}

	if config.Protocol == "udp" {
		args = append(args, "-u", "-b", "0") // Unlimited bandwidth for UDP
	}

	switch direction {
	case "download":
		config.Reverse = true
		args = append(args, "-R") // Reverse mode (server sends, client receives)
	case "bidirectional":
		// Bidirectional test (client <-> server)
		args = append(args, "--bidir")
		config.Reverse = false
	default:
		config.Reverse = false
	}

	m.mu.Lock()
	m.clientStatus.Phase = "testing"
	m.clientStatus.Progress = 30
	m.mu.Unlock()

	// Run iperf3
	//nolint:gosec // G204: binaryPath is from findIperf3Binary() and args are validated
	cmd := exec.CommandContext(ctx, binaryPath, args...)
	output, err := cmd.Output()
	if err != nil {
		// Try to parse error from output
		var iperfOut iperfJSON
		if jsonErr := json.Unmarshal(output, &iperfOut); jsonErr == nil && iperfOut.Error != "" {
			return nil, fmt.Errorf("iperf3 error: %s", iperfOut.Error)
		}
		return nil, fmt.Errorf("iperf3 failed: %w", err)
	}

	m.mu.Lock()
	m.clientStatus.Progress = 80
	m.mu.Unlock()

	// Parse JSON output
	var iperfOut iperfJSON
	if err := json.Unmarshal(output, &iperfOut); err != nil {
		return nil, fmt.Errorf("failed to parse iperf3 output: %w", err)
	}

	if iperfOut.Error != "" {
		return nil, fmt.Errorf("iperf3 error: %s", iperfOut.Error)
	}

	// Build result
	result := &Result{
		Protocol:  config.Protocol,
		Server:    config.Server,
		Port:      config.Port,
		Timestamp: time.Now(),
		Direction: direction,
	}

	switch direction {
	case "download":
		// In reverse mode, we care about what we received
		result.BitsPerSecond = iperfOut.End.SumReceived.BitsPerSecond
		result.Bandwidth = iperfOut.End.SumReceived.BitsPerSecond / 1_000_000
		result.Transfer = iperfOut.End.SumReceived.Bytes / 1_000_000
		result.Duration = iperfOut.End.SumReceived.Seconds
	case "bidirectional":
		result.DownloadBitsPerSecond = iperfOut.End.SumReceived.BitsPerSecond
		result.DownloadBandwidth = iperfOut.End.SumReceived.BitsPerSecond / 1_000_000
		result.DownloadTransfer = iperfOut.End.SumReceived.Bytes / 1_000_000
		result.UploadBitsPerSecond = iperfOut.End.SumSent.BitsPerSecond
		result.UploadBandwidth = iperfOut.End.SumSent.BitsPerSecond / 1_000_000
		result.UploadTransfer = iperfOut.End.SumSent.Bytes / 1_000_000
		// Preserve legacy fields using download direction for compatibility
		result.BitsPerSecond = result.DownloadBitsPerSecond
		result.Bandwidth = result.DownloadBandwidth
		result.Transfer = result.DownloadTransfer
		result.Duration = iperfOut.End.Sum.Seconds
		result.Retransmits = iperfOut.End.SumSent.Retransmits
	default: // upload
		// In normal mode, we care about what we sent
		result.BitsPerSecond = iperfOut.End.SumSent.BitsPerSecond
		result.Bandwidth = iperfOut.End.SumSent.BitsPerSecond / 1_000_000
		result.Transfer = iperfOut.End.SumSent.Bytes / 1_000_000
		result.Duration = iperfOut.End.SumSent.Seconds
		result.Retransmits = iperfOut.End.SumSent.Retransmits
	}

	// UDP-specific fields
	if config.Protocol == "udp" {
		result.Jitter = iperfOut.End.Sum.JitterMs
		result.LostPackets = iperfOut.End.Sum.LostPackets
		result.LostPercent = iperfOut.End.Sum.LostPercent
	}

	m.mu.Lock()
	m.lastResult = result
	m.clientStatus.Phase = "complete"
	m.clientStatus.Progress = 100
	m.mu.Unlock()

	return result, nil
}
