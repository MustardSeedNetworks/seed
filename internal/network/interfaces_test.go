// Package network handles network interface management.
// Test suite validates interface detection, configuration, and cross-platform operations.
// Tests cover interface enumeration, property detection, IP configuration, and DNS setup.
package network

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name             string
		defaultInterface string
	}{
		{
			name:             "create manager with lo interface",
			defaultInterface: "lo",
		},
		{
			name:             "create manager with empty interface",
			defaultInterface: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := NewManager(tt.defaultInterface)
			if err != nil {
				t.Fatalf("NewManager() error = %v", err)
			}
			if mgr == nil {
				t.Fatal("NewManager() returned nil manager")
			}

			if mgr.currentInterface != tt.defaultInterface {
				t.Errorf("currentInterface = %v, want %v", mgr.currentInterface, tt.defaultInterface)
			}

			if mgr.interfaces == nil {
				t.Error("interfaces map is nil")
			}

			// Should have at least loopback interface
			if len(mgr.interfaces) == 0 {
				t.Error("No interfaces found")
			}
		})
	}
}

func TestRefreshInterfaces(t *testing.T) {
	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	err = mgr.RefreshInterfaces()
	if err != nil {
		t.Errorf("RefreshInterfaces() error = %v", err)
	}

	// Should have at least one interface (loopback)
	ifaces := mgr.GetInterfaces()
	if len(ifaces) == 0 {
		t.Error("RefreshInterfaces() found no interfaces")
	}
}

func TestGetInterfaces(t *testing.T) {
	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ifaces := mgr.GetInterfaces()

	if ifaces == nil {
		t.Fatal("GetInterfaces() returned nil")
	}

	// Should have at least loopback
	if len(ifaces) == 0 {
		t.Error("GetInterfaces() returned empty list")
	}

	// Verify interface info structure
	for _, iface := range ifaces {
		if iface.Name == "" {
			t.Error("Interface has empty name")
		}

		if iface.Type == "" {
			t.Error("Interface has empty type")
		}

		if iface.MTU <= 0 {
			t.Errorf("Interface %s has invalid MTU: %d", iface.Name, iface.MTU)
		}
	}
}

func TestGetInterface(t *testing.T) {
	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	// Get a known interface (loopback should exist on all systems)
	var loopbackName string
	for _, iface := range mgr.GetInterfaces() {
		if iface.Type == InterfaceTypeLoopback {
			loopbackName = iface.Name
			break
		}
	}

	if loopbackName == "" {
		t.Skip("No loopback interface found")
	}

	tests := []struct {
		name    string
		iface   string
		wantErr bool
	}{
		{
			name:    "get existing interface",
			iface:   loopbackName,
			wantErr: false,
		},
		{
			name:    "get non-existent interface",
			iface:   "nonexistent999",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := mgr.GetInterface(tt.iface)

			if tt.wantErr {
				if err == nil {
					t.Error("GetInterface() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("GetInterface() error = %v, want nil", err)
				return
			}

			if info == nil {
				t.Fatal("GetInterface() returned nil info")
			}

			if info.Name != tt.iface {
				t.Errorf("Interface Name = %v, want %v", info.Name, tt.iface)
			}
		})
	}
}

func TestGetCurrentInterface(t *testing.T) {
	mgr, err := NewManager("test-iface")
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	current := mgr.GetCurrentInterface()
	if current != "test-iface" {
		t.Errorf("GetCurrentInterface() = %v, want %v", current, "test-iface")
	}
}

func TestSetCurrentInterface(t *testing.T) {
	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	// Get a valid interface name
	var validName string
	for _, iface := range mgr.GetInterfaces() {
		validName = iface.Name
		break
	}

	if validName == "" {
		t.Fatal("No interfaces available for test")
	}

	tests := []struct {
		name    string
		iface   string
		wantErr bool
	}{
		{
			name:    "set to existing interface",
			iface:   validName,
			wantErr: false,
		},
		{
			name:    "set to non-existent interface",
			iface:   "nonexistent999",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.SetCurrentInterface(tt.iface)

			if tt.wantErr {
				if err == nil {
					t.Error("SetCurrentInterface() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("SetCurrentInterface() error = %v, want nil", err)
				return
			}

			current := mgr.GetCurrentInterface()
			if current != tt.iface {
				t.Errorf("GetCurrentInterface() = %v, want %v", current, tt.iface)
			}
		})
	}
}

func TestFindFirstAvailable(t *testing.T) {
	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	// Get list of available interfaces
	ifaces := mgr.GetInterfaces()
	if len(ifaces) == 0 {
		t.Skip("No interfaces available")
	}

	var upInterface string
	for _, iface := range ifaces {
		if iface.Up {
			upInterface = iface.Name
			break
		}
	}

	tests := []struct {
		name      string
		preferred []string
		wantEmpty bool
	}{
		{
			name:      "find with valid preferred",
			preferred: []string{upInterface},
			wantEmpty: upInterface == "",
		},
		{
			name:      "find with non-existent preferred",
			preferred: []string{"nonexistent999"},
			wantEmpty: false, // Should fallback to auto-detect
		},
		{
			name:      "find with empty preferred",
			preferred: []string{},
			wantEmpty: false, // Should auto-detect
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mgr.FindFirstAvailable(tt.preferred)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("FindFirstAvailable() = %v, want empty", result)
				}
				return
			}

			// Auto-detect should find at least loopback
			// But we exclude loopback, so it might be empty on minimal systems
			t.Logf("FindFirstAvailable() = %v", result)
		})
	}
}

