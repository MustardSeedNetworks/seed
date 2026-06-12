package api

// export_sources.go adapts the live diagnostic services to the export use-case's
// Sources port (ADR-0020, WS-A10). Each method gathers and shapes one card from
// the corresponding domain service; an absent service yields ok=false so the
// use-case omits the card. The card-shaping that used to live as a fan-out of
// *Server.export* methods on the handler now lives here, behind the port.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/dhcp"
)

// serverExportSources implements export.Sources over the server's diagnostic
// service accessors.
type serverExportSources struct {
	s *Server
}

func (a serverExportSources) RefreshInterfaces() error {
	return a.s.netManager().RefreshInterfaces()
}

func (a serverExportSources) DeviceMAC(iface string) string {
	if info, err := a.s.netManager().GetInterface(iface); err == nil {
		return info.HardwareAddr
	}
	return ""
}

func (a serverExportSources) IPMode() string { return a.s.config.IP.Mode }

// Cards gathers every present diagnostic card for iface, keyed by name; an absent
// card (its source service unavailable or empty) is omitted.
func (a serverExportSources) Cards(ctx context.Context, iface string) map[string]any {
	cards := make(map[string]any)
	add := func(name string, data map[string]any, ok bool) {
		if ok {
			cards[name] = data
		}
	}
	link, ok := a.linkCard(iface)
	add("link", link, ok)
	ipCfg, ok := a.ipConfigCard(iface)
	add("ipConfig", ipCfg, ok)
	disc, ok := a.discoveryCard()
	add("switch", disc, ok)
	dnsCard, ok := a.dnsCard(ctx)
	add("dns", dnsCard, ok)
	gw, ok := a.gatewayCard()
	add("gateway", gw, ok)
	vlanCard, ok := a.vlanCard()
	add("vlan", vlanCard, ok)
	wifi, ok := a.wifiCard(iface)
	add("wifi", wifi, ok)
	cable, ok := a.cableCard()
	add("cable", cable, ok)
	speed, ok := a.speedtestCard()
	add("speedtest", speed, ok)
	iperfCard, ok := a.iperfCard()
	add("iperf", iperfCard, ok)
	return cards
}

func (a serverExportSources) linkCard(iface string) (map[string]any, bool) {
	linkStatus, err := a.s.netManager().GetLinkStatus(iface)
	if err != nil {
		return nil, false
	}
	return map[string]any{
		"linkUp": linkStatus.LinkUp, "speed": linkStatus.Speed,
		"duplex": linkStatus.Duplex, "autoNeg": linkStatus.AutoNeg,
	}, true
}

func (a serverExportSources) ipConfigCard(iface string) (map[string]any, bool) {
	ifaceInfo, err := a.s.netManager().GetInterface(iface)
	if err != nil {
		return nil, false
	}
	ipData := map[string]any{"addresses": ifaceInfo.Addresses}
	if leaseInfo, leaseErr := dhcp.GetLeaseInfo(iface); leaseErr == nil && leaseInfo != nil {
		ipData["dhcpServer"] = leaseInfo.DHCPServer
		ipData["gateway"] = leaseInfo.Gateway
		ipData["leaseTime"] = leaseInfo.LeaseTime
		ipData["dns"] = leaseInfo.DNS
	}
	return ipData, true
}

func (a serverExportSources) discoveryCard() (map[string]any, bool) {
	if a.s.discoveryService() == nil {
		return nil, false
	}
	neighbors := a.s.discoveryService().GetNeighbors()
	neighborList := make([]map[string]any, 0, len(neighbors))
	for _, n := range neighbors {
		neighborList = append(neighborList, map[string]any{
			"protocol": n.Protocol, "systemName": n.SystemName, "portId": n.PortID,
			"portDescription": n.PortDescription, "managementAddress": n.ManagementAddress,
		})
	}
	return map[string]any{
		"running":   a.s.discoveryService().IsRunning(),
		"neighbors": neighborList,
	}, true
}

