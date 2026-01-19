/**
 * Settings Type Definitions
 *
 * All settings-related interfaces used by SettingsContext and SettingsDrawer.
 * These types define the shape of settings stored in localStorage and backend API.
 */

// ============================================================================
// Save Status
// ============================================================================

export type SaveStatus = 'idle' | 'saving' | 'saved' | 'error';

// ============================================================================
// Threshold Types
// ============================================================================

export interface ThresholdPair {
  good: number;
  warning: number;
}

export interface SettingsThresholds {
  dns: ThresholdPair;
  gateway: ThresholdPair;
  wifi: ThresholdPair;
  customPing: ThresholdPair;
  customTcp: ThresholdPair;
  customHttp: ThresholdPair;
  httpTimings: {
    dns: ThresholdPair;
    tcp: ThresholdPair;
    tls: ThresholdPair;
    ttfb: ThresholdPair;
  };
}

// ============================================================================
// Per-Card Settings
// ============================================================================

/**
 * Settings for individual card behavior.
 * - enabled: Whether the card is shown and functional
 * - autoRunOnLink: Whether the card's tests run when FAB is clicked
 */
export interface CardOption {
  enabled: boolean;
  autoRunOnLink: boolean;
}

/**
 * Performance card has nested options for speedtest and iperf subtests.
 */
export interface PerformanceCardOption extends CardOption {
  speedtest: CardOption;
  iperf: CardOption;
}

/**
 * Per-card settings for visibility and auto-run behavior.
 * Each card can be independently enabled/disabled and configured
 * to run automatically when the FAB button is clicked.
 */
export interface CardSettings {
  link: CardOption;
  switch: CardOption;
  vlan: CardOption;
  network: CardOption; // IPConfig/DHCP card
  gateway: CardOption;
  dns: CardOption;
  healthChecks: CardOption;
  networkDiscovery: CardOption;
  performance: PerformanceCardOption;
}

// ============================================================================
// Display Options
// ============================================================================

/** Unit system for measurements - SAE (feet) or Metric (meters) */
export type UnitSystem = 'sae' | 'metric';

export interface DisplayOptions {
  showPublicIp: boolean;
  /** Unit system for distances and measurements (default: SAE/feet) */
  unitSystem: UnitSystem;
}

// ============================================================================
// Test Configuration Types
// ============================================================================

export interface PingTarget {
  id?: string; // Stable unique ID for React key
  name: string;
  host: string;
  enabled: boolean;
  count?: number;
  criticality?: number; // 1-10 scale for health scoring
}

export interface DnsServer {
  id?: string; // Stable unique ID for React key
  address: string;
  enabled: boolean;
}

export interface TcpPort {
  id?: string; // Stable unique ID for React key
  name: string;
  host: string;
  port: number;
  enabled: boolean;
  criticality?: number; // 1-10 scale for health scoring
}

export interface UdpPort {
  id?: string; // Stable unique ID for React key
  name: string;
  host: string;
  port: number;
  enabled: boolean;
  criticality?: number; // 1-10 scale for health scoring
}

export interface HttpEndpoint {
  id?: string; // Stable unique ID for React key
  name: string;
  url: string;
  expectedStatus: number;
  enabled: boolean;
  criticality?: number; // 1-10 scale for health scoring
}

// ============================================================================
// Extended Health Check Endpoint Types
// ============================================================================

/** RTSP video stream endpoint (surveillance/video systems) */
export interface RtspEndpoint {
  id?: string;
  name: string;
  url: string; // rtsp://host:554/path
  enabled: boolean;
  criticality?: number;
}

/** DICOM medical imaging endpoint */
export interface DicomEndpoint {
  id?: string;
  name: string;
  host: string;
  port: number; // Default: 104
  aeTitle: string; // Application Entity Title
  enabled: boolean;
  criticality?: number;
}

/** SQL database endpoint */
export interface SqlEndpoint {
  id?: string;
  name: string;
  driver: 'postgres' | 'mysql' | 'mssql' | 'oracle';
  host: string;
  port: number;
  database: string;
  username: string;
  enabled: boolean;
  criticality?: number;
}

/** File share endpoint (SMB/NFS) */
export interface FileShareEndpoint {
  id?: string;
  name: string;
  protocol: 'smb' | 'nfs';
  host: string;
  sharePath: string;
  enabled: boolean;
  criticality?: number;
}

