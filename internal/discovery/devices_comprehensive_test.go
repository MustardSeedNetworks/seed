package discovery_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestDiscoveredDevice_AllFields(t *testing.T) {
	now := time.Now()

	device := &discovery.DiscoveredDevice{
		IP:            "192.168.1.100",
		IPv6Address:   "fe80::1",
		IPv6Addresses: []string{"fe80::1", "2001:db8::1", "2001:db8::2"},
		MAC:           "00:11:22:33:44:55",
		Hostname:      "server.example.com",
		NetBIOSName:   "SERVER01",
		MDNSName:      "server.local",
		DisplayName:   "Server 01",
		Vendor:        "Dell Inc.",
		OSGuess:       "Linux",
		TTL:           64,
		DiscoveryMethod: []discovery.Method{
			discovery.MethodARP,
			discovery.MethodPING,
			discovery.MethodMDNS,
		},
		LastSeen:       now,
		IsLocal:        true,
		IsRouter:       false,
		HasDuplicateIP: false,
		DuplicateMACs:  nil,
		LLDPInfo: &discovery.LLDPDeviceInfo{
			ChassisID:  "00:11:22:33:44:55",
			SystemName: "server.example.com",
		},
		CDPInfo: nil,
		EDPInfo: nil,
		NDPInfo: &discovery.NDPDeviceInfo{
			LinkLayerAddress: "00:11:22:33:44:55",
			IsRouter:         false,
		},
		Profile: &discovery.DeviceProfile{
			ProfiledAt: now,
			OpenPorts: []discovery.OpenPort{
				{Port: 22, Service: "ssh"},
				{Port: 80, Service: "http"},
			},
		},
	}

	// Test basic fields
	if device.IP != "192.168.1.100" {
		t.Errorf("IP mismatch: got %q", device.IP)
	}
	if device.MAC != "00:11:22:33:44:55" {
		t.Errorf("MAC mismatch: got %q", device.MAC)
	}
	if len(device.IPv6Addresses) != 3 {
		t.Errorf("IPv6Addresses should have 3 entries, got %d", len(device.IPv6Addresses))
	}
	if len(device.DiscoveryMethod) != 3 {
		t.Errorf("DiscoveryMethod should have 3 entries, got %d", len(device.DiscoveryMethod))
	}

	// Test protocol info
	if device.LLDPInfo == nil {
		t.Error("LLDPInfo should not be nil")
	}
	if device.NDPInfo == nil {
		t.Error("NDPInfo should not be nil")
	}
	if device.CDPInfo != nil {
		t.Error("CDPInfo should be nil")
	}

	// Test profile
	if device.Profile == nil {
		t.Error("Profile should not be nil")
	}
	if len(device.Profile.OpenPorts) != 2 {
		t.Errorf("Profile.OpenPorts should have 2 entries, got %d", len(device.Profile.OpenPorts))
	}
}

