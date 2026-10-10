/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface TopologyLinkListResponse {
  count: number;
  links: TopologyLink[];
}
export interface TopologyLink {
  id: string;
  sourceNodeId: string;
  targetNodeId: string;
  sourceInterface: string;
  targetInterface: string;
  linkType: string;
  status: string;
  speedMbps: number;
  utilizationPct: number;
  firstSeen: string;
  lastSeen: string;
  evidence: {
    [k: string]: unknown;
  };
}