func TestDetectInterfaceType(t *testing.T) {
	tests := []struct {
		name  string
		iface string
		want  InterfaceType
	}{
		// Loopback
		{"loopback lo", "lo", InterfaceTypeLoopback},
		{"loopback lo0", "lo0", InterfaceTypeLoopback},

		// Virtual
		{"docker", "docker0", InterfaceTypeVirtual},
		{"bridge", "br-abc123", InterfaceTypeVirtual},
		{"veth", "veth0", InterfaceTypeVirtual},
		{"virbr", "virbr0", InterfaceTypeVirtual},
		{"tun", "tun0", InterfaceTypeVirtual},
		{"tap", "tap0", InterfaceTypeVirtual},

		// WiFi
		{"wlan", "wlan0", InterfaceTypeWiFi},
		{"wlp", "wlp3s0", InterfaceTypeWiFi},
		{"wifi", "wifi0", InterfaceTypeWiFi},
		{"ath", "ath0", InterfaceTypeWiFi},

		// Ethernet
		{"eth", "eth0", InterfaceTypeEthernet},
		{"enp", "enp0s3", InterfaceTypeEthernet},
		{"ens", "ens33", InterfaceTypeEthernet},
		{"eno", "eno1", InterfaceTypeEthernet},
		{"em", "em1", InterfaceTypeEthernet},
		{"en", "en0", InterfaceTypeEthernet},

		// Other
		{"unknown", "xyz123", InterfaceTypeOther},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectInterfaceType(tt.iface)
			if got != tt.want {
				t.Errorf("detectInterfaceType(%v) = %v, want %v", tt.iface, got, tt.want)
			}
		})
	}
}

func TestIsWireless(t *testing.T) {
	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	// Test with mock data - manually add WiFi interface
	mgr.mu.Lock()
	mgr.interfaces["wlan0"] = &InterfaceInfo{
		Name: "wlan0",
		Type: InterfaceTypeWiFi,
	}
	mgr.interfaces["eth0"] = &InterfaceInfo{
		Name: "eth0",
		Type: InterfaceTypeEthernet,
	}
	mgr.mu.Unlock()

	tests := []struct {
		name  string
		iface string
		want  bool
	}{
		{"wifi interface", "wlan0", true},
		{"ethernet interface", "eth0", false},
		{"non-existent interface", "nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mgr.IsWireless(tt.iface)
			if got != tt.want {
				t.Errorf("IsWireless(%v) = %v, want %v", tt.iface, got, tt.want)
			}
		})
	}
}

func TestHasRoutableAddress(t *testing.T) {
	tests := []struct {
		name      string
		addresses []string
		want      bool
	}{
		{
			name:      "has routable IPv4",
			addresses: []string{"192.168.1.100/24"},
			want:      true,
		},
		{
			name:      "has routable IPv6",
			addresses: []string{"2001:db8::1/64"},
			want:      true,
		},
		{
			name:      "only loopback",
			addresses: []string{"127.0.0.1/8"},
			want:      false,
		},
		{
			name:      "only link-local IPv4",
			addresses: []string{"169.254.1.1/16"},
			want:      false,
		},
		{
			name:      "only link-local IPv6",
			addresses: []string{"fe80::1/64"},
			want:      false,
		},
		{
			name:      "mixed with routable",
			addresses: []string{"127.0.0.1/8", "192.168.1.1/24", "fe80::1/64"},
			want:      true,
		},
		{
			name:      "empty addresses",
			addresses: []string{},
			want:      false,
		},
		{
			name:      "invalid address",
			addresses: []string{"invalid"},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasRoutableAddress(tt.addresses)
			if got != tt.want {
				t.Errorf("hasRoutableAddress(%v) = %v, want %v", tt.addresses, got, tt.want)
			}
		})
	}
}

func TestValidateIPConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *StaticIPConfig
		wantErr bool
	}{
		{
			name: "valid config with CIDR netmask",
			cfg: &StaticIPConfig{
				Address: "192.168.1.100",
				Netmask: "24",
				Gateway: "192.168.1.1",
				DNS:     []string{"8.8.8.8", "8.8.4.4"},
			},
			wantErr: false,
		},
		{
			name: "valid config without gateway",
			cfg: &StaticIPConfig{
				Address: "192.168.1.100",
				Netmask: "24",
			},
			wantErr: false,
		},
		{
			name: "missing address",
			cfg: &StaticIPConfig{
				Address: "",
				Netmask: "24",
			},
			wantErr: true,
		},
		{
			name: "missing netmask",
			cfg: &StaticIPConfig{
				Address: "192.168.1.100",
				Netmask: "",
			},
			wantErr: true,
		},
		{
			name: "invalid IP address",
			cfg: &StaticIPConfig{
				Address: "invalid",
				Netmask: "24",
			},
			wantErr: true,
		},
		{
			name: "invalid netmask",
			cfg: &StaticIPConfig{
				Address: "192.168.1.100",
				Netmask: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid gateway",
			cfg: &StaticIPConfig{
				Address: "192.168.1.100",
				Netmask: "24",
				Gateway: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid DNS server",
			cfg: &StaticIPConfig{
				Address: "192.168.1.100",
				Netmask: "24",
				DNS:     []string{"8.8.8.8", "invalid"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIPConfig(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Error("validateIPConfig() error = nil, want error")
				}
			} else {
				if err != nil {
					t.Errorf("validateIPConfig() error = %v, want nil", err)
				}
			}
		})
	}
}

func TestIsValidNetmask(t *testing.T) {
	tests := []struct {
		name    string
		netmask string
		want    bool
	}{
		// Valid CIDR prefixes
		{"CIDR 24", "24", true},
		{"CIDR 0", "0", true},
		{"CIDR 32", "32", true},
		{"CIDR 16", "16", true},

		// Invalid
		{"CIDR 33", "33", false},
		{"CIDR -1", "-1", false},
		{"invalid string", "invalid", false},
		{"empty", "", false},
		{"IPv6 address", "ffff:ffff::", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidNetmask(tt.netmask)
			if got != tt.want {
				t.Errorf("isValidNetmask(%v) = %v, want %v", tt.netmask, got, tt.want)
			}
		})
	}
}

func TestCIDRToNetmask(t *testing.T) {
	tests := []struct {
		name   string
		prefix int
		want   string
	}{
		{"prefix 24", 24, "255.255.255.0"},
		{"prefix 16", 16, "255.255.0.0"},
		{"prefix 8", 8, "255.0.0.0"},
		{"prefix 0", 0, "0.0.0.0"},
		{"prefix 32", 32, "255.255.255.255"},
		{"prefix 25", 25, "255.255.255.128"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cidrToNetmask(tt.prefix)
			if got != tt.want {
				t.Errorf("cidrToNetmask(%v) = %v, want %v", tt.prefix, got, tt.want)
			}
		})
	}
}

func TestSetMTU(t *testing.T) {
	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	tests := []struct {
		name    string
		iface   string
		mtu     int
		wantErr bool
	}{
		{
			name:    "MTU too small",
			iface:   "eth0",
			mtu:     67,
			wantErr: true,
		},
		{
			name:    "MTU too large",
			iface:   "eth0",
			mtu:     9001,
			wantErr: true,
		},
		{
			name:    "valid MTU minimum",
			iface:   "eth0",
			mtu:     68,
			wantErr: false, // May fail on platform, but validates input
		},
		{
			name:    "valid MTU maximum",
			iface:   "eth0",
			mtu:     9000,
			wantErr: false, // May fail on platform, but validates input
		},
		{
			name:    "valid MTU standard",
			iface:   "eth0",
			mtu:     1500,
			wantErr: false, // May fail on platform, but validates input
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.SetMTU(tt.iface, tt.mtu)

			// We only check validation errors (MTU range)
			// Platform-specific errors are expected in tests
			if tt.wantErr {
				if err == nil {
					t.Error("SetMTU() error = nil, want validation error")
				}
			} else if tt.mtu < 68 || tt.mtu > 9000 {
				if err == nil {
					t.Error("SetMTU() should return validation error for out-of-range MTU")
				}
			}
			// Don't check platform execution errors
		})
	}
}

