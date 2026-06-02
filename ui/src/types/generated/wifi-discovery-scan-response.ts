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
  is_hidden: boolean;
  security_type: string;
  authorization_status: string;
  first_seen: string;
  last_seen: string;
  ap_count?: number;
  best_signal?: number;
  metadata?: {};
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
export interface ChannelUtilization {
  id: string;
  channel: number;
  band: string;
  frequency_mhz: number;
  utilization_percent: number;
  non_wifi_percent: number;
  retry_percent: number;
  ap_count: number;
  client_count: number;
  recorded_at: string;
}
