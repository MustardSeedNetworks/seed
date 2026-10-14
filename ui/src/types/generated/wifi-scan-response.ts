/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface WiFiScanResponse {
  interface: string;
  available: boolean;
  networks: ScannedNetwork[];
  remediation?: string;
  error?: string;
}
export interface ScannedNetwork {
  ssid: string;
  bssid: string;
  signal: number;
  channel: number;
  frequency: number;
  security: string;
  channelWidth: number;
  noiseFloor: number;
  snr: number;
  htMode: string;
  isDFS: boolean;
  lastSeen: string;
  hidden: boolean;
  standard?: string;
  countryCode?: string;
  pmfRequired: boolean;
  wpsEnabled: boolean;
  rrmNeighbor: boolean;
  btmSupported: boolean;
  ftSupported: boolean;
  hasBssLoad: boolean;
  channelUtil: number;
  advertisedStations: number;
}
