/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface WiFiDiscoveryAPsResponse {
  accessPoints: WiFiAccessPoint[];
  total: number;
}
export interface WiFiAccessPoint {
  id: string;
  device_id?: string;
  bssid: string;
  ssid_id?: string;
  ssid_name?: string;
  ap_name?: string;
  vendor?: string;
  channel: number;
  channel_width: number;
  frequency_mhz: number;
  band: string;
  wifi_standard?: string[];
  signal_dbm: number;
  noise_dbm?: number;
  client_count: number;
  max_clients?: number;
  is_authorized: boolean;
  first_seen: string;
  last_seen: string;
  metadata?: {};
}
