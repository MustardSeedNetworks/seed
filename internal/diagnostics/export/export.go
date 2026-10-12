// Package export is the diagnostic-export use-case (ADR-0020, WS-A10): the
// /api/v1/export endpoint's application service. It coordinates the per-card
// diagnostic sources into one export document, owning the assembly logic the
// transport layer used to carry as a fan-out of *Server methods. The card data
// is gathered through the consumer-defined Sources port, satisfied by an adapter
// in the api layer over the live diagnostic services.
package export

import (
	"context"
	"time"
)

// DeviceInfo is the export's device summary.
type DeviceInfo struct {
	Interface string `json:"interface"`
	MAC       string `json:"mac,omitempty"`
	IPMode    string `json:"ipMode"`
}

// Data is the full diagnostic export document.
type Data struct {
	Version   string     `json:"version"`
	Timestamp string     `json:"timestamp"`
	Device    DeviceInfo `json:"device"`
	Cards     Cards      `json:"cards"`
}

// Cards holds one entry per diagnostic card. A card whose source is
// unavailable or has nothing to report is nil and left out of the document.
type Cards struct {
	Link      *LinkCard      `json:"link,omitempty"`
	IPConfig  *IPConfigCard  `json:"ipConfig,omitempty"`
	Switch    *SwitchCard    `json:"switch,omitempty"`
	DNS       *DNSCard       `json:"dns,omitempty"`
	Gateway   *GatewayCard   `json:"gateway,omitempty"`
	VLAN      *VLANCard      `json:"vlan,omitempty"`
	WiFi      *WiFiCard      `json:"wifi,omitempty"`
	Cable     *CableCard     `json:"cable,omitempty"`
	Speedtest *SpeedtestCard `json:"speedtest,omitempty"`
	Iperf     *IperfCard     `json:"iperf,omitempty"`
}

// LinkCard is the exported interface's link state.
type LinkCard struct {
	LinkUp  bool   `json:"linkUp"`
	Speed   string `json:"speed"`
	Duplex  string `json:"duplex"`
	AutoNeg bool   `json:"autoNeg"`
}

// IPConfigCard is the interface's addresses and, when the host holds one, its
// DHCP lease.
type IPConfigCard struct {
	Addresses []string   `json:"addresses"`
	Lease     *LeaseCard `json:"lease,omitempty"`
}

// LeaseCard is the DHCP lease the host holds for the interface.
type LeaseCard struct {
	DHCPServer string   `json:"dhcpServer"`
	Gateway    string   `json:"gateway"`
	LeaseTime  int      `json:"leaseTime"` // seconds
	DNS        []string `json:"dns"`
}

// SwitchCard is the LLDP/CDP neighbor discovery state.
type SwitchCard struct {
	Running   bool             `json:"running"`
	Neighbors []NeighborRecord `json:"neighbors"`
}

// NeighborRecord is one switch or device advertising itself on the link.
type NeighborRecord struct {
	Protocol          string `json:"protocol"`
	SystemName        string `json:"systemName"`
	PortID            string `json:"portId"`
	PortDescription   string `json:"portDescription"`
	ManagementAddress string `json:"managementAddress"`
}

// DNSCard is a DNS test against the configured server.
type DNSCard struct {
	Server       string     `json:"server"`
	TestHostname string     `json:"testHostname"`
	Forward      *DNSLookup `json:"forward,omitempty"`
	Reverse      *DNSLookup `json:"reverse,omitempty"`
}

// DNSLookup is one forward or reverse lookup of the DNS test.
type DNSLookup struct {
	Result []string `json:"result"`
	Time   int64    `json:"time"` // ms
	Status string   `json:"status"`
	Error  string   `json:"error"`
}

// GatewayCard is the default gateway's ping statistics.
type GatewayCard struct {
	Gateway     string  `json:"gateway"`
	Reachable   bool    `json:"reachable"`
	Sent        int     `json:"sent"`
	Received    int     `json:"received"`
	LossPercent float64 `json:"lossPercent"`
	AvgTime     float64 `json:"avgTime"` // ms
	Status      string  `json:"status"`
}

// VLANCard is the VLAN membership seen on the link. A VLAN not detected is
// left out.
type VLANCard struct {
	NativeVlan  *int           `json:"nativeVlan,omitempty"`
	TaggedVlans []int          `json:"taggedVlans"`
	VoiceVlan   *int           `json:"voiceVlan,omitempty"`
	Configured  VLANConfigured `json:"configured"`
}

// VLANConfigured is the VLAN tag the operator configured.
type VLANConfigured struct {
	Enabled bool `json:"enabled"`
	ID      int  `json:"id"`
}

// WiFiCard is the wireless association of the exported interface.
type WiFiCard struct {
	SSID      string `json:"ssid"`
	BSSID     string `json:"bssid"`
	Signal    int    `json:"signal"` // dBm
	Channel   int    `json:"channel"`
	Frequency int    `json:"frequency"` // MHz
	Security  string `json:"security"`
}

// CableCard is the cable test result. Length is left out when the NIC
// measured none.
type CableCard struct {
	Supported bool     `json:"supported"`
	Length    *float64 `json:"length,omitempty"` // meters
	Status    string   `json:"status"`
	Faults    []string `json:"faults"`
}

// SpeedtestCard is the last internet speed test.
type SpeedtestCard struct {
	Download     float64   `json:"download"` // Mbps
	Upload       float64   `json:"upload"`   // Mbps
	Latency      float64   `json:"latency"`  // ms
	Server       string    `json:"server"`
	Location     string    `json:"location"`
	Host         string    `json:"host"`
	Distance     float64   `json:"distance"` // km
	Timestamp    time.Time `json:"timestamp"`
	TestDuration float64   `json:"testDuration"` // seconds
}

// IperfCard is the last iperf3 client test.
type IperfCard struct {
	Bandwidth   float64   `json:"bandwidth"` // Mbps
	Transfer    float64   `json:"transfer"`  // MB
	Retransmits int       `json:"retransmits"`
	Jitter      float64   `json:"jitter"` // ms
	LostPackets int       `json:"lostPackets"`
	LostPercent float64   `json:"lostPercent"`
	Protocol    string    `json:"protocol"`
	Direction   string    `json:"direction"`
	Duration    float64   `json:"duration"` // seconds
	Server      string    `json:"server"`
	Port        int       `json:"port"`
	Timestamp   time.Time `json:"timestamp"`
}

// Sources provides the export's device info and the assembled per-card
// diagnostic data; the per-card gathering/shaping lives in the adapter.
type Sources interface {
	RefreshInterfaces() error
	DeviceMAC(iface string) string
	IPMode() string
	Cards(ctx context.Context, iface string) Cards
}

// Service is the diagnostic-export use-case.
type Service struct {
	src Sources
}

// NewService builds the use-case over its Sources port.
func NewService(src Sources) *Service { return &Service{src: src} }

// Build assembles the export document for iface, stamping it with ver. It first
// refreshes the interface set (an error there aborts the export), then composes
// the device summary and the gathered cards.
func (s *Service) Build(ctx context.Context, iface, ver string) (Data, error) {
	if err := s.src.RefreshInterfaces(); err != nil {
		return Data{}, err
	}
	return Data{
		Version:   ver,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Device: DeviceInfo{
			Interface: iface,
			MAC:       s.src.DeviceMAC(iface),
			IPMode:    s.src.IPMode(),
		},
		Cards: s.src.Cards(ctx, iface),
	}, nil
}
