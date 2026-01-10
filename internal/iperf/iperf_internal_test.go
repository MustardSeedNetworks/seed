package iperf_test

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/iperf"
)

// TestWaitForPortReady tests the port readiness check function.
func TestWaitForPortReady(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		port      int
		timeout   time.Duration
		setup     func() (cleanup func())
		wantError bool
		errorMsg  string
	}{
		{
			name:      "invalid port negative",
			port:      -1,
			timeout:   100 * time.Millisecond,
			wantError: true,
		},
		{
			name:      "invalid port zero",
			port:      0,
			timeout:   100 * time.Millisecond,
			wantError: true,
		},
		{
			name:      "invalid port too high",
			port:      65536,
			timeout:   100 * time.Millisecond,
			wantError: true,
		},
		{
			name:    "port not listening timeout",
			port:    59999, // unlikely to be in use
			timeout: 100 * time.Millisecond,
			setup: func() func() {
				return func() {}
			},
			wantError: true,
			errorMsg:  "not ready",
		},
		{
			name:    "port listening success",
			port:    0, // will be assigned dynamically
			timeout: 2 * time.Second,
			setup: func() func() {
				// Start a TCP listener on a random port
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					return func() {}
				}
				return func() {
					_ = listener.Close()
				}
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cleanup func()
			port := tt.port

			if tt.setup != nil {
				cleanup = tt.setup()
				defer cleanup()

				// For dynamic port test, we need to get the actual port
				if tt.name == "port listening success" {
					listener, err := net.Listen("tcp", "127.0.0.1:0")
					if err != nil {
						t.Skipf("Could not create listener: %v", err)
					}
					defer func() { _ = listener.Close() }()
					port = listener.Addr().(*net.TCPAddr).Port
				}
			}

			err := iperf.WaitForPortReady(port, tt.timeout)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error should contain %q, got: %v", tt.errorMsg, err)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestWaitForPortReadyWithListener tests port readiness with an actual listener.
func TestWaitForPortReadyWithListener(t *testing.T) {
	t.Parallel()

	// Start a TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer func() { _ = listener.Close() }()

	port := listener.Addr().(*net.TCPAddr).Port

	err = iperf.WaitForPortReady(port, 2*time.Second)
	if err != nil {
		t.Errorf("WaitForPortReady should succeed for listening port: %v", err)
	}
}

// TestCompareVersionsEdgeCases tests version comparison edge cases.
func TestCompareVersionsEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{"empty vs empty", "", "", 0},
		{"empty vs version", "", "1.0", -1},
		{"version vs empty", "1.0", "", 1},
		{"single digit", "3", "2", 1},
		{"single digit equal", "5", "5", 0},
		{"major only vs major.minor", "3", "3.0", 0},
		{"three parts equal", "1.2.3", "1.2.3", 0},
		{"three parts less", "1.2.3", "1.2.4", -1},
		{"three parts greater", "1.2.4", "1.2.3", 1},
		{"four parts", "1.2.3.4", "1.2.3.5", -1},
		{"different lengths", "1.2", "1.2.0.0", 0},
		{"non-numeric parts", "abc", "def", 0}, // both parse to 0
		{"mixed numeric", "1.abc", "1.def", 0},
		{"leading zeros", "01.02", "1.2", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := iperf.CompareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

// TestValidateServerIPv6 tests IPv6 address validation.
func TestValidateServerIPv6(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		server  string
		wantErr bool
	}{
		{"IPv6 loopback", "::1", false},
		{"IPv6 full address", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"IPv6 compressed", "2001:db8::1", false},
		{"IPv6 all zeros", "::", false},
		{"IPv6 v4 mapped", "::ffff:192.168.1.1", false},
		{"Invalid IPv6 with zone", "fe80::1%eth0", true}, // zones are not valid in iperf context
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := iperf.ValidateServer(tt.server)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateServer(%q) expected error, got nil", tt.server)
			} else if !tt.wantErr && err != nil {
				t.Errorf("ValidateServer(%q) unexpected error: %v", tt.server, err)
			}
		})
	}
}