/** LDAP/Active Directory endpoint */
export interface LdapEndpoint {
  id?: string;
  name: string;
  host: string;
  port: number; // Default: 389, LDAPS: 636
  useTls: boolean;
  baseDn: string;
  enabled: boolean;
  criticality?: number;
}

/** HL7 MLLP medical messaging endpoint */
export interface Hl7Endpoint {
  id?: string;
  name: string;
  host: string;
  port: number; // Default: 2575
  sendingApp: string;
  sendingFacility: string;
  receivingApp: string;
  receivingFacility: string;
  enabled: boolean;
  criticality?: number;
}

/** FHIR R4 healthcare API endpoint */
export interface FhirEndpoint {
  id?: string;
  name: string;
  baseUrl: string;
  authType: 'none' | 'basic' | 'oauth2';
  enabled: boolean;
  criticality?: number;
}

/** LTI/LMS education endpoint */
export interface LtiEndpoint {
  id?: string;
  name: string;
  launchUrl: string;
  consumerKey: string;
  enabled: boolean;
  criticality?: number;
}

/** OPC-UA industrial endpoint */
export interface OpcuaEndpoint {
  id?: string;
  name: string;
  endpointUrl: string; // opc.tcp://host:4840
  securityMode: 'None' | 'Sign' | 'SignAndEncrypt';
  enabled: boolean;
  criticality?: number;
}

/** Modbus TCP industrial endpoint */
export interface ModbusEndpoint {
  id?: string;
  name: string;
  host: string;
  port: number; // Default: 502
  unitId: number;
  testRegister: number;
  enabled: boolean;
  criticality?: number;
}

/** SLA configuration for an endpoint */
export interface SlaConfig {
  endpointName: string;
  targetUptime: number; // Percentage (e.g., 99.9)
  targetLatencyP95: number; // Milliseconds
  reportingPeriod: 'daily' | 'weekly' | 'monthly';
  enabled: boolean;
}

/** Alert configuration settings */
export interface AlertConfig {
  enabled: boolean;
  consecutiveFailures: number; // Number of failures before alerting
  cooldownMinutes: number; // Minutes between alerts for same endpoint
  digestMode: boolean; // Batch alerts instead of immediate
}

/** Anomaly detection configuration */
export interface AnomalyConfig {
  enabled: boolean;
  stdDevThreshold: number; // Number of standard deviations (default: 2)
  maxSamples: number; // Rolling window size (default: 100)
}

export interface TestsSettings {
  dnsHostname: string;
  dnsServers: DnsServer[];
  pingTargets: PingTarget[];
  tcpPorts: TcpPort[];
  udpPorts: UdpPort[];
  httpEndpoints: HttpEndpoint[];
  // Extended health check endpoints
  rtspEndpoints?: RtspEndpoint[];
  dicomEndpoints?: DicomEndpoint[];
  sqlEndpoints?: SqlEndpoint[];
  fileShareEndpoints?: FileShareEndpoint[];
  ldapEndpoints?: LdapEndpoint[];
  hl7Endpoints?: Hl7Endpoint[];
  fhirEndpoints?: FhirEndpoint[];
  ltiEndpoints?: LtiEndpoint[];
  opcuaEndpoints?: OpcuaEndpoint[];
  modbusEndpoints?: ModbusEndpoint[];
  // Performance test flags
  runPerformance: boolean;
  runSpeedtest: boolean;
  runIperf: boolean;
  runDiscovery: boolean;
  speedtest: {
    serverId: string;
    autoRunOnLink: boolean;
  };
  iperf: {
    autoRunOnLink: boolean;
  };
  // SLA and health monitoring configuration
  slaConfigs?: SlaConfig[];
  alertConfig?: AlertConfig;
  anomalyConfig?: AnomalyConfig;
}

// ============================================================================
// iPerf Settings
// ============================================================================

export interface IperfSettings {
  server: string;
  port: number;
  protocol: 'tcp' | 'udp';
  direction: 'upload' | 'download' | 'bidirectional';
  duration: number;
  serverPort: number;
  enableServer: boolean;
}

export interface IperfSuggestion {
  host: string;
  hostname?: string;
  latencyMs?: number;
  source?: string;
}

