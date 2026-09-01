package enumerate

import (
	"context"
	"encoding/hex"
	"net"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
)

// pcapSnapshotLengthLLDP is the maximum number of bytes to capture from each packet.
// 65535 is the maximum possible packet size for IP.
const pcapSnapshotLengthLLDP = 65535

// LLDPNeighbor represents a discovered LLDP neighbor.
type LLDPNeighbor struct {
	ChassisID          string            `json:"chassisId"`
	ChassisIDType      string            `json:"chassisIdType"`
	PortID             string            `json:"portId"`
	PortIDType         string            `json:"portIdType"`
	PortDescription    string            `json:"portDescription"`
	SystemName         string            `json:"systemName"`
	SystemDescription  string            `json:"systemDescription"`
	SystemCapabilities []string          `json:"systemCapabilities"`
	ManagementAddress  string            `json:"managementAddress"`
	TTL                int               `json:"ttl"`
	LastSeen           time.Time         `json:"lastSeen"`
	SourceMAC          string            `json:"sourceMAC"`
	CustomTLVs         map[string]string `json:"customTLVs,omitempty"`
	// ObservedVLAN is the 802.1Q VLAN the advertisement arrived on, or
	// VLANUntagged when the frame carried no tag. On a trunk port this is the
	// only way to tell which VLAN a neighbour advertises into.
	ObservedVLAN uint16 `json:"observedVlan,omitempty"`
}

// LLDPCapture handles LLDP frame capture on an interface.
type LLDPCapture struct {
	interfaceName string
	opener        capture.Opener
	handle        capture.Handle
	neighbors     map[string]*LLDPNeighbor // keyed by ChassisID+PortID
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	started       bool
}

// NewLLDPCapture creates a new LLDP capture instance bound to the given capture
// Opener (the libpcap adapter in production, a no-op under CGO_ENABLED=0).
// Fixes #903: Context is created in Start() to prevent leaks if Start() is never called.
func NewLLDPCapture(opener capture.Opener, interfaceName string) *LLDPCapture {
	return &LLDPCapture{
		interfaceName: interfaceName,
		opener:        opener,
		neighbors:     make(map[string]*LLDPNeighbor),
	}
}

// Start begins capturing LLDP frames on the bound interface and returns once
// the pcap handle is open. Capture runs in a background goroutine; call Stop
// to end it. Calling Start again while already started is a no-op.
func (c *LLDPCapture) Start() error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}

	// LLDP frames carry EtherType 0x88cc. On a trunk that type field sits behind
	// an 802.1Q tag, where `ether proto` — a bare load at offset 12 — cannot see
	// it, so the second branch matches the tagged form. The untagged branch comes
	// first because the `vlan` keyword shifts the offsets of everything after it.
	// CDP and EDP need no equivalent: they filter on destination MAC, which a tag
	// does not move.
	handle, linkType, err := openProtocolCapture(
		c.opener, c.interfaceName, pcapSnapshotLengthLLDP,
		"ether proto 0x88cc or (vlan and ether proto 0x88cc)",
	)
	if err != nil {
		c.mu.Unlock()
		return err
	}

	c.handle = handle
	c.started = true
	// Fixes #903: Create context here instead of in NewLLDPCapture
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.mu.Unlock()

	go c.captureLoop(c.ctx, handle, linkType)
	return nil
}

// Stop stops capturing LLDP frames.
func (c *LLDPCapture) Stop() {
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

// GetNeighbors returns all discovered LLDP neighbors.
func (c *LLDPCapture) GetNeighbors() []*LLDPNeighbor {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*LLDPNeighbor, 0, len(c.neighbors))
	for _, n := range c.neighbors {
		// Only return neighbors seen within TTL window
		if time.Since(n.LastSeen) < time.Duration(n.TTL)*time.Second {
			result = append(result, n)
		}
	}
	return result
}

