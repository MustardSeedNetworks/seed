/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface DHCPLeaseResponse {
  interface: string;
  success: boolean;
  status: 'success' | 'warning' | 'error';
  serverIp?: string;
  offeredIp?: string;
  subnetMask?: string;
  gateway?: string;
  dnsServers?: string[];
  domainName?: string;
  leaseTimeSec?: number;
  responseTimeMs: number;
  error?: string;
  testedAt: string;
}
