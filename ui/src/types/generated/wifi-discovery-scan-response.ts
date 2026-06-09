/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface WiFiDiscoveryScanResponse {
  networks: WiFiNetwork[];
  accessPoints: WiFiAccessPoint[];
  channelUtilization: ChannelUtilization[];
  scanTime: string;
  interface: string;
}
export interface WiFiNetwork {
  id: string;
  ssid: string;
  isHidden: boolean;
  securityType: string;
  authorizationStatus: string;
  firstSeen: string;
  lastSeen: string;
  apCount?: number;
  bestSignal?: number;
  metadata?: {};
}
export interface WiFiAccessPoint {
  id: string;
  deviceId?: string;
  bssid: string;
  ssidId?: string;
  ssidName?: string;
  apName?: string;
  vendor?: string;
  channel: number;
  channelWidth: number;
  frequencyMhz: number;
  band: string;
  wifiStandard?: string[];
  signalDbm: number;
  noiseDbm?: number;
  clientCount: number;
  maxClients?: number;
  isAuthorized: boolean;
  firstSeen: string;
  lastSeen: string;
  metadata?: {};
}
export interface ChannelUtilization {
  id: string;
  channel: number;
  band: string;
  frequencyMhz: number;
  utilizationPercent: number;
  nonWifiPercent: number;
  retryPercent: number;
  apCount: number;
  clientCount: number;
  recordedAt: string;
}
