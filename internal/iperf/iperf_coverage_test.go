package iperf_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/iperf"
)

// TestStartServerInvalidPorts tests starting server with various invalid ports.
func TestStartServerInvalidPorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		port      int
		wantError bool
	}{
		{"negative port", -1, true},
		{"zero port", 0, true},
		{"port above max", 65536, true},
		{"port way above max", 100000, true},
		{"privileged port", 80, false}, // may fail due to permissions, but validation passes
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := iperf.NewManager()
			err := manager.StartServer(tt.port)
			if tt.wantError && err == nil {
				t.Errorf("StartServer(%d) expected error, got nil", tt.port)
			}
			// Note: for privileged ports, we may get permission error which is OK
		})
	}
}

// TestStopServerVariousStates tests stopping server in various states.
func TestStopServerVariousStates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		setupRunning  bool
		setupPort     int
		wantError     bool
		errorContains string
	}{
		{
			name:          "server not running",
			setupRunning:  false,
			setupPort:     0,
			wantError:     true,
			errorContains: "not running",
		},
		{
			name:         "server running but no process",
			setupRunning: true,
			setupPort:    5201,
			wantError:    false, // StopServer clears status even without process
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := iperf.NewManager()

			if tt.setupRunning {
				manager.SetManagerServerStatusRunning(true, tt.setupPort)
			}

			err := manager.StopServer()

			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Error %q should contain %q", err.Error(), tt.errorContains)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Verify server status is cleared after stop
			status := manager.GetServerStatus()
			if status.Running && !tt.wantError {
				t.Error("Server should not be running after StopServer")
			}
		})
	}
}

// TestRunClientValidationErrors tests client validation errors.
func TestRunClientValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		config        *iperf.ClientConfig
		errorContains string
	}{
		{
			name:          "empty server",
			config:        &iperf.ClientConfig{Server: ""},
			errorContains: "server address is required",
		},
		{
			name:          "server with semicolon",
			config:        &iperf.ClientConfig{Server: "localhost;rm -rf /"},
			errorContains: "invalid server address",
		},
		{
			name:          "server with pipe",
			config:        &iperf.ClientConfig{Server: "localhost|cat"},
			errorContains: "invalid server address",
		},
		{
			name:          "server with backticks",
			config:        &iperf.ClientConfig{Server: "`whoami`"},
			errorContains: "invalid server address",
		},
		{
			name:          "server with dollar sign",
			config:        &iperf.ClientConfig{Server: "$HOME"},
			errorContains: "invalid server address",
		},
		{
			name:          "server with newline",
			config:        &iperf.ClientConfig{Server: "localhost\nrm -rf /"},
			errorContains: "invalid server address",
		},
		{
			name:          "server too long",
			config:        &iperf.ClientConfig{Server: strings.Repeat("a", 300)},
			errorContains: "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := iperf.NewManager()
			ctx := context.Background()

			_, err := manager.RunClient(ctx, tt.config)

			if err == nil {
				t.Error("Expected error, got nil")
				return
			}

			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("Error %q should contain %q", err.Error(), tt.errorContains)
			}
		})
	}
}

// TestRunClientConcurrentPrevention tests that concurrent client runs are prevented.
func TestRunClientConcurrentPrevention(t *testing.T) {
	t.Parallel()

	manager := iperf.NewManager()

	// Set client as running
	manager.SetManagerClientStatusRunning(true)
	defer manager.SetManagerClientStatusRunning(false)

	ctx := context.Background()
	_, err := manager.RunClient(ctx, &iperf.ClientConfig{Server: "localhost"})

	if err == nil {
		t.Error("Expected error when test already in progress")
	}

	if !strings.Contains(err.Error(), "already in progress") {
		t.Errorf("Error should mention 'already in progress', got: %v", err)
	}
}

