/**
 * Default Settings Types
 *
 * These types match the backend's DefaultSettings structure from /api/settings/defaults.
 * The backend is the single source of truth for all default values.
 */

// ============================================================================
// Card Option Defaults
// ============================================================================

export interface CardOptionDefaults {
  enabled: boolean;
  autoRunOnLink: boolean;
}

export interface PerformanceCardDefaults extends CardOptionDefaults {
  speedtest: CardOptionDefaults;
  iperf: CardOptionDefaults;
}

export interface CardSettingsDefaults {
  link: CardOptionDefaults;
  switch: CardOptionDefaults;
  vlan: CardOptionDefaults;
  network: CardOptionDefaults;
  gateway: CardOptionDefaults;
  dns: CardOptionDefaults;
  healthChecks: CardOptionDefaults;
  networkDiscovery: CardOptionDefaults;
  performance: PerformanceCardDefaults;
}

// ============================================================================
// Display Options Defaults
// ============================================================================

export interface DisplayOptionsDefaults {
  showPublicIP: boolean;
  unitSystem: string;
}

// ============================================================================
// Threshold Defaults
// ============================================================================

export interface ThresholdPairDefaults {
  good: number;
  warning: number;
}

export interface HTTPTimingThresholdDefaults {
  dns: ThresholdPairDefaults;
  tcp: ThresholdPairDefaults;
  tls: ThresholdPairDefaults;
  ttfb: ThresholdPairDefaults;
}

export interface ThresholdDefaults {
  dns: ThresholdPairDefaults;
  gateway: ThresholdPairDefaults;
  wifi: ThresholdPairDefaults;
  customPing: ThresholdPairDefaults;
  customTcp: ThresholdPairDefaults;
  customHttp: ThresholdPairDefaults;
  httpTimings: HTTPTimingThresholdDefaults;
}

// ============================================================================
// iPerf Defaults
// ============================================================================

export interface IperfDefaults {
  server: string;
  port: number;
  protocol: string;
  direction: string;
  duration: number;
  serverPort: number;
  enableServer: boolean;
  autoRunOnLink: boolean;
}

// ============================================================================
// Tests Defaults
// ============================================================================

export interface DefaultPingTarget {
  id: string;
  name: string;
  host: string;
  enabled: boolean;
  count?: number;
}

export interface DefaultHTTPEndpoint {
  id: string;
  name: string;
  url: string;
  expectedStatus: number;
  enabled: boolean;
}

export interface SpeedtestDefaults {
  serverId: string;
  autoRunOnLink: boolean;
}

export interface TestsDefaults {
  dnsHostname: string;
  pingTargets: DefaultPingTarget[];
  httpEndpoints: DefaultHTTPEndpoint[];
  runPerformance: boolean;
  runSpeedtest: boolean;
  runIperf: boolean;
  runDiscovery: boolean;
  speedtest: SpeedtestDefaults;
}

// ============================================================================
// Network Discovery Defaults
// ============================================================================

export interface PassiveProtocolDefaults {
  lldp: boolean;
  cdp: boolean;
  edp: boolean;
  ndp: boolean;
}

export interface PortScanDefaults {
  enabled: boolean;
  preset: string;
  tcpPorts: string;
  udpPorts: string;
  bannerTimeoutMs: number;
}

export interface TCPProbeDefaults {
  timeoutMs: number;
  workers: number;
}

export interface DiscoveryOptionsDefaults {
  passiveProtocols: PassiveProtocolDefaults;
  arpScan: boolean;
  icmpScan: boolean;
  portScan: PortScanDefaults;
  tcpProbe: TCPProbeDefaults;
  traceroute: boolean;
  snmpQuery: boolean;
}

export interface DiscoveryTimingDefaults {
  probeIntervalMs: number;
  rescanIntervalMs: number;
  workers: number;
}

export interface DeviceProfilerDefaults {
  enabled: boolean;
  timeoutMs: number;
  maxConcurrent: number;
  quickPorts: number[];
}

export interface FingerprintingDefaults {
  enabled: boolean;
  osDetection: boolean;
  serviceProbes: boolean;
}

export interface NetworkDiscoveryDefaults {
  enabled: boolean;
  arpScanWorkers: number;
  pingTimeoutMs: number;
  scanTimeoutMs: number;
  autoScan: boolean;
  scanIntervalMs: number;
  ouiFilePath: string;
  ipv6Enabled: boolean;
  options: DiscoveryOptionsDefaults;
  timing: DiscoveryTimingDefaults;
  profiler: DeviceProfilerDefaults;
  fingerprinting: FingerprintingDefaults;
}

// ============================================================================
// SNMP Defaults
// ============================================================================

export interface SNMPDefaults {
  communities: string[];
  timeoutMs: number;
  retries: number;
  port: number;
}

// ============================================================================
// Link Defaults
// ============================================================================

export interface LinkDefaults {
  autoNegotiation: boolean;
  speed: string;
  duplex: string;
  availableModes: string[];
}

// ============================================================================
// Cable Test Defaults
// ============================================================================

export interface CableTestDefaults {
  enabled: boolean;
  autoRunOnLinkDown: boolean;
}

// ============================================================================
// Vulnerability Defaults
// ============================================================================

export interface VulnerabilityDefaults {
  enabled: boolean;
  cveDatabase: string;
  nvdApiKey: string;
  updateInterval: number;
  severityThreshold: string;
  maxConcurrent: number;
  autoScan: boolean;
}

// ============================================================================
// Complete Default Settings (from /api/settings/defaults)
// ============================================================================

export interface DefaultSettings {
  cardSettings: CardSettingsDefaults;
  displayOptions: DisplayOptionsDefaults;
  thresholds: ThresholdDefaults;
  iperf: IperfDefaults;
  tests: TestsDefaults;
  networkDiscovery: NetworkDiscoveryDefaults;
  snmp: SNMPDefaults;
  link: LinkDefaults;
  cableTest: CableTestDefaults;
  vulnerability: VulnerabilityDefaults;
}
