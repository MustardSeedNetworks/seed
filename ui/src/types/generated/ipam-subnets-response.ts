/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface IPAMSubnetsResponse {
  subnets: IPAMSubnet[];
  scopes: IPAMScope[];
}
export interface IPAMSubnet {
  cidr: string;
  name?: string;
  source: 'local' | 'configured' | 'learned';
  size: number;
  used: number;
  free: number;
  utilizationPct: number;
  inUse: string[];
}
export interface IPAMScope {
  name?: string;
  start: string;
  end: string;
  size: number;
  used: number;
  free: number;
  utilizationPct: number;
  exhaustion: 'none' | 'near' | 'full';
  inUse: string[];
}
