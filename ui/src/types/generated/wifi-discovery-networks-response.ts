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
  is_hidden: boolean;
  security_type: string;
  authorization_status: string;
  first_seen: string;
  last_seen: string;
  ap_count?: number;
  best_signal?: number;
  metadata?: {};
}