// TestValidateServerHostnameBoundaries tests hostname boundary conditions.
func TestValidateServerHostnameBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		server  string
		wantErr bool
	}{
		{"single char hostname", "a", false},
		{"two char hostname", "ab", false},
		{"63 char label", strings.Repeat("a", 63), false},
		{"64 char label", strings.Repeat("a", 64), true}, // RFC 1035 limit
		{"max length hostname", strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." +
			strings.Repeat("c", 63) + "." + strings.Repeat("d", 60), false}, // 253 chars
		{"too long hostname", strings.Repeat("a", 254), true},
		{"numeric only", "12345", false},     // valid per RFC 1123 (fully numeric hostnames allowed)
		{"starts with digit", "1abc", false}, // valid per RFC 1123
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := iperf.ValidateServer(tt.server)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateServer(%q) expected error, got nil", tt.server)
			} else if !tt.wantErr && err != nil {
				t.Errorf("ValidateServer(%q) unexpected error: %v", tt.server, err)
			}
		})
	}
}

// TestBuildClientArgsAllCombinations tests all argument building combinations.
func TestBuildClientArgsAllCombinations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    iperf.ClientConfig
		direction string
		check     func(t *testing.T, args []string)
	}{
		{
			name: "UDP with download",
			config: iperf.ClientConfig{
				Server:   "test.example.com",
				Port:     5201,
				Duration: 10,
				Parallel: 1,
				Protocol: "udp",
			},
			direction: "download",
			check: func(t *testing.T, args []string) {
				t.Helper()
				hasU := contains(args, "-u")
				hasR := contains(args, "-R")
				if !hasU {
					t.Error("UDP test should have -u flag")
				}
				if !hasR {
					t.Error("Download direction should have -R flag")
				}
			},
		},
		{
			name: "UDP with bidirectional",
			config: iperf.ClientConfig{
				Server:   "test.example.com",
				Port:     5201,
				Duration: 10,
				Parallel: 1,
				Protocol: "udp",
			},
			direction: "bidirectional",
			check: func(t *testing.T, args []string) {
				t.Helper()
				hasU := contains(args, "-u")
				hasBidir := contains(args, "--bidir")
				if !hasU {
					t.Error("UDP test should have -u flag")
				}
				if !hasBidir {
					t.Error("Bidirectional direction should have --bidir flag")
				}
			},
		},
		{
			name: "high parallel streams",
			config: iperf.ClientConfig{
				Server:   "localhost",
				Port:     5201,
				Duration: 10,
				Parallel: 16,
				Protocol: "tcp",
			},
			direction: "upload",
			check: func(t *testing.T, args []string) {
				t.Helper()
				found := false
				for i, arg := range args {
					if arg == "-P" && i+1 < len(args) && args[i+1] == "16" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected -P 16 in args: %v", args)
				}
			},
		},
		{
			name: "long duration",
			config: iperf.ClientConfig{
				Server:   "localhost",
				Port:     5201,
				Duration: 3600, // 1 hour
				Parallel: 1,
				Protocol: "tcp",
			},
			direction: "upload",
			check: func(t *testing.T, args []string) {
				t.Helper()
				found := false
				for i, arg := range args {
					if arg == "-t" && i+1 < len(args) && args[i+1] == "3600" {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected -t 3600 in args: %v", args)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := iperf.BuildClientArgs(&tt.config, tt.direction)
			tt.check(t, args)
		})
	}
}

// TestParseClientResultZeroValues tests parsing with zero values.
func TestParseClientResultZeroValues(t *testing.T) {
	t.Parallel()

	iperfOut := &iperf.IperfJSON{}
	config := &iperf.ClientConfig{
		Server:   "localhost",
		Port:     5201,
		Protocol: "tcp",
	}

	result := iperf.ParseClientResult(iperfOut, config, "upload")

	if result == nil {
		t.Fatal("ParseClientResult should not return nil")
	}
	if result.Bandwidth != 0 {
		t.Errorf("Expected Bandwidth 0, got %v", result.Bandwidth)
	}
	if result.BitsPerSecond != 0 {
		t.Errorf("Expected BitsPerSecond 0, got %v", result.BitsPerSecond)
	}
	if result.Server != "localhost" {
		t.Errorf("Expected Server 'localhost', got %q", result.Server)
	}
}

// TestParseClientResultHighValues tests parsing with very high values.
func TestParseClientResultHighValues(t *testing.T) {
	t.Parallel()

	iperfOut := &iperf.IperfJSON{}
	iperfOut.End.SumSent.BitsPerSecond = 100_000_000_000 // 100 Gbps
	iperfOut.End.SumSent.Bytes = 1_250_000_000_000       // 1.25 TB
	iperfOut.End.SumSent.Seconds = 100

	config := &iperf.ClientConfig{
		Server:   "10gig-server.example.com",
		Port:     5201,
		Protocol: "tcp",
	}

	result := iperf.ParseClientResult(iperfOut, config, "upload")

	if result.Bandwidth != 100000.0 {
		t.Errorf("Expected Bandwidth 100000.0 Mbps, got %v", result.Bandwidth)
	}
}

// TestManagerConcurrentOperations tests concurrent manager operations.
func TestManagerConcurrentOperations(t *testing.T) {
	t.Parallel()

	manager := iperf.NewManager()

	var wg sync.WaitGroup
	numGoroutines := 20
	iterations := 100

	// Concurrent reads
	for range numGoroutines {
		wg.Go(func() {
			for range iterations {
				_ = manager.GetServerStatus()
				_ = manager.GetClientStatus()
				_ = manager.GetLastResult()
			}
		})
	}

	// Concurrent writes using test helpers
	for range numGoroutines / 2 {
		wg.Go(func() {
			for range iterations {
				manager.SetManagerClientStatusRunning(true)
				manager.SetManagerClientStatusRunning(false)
			}
		})
	}

	wg.Wait()
}

// TestSetClientDefaultsNoMutation tests that non-zero values are not mutated.
func TestSetClientDefaultsNoMutation(t *testing.T) {
	t.Parallel()

	config := &iperf.ClientConfig{
		Server:   "custom-server",
		Port:     9999,
		Duration: 60,
		Parallel: 8,
		Protocol: "udp",
	}

	// Store original values
	origPort := config.Port
	origDuration := config.Duration
	origParallel := config.Parallel
	origProtocol := config.Protocol

	iperf.SetClientDefaults(config)

	if config.Port != origPort {
		t.Errorf("Port was mutated: got %d, want %d", config.Port, origPort)
	}
	if config.Duration != origDuration {
		t.Errorf("Duration was mutated: got %d, want %d", config.Duration, origDuration)
	}
	if config.Parallel != origParallel {
		t.Errorf("Parallel was mutated: got %d, want %d", config.Parallel, origParallel)
	}
	if config.Protocol != origProtocol {
		t.Errorf("Protocol was mutated: got %q, want %q", config.Protocol, origProtocol)
	}
}

// TestNormalizeDirectionReverseFlag tests that direction normalization works.
func TestNormalizeDirectionReverseFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		direction         string
		expectedDirection string
	}{
		{"upload stays upload", "upload", "upload"},
		{"download stays download", "download", "download"},
		{"bidirectional stays bidirectional", "bidirectional", "bidirectional"},
		{"empty with reverse false becomes upload", "", "upload"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &iperf.ClientConfig{
				Server:    "localhost",
				Direction: tt.direction,
			}

			result := iperf.NormalizeDirection(config)

			if result != tt.expectedDirection {
				t.Errorf("NormalizeDirection() = %q, want %q", result, tt.expectedDirection)
			}
		})
	}
}

