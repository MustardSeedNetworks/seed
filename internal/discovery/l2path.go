// Package discovery implements multi-protocol network device discovery.
// L2 path discovery traces Layer 2 switch paths between devices using LLDP/CDP neighbor data.
package discovery

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/snmp"
)

// L2PathResult represents the complete Layer 2 path between two endpoints.
type L2PathResult struct {
	Hops []L2Hop `json:"hops"`
}

// L2Hop represents a single hop in the Layer 2 path.
type L2Hop struct {
	Device      string    `json:"device"`      // Switch name
	DeviceIP    string    `json:"deviceIp"`    // Switch management IP
	IngressPort *PortInfo `json:"ingressPort"` // Port where traffic enters
	EgressPort  *PortInfo `json:"egressPort"`  // Port where traffic exits
	Source      string    `json:"source"`      // "lldp", "cdp", "snmp"
}

// PortInfo contains detailed information about a switch port.
type PortInfo struct {
	Name        string `json:"name"`        // "Gi0/1", "eth1", etc.
	Index       int    `json:"index"`       // SNMP ifIndex
	Speed       string `json:"speed"`       // "1Gbps", "10Gbps", etc.
	Duplex      string `json:"duplex"`      // "full", "half", "auto"
	VLANs       []int  `json:"vlans"`       // VLANs on this port
	IsTrunk     bool   `json:"isTrunk"`     // true if port carries multiple VLANs
	ConnectedTo string `json:"connectedTo"` // Device name/MAC on this port
}

// L2PathBuilder builds Layer 2 paths between devices using LLDP/CDP and SNMP.
type L2PathBuilder struct {
	deviceDiscovery *DeviceDiscovery
	snmpConfig      *config.SNMPConfig
}

// NewL2PathBuilder creates a new L2 path builder.
func NewL2PathBuilder(deviceDiscovery *DeviceDiscovery, snmpConfig *config.SNMPConfig) *L2PathBuilder {
	return &L2PathBuilder{
		deviceDiscovery: deviceDiscovery,
		snmpConfig:      snmpConfig,
	}
}

// BuildPath traces the Layer 2 path from source to destination.
// Algorithm:
//  1. Find the first-hop switch using LLDP/CDP neighbor data from source device
//  2. For each switch hop:
//     - Identify ingress port (where source MAC appears)
//     - Identify egress port (LLDP/CDP neighbor toward destination)
//     - Query port details via SNMP if available
//  3. Continue until destination is reached or no more neighbors found.
func (b *L2PathBuilder) BuildPath(ctx context.Context, sourceIP, destIP string) (*L2PathResult, error) {
	result := &L2PathResult{
		Hops: make([]L2Hop, 0),
	}

	// Find source and destination devices
	sourceDevice := b.deviceDiscovery.GetDeviceByIP(sourceIP)
	destDevice := b.deviceDiscovery.GetDeviceByIP(destIP)

	if sourceDevice == nil {
		return nil, fmt.Errorf("source device not found: %s", sourceIP)
	}
	if destDevice == nil {
		return nil, fmt.Errorf("destination device not found: %s", destIP)
	}

	slog.Debug("Building L2 path",
		"source_ip", sourceIP,
		"source_mac", sourceDevice.MAC,
		"dest_ip", destIP,
		"dest_mac", destDevice.MAC)

	// Find the first-hop switch from source device's LLDP/CDP neighbors
	firstHop := b.findFirstHop(sourceDevice)
	if firstHop == nil {
		return nil, fmt.Errorf("no first-hop switch found for source device")
	}

	// Build the path hop by hop
	currentHop := firstHop
	visited := make(map[string]bool) // Prevent loops
	maxHops := 20                    // Safety limit

	for range maxHops {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if currentHop == nil {
			break
		}

		// Mark this hop as visited
		hopKey := currentHop.DeviceIP
		if visited[hopKey] {
			slog.Warn("Loop detected in L2 path", "device", currentHop.Device)
			break
		}
		visited[hopKey] = true

		// Add hop to result
		result.Hops = append(result.Hops, *currentHop)

		// Check if we reached the destination
		if b.isDestinationReached(currentHop, destDevice) {
			slog.Debug("Destination reached", "hops", len(result.Hops))
			break
		}

		// Find the next hop
		currentHop = b.findNextHop(ctx, currentHop, destDevice)
	}

	if len(result.Hops) == 0 {
		return nil, fmt.Errorf("unable to build L2 path - no hops found")
	}

	return result, nil
}

