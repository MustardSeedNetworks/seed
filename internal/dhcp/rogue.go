package dhcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"

	"github.com/krisarmstrong/seed/internal/logging"
)

// Fixes #907: Limits for detected servers map to prevent unbounded growth.
const (
	maxDetectedServers = 1000           // Maximum number of tracked servers
	serverExpiry       = 24 * time.Hour // Expire servers not seen in 24 hours
)

// Packet capture constants for rogue detection.
const (
	// roguePcapSnapshotLen is the snapshot length for pcap capture in rogue detection.
	// 1024 bytes is sufficient for DHCP packets in this context.
	roguePcapSnapshotLen = 1024

	// roguePcapTimeout is the read timeout for pcap in rogue detection.
	roguePcapTimeout = 100 * time.Millisecond
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
	return rd.startLocked()
}

// startLocked starts the detector. Caller must hold rd.mu.Lock().
func (rd *RogueDetector) startLocked() error {
	if rd.running {
		return errors.New("rogue detector already running")
	}

	// Open pcap handle for DHCP traffic (UDP ports 67/68)
	// Use snapshot length of roguePcapSnapshotLen bytes (enough for DHCP packets)
	// Set promiscuous mode to false (we only need broadcast packets)
	// Set timeout to roguePcapTimeout for responsive shutdown
	handle, err := pcap.OpenLive(rd.config.Interface, roguePcapSnapshotLen, false, roguePcapTimeout)
	if err != nil {
		return fmt.Errorf("failed to open pcap: %w (requires CAP_NET_RAW or root)", err)
	}

	// Set BPF filter to capture only DHCP OFFER packets
	// Port 67 is DHCP server, port 68 is DHCP client
	// We want to see OFFER packets (server -> client on port 68)
	filter := "udp and (port 67 or port 68)"
	if filterErr := handle.SetBPFFilter(filter); filterErr != nil {
		handle.Close()
		return fmt.Errorf("failed to set BPF filter: %w", filterErr)
	}

	rd.handle = handle
	rd.running = true

	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	rd.cancel = cancel

	// Start packet capture goroutine
	linkType := handle.LinkType()
	go rd.capturePackets(ctx, handle, linkType)

	logging.GetLogger().Info("Rogue DHCP detector started", "interface", rd.config.Interface)
	return nil
}

// Stop stops the rogue detector.
func (rd *RogueDetector) Stop() error {
	rd.mu.Lock()
	defer rd.mu.Unlock()
	rd.stopLocked()
	return nil
}

