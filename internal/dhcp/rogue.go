// Package dhcp provides DHCP monitoring including rogue DHCP server detection.
package dhcp

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// RogueServer represents a detected rogue DHCP server.
type RogueServer struct {
	IP           string    `json:"ip"`
	MAC          string    `json:"mac"`
	FirstSeen    time.Time `json:"firstSeen"`
	LastSeen     time.Time `json:"lastSeen"`
	OfferCount   int       `json:"offerCount"`
	IsAuthorized bool      `json:"isAuthorized"` // false for rogue servers
}

// RogueDetectorConfig holds configuration for rogue DHCP detection.
type RogueDetectorConfig struct {
	Interface        string   // Network interface to monitor
	KnownServers     []string // List of authorized DHCP server IPs
	AlertOnDetection bool     // Whether to log alerts for rogue servers
}

// RogueDetector monitors for unauthorized DHCP servers on the network.
type RogueDetector struct {
	mu              sync.RWMutex
	config          *RogueDetectorConfig
	running         bool
	handle          *pcap.Handle
	cancel          context.CancelFunc
	detectedServers map[string]*RogueServer // key is IP address
	knownServerSet  map[string]bool         // for fast lookup
}

// NewRogueDetector creates a new rogue DHCP server detector.
// The Interface field in config must be set - no hardcoded defaults are used (#572).
func NewRogueDetector(config *RogueDetectorConfig) *RogueDetector {
	if config == nil {
		config = &RogueDetectorConfig{
			Interface:        "", // Must be set by caller - no hardcoded defaults
			KnownServers:     []string{},
			AlertOnDetection: true,
		}
	}

	// Build known server set for fast lookup
	knownSet := make(map[string]bool)
	for _, server := range config.KnownServers {
		knownSet[server] = true
	}

	return &RogueDetector{
		config:          config,
		detectedServers: make(map[string]*RogueServer),
		knownServerSet:  knownSet,
	}
}

// Start begins monitoring for DHCP OFFER packets.
// Requires CAP_NET_RAW capability or root privileges.
func (rd *RogueDetector) Start() error {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	if rd.running {
		return fmt.Errorf("rogue detector already running")
	}

	// Open pcap handle for DHCP traffic (UDP ports 67/68)
	// Use snapshot length of 1024 bytes (enough for DHCP packets)
	// Set promiscuous mode to false (we only need broadcast packets)
	// Set timeout to 100ms for responsive shutdown
	handle, err := pcap.OpenLive(rd.config.Interface, 1024, false, 100*time.Millisecond)
	if err != nil {
		return fmt.Errorf("failed to open pcap: %w (requires CAP_NET_RAW or root)", err)
	}

	// Set BPF filter to capture only DHCP OFFER packets
	// Port 67 is DHCP server, port 68 is DHCP client
	// We want to see OFFER packets (server -> client on port 68)
	filter := "udp and (port 67 or port 68)"
	if err := handle.SetBPFFilter(filter); err != nil {
		handle.Close()
		return fmt.Errorf("failed to set BPF filter: %w", err)
	}

	rd.handle = handle
	rd.running = true

	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	rd.cancel = cancel

	// Start packet capture goroutine
	go rd.capturePackets(ctx)

	slog.Info("Rogue DHCP detector started", "interface", rd.config.Interface)
	return nil
}

// Stop stops the rogue detector.
func (rd *RogueDetector) Stop() error {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	if !rd.running {
		return nil
	}

	// Cancel the capture goroutine
	if rd.cancel != nil {
		rd.cancel()
	}

	// Close pcap handle
	if rd.handle != nil {
		rd.handle.Close()
	}

	rd.running = false
	rd.handle = nil
	rd.cancel = nil

	slog.Info("Rogue DHCP detector stopped")
	return nil
}

// IsRunning returns whether the detector is currently running.
func (rd *RogueDetector) IsRunning() bool {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	return rd.running
}

// capturePackets is the main packet capture loop.
func (rd *RogueDetector) capturePackets(ctx context.Context) {
	packetSource := gopacket.NewPacketSource(rd.handle, rd.handle.LinkType())

	for {
		select {
		case <-ctx.Done():
			return
		case packet, ok := <-packetSource.Packets():
			if !ok {
				return
			}
			rd.processPacket(packet)
		}
	}
}