// findFirstHop finds the first-hop switch connected to the source device.
func (b *L2PathBuilder) findFirstHop(sourceDevice *DiscoveredDevice) *L2Hop {
	// Check LLDP neighbors first
	if sourceDevice.LLDPInfo != nil {
		hop := &L2Hop{
			Device:   sourceDevice.LLDPInfo.SystemName,
			DeviceIP: sourceDevice.LLDPInfo.ManagementAddress,
			Source:   "lldp",
		}

		// Create ingress port info from LLDP data
		hop.IngressPort = &PortInfo{
			Name:        sourceDevice.LLDPInfo.PortID,
			ConnectedTo: sourceDevice.MAC,
		}
		if sourceDevice.LLDPInfo.PortDescription != "" {
			hop.IngressPort.Name = sourceDevice.LLDPInfo.PortDescription
		}

		return hop
	}

	// Try CDP neighbors
	if sourceDevice.CDPInfo != nil {
		hop := &L2Hop{
			Device:   sourceDevice.CDPInfo.DeviceID,
			DeviceIP: sourceDevice.CDPInfo.ManagementAddress,
			Source:   "cdp",
		}

		// Create ingress port info from CDP data
		hop.IngressPort = &PortInfo{
			Name:        sourceDevice.CDPInfo.PortID,
			ConnectedTo: sourceDevice.MAC,
		}
		if sourceDevice.CDPInfo.NativeVLAN > 0 {
			hop.IngressPort.VLANs = []int{sourceDevice.CDPInfo.NativeVLAN}
		}

		return hop
	}

	// Try EDP neighbors
	if sourceDevice.EDPInfo != nil {
		hopName := sourceDevice.EDPInfo.DisplayName
		if hopName == "" {
			hopName = sourceDevice.EDPInfo.DeviceID
		}

		hop := &L2Hop{
			Device: hopName,
			// Note: EDPDeviceInfo doesn't have ManagementAddress field
			// It's only available in the raw EDPNeighbor data
			Source: "edp",
		}

		// Create ingress port info from EDP data
		hop.IngressPort = &PortInfo{
			Name:        sourceDevice.EDPInfo.PortID,
			ConnectedTo: sourceDevice.MAC,
		}
		if sourceDevice.EDPInfo.VLAN > 0 {
			hop.IngressPort.VLANs = []int{sourceDevice.EDPInfo.VLAN}
		}

		return hop
	}

	return nil
}

// findNextHop finds the next hop in the L2 path toward the destination.
func (b *L2PathBuilder) findNextHop(ctx context.Context, currentHop *L2Hop, destDevice *DiscoveredDevice) *L2Hop {
	// Query SNMP for more detailed port information if available
	if currentHop.DeviceIP != "" && b.snmpConfig != nil {
		b.enrichHopWithSNMP(ctx, currentHop)
	}

	// Find next hop by looking for LLDP/CDP neighbors from current switch
	// that might lead to the destination
	nextHop := b.findNeighborTowardDestination(currentHop, destDevice)

	return nextHop
}

// enrichHopWithSNMP enriches hop information using SNMP queries.
func (b *L2PathBuilder) enrichHopWithSNMP(ctx context.Context, hop *L2Hop) {
	if hop.DeviceIP == "" {
		return
	}

	// Check if SNMP is configured (has communities or v3 credentials)
	if b.snmpConfig == nil || (len(b.snmpConfig.Communities) == 0 && len(b.snmpConfig.V3Credentials) == 0) {
		return
	}

	// Try to get system info to confirm device identity
	systemInfo, err := snmp.GetSystemInfo(ctx, hop.DeviceIP, b.snmpConfig)
	if err != nil {
		slog.Debug("Failed to get SNMP system info", "device", hop.DeviceIP, "error", err)
		return
	}

	// Update device name from SNMP if available
	if systemInfo.SysName != "" && hop.Device == "" {
		hop.Device = systemInfo.SysName
	}

	slog.Debug("Enriched hop with SNMP data",
		"device", hop.Device,
		"sysname", systemInfo.SysName)
}

// findNeighborTowardDestination finds a neighbor that might lead to the destination.
func (b *L2PathBuilder) findNeighborTowardDestination(currentHop *L2Hop, destDevice *DiscoveredDevice) *L2Hop {
	// Get all devices to search for neighbors
	allDevices := b.deviceDiscovery.GetDevices()

	// Look for devices that have LLDP/CDP info pointing to the current hop
	for _, device := range allDevices {
		// Skip the destination device itself (we'll handle it separately)
		if device.MAC == destDevice.MAC {
			continue
		}

		// Check if this device is a neighbor of current hop
		neighborHop := b.checkDeviceAsNeighbor(device, currentHop)
		if neighborHop != nil {
			// Set the egress port on current hop (port leading to this neighbor)
			if currentHop.EgressPort == nil {
				currentHop.EgressPort = neighborHop.EgressPort
			}

			// Set ingress port on next hop
			neighborHop.IngressPort = currentHop.EgressPort

			return neighborHop
		}
	}

	// Check if we've reached the destination
	b.checkDestinationReached(currentHop, destDevice)

	return nil
}