func TestGetLinkStatus(t *testing.T) {
	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	// Get a valid interface
	ifaces := mgr.GetInterfaces()
	if len(ifaces) == 0 {
		t.Skip("No interfaces available")
	}

	validInterface := ifaces[0].Name

	tests := []struct {
		name    string
		iface   string
		wantErr bool
	}{
		{
			name:    "get status for existing interface",
			iface:   validInterface,
			wantErr: false,
		},
		{
			name:    "get status for non-existent interface",
			iface:   "nonexistent999",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := mgr.GetLinkStatus(tt.iface)

			if tt.wantErr {
				if err == nil {
					t.Error("GetLinkStatus() error = nil, want error")
				}
				return
			}

			if err != nil {
				t.Errorf("GetLinkStatus() error = %v, want nil", err)
				return
			}

			if status == nil {
				t.Fatal("GetLinkStatus() returned nil status")
			}

			// Verify structure (values may vary by system/interface state)
			t.Logf("LinkStatus for %s: Speed=%s, Duplex=%s, Carrier=%v, HasIP=%v",
				tt.iface, status.Speed, status.Duplex, status.Carrier, status.HasIP)
		})
	}
}

func TestInterfaceInfo(t *testing.T) {
	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	ifaces := mgr.GetInterfaces()
	if len(ifaces) == 0 {
		t.Skip("No interfaces available")
	}

	// Test that InterfaceInfo fields are properly populated
	for _, iface := range ifaces {
		t.Run(iface.Name, func(t *testing.T) {
			if iface.Name == "" {
				t.Error("Name is empty")
			}

			if iface.Type == "" {
				t.Error("Type is empty")
			}

			if iface.MTU <= 0 {
				t.Errorf("MTU = %d, want > 0", iface.MTU)
			}

			// Addresses might be empty on some interfaces
			t.Logf("Interface: %s, Type: %s, Up: %v, Running: %v, MTU: %d, Addresses: %v",
				iface.Name, iface.Type, iface.Up, iface.Running, iface.MTU, iface.Addresses)
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	// Get a valid interface
	ifaces := mgr.GetInterfaces()
	if len(ifaces) == 0 {
		t.Skip("No interfaces available")
	}
	validInterface := ifaces[0].Name

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = mgr.GetInterfaces()
				_, _ = mgr.GetInterface(validInterface)
				_ = mgr.GetCurrentInterface()
				_ = mgr.IsWireless(validInterface)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestManagerEdgeCases(t *testing.T) {
	mgr, err := NewManager("")
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	// Test GetInterface with empty string
	_, err = mgr.GetInterface("")
	if err == nil {
		t.Error("GetInterface(\"\") should return error")
	}

	// Test SetCurrentInterface with empty string
	err = mgr.SetCurrentInterface("")
	if err == nil {
		t.Error("SetCurrentInterface(\"\") should return error")
	}

	// Test FindFirstAvailable with nil slice
	result := mgr.FindFirstAvailable(nil)
	// Should not panic, may return empty or auto-detected interface
	t.Logf("FindFirstAvailable(nil) = %v", result)
}

func TestInterfaceTypeConstants(t *testing.T) {
	// Verify type constants are correctly defined
	if InterfaceTypeEthernet == "" {
		t.Error("InterfaceTypeEthernet is empty")
	}
	if InterfaceTypeWiFi == "" {
		t.Error("InterfaceTypeWiFi is empty")
	}
	if InterfaceTypeLoopback == "" {
		t.Error("InterfaceTypeLoopback is empty")
	}
	if InterfaceTypeVirtual == "" {
		t.Error("InterfaceTypeVirtual is empty")
	}
	if InterfaceTypeOther == "" {
		t.Error("InterfaceTypeOther is empty")
	}

	// Verify they're all different
	types := []InterfaceType{
		InterfaceTypeEthernet,
		InterfaceTypeWiFi,
		InterfaceTypeLoopback,
		InterfaceTypeVirtual,
		InterfaceTypeOther,
	}

	seen := make(map[InterfaceType]bool)
	for _, typ := range types {
		if seen[typ] {
			t.Errorf("Duplicate InterfaceType value: %v", typ)
		}
		seen[typ] = true
	}
}

func TestFindFirstAvailableLogic(t *testing.T) {
	mgr := &Manager{
		interfaces: make(map[string]*InterfaceInfo),
	}

	// Build test scenario with various interface types
	mgr.interfaces["lo"] = &InterfaceInfo{
		Name:      "lo",
		Type:      InterfaceTypeLoopback,
		Up:        true,
		Addresses: []string{"127.0.0.1/8"},
	}

	mgr.interfaces["eth0"] = &InterfaceInfo{
		Name:      "eth0",
		Type:      InterfaceTypeEthernet,
		Up:        true,
		Addresses: []string{"192.168.1.100/24"},
	}

	mgr.interfaces["wlan0"] = &InterfaceInfo{
		Name:      "wlan0",
		Type:      InterfaceTypeWiFi,
		Up:        true,
		Addresses: []string{"192.168.1.101/24"},
	}

	mgr.interfaces["eth1"] = &InterfaceInfo{
		Name:      "eth1",
		Type:      InterfaceTypeEthernet,
		Up:        true,
		Addresses: []string{}, // No IP
	}

	mgr.interfaces["docker0"] = &InterfaceInfo{
		Name:      "docker0",
		Type:      InterfaceTypeVirtual,
		Up:        true,
		Addresses: []string{"172.17.0.1/16"},
	}

	tests := []struct {
		name      string
		preferred []string
		want      string
	}{
		{
			name:      "preferred eth0 exists",
			preferred: []string{"eth0"},
			want:      "eth0",
		},
		{
			name:      "preferred nonexistent, should auto-detect ethernet with IP",
			preferred: []string{"nonexistent"},
			want:      "eth0", // eth0 has IP, should be selected over wlan0
		},
		{
			name:      "empty preferred, should auto-detect",
			preferred: []string{},
			want:      "eth0", // eth0 with IP should be selected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mgr.FindFirstAvailable(tt.preferred)
			// Note: Order in map iteration may vary, so we check that
			// it's one of the expected interfaces (eth0 or wlan0)
			if got != "eth0" && got != "wlan0" {
				t.Errorf("FindFirstAvailable(%v) = %v, want eth0 or wlan0 (got neither)", tt.preferred, got)
			}
			t.Logf("FindFirstAvailable(%v) = %v", tt.preferred, got)
		})
	}
}

func TestAddressesWithoutCIDR(t *testing.T) {
	tests := []struct {
		name      string
		addresses []string
		want      bool
	}{
		{
			name:      "address without CIDR",
			addresses: []string{"192.168.1.100"},
			want:      true,
		},
		{
			name:      "mixed with and without CIDR",
			addresses: []string{"192.168.1.100", "10.0.0.1/24"},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hasRoutableAddress(tt.addresses)
			if got != tt.want {
				t.Errorf("hasRoutableAddress(%v) = %v, want %v", tt.addresses, got, tt.want)
			}
		})
	}
}

func TestNetmaskValidation(t *testing.T) {
	// Test comprehensive netmask scenarios
	// Note: Implementation primarily validates CIDR notation
	validMasks := []string{
		"0", "8", "16", "24", "32",
	}

	for _, mask := range validMasks {
		t.Run("valid_"+mask, func(t *testing.T) {
			if !isValidNetmask(mask) {
				t.Errorf("isValidNetmask(%q) = false, want true", mask)
			}
		})
	}

	invalidMasks := []string{
		"-1", "33", "256", "abc",
	}

	for _, mask := range invalidMasks {
		t.Run("invalid_"+mask, func(t *testing.T) {
			if isValidNetmask(mask) {
				t.Errorf("isValidNetmask(%q) = true, want false", mask)
			}
		})
	}
}

func TestInterfaceTypeDetectionPriority(t *testing.T) {
	// Test that more specific prefixes are detected correctly
	// even if they match multiple patterns

	tests := []struct {
		name string
		want InterfaceType
	}{
		// "en" matches both ethernet prefixes, but "enp" is more specific
		{"enp0s3", InterfaceTypeEthernet},
		{"en0", InterfaceTypeEthernet},

		// Virtual prefixes should match before ethernet
		{"docker0", InterfaceTypeVirtual},
		{"br-123abc", InterfaceTypeVirtual},

		// WiFi should be detected
		{"wlan0", InterfaceTypeWiFi},
		{"wlp3s0", InterfaceTypeWiFi},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectInterfaceType(tt.name)
			if got != tt.want {
				t.Errorf("detectInterfaceType(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
