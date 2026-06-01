/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface DiscoveryResponse {
  interface: string;
  running: boolean;
  neighbors: DiscoveryNeighborInfo[];
}
export interface DiscoveryNeighborInfo {
  protocol: string;
  chassisId: string;
  portId: string;
  portDescription?: string;
  systemName?: string;
  systemDescription?: string;
  capabilities?: string[];
  managementAddress?: string;
  ttl: number;
  lastSeen: string;
  sourceMAC: string;
}
