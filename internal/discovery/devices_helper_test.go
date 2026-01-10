package discovery_test

import (
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/discovery"
)

func TestMethod_Constants(t *testing.T) {
	tests := []struct {
		method   discovery.Method
		expected string
	}{
		{discovery.MethodARP, "arp"},
		{discovery.MethodNDP, "ndp"},
		{discovery.MethodLLDP, "lldp"},
		{discovery.MethodCDP, "cdp"},
		{discovery.MethodEDP, "edp"},
		{discovery.MethodMDNS, "mdns"},
		{discovery.MethodPING, "ping"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if string(tt.method) != tt.expected {
				t.Errorf("Expected Method=%s, got %s", tt.expected, string(tt.method))
			}
		})
	}
}

func TestDiscoveredDevice_Fields(t *testing.T) {
	now := time.Now()
	device := &discovery.DiscoveredDevice{
		IP:            "192.168.1.1",
		IPv6Address:   "fe80::1",
		IPv6Addresses: []string{"fe80::1", "2001:db8::1"},
		MAC:           "00:11:22:33:44:55",
		Hostname:      "test-device.local",
		NetBIOSName:   "TESTDEVICE",
		MDNSName:      "test-device.local",
		DisplayName:   "Test Device",
		Vendor:        "Test Vendor",
		OSGuess:       "Linux",
		TTL:           64,
		DiscoveryMethod: []discovery.Method{
			discovery.MethodARP,
			discovery.MethodPING,
		},
		LastSeen:       now,
		IsLocal:        true,
		IsRouter:       false,
		HasDuplicateIP: false,
	}

	if device.IP != "192.168.1.1" {
		t.Errorf("Expected IP='192.168.1.1', got %s", device.IP)
	}
	if device.IPv6Address != "fe80::1" {
		t.Errorf("Expected IPv6Address='fe80::1', got %s", device.IPv6Address)
	}
	if len(device.IPv6Addresses) != 2 {
		t.Errorf("Expected 2 IPv6Addresses, got %d", len(device.IPv6Addresses))
	}
	if device.MAC != "00:11:22:33:44:55" {
		t.Errorf("Expected MAC='00:11:22:33:44:55', got %s", device.MAC)
	}
	if device.Hostname != "test-device.local" {
		t.Errorf("Expected Hostname='test-device.local', got %s", device.Hostname)
	}
	if device.NetBIOSName != "TESTDEVICE" {
		t.Errorf("Expected NetBIOSName='TESTDEVICE', got %s", device.NetBIOSName)
	}
	if device.MDNSName != "test-device.local" {
		t.Errorf("Expected MDNSName='test-device.local', got %s", device.MDNSName)
	}
	if device.DisplayName != "Test Device" {
		t.Errorf("Expected DisplayName='Test Device', got %s", device.DisplayName)
	}
	if device.Vendor != "Test Vendor" {
		t.Errorf("Expected Vendor='Test Vendor', got %s", device.Vendor)
	}
	if device.OSGuess != "Linux" {
		t.Errorf("Expected OSGuess='Linux', got %s", device.OSGuess)
	}
	if device.TTL != 64 {
		t.Errorf("Expected TTL=64, got %d", device.TTL)
	}
	if len(device.DiscoveryMethod) != 2 {
		t.Errorf("Expected 2 DiscoveryMethods, got %d", len(device.DiscoveryMethod))
	}
	if !device.LastSeen.Equal(now) {
		t.Error("LastSeen should match now")
	}
	if !device.IsLocal {
		t.Error("Expected IsLocal=true")
	}
	if device.IsRouter {
		t.Error("Expected IsRouter=false")
	}
	if device.HasDuplicateIP {
		t.Error("Expected HasDuplicateIP=false")
	}
}