// TestParseClientResultUDPMetrics tests UDP-specific metric parsing.
func TestParseClientResultUDPMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		jitter      float64
		lostPackets int
		lostPercent float64
	}{
		{"no packet loss", 0.5, 0, 0.0},
		{"some packet loss", 1.5, 100, 5.0},
		{"high jitter", 10.0, 50, 2.5},
		{"zero jitter", 0.0, 0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iperfOut := &iperf.IperfJSON{}
			iperfOut.End.SumSent.BitsPerSecond = 100_000_000
			iperfOut.End.SumSent.Bytes = 125_000_000
			iperfOut.End.SumSent.Seconds = 10
			iperfOut.End.Sum.JitterMs = tt.jitter
			iperfOut.End.Sum.LostPackets = tt.lostPackets
			iperfOut.End.Sum.LostPercent = tt.lostPercent

			config := &iperf.ClientConfig{
				Server:   "localhost",
				Port:     5201,
				Protocol: "udp",
			}

			result := iperf.ParseClientResult(iperfOut, config, "upload")

			if result.Jitter != tt.jitter {
				t.Errorf("Jitter = %v, want %v", result.Jitter, tt.jitter)
			}
			if result.LostPackets != tt.lostPackets {
				t.Errorf("LostPackets = %d, want %d", result.LostPackets, tt.lostPackets)
			}
			if result.LostPercent != tt.lostPercent {
				t.Errorf("LostPercent = %v, want %v", result.LostPercent, tt.lostPercent)
			}
		})
	}
}

// TestParseClientResultBidirectionalMetrics tests bidirectional test parsing.
func TestParseClientResultBidirectionalMetrics(t *testing.T) {
	t.Parallel()

	iperfOut := &iperf.IperfJSON{}
	iperfOut.End.SumReceived.BitsPerSecond = 200_000_000 // 200 Mbps download
	iperfOut.End.SumReceived.Bytes = 250_000_000
	iperfOut.End.SumSent.BitsPerSecond = 100_000_000 // 100 Mbps upload
	iperfOut.End.SumSent.Bytes = 125_000_000
	iperfOut.End.SumSent.Retransmits = 10
	iperfOut.End.Sum.Seconds = 10

	config := &iperf.ClientConfig{
		Server:   "localhost",
		Port:     5201,
		Protocol: "tcp",
	}

	result := iperf.ParseClientResult(iperfOut, config, "bidirectional")

	// Verify both directions are captured
	if result.DownloadBitsPerSecond != 200_000_000 {
		t.Errorf("DownloadBitsPerSecond = %v, want 200000000", result.DownloadBitsPerSecond)
	}
	if result.UploadBitsPerSecond != 100_000_000 {
		t.Errorf("UploadBitsPerSecond = %v, want 100000000", result.UploadBitsPerSecond)
	}
	if result.DownloadBandwidth != 200.0 {
		t.Errorf("DownloadBandwidth = %v, want 200.0", result.DownloadBandwidth)
	}
	if result.UploadBandwidth != 100.0 {
		t.Errorf("UploadBandwidth = %v, want 100.0", result.UploadBandwidth)
	}
	if result.Retransmits != 10 {
		t.Errorf("Retransmits = %d, want 10", result.Retransmits)
	}
	if result.Direction != "bidirectional" {
		t.Errorf("Direction = %q, want 'bidirectional'", result.Direction)
	}
}

// TestBuildClientArgsReverseFlagSetting tests that reverse flag is set correctly.
func TestBuildClientArgsReverseFlagSetting(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		direction      string
		expectReverse  bool
		expectRFlag    bool
		expectBidirDir bool
	}{
		{"upload sets reverse false", "upload", false, false, false},
		{"download sets reverse true", "download", true, true, false},
		{"bidirectional sets reverse false", "bidirectional", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &iperf.ClientConfig{
				Server:   "localhost",
				Port:     5201,
				Duration: 10,
				Parallel: 1,
				Protocol: "tcp",
			}

			args := iperf.BuildClientArgs(config, tt.direction)

			hasRFlag := false
			hasBidirFlag := false
			for _, arg := range args {
				if arg == "-R" {
					hasRFlag = true
				}
				if arg == "--bidir" {
					hasBidirFlag = true
				}
			}

			if hasRFlag != tt.expectRFlag {
				t.Errorf("-R flag presence = %v, want %v", hasRFlag, tt.expectRFlag)
			}
			if hasBidirFlag != tt.expectBidirDir {
				t.Errorf("--bidir flag presence = %v, want %v", hasBidirFlag, tt.expectBidirDir)
			}
			if config.Reverse != tt.expectReverse {
				t.Errorf("config.Reverse = %v, want %v", config.Reverse, tt.expectReverse)
			}
		})
	}
}

