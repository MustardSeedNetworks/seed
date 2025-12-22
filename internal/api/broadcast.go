// Package api implements real-time dashboard data broadcasting via WebSocket.
//
// This file contains the broadcast loop that periodically collects network monitoring
// data and pushes updates to all connected WebSocket clients. The broadcast mechanism
// enables real-time dashboard updates without polling.
//
// Architecture:
//   - Single background goroutine running the broadcast loop
//   - Periodic tick every 5 seconds to collect and broadcast data
//   - Skip collection when no clients are connected (performance optimization)
//   - Each card type has a dedicated collector function
//   - Data is cached briefly to avoid expensive repeated collection
//
// Broadcast cards (updated every 5 seconds):
//   - Link: Interface status, speed, duplex, carrier state
//   - Gateway: Gateway reachability, latency, packet loss
//   - DNS: DNS resolution tests, latency, failures
//   - Switch: LLDP/CDP discovered neighbors
//   - Public IP: External IP address, geolocation
//
// Performance considerations:
//   - Collectors use short timeouts to prevent blocking the broadcast loop
//   - Results are cached to handle multiple concurrent WebSocket clients efficiently
//   - Failed operations are logged but don't stop the broadcast loop
//   - Collectors skip work when no clients are connected
//
// The broadcast loop is started automatically during server initialization
// and runs until the WebSocket hub signals shutdown.
package api

import (
	"context"
	"log/slog"
	"time"
)

const (
	// broadcastInterval specifies how frequently dashboard cards are updated and pushed
	// to WebSocket clients. Set to 5 seconds to balance responsiveness with system load.
	//
	// Rationale:
	//   - Network state changes (link, gateway, DNS) are typically not sub-second events
	//   - 5 seconds provides near-real-time updates without excessive CPU usage
	//   - Some operations (DNS tests, gateway pings) take 1-2 seconds, so 5s allows completion
	//
	// Adjust this value based on:
	//   - Network volatility (unstable networks may benefit from shorter intervals)
	//   - System resources (lower-power devices may need longer intervals)
	//   - Number of concurrent clients (more clients = more broadcast overhead)
	broadcastInterval = 5 * time.Second
)

// startBroadcastLoop starts a background goroutine that periodically collects network
// monitoring data and broadcasts card updates to all connected WebSocket clients.
//
// The loop runs indefinitely until the WebSocket hub signals shutdown. It:
//  1. Wakes every broadcastInterval (5 seconds)
//  2. Checks if any clients are connected (skips work if none)
//  3. Collects data from all card collectors
//  4. Broadcasts updates via the WebSocket hub
//  5. Returns to sleep until next interval
//
// This function is called once during server initialization. Multiple calls will
// create duplicate broadcast loops (avoid this - no deduplication).
//
// The broadcast loop is lightweight when no clients are connected - it only checks
// the client count and immediately continues. Once clients connect, data collection
// begins and continues as long as at least one client remains connected.
//
// Goroutine lifecycle:
//   - Started: During server initialization after WebSocket hub is created
//   - Runs: Until server shutdown or WebSocket hub shutdown signal
//   - Cleanup: Ticker is stopped via defer, goroutine exits on shutdown signal
func (s *Server) startBroadcastLoop() {
	go func() {
		ticker := time.NewTicker(broadcastInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Only broadcast if there are connected clients
				if s.wsHub.ClientCount() == 0 {
					continue
				}

				s.broadcastAllCards()

			case <-s.wsHub.shutdown:
				return
			}
		}
	}()
	slog.Info("WebSocket broadcast loop started", "interval", broadcastInterval)
}

