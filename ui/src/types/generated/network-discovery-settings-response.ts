/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface NetworkDiscoverySettingsResponse {
  enabled: boolean;
  arpScanWorkers: number;
  pingTimeoutMs: number;
  scanTimeoutMs: number;
  autoScan: boolean;
  scanIntervalMs: number;
  ouiFilePath: string;
  options: OptionsResponse;
  timing: TimingResponse;
  profiler: ProfilerResponse;
  fingerprinting: FingerprintingResponse;
  ipv6Enabled: boolean;
}
export interface OptionsResponse {
  passiveProtocols: PassiveProtocolResponse;
  arpScan: boolean;
  icmpScan: boolean;
  portScan: PortScanResponse;
  tcpProbe: TCPProbeSettingsResponse;
  traceroute: boolean;
  snmpQuery: boolean;
}
export interface PassiveProtocolResponse {
  lldp: boolean;
  cdp: boolean;
  edp: boolean;
  ndp: boolean;
}
export interface PortScanResponse {
  enabled: boolean;
  tcpPorts: string;
  udpPorts: string;
  bannerTimeoutMs: number;
}
export interface TCPProbeSettingsResponse {
  timeoutMs: number;
  workers: number;
}
export interface TimingResponse {
  probeIntervalMs: number;
  rescanIntervalMs: number;
  workers: number;
}
export interface ProfilerResponse {
  enabled: boolean;
  timeoutMs: number;
  maxConcurrent: number;
  quickPorts: number[];
}
export interface FingerprintingResponse {
  enabled: boolean;
  osDetection: boolean;
  serviceProbes: boolean;
}
