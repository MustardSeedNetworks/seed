/**
 * Settings Type Definitions
 *
 * All settings-related interfaces used by SettingsContext and SettingsDrawer.
 * These types define the shape of settings stored in localStorage and backend API.
 */

// ============================================================================
// Save Status
// ============================================================================

export type SaveStatus = "idle" | "saving" | "saved" | "error";

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
export type UnitSystem = "sae" | "metric";

export interface DisplayOptions {
  showPublicIP: boolean;
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
}

export interface DNSServer {
  id?: string; // Stable unique ID for React key
  address: string;
  enabled: boolean;
}

export interface TCPPort {
  id?: string; // Stable unique ID for React key
  name: string;
  host: string;
  port: number;
  enabled: boolean;
}

export interface UDPPort {
  id?: string; // Stable unique ID for React key
  name: string;
  host: string;
  port: number;
  enabled: boolean;
}

export interface HTTPEndpoint {
  id?: string; // Stable unique ID for React key
  name: string;
  url: string;
  expectedStatus: number;
  enabled: boolean;
}

export interface TestsSettings {
  dnsHostname: string;
  dnsServers: DNSServer[];
  pingTargets: PingTarget[];
  tcpPorts: TCPPort[];
  udpPorts: UDPPort[];
  httpEndpoints: HTTPEndpoint[];
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
}

// ============================================================================
// iPerf Settings
// ============================================================================

export interface IperfSettings {
  server: string;
  port: number;
  protocol: "tcp" | "udp";
  direction: "upload" | "download" | "bidirectional";
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

export type DiscoveryProfile = "stealth" | "standard" | "full_scan" | "custom";

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
  tcpPorts: string; // Comma-separated ports or ranges (e.g., "22,80,443,8000-8100")
  udpPorts: string; // Comma-separated ports or ranges
  bannerTimeoutMs: number; // Timeout for banner grabbing
}

/** TCP probe configuration */
export interface TCPProbeConfig {
  timeoutMs: number; // Connection timeout
  workers: number; // Concurrent probe workers
}

/** Discovery custom options - granular protocol control */
export interface DiscoveryCustomOptions {
  passiveListen: boolean; // Legacy: enables all passive protocols
  passiveProtocols: PassiveProtocolConfig; // Granular passive protocol control
  arpScan: boolean;
  icmpScan: boolean;
  portScan: PortScanConfig;
  tcpProbe: TCPProbeConfig;
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
  profile: DiscoveryProfile;
  scanning: boolean;
  deviceCount: number;
  lastScan: string;
  subnet: string;
  localIP: string;
  interface: string;
  activeMethods: string[];
  rescanInterval: number;
}

/** Full network discovery settings - matches API response */
export interface NetworkDiscoverySettings {
  // Legacy fields (backward compatibility)
  enabled: boolean;
  arpScanWorkers: number;
  pingTimeoutMs: number;
  scanTimeoutMs: number;
  autoScan: boolean;
  scanIntervalMs: number;
  ouiFilePath: string;