func TestLLDPDeviceInfo_Fields(t *testing.T) {
	info := &discovery.LLDPDeviceInfo{
		ChassisID:         "00:11:22:33:44:55",
		PortID:            "Ethernet0",
		PortDescription:   "Uplink port",
		SystemName:        "switch01",
		SystemDescription: "Network Switch",
		Capabilities:      []string{"Bridge", "Router"},
		ManagementAddress: "192.168.1.1",
	}

	if info.ChassisID != "00:11:22:33:44:55" {
		t.Errorf("Expected ChassisID='00:11:22:33:44:55', got %s", info.ChassisID)
	}
	if info.PortID != "Ethernet0" {
		t.Errorf("Expected PortID='Ethernet0', got %s", info.PortID)
	}
	if info.PortDescription != "Uplink port" {
		t.Errorf("Expected PortDescription='Uplink port', got %s", info.PortDescription)
	}
	if info.SystemName != "switch01" {
		t.Errorf("Expected SystemName='switch01', got %s", info.SystemName)
	}
	if info.SystemDescription != "Network Switch" {
		t.Errorf("Expected SystemDescription='Network Switch', got %s", info.SystemDescription)
	}
	if len(info.Capabilities) != 2 {
		t.Errorf("Expected 2 Capabilities, got %d", len(info.Capabilities))
	}
	if info.ManagementAddress != "192.168.1.1" {
		t.Errorf("Expected ManagementAddress='192.168.1.1', got %s", info.ManagementAddress)
	}
}

func TestCDPDeviceInfo_Fields(t *testing.T) {
	info := &discovery.CDPDeviceInfo{
		DeviceID:          "router01.example.com",
		PortID:            "GigabitEthernet0/1",
		Platform:          "Cisco IOS",
		SoftwareVersion:   "15.1(4)M",
		Capabilities:      []string{"Router", "Switch"},
		ManagementAddress: "10.0.0.1",
		NativeVLAN:        100,
		VoiceVLAN:         200,
	}

	if info.DeviceID != "router01.example.com" {
		t.Errorf("Expected DeviceID='router01.example.com', got %s", info.DeviceID)
	}
	if info.PortID != "GigabitEthernet0/1" {
		t.Errorf("Expected PortID='GigabitEthernet0/1', got %s", info.PortID)
	}
	if info.Platform != "Cisco IOS" {
		t.Errorf("Expected Platform='Cisco IOS', got %s", info.Platform)
	}
	if info.SoftwareVersion != "15.1(4)M" {
		t.Errorf("Expected SoftwareVersion='15.1(4)M', got %s", info.SoftwareVersion)
	}
	if len(info.Capabilities) != 2 {
		t.Errorf("Expected 2 Capabilities, got %d", len(info.Capabilities))
	}
	if info.ManagementAddress != "10.0.0.1" {
		t.Errorf("Expected ManagementAddress='10.0.0.1', got %s", info.ManagementAddress)
	}
	if info.NativeVLAN != 100 {
		t.Errorf("Expected NativeVLAN=100, got %d", info.NativeVLAN)
	}
	if info.VoiceVLAN != 200 {
		t.Errorf("Expected VoiceVLAN=200, got %d", info.VoiceVLAN)
	}
}

func TestEDPDeviceInfo_Fields(t *testing.T) {
	info := &discovery.EDPDeviceInfo{
		DeviceID:        "switch01",
		DisplayName:     "Core Switch",
		PortID:          "port1",
		Platform:        "Extreme Networks",
		SoftwareVersion: "30.1.1",
		VLAN:            10,
	}

	if info.DeviceID != "switch01" {
		t.Errorf("Expected DeviceID='switch01', got %s", info.DeviceID)
	}
	if info.DisplayName != "Core Switch" {
		t.Errorf("Expected DisplayName='Core Switch', got %s", info.DisplayName)
	}
	if info.PortID != "port1" {
		t.Errorf("Expected PortID='port1', got %s", info.PortID)
	}
	if info.Platform != "Extreme Networks" {
		t.Errorf("Expected Platform='Extreme Networks', got %s", info.Platform)
	}
	if info.SoftwareVersion != "30.1.1" {
		t.Errorf("Expected SoftwareVersion='30.1.1', got %s", info.SoftwareVersion)
	}
	if info.VLAN != 10 {
		t.Errorf("Expected VLAN=10, got %d", info.VLAN)
	}
}