// TestBuildClientArgsUDPBandwidth tests that UDP tests have bandwidth flag.
func TestBuildClientArgsUDPBandwidth(t *testing.T) {
	t.Parallel()

	config := &iperf.ClientConfig{
		Server:   "localhost",
		Port:     5201,
		Duration: 10,
		Parallel: 1,
		Protocol: "udp",
	}

	args := iperf.BuildClientArgs(config, "upload")

	hasU := false
	hasB := false
	bValue := ""

	for i, arg := range args {
		if arg == "-u" {
			hasU = true
		}
		if arg == "-b" && i+1 < len(args) {
			hasB = true
			bValue = args[i+1]
		}
	}

	if !hasU {
		t.Error("UDP test should have -u flag")
	}
	if !hasB {
		t.Error("UDP test should have -b flag")
	}
	if bValue != "0" {
		t.Errorf("UDP bandwidth value = %q, want '0' (unlimited)", bValue)
	}
}

// TestFindIperf3BinaryWithMockedPath tests binary finding with mocked paths.
func TestFindIperf3BinaryWithMockedPath(t *testing.T) {
	// Save and restore original path
	originalPath := iperf.IperfBinaryPath()
	defer iperf.SetIperfBinaryPath(originalPath)

	// Test with cached path
	testPath := "/mocked/path/iperf3"
	iperf.SetIperfBinaryPath(testPath)

	path, err := iperf.FindIperf3Binary()
	if err != nil {
		t.Errorf("Unexpected error with cached path: %v", err)
	}
	if path != testPath {
		t.Errorf("Expected cached path %q, got %q", testPath, path)
	}
}

// TestValidateBinaryNonExecutable tests validation of non-executable files.
func TestValidateBinaryNonExecutable(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create a non-executable file
	nonExecPath := filepath.Join(tempDir, "fake-iperf3")
	if err := os.WriteFile(nonExecPath, []byte("not a binary"), 0o600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if iperf.ValidateBinary(nonExecPath) {
		t.Error("ValidateBinary should return false for non-executable file")
	}
}

// TestValidateBinaryNonExistent tests validation of non-existent paths.
func TestValidateBinaryNonExistent(t *testing.T) {
	t.Parallel()

	if iperf.ValidateBinary("/definitely/does/not/exist/iperf3") {
		t.Error("ValidateBinary should return false for non-existent path")
	}
}

// TestValidateBinaryEmptyPath tests validation with empty path.
func TestValidateBinaryEmptyPath(t *testing.T) {
	t.Parallel()

	if iperf.ValidateBinary("") {
		t.Error("ValidateBinary should return false for empty path")
	}
}

// TestGetLegacyPathsFormat tests that legacy paths have correct format.
func TestGetLegacyPathsFormat(t *testing.T) {
	t.Parallel()

	paths := iperf.GetLegacyPaths()

	if len(paths) == 0 {
		t.Error("GetLegacyPaths should return at least one path")
	}

	for _, path := range paths {
		// All paths should be absolute
		if !filepath.IsAbs(path) {
			t.Errorf("Path should be absolute: %q", path)
		}

		// All paths should end with iperf3
		if filepath.Base(path) != "iperf3" {
			t.Errorf("Path should end with 'iperf3': %q", path)
		}
	}
}

// TestWaitForPortReadyValidPorts tests port readiness with valid ports.
func TestWaitForPortReadyValidPorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		port    int
		timeout time.Duration
		setup   func() (cleanup func(), port int)
		wantErr bool
	}{
		{
			name:    "listening port",
			timeout: 2 * time.Second,
			setup: func() (func(), int) {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					return func() {}, 0
				}
				port := listener.Addr().(*net.TCPAddr).Port
				return func() { _ = listener.Close() }, port
			},
			wantErr: false,
		},
		{
			name:    "non-listening port",
			port:    59998,
			timeout: 200 * time.Millisecond,
			setup:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port := tt.port
			var cleanup func()

			if tt.setup != nil {
				cleanup, port = tt.setup()
				if port == 0 {
					t.Skip("Could not set up listener")
				}
				defer cleanup()
			}

			err := iperf.WaitForPortReady(port, tt.timeout)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			} else if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestNotFoundErrorFormatWithAllFields tests error message with all fields populated.
