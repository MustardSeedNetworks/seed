package mcp

import (
	"context"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/iperf"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/sap/dns"
	"github.com/krisarmstrong/seed/internal/sap/gateway"
	"github.com/krisarmstrong/seed/internal/sap/speedtest"
)

// WiFiNetwork represents a discovered WiFi network.
type WiFiNetwork struct {
	SSID      string `json:"ssid"`
	BSSID     string `json:"bssid"`
	Signal    int    `json:"signal"` // dBm
	Channel   int    `json:"channel"`
	Frequency int    `json:"frequency"` // MHz
	Security  string `json:"security"`
}

// WiFiConnectionInfo represents the current WiFi connection.
type WiFiConnectionInfo struct {
	SSID      string `json:"ssid"`
	BSSID     string `json:"bssid"`
	Signal    int    `json:"signal"` // dBm
	Channel   int    `json:"channel"`
	Frequency int    `json:"frequency"` // MHz
	Security  string `json:"security"`
}

// DiscoveryService provides access to the unified discovery orchestrator.
type DiscoveryService interface {
	Scan(ctx context.Context) error
	GetDevices() []*discovery.DiscoveredDevice
	GetStatus() *discovery.ServiceStatus
	GetOptions() config.DiscoveryOptions
}

// DeviceDiscovery provides access to device discovery functionality.
type DeviceDiscovery interface {
	GetDiscoveredDevices() []*discovery.DiscoveredDevice
	Scan(ctx context.Context) error
	SetAdditionalSubnets(cidrs []string) error
}

// NetworkManager provides network interface management.
type NetworkManager interface {
	GetInterfaces() []*network.InterfaceInfo
	GetInterface(name string) (*network.InterfaceInfo, error)
	GetCurrentInterface() string
}

// LinkMonitor provides link status monitoring.
type LinkMonitor interface {
	GetState() network.LinkState
	IsUp() bool
}

// VLANManager provides VLAN information.
type VLANManager interface {
	GetInfo() any
}

// DNSTester provides DNS testing functionality.
type DNSTester interface {
	Test(ctx context.Context) *dns.TestResult
}

// GatewayTester provides gateway testing functionality.
type GatewayTester interface {
	Test() *gateway.PingStats
	Ping() *gateway.PingStats
}

// SpeedtestTester provides internet speed testing.
type SpeedtestTester interface {
	Run(ctx context.Context) (*speedtest.Result, error)
	GetStatus() speedtest.Status
}

// IperfManager provides iPerf3 throughput testing.
type IperfManager interface {
	RunClient(ctx context.Context, config *iperf.ClientConfig) (*iperf.Result, error)
	GetClientStatus() iperf.ClientStatus
}

// WiFiScanner provides WiFi network scanning.
type WiFiScanner interface {
	Scan(ctx context.Context) ([]WiFiNetwork, error)
}

// WiFiManager provides WiFi connection information.
type WiFiManager interface {
	GetCurrentNetwork() (*WiFiConnectionInfo, error)
	GetSignalStrength() (int, error)
}

// RogueDetector provides rogue DHCP server detection.
type RogueDetector interface {
	GetDetectedServers() any
	IsRunning() bool
}

// VulnScanner provides vulnerability scanning.
type VulnScanner interface {
	ScanDevice(ctx context.Context, device *discovery.DiscoveredDevice) (any, error)
	GetAllVulnerabilities() any
}

// PublicIPChecker provides public IP detection.
type PublicIPChecker interface {
	GetPublicIP(ctx context.Context) any
}

// TracerouteResult represents the result of a traceroute operation.
type TracerouteResult = discovery.TracerouteResult

// TracerouteHop represents a single hop in a traceroute.
type TracerouteHop = discovery.TracerouteHop

// Tracer provides traceroute functionality.
type Tracer interface {
	Trace(ctx context.Context, target string) (*TracerouteResult, error)
}

// PortScanResult represents the result of a port scan.
type PortScanResult = discovery.PortScanResult

// PortScanner provides port scanning functionality.
type PortScanner interface {
	Scan(ctx context.Context, host string, ports []int) (*PortScanResult, error)
}

// TCPProbeResult represents the result of a TCP probe.
type TCPProbeResult struct {
	Host    string        `json:"host"`
	Port    int           `json:"port"`
	Open    bool          `json:"open"`
	Latency time.Duration `json:"latency"`
	Error   string        `json:"error,omitempty"`
}

// TCPProber provides TCP port probing.
type TCPProber interface {
	Probe(ctx context.Context, host string, port int, timeout time.Duration) (*TCPProbeResult, error)
}
