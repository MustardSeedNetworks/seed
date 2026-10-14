/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
export interface PortScanResponse {
  ip: string;
  hostname?: string;
  services: ServiceInfo[];
  scanTimeMs: number;
  error?: string;
}
export interface ServiceInfo {
  port: number;
  state: 'open' | 'closed' | 'filtered';
  service: string;
  banner?: string;
  version?: string;
  protocol?: string;
}