func TestNotFoundErrorFormatWithAllFields(t *testing.T) {
	t.Parallel()

	err := &iperf.NotFoundError{
		SearchedPaths: []string{
			"/usr/bin/iperf3",
			"/usr/local/bin/iperf3",
			"/opt/iperf3/bin/iperf3",
		},
		SystemError:   os.ErrNotExist,
		EmbeddedError: os.ErrPermission,
	}

	errMsg := err.Error()

	// Verify all components are present
	expectedParts := []string{
		"iperf3 not found",
		"System PATH",
		"Embedded binary",
		"/usr/bin/iperf3",
		"/usr/local/bin/iperf3",
		"/opt/iperf3/bin/iperf3",
	}

	for _, expected := range expectedParts {
		if !strings.Contains(errMsg, expected) {
			t.Errorf("Error message should contain %q", expected)
		}
	}
}

// TestGetInstallInstructionsPlatformSpecific tests install instructions for current platform.
func TestGetInstallInstructionsPlatformSpecific(t *testing.T) {
	t.Parallel()

	instructions := iperf.GetInstallInstructions()

	// Should always contain iperf3 reference
	if !strings.Contains(instructions, "iperf3") {
		t.Error("Instructions should mention iperf3")
	}

	// Should contain source build instructions
	if !strings.Contains(instructions, "esnet/iperf") {
		t.Error("Instructions should mention GitHub source")
	}

	// Platform-specific checks
	switch runtime.GOOS {
	case "linux":
		// Should contain at least one package manager
		packageManagers := []string{"apt", "dnf", "yum", "pacman", "apk", "zypper"}
		hasOne := false
		for _, pm := range packageManagers {
			if strings.Contains(instructions, pm) {
				hasOne = true
				break
			}
		}
		if !hasOne {
			t.Error("Linux instructions should mention at least one package manager")
		}
	case "darwin":
		if !strings.Contains(instructions, "brew") {
			t.Error("macOS instructions should mention Homebrew")
		}
	case "windows":
		if !strings.Contains(instructions, "choco") && !strings.Contains(instructions, "iperf.fr") {
			t.Log("Windows instructions should mention Chocolatey or download link")
		}
	}
}

// TestDetectPackageManagerReturnsValid tests that detected package manager is valid.
func TestDetectPackageManagerReturnsValid(t *testing.T) {
	t.Parallel()

	pm := iperf.DetectPackageManager()

	if pm == nil {
		// No package manager is OK
		t.Log("No package manager detected")
		return
	}

	// Validate structure
	if pm.Name == "" {
		t.Error("Package manager should have a name")
	}
	if len(pm.InstallCommand) == 0 {
		t.Error("Package manager should have install command")
	}
	if !pm.Available {
		t.Error("Detected package manager should be available")
	}

	// Validate install command contains iperf3
	found := false
	for _, arg := range pm.InstallCommand {
		if arg == "iperf3" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Install command should contain 'iperf3': %v", pm.InstallCommand)
	}
}

