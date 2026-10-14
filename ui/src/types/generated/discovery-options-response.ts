/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface DiscoveryOptionsResponse {
  options: DiscoveryOptions;
}
export interface DiscoveryOptions {
  passiveProtocols: PassiveProtocolConfig;
  arpScan: boolean;
  icmpScan: boolean;
  portScan: PortScanConfig;
  traceroute: boolean;
  snmpQuery: boolean;
}
export interface PassiveProtocolConfig {
  lldp: boolean;
  cdp: boolean;
  edp: boolean;
  ndp: boolean;
}
export interface PortScanConfig {
  enabled: boolean;
  preset: string;
  tcpPorts: string;
}
