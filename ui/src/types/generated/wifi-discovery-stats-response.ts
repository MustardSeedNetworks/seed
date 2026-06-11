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
  totalNetworks: number;
  hiddenNetworks: number;
  totalAps: number;
  authorizedAps: number;
  unauthorizedAps: number;
  totalClients: number;
  channelsByBand: {
    [k: string]: number;
  };
  securityBreakdown: {
    [k: string]: number;
  };
  vendorBreakdown: {
    [k: string]: number;
  };
  lastScanTime: string;
}