func TestNDPDeviceInfo_Fields(t *testing.T) {
	now := time.Now()
	info := &discovery.NDPDeviceInfo{
		LinkLayerAddress:  "00:11:22:33:44:55",
		IsRouter:          true,
		ReachableTime:     30000,
		RetransTimer:      1000,
		Flags:             0x80,
		LastAdvertisement: now,
	}

	if info.LinkLayerAddress != "00:11:22:33:44:55" {
		t.Errorf("Expected LinkLayerAddress='00:11:22:33:44:55', got %s", info.LinkLayerAddress)
	}
	if !info.IsRouter {
		t.Error("Expected IsRouter=true")
	}
	if info.ReachableTime != 30000 {
		t.Errorf("Expected ReachableTime=30000, got %d", info.ReachableTime)
	}
	if info.RetransTimer != 1000 {
		t.Errorf("Expected RetransTimer=1000, got %d", info.RetransTimer)
	}
	if info.Flags != 0x80 {
		t.Errorf("Expected Flags=0x80, got %d", info.Flags)
	}
	if !info.LastAdvertisement.Equal(now) {
		t.Error("Expected LastAdvertisement to match")
	}
}

func TestStatus_Fields(t *testing.T) {
	now := time.Now()
	status := &discovery.Status{
		Scanning:    true,
		DeviceCount: 50,
		LastScan:    now,
		Subnet:      "192.168.1.0/24",
		LocalIP:     "192.168.1.100",
		Interface:   "eth0",
	}

	if !status.Scanning {
		t.Error("Expected Scanning=true")
	}
	if status.DeviceCount != 50 {
		t.Errorf("Expected DeviceCount=50, got %d", status.DeviceCount)
	}
	if !status.LastScan.Equal(now) {
		t.Error("Expected LastScan to match")
	}
	if status.Subnet != "192.168.1.0/24" {
		t.Errorf("Expected Subnet='192.168.1.0/24', got %s", status.Subnet)
	}
	if status.LocalIP != "192.168.1.100" {
		t.Errorf("Expected LocalIP='192.168.1.100', got %s", status.LocalIP)
	}
	if status.Interface != "eth0" {
		t.Errorf("Expected Interface='eth0', got %s", status.Interface)
	}
}

func TestComputeDisplayName_Priorities(t *testing.T) {
	tests := []struct {
		name     string
		device   *discovery.DiscoveredDevice
		expected string
	}{
		{
			name: "LLDP_priority",
			device: &discovery.DiscoveredDevice{
				IP: "192.168.1.1",
				LLDPInfo: &discovery.LLDPDeviceInfo{
					SystemName: "lldp-switch",
				},
				CDPInfo:     &discovery.CDPDeviceInfo{DeviceID: "cdp-device"},
				MDNSName:    "mdns.local",
				NetBIOSName: "NETBIOS",
				Hostname:    "hostname.local",
			},
			expected: "lldp-switch",
		},
		{
			name: "CDP_priority",
			device: &discovery.DiscoveredDevice{
				IP:          "192.168.1.1",
				CDPInfo:     &discovery.CDPDeviceInfo{DeviceID: "cdp-router"},
				MDNSName:    "mdns.local",
				NetBIOSName: "NETBIOS",
				Hostname:    "hostname.local",
			},
			expected: "cdp-router",
		},
		{
			name: "EDP_priority",
			device: &discovery.DiscoveredDevice{
				IP:          "192.168.1.1",
				EDPInfo:     &discovery.EDPDeviceInfo{DisplayName: "edp-switch"},
				MDNSName:    "mdns.local",
				NetBIOSName: "NETBIOS",
				Hostname:    "hostname.local",
			},
			expected: "edp-switch",
		},
		{
			name: "mDNS_priority",
			device: &discovery.DiscoveredDevice{
				IP:          "192.168.1.1",
				MDNSName:    "johns-macbook.local",
				NetBIOSName: "NETBIOS",
				Hostname:    "hostname.local",
			},
			expected: "johns-macbook.local",
		},
		{
			name: "NetBIOS_priority",
			device: &discovery.DiscoveredDevice{
				IP:          "192.168.1.1",
				NetBIOSName: "DESKTOP-ABC123",
				Hostname:    "hostname.local",
			},
			expected: "DESKTOP-ABC123",
		},
		{
			name: "Hostname_priority",
			device: &discovery.DiscoveredDevice{
				IP:       "192.168.1.1",
				Hostname: "server.example.com",
			},
			expected: "server.example.com",
		},
		{
			name: "IP_fallback",
			device: &discovery.DiscoveredDevice{
				IP:  "192.168.1.1",
				MAC: "00:11:22:33:44:55",
			},
			expected: "192.168.1.1",
		},
		{
			name: "MAC_last_resort",
			device: &discovery.DiscoveredDevice{
				MAC: "AA:BB:CC:DD:EE:FF",
			},
			expected: "AA:BB:CC:DD:EE:FF",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.device.ComputeDisplayName()
			if result != tt.expected {
				t.Errorf("Expected DisplayName=%q, got %q", tt.expected, result)
			}
		})
	}
}