// ============================================================================
// Network Discovery Settings (fixes #773, #774)
// ============================================================================

/** Port preset for quick configuration */
export type PortPreset = 'common' | 'secure' | 'insecure' | 'custom';

/** Passive protocol configuration (LLDP, CDP, EDP, NDP) */
export interface PassiveProtocolConfig {
  lldp: boolean; // IEEE 802.1AB Link Layer Discovery Protocol
  cdp: boolean; // Cisco Discovery Protocol
  edp: boolean; // Extreme Discovery Protocol
  ndp: boolean; // IPv6 Neighbor Discovery Protocol
}

/** Port scan configuration */
export interface PortScanConfig {
  enabled: boolean;
  preset: PortPreset; // Quick selection of port sets
  tcpPorts: string; // Comma-separated ports or ranges (e.g., "22,80,443,8000-8100")
  udpPorts: string; // Comma-separated ports or ranges
  bannerTimeoutMs: number; // Timeout for banner grabbing
}

/** TCP probe configuration */
export interface TcpProbeConfig {
  timeoutMs: number; // Connection timeout
  workers: number; // Concurrent probe workers
}

/** Discovery options - granular protocol control */
export interface DiscoveryOptions {
  passiveProtocols: PassiveProtocolConfig; // Granular passive protocol control
  arpScan: boolean;
  icmpScan: boolean;
  portScan: PortScanConfig;
  tcpProbe: TcpProbeConfig;
  traceroute: boolean;
  snmpQuery: boolean;
}

/** Discovery timing configuration */
export interface DiscoveryTimingConfig {
  probeIntervalMs: number; // Time between probes
  rescanIntervalMs: number; // Time between full rescans
  workers: number; // Concurrent workers
}

/** Device profiler configuration */
export interface DeviceProfilerConfig {
  enabled: boolean;
  timeoutMs: number;
  maxConcurrent: number;
  quickPorts: number[];
}

/** Fingerprinting configuration */
export interface FingerprintingConfig {
  enabled: boolean;
  osDetection: boolean;
  serviceProbes: boolean;
}

export interface DiscoveryServiceStatus {
  running: boolean;
  scanning: boolean;
  deviceCount: number;
  lastScan: string;
  subnet: string;
  // biome-ignore lint/style/useNamingConvention: Matches backend API json:"localIP"
  localIP: string;
  interface: string;
  activeMethods: string[];
  rescanInterval: number; // Backend sends time.Duration (nanoseconds)
}

/** Full network discovery settings - matches API response */
export interface NetworkDiscoverySettings {
  // Core settings
  enabled: boolean;
  arpScanWorkers: number;
  pingTimeoutMs: number;
  scanTimeoutMs: number;
  autoScan: boolean;
  scanIntervalMs: number;
  // Note: OUI database is baked into binary at build time - no runtime path needed

  // Configuration objects
  options: DiscoveryOptions;
  timing: DiscoveryTimingConfig;
  profiler: DeviceProfilerConfig;
  fingerprinting: FingerprintingConfig;
  ipv6Enabled: boolean;
}

export interface SubnetConfig {
  cidr: string;
  name: string;
  enabled: boolean;
}

// ============================================================================
// IP Configuration
// ============================================================================

export interface IpSettings {
  mode: 'dhcp' | 'static';
  address: string;
  netmask: string;
  gateway: string;
  dns: string[];
}

// ============================================================================
// WiFi Settings
// ============================================================================

export interface WiFiSettings {
  interface: string;
  availableWifi: string[];
  isWireless: boolean;
}

// ============================================================================
// SNMP Settings
// ============================================================================

export interface SnmpV3Credential {
  id?: string; // Stable unique ID for React key
  name: string;
  username: string;
  authProtocol: string; // "MD5", "SHA", "SHA256", "SHA512", or "" for noAuth
  authPassword: string;
  privProtocol: string; // "DES", "AES", "AES192", "AES256", or "" for noPriv
  privPassword: string;
  contextName: string;
  securityLevel: string; // "noAuthNoPriv", "authNoPriv", "authPriv"
}

export interface SnmpSettings {
  communities: string[];
  v3Credentials: SnmpV3Credential[];
  timeout: number; // milliseconds
  retries: number;
  port: number;
}

// ============================================================================
// Logs
// ============================================================================

export interface LogsResponse {
  path: string;
  lines: string[];
}

