package config

import (
	"net"
	"testing"
)

func TestHasIPv4Address(t *testing.T) {
	// Test with loopback (should exist on all systems)
	if !hasIPv4Address("lo") && !hasIPv4Address("lo0") {
		// Some systems might not have loopback configured the same way
		t.Log("Loopback interface test skipped - no lo or lo0 found")
	}

	// Test with non-existent interface
	if hasIPv4Address("nonexistent123456") {
		t.Error("hasIPv4Address should return false for non-existent interface")
	}

	// Test with empty name
	if hasIPv4Address("") {
		t.Error("hasIPv4Address should return false for empty interface name")
	}
}

func TestDetectActiveInterface(t *testing.T) {
	detected := detectActiveInterface()

	// Should detect at least one interface on any system with networking
	if detected == "" {
		t.Log("Warning: no active interface detected - this is expected in isolated test environments")
		return
	}

	// Verify the detected interface actually has an IPv4 address
	if !hasIPv4Address(detected) {
		t.Errorf("detectActiveInterface returned %q but it has no IPv4 address", detected)
	}

	// Verify the detected interface is not a virtual/bridge interface
	iface, err := net.InterfaceByName(detected)
	if err != nil {
		t.Errorf("detected interface %q doesn't exist: %v", detected, err)
		return
	}

	// Should not be loopback
	if iface.Flags&net.FlagLoopback != 0 {
		t.Errorf("detectActiveInterface should not return loopback interface, got %q", detected)
	}

	// Should be up
	if iface.Flags&net.FlagUp == 0 {
		t.Errorf("detectActiveInterface should only return UP interfaces, got %q", detected)
	}

	t.Logf("Detected interface: %s", detected)
}

func TestGetActiveInterface(t *testing.T) {
	tests := []struct {
		name           string
		config         InterfaceConfig
		expectFallback bool
	}{
		{
			name: "non-existent default interface",
			config: InterfaceConfig{
				Default:   "nonexistent123456",
				Fallbacks: []string{},
			},
			expectFallback: true,
		},
		{
			name: "empty config",
			config: InterfaceConfig{
				Default:   "",
				Fallbacks: []string{},
			},
			expectFallback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Interface: tt.config,
			}

			iface, usedFallback := cfg.GetActiveInterface()

			// Should return something
			if iface == "" {
				t.Error("GetActiveInterface should not return empty string")
			}

			// Check fallback flag
			if usedFallback != tt.expectFallback {
				t.Errorf("GetActiveInterface usedFallback = %v, want %v", usedFallback, tt.expectFallback)
			}
		})
	}
}

func TestGetActiveInterfaceWithValidDefault(t *testing.T) {
	// Find a valid interface on this system
	validIface := detectActiveInterface()
	if validIface == "" {
		t.Skip("No active interface available for testing")
	}

	cfg := &Config{
		Interface: InterfaceConfig{
			Default:   validIface,
			Fallbacks: []string{},
		},
	}

	iface, usedFallback := cfg.GetActiveInterface()

	if iface != validIface {
		t.Errorf("GetActiveInterface = %q, want %q", iface, validIface)
	}

	if usedFallback {
		t.Error("GetActiveInterface should not use fallback when default is valid")
	}
}

func TestGetActiveInterfaceWithFallback(t *testing.T) {
	// Find a valid interface on this system
	validIface := detectActiveInterface()
	if validIface == "" {
		t.Skip("No active interface available for testing")
	}

	cfg := &Config{
		Interface: InterfaceConfig{
			Default:   "nonexistent123456",
			Fallbacks: []string{"also_nonexistent", validIface},
		},
	}

	iface, usedFallback := cfg.GetActiveInterface()

	if iface != validIface {
		t.Errorf("GetActiveInterface = %q, want %q", iface, validIface)
	}

	if !usedFallback {
		t.Error("GetActiveInterface should use fallback when default is invalid")
	}
}
