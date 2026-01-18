package shell

import (
	"net"
	"time"
)

// Device represents a discovered network device.
type Device struct {
	ID         string            `json:"id"`
	IPAddress  net.IP            `json:"ipAddress"`
	MACAddress string            `json:"macAddress"`
	Hostname   string            `json:"hostname,omitempty"`
	Vendor     string            `json:"vendor,omitempty"`
	DeviceType DeviceType        `json:"deviceType"`
	OS         string            `json:"os,omitempty"`
	Services   []Service         `json:"services,omitempty"`
	Interfaces []DeviceInterface `json:"interfaces,omitempty"`
	FirstSeen  time.Time         `json:"firstSeen"`
	LastSeen   time.Time         `json:"lastSeen"`
	IsOnline   bool              `json:"isOnline"`
	IsGateway  bool              `json:"isGateway"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// DeviceType categorizes network devices.
type DeviceType string

// DeviceType values.
const (
	DeviceTypeRouter      DeviceType = "router"
	DeviceTypeSwitch      DeviceType = "switch"
	DeviceTypeAP          DeviceType = "access_point"
	DeviceTypeServer      DeviceType = "server"
	DeviceTypeWorkstation DeviceType = "workstation"
	DeviceTypeMobile      DeviceType = "mobile"
	DeviceTypeIoT         DeviceType = "iot"
	DeviceTypePrinter     DeviceType = "printer"
	DeviceTypeCamera      DeviceType = "camera"
	DeviceTypeUnknown     DeviceType = "unknown"
)

// DeviceInterface represents a network interface on a device.
type DeviceInterface struct {
	Name        string   `json:"name"`
	MACAddress  string   `json:"macAddress"`
	IPAddresses []string `json:"ipAddresses"`
	Type        string   `json:"type"` // ethernet, wifi, etc.
	Speed       string   `json:"speed,omitempty"`
	Status      string   `json:"status"`
}

// Service represents a network service on a device.
type Service struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // tcp, udp
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Banner   string `json:"banner,omitempty"`
	State    string `json:"state"` // open, filtered, closed
}

// DiscoveryResult contains results from a discovery scan.
type DiscoveryResult struct {
	Devices        []Device      `json:"devices"`
	NewDevices     int           `json:"newDevices"`
	UpdatedDevices int           `json:"updatedDevices"`
	OfflineDevices int           `json:"offlineDevices"`
	ScanDuration   time.Duration `json:"scanDuration"`
	ScanDurationMs float64       `json:"scanDurationMs"`
	StartedAt      time.Time     `json:"startedAt"`
	CompletedAt    time.Time     `json:"completedAt"`
}

// DiscoveryOptions configures a discovery scan.
type DiscoveryOptions struct {
	Interface     string        `json:"interface,omitempty"`
	Subnets       []string      `json:"subnets,omitempty"`
	EnableARP     bool          `json:"enableArp"`
	EnableICMP    bool          `json:"enableIcmp"`
	EnableNDP     bool          `json:"enableNdp"`
	EnableLLDP    bool          `json:"enableLldp"`
	EnableCDP     bool          `json:"enableCdp"`
	EnableSNMP    bool          `json:"enableSnmp"`
	PortScan      bool          `json:"portScan"`
	PortScanPorts []int         `json:"portScanPorts,omitempty"`
	Timeout       time.Duration `json:"timeout"`
	Concurrency   int           `json:"concurrency"`
}

// Vulnerability represents a discovered security vulnerability.
type Vulnerability struct {
	ID              string       `json:"id"`
	DeviceID        string       `json:"deviceId"`
	CVEID           string       `json:"cveId,omitempty"`
	Title           string       `json:"title"`
	Description     string       `json:"description"`
	Severity        VulnSeverity `json:"severity"`
	CVSSScore       float64      `json:"cvssScore,omitempty"`
	CVSSVector      string       `json:"cvssVector,omitempty"`
	AffectedPort    int          `json:"affectedPort,omitempty"`
	AffectedService string       `json:"affectedService,omitempty"`
	Remediation     string       `json:"remediation,omitempty"`
	References      []string     `json:"references,omitempty"`
	IsKEV           bool         `json:"isKev"` // CISA KEV
	IsExploited     bool         `json:"isExploited"`
	DiscoveredAt    time.Time    `json:"discoveredAt"`
	Status          VulnStatus   `json:"status"`
}

// VulnSeverity categorizes vulnerability severity.
type VulnSeverity string

// VulnSeverity values.
const (
	SeverityCritical VulnSeverity = "critical"
	SeverityHigh     VulnSeverity = "high"
	SeverityMedium   VulnSeverity = "medium"
	SeverityLow      VulnSeverity = "low"
	SeverityInfo     VulnSeverity = "info"
)

// VulnStatus tracks vulnerability remediation status.
type VulnStatus string

// VulnStatus values.
const (
	VulnStatusNew           VulnStatus = "new"
	VulnStatusAcknowledged  VulnStatus = "acknowledged"
	VulnStatusInProgress    VulnStatus = "in_progress"
	VulnStatusResolved      VulnStatus = "resolved"
	VulnStatusFalsePositive VulnStatus = "false_positive"
)

// VulnerabilityScan contains results from a vulnerability scan.
type VulnerabilityScan struct {
	ID              string          `json:"id"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	DevicesScanned  int             `json:"devicesScanned"`
	TotalCritical   int             `json:"totalCritical"`
	TotalHigh       int             `json:"totalHigh"`
	TotalMedium     int             `json:"totalMedium"`
	TotalLow        int             `json:"totalLow"`
	ScanDuration    time.Duration   `json:"scanDuration"`
	ScanDurationMs  float64         `json:"scanDurationMs"`
	StartedAt       time.Time       `json:"startedAt"`
	CompletedAt     time.Time       `json:"completedAt"`
}

// PostureScore represents a security posture assessment score.
type PostureScore struct {
	Overall      int            `json:"overall"` // 0-100
	Categories   map[string]int `json:"categories"`
	Issues       []PostureIssue `json:"issues"`
	Improvements []string       `json:"improvements,omitempty"`
	AssessedAt   time.Time      `json:"assessedAt"`
}

// PostureIssue represents a security posture finding.
type PostureIssue struct {
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Remediation string `json:"remediation,omitempty"`
}

// RogueDevice represents a potentially unauthorized device.
type RogueDevice struct {
	Device       Device    `json:"device"`
	Reason       string    `json:"reason"`
	RiskLevel    string    `json:"riskLevel"` // high, medium, low
	DetectedAt   time.Time `json:"detectedAt"`
	Acknowledged bool      `json:"acknowledged"`
}

// RogueAlert represents an alert for a rogue device.
type RogueAlert struct {
	ID             string      `json:"id"`
	Device         RogueDevice `json:"device"`
	AlertType      string      `json:"alertType"`
	Message        string      `json:"message"`
	CreatedAt      time.Time   `json:"createdAt"`
	AcknowledgedAt *time.Time  `json:"acknowledgedAt,omitempty"`
}