func TestComputeDisplayName_AllPriorities(t *testing.T) {
	tests := []struct {
		name     string
		device   *discovery.DiscoveredDevice
		expected string
	}{
		{
			name: "lldp_highest_priority",
			device: &discovery.DiscoveredDevice{
				IP: "192.168.1.1",
				LLDPInfo: &discovery.LLDPDeviceInfo{
					SystemName: "switch-lldp",
				},
				CDPInfo:     &discovery.CDPDeviceInfo{DeviceID: "switch-cdp"},
				EDPInfo:     &discovery.EDPDeviceInfo{DisplayName: "switch-edp"},
				MDNSName:    "switch.local",
				NetBIOSName: "SWITCH",
				Hostname:    "switch.example.com",
			},
			expected: "switch-lldp",
		},
		{
			name: "cdp_when_no_lldp",
			device: &discovery.DiscoveredDevice{
				IP:          "192.168.1.1",
				CDPInfo:     &discovery.CDPDeviceInfo{DeviceID: "router-cdp"},
				EDPInfo:     &discovery.EDPDeviceInfo{DisplayName: "router-edp"},
				MDNSName:    "router.local",
				NetBIOSName: "ROUTER",
				Hostname:    "router.example.com",
			},
			expected: "router-cdp",
		},
		{
			name: "edp_when_no_lldp_cdp",
			device: &discovery.DiscoveredDevice{
				IP:          "192.168.1.1",
				EDPInfo:     &discovery.EDPDeviceInfo{DisplayName: "switch-edp"},
				MDNSName:    "switch.local",
				NetBIOSName: "SWITCH",
				Hostname:    "switch.example.com",
			},
			expected: "switch-edp",
		},
		{
			name: "mdns_when_no_network_protocols",
			device: &discovery.DiscoveredDevice{
				IP:          "192.168.1.1",
				MDNSName:    "macbook.local",
				NetBIOSName: "MACBOOK",
				Hostname:    "macbook.example.com",
			},
			expected: "macbook.local",
		},
		{
			name: "netbios_when_no_mdns",
			device: &discovery.DiscoveredDevice{
				IP:          "192.168.1.1",
				NetBIOSName: "DESKTOP-WIN10",
				Hostname:    "desktop.example.com",
			},
			expected: "DESKTOP-WIN10",
		},
		{
			name: "hostname_when_no_mdns_netbios",
			device: &discovery.DiscoveredDevice{
				IP:       "192.168.1.1",
				Hostname: "server.example.com",
			},
			expected: "server.example.com",
		},
		{
			name: "ip_fallback",
			device: &discovery.DiscoveredDevice{
				IP:  "192.168.1.1",
				MAC: "00:11:22:33:44:55",
			},
			expected: "192.168.1.1",
		},
		{
			name: "mac_last_resort",
			device: &discovery.DiscoveredDevice{
				MAC: "AA:BB:CC:DD:EE:FF",
			},
			expected: "AA:BB:CC:DD:EE:FF",
		},
		{
			name: "empty_lldp_falls_through",
			device: &discovery.DiscoveredDevice{
				IP: "192.168.1.1",
				LLDPInfo: &discovery.LLDPDeviceInfo{
					SystemName: "", // Empty
				},
				MDNSName: "device.local",
			},
			expected: "device.local",
		},
		{
			name: "empty_cdp_falls_through",
			device: &discovery.DiscoveredDevice{
				IP: "192.168.1.1",
				CDPInfo: &discovery.CDPDeviceInfo{
					DeviceID: "", // Empty
				},
				NetBIOSName: "DEVICE",
			},
			expected: "DEVICE",
		},
		{
			name: "empty_edp_falls_through",
			device: &discovery.DiscoveredDevice{
				IP: "192.168.1.1",
				EDPInfo: &discovery.EDPDeviceInfo{
					DisplayName: "", // Empty
				},
				Hostname: "device.local",
			},
			expected: "device.local",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.device.ComputeDisplayName()
			if result != tt.expected {
				t.Errorf("ComputeDisplayName() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestProtocolDeviceInfo_Fields(t *testing.T) {
	t.Run("lldp_info", func(t *testing.T) {
		info := &discovery.LLDPDeviceInfo{
			ChassisID:         "00:11:22:33:44:55",
			PortID:            "GigabitEthernet0/1",
			PortDescription:   "Uplink to Core",
			SystemName:        "access-switch-01",
			SystemDescription: "Cisco IOS Software, C2960 Software",
			Capabilities:      []string{"Bridge", "Router"},
			ManagementAddress: "10.0.0.10",
		}

		if info.ChassisID != "00:11:22:33:44:55" {
			t.Errorf("ChassisID mismatch: got %q", info.ChassisID)
		}
		if len(info.Capabilities) != 2 {
			t.Errorf("Capabilities should have 2 entries, got %d", len(info.Capabilities))
		}
	})

	t.Run("cdp_info", func(t *testing.T) {
		info := &discovery.CDPDeviceInfo{
			DeviceID:          "core-router.example.com",
			PortID:            "TenGigabitEthernet0/0/0",
			Platform:          "Cisco Nexus9000 C9300",
			SoftwareVersion:   "16.12.4",
			Capabilities:      []string{"Router", "Switch", "IGMP"},
			ManagementAddress: "10.0.0.1",
			NativeVLAN:        1,
			VoiceVLAN:         100,
		}

		if info.DeviceID != "core-router.example.com" {
			t.Errorf("DeviceID mismatch: got %q", info.DeviceID)
		}
		if info.NativeVLAN != 1 {
			t.Errorf("NativeVLAN should be 1, got %d", info.NativeVLAN)
		}
		if info.VoiceVLAN != 100 {
			t.Errorf("VoiceVLAN should be 100, got %d", info.VoiceVLAN)
		}
	})

	t.Run("edp_info", func(t *testing.T) {
		info := &discovery.EDPDeviceInfo{
			DeviceID:        "extreme-switch-01",
			DisplayName:     "Extreme Networks Switch",
			PortID:          "1:25",
			Platform:        "Summit X460",
			SoftwareVersion: "31.7.1.4",
			VLAN:            10,
		}

		if info.DeviceID != "extreme-switch-01" {
			t.Errorf("DeviceID mismatch: got %q", info.DeviceID)
		}
		if info.VLAN != 10 {
			t.Errorf("VLAN should be 10, got %d", info.VLAN)
		}
	})

	t.Run("ndp_info", func(t *testing.T) {
		now := time.Now()
		info := &discovery.NDPDeviceInfo{
			LinkLayerAddress:  "00:11:22:33:44:55",
			IsRouter:          true,
			ReachableTime:     30000,
			RetransTimer:      1000,
			Flags:             0x80,
			LastAdvertisement: now,
		}

		if !info.IsRouter {
			t.Error("IsRouter should be true")
		}
		if info.ReachableTime != 30000 {
			t.Errorf("ReachableTime should be 30000, got %d", info.ReachableTime)
		}
		if info.Flags != 0x80 {
			t.Errorf("Flags should be 0x80, got %d", info.Flags)
		}
	})
}

func TestDiscoveredDevice_DuplicateIP_Comprehensive(t *testing.T) {
	device := &discovery.DiscoveredDevice{
		IP:             "192.168.1.1",
		MAC:            "00:11:22:33:44:55",
		HasDuplicateIP: true,
		DuplicateMACs: []string{
			"AA:BB:CC:DD:EE:FF",
			"11:22:33:44:55:66",
			"22:33:44:55:66:77",
		},
	}

	if !device.HasDuplicateIP {
		t.Error("HasDuplicateIP should be true")
	}
	if len(device.DuplicateMACs) != 3 {
		t.Errorf("DuplicateMACs should have 3 entries, got %d", len(device.DuplicateMACs))
	}
}

func TestDeviceProfile_Comprehensive(t *testing.T) {
	now := time.Now()

	profile := &discovery.DeviceProfile{
		ProfiledAt: now,
		OpenPorts: []discovery.OpenPort{
			{Port: 22, Service: "ssh", Banner: "SSH-2.0-OpenSSH_8.4"},
			{Port: 80, Service: "http", Banner: ""},
			{Port: 443, Service: "https", Banner: ""},
			{Port: 3306, Service: "mysql", Banner: "5.7.36"},
		},
		MDNSServices: []discovery.MDNSService{
			{Name: "_ssh._tcp.local.", Port: 22},
			{Name: "_http._tcp.local.", Port: 80},
		},
		HTTPInfo: &discovery.HTTPInfo{
			Server: "nginx/1.18.0",
			Port:   80,
		},
		DeviceType:  "server",
		DeviceIcons: []string{"server", "linux"},
	}

	if profile.ProfiledAt.IsZero() {
		t.Error("ProfiledAt should be set")
	}
	if len(profile.OpenPorts) != 4 {
		t.Errorf("OpenPorts should have 4 entries, got %d", len(profile.OpenPorts))
	}
	if len(profile.MDNSServices) != 2 {
		t.Errorf("MDNSServices should have 2 entries, got %d", len(profile.MDNSServices))
	}
	if profile.HTTPInfo == nil {
		t.Error("HTTPInfo should not be nil")
	}
	if profile.DeviceType != "server" {
		t.Errorf("DeviceType should be 'server', got %q", profile.DeviceType)
	}
	if len(profile.DeviceIcons) != 2 {
		t.Errorf("DeviceIcons should have 2 entries, got %d", len(profile.DeviceIcons))
	}
}

func TestOpenPort_Comprehensive(t *testing.T) {
	tests := []struct {
		name string
		port discovery.OpenPort
	}{
		{
			name: "ssh_with_banner",
			port: discovery.OpenPort{
				Port:    22,
				Service: "ssh",
				Banner:  "SSH-2.0-OpenSSH_8.4p1 Ubuntu-5ubuntu1",
			},
		},
		{
			name: "https_no_banner",
			port: discovery.OpenPort{
				Port:    443,
				Service: "https",
				Banner:  "",
			},
		},
		{
			name: "custom_port",
			port: discovery.OpenPort{
				Port:    8080,
				Service: "http-alt",
				Banner:  "HTTP/1.1 200 OK",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.port.Port <= 0 || tt.port.Port > 65535 {
				t.Errorf("Port should be valid, got %d", tt.port.Port)
			}
			if tt.port.Service == "" {
				t.Error("Service should not be empty")
			}
		})
	}
}

func TestHTTPInfo_Comprehensive(t *testing.T) {
	info := &discovery.HTTPInfo{
		Server: "Apache/2.4.41 (Ubuntu)",
		Port:   80,
	}

	if info.Server != "Apache/2.4.41 (Ubuntu)" {
		t.Errorf("Server mismatch: got %q", info.Server)
	}
	if info.Port != 80 {
		t.Errorf("Port should be 80, got %d", info.Port)
	}
}

func TestMDNSService_Comprehensive(t *testing.T) {
	service := discovery.MDNSService{
		Name: "_airplay._tcp.local.",
		Port: 7000,
	}

	if service.Name != "_airplay._tcp.local." {
		t.Errorf("Name mismatch: got %q", service.Name)
	}
	if service.Port != 7000 {
		t.Errorf("Port should be 7000, got %d", service.Port)
	}
}

func TestStatus_Comprehensive(t *testing.T) {
	now := time.Now()

	status := &discovery.Status{
		Scanning:    true,
		DeviceCount: 42,
		LastScan:    now,
		Subnet:      "192.168.1.0/24",
		LocalIP:     "192.168.1.100",
		Interface:   "en0",
	}

	if !status.Scanning {
		t.Error("Scanning should be true")
	}
	if status.DeviceCount != 42 {
		t.Errorf("DeviceCount should be 42, got %d", status.DeviceCount)
	}
	if !status.LastScan.Equal(now) {
		t.Error("LastScan mismatch")
	}
	if status.Subnet != "192.168.1.0/24" {
		t.Errorf("Subnet should be '192.168.1.0/24', got %q", status.Subnet)
	}
	if status.LocalIP != "192.168.1.100" {
		t.Errorf("LocalIP should be '192.168.1.100', got %q", status.LocalIP)
	}
	if status.Interface != "en0" {
		t.Errorf("Interface should be 'en0', got %q", status.Interface)
	}
}

func TestDeviceDiscovery_Basic(t *testing.T) {
	// Test creation with loopback interface
	dd := discovery.NewDeviceDiscovery("lo0")
	if dd == nil {
		dd = discovery.NewDeviceDiscovery("lo")
	}

	if dd == nil {
		t.Skip("No loopback interface available")
	}

	// Test GetInterfaceName
	iface := dd.GetInterfaceName()
	if iface != "lo0" && iface != "lo" {
		t.Errorf("Interface should be 'lo0' or 'lo', got %q", iface)
	}

	// Test Count on empty discovery
	count := dd.Count()
	if count != 0 {
		t.Errorf("Count should be 0 initially, got %d", count)
	}

	// Test IsScanning
	if dd.IsScanning() {
		t.Error("Should not be scanning initially")
	}

	// Test GetDevices on empty
	devices := dd.GetDevices()
	if len(devices) != 0 {
		t.Errorf("GetDevices should return empty slice, got %d", len(devices))
	}

	// Test GetDevice on empty
	device := dd.GetDevice("00:11:22:33:44:55")
	if device != nil {
		t.Error("GetDevice should return nil for non-existent MAC")
	}

	// Test GetDeviceByIP on empty
	deviceByIP := dd.GetDeviceByIP("192.168.1.1")
	if deviceByIP != nil {
		t.Error("GetDeviceByIP should return nil for non-existent IP")
	}

	// Test SetNameResolution
	dd.SetNameResolution(true)
	dd.SetNameResolution(false)

	// Test ClearDevices
	dd.ClearDevices()
	if dd.Count() != 0 {
		t.Error("Count should be 0 after ClearDevices")
	}

	// Test GetStatus
	status := dd.GetStatus()
	if status == nil {
		t.Fatal("GetStatus should not return nil")
	}
	if status.Scanning {
		t.Error("Status.Scanning should be false")
	}
}

func TestErrScanInProgress_Comprehensive(t *testing.T) {
	err := discovery.ErrScanInProgress

	if err == nil {
		t.Fatal("ErrScanInProgress should not be nil")
	}
	if err.Error() != "scan already in progress" {
		t.Errorf("Error message mismatch: got %q", err.Error())
	}
}

func TestIsLocallyAdministeredMAC_Comprehensive(t *testing.T) {
	tests := []struct {
		name     string
		mac      string
		expected bool
	}{
		// LAA MACs (second bit of first octet set)
		{"laa_02", "02:00:00:00:00:00", true},
		{"laa_06", "06:00:00:00:00:00", true},
		{"laa_0A", "0A:11:22:33:44:55", true},
		{"laa_0E", "0E:00:00:00:00:00", true},
		{"laa_vmware", "02:50:56:00:00:01", true},
		{"laa_hyperv", "02:15:5D:00:00:01", true},
		{"laa_docker", "02:42:AC:11:00:02", true},
		{"laa_xen", "02:00:00:00:00:01", true},

		// UAA MACs (second bit NOT set)
		{"uaa_00", "00:11:22:33:44:55", false},
		{"uaa_04", "04:00:00:00:00:00", false},
		{"uaa_08", "08:00:00:00:00:00", false},
		{"uaa_apple", "A4:83:E7:12:34:56", false},
		{"uaa_dell", "D0:67:E5:12:34:56", false},
		{"uaa_cisco", "00:1A:2B:3C:4D:5E", false},

		// Edge cases
		{"empty", "", false},
		{"too_short", "0", false},
		{"single_char", "A", false},
		{"lowercase_laa", "0a:11:22:33:44:55", true},
		{"lowercase_uaa", "00:11:22:33:44:55", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := discovery.ExportIsLocallyAdministeredMAC(tt.mac)
			if result != tt.expected {
				t.Errorf(
					"isLocallyAdministeredMAC(%q) = %v, expected %v",
					tt.mac,
					result,
					tt.expected,
				)
			}
		})
	}
}