// checkDeviceAsNeighbor checks if a device is a neighbor of the current hop.
func (b *L2PathBuilder) checkDeviceAsNeighbor(device *DiscoveredDevice, currentHop *L2Hop) *L2Hop {
	// Check LLDP neighbor
	if device.LLDPInfo != nil && b.deviceMatchesHop(device.LLDPInfo.SystemName, device.LLDPInfo.ManagementAddress, currentHop) {
		return &L2Hop{
			Device:   device.LLDPInfo.SystemName,
			DeviceIP: device.LLDPInfo.ManagementAddress,
			Source:   "lldp",
			EgressPort: &PortInfo{
				Name:        device.LLDPInfo.PortID,
				ConnectedTo: device.MAC,
			},
		}
	}

	// Check CDP neighbor
	if device.CDPInfo != nil && b.deviceMatchesHop(device.CDPInfo.DeviceID, device.CDPInfo.ManagementAddress, currentHop) {
		hop := &L2Hop{
			Device:   device.CDPInfo.DeviceID,
			DeviceIP: device.CDPInfo.ManagementAddress,
			Source:   "cdp",
			EgressPort: &PortInfo{
				Name:        device.CDPInfo.PortID,
				ConnectedTo: device.MAC,
			},
		}
		if device.CDPInfo.NativeVLAN > 0 {
			hop.EgressPort.VLANs = []int{device.CDPInfo.NativeVLAN}
		}
		return hop
	}

	// Check EDP neighbor
	if device.EDPInfo != nil {
		hopName := device.EDPInfo.DisplayName
		if hopName == "" {
			hopName = device.EDPInfo.DeviceID
		}
		// Note: EDPDeviceInfo doesn't have ManagementAddress, so we only match by name
		if b.deviceMatchesHop(hopName, "", currentHop) {
			hop := &L2Hop{
				Device: hopName,
				Source: "edp",
				EgressPort: &PortInfo{
					Name:        device.EDPInfo.PortID,
					ConnectedTo: device.MAC,
				},
			}
			if device.EDPInfo.VLAN > 0 {
				hop.EgressPort.VLANs = []int{device.EDPInfo.VLAN}
			}
			return hop
		}
	}

	return nil
}

// checkDestinationReached checks if the destination device is directly connected to current hop.
func (b *L2PathBuilder) checkDestinationReached(currentHop *L2Hop, destDevice *DiscoveredDevice) {
	// Check if destination device has LLDP/CDP pointing to current hop
	if destDevice.LLDPInfo != nil && b.deviceMatchesHop(destDevice.LLDPInfo.SystemName, destDevice.LLDPInfo.ManagementAddress, currentHop) {
		currentHop.EgressPort = &PortInfo{
			Name:        destDevice.LLDPInfo.PortID,
			ConnectedTo: destDevice.MAC,
		}
		return
	}

	if destDevice.CDPInfo != nil && b.deviceMatchesHop(destDevice.CDPInfo.DeviceID, destDevice.CDPInfo.ManagementAddress, currentHop) {
		currentHop.EgressPort = &PortInfo{
			Name:        destDevice.CDPInfo.PortID,
			ConnectedTo: destDevice.MAC,
		}
		return
	}

	if destDevice.EDPInfo != nil {
		hopName := destDevice.EDPInfo.DisplayName
		if hopName == "" {
			hopName = destDevice.EDPInfo.DeviceID
		}
		if b.deviceMatchesHop(hopName, "", currentHop) {
			currentHop.EgressPort = &PortInfo{
				Name:        destDevice.EDPInfo.PortID,
				ConnectedTo: destDevice.MAC,
			}
		}
	}
}

// deviceMatchesHop checks if a device name/IP matches the current hop.
func (b *L2PathBuilder) deviceMatchesHop(name, ip string, hop *L2Hop) bool {
	if name != "" && hop.Device != "" && strings.EqualFold(name, hop.Device) {
		return true
	}
	if ip != "" && hop.DeviceIP != "" && ip == hop.DeviceIP {
		return true
	}
	return false
}

// isDestinationReached checks if the current hop has reached the destination.
func (b *L2PathBuilder) isDestinationReached(hop *L2Hop, destDevice *DiscoveredDevice) bool {
	if hop.EgressPort != nil && hop.EgressPort.ConnectedTo == destDevice.MAC {
		return true
	}
	return false
}