// TestConstantsValues tests that constants have expected values.
func TestConstantsValues(t *testing.T) {
	t.Parallel()

	if iperf.VersionCheckTimeout != 5*time.Second {
		t.Errorf("VersionCheckTimeout = %v, want 5s", iperf.VersionCheckTimeout)
	}

	if iperf.ServerStartTimeout != 10*time.Second {
		t.Errorf("ServerStartTimeout = %v, want 10s", iperf.ServerStartTimeout)
	}

	if iperf.PortCheckTimeout != 2*time.Second {
		t.Errorf("PortCheckTimeout = %v, want 2s", iperf.PortCheckTimeout)
	}

	if iperf.MaxHostnameLength != 253 {
		t.Errorf("MaxHostnameLength = %d, want 253", iperf.MaxHostnameLength)
	}

	if iperf.DirectionDownload != "download" {
		t.Errorf("DirectionDownload = %q, want 'download'", iperf.DirectionDownload)
	}

	if iperf.DirectionUpload != "upload" {
		t.Errorf("DirectionUpload = %q, want 'upload'", iperf.DirectionUpload)
	}

	if iperf.DirectionBidirectional != "bidirectional" {
		t.Errorf("DirectionBidirectional = %q, want 'bidirectional'", iperf.DirectionBidirectional)
	}

	if iperf.BytesToMegabits != 1_000_000 {
		t.Errorf("BytesToMegabits = %d, want 1000000", iperf.BytesToMegabits)
	}
}

