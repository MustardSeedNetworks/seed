/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface ExportResponse {
  version: string;
  timestamp: string;
  device: DeviceInfo;
  cards: Cards;
}
export interface DeviceInfo {
  interface: string;
  mac?: string;
  ipMode: string;
}
export interface Cards {
  link?: LinkCard;
  ipConfig?: IPConfigCard;
  switch?: SwitchCard;
  dns?: DNSCard;
  gateway?: GatewayCard;
  vlan?: VLANCard;
  wifi?: WiFiCard;
  cable?: CableCard;
  speedtest?: SpeedtestCard;
  iperf?: IperfCard;
}
export interface LinkCard {
  linkUp: boolean;
  speed: string;
  duplex: string;
  autoNeg: boolean;
}
export interface IPConfigCard {
  addresses: string[];
  lease?: LeaseCard;
}
export interface LeaseCard {
  dhcpServer: string;
  gateway: string;
  leaseTime: number;
  dns: string[];
}
export interface SwitchCard {
  running: boolean;
  neighbors: NeighborRecord[];
}
export interface NeighborRecord {
  protocol: string;
  systemName: string;
  portId: string;
  portDescription: string;
  managementAddress: string;
}
export interface DNSCard {
  server: string;
  testHostname: string;
  forward?: DNSLookup;
  reverse?: DNSLookup;
}
export interface DNSLookup {
  result: string[];
  time: number;
  status: string;
  error: string;
}
export interface GatewayCard {
  gateway: string;
  reachable: boolean;
  sent: number;
  received: number;
  lossPercent: number;
  avgTime: number;
  status: string;
}
export interface VLANCard {
  nativeVlan?: number;
  taggedVlans: number[];
  voiceVlan?: number;
  configured: VLANConfigured;
}
export interface VLANConfigured {
  enabled: boolean;
  id: number;
}
export interface WiFiCard {
  ssid: string;
  bssid: string;
  signal: number;
  channel: number;
  frequency: number;
  security: string;
}
export interface CableCard {
  supported: boolean;
  length?: number;
  status: string;
  faults: string[];
}
export interface SpeedtestCard {
  download: number;
  upload: number;
  latency: number;
  server: string;
  location: string;
  host: string;
  distance: number;
  timestamp: string;
  testDuration: number;
}
export interface IperfCard {
  bandwidth: number;
  transfer: number;
  retransmits: number;
  jitter: number;
  lostPackets: number;
  lostPercent: number;
  protocol: string;
  direction: string;
  duration: number;
  server: string;
  port: number;
  timestamp: string;
}
