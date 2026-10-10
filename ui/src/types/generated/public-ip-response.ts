/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface PublicIPResponse {
  ipv4?: string;
  ipv6?: string;
  lastChecked: string;
  error?: string;
  isp?: string;
  asn?: string;
  org?: string;
  city?: string;
  region?: string;
  country?: string;
  countryCode?: string;
  lat?: number;
  lon?: number;
  history?: HistoryEntry[];
}
export interface HistoryEntry {
  ip: string;
  firstSeen: string;
  lastSeen: string;
  city?: string;
  country?: string;
}
