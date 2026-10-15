/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface IPSLAHistoryResponse {
  targetId: string;
  index: number;
  range: '1h' | '24h' | '7d';
  from: string;
  to: string;
  bucketSeconds: number;
  points: IPSLARunResponse[];
}
export interface IPSLARunResponse {
  sampledAt: string;
  runs: number;
  completedPct: number;
  rttMs?: number;
  jitterMs?: number;
  lossPct?: number;
  mos?: number;
}