  // Profile-based configuration
  profile: DiscoveryProfile;
  customOptions: DiscoveryCustomOptions;
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

export interface IPSettings {
  mode: "dhcp" | "static";
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

export interface SNMPv3Credential {
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

export interface SNMPSettings {
  communities: string[];
  v3Credentials: SNMPv3Credential[];
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

export const DEFAULT_DISPLAY_OPTIONS: DisplayOptions = {
  showPublicIP: true,
  unitSystem: "sae", // Default to SAE (feet) for US users
};

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

export const DEFAULT_IPERF_SETTINGS: IperfSettings = {
  server: "",
  port: 5201,
  protocol: "tcp",
  direction: "download",
  duration: 10,
  serverPort: 5201,
  enableServer: true,
};

// Fixes #730: Add sensible default health check tests so the card appears by default
export const DEFAULT_TESTS_SETTINGS: TestsSettings = {
  dnsHostname: "google.com",
  dnsServers: [],
  pingTargets: [
    {
      id: "default-google-dns",
      name: "Google DNS",
      host: "8.8.8.8",
      enabled: true,
      count: 3,
    },
    {
      id: "default-cloudflare-dns",
      name: "Cloudflare",
      host: "1.1.1.1",
      enabled: true,
      count: 3,
    },
  ],
  tcpPorts: [],
  udpPorts: [],
  httpEndpoints: [
    {
      id: "default-google",
      name: "Google",
      url: "https://www.google.com",
      expectedStatus: 200,
      enabled: true,
    },
  ],
  runPerformance: false,
  runSpeedtest: false,
  runIperf: false,
  runDiscovery: false,
  speedtest: {
    serverId: "",
    autoRunOnLink: true, // Fixes #728: Match DEFAULT_CARD_SETTINGS
  },
  iperf: {
    autoRunOnLink: false,
  },
};

export const DEFAULT_NETWORK_DISCOVERY_SETTINGS: NetworkDiscoverySettings = {
  // Legacy fields
  enabled: true,
  arpScanWorkers: 50,
  pingTimeoutMs: 500,
  scanTimeoutMs: 30000,
  autoScan: true,
  scanIntervalMs: 600000, // 10 minutes
  ouiFilePath: "",

  // Profile-based configuration
  profile: "standard",
  ipv6Enabled: true,
  customOptions: {
    passiveListen: true,
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
      tcpPorts: "22,80,443,8080-8100",
      udpPorts: "53,123,161",
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

export const DEFAULT_SNMP_SETTINGS: SNMPSettings = {
  communities: ["public"],
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
  | "auto"
  | "10" // 10BASE-T
  | "100" // 100BASE-TX
  | "1000" // 1000BASE-T (1G)
  | "2500" // 2.5GBASE-T
  | "5000" // 5GBASE-T
  | "10000" // 10GBASE-T
  | "25000" // 25GBASE-CR/SR (fiber)
  | "40000" // 40GBASE-CR4/SR4 (fiber)
  | "100000"; // 100GBASE-CR4/SR4 (fiber)

/** Supported duplex modes */
export type DuplexMode = "auto" | "full" | "half";

export interface LinkSettings {
  /** Whether to use auto-negotiation */
  autoNegotiation: boolean;
  /** Fixed speed when auto-negotiation is disabled (Mbps) */
  speed: LinkSpeed;
  /** Duplex mode when auto-negotiation is disabled */
  duplex: DuplexMode;
  /** Available speed/duplex modes for the interface */
  availableModes: string[];
}

// ============================================================================
// Cable Test Settings
// ============================================================================

export interface CableTestSettings {
  /** Whether the cable test card is enabled */
  enabled: boolean;
  /** Automatically run cable test when link is down */
  autoRunOnLinkDown: boolean;
  /** Unit for cable length display */
  lengthUnit: "feet" | "meters";
}

export const DEFAULT_LINK_SETTINGS: LinkSettings = {
  autoNegotiation: true,
  speed: "auto",
  duplex: "auto",
  availableModes: [],
};

export const DEFAULT_CABLE_TEST_SETTINGS: CableTestSettings = {
  enabled: true,
  autoRunOnLinkDown: false,
  lengthUnit: "feet",
};

// ============================================================================
// Vulnerability Scanning Settings
// ============================================================================

export type CVEDatabase = "nvd" | "local";
export type SeverityLevel = "low" | "medium" | "high" | "critical";

export interface VulnerabilityScanSettings {
  enabled: boolean;
  cveDatabase: CVEDatabase;
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

export const DEFAULT_VULNERABILITY_SETTINGS: VulnerabilityScanSettings = {
  enabled: false,
  cveDatabase: "nvd",
  nvdApiKey: "",
  updateInterval: 86400, // 24 hours
  severityThreshold: "medium",
  maxConcurrent: 5,
  autoScan: false,
};

// ============================================================================
// localStorage Keys
// ============================================================================

export const STORAGE_KEYS = {
  CARD_SETTINGS: "seed-card-settings",
  DISPLAY_OPTIONS: "seed-display-options",
  IPERF_SETTINGS: "seed-iperf-settings",
  THEME: "seed-theme",
} as const;
