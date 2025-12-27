// Package discovery implements multi-protocol network device discovery.
// EDP (Extreme Discovery Protocol) support enables discovery of Extreme Networks equipment
// and compatible devices that advertise their device ID, port information, VLAN membership,
// and IP addressing via Ethernet frames.
package discovery

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// EDP TLV Types (Extreme Discovery Protocol).
const (
	EDPTLVNull    uint8 = 0x00
	EDPTLVDisplay uint8 = 0x01
	EDPTLVInfo    uint8 = 0x02
	EDPTLVVlan    uint8 = 0x05
	EDPTLVESRP    uint8 = 0x06
	EDPTLVUnknown uint8 = 0x07
	EDPTLVIPAddr  uint8 = 0x99
)

// EDPNeighbor represents a discovered EDP neighbor.
type EDPNeighbor struct {
	DeviceID          string    `json:"deviceId"`
	PortID            string    `json:"portId"`
	DisplayName       string    `json:"displayName,omitempty"`
	SoftwareVersion   string    `json:"softwareVersion,omitempty"`
	Platform          string    `json:"platform,omitempty"`
	ManagementAddress string    `json:"managementAddress,omitempty"`
	VLAN              int       `json:"vlan,omitempty"`
	VLANName          string    `json:"vlanName,omitempty"`
	TTL               int       `json:"ttl"`
	LastSeen          time.Time `json:"lastSeen"`
	SourceMAC         string    `json:"sourceMAC"`
}

// EDPCapture handles EDP frame capture on an interface.
type EDPCapture struct {
	interfaceName string
	handle        *pcap.Handle
	neighbors     map[string]*EDPNeighbor // keyed by DeviceID+PortID
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	started       bool
}

// NewEDPCapture creates a new EDP capture instance.
// Fixes #903: Context is created in Start() to prevent leaks if Start() is never called.
func NewEDPCapture(interfaceName string) *EDPCapture {
	return &EDPCapture{
		interfaceName: interfaceName,
		neighbors:     make(map[string]*EDPNeighbor),
	}
}

// Start begins capturing EDP frames.
//
//nolint:dupl // CDP/LLDP/EDP capture Start() methods share structure but have protocol-specific filters
func (c *EDPCapture) Start() error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}

	// Open capture handle
	handle, err := pcap.OpenLive(c.interfaceName, 65535, true, pcap.BlockForever)
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to open capture: %w", err)
	}

	// Set BPF filter for EDP (dst MAC 00:E0:2B:00:00:00)
	if err := handle.SetBPFFilter("ether dst 00:e0:2b:00:00:00"); err != nil {
		handle.Close()
		c.mu.Unlock()
		return fmt.Errorf("failed to set BPF filter: %w", err)
	}

	c.handle = handle
	c.started = true
	// Fixes #903: Create context here instead of in NewEDPCapture
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.mu.Unlock()

	go c.captureLoop()
	return nil
}

// Stop stops capturing EDP frames.
func (c *EDPCapture) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Fixes #903: Check cancel is not nil (Start() may not have been called)
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}

	if c.handle != nil {
		c.handle.Close()
		c.handle = nil
	}
	c.started = false
}

// GetNeighbors returns all discovered EDP neighbors.
func (c *EDPCapture) GetNeighbors() []*EDPNeighbor {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*EDPNeighbor, 0, len(c.neighbors))
	for _, n := range c.neighbors {
		// Only return neighbors seen within TTL window (default 180s for EDP)
		ttl := n.TTL
		if ttl == 0 {
			ttl = 180 // Default EDP TTL
		}
		if time.Since(n.LastSeen) < time.Duration(ttl)*time.Second {
			result = append(result, n)
		}
	}
	return result
}

// captureLoop continuously captures and processes EDP frames.
func (c *EDPCapture) captureLoop() {
	packetSource := gopacket.NewPacketSource(c.handle, c.handle.LinkType())

	for {
		select {
		case <-c.ctx.Done():
			return
		case packet, ok := <-packetSource.Packets():
			if !ok {
				return
			}
			c.processPacket(packet)
		}
	}
}

