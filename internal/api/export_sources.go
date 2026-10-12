package api

// export_sources.go adapts the live diagnostic services to the export use-case's
// Sources port (ADR-0020, WS-A10). Each method gathers and shapes one card from
// the corresponding domain service; an absent service yields a nil card, which
// the document leaves out. The card-shaping that used to live as a fan-out of
// *Server.export* methods on the handler now lives here, behind the port.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/export"
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

// Cards gathers every present diagnostic card for iface.
func (a serverExportSources) Cards(ctx context.Context, iface string) export.Cards {
	return export.Cards{
		Link:      a.linkCard(iface),
		IPConfig:  a.ipConfigCard(iface),
		Switch:    a.switchCard(),
		DNS:       a.dnsCard(ctx),
		Gateway:   a.gatewayCard(),
		VLAN:      a.vlanCard(),
		WiFi:      a.wifiCard(iface),
		Cable:     a.cableCard(),
		Speedtest: a.speedtestCard(),
		Iperf:     a.iperfCard(),
	}
}

func (a serverExportSources) linkCard(iface string) *export.LinkCard {
	linkStatus, err := a.s.netManager().GetLinkStatus(iface)
	if err != nil {
		return nil
	}
	return &export.LinkCard{
		LinkUp: linkStatus.LinkUp, Speed: linkStatus.Speed,
		Duplex: linkStatus.Duplex, AutoNeg: linkStatus.AutoNeg,
	}
}

func (a serverExportSources) ipConfigCard(iface string) *export.IPConfigCard {
	ifaceInfo, err := a.s.netManager().GetInterface(iface)
	if err != nil {
		return nil
	}
	card := &export.IPConfigCard{Addresses: emptyIfNil(ifaceInfo.Addresses)}
	if leaseInfo, leaseErr := dhcp.GetLeaseInfo(iface); leaseErr == nil && leaseInfo != nil {
		card.Lease = &export.LeaseCard{
			DHCPServer: leaseInfo.DHCPServer, Gateway: leaseInfo.Gateway,
			LeaseTime: leaseInfo.LeaseTime, DNS: emptyIfNil(leaseInfo.DNS),
		}
	}
	return card
}

func (a serverExportSources) switchCard() *export.SwitchCard {
	if a.s.discoveryService() == nil {
		return nil
	}
	neighbors := a.s.discoveryService().GetNeighbors()
	records := make([]export.NeighborRecord, 0, len(neighbors))
	for _, n := range neighbors {
		records = append(records, export.NeighborRecord{
			Protocol: string(n.Protocol), SystemName: n.SystemName, PortID: n.PortID,
			PortDescription: n.PortDescription, ManagementAddress: n.ManagementAddress,
		})
	}
	return &export.SwitchCard{Running: a.s.discoveryService().IsRunning(), Neighbors: records}
}

func (a serverExportSources) dnsCard(ctx context.Context) *export.DNSCard {
	if a.s.dnsTester() == nil {
		return nil
	}
	result := a.s.dnsTester().Test(ctx)
	return &export.DNSCard{
		Server:       result.Server,
		TestHostname: result.TestHostname,
		Forward:      exportLookup(result.Forward),
		Reverse:      exportLookup(result.Reverse),
	}
}

func exportLookup(r *dns.LookupResult) *export.DNSLookup {
	if r == nil {
		return nil
	}
	return &export.DNSLookup{
		Result: emptyIfNil(r.Resolved), Time: r.Time.Milliseconds(),
		Status: string(r.Status), Error: r.Error,
	}
}

func (a serverExportSources) gatewayCard() *export.GatewayCard {
	if a.s.gatewayTester() == nil {
		return nil
	}
	stats := a.s.gatewayTester().GetStats()
	return &export.GatewayCard{
		Gateway: stats.Gateway, Reachable: stats.Reachable, Sent: stats.Sent,
		Received: stats.Received, LossPercent: stats.LossPercent,
		AvgTime: stats.AvgTime, Status: string(stats.Status),
	}
}

func (a serverExportSources) vlanCard() *export.VLANCard {
	if a.s.vlanManager() == nil {
		return nil
	}
	vlanInfo := a.s.vlanManager().GetInfo()
	return &export.VLANCard{
		NativeVlan: vlanInfo.NativeVlan, TaggedVlans: emptyIfNil(vlanInfo.TaggedVlans),
		VoiceVlan: vlanInfo.VoiceVlan,
		Configured: export.VLANConfigured{
			Enabled: vlanInfo.Configured.Enabled, ID: vlanInfo.Configured.ID,
		},
	}
}

func (a serverExportSources) wifiCard(iface string) *export.WiFiCard {
	if !a.s.netManager().IsWireless(iface) || a.s.wifiManager() == nil {
		return nil
	}
	wifiInfo := a.s.wifiManager().GetInfo()
	if wifiInfo.SSID == "" {
		return nil
	}
	return &export.WiFiCard{
		SSID: wifiInfo.SSID, BSSID: wifiInfo.BSSID, Signal: wifiInfo.Signal,
		Channel: wifiInfo.Channel, Frequency: wifiInfo.Frequency, Security: wifiInfo.Security,
	}
}

func (a serverExportSources) cableCard() *export.CableCard {
	if a.s.cableTester() == nil {
		return nil
	}
	cableResult := a.s.cableTester().Test()
	return &export.CableCard{
		Supported: cableResult.Supported, Length: cableResult.Length,
		Status: string(cableResult.Status), Faults: emptyIfNil(cableResult.Faults),
	}
}

func (a serverExportSources) speedtestCard() *export.SpeedtestCard {
	if a.s.speedtestTester() == nil {
		return nil
	}
	result := a.s.speedtestTester().GetLastResult()
	if result == nil {
		return nil
	}
	return &export.SpeedtestCard{
		Download: result.Download, Upload: result.Upload, Latency: result.Latency,
		Server: result.Server, Location: result.Location, Host: result.Host,
		Distance: result.Distance, Timestamp: result.Timestamp, TestDuration: result.TestDuration,
	}
}

func (a serverExportSources) iperfCard() *export.IperfCard {
	if a.s.iperfManager() == nil {
		return nil
	}
	result := a.s.iperfManager().GetLastResult()
	if result == nil {
		return nil
	}
	return &export.IperfCard{
		Bandwidth: result.Bandwidth, Transfer: result.Transfer, Retransmits: result.Retransmits,
		Jitter: result.Jitter, LostPackets: result.LostPackets, LostPercent: result.LostPercent,
		Protocol: result.Protocol, Direction: result.Direction, Duration: result.Duration,
		Server: result.Server, Port: result.Port, Timestamp: result.Timestamp,
	}
}
