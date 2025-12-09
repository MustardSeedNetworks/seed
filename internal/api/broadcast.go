package api

import (
	"context"
	"log"
	"time"
)

const (
	// broadcastInterval is how often we push updates to WebSocket clients.
	broadcastInterval = 5 * time.Second
)

// startBroadcastLoop starts a background goroutine that periodically
// collects data and broadcasts card updates to all connected WebSocket clients.
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
	log.Printf("WebSocket broadcast loop started (interval: %v)", broadcastInterval)
}

// broadcastAllCards collects and broadcasts all card data to connected clients.
func (s *Server) broadcastAllCards() {
	// Link card
	if linkData := s.collectLinkData(); linkData != nil {
		s.wsHub.BroadcastCardUpdate("link", linkData)
	}

	// Gateway card
	if gatewayData := s.collectGatewayData(); gatewayData != nil {
		s.wsHub.BroadcastCardUpdate("gateway", gatewayData)
	}

	// DNS card
	if dnsData := s.collectDNSData(); dnsData != nil {
		s.wsHub.BroadcastCardUpdate("dns", dnsData)
	}

	// Discovery/Switch card
	if switchData := s.collectDiscoveryData(); switchData != nil {
		s.wsHub.BroadcastCardUpdate("switch", switchData)
	}

	// Public IP card
	if publicIPData := s.collectPublicIPData(); publicIPData != nil {
		s.wsHub.BroadcastCardUpdate("publicip", publicIPData)
	}
}

// collectLinkData gathers link status data for the current interface.
func (s *Server) collectLinkData() map[string]interface{} {
	if s.netManager == nil {
		return nil
	}

	if err := s.netManager.RefreshInterfaces(); err != nil {
		log.Printf("failed to refresh interfaces: %v", err)
		return nil
	}
	currentIface := s.netManager.GetCurrentInterface()

	ifaceInfo, err := s.netManager.GetInterface(currentIface)
	if err != nil {
		return nil
	}

	linkStatus, err := s.netManager.GetLinkStatus(currentIface)
	if err != nil {
		log.Printf("failed to get link status for %s: %v", currentIface, err)
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