// stopLocked stops the detector. Caller must hold rd.mu.Lock().
func (rd *RogueDetector) stopLocked() {
	if !rd.running {
		return
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

	logging.GetLogger().Info("Rogue DHCP detector stopped")
}

// IsRunning returns whether the detector is currently running.
func (rd *RogueDetector) IsRunning() bool {
	rd.mu.RLock()
	defer rd.mu.RUnlock()
	return rd.running
}

// capturePackets is the main packet capture loop.
func (rd *RogueDetector) capturePackets(ctx context.Context, handle *pcap.Handle, linkType layers.LinkType) {
	// Ensure cleanup happens if capturePackets exits unexpectedly (fixes #831)
	defer func() {
		rd.mu.Lock()
		if rd.running {
			// goroutine exited unexpectedly, clean up resources
			if rd.handle != nil {
				rd.handle.Close()
				rd.handle = nil
			}
			rd.running = false
			rd.cancel = nil
			logging.GetLogger().Warn("Rogue DHCP detector capture loop exited unexpectedly")
		}
		rd.mu.Unlock()
	}()

	if handle == nil {
		return
	}

	packetSource := gopacket.NewPacketSource(handle, linkType)

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
	dhcp := rd.extractDHCPOffer(packet)
	if dhcp == nil {
		return
	}

	serverIP, serverMAC := rd.extractServerInfo(packet, dhcp)
	if serverIP == "" {
		return
	}

	rd.recordDetectedServer(serverIP, serverMAC)
}

// extractDHCPOffer extracts the DHCP layer if the packet is a DHCP OFFER.
// Returns nil if the packet is not a valid DHCP OFFER.
func (rd *RogueDetector) extractDHCPOffer(packet gopacket.Packet) *layers.DHCPv4 {
	dhcpLayer := packet.Layer(layers.LayerTypeDHCPv4)
	if dhcpLayer == nil {
		return nil
	}

	dhcp, ok := dhcpLayer.(*layers.DHCPv4)
	if !ok {
		return nil
	}

	// We only care about DHCP OFFER packets (replies with message type OFFER)
	if dhcp.Operation != layers.DHCPOpReply {
		return nil
	}

	messageType := rd.getDHCPMessageType(dhcp)
	if messageType != layers.DHCPMsgTypeOffer {
		return nil
	}

	return dhcp
}

// extractServerInfo extracts the server IP and MAC address from a DHCP packet.
// Returns empty strings if the information cannot be extracted.
func (rd *RogueDetector) extractServerInfo(
	packet gopacket.Packet,
	dhcp *layers.DHCPv4,
) (string, string) {
	// Try to get server identifier from DHCP options first
	serverIP := rd.getServerIdentifier(dhcp)
	if serverIP == "" {
		// Fallback to source IP if server identifier not found
		serverIP = rd.extractSourceIP(packet)
	}

	serverMAC := rd.extractSourceMAC(packet)
	return serverIP, serverMAC
}

// extractSourceIP extracts the source IP address from the packet's IPv4 layer.
func (rd *RogueDetector) extractSourceIP(packet gopacket.Packet) string {
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer == nil {
		return ""
	}

	ip, ok := ipLayer.(*layers.IPv4)
	if !ok {
		return ""
	}

	return ip.SrcIP.String()
}

// extractSourceMAC extracts the source MAC address from the packet's Ethernet layer.
func (rd *RogueDetector) extractSourceMAC(packet gopacket.Packet) string {
	ethLayer := packet.Layer(layers.LayerTypeEthernet)
	if ethLayer == nil {
		return ""
	}

	eth, ok := ethLayer.(*layers.Ethernet)
	if !ok {
		return ""
	}

	return eth.SrcMAC.String()
}

// recordDetectedServer records a detected DHCP server, handling both new
// servers and updates to existing ones. Caller must not hold rd.mu.
func (rd *RogueDetector) recordDetectedServer(serverIP, serverMAC string) {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	now := time.Now()
	rd.pruneExpiredServers(now)

	server, exists := rd.detectedServers[serverIP]
	if !exists {
		rd.addNewServer(serverIP, serverMAC, now)
	} else {
		rd.updateExistingServer(server, serverMAC, now)
	}
}

// pruneExpiredServers removes servers not seen within the expiry window.
// Fixes #907: Prevents unbounded growth of detected servers map.
// Caller must hold rd.mu.Lock().
func (rd *RogueDetector) pruneExpiredServers(now time.Time) {
	if len(rd.detectedServers) <= maxDetectedServers/2 {
		return
	}

	for ip, srv := range rd.detectedServers {
		if now.Sub(srv.LastSeen) > serverExpiry {
			delete(rd.detectedServers, ip)
		}
	}
}

// addNewServer adds a newly detected DHCP server to the tracking map.
// Fixes #907: Enforces hard limit on tracked servers.
// Caller must hold rd.mu.Lock().
func (rd *RogueDetector) addNewServer(serverIP, serverMAC string, now time.Time) {
	// Hard limit - don't add more servers if at capacity
	if len(rd.detectedServers) >= maxDetectedServers {
		logging.GetLogger().
			Warn("Detected servers limit reached, skipping new server", "ip", serverIP)
		return
	}

	isKnown := rd.knownServerSet[serverIP]
	server := &RogueServer{
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
		logging.GetLogger().Warn("Rogue DHCP server detected", "ip", serverIP, "mac", serverMAC)
	}
}

// updateExistingServer updates the tracking information for a known server.
// Caller must hold rd.mu.Lock().
func (rd *RogueDetector) updateExistingServer(server *RogueServer, serverMAC string, now time.Time) {
	server.LastSeen = now
	server.OfferCount++
	if serverMAC != "" && server.MAC == "" {
		server.MAC = serverMAC
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

// SetInterface changes the monitored network interface.
// If the detector is running, it will be stopped and restarted on the new interface.
// This ensures rogue DHCP detection continues on the correct network segment. (fixes #838)
// Fixes #898: Hold lock through entire stop/update/restart to prevent race condition.
func (rd *RogueDetector) SetInterface(iface string) error {
	rd.mu.Lock()
	defer rd.mu.Unlock()

	wasRunning := rd.running

	if wasRunning {
		rd.stopLocked()
	}

	rd.config.Interface = iface

	if wasRunning {
		if err := rd.startLocked(); err != nil {
			return fmt.Errorf("failed to restart on new interface: %w", err)
		}
	}
	return nil
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
