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
// FAB & Display Options (localStorage)
// ============================================================================

export interface FABOptions {
  runLink: boolean;
  runSwitch: boolean;
  runVLAN: boolean;
  runIPConfig: boolean;
  runGateway: boolean;
  runDNS: boolean;
  runHealthChecks: boolean;
  runNetworkDiscovery: boolean;
  runSpeedtest: boolean;
  runIperf: boolean;
  runPerformance: boolean;
  autoScanOnLink: boolean;
}

export interface DisplayOptions {
  showPublicIP: boolean;
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
// Logs
// ============================================================================

export interface LogsResponse {
  path: string;
  lines: string[];
}

// ============================================================================
// Default Values
// ============================================================================

export const DEFAULT_FAB_OPTIONS: FABOptions = {
  runLink: true,
  runSwitch: true,
  runVLAN: true,
  runIPConfig: true,
  runGateway: true,
  runDNS: true,
  runHealthChecks: true,
  runNetworkDiscovery: true,
  runSpeedtest: true,
  runIperf: false,
  runPerformance: true,
  autoScanOnLink: true,
};

export const DEFAULT_DISPLAY_OPTIONS: DisplayOptions = {
  showPublicIP: true,
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
  enableServer: false,
};

export const DEFAULT_TESTS_SETTINGS: TestsSettings = {
  dnsHostname: "google.com",
  dnsServers: [],
  pingTargets: [],
  tcpPorts: [],
  udpPorts: [],
  httpEndpoints: [],
  runPerformance: false,
  runSpeedtest: false,
  runIperf: false,
  runDiscovery: false,
  speedtest: {
    serverId: "",
    autoRunOnLink: false,
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

// ============================================================================
// localStorage Keys
// ============================================================================

export const STORAGE_KEYS = {
  FAB_OPTIONS: "netscope-fab-options",
  DISPLAY_OPTIONS: "netscope-display-options",
  IPERF_SETTINGS: "netscope-iperf-settings",
  THEME: "netscope-theme",
} as const;
