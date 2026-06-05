package config

// config_constants.go holds the default value, port, timing, and threshold
// constants used by DefaultConfig and friends. They live in their own file so
// config.go can stay focused on the Config struct itself.

// IP configuration mode constants.
const (
	ipModeDHCP   = "dhcp"
	ipModeStatic = "static"
)

// Default configuration value constants.
const (
	// defaultHTTPSPort is the default port for HTTPS server (standard HTTPS alternate port).
	defaultHTTPSPort = 8443

	// defaultStartupRetries is the number of retries when finding interface at startup.
	defaultStartupRetries = 3

	// defaultStartupRetryWaitSec is the delay in seconds between startup retries.
	defaultStartupRetryWaitSec = 5

	// defaultDiscoveryTimeoutSec is the default timeout in seconds for switch discovery.
	defaultDiscoveryTimeoutSec = 30

	// defaultSNMPPort is the standard SNMP protocol port.
	defaultSNMPPort = 161

	// defaultSNMPTimeoutSec is the default timeout in seconds for SNMP queries.
	defaultSNMPTimeoutSec = 5

	// defaultSNMPRetries is the default number of retries for failed SNMP queries.
	defaultSNMPRetries = 2

	// defaultSNMPMaxRepetitions is the default number of OID values per GetBulk request.
	defaultSNMPMaxRepetitions = 10

	// defaultDNSTimeoutSec is the default timeout in seconds for DNS queries.
	defaultDNSTimeoutSec = 5

	// defaultIperfPort is the standard iperf3 server port.
	defaultIperfPort = 5201

	// defaultIperfDurationSec is the default iperf test duration in seconds.
	defaultIperfDurationSec = 10
)

// Logging default constants.
const (
	// defaultLogMaxSizeMB is the maximum size in megabytes before log rotation.
	defaultLogMaxSizeMB = 100

	// defaultLogMaxBackups is the number of old log files to keep.
	defaultLogMaxBackups = 5

	// defaultLogMaxAgeDays is the maximum number of days to retain old log files.
	defaultLogMaxAgeDays = 30
)

// API and rate limiting constants.
const (
	// defaultRateLimitPerMinute is the default API rate limit per client per minute.
	defaultRateLimitPerMinute = 60
)

// Database default constants.
const (
	// defaultDBRetentionDays is the default number of days to retain historical data.
	defaultDBRetentionDays = 90

	// defaultDBMaxConnections is the default maximum number of database connections.
	defaultDBMaxConnections = 10

	// defaultBackupMaxCount is the default maximum number of config backups to retain.
	defaultBackupMaxCount = 10
)

// Network port constants for common services.
const (
	// portHTTPS is the standard HTTPS port.
	portHTTPS = 443

	// portDICOM is the standard DICOM medical imaging protocol port.
	portDICOM = 104

	// portFTP is the standard FTP control port.
	portFTP = 21

	// portSMB is the standard SMB/CIFS file sharing port.
	portSMB = 445

	// portRTSP is the standard RTSP streaming protocol port.
	portRTSP = 554

	// portPostgreSQL is the standard PostgreSQL database port.
	portPostgreSQL = 5432

	// portSSH is the standard SSH/SFTP port.
	portSSH = 22

	// portDNS is the standard DNS port.
	portDNS = 53

	// portNTP is the standard NTP time synchronization port.
	portNTP = 123

	// portHTTP is the standard HTTP port for default health check endpoints.
	portHTTP = 80

	// portHTTPAlt is the alternate HTTP port commonly used for web servers.
	portHTTPAlt = 8080
)

// HTTP response status codes.
const (
	// httpStatusOK is the HTTP 200 OK status code.
	httpStatusOK = 200
)

// Timeout and timing constants.
const (
	// defaultBannerTimeoutSec is the timeout in seconds for service banner grabbing.
	defaultBannerTimeoutSec = 2

	// defaultTracerouteTimeoutSec is the timeout in seconds for traceroute TCP probes.
	defaultTracerouteTimeoutSec = 2

	// defaultTracerouteWorkers is the number of concurrent traceroute workers.
	defaultTracerouteWorkers = 20

	// defaultMDNSTimeoutSec is the timeout in seconds for mDNS/device profiling operations.
	defaultMDNSTimeoutSec = 2

	// defaultMDNSMaxConcurrent is the maximum concurrent mDNS/profiler operations.
	defaultMDNSMaxConcurrent = 5

	// defaultProbeIntervalMs is the interval in milliseconds between network probes.
	defaultProbeIntervalMs = 75

	// defaultRescanIntervalMin is the interval in minutes between full network rescans.
	defaultRescanIntervalMin = 10

	// defaultARPWorkers is the number of concurrent ARP scan workers.
	defaultARPWorkers = 50

	// defaultPingTimeoutMs is the timeout in milliseconds for ICMP ping operations.
	defaultPingTimeoutMs = 500

	// defaultScanTimeoutSec is the total timeout in seconds for network scans.
	defaultScanTimeoutSec = 30

	// defaultOUIMaxAgeDays is the maximum age in days for OUI database before refresh.
	defaultOUIMaxAgeDays = 30

	// defaultSessionTimeoutHours is the default session timeout in hours.
	defaultSessionTimeoutHours = 24

	// defaultVulnUpdateIntervalSec is the interval in seconds for vulnerability database updates (24 hours).
	defaultVulnUpdateIntervalSec = 86400
)

// Threshold timing constants (in milliseconds).
const (
	// thresholdDHCPTotalWarningMs is the warning threshold for total DHCP time.
	thresholdDHCPTotalWarningMs = 500

	// thresholdDHCPPhaseWarningMs is the warning threshold for DHCP phase time.
	thresholdDHCPPhaseWarningMs = 200

	// thresholdDNSWarningMs is the warning threshold for DNS resolution.
	thresholdDNSWarningMs = 100

	// thresholdPingWarningMs is the warning threshold for ping latency.
	thresholdPingWarningMs = 50

	// thresholdPingCriticalMs is the critical threshold for ping latency.
	thresholdPingCriticalMs = 200

	// thresholdCustomPingCriticalMs is the critical threshold for custom ping tests.
	thresholdCustomPingCriticalMs = 100

	// thresholdTCPWarningMs is the warning threshold for TCP connections.
	thresholdTCPWarningMs = 100

	// thresholdHTTPWarningMs is the warning threshold for HTTP requests.
	thresholdHTTPWarningMs = 500

	// thresholdTLSWarningMs is the warning threshold for TLS handshakes.
	thresholdTLSWarningMs = 150
)

// Logging value constants. These are the canonical level/format strings used
// in both DefaultConfig and validateLoggingConfig.
const (
	logLevelInfo  = "info"
	logFormatText = "text"
	logFormatJSON = "json"
)

// Threshold integer constants.
const (
	// thresholdWiFiSignalWarningDBm is the warning threshold for WiFi signal strength in dBm.
	thresholdWiFiSignalWarningDBm = -70

	// thresholdWiFiSignalCriticalDBm is the critical threshold for WiFi signal strength in dBm.
	thresholdWiFiSignalCriticalDBm = -80

	// thresholdLinkFlapWarning is the warning threshold for link flaps in 24 hours.
	thresholdLinkFlapWarning = 3

	// thresholdLinkFlapCritical is the critical threshold for link flaps in 24 hours.
	thresholdLinkFlapCritical = 5

	// thresholdCertExpiryWarningDays is days until certificate expiry warning.
	thresholdCertExpiryWarningDays = 30

	// thresholdCertExpiryCriticalDays is days until certificate expiry critical alert.
	thresholdCertExpiryCriticalDays = 7
)
