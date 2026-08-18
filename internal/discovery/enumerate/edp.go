package enumerate

// EDP (Extreme Discovery Protocol) support enables discovery of Extreme Networks equipment
// and compatible devices that advertise their device ID, port information, VLAN membership,
// and IP addressing via Ethernet frames.

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
)

// pcapSnapshotLengthEDP is the maximum number of bytes to capture from each packet.
// 65535 is the maximum possible packet size for IP.
const pcapSnapshotLengthEDP = 65535

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

// EDP protocol parsing constants.
const (
	edpHeaderSize       = 16   // EDP header: version..machine ID inclusive
	edpMachineIDOffset  = 10   // machine ID (MAC) offset within the header
	edpVersion          = 0x01 // EDP protocol version this parser accepts
	edpTLVHeaderSize    = 4    // TLV header size (marker + type + length)
	edpTLVMarker        = 0x99 // TLV marker byte
	edpInfoSlotPortSize = 4    // Slot + port field size
	edpInfoVLANOffset   = 8    // Offset to VLAN in info TLV
	edpVLANIDSize       = 2    // VLAN ID field size
	edpVLANNameOffset   = 4    // Offset to VLAN name
	edpIPAddrSize       = 4    // IPv4 address size
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
	// ObservedVLAN is the 802.1Q VLAN the advertisement arrived on, or
	// VLANUntagged when the frame carried no tag. On a trunk port this is the
	// only way to tell which VLAN a neighbour advertises into.
	ObservedVLAN uint16 `json:"observedVlan,omitempty"`
	// MachineID is the MAC from the EDP header, used as the device identity
	// when the advertisement carries no Display TLV.
	MachineID string `json:"machineId,omitempty"`
}

// EDPCapture handles EDP frame capture on an interface.
type EDPCapture struct {
	interfaceName string
	opener        capture.Opener
	handle        capture.Handle
	neighbors     map[string]*EDPNeighbor // keyed by DeviceID+PortID
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	started       bool
}

// NewEDPCapture creates a new EDP capture instance bound to the given capture
// Opener (the libpcap adapter in production, a no-op under CGO_ENABLED=0).
// Fixes #903: Context is created in Start() to prevent leaks if Start() is never called.
func NewEDPCapture(opener capture.Opener, interfaceName string) *EDPCapture {
	return &EDPCapture{
		interfaceName: interfaceName,
		opener:        opener,
		neighbors:     make(map[string]*EDPNeighbor),
	}
}

// Start begins capturing EDP frames on the bound interface and returns once
// the pcap handle is open. Capture runs in a background goroutine; call Stop
// to end it. Calling Start again while already started is a no-op.
func (c *EDPCapture) Start() error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}

	// EDP frames are sent to the well-known dst MAC 00:e0:2b:00:00:00.
	handle, linkType, err := openProtocolCapture(
		c.opener, c.interfaceName, pcapSnapshotLengthEDP, "ether dst 00:e0:2b:00:00:00",
	)
	if err != nil {
		c.mu.Unlock()
		return err
	}

	c.handle = handle
	c.started = true
	// Fixes #903: Create context here instead of in NewEDPCapture
	c.ctx, c.cancel = context.WithCancel(context.Background())
	ctx := c.ctx
	c.mu.Unlock()

	go c.captureLoop(ctx, handle, linkType)
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
func (c *EDPCapture) captureLoop(ctx context.Context, handle capture.Handle, linkType layers.LinkType) {
	if handle == nil {
		return
	}

	packets := readTaggedPackets(handle, linkType)

	for {
		select {
		case <-ctx.Done():
			return
		case tp, ok := <-packets:
			if !ok {
				return
			}
			c.processPacket(tp.Packet, tp.VLAN)
		}
	}
}

// processPacket extracts EDP information from a captured packet.
func (c *EDPCapture) processPacket(packet gopacket.Packet, vlan uint16) {
	neighbor := &EDPNeighbor{
		LastSeen: time.Now(),
	}

	// Extract source MAC from ethernet layer
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		neighbor.SourceMAC = eth.SrcMAC.String()
	}
	neighbor.ObservedVLAN = vlan

	// EDP rides on 802.3 + LLC/SNAP (OUI 00:E0:2B, PID 0x00BB), so the EDP
	// header starts at the SNAP payload. gopacket decodes the SNAP layer itself;
	// falling back to the LLC payload would leave the 5-byte SNAP header in
	// front of the version byte.
	snapLayer := packet.Layer(layers.LayerTypeSNAP)
	if snapLayer == nil {
		return
	}
	payload := snapLayer.LayerPayload()
	if len(payload) < edpHeaderSize {
		return
	}

	// EDP header, 16 bytes:
	//   0      version (1)
	//   1      reserved (1)
	//   2-3    length (2)
	//   4-5    checksum (2)
	//   6-7    sequence (2)
	//   8-9    machine ID type (2)
	//   10-15  machine ID (6, a MAC when the type is 0)
	// then 0x99-marker TLVs.
	if payload[0] != edpVersion {
		return
	}

	neighbor.MachineID = net.HardwareAddr(payload[edpMachineIDOffset:edpHeaderSize]).String()

	c.parseEDPTLVs(payload[edpHeaderSize:], neighbor)

	// The Display TLV carries the operator-facing name; the header only has a
	// MAC. Fall back to that MAC so a neighbour without a Display TLV is still
	// identifiable rather than keyed on an empty string.
	neighbor.DeviceID = neighbor.DisplayName
	if neighbor.DeviceID == "" {
		neighbor.DeviceID = neighbor.MachineID
	}

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

	for offset+edpTLVHeaderSize <= len(data) {
		// TLV header: 1 byte marker (0x99), 1 byte type, 2 bytes length
		if data[offset] != edpTLVMarker {
			// Every EDP TLV is marker-prefixed. A byte here that is not 0x99
			// means the walk has derailed, so stop rather than guess at an
			// alternative framing.
			break
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
		if len(data) >= edpInfoSlotPortSize {
			// First 2 bytes: slot
			// Next 2 bytes: port
			slot := binary.BigEndian.Uint16(data[0:2])
			port := binary.BigEndian.Uint16(data[2:4])
			neighbor.PortID = fmt.Sprintf("%d:%d", slot, port)
		}
		// Additional info may follow
		if len(data) >= edpInfoVLANOffset {
			neighbor.VLAN = int(binary.BigEndian.Uint16(data[6:8]))
		}
	case EDPTLVVlan:
		// VLAN TLV
		if len(data) >= edpVLANIDSize {
			neighbor.VLAN = int(binary.BigEndian.Uint16(data[0:2]))
		}
		// VLAN name may follow
		if len(data) > edpVLANNameOffset {
			neighbor.VLANName = trimNull(string(data[edpVLANNameOffset:]))
		}
	case EDPTLVIPAddr:
		// IP Address TLV
		if len(data) >= edpIPAddrSize {
			neighbor.ManagementAddress = net.IP(data[0:edpIPAddrSize]).String()
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
