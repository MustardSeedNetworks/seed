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
// Network Discovery Settings
// ============================================================================

export type DiscoveryProfile = "stealth" | "standard" | "full_scan" | "custom";

export interface DiscoveryCustomOptions {
  passiveListen: boolean;
  arpScan: boolean;
  icmpScan: boolean;
  portScan: {
    enabled: boolean;
    ports: number[];
    topPorts: number;
  };
  traceroute: boolean;
  snmpQuery: boolean;
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

export interface NetworkDiscoverySettings {
  enabled: boolean;
  profile: DiscoveryProfile;
  arpScanWorkers: number;
  pingTimeoutMs: number;
  scanTimeoutMs: number;
  autoScan: boolean;
  scanIntervalMs: number;
  ouiFilePath: string;
  customOptions: DiscoveryCustomOptions;
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
  enabled: true,
  profile: "standard",
  arpScanWorkers: 50,
  pingTimeoutMs: 500,
  scanTimeoutMs: 30000,
  autoScan: false,
  scanIntervalMs: 300000,
  ouiFilePath: "",
  customOptions: {
    passiveListen: true,
    arpScan: true,
    icmpScan: true,
    portScan: {
      enabled: false,
      ports: [],
      topPorts: 100,
    },
    traceroute: false,
    snmpQuery: false,
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
// localStorage Keys
// ============================================================================

export const STORAGE_KEYS = {
  CARD_SETTINGS: "seed-card-settings",
  DISPLAY_OPTIONS: "seed-display-options",
  IPERF_SETTINGS: "seed-iperf-settings",
  THEME: "seed-theme",
} as const;
