package discovery

// profiler_types.go contains the value types emitted by DeviceProfiler:
// DeviceProfile / OpenPort / HTTPInfo / SNMPInfo / MDNSService, the
// resolved-names + profiling-status records, and the small device-classifier
// pattern struct. Constants used across profiler files also live here.

import "time"

// Discovery vendor / web-server name constants reused across the profiler
// classifier and fingerprint signatures.
const (
	vendorFortinet  = "fortinet"
	webServerApache = "apache"
)

// Device type constants for profiler classification.
const (
	deviceTypePrinter       = "printer"
	deviceTypeNetworkDevice = "network-device"
	deviceTypeServer        = "server"
	deviceTypeRouter        = "router"
	deviceTypeSwitch        = "switch"
	deviceTypeFirewall      = "firewall"
	deviceTypeNAS           = "nas"
)

// Profiler timing and buffer constants.
const (
	profilerQueueSize         = 100  // Size of profiling queue channel
	profilerDefaultTimeoutS   = 10   // Default timeout for profiling operations in seconds
	profilerBannerReadMs      = 500  // Timeout for reading service banners in milliseconds
	profilerBannerBufferSize  = 256  // Buffer size for reading service banners
	profilerHTTPBodyLimit     = 8192 // Maximum HTTP response body to read
	profilerLogTruncateLen    = 50   // Maximum length for log message truncation
	profilerMinTruncateLen    = 3    // Minimum length for truncation with ellipsis
	profilerTitleMaxLen       = 100  // Maximum length for extracted HTML titles
	profilerTimeoutS          = 2    // Default profiler timeout in seconds
	profilerMaxConcurrent     = 10   // Default max concurrent profiling operations
	profilerProbeDelayMs      = 50   // Default probe delay in milliseconds
	profilerHostDelayMs       = 20   // Default host delay in milliseconds
	profilerNameResolveTimeMs = 500  // Default name resolution timeout in milliseconds
)

// Common port numbers for service classification.
const (
	portFTP        = 21
	portSSHProf    = 22
	portTelnet     = 23
	portSMTP       = 25
	portDNS        = 53
	portHTTPProf   = 80
	portPOP3       = 110
	portIMAP       = 143
	portSNMP       = 161
	portSMTPSubmit = 587
	portMySQL      = 3306
	portPostgreSQL = 5432
	portRedis      = 6379
	portHTTPAltP   = 8080
	portHTTPSProf  = 443
	portHTTPSAltP  = 8443
	portJetDirect  = 9100
	portLPD        = 515
	portIPP        = 631
	portMongoDB    = 27017
)

// DeviceProfile contains auto-discovered profile information about a device.
type DeviceProfile struct {
	ProfiledAt   time.Time     `json:"profiledAt"`
	OpenPorts    []OpenPort    `json:"openPorts,omitempty"`
	HTTPInfo     *HTTPInfo     `json:"httpInfo,omitempty"`
	SNMPInfo     *SNMPInfo     `json:"snmpInfo,omitempty"`
	MDNSServices []MDNSService `json:"mdnsServices,omitempty"`
	DeviceType   string        `json:"deviceType,omitempty"`  // Inferred type: router, switch, printer, server, etc.
	DeviceIcons  []string      `json:"deviceIcons,omitempty"` // Icon hints for UI: web, ssh, snmp, printer, etc.
}

// OpenPort represents an open port found during profiling.
type OpenPort struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // tcp or udp
	Service  string `json:"service,omitempty"`
	Banner   string `json:"banner,omitempty"`
	IsOpen   bool   `json:"isOpen"`
}

// HTTPInfo contains HTTP/HTTPS probe results.
type HTTPInfo struct {
	Port       int    `json:"port"`
	StatusCode int    `json:"statusCode"`
	Title      string `json:"title,omitempty"`
	Server     string `json:"server,omitempty"`
	IsHTTPS    bool   `json:"isHttps"`
}

// SNMPInfo contains SNMP probe results.
type SNMPInfo struct {
	SysDescr    string `json:"sysDescr,omitempty"`
	SysName     string `json:"sysName,omitempty"`
	SysContact  string `json:"sysContact,omitempty"`
	SysLocation string `json:"sysLocation,omitempty"`
}

// MDNSService represents an mDNS/Bonjour advertised service.
type MDNSService struct {
	Name string            `json:"name"`
	Type string            `json:"type"`
	Port int               `json:"port,omitempty"`
	TXT  map[string]string `json:"txt,omitempty"`
}

// ResolvedNames holds resolved names for a device.
type ResolvedNames struct {
	Hostname    string // DNS PTR resolved name
	NetBIOSName string // Windows NetBIOS name
	MDNSName    string // mDNS/Bonjour .local name
}

// ProfilingStatus represents the current state of the device profiler.
type ProfilingStatus struct {
	TotalProfiled int      `json:"totalProfiled"` // Number of devices successfully profiled
	InProgress    int      `json:"inProgress"`    // Number of devices currently being profiled
	QueueLength   int      `json:"queueLength"`   // Number of devices waiting to be profiled
	ProfilingIPs  []string `json:"profilingIps"`  // IPs currently being profiled
	Enabled       bool     `json:"enabled"`       // Whether profiling is enabled
	MaxConcurrent int      `json:"maxConcurrent"` // Maximum concurrent profiling operations
	PortsToScan   int      `json:"portsToScan"`   // Number of ports being scanned per device
	ScanIntensity string   `json:"scanIntensity"` // Current port scan intensity level
}

// httpDeviceMatch defines a pattern match for HTTP-based device detection.
type httpDeviceMatch struct {
	titlePatterns  []string
	serverPatterns []string
	deviceType     string
	icon           string
}
