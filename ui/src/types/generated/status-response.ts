/**
 * AUTO-GENERATED FILE. DO NOT EDIT BY HAND.
 *
 * Regenerate with: `npm run gen-types` (or `make schema && npm run gen-types`
 * after Go DTO changes). The schema source of truth lives at
 * docs/schemas/api/; the Go DTO source lives at internal/api/.
 */
/**
 * StatusResponse — generated from the Go struct in internal/api; refresh with `make schema` after struct changes.
 */
export interface StatusResponse {
  status: string;
  version: string;
  uptime: number;
  interface: string;
  isWireless: boolean;
  icmpAvailable: boolean;
}
