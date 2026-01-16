package discovery

// This file implements Wake-on-LAN (WoL) functionality for network device discovery.
//
// Wake-on-LAN allows waking up computers remotely by sending a "magic packet"
// containing the target device's MAC address. This is useful for:
// - Waking up workstations for remote management
// - Power management and scheduling
// - Remote access preparation

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	// wolMagicPacketSize is 6 bytes FF + 16 * 6 bytes MAC.
	wolMagicPacketSize = 102
	// wolDefaultPort is the standard WoL port.
	wolDefaultPort = 9
	// wolBroadcastIP is the default broadcast address.
	wolBroadcastIP = "255.255.255.255"
	// wolTimeout is the default timeout for sending WoL packets.
	wolTimeout = 2 * time.Second
	// wolMACLength is the standard MAC address length in bytes.
	wolMACLength = 6

	// WoLStatusUntested indicates WoL capability has not been tested.
	WoLStatusUntested = "untested"
	// WoLStatusSuccess indicates a WoL packet was sent successfully.
	WoLStatusSuccess = "success"
	// WoLStatusFailed indicates a WoL packet send failed.
	WoLStatusFailed = "failed"
)

// WoLPacket represents a Wake-on-LAN magic packet.
type WoLPacket struct {
	MAC       string // Target MAC address (e.g., "AA:BB:CC:DD:EE:FF")
	SecureOn  []byte // Optional SecureOn password (4-6 bytes)
	Broadcast string // Broadcast address (default: 255.255.255.255)
	Port      int    // UDP port (default: 9)
}

// WoLResult represents the result of a WoL packet send attempt.
type WoLResult struct {
	MAC       string
	Success   bool
	Error     error
	Timestamp time.Time
}

// SendWakeOnLAN sends a Wake-on-LAN magic packet to wake up a device.
// The magic packet consists of 6 bytes of 0xFF followed by the target MAC
// address repeated 16 times.
func SendWakeOnLAN(ctx context.Context, mac string) (*WoLResult, error) {
	return SendWakeOnLANWithOptions(ctx, &WoLPacket{
		MAC:       mac,
		Broadcast: wolBroadcastIP,
		Port:      wolDefaultPort,
	})
}

// SendWakeOnLANWithOptions sends a WoL packet with custom options.
func SendWakeOnLANWithOptions(ctx context.Context, packet *WoLPacket) (*WoLResult, error) {
	if packet == nil {
		return nil, errors.New("packet cannot be nil")
	}

	result := &WoLResult{
		MAC:       packet.MAC,
		Timestamp: time.Now(),
	}

	// Parse MAC address
	macBytes, err := parseMACAddress(packet.MAC)
	if err != nil {
		result.Error = fmt.Errorf("invalid MAC address: %w", err)
		return result, result.Error
	}

	// Build magic packet
	magicPacket := buildMagicPacket(macBytes, packet.SecureOn)

	// Set broadcast address and port
	broadcast := packet.Broadcast
	if broadcast == "" {
		broadcast = wolBroadcastIP
	}
	port := packet.Port
	if port == 0 {
		port = wolDefaultPort
	}

	// Send packet via UDP broadcast
	sendErr := sendUDPBroadcast(ctx, broadcast, port, magicPacket)
	if sendErr != nil {
		result.Error = fmt.Errorf("send failed: %w", sendErr)
		return result, result.Error
	}

	result.Success = true
	return result, nil
}

// parseMACAddress parses a MAC address string into bytes.
// Supports formats: "AA:BB:CC:DD:EE:FF", "AA-BB-CC-DD-EE-FF", "AABBCCDDEEFF".
func parseMACAddress(mac string) ([]byte, error) {
	// Normalize separator
	mac = strings.ReplaceAll(mac, "-", ":")
	mac = strings.ReplaceAll(mac, ".", ":")

	// Handle no-separator format
	if !strings.Contains(mac, ":") && len(mac) == 12 {
		var parts []string
		for i := 0; i < 12; i += 2 {
			parts = append(parts, mac[i:i+2])
		}
		mac = strings.Join(parts, ":")
	}

	hwAddr, err := net.ParseMAC(mac)
	if err != nil {
		return nil, err
	}

	if len(hwAddr) != wolMACLength {
		return nil, errors.New("MAC address must be 6 bytes")
	}

	return hwAddr, nil
}

