/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface InterfaceHistoryResponse {
  targetId: string;
  ifIndex: number;
  range: '1h' | '24h' | '7d';
  from: string;
  to: string;
  bucketSeconds: number;
  points: InterfaceRatesResponse[];
}
export interface InterfaceRatesResponse {
  sampledAt: string;
  inOctetsPerSec?: number;
  outOctetsPerSec?: number;
  inUtilizationPct?: number;
  outUtilizationPct?: number;
  inErrorsPerSec: number;
  outErrorsPerSec: number;
  inDiscardsPerSec: number;
  outDiscardsPerSec: number;
}