func TestDiscoveredDevice_DuplicateIP(t *testing.T) {
	device := &discovery.DiscoveredDevice{
		IP:             "192.168.1.1",
		MAC:            "00:11:22:33:44:55",
		HasDuplicateIP: true,
		DuplicateMACs:  []string{"AA:BB:CC:DD:EE:FF", "11:22:33:44:55:66"},
	}

	if !device.HasDuplicateIP {
		t.Error("Expected HasDuplicateIP=true")
	}
	if device.IP != "192.168.1.1" {
		t.Errorf("Expected IP='192.168.1.1', got %s", device.IP)
	}
	if device.MAC != "00:11:22:33:44:55" {
		t.Errorf("Expected MAC='00:11:22:33:44:55', got %s", device.MAC)
	}
	if len(device.DuplicateMACs) != 2 {
		t.Errorf("Expected 2 DuplicateMACs, got %d", len(device.DuplicateMACs))
	}
	if device.DuplicateMACs[0] != "AA:BB:CC:DD:EE:FF" {
		t.Errorf("Expected DuplicateMACs[0]='AA:BB:CC:DD:EE:FF', got %s", device.DuplicateMACs[0])
	}
}

func TestDeviceDiscovery_GetInterfaceName(t *testing.T) {
	dd := discovery.NewDeviceDiscovery("lo")

	interfaceName := dd.GetInterfaceName()
	if interfaceName != "lo" {
		t.Errorf("Expected InterfaceName='lo', got %s", interfaceName)
	}
}

func TestDeviceDiscovery_SetNameResolution(t *testing.T) {
	t.Parallel()

	dd := discovery.NewDeviceDiscovery("lo")

	// Enable name resolution
	dd.SetNameResolution(true)

	// Disable name resolution
	dd.SetNameResolution(false)

	// Should not panic
}

func TestErrScanInProgress(t *testing.T) {
	err := discovery.ErrScanInProgress

	if err.Error() != "scan already in progress" {
		t.Errorf("Expected error message 'scan already in progress', got %s", err.Error())
	}
}

func TestIsLocallyAdministeredMAC(t *testing.T) {
	tests := []struct {
		name     string
		mac      string
		expected bool
	}{
		{"LAA_second_bit_set", "02:00:00:00:00:00", true},
		{"LAA_second_bit_set_upper", "0A:11:22:33:44:55", true},
		{"UAA_no_bit_set", "00:11:22:33:44:55", false},
		{"UAA_common_vendor", "00:1A:2B:3C:4D:5E", false},
		{"LAA_vmware_style", "02:50:56:00:00:01", true},
		{"empty_string", "", false},
		{"too_short", "0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := discovery.ExportIsLocallyAdministeredMAC(tt.mac)
			if result != tt.expected {
				t.Errorf("isLocallyAdministeredMAC(%q) = %v, expected %v", tt.mac, result, tt.expected)
			}
		})
	}
}
