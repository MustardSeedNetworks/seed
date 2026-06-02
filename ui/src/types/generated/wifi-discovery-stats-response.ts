/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface WiFiDiscoveryStatsResponse {
  stats: WiFiDiscoveryStats;
}
export interface WiFiDiscoveryStats {
  total_networks: number;
  hidden_networks: number;
  total_aps: number;
  authorized_aps: number;
  unauthorized_aps: number;
  total_clients: number;
  channels_by_band: {
    [k: string]: number;
  };
  security_breakdown: {
    [k: string]: number;
  };
  vendor_breakdown: {
    [k: string]: number;
  };
  last_scan_time: string;
}