// broadcastAllCards collects and broadcasts all dashboard card data to connected clients.
//
// This function orchestrates the data collection process by calling individual collector
// functions for each card type. Each collector:
//   - Returns nil if data collection fails or is unavailable
//   - Returns a map[string]interface{} with card-specific data on success
//   - Uses short timeouts to prevent blocking the broadcast loop
//   - May cache results briefly to reduce repeated expensive operations
//
// Collectors are independent - failure of one doesn't affect others. Failed collections
// are logged but don't prevent other cards from being updated. This ensures partial
// updates are still delivered to clients even if some data sources are unavailable.
//
// Card update order is not significant - all updates are sent concurrently by the hub.
// The order here reflects typical user priority/importance rather than technical dependency.
//
// Multi-interface support (#754): Interface-specific cards include the interface name
// in the broadcast message so clients can filter updates based on their selected interface.
//
// Called by: startBroadcastLoop every 5 seconds when clients are connected.
func (s *Server) broadcastAllCards() {
	// Get current interface for scoped broadcasts
	currentIface := ""
	if s.netManager != nil {
		currentIface = s.netManager.GetCurrentInterface()
	}

	// Link card (interface-specific)
	if linkData := s.collectLinkData(); linkData != nil {
		s.wsHub.BroadcastCardUpdateForInterface("link", linkData, currentIface)
	}

	// Gateway card (interface-specific - routes through selected interface)
	if gatewayData := s.collectGatewayData(); gatewayData != nil {
		s.wsHub.BroadcastCardUpdateForInterface("gateway", gatewayData, currentIface)
	}

	// DNS card (interface-specific - uses interface's DNS servers)
	if dnsData := s.collectDNSData(); dnsData != nil {
		s.wsHub.BroadcastCardUpdateForInterface("dns", dnsData, currentIface)
	}

	// Discovery/Switch card (interface-specific - LLDP/CDP on selected interface)
	if switchData := s.collectDiscoveryData(); switchData != nil {
		s.wsHub.BroadcastCardUpdateForInterface("switch", switchData, currentIface)
	}

	// Public IP card (global - not interface-specific)
	if publicIPData := s.collectPublicIPData(); publicIPData != nil {
		s.wsHub.BroadcastCardUpdate("publicip", publicIPData)
	}
}

// collectLinkData gathers physical and logical link status for the current network interface.
//
// Collected data includes:
//   - Interface name, type, and hardware address
//   - Physical link state: carrier detected (Layer 2), speed, duplex
//   - Logical link state: interface up flag, running flag, has routable IP (Layer 3)
//   - Network addresses: IPv4 and IPv6 addresses with prefix lengths
//   - MTU (Maximum Transmission Unit)
//
// The function distinguishes between:
//   - Carrier: Physical link detected (cable connected, WiFi associated) - Layer 2
//   - HasIP: Routable IP address assigned (successful DHCP or static config) - Layer 3
//   - Up: Interface administratively enabled (ifconfig up / ip link set up)
//   - Running: Interface operational and ready for traffic
//
// This distinction is important for troubleshooting:
//   - Carrier + no HasIP: Physical connection OK, DHCP/IP config failed
//   - No carrier + has IP: Cable unplugged or WiFi disconnected (stale IP)
//   - Both false: Cable unplugged or interface down
//   - Both true: Fully operational connection
//
// Returns:
//   - map[string]interface{} containing link status data, or
//   - nil if network manager is unavailable or interface refresh fails
//
// Performance: This function refreshes the interface list on each call, which involves
// system calls. The 5-second broadcast interval is chosen to balance update frequency
// with this overhead.
func (s *Server) collectLinkData() map[string]interface{} {
	if s.netManager == nil {
		return nil
	}

	if err := s.netManager.RefreshInterfaces(); err != nil {
		slog.Warn("Failed to refresh interfaces", "error", err)
		return nil
	}
	currentIface := s.netManager.GetCurrentInterface()

	ifaceInfo, err := s.netManager.GetInterface(currentIface)
	if err != nil {
		return nil
	}

	linkStatus, err := s.netManager.GetLinkStatus(currentIface)
	if err != nil {
		slog.Warn("Failed to get link status", "interface", currentIface, "error", err)
		return nil
	}

	data := map[string]interface{}{
		"interface": currentIface,
		"linkUp":    false,
		"mtu":       ifaceInfo.MTU,
	}

	if linkStatus != nil {
		data["linkUp"] = linkStatus.LinkUp
		data["carrier"] = linkStatus.Carrier
		data["hasIP"] = linkStatus.HasIP
		data["speed"] = linkStatus.Speed
		data["duplex"] = linkStatus.Duplex
		data["advertisedSpeeds"] = linkStatus.Advertised
		data["autoNeg"] = linkStatus.AutoNeg
	}

	return data
}

