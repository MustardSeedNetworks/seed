/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface WiFiDiscoveryNetworksResponse {
  networks: WiFiNetwork[];
  total: number;
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
