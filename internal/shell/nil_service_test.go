// Package shell_test provides tests for nil/uninitialized service scenarios.
// These tests cover error handling when services are not properly initialized.
package shell_test

import (
	"context"
	"errors"
	"testing"

	"github.com/krisarmstrong/seed/internal/shell"
)

// ========== Nil Discovery Service Tests ==========

// TestDiscoveryServiceNilDeviceDiscovery tests behavior when deviceDiscovery is nil.
func TestDiscoveryServiceNilDeviceDiscovery(t *testing.T) {
	t.Parallel()

	// Create service with nil deviceDiscovery
	service := shell.DiscoveryServiceWithNilDeviceDiscovery()

	ctx := context.Background()

	// Discover should return ErrNotInitialized
	result, err := service.Discover(ctx, nil)
	if !errors.Is(err, shell.ErrNotInitialized) {
		t.Errorf("Discover() error = %v, want ErrNotInitialized", err)
	}
	if result != nil {
		t.Error("Discover() result should be nil when error is returned")
	}

	// GetDevices should return ErrNotInitialized
	devices, err := service.GetDevices(ctx)
	if !errors.Is(err, shell.ErrNotInitialized) {
		t.Errorf("GetDevices() error = %v, want ErrNotInitialized", err)
	}
	if devices != nil {
		t.Error("GetDevices() result should be nil when error is returned")
	}

	// GetDevice should return ErrNotInitialized
	device, err := service.GetDevice(ctx, "test-id")
	if !errors.Is(err, shell.ErrNotInitialized) {
		t.Errorf("GetDevice() error = %v, want ErrNotInitialized", err)
	}
	if device != nil {
		t.Error("GetDevice() result should be nil when error is returned")
	}
}

// TestDiscoveryServiceNilDeviceDiscoveryStop tests Stop with nil internal services.
func TestDiscoveryServiceNilDeviceDiscoveryStop(t *testing.T) {
	t.Parallel()

	// Create service with nil everything
	service := shell.DiscoveryServiceWithNilDeviceDiscovery()

	// Stop should not panic even with nil internal services
	service.Stop()
}

// TestDiscoveryServiceNilWithCancel tests Stop with cancel function set.
func TestDiscoveryServiceNilWithCancel(t *testing.T) {
	t.Parallel()

	// Create service with nil deviceDiscovery
	service := shell.DiscoveryServiceWithNilDeviceDiscovery()
	accessor := shell.DiscoveryServiceTestAccessor{Service: service}

	// Set a cancel function
	cancelled := false
	accessor.SetCancel(func() { cancelled = true })

	// Stop should call the cancel function
	service.Stop()

	if !cancelled {
		t.Error("Stop should call the cancel function")
	}
}

// ========== Nil Vulnerability Service Tests ==========

// TestVulnerabilityServiceNilScanner tests behavior when scanner is nil.
func TestVulnerabilityServiceNilScanner(t *testing.T) {
	t.Parallel()

	// Create service with nil scanner
	service := shell.VulnerabilityServiceWithNilScanner()

	ctx := context.Background()

	// Scan should return ErrNotInitialized
	result, err := service.Scan(ctx, []string{"127.0.0.1"})
	if !errors.Is(err, shell.ErrNotInitialized) {
		t.Errorf("Scan() error = %v, want ErrNotInitialized", err)
	}
	if result != nil {
		t.Error("Scan() result should be nil when error is returned")
	}

	// GetVulnerabilities should return ErrNotInitialized
	vulns, err := service.GetVulnerabilities(ctx)
	if !errors.Is(err, shell.ErrNotInitialized) {
		t.Errorf("GetVulnerabilities() error = %v, want ErrNotInitialized", err)
	}
	if vulns != nil {
		t.Error("GetVulnerabilities() result should be nil when error is returned")
	}

	// GetDeviceVulnerabilities should return ErrNotInitialized
	vulns, err = service.GetDeviceVulnerabilities(ctx, "127.0.0.1")
	if !errors.Is(err, shell.ErrNotInitialized) {
		t.Errorf("GetDeviceVulnerabilities() error = %v, want ErrNotInitialized", err)
	}
	if vulns != nil {
		t.Error("GetDeviceVulnerabilities() result should be nil when error is returned")
	}
}

