/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface InterfaceStatsListResponse {
  interfaces: InterfaceStatsResponse[];
  count: number;
}
export interface InterfaceStatsResponse {
  targetId: string;
  targetName: string;
  ifIndex: number;
  name: string;
  alias?: string;
  operStatus: 'up' | 'down' | 'testing' | 'unknown' | 'dormant' | 'notPresent' | 'lowerLayerDown';
  speedBps: number;
  rates?: InterfaceRatesResponse;
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