// processPacket analyzes a captured packet for DHCP OFFER messages.
func (rd *RogueDetector) processPacket(packet gopacket.Packet) {
	// Extract DHCP layer
	dhcpLayer := packet.Layer(layers.LayerTypeDHCPv4)
	if dhcpLayer == nil {
		return
	}

	dhcp, ok := dhcpLayer.(*layers.DHCPv4)
	if !ok {
		return
	}

	// We only care about DHCP OFFER packets
	if dhcp.Operation != layers.DHCPOpReply {
		return
	}

	// Check if it's an OFFER (MessageType option = 2)
	messageType := rd.getDHCPMessageType(dhcp)
	if messageType != layers.DHCPMsgTypeOffer {
		return
	}

	// Extract server identifier from DHCP options
	serverIP := rd.getServerIdentifier(dhcp)
	if serverIP == "" {
		// Fallback to source IP if server identifier not found
		if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
			ip, ok := ipLayer.(*layers.IPv4)
			if !ok {
				return
			}
			serverIP = ip.SrcIP.String()
		} else {
			return
		}
	}

	// Extract source MAC address
	serverMAC := ""
	if ethLayer := packet.Layer(layers.LayerTypeEthernet); ethLayer != nil {
		eth, ok := ethLayer.(*layers.Ethernet)
		if ok {
			serverMAC = eth.SrcMAC.String()
		}
	}

	// Check if this is a known/authorized server
	rd.mu.Lock()
	defer rd.mu.Unlock()

	isKnown := rd.knownServerSet[serverIP]
	now := time.Now()

	server, exists := rd.detectedServers[serverIP]
	if !exists {
		// New server detected
		server = &RogueServer{
			IP:           serverIP,
			MAC:          serverMAC,
			FirstSeen:    now,
			LastSeen:     now,
			OfferCount:   1,
			IsAuthorized: isKnown,
		}
		rd.detectedServers[serverIP] = server

		// Alert if it's a rogue server
		if !isKnown && rd.config.AlertOnDetection {
			slog.Warn("Rogue DHCP server detected", "ip", serverIP, "mac", serverMAC)
		}
	} else {
		// Update existing server
		server.LastSeen = now
		server.OfferCount++
		if serverMAC != "" && server.MAC == "" {
			server.MAC = serverMAC
		}
	}
}

// getDHCPMessageType extracts the DHCP message type from options.
func (rd *RogueDetector) getDHCPMessageType(dhcp *layers.DHCPv4) layers.DHCPMsgType {
	for _, option := range dhcp.Options {
		if option.Type == layers.DHCPOptMessageType && len(option.Data) == 1 {
			return layers.DHCPMsgType(option.Data[0])
		}
	}
	return 0
}

// getServerIdentifier extracts the DHCP server identifier from options.
func (rd *RogueDetector) getServerIdentifier(dhcp *layers.DHCPv4) string {
	for _, option := range dhcp.Options {
		if option.Type == layers.DHCPOptServerID && len(option.Data) == 4 {
			return net.IP(option.Data).String()
		}
	}
	return ""
}

// GetDetectedServers returns all detected DHCP servers.
func (rd *RogueDetector) GetDetectedServers() []*RogueServer {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	servers := make([]*RogueServer, 0, len(rd.detectedServers))
	for _, server := range rd.detectedServers {
		// Create a copy to avoid race conditions
		serverCopy := *server
		servers = append(servers, &serverCopy)
	}

	return servers
}

// GetRogueServers returns only unauthorized (rogue) DHCP servers.
func (rd *RogueDetector) GetRogueServers() []*RogueServer {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	rogues := make([]*RogueServer, 0)
	for _, server := range rd.detectedServers {
		if !server.IsAuthorized {
			serverCopy := *server
			rogues = append(rogues, &serverCopy)
		}
	}

	return rogues
}

// UpdateKnownServers updates the list of authorized DHCP servers.
func (rd *RogueDetector) UpdateKnownServers(servers []string) {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	// Update known server set
	rd.knownServerSet = make(map[string]bool)
	for _, server := range servers {
		rd.knownServerSet[server] = true
	}

	// Update authorization status of detected servers
	for _, server := range rd.detectedServers {
		server.IsAuthorized = rd.knownServerSet[server.IP]
	}

	rd.config.KnownServers = servers
}

// ClearDetectedServers clears the list of detected servers.
func (rd *RogueDetector) ClearDetectedServers() {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.detectedServers = make(map[string]*RogueServer)
}

// GetConfig returns the current detector configuration.
func (rd *RogueDetector) GetConfig() *RogueDetectorConfig {
	rd.mu.RLock()
	defer rd.mu.RUnlock()

	// Return a copy to prevent external modification
	configCopy := *rd.config
	configCopy.KnownServers = make([]string, len(rd.config.KnownServers))
	copy(configCopy.KnownServers, rd.config.KnownServers)

	return &configCopy
}