// TestBinaryPathCaching tests the binary path caching mechanism.
func TestBinaryPathCaching(t *testing.T) {
	// Save and restore original path
	originalPath := iperf.IperfBinaryPath()
	defer iperf.SetIperfBinaryPath(originalPath)

	// Test set and get
	testPath := "/test/path/iperf3"
	iperf.SetIperfBinaryPath(testPath)

	if got := iperf.IperfBinaryPath(); got != testPath {
		t.Errorf("IperfBinaryPath() = %q, want %q", got, testPath)
	}

	// Test clear
	iperf.ClearIperfBinaryPath()
	if got := iperf.IperfBinaryPath(); got != "" {
		t.Errorf("After clear, IperfBinaryPath() = %q, want empty", got)
	}
}

// TestGetLegacyPathsContent tests that legacy paths contain expected elements.
func TestGetLegacyPathsContent(t *testing.T) {
	t.Parallel()

	paths := iperf.GetLegacyPaths()

	// All paths should end with iperf3
	for _, path := range paths {
		if filepath.Base(path) != "iperf3" {
			t.Errorf("Path %q does not end with 'iperf3'", path)
		}
	}

	// Should have at least one path containing "bin"
	hasBinPath := false
	for _, path := range paths {
		if strings.Contains(path, "bin") {
			hasBinPath = true
			break
		}
	}
	if !hasBinPath && len(paths) > 0 {
		t.Error("Expected at least one path containing 'bin'")
	}
}

// TestValidateBinaryWithRealBinary tests binary validation with real binaries if available.
func TestValidateBinaryWithRealBinary(t *testing.T) {
	t.Parallel()

	// Skip if iperf3 is not installed
	if os.Getenv("SKIP_IPERF_TEST") == "1" {
		t.Skip("Skipping real binary test (SKIP_IPERF_TEST=1)")
	}

	// Try to find a real iperf3 binary
	path, err := iperf.FindSystemIperf3()
	if err != nil {
		t.Skip("iperf3 not found in system PATH")
	}

	if !iperf.ValidateBinary(path) {
		t.Errorf("ValidateBinary(%q) should return true for real iperf3", path)
	}
}

// TestFindIperf3BinaryWithCacheClear tests finding binary after cache clear.
func TestFindIperf3BinaryWithCacheClear(t *testing.T) {
	if os.Getenv("SKIP_IPERF_TEST") == "1" {
		t.Skip("Skipping iperf test (SKIP_IPERF_TEST=1)")
	}

	// Save original path
	originalPath := iperf.IperfBinaryPath()
	defer iperf.SetIperfBinaryPath(originalPath)

	// Clear cache
	iperf.ClearIperfBinaryPath()

	// Try to find binary
	path, err := iperf.FindIperf3Binary()
	if err != nil {
		t.Skipf("iperf3 not found: %v", err)
	}

	if path == "" {
		t.Error("FindIperf3Binary returned empty path without error")
	}

	// Verify cache was updated
	if iperf.IperfBinaryPath() != path {
		t.Error("Cache should be updated after finding binary")
	}
}

// TestManagerRunClientCancelledContext tests client with cancelled context.
func TestManagerRunClientCancelledContext(t *testing.T) {
	if os.Getenv("SKIP_IPERF_TEST") == "1" {
		t.Skip("Skipping iperf test (SKIP_IPERF_TEST=1)")
	}

	manager := iperf.NewManager()

	// Create already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := manager.RunClient(ctx, &iperf.ClientConfig{
		Server: "localhost",
		Port:   5201,
	})

	// Should fail due to cancelled context or connection failure
	if err == nil {
		t.Error("Expected error with cancelled context")
	}
}

// contains checks if a string slice contains a specific item.
func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}
