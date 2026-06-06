/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface WiFiAirspaceResponse {
  ssids: SSIDGroup[];
  status: Status;
}
export interface SSIDGroup {
  ssid: string;
  hidden: boolean;
  apCount: number;
  bssCount: number;
  stationCount: number;
  aps: APGroup[];
}
export interface APGroup {
  key: string;
  vendor?: string;
  bsses: BSSView[];
}
export interface BSSView {
  bssid: string;
  ssid: string;
  hidden: boolean;
  band: string;
  channel: number;
  security: string;
  standard: string;
  countryCode?: string;
  pmfRequired: boolean;
  rrmNeighbor: boolean;
  btmSupported: boolean;
  ftSupported: boolean;
  wpsEnabled: boolean;
  signalDbm: number;
  beacons: number;
  lastSeen: string;
  stations: StationView[];
}
export interface StationView {
  mac: string;
  signalDbm: number;
  frames: number;
  lastSeen: string;
}
export interface Status {
  captureActive: boolean;
  source?: string;
  ssids: number;
  aps: number;
  bsses: number;
  stations: number;
  anomalies: number;
  lastEvaluated?: string;
}
