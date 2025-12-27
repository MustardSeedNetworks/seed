// Package discovery implements multi-protocol network device discovery.
// CDP (Cisco Discovery Protocol) support enables discovery of Cisco networking equipment
// and compatible devices that advertise their capabilities, platform, and management information
// on the local network segment via Ethernet frames.
package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// CDPNeighbor represents a discovered CDP neighbor.
type CDPNeighbor struct {
	DeviceID          string    `json:"deviceId"`
	PortID            string    `json:"portId"`
	Platform          string    `json:"platform"`
	SoftwareVersion   string    `json:"softwareVersion"`
	Capabilities      []string  `json:"capabilities"`
	NativeVLAN        int       `json:"nativeVlan,omitempty"`
	Duplex            string    `json:"duplex,omitempty"`
	ManagementAddress string    `json:"managementAddress,omitempty"`
	VTPDomain         string    `json:"vtpDomain,omitempty"`
	TTL               int       `json:"ttl"`
	LastSeen          time.Time `json:"lastSeen"`
	SourceMAC         string    `json:"sourceMAC"`
}

// CDPCapture handles CDP frame capture on an interface.
type CDPCapture struct {
	interfaceName string
	handle        *pcap.Handle
	neighbors     map[string]*CDPNeighbor // keyed by DeviceID+PortID
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	started       bool
}

// NewCDPCapture creates a new CDP capture instance.
// Fixes #903: Context is created in Start() to prevent leaks if Start() is never called.
func NewCDPCapture(interfaceName string) *CDPCapture {
	return &CDPCapture{
		interfaceName: interfaceName,
		neighbors:     make(map[string]*CDPNeighbor),
	}
}

// Start begins capturing CDP frames.
//
//nolint:dupl // CDP/LLDP/EDP capture Start() methods share structure but have protocol-specific filters
func (c *CDPCapture) Start() error {
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

	// Set BPF filter for CDP (dst MAC 01:00:0c:cc:cc:cc)
	if err := handle.SetBPFFilter("ether dst 01:00:0c:cc:cc:cc"); err != nil {
		handle.Close()
		c.mu.Unlock()
		return fmt.Errorf("failed to set BPF filter: %w", err)
	}

	c.handle = handle
	c.started = true
	// Fixes #903: Create context here instead of in NewCDPCapture
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.mu.Unlock()

	go c.captureLoop()
	return nil
}

// Stop stops capturing CDP frames.
func (c *CDPCapture) Stop() {
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

// GetNeighbors returns all discovered CDP neighbors.
func (c *CDPCapture) GetNeighbors() []*CDPNeighbor {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*CDPNeighbor, 0, len(c.neighbors))
	for _, n := range c.neighbors {
		// Only return neighbors seen within TTL window
		if time.Since(n.LastSeen) < time.Duration(n.TTL)*time.Second {
			result = append(result, n)
		}
	}
	return result
}

// captureLoop continuously captures and processes CDP frames.
func (c *CDPCapture) captureLoop() {
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

// processPacket extracts CDP information from a captured packet.
func (c *CDPCapture) processPacket(packet gopacket.Packet) {
	// Check for CDP layer
	cdpLayer := packet.Layer(layers.LayerTypeCiscoDiscovery)
	if cdpLayer == nil {
		return
	}

	cdp, ok := cdpLayer.(*layers.CiscoDiscovery)
	if !ok {
		return
	}

	neighbor := &CDPNeighbor{
		LastSeen: time.Now(),
		TTL:      int(cdp.TTL),
	}

	// Extract source MAC from ethernet layer
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if eth, ok := ethLayer.(*layers.Ethernet); ok {
		neighbor.SourceMAC = eth.SrcMAC.String()
	}

	// Parse CDP TLVs from the Info layer
	cdpInfoLayer := packet.Layer(layers.LayerTypeCiscoDiscoveryInfo)
	if cdpInfoLayer != nil {
		cdpInfo, ok := cdpInfoLayer.(*layers.CiscoDiscoveryInfo)
		if ok {
			neighbor.DeviceID = cdpInfo.DeviceID
			neighbor.PortID = cdpInfo.PortID
			neighbor.Platform = cdpInfo.Platform
			neighbor.SoftwareVersion = cdpInfo.Version
			neighbor.VTPDomain = cdpInfo.VTPDomain
			neighbor.NativeVLAN = int(cdpInfo.NativeVLAN)

			// Parse capabilities
			neighbor.Capabilities = parseCDPCapabilities(cdpInfo.Capabilities)

			// Parse duplex
			if cdpInfo.FullDuplex {
				neighbor.Duplex = "full"
			} else {
				neighbor.Duplex = "half"
			}

			// Parse management addresses
			if len(cdpInfo.MgmtAddresses) > 0 {
				neighbor.ManagementAddress = cdpInfo.MgmtAddresses[0].String()
			}
		}
	}

	// Store neighbor (keyed by DeviceID + PortID)
	key := neighbor.DeviceID + ":" + neighbor.PortID
	c.mu.Lock()
	c.neighbors[key] = neighbor
	c.mu.Unlock()
}

// parseCDPCapabilities converts CDP capability struct to readable strings.
func parseCDPCapabilities(caps layers.CDPCapabilities) []string {
	var result []string

	if caps.L3Router {
		result = append(result, "Router")
	}
	if caps.TBBridge {
		result = append(result, "Bridge")
	}
	if caps.SPBridge {
		result = append(result, "Source Route Bridge")
	}
	if caps.L2Switch {
		result = append(result, "Switch")
	}
	if caps.IsHost {
		result = append(result, "Host")
	}
	if caps.IGMPFilter {
		result = append(result, "IGMP Filter")
	}
	if caps.L1Repeater {
		result = append(result, "Repeater")
	}
	if caps.IsPhone {
		result = append(result, "Phone")
	}

	return result
}

// IsRunning returns true if capture is active.
func (c *CDPCapture) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}
