/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface DNSResponse {
  interface: string;
  server: string;
  servers: string[];
  testHostname: string;
  forward?: DNSLookupResult;
  forwardIpv6?: DNSLookupResult;
  reverse?: DNSLookupResult;
  reverseIpv6?: DNSLookupResult;
  perServerResults?: DNSServerTestResult[];
}
export interface DNSLookupResult {
  result: string;
  time: number;
  timeMs: number;
  status: string;
  error?: string;
  resolved?: string[];
}
export interface DNSServerTestResult {
  server: string;
  forward?: DNSLookupResult;
  forwardIpv6?: DNSLookupResult;
  status: string;
  avgTimeMs: number;
}
