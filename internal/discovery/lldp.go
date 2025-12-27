// Package discovery handles network discovery protocols (LLDP, CDP, EDP).
package discovery

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

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
}

// LLDPCapture handles LLDP frame capture on an interface.
type LLDPCapture struct {
	interfaceName string
	handle        *pcap.Handle
	neighbors     map[string]*LLDPNeighbor // keyed by ChassisID+PortID
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	started       bool
}

// NewLLDPCapture creates a new LLDP capture instance.
// Fixes #903: Context is created in Start() to prevent leaks if Start() is never called.
func NewLLDPCapture(interfaceName string) *LLDPCapture {
	return &LLDPCapture{
		interfaceName: interfaceName,
		neighbors:     make(map[string]*LLDPNeighbor),
	}
}

// Start begins capturing LLDP frames.
//
//nolint:dupl // CDP/LLDP/EDP capture Start() methods share structure but have protocol-specific filters
func (c *LLDPCapture) Start() error {
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

	// Set BPF filter for LLDP (EtherType 0x88cc)
	if err := handle.SetBPFFilter("ether proto 0x88cc"); err != nil {
		handle.Close()
		c.mu.Unlock()
		return fmt.Errorf("failed to set BPF filter: %w", err)
	}

	c.handle = handle
	c.started = true
	// Fixes #903: Create context here instead of in NewLLDPCapture
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.mu.Unlock()

	go c.captureLoop()
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
func (c *LLDPCapture) captureLoop() {
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

// processPacket extracts LLDP information from a captured packet.
func (c *LLDPCapture) processPacket(packet gopacket.Packet) {
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
	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		neighbor.SourceMAC = eth.SrcMAC.String()
	}

	// Parse Chassis ID
	neighbor.ChassisID = string(lldp.ChassisID.ID)
	neighbor.ChassisIDType = lldp.ChassisID.Subtype.String()

	// Parse Port ID
	neighbor.PortID = string(lldp.PortID.ID)
	neighbor.PortIDType = lldp.PortID.Subtype.String()

	// Parse TTL
	neighbor.TTL = int(lldp.TTL)

	// Parse LLDP Info layer for optional TLVs
	lldpInfoLayer := packet.Layer(layers.LayerTypeLinkLayerDiscoveryInfo)
	if lldpInfoLayer != nil {
		lldpInfo, ok := lldpInfoLayer.(*layers.LinkLayerDiscoveryInfo)
		if ok {
			// Port Description
			neighbor.PortDescription = lldpInfo.PortDescription

			// System Name
			neighbor.SystemName = lldpInfo.SysName

			// System Description
			neighbor.SystemDescription = lldpInfo.SysDescription

			// System Capabilities
			neighbor.SystemCapabilities = parseSystemCapabilities(lldpInfo.SysCapabilities.SystemCap)

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