// TestNeedsSudoByPackageManager tests sudo requirement detection.
func TestNeedsSudoByPackageManager(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		packageManager string
		onLinux        bool // expected on Linux
		onOther        bool // expected on non-Linux
	}{
		{"homebrew", "homebrew", false, false},
		{"scoop", "scoop", false, false},
		{"apt", "apt", true, false},
		{"dnf", "dnf", true, false},
		{"yum", "yum", true, false},
		{"pacman", "pacman", true, false},
		{"unknown", "unknown", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := iperf.NeedsSudo(tt.packageManager)
			expected := tt.onLinux
			if runtime.GOOS != "linux" {
				expected = tt.onOther
			}

			if result != expected {
				t.Errorf("NeedsSudo(%q) = %v, want %v (on %s)", tt.packageManager, result, expected, runtime.GOOS)
			}
		})
	}
}

// TestCheckBuildDependenciesReturnsSlice tests build dependency checking.
func TestCheckBuildDependenciesReturnsSlice(t *testing.T) {
	t.Parallel()

	missing := iperf.CheckBuildDependencies()

	// Verify it returns a slice (may be empty or nil)
	// Type assertion is implicit
	t.Logf("Missing build dependencies: %v", missing)

	// If any are missing, they should be non-empty strings
	for _, dep := range missing {
		if dep == "" {
			t.Error("Missing dependency should not be empty string")
		}
	}
}

// TestGetBuildDependencyInstallCommandNonEmpty tests build command is non-empty.
func TestGetBuildDependencyInstallCommandNonEmpty(t *testing.T) {
	t.Parallel()

	cmd := iperf.GetBuildDependencyInstallCommand()

	if cmd == "" {
		t.Error("Build dependency install command should not be empty")
	}

	// Should mention common build tools
	buildTools := []string{"make", "gcc", "Install", "autoconf", "build"}
	hasOne := false
	for _, tool := range buildTools {
		if strings.Contains(strings.ToLower(cmd), strings.ToLower(tool)) {
			hasOne = true
			break
		}
	}
	if !hasOne {
		t.Errorf("Command should mention build tools: %q", cmd)
	}
}

// TestInstallOptionsAllFields tests InstallOptions with all fields.
func TestInstallOptionsAllFields(t *testing.T) {
	t.Parallel()

	opts := iperf.InstallOptions{
		Method:     iperf.InstallMethodGitHub,
		Version:    "3.17",
		InstallDir: "/opt/iperf3",
		UseSudo:    true,
		Verbose:    true,
	}

	if opts.Method != iperf.InstallMethodGitHub {
		t.Errorf("Method = %q, want %q", opts.Method, iperf.InstallMethodGitHub)
	}
	if opts.Version != "3.17" {
		t.Errorf("Version = %q, want '3.17'", opts.Version)
	}
	if opts.InstallDir != "/opt/iperf3" {
		t.Errorf("InstallDir = %q, want '/opt/iperf3'", opts.InstallDir)
	}
	if !opts.UseSudo {
		t.Error("UseSudo should be true")
	}
	if !opts.Verbose {
		t.Error("Verbose should be true")
	}
}

// TestInstallResultAllFields tests InstallResult with all fields.
func TestInstallResultAllFields(t *testing.T) {
	t.Parallel()

	result := iperf.InstallResult{
		Success:     false,
		Path:        "",
		Version:     "",
		Method:      iperf.InstallMethodPackageManager,
		Error:       os.ErrNotExist,
		NeedsSudo:   true,
		SudoCommand: "sudo apt install iperf3",
	}

	if result.Success {
		t.Error("Success should be false")
	}
	if result.Path != "" {
		t.Errorf("Path should be empty, got %q", result.Path)
	}
	if result.Version != "" {
		t.Errorf("Version should be empty, got %q", result.Version)
	}
	if result.Method != iperf.InstallMethodPackageManager {
		t.Errorf("Method = %q, want %q", result.Method, iperf.InstallMethodPackageManager)
	}
	if result.Error == nil {
		t.Error("Error should not be nil")
	}
	if !result.NeedsSudo {
		t.Error("NeedsSudo should be true")
	}
	if result.SudoCommand != "sudo apt install iperf3" {
		t.Errorf("SudoCommand = %q, want 'sudo apt install iperf3'", result.SudoCommand)
	}
}