// TestVulnerabilityServiceNilScannerStop tests Stop with nil scanner.
func TestVulnerabilityServiceNilScannerStop(t *testing.T) {
	t.Parallel()

	// Create service with nil scanner
	service := shell.VulnerabilityServiceWithNilScanner()

	// Stop should not panic even with nil scanner
	service.Stop()
}

// TestVulnerabilityServiceNilWithCancel tests Stop with cancel function set.
func TestVulnerabilityServiceNilWithCancel(t *testing.T) {
	t.Parallel()

	// Create service with nil scanner
	service := shell.VulnerabilityServiceWithNilScanner()
	accessor := shell.VulnerabilityServiceTestAccessor{Service: service}

	// Set a cancel function
	cancelled := false
	accessor.SetCancel(func() { cancelled = true })

	// Stop should call the cancel function
	service.Stop()

	if !cancelled {
		t.Error("Stop should call the cancel function")
	}
}

// ========== Nil Rogue Service Tests ==========

// TestRogueServiceNilDetector tests behavior when detector is nil.
func TestRogueServiceNilDetector(t *testing.T) {
	t.Parallel()

	// Create service with nil detector
	service := shell.RogueServiceWithNilDetector()

	ctx := context.Background()

	// GetRogueDevices should return ErrNotInitialized
	devices, err := service.GetRogueDevices(ctx)
	if !errors.Is(err, shell.ErrNotInitialized) {
		t.Errorf("GetRogueDevices() error = %v, want ErrNotInitialized", err)
	}
	if devices != nil {
		t.Error("GetRogueDevices() result should be nil when error is returned")
	}

	// GetAlerts should return ErrNotInitialized
	alerts, err := service.GetAlerts(ctx)
	if !errors.Is(err, shell.ErrNotInitialized) {
		t.Errorf("GetAlerts() error = %v, want ErrNotInitialized", err)
	}
	if alerts != nil {
		t.Error("GetAlerts() result should be nil when error is returned")
	}
}

// TestRogueServiceNilDetectorStop tests Stop with nil detector.
func TestRogueServiceNilDetectorStop(t *testing.T) {
	t.Parallel()

	// Create service with nil detector
	service := shell.RogueServiceWithNilDetector()

	// Stop should not panic even with nil detector
	service.Stop()
}

// TestRogueServiceNilWithCancel tests Stop with cancel function set.
func TestRogueServiceNilWithCancel(t *testing.T) {
	t.Parallel()

	// Create service with nil detector
	service := shell.RogueServiceWithNilDetector()
	accessor := shell.RogueServiceTestAccessor{Service: service}

	// Set a cancel function
	cancelled := false
	accessor.SetCancel(func() { cancelled = true })

	// Stop should call the cancel function
	service.Stop()

	if !cancelled {
		t.Error("Stop should call the cancel function")
	}
}

// ========== Accessor Method Tests for Nil Services ==========

// TestNilServiceAccessorMethods tests accessor methods on nil services.
func TestNilServiceAccessorMethods(t *testing.T) {
	t.Parallel()

	// Discovery service accessors
	discService := shell.DiscoveryServiceWithNilDeviceDiscovery()
	if discService.Service() != nil {
		t.Error("Service() should return nil for uninitialized service")
	}
	if discService.DeviceDiscovery() != nil {
		t.Error("DeviceDiscovery() should return nil for uninitialized service")
	}

	// Vulnerability service accessors
	vulnService := shell.VulnerabilityServiceWithNilScanner()
	if vulnService.Scanner() != nil {
		t.Error("Scanner() should return nil for uninitialized service")
	}

	// Rogue service accessors
	rogueService := shell.RogueServiceWithNilDetector()
	if rogueService.Detector() != nil {
		t.Error("Detector() should return nil for uninitialized service")
	}
}