// processPacket extracts EDP information from a captured packet.
func (c *EDPCapture) processPacket(packet gopacket.Packet) {
	neighbor := &EDPNeighbor{
		LastSeen: time.Now(),
	}

	// Extract source MAC from ethernet layer
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		neighbor.SourceMAC = eth.SrcMAC.String()
	}

	// EDP uses LLC/SNAP encapsulation
	// We need to parse the payload manually
	llcLayer := packet.Layer(layers.LayerTypeLLC)
	if llcLayer == nil {
		return
	}

	// Get the payload after LLC header
	payload := llcLayer.LayerPayload()
	if len(payload) < 8 {
		return
	}

	// EDP header structure:
	// Bytes 0-1: Protocol version (0x0001)
	// Bytes 2-3: Reserved
	// Bytes 4-5: Sequence number
	// Bytes 6-7: Machine ID length
	// Then: Machine ID, followed by TLVs

	// Skip SNAP header if present (5 bytes: 3 OUI + 2 protocol ID)
	if len(payload) > 5 && payload[0] == 0x00 && payload[1] == 0xe0 && payload[2] == 0x2b {
		payload = payload[5:]
	}

	// Parse EDP header
	if len(payload) < 8 {
		return
	}

	// Check version (should be 1)
	version := binary.BigEndian.Uint16(payload[0:2])
	if version != 1 {
		return
	}

	machineIDLen := binary.BigEndian.Uint16(payload[6:8])
	if len(payload) < 8+int(machineIDLen) {
		return
	}

	neighbor.DeviceID = string(payload[8 : 8+machineIDLen])

	// Parse TLVs starting after machine ID
	tlvOffset := 8 + int(machineIDLen)
	c.parseEDPTLVs(payload[tlvOffset:], neighbor)

	// Store neighbor (keyed by DeviceID + SourceMAC if no PortID)
	key := neighbor.DeviceID + ":" + neighbor.SourceMAC
	if neighbor.PortID != "" {
		key = neighbor.DeviceID + ":" + neighbor.PortID
	}
	c.mu.Lock()
	c.neighbors[key] = neighbor
	c.mu.Unlock()
}

// parseEDPTLVs parses EDP TLV data.
func (c *EDPCapture) parseEDPTLVs(data []byte, neighbor *EDPNeighbor) {
	offset := 0

	for offset+4 <= len(data) {
		// TLV header: 1 byte marker (0x99), 1 byte type, 2 bytes length
		if data[offset] != 0x99 {
			// Try alternative format: 1 byte type, 1 byte reserved, 2 bytes length
			tlvType := data[offset]
			tlvLen := binary.BigEndian.Uint16(data[offset+2 : offset+4])

			if tlvLen < 4 || offset+int(tlvLen) > len(data) {
				break
			}

			tlvData := data[offset+4 : offset+int(tlvLen)]
			c.parseEDPTLV(tlvType, tlvData, neighbor)
			offset += int(tlvLen)
			continue
		}

		// Standard format with 0x99 marker
		tlvType := data[offset+1]
		tlvLen := binary.BigEndian.Uint16(data[offset+2 : offset+4])

		if tlvLen < 4 || offset+int(tlvLen) > len(data) {
			break
		}

		tlvData := data[offset+4 : offset+int(tlvLen)]
		c.parseEDPTLV(tlvType, tlvData, neighbor)
		offset += int(tlvLen)
	}
}

// parseEDPTLV parses a single EDP TLV.
func (c *EDPCapture) parseEDPTLV(tlvType uint8, data []byte, neighbor *EDPNeighbor) {
	switch tlvType {
	case EDPTLVNull:
		// End of TLVs
		return
	case EDPTLVDisplay:
		// Display string (device name)
		if len(data) > 0 {
			neighbor.DisplayName = trimNull(string(data))
		}
	case EDPTLVInfo:
		// Device info TLV
		// Contains: slot, port, vlan info, and more
		if len(data) >= 4 {
			// First 2 bytes: slot
			// Next 2 bytes: port
			slot := binary.BigEndian.Uint16(data[0:2])
			port := binary.BigEndian.Uint16(data[2:4])
			neighbor.PortID = fmt.Sprintf("%d:%d", slot, port)
		}
		// Additional info may follow
		if len(data) >= 8 {
			neighbor.VLAN = int(binary.BigEndian.Uint16(data[6:8]))
		}
	case EDPTLVVlan:
		// VLAN TLV
		if len(data) >= 2 {
			neighbor.VLAN = int(binary.BigEndian.Uint16(data[0:2]))
		}
		// VLAN name may follow
		if len(data) > 4 {
			neighbor.VLANName = trimNull(string(data[4:]))
		}
	case EDPTLVIPAddr:
		// IP Address TLV
		if len(data) >= 4 {
			neighbor.ManagementAddress = net.IP(data[0:4]).String()
		}
	}
}

// trimNull removes null bytes from the end of a string.
func trimNull(s string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] != 0 {
			return s[:i+1]
		}
	}
	return ""
}

// IsRunning returns true if capture is active.
func (c *EDPCapture) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}
