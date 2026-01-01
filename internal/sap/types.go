package sap

import (
	"time"
)

// LinkStatus represents network interface status.
type LinkStatus struct {
	Interface  string    `json:"interface"`
	State      LinkState `json:"state"`
	Speed      string    `json:"speed"`
	Duplex     string    `json:"duplex"`
	MTU        int       `json:"mtu"`
	MACAddress string    `json:"macAddress"`
	IPAddress  string    `json:"ipAddress,omitempty"`
	Gateway    string    `json:"gateway,omitempty"`
	Carrier    bool      `json:"carrier"`
	TxBytes    uint64    `json:"txBytes"`
	RxBytes    uint64    `json:"rxBytes"`
	TxPackets  uint64    `json:"txPackets"`
	RxPackets  uint64    `json:"rxPackets"`
	TxErrors   uint64    `json:"txErrors"`
	RxErrors   uint64    `json:"rxErrors"`
	TxDropped  uint64    `json:"txDropped"`
	RxDropped  uint64    `json:"rxDropped"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// LinkState represents link operational state.
type LinkState string

// LinkState values.
const (
	LinkStateUp      LinkState = "up"
	LinkStateDown    LinkState = "down"
	LinkStateDormant LinkState = "dormant"
	LinkStateUnknown LinkState = "unknown"
)

// CableTestResult contains TDR cable test results.
type CableTestResult struct {
	Interface   string       `json:"interface"`
	Status      CableStatus  `json:"status"`
	Length      float64      `json:"lengthMeters,omitempty"`
	PairResults []PairResult `json:"pairResults,omitempty"`
	TestedAt    time.Time    `json:"testedAt"`
}

// CableStatus represents overall cable status.
type CableStatus string

// CableStatus values.
const (
	CableStatusOK        CableStatus = "ok"
	CableStatusOpen      CableStatus = "open"
	CableStatusShort     CableStatus = "short"
	CableStatusImpedance CableStatus = "impedance_mismatch"
	CableStatusUnknown   CableStatus = "unknown"
)

// PairResult contains test results for a single wire pair.
type PairResult struct {
	Pair      int         `json:"pair"` // 1-4
	Status    CableStatus `json:"status"`
	Length    float64     `json:"lengthMeters,omitempty"`
	Impedance float64     `json:"impedanceOhms,omitempty"`
}

// DHCPTestResult contains DHCP test results.
type DHCPTestResult struct {
	Success      bool          `json:"success"`
	ServerIP     string        `json:"serverIp,omitempty"`
	OfferedIP    string        `json:"offeredIp,omitempty"`
	SubnetMask   string        `json:"subnetMask,omitempty"`
	Gateway      string        `json:"gateway,omitempty"`
	DNSServers   []string      `json:"dnsServers,omitempty"`
	LeaseTime    time.Duration `json:"leaseTime,omitempty"`
	LeaseTimeSec int           `json:"leaseTimeSec,omitempty"`
	ResponseTime time.Duration `json:"responseTime"`
	ResponseMs   float64       `json:"responseTimeMs"`
	Error        string        `json:"error,omitempty"`
	TestedAt     time.Time     `json:"testedAt"`
}

// DNSTestResult contains DNS test results.
type DNSTestResult struct {
	Query         string        `json:"query"`
	Server        string        `json:"server"`
	Success       bool          `json:"success"`
	Answers       []DNSAnswer   `json:"answers,omitempty"`
	ResponseTime  time.Duration `json:"responseTime"`
	ResponseMs    float64       `json:"responseTimeMs"`
	DNSSEC        bool          `json:"dnssec"`
	Authoritative bool          `json:"authoritative"`
	Error         string        `json:"error,omitempty"`
	TestedAt      time.Time     `json:"testedAt"`
}

// DNSAnswer represents a DNS response record.
type DNSAnswer struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   int    `json:"ttl"`
}

// GatewayHealth represents gateway health status.
type GatewayHealth struct {
	IP         string        `json:"ip"`
	Reachable  bool          `json:"reachable"`
	RTT        time.Duration `json:"rtt"`
	RTTMs      float64       `json:"rttMs"`
	PacketLoss float64       `json:"packetLossPercent"`
	Jitter     float64       `json:"jitterMs"`
	Status     HealthStatus  `json:"status"`
	Uptime     time.Duration `json:"uptime,omitempty"`
	LastCheck  time.Time     `json:"lastCheck"`
}

// HealthStatus represents health check status.
type HealthStatus string

// HealthStatus values.
const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// SNMPDevice contains SNMP-collected device information.
type SNMPDevice struct {
	IP          string          `json:"ip"`
	SysName     string          `json:"sysName,omitempty"`
	SysDescr    string          `json:"sysDescr,omitempty"`
	SysLocation string          `json:"sysLocation,omitempty"`
	SysContact  string          `json:"sysContact,omitempty"`
	SysUpTime   time.Duration   `json:"sysUpTime,omitempty"`
	Interfaces  []SNMPInterface `json:"interfaces,omitempty"`
	VLANs       []SNMPVLAN      `json:"vlans,omitempty"`
	MACTable    []MACTableEntry `json:"macTable,omitempty"`
	Custom      map[string]any  `json:"custom,omitempty"`
	CollectedAt time.Time       `json:"collectedAt"`
}

// SNMPInterface represents an SNMP-collected interface.
type SNMPInterface struct {
	Index       int    `json:"index"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`
	Speed       uint64 `json:"speedBps"`
	AdminStatus string `json:"adminStatus"`
	OperStatus  string `json:"operStatus"`
	InOctets    uint64 `json:"inOctets"`
	OutOctets   uint64 `json:"outOctets"`
	InErrors    uint64 `json:"inErrors"`
	OutErrors   uint64 `json:"outErrors"`
}

// SNMPVLAN represents an SNMP-collected VLAN.
type SNMPVLAN struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Ports  []int  `json:"ports,omitempty"`
}

// MACTableEntry represents a MAC address table entry.
type MACTableEntry struct {
	MACAddress string `json:"macAddress"`
	Port       int    `json:"port"`
	VLANID     int    `json:"vlanId,omitempty"`
	Type       string `json:"type"` // dynamic, static
}

// SpeedtestResult contains internet speed test results.
type SpeedtestResult struct {
	DownloadMbps float64       `json:"downloadMbps"`
	UploadMbps   float64       `json:"uploadMbps"`
	PingMs       float64       `json:"pingMs"`
	JitterMs     float64       `json:"jitterMs"`
	ServerName   string        `json:"serverName,omitempty"`
	ServerID     string        `json:"serverId,omitempty"`
	ISP          string        `json:"isp,omitempty"`
	TestDuration time.Duration `json:"testDuration"`
	TestedAt     time.Time     `json:"testedAt"`
}

// IPerfResult contains iPerf test results.
type IPerfResult struct {
	Protocol      string        `json:"protocol"`  // tcp, udp
	Direction     string        `json:"direction"` // send, receive, bidirectional
	BandwidthMbps float64       `json:"bandwidthMbps"`
	TransferMB    float64       `json:"transferMB"`
	Duration      time.Duration `json:"duration"`
	DurationSec   float64       `json:"durationSec"`
	Jitter        float64       `json:"jitterMs,omitempty"`
	PacketLoss    float64       `json:"packetLossPercent,omitempty"`
	Retransmits   int           `json:"retransmits,omitempty"`
	ServerAddr    string        `json:"serverAddr"`
	TestedAt      time.Time     `json:"testedAt"`
}

// VLANConfig represents VLAN configuration.
type VLANConfig struct {
	ID          int      `json:"id"`
	Name        string   `json:"name,omitempty"`
	Interface   string   `json:"interface"`
	IPAddress   string   `json:"ipAddress,omitempty"`
	SubnetMask  string   `json:"subnetMask,omitempty"`
	Gateway     string   `json:"gateway,omitempty"`
	Tagged      bool     `json:"tagged"`
	MemberPorts []string `json:"memberPorts,omitempty"`
}

// TelemetrySnapshot contains a point-in-time telemetry snapshot.
type TelemetrySnapshot struct {
	Timestamp    time.Time        `json:"timestamp"`
	Links        []LinkStatus     `json:"links,omitempty"`
	Gateway      *GatewayHealth   `json:"gateway,omitempty"`
	DNS          *DNSTestResult   `json:"dns,omitempty"`
	DHCP         *DHCPTestResult  `json:"dhcp,omitempty"`
	Bandwidth    *BandwidthSample `json:"bandwidth,omitempty"`
	SystemHealth *SystemHealth    `json:"systemHealth,omitempty"`
}

// BandwidthSample contains bandwidth measurement.
type BandwidthSample struct {
	Interface     string    `json:"interface"`
	TxBytesPerSec float64   `json:"txBytesPerSec"`
	RxBytesPerSec float64   `json:"rxBytesPerSec"`
	TxMbps        float64   `json:"txMbps"`
	RxMbps        float64   `json:"rxMbps"`
	Utilization   float64   `json:"utilizationPercent"`
	SampledAt     time.Time `json:"sampledAt"`
}

// SystemHealth contains system-level health metrics.
type SystemHealth struct {
	CPUPercent    float64       `json:"cpuPercent"`
	MemoryPercent float64       `json:"memoryPercent"`
	DiskPercent   float64       `json:"diskPercent"`
	Temperature   float64       `json:"temperatureC,omitempty"`
	Uptime        time.Duration `json:"uptime"`
	LoadAverage   []float64     `json:"loadAverage,omitempty"`
	SampledAt     time.Time     `json:"sampledAt"`
}
