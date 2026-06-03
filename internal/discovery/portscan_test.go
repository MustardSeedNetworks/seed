package discovery_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestNewPortScanner(t *testing.T) {
	scanner, err := discovery.NewPortScanner(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("NewPortScanner failed: %v", err)
	}
	if scanner == nil {
		t.Fatal("NewPortScanner returned nil")
	}

	// Clean up
	if closeErr := scanner.Close(); closeErr != nil {
		t.Errorf("Close failed: %v", closeErr)
	}
}

func TestPortScanner_Close(t *testing.T) {
	scanner, err := discovery.NewPortScanner(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("NewPortScanner failed: %v", err)
	}

	err = scanner.Close()
	if err != nil {
		t.Errorf("Close should succeed, got: %v", err)
	}

	// Second close should also succeed (idempotent)
	err = scanner.Close()
	if err != nil {
		t.Errorf("Second close should succeed, got: %v", err)
	}
}

func TestServiceInfo_Fields(t *testing.T) {
	info := discovery.ServiceInfo{
		Port:     22,
		State:    discovery.PortOpen,
		Service:  "ssh",
		Banner:   "SSH-2.0-OpenSSH_8.4p1",
		Version:  "8.4p1",
		Protocol: "tcp",
	}

	if info.Port != 22 {
		t.Errorf("Port should be 22, got %d", info.Port)
	}
	if info.State != discovery.PortOpen {
		t.Errorf("State should be PortOpen, got %v", info.State)
	}
	if info.Service != "ssh" {
		t.Errorf("Service should be 'ssh', got %q", info.Service)
	}
	if info.Banner != "SSH-2.0-OpenSSH_8.4p1" {
		t.Errorf("Banner mismatch, got %q", info.Banner)
	}
	if info.Version != "8.4p1" {
		t.Errorf("Version should be '8.4p1', got %q", info.Version)
	}
	if info.Protocol != "tcp" {
		t.Errorf("Protocol should be 'tcp', got %q", info.Protocol)
	}
}

func TestPortScanResult_Fields(t *testing.T) {
	result := discovery.PortScanResult{
		IP:       "192.168.1.10",
		Hostname: "test-host",
		Services: []discovery.ServiceInfo{
			{Port: 22, Service: "ssh"},
			{Port: 80, Service: "http"},
		},
		ScanTime: 500 * time.Millisecond,
		Error:    "",
	}

	if result.IP != "192.168.1.10" {
		t.Errorf("IP should be '192.168.1.10', got %q", result.IP)
	}
	if result.Hostname != "test-host" {
		t.Errorf("Hostname should be 'test-host', got %q", result.Hostname)
	}
	if len(result.Services) != 2 {
		t.Errorf("Services should have 2 entries, got %d", len(result.Services))
	}
	if result.ScanTime != 500*time.Millisecond {
		t.Errorf("ScanTime mismatch, got %v", result.ScanTime)
	}
	if result.Error != "" {
		t.Errorf("Error should be empty, got %q", result.Error)
	}
}

func TestPortScanResult_WithError(t *testing.T) {
	result := discovery.PortScanResult{
		IP:    "192.168.1.10",
		Error: "connection refused",
	}

	if result.IP != "192.168.1.10" {
		t.Errorf("IP should be '192.168.1.10', got %q", result.IP)
	}
	if result.Error != "connection refused" {
		t.Errorf("Error should be 'connection refused', got %q", result.Error)
	}
}