// collectGatewayData gathers gateway ping data from cached stats.
func (s *Server) collectGatewayData() map[string]interface{} {
	if s.gatewayTester == nil {
		return nil
	}

	stats := s.gatewayTester.GetStats()
	if stats == nil {
		return nil
	}

	return map[string]interface{}{
		"gateway":     stats.Gateway,
		"reachable":   stats.Reachable,
		"sent":        stats.Sent,
		"received":    stats.Received,
		"lossPercent": stats.LossPercent,
		"minTime":     stats.MinTime,
		"maxTime":     stats.MaxTime,
		"avgTime":     stats.AvgTime,
		"lastTime":    stats.LastTime,
		"status":      stats.Status,
	}
}

// collectDNSData gathers DNS test data.
func (s *Server) collectDNSData() map[string]interface{} {
	if s.dnsTester == nil {
		return nil
	}

	// Use a short timeout context for the test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results := s.dnsTester.Test(ctx)
	if results == nil {
		return nil
	}

	data := map[string]interface{}{
		"server":       results.Server,
		"servers":      results.Servers,
		"testHostname": results.TestHostname,
	}

	if results.Forward != nil {
		data["forward"] = map[string]interface{}{
			"result":   results.Forward.Result,
			"timeMs":   results.Forward.TimeMs,
			"status":   results.Forward.Status,
			"error":    results.Forward.Error,
			"resolved": results.Forward.Resolved,
		}
	}

	if results.ForwardIPv6 != nil {
		data["forwardIpv6"] = map[string]interface{}{
			"result":   results.ForwardIPv6.Result,
			"timeMs":   results.ForwardIPv6.TimeMs,
			"status":   results.ForwardIPv6.Status,
			"error":    results.ForwardIPv6.Error,
			"resolved": results.ForwardIPv6.Resolved,
		}
	}

	if results.Reverse != nil {
		data["reverse"] = map[string]interface{}{
			"result":   results.Reverse.Result,
			"timeMs":   results.Reverse.TimeMs,
			"status":   results.Reverse.Status,
			"error":    results.Reverse.Error,
			"resolved": results.Reverse.Resolved,
		}
	}

	return data
}

// collectDiscoveryData gathers LLDP/CDP/EDP neighbor data.
func (s *Server) collectDiscoveryData() map[string]interface{} {
	if s.discoveryManager == nil {
		return nil
	}

	neighbors := s.discoveryManager.GetNeighbors()
	if len(neighbors) == 0 {
		return nil
	}

	// Return first neighbor as the "nearest switch"
	neighbor := neighbors[0]
	return map[string]interface{}{
		"protocol":          neighbor.Protocol,
		"switchName":        neighbor.SystemName,
		"portId":            neighbor.PortID,
		"portDescription":   neighbor.PortDescription,
		"managementIp":      neighbor.ManagementAddress,
		"systemDescription": neighbor.SystemDescription,
	}
}

// collectPublicIPData gathers public IP address information.
func (s *Server) collectPublicIPData() map[string]interface{} {
	if s.publicipChecker == nil {
		return nil
	}

	// Use cached result (non-blocking call with short timeout context)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	result := s.publicipChecker.GetPublicIP(ctx)
	if result == nil {
		return nil
	}

	return map[string]interface{}{
		"ipv4":        result.IPv4,
		"ipv6":        result.IPv6,
		"lastChecked": result.LastChecked.Format(time.RFC3339),
		"error":       result.Error,
	}
}