// buildMagicPacket creates a Wake-on-LAN magic packet.
// The packet consists of 6 bytes of 0xFF (synchronization stream),
// target MAC address repeated 16 times (96 bytes), and an
// optional SecureOn password (4-6 bytes).
func buildMagicPacket(mac []byte, secureOn []byte) []byte {
	// Calculate packet size
	packetSize := wolMagicPacketSize
	if len(secureOn) > 0 {
		packetSize += len(secureOn)
	}

	packet := make([]byte, packetSize)

	// First 6 bytes are 0xFF
	for i := range 6 {
		packet[i] = 0xFF
	}

	// Repeat MAC address 16 times
	for i := range 16 {
		copy(packet[6+i*6:6+(i+1)*6], mac)
	}

	// Append SecureOn password if provided
	if len(secureOn) > 0 {
		copy(packet[wolMagicPacketSize:], secureOn)
	}

	return packet
}

// sendUDPBroadcast sends a UDP packet to the broadcast address.
func sendUDPBroadcast(ctx context.Context, broadcast string, port int, data []byte) error {
	addr := net.JoinHostPort(broadcast, strconv.Itoa(port))

	// Use context for timeout
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, wolTimeout)
		defer cancel()
		deadline, _ = ctx.Deadline()
	}

	// Resolve UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("resolve address: %w", err)
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Set deadline
	_ = conn.SetDeadline(deadline)

	// Send magic packet
	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

// getWoLDeviceTypeSupport returns a map of device types to WoL support (true=yes, false=no).
func getWoLDeviceTypeSupport() map[string]bool {
	return map[string]bool{
		"switch": false, "router": false, "firewall": false,
		"access-point": false, "network-device": false,
		"printer": false, "print-server": false,
		"ip-camera": false, "camera": false,
		"computer": true, "desktop": true, "workstation": true, "server": true,
	}
}

// InferWoLCapability guesses whether a device likely supports Wake-on-LAN
// based on its device type and characteristics.
func InferWoLCapability(device *DiscoveredDevice) *bool {
	if device == nil {
		return nil
	}
	if result := inferWoLFromProfile(device); result != nil {
		return result
	}
	if result := inferWoLFromSNMP(device); result != nil {
		return result
	}
	return inferWoLFromOS(device)
}

func inferWoLFromProfile(device *DiscoveredDevice) *bool {
	if device.Profile == nil {
		return nil
	}
	deviceType := strings.ToLower(device.Profile.DeviceType)
	if deviceType == "laptop" || deviceType == "notebook" {
		return nil // Unknown - often disabled on laptops
	}
	if supported, ok := getWoLDeviceTypeSupport()[deviceType]; ok {
		return &supported
	}
	return nil
}

func inferWoLFromSNMP(device *DiscoveredDevice) *bool {
	if device.Profile == nil || device.Profile.SNMPInfo == nil {
		return nil
	}
	sysDescr := strings.ToLower(device.Profile.SNMPInfo.SysDescr)
	if containsAny(sysDescr, "switch", "router", "cisco", "juniper", "ubiquiti", "mikrotik") {
		f := false
		return &f
	}
	if containsAny(sysDescr, "windows", "linux") {
		t := true
		return &t
	}
	return nil
}

func inferWoLFromOS(device *DiscoveredDevice) *bool {
	if device.OSGuess == "" {
		return nil
	}
	osGuess := strings.ToLower(device.OSGuess)
	if containsAny(osGuess, "windows", "linux", "macos") {
		t := true
		return &t
	}
	if containsAny(osGuess, "ios", "switch") {
		f := false
		return &f
	}
	return nil
}

// SendWakeOnLANToSubnet sends WoL packets to the subnet broadcast address.
// This is useful when the device is on a different subnet.
func SendWakeOnLANToSubnet(ctx context.Context, mac, subnetBroadcast string) (*WoLResult, error) {
	return SendWakeOnLANWithOptions(ctx, &WoLPacket{
		MAC:       mac,
		Broadcast: subnetBroadcast,
		Port:      wolDefaultPort,
	})
}

// WakeDevices sends WoL packets to multiple devices.
func WakeDevices(ctx context.Context, macs []string) []WoLResult {
	results := make([]WoLResult, 0, len(macs))

	for _, mac := range macs {
		result, _ := SendWakeOnLAN(ctx, mac)
		if result != nil {
			results = append(results, *result)
		}
	}

	return results
}