func (a serverExportSources) dnsCard(ctx context.Context) (map[string]any, bool) {
	if a.s.dnsTester() == nil {
		return nil, false
	}
	result := a.s.dnsTester().Test(ctx)
	dnsData := map[string]any{"server": result.Server, "testHostname": result.TestHostname}
	if result.Forward != nil {
		dnsData["forward"] = map[string]any{
			"result": result.Forward.Resolved, "time": result.Forward.Time.Milliseconds(),
			"status": result.Forward.Status, "error": result.Forward.Error,
		}
	}
	if result.Reverse != nil {
		dnsData["reverse"] = map[string]any{
			"result": result.Reverse.Resolved, "time": result.Reverse.Time.Milliseconds(),
			"status": result.Reverse.Status, "error": result.Reverse.Error,
		}
	}
	return dnsData, true
}

func (a serverExportSources) gatewayCard() (map[string]any, bool) {
	if a.s.gatewayTester() == nil {
		return nil, false
	}
	stats := a.s.gatewayTester().GetStats()
	return map[string]any{
		"gateway": stats.Gateway, "reachable": stats.Reachable, "sent": stats.Sent,
		"received": stats.Received, "lossPercent": stats.LossPercent,
		"avgTime": stats.AvgTime, "status": stats.Status,
	}, true
}

func (a serverExportSources) vlanCard() (map[string]any, bool) {
	if a.s.vlanManager() == nil {
		return nil, false
	}
	vlanInfo := a.s.vlanManager().GetInfo()
	return map[string]any{
		"nativeVlan": vlanInfo.NativeVlan, "taggedVlans": vlanInfo.TaggedVlans,
		"voiceVlan": vlanInfo.VoiceVlan, "configured": vlanInfo.Configured,
	}, true
}

func (a serverExportSources) wifiCard(iface string) (map[string]any, bool) {
	if !a.s.netManager().IsWireless(iface) || a.s.wifiManager() == nil {
		return nil, false
	}
	wifiInfo := a.s.wifiManager().GetInfo()
	if wifiInfo.SSID == "" {
		return nil, false
	}
	return map[string]any{
		"ssid": wifiInfo.SSID, "bssid": wifiInfo.BSSID, "signal": wifiInfo.Signal,
		"channel": wifiInfo.Channel, "frequency": wifiInfo.Frequency, "security": wifiInfo.Security,
	}, true
}

func (a serverExportSources) cableCard() (map[string]any, bool) {
	if a.s.cableTester() == nil {
		return nil, false
	}
	cableResult := a.s.cableTester().Test()
	return map[string]any{
		"supported": cableResult.Supported, "length": cableResult.Length,
		"status": cableResult.Status, "faults": cableResult.Faults,
	}, true
}

func (a serverExportSources) speedtestCard() (map[string]any, bool) {
	if a.s.speedtestTester() == nil {
		return nil, false
	}
	result := a.s.speedtestTester().GetLastResult()
	if result == nil {
		return nil, false
	}
	return map[string]any{
		"download": result.Download, "upload": result.Upload, "latency": result.Latency,
		"server": result.Server, "location": result.Location, "host": result.Host,
		"distance": result.Distance, "timestamp": result.Timestamp, "testDuration": result.TestDuration,
	}, true
}

func (a serverExportSources) iperfCard() (map[string]any, bool) {
	if a.s.iperfManager() == nil {
		return nil, false
	}
	result := a.s.iperfManager().GetLastResult()
	if result == nil {
		return nil, false
	}
	return map[string]any{
		"bandwidth": result.Bandwidth, "transfer": result.Transfer, "retransmits": result.Retransmits,
		"jitter": result.Jitter, "lostPackets": result.LostPackets, "lostPercent": result.LostPercent,
		"protocol": result.Protocol, "direction": result.Direction, "duration": result.Duration,
		"server": result.Server, "port": result.Port, "timestamp": result.Timestamp,
	}, true
}