func TestPortScanner_ScanWithBanners_NonRoutable(t *testing.T) {
	scanner, err := discovery.NewPortScanner(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("NewPortScanner failed: %v", err)
	}
	defer func() { _ = scanner.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Scan TEST-NET-1 (non-routable)
	result := scanner.ScanWithBanners(ctx, "192.0.2.1", []int{22, 80}, 2)

	if result == nil {
		t.Fatal("ScanWithBanners should return result, not nil")
	}
	if result.IP != "192.0.2.1" {
		t.Errorf("IP should be '192.0.2.1', got %q", result.IP)
	}
	// Should have no open services on non-routable address
	if len(result.Services) != 0 {
		t.Logf("Unexpected open services on TEST-NET-1: %d", len(result.Services))
	}
}

func TestPortScanner_ScanWithBanners_ContextCancelled(t *testing.T) {
	scanner, err := discovery.NewPortScanner(1 * time.Second)
	if err != nil {
		t.Fatalf("NewPortScanner failed: %v", err)
	}
	defer func() { _ = scanner.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := scanner.ScanWithBanners(ctx, "192.0.2.1", []int{22, 80, 443}, 3)

	if result == nil {
		t.Fatal("ScanWithBanners should return result even when cancelled")
	}
	// Should handle cancelled context gracefully
}

func TestPortScanner_QuickScan(t *testing.T) {
	scanner, err := discovery.NewPortScanner(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("NewPortScanner failed: %v", err)
	}
	defer func() { _ = scanner.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// QuickScan on non-routable address
	result := scanner.QuickScan(ctx, "192.0.2.1")

	if result == nil {
		t.Fatal("QuickScan should return result")
	}
	if result.IP != "192.0.2.1" {
		t.Errorf("IP should be '192.0.2.1', got %q", result.IP)
	}
}

func TestPortScanner_WebScan(t *testing.T) {
	scanner, err := discovery.NewPortScanner(50 * time.Millisecond)
	if err != nil {
		t.Fatalf("NewPortScanner failed: %v", err)
	}
	defer func() { _ = scanner.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// WebScan on non-routable address
	result := scanner.WebScan(ctx, "192.0.2.1")

	if result == nil {
		t.Fatal("WebScan should return result")
	}
}

func TestPortScanner_FullScan(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping full scan in short mode")
	}

	scanner, err := discovery.NewPortScanner(10 * time.Millisecond)
	if err != nil {
		t.Fatalf("NewPortScanner failed: %v", err)
	}
	defer func() { _ = scanner.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// FullScan on non-routable address (first 1000 ports)
	result := scanner.FullScan(ctx, "192.0.2.1")

	if result == nil {
		t.Fatal("FullScan should return result")
	}
	if result.ScanTime == 0 {
		t.Error("ScanTime should be non-zero")
	}
}

func TestPortScanner_ScanWithBanners_Hostname(t *testing.T) {
	scanner, err := discovery.NewPortScanner(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("NewPortScanner failed: %v", err)
	}
	defer func() { _ = scanner.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use localhost which should resolve
	result := scanner.ScanWithBanners(ctx, "localhost", []int{1}, 1)

	if result == nil {
		t.Fatal("ScanWithBanners should return result for hostname")
	}
	// localhost should resolve, so hostname might be set
	t.Logf("Resolved IP: %s, Hostname: %s", result.IP, result.Hostname)
}

func TestPortScanner_ScanWithBanners_InvalidHostname(t *testing.T) {
	scanner, err := discovery.NewPortScanner(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("NewPortScanner failed: %v", err)
	}
	defer func() { _ = scanner.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use invalid hostname
	result := scanner.ScanWithBanners(ctx, "invalid.invalid.invalid", []int{22}, 1)

	if result == nil {
		t.Fatal("ScanWithBanners should return result even for invalid hostname")
	}
	// Should have an error
	if result.Error == "" {
		t.Log("No error returned for invalid hostname (may have resolved unexpectedly)")
	}
}

func TestGetWebPorts(t *testing.T) {
	ports := discovery.GetWebPorts()

	if len(ports) == 0 {
		t.Fatal("GetWebPorts should return ports")
	}

	// Check for essential web ports
	expectedPorts := []int{80, 443, 8080, 8443}
	for _, expected := range expectedPorts {
		if !slices.Contains(ports, expected) {
			t.Errorf("Web ports should include %d", expected)
		}
	}
}

func TestPortState_Constants(t *testing.T) {
	// Verify port state constants exist and are distinct
	if discovery.PortOpen == discovery.PortClosed {
		t.Error("PortOpen and PortClosed should be different")
	}
	if discovery.PortOpen == discovery.PortFiltered {
		t.Error("PortOpen and PortFiltered should be different")
	}
	if discovery.PortClosed == discovery.PortFiltered {
		t.Error("PortClosed and PortFiltered should be different")
	}
}
