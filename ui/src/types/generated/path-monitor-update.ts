/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface PathMonitorUpdate {
  target: string;
  rounds: number;
  hops: HopStats[];
}
export interface HopStats {
  ttl: number;
  addresses: HopAddress[];
  sent: number;
  received: number;
  lossPct: number;
  lastRtt: number;
  bestRtt: number;
  worstRtt: number;
  avgRtt: number;
  jitter: number;
}
export interface HopAddress {
  ip: string;
  hostname?: string;
  count: number;
}
