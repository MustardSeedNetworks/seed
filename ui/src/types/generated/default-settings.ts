/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
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
export interface CardOptionDefaults {
  enabled: boolean;
  autoRunOnLink: boolean;
}
export interface PerformanceCardDefaults {
  enabled: boolean;
  autoRunOnLink: boolean;
  speedtest: CardOptionDefaults;
  iperf: CardOptionDefaults;
}
export interface DisplayOptionsDefaults {
  showPublicIP: boolean;
  unitSystem: string;
}
export interface ThresholdDefaults {
  dns: ThresholdPairDefaults;
  gateway: ThresholdPairDefaults;
  wifi: ThresholdPairIntDefaults;
  customPing: ThresholdPairDefaults;
  customTcp: ThresholdPairDefaults;
  customHttp: ThresholdPairDefaults;
  httpTimings: HTTPTimingThresholdDefaults;
}
export interface ThresholdPairDefaults {
  good: number;
  warning: number;
}
export interface ThresholdPairIntDefaults {
  good: number;
  warning: number;
}
export interface HTTPTimingThresholdDefaults {
  dns: ThresholdPairDefaults;
  tcp: ThresholdPairDefaults;
  tls: ThresholdPairDefaults;
  ttfb: ThresholdPairDefaults;
}
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
export interface NetworkDiscoveryDefaults {
  enabled: boolean;
  scanTimeoutMs: number;
  ipv6Enabled: boolean;
  options: DiscoveryOptionsDefaults;
  timing: DiscoveryTimingDefaults;
  fingerprinting: FingerprintingDefaults;
}
export interface DiscoveryOptionsDefaults {
  passiveProtocols: PassiveProtocolDefaults;
  arpScan: boolean;
  icmpScan: boolean;
  portScan: PortScanDefaults;
  traceroute: boolean;
  snmpQuery: boolean;
}
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
}
export interface DiscoveryTimingDefaults {
  rescanIntervalMs: number;
}
export interface FingerprintingDefaults {
  enabled: boolean;
  osDetection: boolean;
  serviceProbes: boolean;
}
export interface SNMPDefaults {
  timeoutMs: number;
  retries: number;
  port: number;
}
export interface LinkDefaults {
  mode: string;
  availableModes: string[];
}
export interface CableTestDefaults {
  enabled: boolean;
}
export interface VulnerabilityDefaults {
  enabled: boolean;
  cveDatabase: string;
  nvdApiKey: string;
  updateInterval: number;
  severityThreshold: string;
  maxConcurrent: number;
  autoScan: boolean;
}
