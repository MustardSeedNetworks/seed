/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface TCPProbeResponse {
  target: string;
  results: TCPProbeResult[];
}
export interface TCPProbeResult {
  ip: string;
  port: number;
  state: string;
  ttl: number;
  rtt: number;
  flags?: number;
  error?: string;
}