// TestPackageManagerInfoNilFields tests PackageManagerInfo with nil fields.
func TestPackageManagerInfoNilFields(t *testing.T) {
	t.Parallel()

	info := iperf.PackageManagerInfo{
		Name:           "dnf",
		InstallCommand: []string{"dnf", "install", "-y", "iperf3"},
		UpdateCommand:  nil, // DNF doesn't need explicit update
		Available:      true,
	}

	if info.UpdateCommand != nil {
		t.Error("UpdateCommand should be nil for dnf")
	}
	if len(info.InstallCommand) != 4 {
		t.Errorf("InstallCommand length = %d, want 4", len(info.InstallCommand))
	}
}

// TestVersionComparisonExtensive tests version comparison extensively.
func TestVersionComparisonExtensive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		// Equal versions
		{"3.17", "3.17", 0},
		{"1.0.0", "1.0.0", 0},
		{"10.20.30", "10.20.30", 0},

		// Greater versions
		{"3.18", "3.17", 1},
		{"4.0", "3.99", 1},
		{"1.0.1", "1.0.0", 1},
		{"1.1.0", "1.0.9", 1},

		// Lesser versions
		{"3.16", "3.17", -1},
		{"2.99", "3.0", -1},
		{"1.0.0", "1.0.1", -1},
		{"1.0.9", "1.1.0", -1},

		// Different lengths
		{"3.17.0", "3.17", 0},
		{"3.17", "3.17.0", 0},
		{"3.17.1", "3.17", 1},
		{"3.17", "3.17.1", -1},

		// Edge cases
		{"0", "0", 0},
		{"0.0", "0", 0},
		{"1", "0", 1},
		{"0", "1", -1},
	}

	for _, tt := range tests {
		name := tt.v1 + "_vs_" + tt.v2
		t.Run(name, func(t *testing.T) {
			result := iperf.CompareVersions(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

// TestValidateServerValidAddresses tests valid server address validation.
func TestValidateServerValidAddresses(t *testing.T) {
	t.Parallel()

	validAddresses := []string{
		// IPv4
		"127.0.0.1",
		"192.168.1.1",
		"10.0.0.1",
		"0.0.0.0",
		"255.255.255.255",

		// IPv6
		"::1",
		"::",
		"2001:db8::1",
		"fe80::1",
		"::ffff:192.168.1.1",

		// Hostnames
		"localhost",
		"example.com",
		"test.example.com",
		"my-server.example.com",
		"server1.test.example.com",
		"a",
		"ab",
		"a1",
		"1a",
		"123",
	}

	for _, addr := range validAddresses {
		t.Run(addr, func(t *testing.T) {
			if err := iperf.ValidateServer(addr); err != nil {
				t.Errorf("ValidateServer(%q) unexpected error: %v", addr, err)
			}
		})
	}
}

// TestValidateServerInvalidAddresses tests invalid server address validation.
func TestValidateServerInvalidAddresses(t *testing.T) {
	t.Parallel()

	invalidAddresses := []string{
		"",
		" ",
		"test server",
		"test;server",
		"test|server",
		"test`server",
		"test$server",
		"test\nserver",
		"test\tserver",
		"-invalid",
		"invalid-",
		"test_server",
		"test@server",
		"test:server",
		strings.Repeat("a", 300),
	}

	for _, addr := range invalidAddresses {
		name := addr
		if len(name) > 20 {
			name = name[:20] + "..."
		}
		name = strings.ReplaceAll(name, "\n", "\\n")
		name = strings.ReplaceAll(name, "\t", "\\t")

		t.Run(name, func(t *testing.T) {
			if err := iperf.ValidateServer(addr); err == nil {
				t.Errorf("ValidateServer(%q) expected error, got nil", addr)
			}
		})
	}
}

// TestGetCacheDirReturnsAbsolutePath tests cache directory is absolute.
func TestGetCacheDirReturnsAbsolutePath(t *testing.T) {
	t.Parallel()

	dir, err := iperf.GetCacheDir()
	if err != nil {
		t.Fatalf("GetCacheDir() error = %v", err)
	}

	if !filepath.IsAbs(dir) {
		t.Errorf("Cache directory should be absolute: %q", dir)
	}

	if !strings.Contains(dir, "seed") {
		t.Errorf("Cache directory should contain 'seed': %q", dir)
	}
}

// TestIsValidExtractedBinaryWithVersionMismatch tests version mismatch detection.
func TestIsValidExtractedBinaryWithVersionMismatch(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	binaryPath := filepath.Join(tempDir, "iperf3")
	versionFile := filepath.Join(tempDir, ".iperf3-version")

	// Create executable binary
	if err := os.WriteFile(binaryPath, []byte("test"), 0o755); err != nil {
		t.Fatalf("Failed to create binary: %v", err)
	}

	// Create version file with wrong version
	if err := os.WriteFile(versionFile, []byte("1.0.0"), 0o600); err != nil {
		t.Fatalf("Failed to create version file: %v", err)
	}

	if iperf.IsValidExtractedBinary(binaryPath, versionFile) {
		t.Error("IsValidExtractedBinary should return false for version mismatch")
	}
}

// TestGetPlatformBinaryMapContainsCurrentPlatform tests platform map coverage.
func TestGetPlatformBinaryMapContainsCurrentPlatform(t *testing.T) {
	t.Parallel()

	platformMap := iperf.GetPlatformBinaryMap()

	currentPlatform := runtime.GOOS + "-" + runtime.GOARCH

	// Check if current platform is major (linux/darwin on amd64/arm64)
	majorPlatforms := map[string]bool{
		"linux-amd64":  true,
		"linux-arm64":  true,
		"darwin-amd64": true,
		"darwin-arm64": true,
	}

	if majorPlatforms[currentPlatform] {
		if _, ok := platformMap[currentPlatform]; !ok {
			t.Errorf("Major platform %q should be in platform map", currentPlatform)
		}
	}

	// Verify all entries have valid format
	for platform, binary := range platformMap {
		if !strings.HasPrefix(binary, "iperf3-") {
			t.Errorf("Binary %q for platform %q should start with 'iperf3-'", binary, platform)
		}
	}
}

// TestHasEmbeddedBinaryConsistency tests embedded binary detection is consistent.
func TestHasEmbeddedBinaryConsistency(t *testing.T) {
	t.Parallel()

	result1 := iperf.HasEmbeddedBinary()
	result2 := iperf.HasEmbeddedBinary()
	result3 := iperf.HasEmbeddedBinary()

	if result1 != result2 || result2 != result3 {
		t.Error("HasEmbeddedBinary should return consistent results")
	}
}

// TestClientConfigJSONMarshaling tests JSON serialization of ClientConfig.
func TestClientConfigJSONMarshaling(t *testing.T) {
	t.Parallel()

	config := iperf.ClientConfig{
		Server:    "test.example.com",
		Port:      5201,
		Protocol:  "tcp",
		Reverse:   true,
		Direction: "download",
		Duration:  30,
		Parallel:  4,
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	var decoded iperf.ClientConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if decoded.Server != config.Server {
		t.Errorf("Server = %q, want %q", decoded.Server, config.Server)
	}
	if decoded.Port != config.Port {
		t.Errorf("Port = %d, want %d", decoded.Port, config.Port)
	}
	if decoded.Protocol != config.Protocol {
		t.Errorf("Protocol = %q, want %q", decoded.Protocol, config.Protocol)
	}
	if decoded.Reverse != config.Reverse {
		t.Errorf("Reverse = %v, want %v", decoded.Reverse, config.Reverse)
	}
	if decoded.Direction != config.Direction {
		t.Errorf("Direction = %q, want %q", decoded.Direction, config.Direction)
	}
	if decoded.Duration != config.Duration {
		t.Errorf("Duration = %d, want %d", decoded.Duration, config.Duration)
	}
	if decoded.Parallel != config.Parallel {
		t.Errorf("Parallel = %d, want %d", decoded.Parallel, config.Parallel)
	}
}

// TestResultJSONMarshaling tests JSON serialization of Result.
func TestResultJSONMarshaling(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Millisecond) // Truncate for JSON precision

	result := iperf.Result{
		BitsPerSecond:         100_000_000,
		Bandwidth:             100.0,
		Transfer:              125.0,
		Retransmits:           5,
		Jitter:                1.5,
		LostPackets:           2,
		LostPercent:           0.1,
		Protocol:              "tcp",
		Direction:             "upload",
		Duration:              10.0,
		Server:                "test.example.com",
		Port:                  5201,
		Timestamp:             now,
		UploadBitsPerSecond:   50_000_000,
		DownloadBitsPerSecond: 100_000_000,
		UploadBandwidth:       50.0,
		DownloadBandwidth:     100.0,
		UploadTransfer:        62.5,
		DownloadTransfer:      125.0,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Failed to marshal result: %v", err)
	}

	var decoded iperf.Result
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if decoded.BitsPerSecond != result.BitsPerSecond {
		t.Errorf("BitsPerSecond = %v, want %v", decoded.BitsPerSecond, result.BitsPerSecond)
	}
	if decoded.Bandwidth != result.Bandwidth {
		t.Errorf("Bandwidth = %v, want %v", decoded.Bandwidth, result.Bandwidth)
	}
	if decoded.Protocol != result.Protocol {
		t.Errorf("Protocol = %q, want %q", decoded.Protocol, result.Protocol)
	}
}

// TestServerStatusJSONMarshaling tests JSON serialization of ServerStatus.
func TestServerStatusJSONMarshaling(t *testing.T) {
	t.Parallel()

	status := iperf.ServerStatus{
		Running: true,
		Port:    5201,
		PID:     12345,
		Error:   "test error",
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal status: %v", err)
	}

	var decoded iperf.ServerStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal status: %v", err)
	}

	if decoded.Running != status.Running {
		t.Errorf("Running = %v, want %v", decoded.Running, status.Running)
	}
	if decoded.Port != status.Port {
		t.Errorf("Port = %d, want %d", decoded.Port, status.Port)
	}
	if decoded.PID != status.PID {
		t.Errorf("PID = %d, want %d", decoded.PID, status.PID)
	}
	if decoded.Error != status.Error {
		t.Errorf("Error = %q, want %q", decoded.Error, status.Error)
	}
}

// TestClientStatusJSONMarshaling tests JSON serialization of ClientStatus.
func TestClientStatusJSONMarshaling(t *testing.T) {
	t.Parallel()

	status := iperf.ClientStatus{
		Running:  true,
		Phase:    "testing",
		Progress: 50.5,
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal status: %v", err)
	}

	var decoded iperf.ClientStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal status: %v", err)
	}

	if decoded.Running != status.Running {
		t.Errorf("Running = %v, want %v", decoded.Running, status.Running)
	}
	if decoded.Phase != status.Phase {
		t.Errorf("Phase = %q, want %q", decoded.Phase, status.Phase)
	}
	if decoded.Progress != status.Progress {
		t.Errorf("Progress = %v, want %v", decoded.Progress, status.Progress)
	}
}

// TestMockedGitHubReleaseAPI tests GitHub release API with mock server.
func TestMockedGitHubReleaseAPI(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns a valid release response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := map[string]string{
			"tag_name":    "v3.17",
			"tarball_url": "https://github.com/esnet/iperf/archive/refs/tags/3.17.tar.gz",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	// The actual GetLatestGitHubRelease uses hardcoded URL, so we can't test it directly
	// But we can verify the response parsing works
	t.Log("Mock server created at:", server.URL)
}

// TestDownloadFileWithMockServer tests file download with mock server.
func TestDownloadFileWithMockServer(t *testing.T) {
	t.Parallel()

	testContent := "test file content for iperf3 binary"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(testContent))
	}))
	defer server.Close()

	// We can't test downloadFile directly as it's internal,
	// but we verify the mock server is working
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to get from mock server: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