// captureLoop continuously captures and processes LLDP frames.
func (c *LLDPCapture) captureLoop(ctx context.Context, handle capture.Handle, linkType layers.LinkType) {
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

// processPacket extracts LLDP information from a captured packet.
func (c *LLDPCapture) processPacket(packet gopacket.Packet, vlan uint16) {
	lldpLayer := packet.Layer(layers.LayerTypeLinkLayerDiscovery)
	if lldpLayer == nil {
		return
	}

	lldp, ok := lldpLayer.(*layers.LinkLayerDiscovery)
	if !ok {
		return
	}

	neighbor := &LLDPNeighbor{
		LastSeen:   time.Now(),
		CustomTLVs: make(map[string]string),
	}

	// Extract source MAC from ethernet layer
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if eth, ethOK := ethLayer.(*layers.Ethernet); ethOK {
		neighbor.SourceMAC = eth.SrcMAC.String()
	}
	neighbor.ObservedVLAN = vlan

	// Parse Chassis ID
	neighbor.ChassisID = formatLLDPChassisID(lldp.ChassisID)
	neighbor.ChassisIDType = lldp.ChassisID.Subtype.String()

	// Parse Port ID
	neighbor.PortID = formatLLDPPortID(lldp.PortID)
	neighbor.PortIDType = lldp.PortID.Subtype.String()

	// Parse TTL
	neighbor.TTL = int(lldp.TTL)

	// Parse LLDP Info layer for optional TLVs
	lldpInfoLayer := packet.Layer(layers.LayerTypeLinkLayerDiscoveryInfo)
	if lldpInfoLayer != nil {
		lldpInfo, infoOK := lldpInfoLayer.(*layers.LinkLayerDiscoveryInfo)
		if infoOK {
			// Port Description
			neighbor.PortDescription = lldpInfo.PortDescription

			// System Name
			neighbor.SystemName = lldpInfo.SysName

			// System Description
			neighbor.SystemDescription = lldpInfo.SysDescription

			// System Capabilities
			neighbor.SystemCapabilities = parseSystemCapabilities(
				lldpInfo.SysCapabilities.SystemCap,
			)

			// Management Address
			if len(lldpInfo.MgmtAddress.Address) > 0 {
				neighbor.ManagementAddress = net.IP(lldpInfo.MgmtAddress.Address).String()
			}
		}
	}

	// Store neighbor (keyed by ChassisID + PortID)
	key := neighbor.ChassisID + ":" + neighbor.PortID
	c.mu.Lock()
	c.neighbors[key] = neighbor
	c.mu.Unlock()
}

// parseSystemCapabilities converts capability struct to readable strings.
func parseSystemCapabilities(caps layers.LLDPCapabilities) []string {
	var result []string

	if caps.Other {
		result = append(result, "Other")
	}
	if caps.Repeater {
		result = append(result, "Repeater")
	}
	if caps.Bridge {
		result = append(result, "Bridge")
	}
	if caps.WLANAP {
		result = append(result, "WLAN AP")
	}
	if caps.Router {
		result = append(result, "Router")
	}
	if caps.Phone {
		result = append(result, "Phone")
	}
	if caps.DocSis {
		result = append(result, "DOCSIS")
	}
	if caps.StationOnly {
		result = append(result, "Station")
	}
	if caps.CVLAN {
		result = append(result, "C-VLAN")
	}
	if caps.SVLAN {
		result = append(result, "S-VLAN")
	}

	return result
}

// IsRunning returns true if capture is active.
func (c *LLDPCapture) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// LLDP identifiers are only text for some subtypes. For macAddress and
// networkAddress the TLV carries raw bytes, and casting those to a string put
// unprintable data straight into the API responses (#1932). The subtype is
// decoded already, so the formatting below reads it rather than guessing from
// the bytes.
const (
	// lldpMACLen and lldpEUI64Len are the two link-layer address widths a
	// macAddress subtype may carry.
	lldpMACLen   = 6
	lldpEUI64Len = 8

	// An IANA address-family byte prefixes a networkAddress value.
	lldpAddrFamilyIPv4 = 1
	lldpAddrFamilyIPv6 = 2
	lldpIPv4Len        = 4
	lldpIPv6Len        = 16
)

// formatLLDPChassisID renders a chassis ID according to its subtype.
func formatLLDPChassisID(id layers.LLDPChassisID) string {
	switch id.Subtype {
	case layers.LLDPChassisIDSubTypeMACAddr:
		return formatLLDPMAC(id.ID)
	case layers.LLDPChassisIDSubTypeNetworkAddr:
		return formatLLDPNetworkAddr(id.ID)
	case layers.LLDPChassisIDSubTypeChassisComp,
		layers.LLDPChassisIDSubtypeIfaceAlias,
		layers.LLDPChassisIDSubTypePortComp,
		layers.LLDPChassisIDSubtypeIfaceName,
		layers.LLDPChassisIDSubTypeLocal,
		layers.LLDPChassisIDSubTypeReserved:
		return formatLLDPText(id.ID)
	default:
		return formatLLDPText(id.ID)
	}
}

// formatLLDPPortID renders a port ID according to its subtype.
func formatLLDPPortID(id layers.LLDPPortID) string {
	switch id.Subtype {
	case layers.LLDPPortIDSubtypeMACAddr:
		return formatLLDPMAC(id.ID)
	case layers.LLDPPortIDSubtypeNetworkAddr:
		return formatLLDPNetworkAddr(id.ID)
	case layers.LLDPPortIDSubtypeIfaceAlias,
		layers.LLDPPortIDSubtypePortComp,
		layers.LLDPPortIDSubtypeIfaceName,
		layers.LLDPPortIDSubtypeAgentCircuitID,
		layers.LLDPPortIDSubtypeLocal,
		layers.LLDPPortIDSubtypeReserved:
		return formatLLDPText(id.ID)
	default:
		return formatLLDPText(id.ID)
	}
}

// formatLLDPMAC renders a link-layer address the way the rest of this package
// does. A value that is not an address width falls back to hex rather than
// being dropped: an identifier we cannot name is still an identifier.
func formatLLDPMAC(raw []byte) string {
	if len(raw) == lldpMACLen || len(raw) == lldpEUI64Len {
		return net.HardwareAddr(raw).String()
	}

	return hex.EncodeToString(raw)
}

// formatLLDPNetworkAddr renders a networkAddress value, which IEEE 802.1AB
// prefixes with an IANA address-family byte.
func formatLLDPNetworkAddr(raw []byte) string {
	if len(raw) < 1 {
		return ""
	}

	addr := raw[1:]
	switch {
	case raw[0] == lldpAddrFamilyIPv4 && len(addr) == lldpIPv4Len,
		raw[0] == lldpAddrFamilyIPv6 && len(addr) == lldpIPv6Len:
		return net.IP(addr).String()
	default:
		return hex.EncodeToString(raw)
	}
}

// formatLLDPText renders a subtype whose value is text. A device that
// advertises a text subtype with bytes that are not text still exists, and hex
// beats the replacement characters JSON encoding would otherwise produce.
func formatLLDPText(raw []byte) string {
	if !utf8.Valid(raw) {
		return hex.EncodeToString(raw)
	}

	return string(raw)
}