// ============================================================================
// Default Values
// ============================================================================
//
// DEPRECATED: These constants are now fallbacks only.
// The single source of truth for defaults is the backend API:
//   GET /api/settings/defaults
//
// Use the useDefaults() hook from "../hooks/useDefaults" to get defaults.
// These constants will be removed in a future version.
// ============================================================================

/** @deprecated Use useDefaults() hook instead - backend is single source of truth */
export const DEFAULT_CARD_SETTINGS: CardSettings = {
  link: { enabled: true, autoRunOnLink: true },
  switch: { enabled: true, autoRunOnLink: true },
  vlan: { enabled: true, autoRunOnLink: true },
  network: { enabled: true, autoRunOnLink: true },
  gateway: { enabled: true, autoRunOnLink: true },
  dns: { enabled: true, autoRunOnLink: true },
  healthChecks: { enabled: true, autoRunOnLink: true },
  networkDiscovery: { enabled: true, autoRunOnLink: true },
  performance: {
    enabled: true,
    autoRunOnLink: true,
    speedtest: { enabled: true, autoRunOnLink: true },
    iperf: { enabled: false, autoRunOnLink: false },
  },
};

/** @deprecated Use useDefaults() hook instead - backend is single source of truth */
export const DEFAULT_DISPLAY_OPTIONS: DisplayOptions = {
  showPublicIp: true,
  unitSystem: 'sae', // Default to SAE (feet) for US users
};

/** @deprecated Use useDefaults() hook instead - backend is single source of truth */
export const DEFAULT_THRESHOLDS: SettingsThresholds = {
  dns: { good: 50, warning: 100 },
  gateway: { good: 20, warning: 50 },
  wifi: { good: -50, warning: -70 },
  customPing: { good: 50, warning: 100 },
  customTcp: { good: 100, warning: 200 },
  customHttp: { good: 500, warning: 1000 },
  httpTimings: {
    dns: { good: 50, warning: 100 },
    tcp: { good: 50, warning: 100 },
    tls: { good: 100, warning: 200 },
    ttfb: { good: 200, warning: 500 },
  },
};

/** @deprecated Use useDefaults() hook instead - backend is single source of truth */
export const DEFAULT_IPERF_SETTINGS: IperfSettings = {
  server: '',
  port: 5201,
  protocol: 'tcp',
  direction: 'download',
  duration: 10,
  serverPort: 5201,
  enableServer: true,
};

/**
 * Fixes #730: Add sensible default health check tests so the card appears by default
 * @deprecated Use useDefaults() hook instead - backend is single source of truth
 */
export const DEFAULT_TESTS_SETTINGS: TestsSettings = {
  dnsHostname: 'google.com',
  dnsServers: [],
  pingTargets: [
    {
      id: 'default-google-dns',
      name: 'Google DNS',
      host: '8.8.8.8',
      enabled: true,
      count: 3,
    },
    {
      id: 'default-cloudflare-dns',
      name: 'Cloudflare',
      host: '1.1.1.1',
      enabled: true,
      count: 3,
    },
  ],
  tcpPorts: [],
  udpPorts: [],
  httpEndpoints: [
    {
      id: 'default-google',
      name: 'Google',
      url: 'https://www.google.com',
      expectedStatus: 200,
      enabled: true,
    },
  ],
  runPerformance: false,
  runSpeedtest: false,
  runIperf: false,
  runDiscovery: false,
  speedtest: {
    serverId: '',
    autoRunOnLink: true, // Fixes #728: Match DEFAULT_CARD_SETTINGS
  },
  iperf: {
    autoRunOnLink: false,
  },
};

