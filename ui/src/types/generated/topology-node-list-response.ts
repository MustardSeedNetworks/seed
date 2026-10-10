/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface TopologyNodeListResponse {
  count: number;
  nodes: TopologyNode[];
}
export interface TopologyNode {
  id: string;
  clientId: string;
  identityHash: string;
  displayName: string;
  deviceType: string;
  chassisId: string;
  sysName: string;
  primaryMac: string;
  primaryIp: string;
  firstSeen: string;
  lastSeen: string;
  metadata: {
    [k: string]: unknown;
  };
}
