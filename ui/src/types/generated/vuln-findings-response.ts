/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface VulnFindingsResponse {
  count: number;
  findings: VulnFindingResponse[];
}
export interface VulnFindingResponse {
  id: number;
  deviceId: string;
  deviceIp: string;
  hostname?: string;
  cveId: string;
  severity: string;
  score: number;
  description?: string;
  affectedComponent?: string;
  affectedVersion?: string;
  status: string;
  detectedAt: string;
  resolvedAt?: string;
}