/** @deprecated Use useDefaults() hook instead - backend is single source of truth */
export const DEFAULT_NETWORK_DISCOVERY_SETTINGS: NetworkDiscoverySettings = {
  // Core settings
  enabled: true,
  arpScanWorkers: 50,
  pingTimeoutMs: 500,
  scanTimeoutMs: 30000,
  autoScan: true,
  scanIntervalMs: 600000, // 10 minutes

  // Configuration objects
  ipv6Enabled: true,
  options: {
    passiveProtocols: {
      lldp: true,
      cdp: true,
      edp: true,
      ndp: true,
    },
    arpScan: true,
    icmpScan: true,
    portScan: {
      enabled: false,
      preset: 'common',
      tcpPorts: '22,80,443,8080-8100',
      udpPorts: '53,123,161',
      bannerTimeoutMs: 2000,
    },
    tcpProbe: {
      timeoutMs: 2000,
      workers: 20,
    },
    traceroute: false,
    snmpQuery: false,
  },
  timing: {
    probeIntervalMs: 75,
    rescanIntervalMs: 600000, // 10 minutes
    workers: 50,
  },
  profiler: {
    enabled: true,
    timeoutMs: 2000,
    maxConcurrent: 5,
    quickPorts: [22, 80, 443, 8080],
  },
  fingerprinting: {
    enabled: false,
    osDetection: false,
    serviceProbes: false,
  },
};

/** @deprecated Use useDefaults() hook instead - backend is single source of truth */
export const DEFAULT_SNMP_SETTINGS: SnmpSettings = {
  communities: ['public'],
  v3Credentials: [],
  timeout: 5000, // 5 seconds
  retries: 2,
  port: 161,
};

// ============================================================================
// Link Settings
// ============================================================================

/** Supported link speed values (Mbps) - includes copper and fiber speeds */
export type LinkSpeed =
  | 'auto'
  | '10' // 10BASE-T
  | '100' // 100BASE-TX
  | '1000' // 1000BASE-T (1G)
  | '2500' // 2.5GBASE-T
  | '5000' // 5GBASE-T
  | '10000' // 10GBASE-T
  | '25000' // 25GBASE-CR/SR (fiber)
  | '40000' // 40GBASE-CR4/SR4 (fiber)
  | '100000'; // 100GBASE-CR4/SR4 (fiber)

/** Supported duplex modes */
export type DuplexMode = 'auto' | 'full' | 'half';

export interface LinkSettings {
  /** Combined speed/duplex mode (e.g., "100/full", "1000/full") or "auto" for auto-negotiation */
  mode: string;
  /** Available speed/duplex modes for the interface (e.g., ["10/half", "100/full", "1000/full"]) */
  availableModes: string[];
}

// ============================================================================
// Cable Test Settings
// ============================================================================

export interface CableTestSettings {
  /** Whether the cable test card is enabled (requires PHY TDR support) */
  enabled: boolean;
}

/** @deprecated Use useDefaults() hook instead - backend is single source of truth */
export const DEFAULT_LINK_SETTINGS: LinkSettings = {
  mode: 'auto',
  availableModes: [],
};

/** @deprecated Use useDefaults() hook instead - backend is single source of truth */
export const DEFAULT_CABLE_TEST_SETTINGS: CableTestSettings = {
  enabled: true,
};

// ============================================================================
// Vulnerability Scanning Settings
// ============================================================================

export type CveDatabase = 'nvd' | 'local';
export type SeverityLevel = 'low' | 'medium' | 'high' | 'critical';

export interface VulnerabilityScanSettings {
  enabled: boolean;
  cveDatabase: CveDatabase;
  nvdApiKey: string;
  updateInterval: number; // seconds between database updates
  severityThreshold: SeverityLevel;
  maxConcurrent: number; // max concurrent vulnerability checks
  autoScan: boolean; // auto-scan after device discovery
}

export interface VulnerabilityScanStatus {
  enabled: boolean;
  scanning: boolean;
  stats: {
    totalDevices: number;
    scannedDevices: number;
    totalVulnerabilities: number;
    bySeverity: Record<string, number>;
    lastScanTime: string;
  };
  severityFilter: string;
}

/** @deprecated Use useDefaults() hook instead - backend is single source of truth */
export const DEFAULT_VULNERABILITY_SETTINGS: VulnerabilityScanSettings = {
  enabled: true, // Enable by default for security visibility
  cveDatabase: 'nvd', // NVD works without API key (rate limited)
  nvdApiKey: '',
  updateInterval: 86400, // 24 hours
  severityThreshold: 'medium',
  maxConcurrent: 5,
  autoScan: true, // Auto-scan after device discovery
};

// ============================================================================
// localStorage Keys
// ============================================================================

export const STORAGE_KEYS = {
  cardSettings: 'seed-card-settings',
  displayOptions: 'seed-display-options',
  iperfSettings: 'seed-iperf-settings',
  theme: 'seed-theme',
} as const;
